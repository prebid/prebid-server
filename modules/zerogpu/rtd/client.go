package rtd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/prebid/prebid-server/v4/logger"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

const (
	// statusInsufficientQuota is ZeroGPU's non-standard quota-exhausted status.
	statusInsufficientQuota = 420

	// maxResponseBytes bounds how much of an upstream response is read. A
	// classification is on the order of a kilobyte.
	maxResponseBytes = 1 << 20
)

// responsesRequest is the body for the /v1/responses API.
type responsesRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

// apiEnvelope is the /v1/responses wrapper. The classification itself is a JSON
// string nested inside it, so parsing happens in two stages.
type apiEnvelope struct {
	Output []struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
}

// payload returns the embedded classification JSON string, or "" when the
// envelope carries none.
func (e apiEnvelope) payload() string {
	for _, out := range e.Output {
		for _, c := range out.Content {
			if c.Text != "" {
				return c.Text
			}
		}
	}
	return ""
}

// classification is the inner JSON payload produced by the domain classifier.
type classification struct {
	Audience []scoredID     `json:"audience"`
	Content  contentByTaxon `json:"content"`
}

type contentByTaxon struct {
	IAB10 []scoredCode `json:"iab_1_0"`
	IAB22 []scoredID   `json:"iab_2_2"`
}

type scoredID struct {
	ID    int     `json:"id"`
	Score float64 `json:"score"`
}

type scoredCode struct {
	Code  string  `json:"code"`
	Score float64 `json:"score"`
}

// segments is the cached, already-filtered result for one domain. Fields are
// abbreviated because every byte is stored in the in-memory cache.
type segments struct {
	Content22 []string `json:"c22,omitempty"`
	Content10 []string `json:"c10,omitempty"`
	Audience  []string `json:"aud,omitempty"`
}

func (s segments) isEmpty() bool {
	return len(s.Content22) == 0 && len(s.Content10) == 0 && len(s.Audience) == 0
}

// apiError distinguishes conditions that resolve on their own (a cold domain
// still warming ZeroGPU's server-side cache, a 5xx, a timeout) from stable ones
// (bad key, exhausted quota, unclassifiable domain). Only the cache TTL and log
// level differ - the auction proceeds either way.
type apiError struct {
	msg       string
	transient bool
}

func (e apiError) Error() string { return e.msg }

func transientErrorf(format string, args ...any) apiError {
	return apiError{msg: fmt.Sprintf(format, args...), transient: true}
}

func permanentErrorf(format string, args ...any) apiError {
	return apiError{msg: fmt.Sprintf(format, args...)}
}

// isTransient reports whether err warrants the short retry TTL.
func isTransient(err error) bool {
	var apiErr apiError
	if errors.As(err, &apiErr) {
		return apiErr.transient
	}
	// Transport failures and context deadlines are always worth retrying.
	return true
}

func (m *Module) cacheKey(domain string) []byte {
	return []byte("zerogpu:" + m.cfg.Model + ":" + domain)
}

// lookup returns the cached segments for a domain. The second return value
// reports whether the cache had an answer at all - a cached empty result (an
// unclassifiable domain, or a suppressed failure) is a hit carrying no
// segments, which is different from never having asked.
//
// This never performs I/O. The auction is never blocked on ZeroGPU.
func (m *Module) lookup(domain string) (segments, bool) {
	key := m.cacheKey(domain)

	cached, err := m.cache.Get(key)
	if err != nil {
		return segments{}, false
	}

	var s segments
	if err := jsonutil.Unmarshal(cached, &s); err != nil {
		// A corrupt entry is not worth preserving.
		m.cache.Del(key)
		return segments{}, false
	}
	return s, true
}

// warm populates the cache for a domain in the background so later auctions on
// the same domain can be enriched from memory.
//
// Concurrent auctions for the same uncached domain collapse onto a single
// request: only the goroutine that claims the domain fetches. The fetch uses
// the module's own lifetime context, never the hook context, because the hook
// context is cancelled the moment the execution plan's group timeout elapses -
// which would abort the warm-up before it could finish and leave the domain
// permanently cold.
func (m *Module) warm(domain string) {
	// Refuse new work once the host is tearing the module down, so no goroutine
	// is registered after Shutdown has begun waiting for them.
	if m.bgCtx.Err() != nil {
		return
	}
	if _, alreadyWarming := m.inFlight.LoadOrStore(domain, struct{}{}); alreadyWarming {
		return
	}

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		defer m.inFlight.Delete(domain)

		defer func() {
			if r := recover(); r != nil {
				logger.Errorf("[zerogpu.rtd] panic while warming %q: %v", domain, r)
			}
		}()

		ctx, cancel := context.WithTimeout(m.bgCtx, time.Duration(m.cfg.Timeout)*time.Millisecond)
		defer cancel()

		key := m.cacheKey(domain)
		s, err := m.fetch(ctx, domain)
		if err != nil {
			m.cacheNegative(key, err)
			return
		}
		m.cacheResult(key, s)
	}()
}

// cacheResult stores a successful classification. Empty results use the
// negative TTL so an unclassifiable domain is not re-queried for a full day.
func (m *Module) cacheResult(key []byte, s segments) {
	ttl := m.cfg.CacheTTLSeconds
	if s.isEmpty() {
		ttl = m.cfg.NegativeCacheTTLSeconds
	}
	encoded, err := jsonutil.Marshal(s)
	if err != nil {
		return
	}
	if err := m.cache.Set(key, encoded, ttl); err != nil {
		logger.Infof("[zerogpu.rtd] could not cache classification: %v", err)
	}
}

// cacheNegative suppresses repeat calls after a failure. Transient failures get
// the short retry TTL because ZeroGPU warms its server-side cache on the first
// request for a new domain, so retrying shortly is likely to succeed.
func (m *Module) cacheNegative(key []byte, cause error) {
	ttl := m.cfg.NegativeCacheTTLSeconds
	if isTransient(cause) {
		ttl = m.cfg.RetryCacheTTLSeconds
	}
	if ttl <= 0 {
		return
	}
	if err := m.cache.Set(key, []byte("{}"), ttl); err != nil {
		logger.Infof("[zerogpu.rtd] could not negative-cache domain: %v", err)
	}
}

// fetch performs the classification call and filters the response.
func (m *Module) fetch(ctx context.Context, domain string) (segments, error) {
	body, err := jsonutil.Marshal(responsesRequest{Model: m.cfg.Model, Input: domain})
	if err != nil {
		return segments{}, permanentErrorf("failed to encode request: %s", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.cfg.Endpoint, bytes.NewReader(body))
	if err != nil {
		return segments{}, permanentErrorf("failed to build request: %s", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", m.cfg.APIKey)

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return segments{}, transientErrorf("request to ZeroGPU failed: %s", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return segments{}, statusError(resp.StatusCode, domain)
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return segments{}, transientErrorf("failed to read ZeroGPU response: %s", err)
	}

	var envelope apiEnvelope
	if err := jsonutil.Unmarshal(raw, &envelope); err != nil {
		return segments{}, permanentErrorf("failed to decode ZeroGPU response: %s", err)
	}

	payload := envelope.payload()
	if payload == "" {
		return segments{}, permanentErrorf("ZeroGPU response contained no classification payload")
	}

	var result classification
	if err := jsonutil.Unmarshal([]byte(payload), &result); err != nil {
		return segments{}, permanentErrorf("failed to parse classification payload: %s", err)
	}

	return m.filter(result), nil
}

// statusError maps a documented ZeroGPU status onto the retry policy and emits
// a log line at a level matching how actionable the condition is.
func statusError(status int, domain string) apiError {
	switch status {
	case http.StatusBadRequest:
		logger.Warnf("[zerogpu.rtd] ZeroGPU rejected domain %q as a bad request", domain)
		return permanentErrorf("ZeroGPU returned status %d", status)
	case http.StatusUnauthorized, http.StatusForbidden:
		logger.Errorf("[zerogpu.rtd] ZeroGPU returned status %d - check api_key and model access", status)
		return permanentErrorf("ZeroGPU returned status %d", status)
	case statusInsufficientQuota:
		logger.Errorf("[zerogpu.rtd] ZeroGPU returned status %d - insufficient quota", status)
		return permanentErrorf("ZeroGPU returned status %d", status)
	default:
		// 500 and any undocumented status are treated as transient.
		logger.Infof("[zerogpu.rtd] ZeroGPU returned status %d for domain %q", status, domain)
		return transientErrorf("ZeroGPU returned status %d", status)
	}
}

// filter drops low-confidence categories, applies the per-taxonomy cap and
// converts identifiers to the strings ORTB segments require. Taxonomies the
// host has not enabled are skipped so they are never cached or emitted.
func (m *Module) filter(c classification) segments {
	var s segments

	for _, cat := range c.Content.IAB22 {
		if cat.Score < m.cfg.MinScore || cat.ID == 0 {
			continue
		}
		if m.capped(len(s.Content22)) {
			break
		}
		s.Content22 = append(s.Content22, strconv.Itoa(cat.ID))
	}

	if m.cfg.EnrichContent10 {
		for _, cat := range c.Content.IAB10 {
			if cat.Score < m.cfg.MinScore || cat.Code == "" {
				continue
			}
			if m.capped(len(s.Content10)) {
				break
			}
			s.Content10 = append(s.Content10, cat.Code)
		}
	}

	if m.cfg.EnrichUserAudience {
		for _, cat := range c.Audience {
			if cat.Score < m.cfg.MinScore || cat.ID == 0 {
				continue
			}
			if m.capped(len(s.Audience)) {
				break
			}
			s.Audience = append(s.Audience, strconv.Itoa(cat.ID))
		}
	}

	return s
}

// capped reports whether the per-taxonomy segment limit has been reached.
func (m *Module) capped(count int) bool {
	return m.cfg.MaxSegments > 0 && count >= m.cfg.MaxSegments
}
