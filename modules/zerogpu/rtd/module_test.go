package rtd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/hooks/hookanalytics"
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/modules/moduledeps"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuilder(t *testing.T) {
	deps := moduledeps.ModuleDeps{HTTPClient: http.DefaultClient}

	built, err := Builder(json.RawMessage(`{"enabled": true, "api_key": "test-key"}`), deps)
	require.NoError(t, err)
	require.IsType(t, &Module{}, built)

	module := built.(*Module)
	assert.Equal(t, DefaultEndpoint, module.cfg.Endpoint)
	assert.Equal(t, DefaultModel, module.cfg.Model)
	assert.Equal(t, "test-key", module.cfg.APIKey)
	require.NotNil(t, module.httpClient)
	assert.Equal(t, http.DefaultClient.Transport, module.httpClient.Transport)
	assert.NotNil(t, module.cache)
}

func TestBuilderWithoutHostHTTPClient(t *testing.T) {
	built, err := Builder(json.RawMessage(`{"api_key": "test-key"}`), moduledeps.ModuleDeps{})
	require.NoError(t, err)

	module := built.(*Module)
	assert.Nil(t, module.httpClient.Transport, "falls back to the default transport")
}

func TestBuilderInvalidConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     string
		wantErr string
	}{
		{"missing api key", `{"enabled": true}`, "api_key is required"},
		{"malformed json", `{"api_key":`, "failed to parse config"},
		{"bad endpoint", `{"api_key":"k","endpoint":"ftp://x/y"}`, "http or https"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			built, err := Builder(json.RawMessage(test.cfg), moduledeps.ModuleDeps{HTTPClient: http.DefaultClient})
			require.Error(t, err)
			assert.Nil(t, built)
			assert.Contains(t, err.Error(), test.wantErr)
		})
	}
}

func TestHandleProcessedAuctionHookEnrichesRequest(t *testing.T) {
	server := newClassifierServer(t, http.StatusOK, responsesEnvelope(classificationJSON))
	defer server.Close()

	module := newTestModule(t, server.URL, nil)
	primeCache(t, module, "coursera.com")

	payload := newPayload(&openrtb2.BidRequest{
		ID:   "req-1",
		Site: &openrtb2.Site{Page: "https://www.coursera.com/learn/python"},
	})

	result, err := module.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{}, payload)
	require.NoError(t, err)
	assert.False(t, result.Reject)
	require.Len(t, result.ChangeSet.Mutations(), 1)

	mutation := result.ChangeSet.Mutations()[0]
	assert.Equal(t, hookstage.MutationAdd, mutation.Type())
	assert.Equal(t, []string{"bidrequest", "site", "content", "data"}, mutation.Key())

	applyMutations(t, result, payload)

	require.NotNil(t, payload.Request.Site.Content)
	require.Len(t, payload.Request.Site.Content.Data, 1)
	data := payload.Request.Site.Content.Data[0]
	assert.Equal(t, DefaultDataProviderName, data.Name)
	assert.JSONEq(t, `{"segtax":6}`, string(data.Ext))
	assert.Equal(t, []openrtb2.Segment{{ID: "132"}, {ID: "148"}}, data.Segment)

	values := activityValues(t, result.AnalyticsTags, hookanalytics.ActivityStatusSuccess)
	assert.Equal(t, "coursera.com", values["domain"])
	assert.Equal(t, 2, values["content_2_2_count"])
}

// TestHandleProcessedAuctionHookNeverBlocks is the core guarantee of the async
// design: an uncached domain returns immediately and unenriched, and only then
// is the cache warmed for subsequent auctions.
func TestHandleProcessedAuctionHookNeverBlocks(t *testing.T) {
	release := make(chan struct{})
	var served int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release // the API is arbitrarily slow
		atomic.AddInt32(&served, 1)
		_, _ = w.Write([]byte(responsesEnvelope(classificationJSON)))
	}))
	defer server.Close()

	module := newTestModule(t, server.URL, nil)
	payload := newPayload(&openrtb2.BidRequest{Site: &openrtb2.Site{Domain: "coursera.com"}})

	start := time.Now()
	result, err := module.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{}, payload)
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Empty(t, result.ChangeSet.Mutations(), "an uncached domain must not enrich")
	assert.Nil(t, payload.Request.Site.Content)
	assert.Less(t, elapsed, 50*time.Millisecond, "the hook must not wait on the API")
	assert.Zero(t, atomic.LoadInt32(&served), "the API has not responded yet")

	values := activityValues(t, result.AnalyticsTags, hookanalytics.ActivityStatusSuccess)
	assert.Equal(t, "domain not yet cached; warming in the background", values["reason"])

	// Let the warm-up finish; the next auction on this domain is enriched.
	close(release)
	awaitWarmUps(module)

	next := newPayload(&openrtb2.BidRequest{Site: &openrtb2.Site{Domain: "coursera.com"}})
	result, err = module.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{}, next)
	require.NoError(t, err)
	require.Len(t, result.ChangeSet.Mutations(), 1)
}

func TestHandleProcessedAuctionHookFailsOpen(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{"bad request", http.StatusBadRequest, ``},
		{"unauthorized", http.StatusUnauthorized, ``},
		{"forbidden", http.StatusForbidden, ``},
		{"insufficient quota", statusInsufficientQuota, ``},
		{"server error", http.StatusInternalServerError, ``},
		{"unparseable body", http.StatusOK, `garbage`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := newClassifierServer(t, test.status, test.body)
			defer server.Close()

			module := newTestModule(t, server.URL, nil)
			payload := newPayload(&openrtb2.BidRequest{Site: &openrtb2.Site{Domain: "coursera.com"}})

			// First auction: uncached, so it schedules a warm-up that fails.
			result, err := module.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{}, payload)
			require.NoError(t, err)
			assert.False(t, result.Reject)
			assert.Empty(t, result.ChangeSet.Mutations())

			awaitWarmUps(module)

			// Second auction: the failure is cached, so the auction still
			// proceeds cleanly with no mutation and no error.
			next := newPayload(&openrtb2.BidRequest{Site: &openrtb2.Site{Domain: "coursera.com"}})
			result, err = module.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{}, next)
			require.NoError(t, err)
			assert.False(t, result.Reject)
			assert.Empty(t, result.ChangeSet.Mutations())
			assert.Nil(t, next.Request.Site.Content)

			values := activityValues(t, result.AnalyticsTags, hookanalytics.ActivityStatusSuccess)
			assert.Equal(t, "no categories available for this domain", values["reason"])
		})
	}
}

func TestHandleProcessedAuctionHookIgnoresHookContextCancellation(t *testing.T) {
	server := newClassifierServer(t, http.StatusOK, responsesEnvelope(classificationJSON))
	defer server.Close()

	module := newTestModule(t, server.URL, nil)
	primeCache(t, module, "coursera.com")

	payload := newPayload(&openrtb2.BidRequest{Site: &openrtb2.Site{Domain: "coursera.com"}})

	// The hook does no I/O, so even an already-cancelled context - which is
	// what a zero group timeout produces - must still enrich.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := module.HandleProcessedAuctionHook(ctx, hookstage.ModuleInvocationContext{}, payload)

	require.NoError(t, err)
	assert.Len(t, result.ChangeSet.Mutations(), 1)
}

// TestWarmSurvivesHookContextCancellation guards the reason warm-ups use the
// module's own context: a warm-up tied to the hook context would be killed at
// the group timeout and the domain would never become cached.
func TestWarmSurvivesHookContextCancellation(t *testing.T) {
	server := newClassifierServer(t, http.StatusOK, responsesEnvelope(classificationJSON))
	defer server.Close()

	module := newTestModule(t, server.URL, nil)
	payload := newPayload(&openrtb2.BidRequest{Site: &openrtb2.Site{Domain: "coursera.com"}})

	ctx, cancel := context.WithCancel(context.Background())
	_, err := module.HandleProcessedAuctionHook(ctx, hookstage.ModuleInvocationContext{}, payload)
	require.NoError(t, err)
	cancel() // the auction ends immediately, as it would at a group timeout

	awaitWarmUps(module)

	segs, cached := module.lookup("coursera.com")
	assert.True(t, cached)
	assert.Equal(t, []string{"132", "148"}, segs.Content22)
}

func TestHandleProcessedAuctionHookSkips(t *testing.T) {
	tests := []struct {
		name      string
		miCtx     hookstage.ModuleInvocationContext
		payload   hookstage.ProcessedAuctionRequestPayload
		mutate    func(*Config)
		wantTag   bool
		wantValue string
	}{
		{
			name:    "nil request wrapper",
			payload: hookstage.ProcessedAuctionRequestPayload{},
		},
		{
			name:    "nil bid request",
			payload: hookstage.ProcessedAuctionRequestPayload{Request: &openrtb_ext.RequestWrapper{}},
		},
		{
			name:    "account not on allow list",
			miCtx:   hookstage.ModuleInvocationContext{AccountID: "9999"},
			payload: newPayload(&openrtb2.BidRequest{Site: &openrtb2.Site{Domain: "coursera.com"}}),
			mutate:  func(c *Config) { c.AccountFilter.AllowList = []string{"1001"} },
		},
		{
			name:      "no resolvable domain",
			payload:   newPayload(&openrtb2.BidRequest{Site: &openrtb2.Site{Page: "http://localhost/test"}}),
			wantTag:   true,
			wantValue: "no domain available on the request",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Error("the classifier must not be called")
			}))
			defer server.Close()

			module := newTestModule(t, server.URL, test.mutate)
			result, err := module.HandleProcessedAuctionHook(context.Background(), test.miCtx, test.payload)

			require.NoError(t, err)
			assert.Empty(t, result.ChangeSet.Mutations())

			if test.wantTag {
				values := activityValues(t, result.AnalyticsTags, hookanalytics.ActivityStatusSuccess)
				assert.Equal(t, test.wantValue, values["reason"])
			} else {
				assert.Empty(t, result.AnalyticsTags.Activities)
			}
		})
	}
}

func TestHandleProcessedAuctionHookAllowsListedAccount(t *testing.T) {
	server := newClassifierServer(t, http.StatusOK, responsesEnvelope(classificationJSON))
	defer server.Close()

	module := newTestModule(t, server.URL, func(c *Config) {
		c.AccountFilter.AllowList = []string{"1001"}
	})
	primeCache(t, module, "coursera.com")

	payload := newPayload(&openrtb2.BidRequest{Site: &openrtb2.Site{Domain: "coursera.com"}})

	result, err := module.HandleProcessedAuctionHook(
		context.Background(),
		hookstage.ModuleInvocationContext{AccountID: "1001"},
		payload,
	)
	require.NoError(t, err)
	assert.Len(t, result.ChangeSet.Mutations(), 1)
}

func TestHandleProcessedAuctionHookNoQualifyingSegments(t *testing.T) {
	server := newClassifierServer(t, http.StatusOK, responsesEnvelope(classificationJSON))
	defer server.Close()

	// A min_score above every score in the fixture filters everything out, so
	// the domain caches as an empty - but present - result.
	module := newTestModule(t, server.URL, func(c *Config) { c.MinScore = 0.999 })
	primeCache(t, module, "coursera.com")

	payload := newPayload(&openrtb2.BidRequest{Site: &openrtb2.Site{Domain: "coursera.com"}})

	result, err := module.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{}, payload)
	require.NoError(t, err)

	assert.Empty(t, result.ChangeSet.Mutations())
	assert.Nil(t, payload.Request.Site.Content)

	values := activityValues(t, result.AnalyticsTags, hookanalytics.ActivityStatusSuccess)
	assert.Equal(t, "no categories available for this domain", values["reason"])
}

func TestHandleProcessedAuctionHookAllTaxonomies(t *testing.T) {
	server := newClassifierServer(t, http.StatusOK, responsesEnvelope(classificationJSON))
	defer server.Close()

	module := newTestModule(t, server.URL, func(c *Config) {
		c.EnrichContent10 = true
		c.EnrichUserAudience = true
	})
	primeCache(t, module, "coursera.com")

	payload := newPayload(&openrtb2.BidRequest{Site: &openrtb2.Site{Domain: "coursera.com"}})

	result, err := module.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{}, payload)
	require.NoError(t, err)
	applyMutations(t, result, payload)

	require.Len(t, payload.Request.Site.Content.Data, 2)
	require.Len(t, payload.Request.User.Data, 1)

	values := activityValues(t, result.AnalyticsTags, hookanalytics.ActivityStatusSuccess)
	assert.Equal(t, 2, values["content_2_2_count"])
	assert.Equal(t, 2, values["content_1_0_count"])
	assert.Equal(t, 2, values["audience_count"])
}

// TestHandleProcessedAuctionHookWithFPDBidders covers the interaction with
// firstpartydata.ExtractOpenRtbGlobalFPD, which strips and redistributes
// site.content.data per bidder only when ext.prebid.data.bidders is set. The
// module's job is to inject the same segments either way; core decides where
// they end up.
func TestHandleProcessedAuctionHookWithFPDBidders(t *testing.T) {
	tests := []struct {
		name string
		ext  json.RawMessage
	}{
		{"without ext.prebid.data.bidders", nil},
		{"with ext.prebid.data.bidders", json.RawMessage(`{"prebid":{"data":{"bidders":["appnexus"]}}}`)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := newClassifierServer(t, http.StatusOK, responsesEnvelope(classificationJSON))
			defer server.Close()

			module := newTestModule(t, server.URL, nil)
			primeCache(t, module, "coursera.com")

			payload := newPayload(&openrtb2.BidRequest{
				ID:   "req-1",
				Site: &openrtb2.Site{Domain: "coursera.com"},
				Ext:  test.ext,
			})

			result, err := module.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{}, payload)
			require.NoError(t, err)
			applyMutations(t, result, payload)

			require.Len(t, payload.Request.Site.Content.Data, 1)
			assert.JSONEq(t, `{"segtax":6}`, string(payload.Request.Site.Content.Data[0].Ext))

			// Rebuilding must preserve the injected segments.
			require.NoError(t, payload.Request.RebuildRequest())
			require.Len(t, payload.Request.Site.Content.Data, 1)
		})
	}
}

func newPayload(request *openrtb2.BidRequest) hookstage.ProcessedAuctionRequestPayload {
	return hookstage.ProcessedAuctionRequestPayload{
		Request: &openrtb_ext.RequestWrapper{BidRequest: request},
	}
}

// applyMutations runs the staged mutations the way the hook executor does.
func applyMutations(t *testing.T, result hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload], payload hookstage.ProcessedAuctionRequestPayload) {
	t.Helper()
	for _, mutation := range result.ChangeSet.Mutations() {
		_, err := mutation.Apply(payload)
		require.NoError(t, err)
	}
}

// activityValues asserts a single analytics activity with the expected status
// and returns its values map.
func activityValues(t *testing.T, tags hookanalytics.Analytics, want hookanalytics.ActivityStatus) map[string]interface{} {
	t.Helper()
	require.Len(t, tags.Activities, 1)
	activity := tags.Activities[0]
	assert.Equal(t, analyticsActivity, activity.Name)
	assert.Equal(t, want, activity.Status)
	require.Len(t, activity.Results, 1)
	return activity.Results[0].Values
}
