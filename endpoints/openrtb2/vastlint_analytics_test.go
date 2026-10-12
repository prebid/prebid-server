package openrtb2

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"
	analyticsBuild "github.com/prebid/prebid-server/v4/analytics/build"
	"github.com/prebid/prebid-server/v4/analytics/vastlint/api"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/exchange"
	"github.com/prebid/prebid-server/v4/hooks"
	metricsConfig "github.com/prebid/prebid-server/v4/metrics/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/ortb"
	"github.com/prebid/prebid-server/v4/stored_requests/backends/empty_fetcher"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type vastlintResponseExchange struct{ markup string }

func (e vastlintResponseExchange) HoldAuction(_ context.Context, r *exchange.AuctionRequest, _ *exchange.DebugLog) (*exchange.AuctionResponse, error) {
	return &exchange.AuctionResponse{BidResponse: &openrtb2.BidResponse{ID: r.BidRequestWrapper.ID, Cur: "USD", SeatBid: []openrtb2.SeatBid{{Seat: "appnexus", Bid: []openrtb2.Bid{{ID: "bid-1", ImpID: "imp-1", Price: 1.23, AdM: e.markup, Ext: json.RawMessage(`{"prebid":{"type":"video"}}`)}}}}}}, nil
}

type slowVastlintServer struct {
	api.UnimplementedVastlintServiceServer
	received chan string
}

func (s slowVastlintServer) Validate(ctx context.Context, r *api.ValidateRequest) (*api.ValidateResponse, error) {
	s.received <- r.Document
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestVastlintAnalyticsPreservesEndpointResponse(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer()
	received := make(chan string, 2)
	api.RegisterVastlintServiceServer(server, slowVastlintServer{received: received})
	go server.Serve(listener)
	defer server.Stop()
	const markup = `  <VAST version="2.0"><Ad>malformed`
	const request = `{"id":"request-1","site":{"page":"https://example.com","publisher":{"id":"host-account"}},"imp":[{"id":"imp-1","video":{"mimes":["video/mp4"],"protocols":[2],"minduration":1,"maxduration":30},"ext":{"appnexus":{"placementId":1}}}],"tmax":500}`
	var original []byte
	for _, enabled := range []bool{false, true} {
		cfg := &config.Configuration{MaxRequestSize: maxSize}
		require.NoError(t, cfg.MarshalAccountDefaults())
		runner := analyticsBuild.New(&config.Analytics{Vastlint: config.VastlintAnalytics{Enabled: enabled, Endpoint: listener.Addr().String(), AllowInsecure: true, Accounts: []string{"host-account"}, SamplePercent: 100, TimeoutMS: 5000, MaxAgeMS: 10000}})
		endpoint, err := NewEndpoint(fakeUUIDGenerator{}, vastlintResponseExchange{markup: markup}, ortb.NewRequestValidator(openrtb_ext.BuildBidderMap(), map[string]string{}, mockBidderParamValidator{}),
			empty_fetcher.EmptyFetcher{}, empty_fetcher.EmptyFetcher{}, cfg, &metricsConfig.NilMetricsEngine{}, runner, map[string]string{}, nil, openrtb_ext.BuildBidderMap(), empty_fetcher.EmptyFetcher{}, hooks.EmptyPlanBuilder{}, nil)
		require.NoError(t, err)
		recorder := httptest.NewRecorder()
		done := make(chan struct{})
		go func() {
			endpoint(recorder, httptest.NewRequest("POST", "/openrtb2/auction", bytes.NewBufferString(request)), nil)
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("endpoint waited for the five-second validator deadline")
		}
		require.Equal(t, 200, recorder.Code, recorder.Body.String())
		if !enabled {
			original = append([]byte(nil), recorder.Body.Bytes()...)
		} else {
			require.JSONEq(t, string(original), recorder.Body.String())
			select {
			case xml := <-received:
				require.Equal(t, markup, xml)
			case <-time.After(2 * time.Second):
				t.Fatal("final-response VAST was not collected")
			}
		}
		runner.Shutdown()
	}
}
