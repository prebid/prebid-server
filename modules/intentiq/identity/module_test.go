package identity

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/hooks"
	"github.com/prebid/prebid-server/v4/hooks/hookexecution"
	metricsConfig "github.com/prebid/prebid-server/v4/metrics/config"
	"github.com/prebid/prebid-server/v4/modules/moduledeps"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCOPPASkipsEnrichmentAndReportingThroughExecutor(t *testing.T) {
	for _, test := range []struct {
		name      string
		coppa     int8
		wantCalls bool
	}{
		{name: "coppa=1", coppa: 1},
		{name: "coppa=0", coppa: 0, wantCalls: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			s2s, s2sHits := countingServer(t)
			reports, reportHits := countingServer(t)
			built, err := Builder(json.RawMessage(fmt.Sprintf(
				`{"partner_id":"p","api_endpoint":%q,"reports_endpoint":%q}`, s2s.URL, reports.URL,
			)), moduledeps.ModuleDeps{HTTPClient: s2s.Client()})
			require.NoError(t, err)
			module := built.(*Module)
			executor := newExecutor(t, module)

			require.NoError(t, executor.ExecuteProcessedAuctionStage(&openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
				ID: "auction", Regs: &openrtb2.Regs{COPPA: test.coppa}, Device: &openrtb2.Device{IP: "192.0.2.1", UA: "ua", IFA: "ifa"},
			}}))
			executor.ExecuteAuctionResponseStage(&openrtb2.BidResponse{SeatBid: []openrtb2.SeatBid{{
				Seat: "bidder", Bid: []openrtb2.Bid{{ID: "1"}},
			}}})
			require.NoError(t, module.Shutdown())

			if test.wantCalls {
				assert.Positive(t, s2sHits.Load())
				assert.Equal(t, int32(1), reportHits.Load())
				return
			}
			assert.Zero(t, s2sHits.Load())
			assert.Zero(t, reportHits.Load())
		})
	}
}

func countingServer(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	return server, &hits
}

func newExecutor(t *testing.T, module *Module) hookexecution.StageExecutor {
	t.Helper()
	hooksConfig := config.Hooks{Enabled: true}
	require.NoError(t, json.Unmarshal([]byte(`{"endpoints":{"/openrtb2/auction":{"stages":{
		"processed_auction_request":{"groups":[{"timeout":1000,"hook_sequence":[{"module_code":"intentiq.identity","hook_impl_code":"enrich"}]}]},
		"auction_response":{"groups":[{"timeout":1000,"hook_sequence":[{"module_code":"intentiq.identity","hook_impl_code":"report"}]}]}
	}}}}`), &hooksConfig.HostExecutionPlan))
	repo, err := hooks.NewHookRepository(map[string]interface{}{"intentiq.identity": module})
	require.NoError(t, err)
	return hookexecution.NewHookExecutor(hooks.NewExecutionPlanBuilder(hooksConfig, repo),
		hookexecution.EndpointAuction, &metricsConfig.NilMetricsEngine{})
}
