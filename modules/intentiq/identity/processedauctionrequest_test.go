package identity

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/cache"
	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/cache/provider/memory"
	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/enrichment"
	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

const eidsBody = `{"data":{"eids":[{"source":"intentiq.com","uids":[{"id":"abc","atype":1}]}]}}`
const emptyBody = `{"data":{"eids":[]}}`

// countMetrics is a counting Metrics stub for the enrich-hook tests.
type countMetrics struct {
	metrics.Noop
	requests   int
	apiSuccess int
	apiError   int
	enriched   int
	// apiErrors and notEnriched tally label values, keyed by reason.
	apiErrors   map[string]string // reason -> status_code
	notEnriched map[string]int
	lookups     map[string]int // "<result>:<layer>"
}

func (c *countMetrics) Requests(string)   { c.requests++ }
func (c *countMetrics) APISuccess(string) { c.apiSuccess++ }
func (c *countMetrics) Enriched(string)   { c.enriched++ }

func (c *countMetrics) APIError(_, reason, statusCode string) {
	c.apiError++
	if c.apiErrors == nil {
		c.apiErrors = map[string]string{}
	}
	c.apiErrors[reason] = statusCode
}

func (c *countMetrics) NotEnriched(reason, _ string) {
	if c.notEnriched == nil {
		c.notEnriched = map[string]int{}
	}
	c.notEnriched[reason]++
}

func (c *countMetrics) CacheLookup(result, layer, _ string) {
	if c.lookups == nil {
		c.lookups = map[string]int{}
	}
	c.lookups[result+":"+layer]++
}

// capture records the URL/consent header the module sent to the fake IIQ backend.
type capture struct {
	rawQuery string
	consent  string
	hits     int
}

func newServer(t testing.TB, body string, cap *capture) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cap.hits++
		cap.rawQuery = r.URL.RawQuery
		cap.consent = r.Header.Get(enrichment.GDPRConsentHeader)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newModule(endpoint string, client *http.Client, mtr metrics.Metrics) *Module {
	return &Module{
		cfg:          Config{APIEndpoint: endpoint, PartnerID: "123", Timeout: 1000},
		httpClient:   client,
		api:          enrichment.NewClient(client),
		keyExtractor: NewFirstPartyKeyExtractor(10),
		metrics:      mtr,
		cache:        nil, // no-cache direct path
	}
}

func runHook(t *testing.T, m *Module, req *openrtb2.BidRequest) (hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload], hookstage.ProcessedAuctionRequestPayload) {
	t.Helper()
	payload := hookstage.ProcessedAuctionRequestPayload{Request: &openrtb_ext.RequestWrapper{BidRequest: req}}
	res, err := m.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{}, payload)
	require.NoError(t, err)
	return res, payload
}

// applyMutations runs every mutation in the change set against the payload (as the framework would).
func applyMutations(t *testing.T, res hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload], payload hookstage.ProcessedAuctionRequestPayload) hookstage.ProcessedAuctionRequestPayload {
	t.Helper()
	p := payload
	for _, mut := range res.ChangeSet.Mutations() {
		var err error
		p, err = mut.Apply(p)
		require.NoError(t, err)
	}
	return p
}

func flowFrom(t *testing.T, res hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload]) flowContext {
	t.Helper()
	fc, ok := getFlowContext(res.ModuleContext)
	require.True(t, ok, "flow context must be set in ModuleContext")
	return fc
}

func TestEnrichAppendsResolvedEids(t *testing.T) {
	cap := &capture{}
	m := newModule(newServer(t, eidsBody, cap).URL, http.DefaultClient, &countMetrics{})

	res, payload := runHook(t, m, &openrtb2.BidRequest{})
	require.Len(t, res.ChangeSet.Mutations(), 1)

	updated := applyMutations(t, res, payload)
	eids := updated.Request.User.EIDs
	require.Len(t, eids, 1)
	assert.Equal(t, "intentiq.com", eids[0].Source)
	assert.Equal(t, "abc", eids[0].UIDs[0].ID)
}

func TestEnrichAppendsAfterExistingUserEids(t *testing.T) {
	cap := &capture{}
	m := newModule(newServer(t, eidsBody, cap).URL, http.DefaultClient, &countMetrics{})

	req := &openrtb2.BidRequest{User: &openrtb2.User{EIDs: []openrtb2.EID{
		{Source: "pubcid.org", UIDs: []openrtb2.UID{{ID: "existing-uid"}}},
	}}}
	res, payload := runHook(t, m, req)
	updated := applyMutations(t, res, payload)

	require.Len(t, updated.Request.User.EIDs, 2)
	assert.Equal(t, "pubcid.org", updated.Request.User.EIDs[0].Source)
	assert.Equal(t, "intentiq.com", updated.Request.User.EIDs[1].Source)
}

func TestEnrichEidsNoneWhenEmptyData(t *testing.T) {
	cap := &capture{}
	metrics := &countMetrics{}
	m := newModule(newServer(t, emptyBody, cap).URL, http.DefaultClient, metrics)

	res, _ := runHook(t, m, &openrtb2.BidRequest{})
	assert.Empty(t, res.ChangeSet.Mutations())
	assert.Equal(t, 1, metrics.notEnriched["no_ids"])
	assert.Equal(t, 1, metrics.apiSuccess)
	assert.Equal(t, 0, metrics.enriched)
}

// Lenient parse: data:"" (empty string, not an object) is a valid empty response, not an API error.
func TestEnrichLenientEmptyStringData(t *testing.T) {
	cap := &capture{}
	metrics := &countMetrics{}
	body := `{"adt":4,"ct":2,"data":"","cttl":600000,"tc":36}`
	m := newModule(newServer(t, body, cap).URL, http.DefaultClient, metrics)

	res, _ := runHook(t, m, &openrtb2.BidRequest{})
	assert.Empty(t, res.ChangeSet.Mutations())
	assert.Equal(t, 1, metrics.apiSuccess)
	assert.Equal(t, 0, metrics.apiError)
	assert.Equal(t, 1, metrics.notEnriched["no_ids"])
}

func TestEnrichSendsUrlParamsAndConsentHeader(t *testing.T) {
	cap := &capture{}
	m := newModule("", http.DefaultClient, &countMetrics{})
	srv := newServer(t, emptyBody, cap)
	m.cfg.APIEndpoint = srv.URL

	gdpr := int8(1)
	req := &openrtb2.BidRequest{
		Device: &openrtb2.Device{IP: "1.2.3.4", UA: "Mozilla/5.0 (iPhone)"},
		Regs:   &openrtb2.Regs{GDPR: &gdpr, USPrivacy: "1YNN"},
		User:   &openrtb2.User{Consent: "CO-TCF-STRING"},
	}
	runHook(t, m, req)

	require.Equal(t, 1, cap.hits)
	// Raw query preserves the encoding (%20 for space).
	assert.Contains(t, cap.rawQuery, "at=39")
	assert.Contains(t, cap.rawQuery, "dpi=123")
	assert.Contains(t, cap.rawQuery, "source=pbgo")
	assert.Contains(t, cap.rawQuery, "uas=Mozilla%2F5.0%20%28iPhone%29")
	assert.Contains(t, cap.rawQuery, "gdpr=1")
	assert.Contains(t, cap.rawQuery, "us_privacy=1YNN")

	parsed, err := url.ParseQuery(cap.rawQuery)
	require.NoError(t, err)
	assert.Equal(t, "1.2.3.4", parsed.Get("ip"))

	// Consent travels in the header, not the query.
	assert.Equal(t, "CO-TCF-STRING", cap.consent)
	assert.NotContains(t, cap.rawQuery, "CO-TCF-STRING")
}

func TestEnrichNoConsentHeaderWhenAbsent(t *testing.T) {
	cap := &capture{}
	m := newModule(newServer(t, emptyBody, cap).URL, http.DefaultClient, &countMetrics{})

	runHook(t, m, &openrtb2.BidRequest{})
	assert.Equal(t, "", cap.consent)
}

func TestEnrichNoEndpointSkip(t *testing.T) {
	metrics := &countMetrics{}
	m := newModule("", http.DefaultClient, metrics)

	res, payload := runHook(t, m, &openrtb2.BidRequest{ID: "auc-1"})
	assert.Empty(t, res.ChangeSet.Mutations())
	assert.Equal(t, 1, metrics.notEnriched["no_endpoint"])
	assert.Equal(t, 0, metrics.apiSuccess)

	// Flow context still set with the known auction fields.
	fc := flowFrom(t, res)
	assert.Equal(t, "auc-1", fc.auctionID)
	// No mutation applied.
	updated := applyMutations(t, res, payload)
	assert.Nil(t, updated.Request.User)
}

func TestEnrichUpstreamErrorIsNoOp(t *testing.T) {
	// Server that resets the connection / returns error: point the client at a closed server.
	cap := &capture{}
	srv := newServer(t, emptyBody, cap)
	badURL := srv.URL
	srv.Close() // force a connection error

	metrics := &countMetrics{}
	m := newModule(badURL, http.DefaultClient, metrics)

	res, payload := runHook(t, m, &openrtb2.BidRequest{})
	assert.Empty(t, res.ChangeSet.Mutations())
	assert.Equal(t, 1, metrics.apiError)
	assert.Equal(t, 0, metrics.apiSuccess)

	updated := applyMutations(t, res, payload)
	assert.Nil(t, updated.Request.User)

	// Flow context is still set (abTestUuid/tc unknown on error).
	fc := flowFrom(t, res)
	assert.Empty(t, fc.abTestUUID)
	assert.Nil(t, fc.terminationCause)
}

func TestEnrichFlowContextCarriesTerminationCauseAndRequestFields(t *testing.T) {
	cap := &capture{}
	body := `{"data":{"eids":[{"source":"intentiq.com","uids":[{"id":"abc"}]}]},"abTestUuid":"ab-1","tc":5}`
	m := newModule(newServer(t, body, cap).URL, http.DefaultClient, &countMetrics{})

	req := &openrtb2.BidRequest{
		ID:     "auction-9",
		Site:   &openrtb2.Site{Domain: "example.com"},
		Device: &openrtb2.Device{IP: "9.9.9.9", UA: "UA-X"},
	}
	res, _ := runHook(t, m, req)

	fc := flowFrom(t, res)
	assert.Equal(t, "auction-9", fc.auctionID)
	assert.Equal(t, "example.com", fc.ref)
	assert.Equal(t, "9.9.9.9", fc.ip)
	assert.Equal(t, "UA-X", fc.ua)
	assert.Equal(t, "ab-1", fc.abTestUUID)
	require.NotNil(t, fc.terminationCause)
	assert.Equal(t, int64(5), *fc.terminationCause)
}

func TestEnrichFlowContextIpFallsBackToIpv6(t *testing.T) {
	cap := &capture{}
	m := newModule(newServer(t, emptyBody, cap).URL, http.DefaultClient, &countMetrics{})

	req := &openrtb2.BidRequest{Device: &openrtb2.Device{IPv6: "2001:db8::1"}}
	res, _ := runHook(t, m, req)

	assert.Equal(t, "2001:db8::1", flowFrom(t, res).ip)
}

// tc is no longer a metric, but it still rides in the flow context for the impression report.
func TestResolveEidsCarriesTerminationCauseInFlowContext(t *testing.T) {
	cap := &capture{}
	metrics := &countMetrics{}
	body := `{"data":{"eids":[]},"tc":120088}`
	m := newModule(newServer(t, body, cap).URL, http.DefaultClient, metrics)

	res, _ := runHook(t, m, &openrtb2.BidRequest{})
	fc := flowFrom(t, res)
	require.NotNil(t, fc.terminationCause)
	assert.Equal(t, int64(120088), *fc.terminationCause)
}

// A non-2xx is a failed resolution, not a success: without the status check a 429 or 500 carrying a
// JSON-ish body was counted as api_success and silently produced no eids.
func TestEnrichNon2xxCountsAsClassifiedAPIError(t *testing.T) {
	for _, tc := range []struct {
		name       string
		status     int
		wantReason string
	}{
		{"too many requests", http.StatusTooManyRequests, "status"},
		{"server error", http.StatusInternalServerError, "status"},
		{"bad request", http.StatusBadRequest, "status"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(`{"data":{"eids":[]}}`))
			}))
			defer srv.Close()

			mtr := &countMetrics{}
			m := newModule(srv.URL, http.DefaultClient, mtr)
			res, _ := runHook(t, m, &openrtb2.BidRequest{})

			// Fail open: the auction proceeds, just without enrichment.
			assert.Empty(t, res.ChangeSet.Mutations())
			assert.Equal(t, 0, mtr.apiSuccess, "a non-2xx must not count as success")
			assert.Equal(t, 1, mtr.apiError)
			assert.Equal(t, strconv.Itoa(tc.status), mtr.apiErrors[tc.wantReason])
		})
	}
}

// newCachedModule builds a module on the cache path; newModule takes the direct one.
func newCachedModule(t testing.TB, endpoint string, mtr metrics.Metrics) *Module {
	t.Helper()
	store := memory.New()
	t.Cleanup(func() { _ = store.Close() })

	// Defaults carry the TTL ceilings, and EffectiveTTL is min(ttl, ceiling): a zero ceiling would
	// expire every entry on write.
	cfg := defaultConfig()
	cfg.APIEndpoint = endpoint
	cfg.PartnerID = "123"
	cfg.Timeout = 1000
	cfg.Cache.Enabled = true

	m := newModule(endpoint, http.DefaultClient, mtr)
	m.cfg = cfg
	m.keyExtractor = NewFirstPartyKeyExtractor(cfg.Cache.MaxKeys)
	m.cache = cache.NewIdentityCache(cfg.Cache.MaxSize, cfg.Cache.TTLPolicy(), store, &noopCacheMetrics{})
	return m
}

type noopCacheMetrics struct{}

func (noopCacheMetrics) CacheLookup(result, layer, dpi string)         {}
func (noopCacheMetrics) L1PutError()                                   {}
func (noopCacheMetrics) L2GetLatency(d time.Duration)                  {}
func (noopCacheMetrics) L2PutLatency(d time.Duration)                  {}
func (noopCacheMetrics) L2Request(op, result string)                   {}
func (noopCacheMetrics) RegisterL1Gauges(size, evictions func() int64) {}

func cachedReq() *openrtb2.BidRequest {
	return &openrtb2.BidRequest{
		ID:     "auc-cached",
		Site:   &openrtb2.Site{Domain: "example.com"},
		Device: &openrtb2.Device{IP: "1.2.3.4", UA: "UA-cached"},
	}
}

// Without this the impression report only carried them on the one request that made the live call.
func TestEnrichCacheHitKeepsAbTestUUIDAndTc(t *testing.T) {
	cap := &capture{}
	body := `{"data":{"eids":[{"source":"intentiq.com","uids":[{"id":"abc"}]}]},"abTestUuid":"ab-1","tc":5}`
	m := newCachedModule(t, newServer(t, body, cap).URL, &countMetrics{})

	res1, _ := runHook(t, m, cachedReq())
	require.Equal(t, "ab-1", flowFrom(t, res1).abTestUUID)
	require.Equal(t, 1, cap.hits)

	res2, _ := runHook(t, m, cachedReq())
	assert.Equal(t, 1, cap.hits, "served from cache")

	fc := flowFrom(t, res2)
	assert.Equal(t, "ab-1", fc.abTestUUID)
	require.NotNil(t, fc.terminationCause)
	assert.Equal(t, int64(5), *fc.terminationCause)
}

// A terminated id is cached as a negative entry; tc is the only record of why.
func TestEnrichNegativeCacheKeepsAbTestUUIDAndTc(t *testing.T) {
	cap := &capture{}
	body := `{"data":{"eids":[]},"abTestUuid":"ab-2","tc":120088}`
	m := newCachedModule(t, newServer(t, body, cap).URL, &countMetrics{})

	runHook(t, m, cachedReq())
	require.Equal(t, 1, cap.hits)

	res2, _ := runHook(t, m, cachedReq())
	assert.Equal(t, 1, cap.hits, "negative entry suppresses the upstream call")

	fc := flowFrom(t, res2)
	assert.Equal(t, "ab-2", fc.abTestUUID)
	require.NotNil(t, fc.terminationCause)
	assert.Equal(t, int64(120088), *fc.terminationCause)
}
