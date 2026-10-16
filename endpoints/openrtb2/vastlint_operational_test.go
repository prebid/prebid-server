//go:build vastlint_operational && (darwin || linux)

package openrtb2

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	ortb2 "github.com/prebid/openrtb/v20/openrtb2"
	analyticsBuild "github.com/prebid/prebid-server/v4/analytics/build"
	"github.com/prebid/prebid-server/v4/analytics/vastlint/api"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/exchange"
	"github.com/prebid/prebid-server/v4/hooks"
	"github.com/prebid/prebid-server/v4/metrics"
	metricsConfig "github.com/prebid/prebid-server/v4/metrics/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/ortb"
	"github.com/prebid/prebid-server/v4/stored_requests/backends/empty_fetcher"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type operationalMetrics struct {
	metricsConfig.NilMetricsEngine
	mu       sync.Mutex
	counters map[string]uint64
}

func (m *operationalMetrics) RecordVastlintEvent(event metrics.VastlintEvent, count uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[string(event)] += count
}

type operationalExchange struct {
	banner bool
	markup string
}

func (e operationalExchange) HoldAuction(_ context.Context, r *exchange.AuctionRequest, _ *exchange.DebugLog) (*exchange.AuctionResponse, error) {
	bid := ortb2.Bid{ID: "bid-1", ImpID: "imp-1", Price: 1.23, AdM: e.markup, MType: 2, Ext: json.RawMessage(`{"prebid":{"type":"video"}}`)}
	if e.banner {
		bid.MType = 1
		bid.Ext = json.RawMessage(`{"prebid":{"type":"banner"}}`)
	}
	return &exchange.AuctionResponse{BidResponse: &ortb2.BidResponse{ID: r.BidRequestWrapper.ID, Cur: "USD", Ext: json.RawMessage(`{}`), SeatBid: []ortb2.SeatBid{{Seat: "appnexus", Bid: []ortb2.Bid{bid}}}}}, nil
}

type operationalSlowValidator struct {
	api.UnimplementedVastlintServiceServer
}

func (operationalSlowValidator) Validate(ctx context.Context, _ *api.ValidateRequest) (*api.ValidateResponse, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

// Run one scenario per process so the peak RSS is attributable to that scenario.
// Client, endpoint and (only for slow/saturated cases) fake server share a process.
func TestVastlintOperational(t *testing.T) {
	scenario, traffic := os.Getenv("VASTLINT_LOAD_SCENARIO"), os.Getenv("VASTLINT_LOAD_TRAFFIC")
	require.Contains(t, []string{"disabled", "healthy", "unreachable", "slow", "saturated"}, scenario)
	require.Contains(t, []string{"video", "banner", "excluded", "large_user"}, traffic)
	n := 2000
	if raw := os.Getenv("VASTLINT_LOAD_REQUESTS"); raw != "" {
		var err error
		n, err = strconv.Atoi(raw)
		require.NoError(t, err)
		require.Greater(t, n, 0)
	}
	endpoint := os.Getenv("VASTLINT_TEST_ENDPOINT")
	cfg := config.VastlintAnalytics{Enabled: scenario != "disabled", Endpoint: endpoint, AllowInsecure: true, Accounts: []string{"host-account"}, SamplePercent: 1}
	if raw := os.Getenv("VASTLINT_LOAD_SAMPLE_PERCENT"); raw != "" {
		var err error
		cfg.SamplePercent, err = strconv.Atoi(raw)
		require.NoError(t, err)
	}
	if scenario == "unreachable" {
		cfg.Endpoint = "127.0.0.1:1"
	}
	if scenario == "slow" || scenario == "saturated" {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		server := grpc.NewServer()
		api.RegisterVastlintServiceServer(server, operationalSlowValidator{})
		go server.Serve(listener)
		defer server.Stop()
		cfg.Endpoint = listener.Addr().String()
		if scenario == "saturated" {
			cfg.QueueSize = 1
		}
	}
	if scenario == "disabled" && cfg.Endpoint == "" {
		cfg.Endpoint = "127.0.0.1:1"
	}
	recorder := &operationalMetrics{counters: map[string]uint64{}}
	runner := analyticsBuild.New(&config.Analytics{Vastlint: cfg}, recorder)
	configuration := &config.Configuration{MaxRequestSize: 1024 * 1024}
	require.NoError(t, configuration.MarshalAccountDefaults())
	const prefix = `<VAST version="2.0"><!--`
	const suffix = `--></VAST>`
	markup := prefix + strings.Repeat("x", 64*1024-len(prefix)-len(suffix)) + suffix
	if traffic == "banner" {
		markup = "<div>ad</div>"
	}
	handler, err := NewEndpoint(fakeUUIDGenerator{}, operationalExchange{banner: traffic == "banner", markup: markup}, ortb.NewRequestValidator(openrtb_ext.BuildBidderMap(), map[string]string{}, mockBidderParamValidator{}), empty_fetcher.EmptyFetcher{}, empty_fetcher.EmptyFetcher{}, configuration, recorder, runner, map[string]string{}, nil, openrtb_ext.BuildBidderMap(), empty_fetcher.EmptyFetcher{}, hooks.EmptyPlanBuilder{}, nil)
	require.NoError(t, err)
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { handler(w, r, nil) }))
	defer httpServer.Close()
	account := "host-account"
	if traffic == "excluded" {
		account = "excluded-account"
	}
	user := ""
	if traffic == "large_user" || traffic == "excluded" {
		user = `,"user":{"ext":{"padding":"` + strings.Repeat("x", 65536) + `"}}`
	}
	imp := `"video":{"mimes":["video/mp4"],"protocols":[2],"minduration":1,"maxduration":30}`
	if traffic == "banner" {
		imp = `"banner":{"format":[{"w":300,"h":250}]}`
	}
	body := []byte(`{"id":"request-1","regs":{"gdpr":0},"site":{"page":"https://example.com","publisher":{"id":"` + account + `"}},"imp":[{"id":"imp-1",` + imp + `,"ext":{"appnexus":{"placementId":1}}}],"tmax":500` + user + `}`)
	transport := &http.Transport{MaxIdleConnsPerHost: 16}
	defer transport.CloseIdleConnections()
	client := http.Client{Transport: transport, Timeout: 5 * time.Second}
	request := func() bool {
		response, err := client.Post(httpServer.URL+"/openrtb2/auction", "application/json", bytes.NewReader(body))
		if err != nil {
			return false
		}
		defer response.Body.Close()
		bytes, err := io.ReadAll(response.Body)
		if err != nil || response.StatusCode != 200 {
			return false
		}
		var result ortb2.BidResponse
		return json.Unmarshal(bytes, &result) == nil && len(result.SeatBid) == 1 && len(result.SeatBid[0].Bid) == 1 && result.SeatBid[0].Bid[0].AdM == markup && result.SeatBid[0].Bid[0].Price == 1.23
	}
	for i := 0; i < 100; i++ {
		require.True(t, request())
	}
	runtime.GC()
	var before, after runtime.MemStats
	var cpuBefore, cpuAfter syscall.Rusage
	runtime.ReadMemStats(&before)
	require.NoError(t, syscall.Getrusage(syscall.RUSAGE_SELF, &cpuBefore))
	latencies := make([]int64, n)
	var next atomic.Int64
	var failed atomic.Int64
	var workers sync.WaitGroup
	started := time.Now()
	for i := 0; i < 16; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for {
				j := int(next.Add(1)) - 1
				if j >= n {
					return
				}
				start := time.Now()
				ok := request()
				latencies[j] = time.Since(start).Nanoseconds()
				if !ok {
					failed.Add(1)
				}
			}
		}()
	}
	workers.Wait()
	elapsed := time.Since(started)
	runtime.ReadMemStats(&after)
	require.NoError(t, syscall.Getrusage(syscall.RUSAGE_SELF, &cpuAfter))
	runner.Shutdown()
	require.Zero(t, failed.Load(), "response content or HTTP status changed")
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	cpu := func(r syscall.Rusage) float64 {
		return float64(r.Utime.Sec+r.Stime.Sec) + float64(r.Utime.Usec+r.Stime.Usec)/1e6
	}
	rss := cpuAfter.Maxrss
	if runtime.GOOS == "linux" {
		rss *= 1024
	}
	result := map[string]any{"scenario": scenario, "sample_percent": cfg.SamplePercent, "traffic": traffic, "requests": n, "warmup": 100, "concurrency": 16, "elapsed_seconds": elapsed.Seconds(), "qps": float64(n) / elapsed.Seconds(), "p50_ms": float64(latencies[n/2]) / 1e6, "p95_ms": float64(latencies[n*95/100]) / 1e6, "p99_ms": float64(latencies[n*99/100]) / 1e6, "cpu_seconds": cpu(cpuAfter) - cpu(cpuBefore), "allocated_bytes_per_request": float64(after.TotalAlloc-before.TotalAlloc) / float64(n), "allocations_per_request": float64(after.Mallocs-before.Mallocs) / float64(n), "peak_rss_bytes": rss, "response_failures": failed.Load(), "counters_including_warmup": recorder.counters}
	encoded, err := json.Marshal(result)
	require.NoError(t, err)
	t.Log(string(encoded))
	if path := os.Getenv("VASTLINT_LOAD_RESULT"); path != "" {
		require.NoError(t, os.WriteFile(path, append(encoded, '\n'), 0600))
	}
}
