package identity

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"testing"
	"time"

	iiqenrichment "github.com/Intent-IQ/identity-go/enrichment"
	iiqreporting "github.com/Intent-IQ/identity-go/reporting"
	"github.com/alicebob/miniredis/v2"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/modules/moduledeps"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubEnricher struct {
	result iiqenrichment.Result
	err    error
	req    iiqenrichment.Request
}

func (s *stubEnricher) Enrich(_ context.Context, req iiqenrichment.Request) (iiqenrichment.Result, error) {
	s.req = req
	return s.result, s.err
}
func (*stubEnricher) Shutdown(context.Context) error { return nil }

type stubReporter struct{ requests chan iiqreporting.Request }

func (s *stubReporter) TryReport(req iiqreporting.Request) iiqreporting.ReportResult {
	s.requests <- req
	return iiqreporting.ReportAccepted
}
func (*stubReporter) Shutdown(context.Context) error { return nil }

func TestConfigValidateAsyncRequirements(t *testing.T) {
	wait := int64(10)
	config := defaultConfig()
	config.WaitTimeoutMS = &wait
	config.Cache.Enabled = false
	require.EqualError(t, config.validate(), "cache must be enabled when wait_timeout is less than timeout")

	config.Cache.Enabled = true
	require.EqualError(t, config.validate(), "max_concurrent_calls must be positive when wait_timeout is less than timeout")

	config.MaxConcurrentCalls = 2
	require.NoError(t, config.validate())
}

func TestConfigResolveKeepsProcessResources(t *testing.T) {
	config := defaultConfig()
	config.PartnerID = "host"
	config.MaxConcurrentCalls = 7
	config.Cache.Enabled = true
	resolved := config.resolve(json.RawMessage(`{"partner_id":"account","max_concurrent_calls":99,"cache":{"enabled":false}}`))
	assert.Equal(t, "account", resolved.PartnerID)
	assert.Equal(t, 7, resolved.MaxConcurrentCalls)
	assert.True(t, resolved.Cache.Enabled)
}

func TestBuilderWithoutCache(t *testing.T) {
	built, err := Builder(json.RawMessage(`{"partner_id":"p","timeout":100}`), moduledeps.ModuleDeps{})
	require.NoError(t, err)
	module := built.(*Module)
	require.NotNil(t, module.enricher)
	require.NotNil(t, module.reporter)
	require.NoError(t, module.Shutdown())
}

func TestProcessedAuctionEnrichesAndStoresContext(t *testing.T) {
	tc := int64(5)
	enricher := &stubEnricher{result: iiqenrichment.Result{
		EIDs:       []openrtb2.EID{{Source: "intentiq.com", UIDs: []openrtb2.UID{{ID: "iiq"}}}},
		ABTestUUID: "ab", TerminationCause: &tc,
	}}
	module := &Module{config: defaultConfig(), enricher: enricher}
	request := &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
		ID: "auction", Site: &openrtb2.Site{Domain: "example.com"},
		Device: &openrtb2.Device{IP: "192.0.2.1", UA: "ua"},
		User:   &openrtb2.User{EIDs: []openrtb2.EID{{Source: "existing.com"}}},
	}}
	payload := hookstage.ProcessedAuctionRequestPayload{Request: request}

	result, err := module.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{}, payload)
	require.NoError(t, err)
	require.Len(t, result.ChangeSet.Mutations(), 1)
	_, err = result.ChangeSet.Mutations()[0].Apply(payload)
	require.NoError(t, err)
	require.Len(t, request.User.EIDs, 2)
	assert.Equal(t, "existing.com", request.User.EIDs[0].Source)
	assert.Equal(t, "iiq", request.User.EIDs[1].UIDs[0].ID)
	assert.Equal(t, request.BidRequest, enricher.req.Auction)

	stored, ok := loadAuctionContext(result.ModuleContext)
	require.True(t, ok)
	assert.Equal(t, "auction", stored.AuctionID)
	assert.Equal(t, "example.com", stored.Reference)
	assert.Equal(t, "ab", stored.ABTestUUID)
	assert.Equal(t, tc, *stored.TerminationCause)
}

func TestProcessedAuctionFailsOpen(t *testing.T) {
	module := &Module{config: defaultConfig(), enricher: &stubEnricher{err: errors.New("unavailable")}}
	result, err := module.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{}, hookstage.ProcessedAuctionRequestPayload{})
	require.NoError(t, err)
	assert.False(t, result.Reject)
	assert.Empty(t, result.ChangeSet.Mutations())
	require.NotNil(t, result.ModuleContext)
}

func TestAuctionResponseReportsEveryBid(t *testing.T) {
	reporter := &stubReporter{requests: make(chan iiqreporting.Request, 2)}
	config := defaultConfig()
	config.PartnerID = "partner"
	config.ReportsEndpoint = "https://reports.example"
	module := &Module{config: config, reporter: reporter}
	tc := int64(41)
	ctx := storeAuctionContext(auctionContext{
		AuctionID: "auction", Reference: "example.com", IP: "192.0.2.1",
		UserAgent: "ua", ABTestUUID: "ab", TerminationCause: &tc,
	})
	response := &openrtb2.BidResponse{Cur: "EUR", SeatBid: []openrtb2.SeatBid{{
		Seat: "bidder", Bid: []openrtb2.Bid{{ID: "1"}, {ID: "2"}},
	}}}

	_, err := module.HandleAuctionResponseHook(context.Background(), hookstage.ModuleInvocationContext{ModuleContext: ctx}, hookstage.AuctionResponsePayload{BidResponse: response})
	require.NoError(t, err)
	for range 2 {
		select {
		case got := <-reporter.requests:
			assert.Equal(t, "partner", got.PartnerID)
			assert.Equal(t, "auction", got.AuctionID)
			assert.Equal(t, "bidder", got.BidderCode)
			assert.Equal(t, "EUR", got.Currency)
			assert.Equal(t, "https://reports.example", got.Endpoint)
			assert.Equal(t, "example.com", got.Reference)
			assert.Equal(t, "192.0.2.1", got.IP)
			assert.Equal(t, "ua", got.UserAgent)
			assert.Equal(t, "ab", got.ABTestUUID)
			assert.Equal(t, &tc, got.TerminationCause)
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for report")
		}
	}
}

func TestNewAuctionContext(t *testing.T) {
	tests := []struct {
		name    string
		request *openrtb2.BidRequest
		want    auctionContext
	}{
		{name: "nil request"},
		{
			name: "site page fallback and ipv6 fallback",
			request: &openrtb2.BidRequest{
				ID: "a", Site: &openrtb2.Site{Domain: " ", Page: "https://example.com/p"},
				Device: &openrtb2.Device{IPv6: "2001:db8::1", UA: "ua"},
			},
			want: auctionContext{AuctionID: "a", Reference: "https://example.com/p", IP: "2001:db8::1", UserAgent: "ua"},
		},
		{
			name:    "app bundle",
			request: &openrtb2.BidRequest{ID: "a", App: &openrtb2.App{Bundle: "com.example", Name: "Example"}},
			want:    auctionContext{AuctionID: "a", Reference: "com.example"},
		},
		{
			name:    "app name fallback",
			request: &openrtb2.BidRequest{ID: "a", App: &openrtb2.App{Name: "Example"}},
			want:    auctionContext{AuctionID: "a", Reference: "Example"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, newAuctionContext(test.request))
		})
	}
}

func TestAuctionResponseSkipsWithoutEndpointOrResponse(t *testing.T) {
	reporter := &stubReporter{requests: make(chan iiqreporting.Request, 1)}
	config := defaultConfig()
	config.ReportsEndpoint = "https://reports.example"
	module := &Module{config: config, reporter: reporter}

	_, err := module.HandleAuctionResponseHook(context.Background(), hookstage.ModuleInvocationContext{}, hookstage.AuctionResponsePayload{})
	require.NoError(t, err)

	module.config.ReportsEndpoint = ""
	response := &openrtb2.BidResponse{SeatBid: []openrtb2.SeatBid{{Bid: []openrtb2.Bid{{ID: "1"}}}}}
	_, err = module.HandleAuctionResponseHook(context.Background(), hookstage.ModuleInvocationContext{}, hookstage.AuctionResponsePayload{BidResponse: response})
	require.NoError(t, err)

	assert.Empty(t, reporter.requests)
}

func TestAuctionResponseUsesAccountEndpointWithoutModuleContext(t *testing.T) {
	reporter := &stubReporter{requests: make(chan iiqreporting.Request, 1)}
	module := &Module{config: defaultConfig(), reporter: reporter}
	invocation := hookstage.ModuleInvocationContext{
		AccountConfig: json.RawMessage(`{"partner_id":"account","reports_endpoint":"https://account.example"}`),
	}
	response := &openrtb2.BidResponse{SeatBid: []openrtb2.SeatBid{{Seat: "bidder", Bid: []openrtb2.Bid{{ID: "1"}}}}}

	_, err := module.HandleAuctionResponseHook(context.Background(), invocation, hookstage.AuctionResponsePayload{BidResponse: response})
	require.NoError(t, err)
	got := <-reporter.requests
	assert.Equal(t, "https://account.example", got.Endpoint)
	assert.Equal(t, "account", got.PartnerID)
	assert.Empty(t, got.AuctionID)
	assert.Nil(t, got.TerminationCause)
}

func gatheredFamilies(t *testing.T, registry *prometheus.Registry) int {
	t.Helper()
	families, err := registry.Gather()
	require.NoError(t, err)
	return len(families)
}

func TestBuilderRegistersMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	built, err := Builder(json.RawMessage(`{"partner_id":"p"}`), moduledeps.ModuleDeps{MetricsRegisterer: registry})
	require.NoError(t, err)
	defer built.(*Module).Shutdown()
	assert.NotZero(t, gatheredFamilies(t, registry))
}

func TestBuilderMetricsDisabledRegistersNothing(t *testing.T) {
	registry := prometheus.NewRegistry()
	built, err := Builder(json.RawMessage(`{"partner_id":"p","metrics_enabled":false}`), moduledeps.ModuleDeps{MetricsRegisterer: registry})
	require.NoError(t, err)
	defer built.(*Module).Shutdown()
	assert.Zero(t, gatheredFamilies(t, registry))
}

func TestBuilderAcceptsFrameworkEnabledKey(t *testing.T) {
	built, err := Builder(json.RawMessage(`{"enabled":true,"partner_id":"p"}`), moduledeps.ModuleDeps{})
	require.NoError(t, err)
	require.NoError(t, built.(*Module).Shutdown())
}

func TestBuilderCacheProviders(t *testing.T) {
	server := miniredis.RunT(t)
	port, err := strconv.Atoi(server.Port())
	require.NoError(t, err)

	for _, provider := range []string{"redis", "valkey"} {
		t.Run(provider, func(t *testing.T) {
			raw, err := json.Marshal(map[string]any{
				"partner_id": "p",
				"cache":      map[string]any{"enabled": true, "provider": provider},
				provider:     map[string]any{"host": server.Host(), "port": port},
			})
			require.NoError(t, err)
			built, err := Builder(raw, moduledeps.ModuleDeps{})
			require.NoError(t, err)
			module := built.(*Module)
			require.NotNil(t, module.store)
			require.NoError(t, module.Shutdown())
		})
	}
}

func TestBuilderRejectsInvalidConfig(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		err  string
	}{
		{name: "malformed json", raw: `{bad`, err: "parse config"},
		{name: "invalid timeout", raw: `{"timeout":0}`, err: "timeout must be positive"},
		{name: "unknown provider", raw: `{"cache":{"enabled":true,"provider":"memcached"}}`, err: `unknown cache provider "memcached"`},
		{name: "missing provider config", raw: `{"cache":{"enabled":true,"provider":"redis"}}`, err: "redis configuration is required"},
		{name: "unreachable backend", raw: `{"cache":{"enabled":true,"provider":"redis"},"redis":{"host":"127.0.0.1","port":1,"connect_timeout_ms":100}}`, err: "create cache store"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Builder(json.RawMessage(test.raw), moduledeps.ModuleDeps{})
			require.ErrorContains(t, err, test.err)
		})
	}
}
