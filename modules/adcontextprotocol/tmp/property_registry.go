package tmp

import (
	"container/list"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/adcontextprotocol/adcp-go/tmproto"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

// PropertyRecord is the subset of a registry property record the module needs.
// The registry may return more fields — we ignore what we don't use.
type PropertyRecord struct {
	PropertyRID  string               `json:"property_rid"`
	PropertyID   string               `json:"property_id"`
	PropertyType tmproto.PropertyType `json:"property_type"`
	Domain       string               `json:"domain"`
}

// registryResponse mirrors the resolve endpoint's JSON envelope. The spec at
// agenticadvertising.org returns either a single property or a "not found"
// signal — modeled here so callers can distinguish "no such domain" from an
// upstream error.
type registryResponse struct {
	Property *PropertyRecord `json:"property"`
	Found    *bool           `json:"found,omitempty"`
}

// propertyResolver resolves site.domain / app.bundle → PropertyRecord with an
// in-memory expirable LRU cache. The first call from a cold domain may miss
// the auction's timeout budget; subsequent calls hit the cache.
type propertyResolver struct {
	cfg    PropertyRegistryConfig
	http   *http.Client
	mu     sync.Mutex
	order  *list.List
	items  map[string]*list.Element
	single singleflight
}

type cacheEntry struct {
	key     string
	record  *PropertyRecord // nil = negative cache (domain not registered)
	expires time.Time
}

func newPropertyResolver(cfg PropertyRegistryConfig, transport http.RoundTripper) *propertyResolver {
	return &propertyResolver{
		cfg: cfg,
		http: &http.Client{
			Timeout:   time.Duration(cfg.TimeoutMs) * time.Millisecond,
			Transport: transport,
		},
		order: list.New(),
		items: make(map[string]*list.Element),
	}
}

// maxDomainKeyLen caps the length of a cache key / registry query
// parameter derived from the bid request's site.domain or app.bundle.
// 253 is the RFC 1035 max length of a fully-qualified domain name; app
// bundle IDs also fit comfortably below it. Longer inputs are rejected
// so a hostile bid request cannot inflate the LRU or amplify to the
// registry with garbage keys.
const maxDomainKeyLen = 253

// Resolve looks up a property by canonical domain (site) or bundle (app).
// Returns (record, true, nil) on hit, (nil, false, nil) on cached negative,
// (nil, false, err) on registry error.
func (p *propertyResolver) Resolve(ctx context.Context, domain string) (*PropertyRecord, bool, error) {
	key := strings.ToLower(strings.TrimSpace(domain))
	if key == "" {
		return nil, false, errors.New("empty domain")
	}
	if len(key) > maxDomainKeyLen {
		return nil, false, fmt.Errorf("domain length %d exceeds cap %d", len(key), maxDomainKeyLen)
	}
	if !isValidDomainOrBundle(key) {
		return nil, false, errors.New("domain contains invalid characters")
	}

	if rec, ok, fresh := p.cacheGet(key); fresh {
		return rec, ok, nil
	}

	// Single-flight: collapse concurrent misses for the same domain onto one
	// HTTP call. The leader's fetch runs in a fresh context so followers are
	// not tied to whichever caller happened to arrive first — if that caller's
	// auction times out, the leader keeps going and future callers get the
	// result from cache.
	rec, err := p.single.do(key, func() (*PropertyRecord, error) {
		leaderCtx, cancel := context.WithTimeout(context.Background(), time.Duration(p.cfg.TimeoutMs)*time.Millisecond)
		defer cancel()
		return p.fetch(leaderCtx, key)
	})
	if err != nil {
		return nil, false, err
	}
	if rec == nil {
		p.cachePut(key, nil, time.Duration(p.cfg.NegativeCacheTTLSeconds)*time.Second)
		return nil, false, nil
	}
	p.cachePut(key, rec, time.Duration(p.cfg.CacheTTLSeconds)*time.Second)
	return rec, true, nil
}

func (p *propertyResolver) fetch(ctx context.Context, domain string) (*PropertyRecord, error) {
	q := url.Values{}
	q.Set("domain", domain)
	fullURL := p.cfg.Endpoint
	if strings.Contains(fullURL, "?") {
		fullURL += "&" + q.Encode()
	} else {
		fullURL += "?" + q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, err
	}
	if p.cfg.AuthBearer != "" {
		req.Header.Set("Authorization", "Bearer "+p.cfg.AuthBearer)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := p.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		// Drain and close so keep-alive can reuse the connection.
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
		_ = resp.Body.Close()
	}()

	switch resp.StatusCode {
	case http.StatusOK:
		// 64 KiB is generous for a single property record.
		raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
		if err != nil {
			return nil, fmt.Errorf("registry read: %w", err)
		}
		// Registry implementations vary. Try the wrapped
		// {"property": {...}} envelope first; if that yields no
		// property_rid, try decoding the payload as a bare
		// PropertyRecord — some deployments (including
		// agenticadvertising.org's /api/properties/resolve) return the
		// record directly, not nested.
		var body registryResponse
		if err := jsonutil.Unmarshal(raw, &body); err != nil {
			return nil, fmt.Errorf("registry decode: %w", err)
		}
		if body.Property != nil && body.Property.PropertyRID != "" {
			return body.Property, nil
		}
		if body.Found != nil && !*body.Found {
			return nil, nil
		}
		var bare PropertyRecord
		if err := jsonutil.Unmarshal(raw, &bare); err == nil && bare.PropertyRID != "" {
			return &bare, nil
		}
		return nil, nil
	case http.StatusNotFound:
		return nil, nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, fmt.Errorf("registry auth failed: status %d", resp.StatusCode)
	default:
		return nil, fmt.Errorf("registry status %d", resp.StatusCode)
	}
}

func (p *propertyResolver) cacheGet(key string) (*PropertyRecord, bool, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	el, ok := p.items[key]
	if !ok {
		return nil, false, false
	}
	entry := el.Value.(*cacheEntry)
	if time.Now().After(entry.expires) {
		p.order.Remove(el)
		delete(p.items, key)
		return nil, false, false
	}
	p.order.MoveToFront(el)
	return entry.record, entry.record != nil, true
}

func (p *propertyResolver) cachePut(key string, rec *PropertyRecord, ttl time.Duration) {
	if ttl <= 0 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if el, ok := p.items[key]; ok {
		entry := el.Value.(*cacheEntry)
		entry.record = rec
		entry.expires = time.Now().Add(ttl)
		p.order.MoveToFront(el)
		return
	}
	entry := &cacheEntry{key: key, record: rec, expires: time.Now().Add(ttl)}
	el := p.order.PushFront(entry)
	p.items[key] = el
	for p.order.Len() > p.cfg.CacheSize {
		back := p.order.Back()
		if back == nil {
			break
		}
		p.order.Remove(back)
		delete(p.items, back.Value.(*cacheEntry).key)
	}
}

// singleflight collapses concurrent fetches for the same key onto one call.
// A tiny local implementation avoids pulling golang.org/x/sync just for this.
type singleflight struct {
	mu    sync.Mutex
	calls map[string]*sfCall
}

type sfCall struct {
	wg  sync.WaitGroup
	rec *PropertyRecord
	err error
}

func (s *singleflight) do(key string, fn func() (*PropertyRecord, error)) (*PropertyRecord, error) {
	s.mu.Lock()
	if s.calls == nil {
		s.calls = make(map[string]*sfCall)
	}
	if c, ok := s.calls[key]; ok {
		s.mu.Unlock()
		c.wg.Wait()
		return c.rec, c.err
	}
	c := &sfCall{}
	c.wg.Add(1)
	s.calls[key] = c
	s.mu.Unlock()

	c.rec, c.err = fn()
	c.wg.Done()

	s.mu.Lock()
	delete(s.calls, key)
	s.mu.Unlock()
	return c.rec, c.err
}

// isValidDomainOrBundle returns true when the input contains only
// characters plausible in a DNS name or an app-store bundle identifier.
// Rejects whitespace, control chars, URL delimiters, and quoting —
// anything a hostile bid request might inject to smuggle payloads
// through the registry lookup or bloat the LRU key space with garbage.
// Intentionally permissive on shape: no length-per-label check, no dot
// requirement — DNS labels can be single-character and bundle ids like
// `com.example.app` and `com.example-app.v2` are both valid.
func isValidDomainOrBundle(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z':
		case c >= '0' && c <= '9':
		case c == '.' || c == '-' || c == '_':
		default:
			return false
		}
	}
	return true
}
