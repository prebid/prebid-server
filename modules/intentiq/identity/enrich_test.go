package identity

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

const eidsBody = `{"data":{"eids":[{"source":"intentiq.com","uids":[{"id":"abc","atype":1}]}]}}`
const emptyBody = `{"data":{"eids":[]}}`

// countMetrics is a counting Metrics stub for the enrich-hook tests.
type countMetrics struct {
	noopMetrics
	apiSuccess     int
	apiError       int
	enriched       int
	eidsNone       int
	skipNoEndpoint int
	tc             []int64
}

func (c *countMetrics) APISuccess(string)                   { c.apiSuccess++ }
func (c *countMetrics) APIError(string)                     { c.apiError++ }
func (c *countMetrics) Enriched(string)                     { c.enriched++ }
func (c *countMetrics) EidsNone(string)                     { c.eidsNone++ }
func (c *countMetrics) SkipNoEndpoint(string)               { c.skipNoEndpoint++ }
func (c *countMetrics) TerminationCause(tc int64, _ string) { c.tc = append(c.tc, tc) }

// capture records the URL/consent header the module sent to the fake IIQ backend.
type capture struct {
	rawQuery string
	consent  string
	hits     int
}

func newServer(t *testing.T, body string, cap *capture) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cap.hits++
		cap.rawQuery = r.URL.RawQuery
		cap.consent = r.Header.Get(gdprConsentHeader)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newModule(endpoint string, client *http.Client, metrics Metrics) *Module {
	return &Module{
		cfg:          Config{APIEndpoint: endpoint, PartnerID: "123", Timeout: 1000},
		httpClient:   client,
		keyExtractor: NewFirstPartyKeyExtractor(10),
		metrics:      metrics,
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
	assert.Equal(t, 1, metrics.eidsNone)
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
	assert.Equal(t, 1, metrics.eidsNone)
	assert.Equal(t, []int64{36}, metrics.tc)
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
	assert.Equal(t, 1, metrics.skipNoEndpoint)
	assert.Equal(t, 0, metrics.apiSuccess)

	// Flow context still set with the known start/auction fields.
	fc := flowFrom(t, res)
	assert.Equal(t, "auc-1", fc.auctionID)
	assert.False(t, fc.start.IsZero())
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
	assert.False(t, fc.start.IsZero())
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

func TestResolveEidsDropsOutOfRangeTerminationCauseMetric(t *testing.T) {
	cap := &capture{}
	metrics := &countMetrics{}
	body := `{"data":{"eids":[]},"tc":120088}`
	m := newModule(newServer(t, body, cap).URL, http.DefaultClient, metrics)

	res, _ := runHook(t, m, &openrtb2.BidRequest{})
	// tc is out of [0,200): no metric recorded, but it still rides in the flow context.
	assert.Empty(t, metrics.tc)
	fc := flowFrom(t, res)
	require.NotNil(t, fc.terminationCause)
	assert.Equal(t, int64(120088), *fc.terminationCause)
}

func TestEidsLenientHelper(t *testing.T) {
	assert.Nil(t, iiqResponse{Data: []byte(`""`)}.eids())
	assert.Nil(t, iiqResponse{Data: nil}.eids())
	assert.Nil(t, iiqResponse{Data: []byte(`  `)}.eids())

	got := iiqResponse{Data: []byte(`{"eids":[{"source":"intentiq.com","uids":[{"id":"x"}]}]}`)}.eids()
	require.Len(t, got, 1)
	assert.Equal(t, "intentiq.com", got[0].Source)
}
