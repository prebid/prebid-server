package vastlint

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/analytics"
	"github.com/prebid/prebid-server/v4/analytics/vastlint/api"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type fakeValidator func(context.Context, *api.ValidateRequest) (*api.ValidateResponse, error)

func (f fakeValidator) Validate(ctx context.Context, request *api.ValidateRequest, _ ...grpc.CallOption) (*api.ValidateResponse, error) {
	return f(ctx, request)
}
func goodResponse() *api.ValidateResponse {
	return &api.ValidateResponse{SummaryOnly: true, Verdict: &api.Verdict{Valid: true, Summary: &api.Summary{}, Provenance: &api.Provenance{EngineVersion: "test", CatalogDigest: "sha256:" + strings.Repeat("a", 64)}}}
}
func enabledConfig() config.VastlintAnalytics {
	return config.VastlintAnalytics{Enabled: true, Endpoint: "localhost:50051", AllowInsecure: true, Accounts: []string{"host-account"}, SamplePercent: 100}
}
func testModule(t *testing.T) *Module {
	t.Helper()
	cfg, err := normalize(enabledConfig())
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	m := &Module{cfg: cfg, accounts: map[string]struct{}{"host-account": {}}, ctx: ctx, cancel: cancel, done: make(chan struct{}),
		jobs: make(chan document, cfg.QueueSize), slots: make(chan struct{}, cfg.QueueSize), now: time.Now, sample: func() int { return 0 },
		refill: time.Now(), eventTokens: float64(cfg.MaxEventsPerSecond), byteTokens: float64(cfg.MaxBytesPerSecond)}
	m.client = fakeValidator(func(context.Context, *api.ValidateRequest) (*api.ValidateResponse, error) { return goodResponse(), nil })
	t.Cleanup(cancel)
	return m
}
func eventWith(bids ...openrtb2.Bid) *analytics.AuctionObject {
	response := &openrtb2.BidResponse{SeatBid: []openrtb2.SeatBid{{Bid: bids}}}
	return &analytics.AuctionObject{Status: 200, Account: &config.Account{ID: "host-account"}, Response: response, FinalResponse: response}
}
func video(xml string) openrtb2.Bid { return openrtb2.Bid{MType: 2, AdM: xml} }

func TestConfigAndLifecycle(t *testing.T) {
	disabled, err := NewModule(config.VastlintAnalytics{})
	require.NoError(t, err)
	require.Nil(t, disabled.conn)
	disabled.LogAuctionObject(eventWith(video("<VAST/>")))
	require.Zero(t, disabled.Stats()["auctions"])
	disabled.Shutdown()
	disabled.Shutdown()
	require.True(t, disabled.RequiresCreativeDataPermissions())
	disabled.LogVideoObject(nil)
	disabled.LogAmpObject(nil)
	disabled.LogCookieSyncObject(nil)
	disabled.LogSetUIDObject(nil)
	disabled.LogNotificationEventObject(nil)
	for _, insecure := range []bool{false, true} {
		cfg := enabledConfig()
		cfg.AllowInsecure = insecure
		m, err := NewModule(cfg)
		require.NoError(t, err)
		m.Shutdown()
		require.True(t, m.stopped.Load())
	}
	for _, mutate := range []func(*config.VastlintAnalytics){
		func(c *config.VastlintAnalytics) { c.Endpoint = "url://host:12" }, func(c *config.VastlintAnalytics) { c.Endpoint = "host:0" },
		func(c *config.VastlintAnalytics) { c.Endpoint = "host:65536" }, func(c *config.VastlintAnalytics) { c.Endpoint = "host" },
		func(c *config.VastlintAnalytics) { c.Accounts = nil }, func(c *config.VastlintAnalytics) { c.Accounts = []string{""} },
		func(c *config.VastlintAnalytics) { c.Accounts = []string{strings.Repeat("x", 129)} }, func(c *config.VastlintAnalytics) { c.Accounts = make([]string, 257) },
		func(c *config.VastlintAnalytics) { c.SamplePercent = 101 }, func(c *config.VastlintAnalytics) { c.SamplePercent = -1 },
		func(c *config.VastlintAnalytics) { c.MaxDocumentBytes = 128*1024 + 1 }, func(c *config.VastlintAnalytics) { c.QueueSize = 129 },
		func(c *config.VastlintAnalytics) { c.QueueSize = 128; c.MaxDocumentBytes = 128 * 1024 },
		func(c *config.VastlintAnalytics) { c.TimeoutMS = -1 }, func(c *config.VastlintAnalytics) { c.TimeoutMS = 5001 },
		func(c *config.VastlintAnalytics) { c.MaxAgeMS = 10001 }, func(c *config.VastlintAnalytics) { c.MaxEventsPerSecond = 10001 },
		func(c *config.VastlintAnalytics) { c.MaxBytesPerSecond = 1 }, func(c *config.VastlintAnalytics) { c.MaxBytesPerSecond = 16*1024*1024 + 1 },
	} {
		cfg := enabledConfig()
		mutate(&cfg)
		_, err := NewModule(cfg)
		require.Error(t, err)
	}
}

func TestEligibilityBoundsAndOwnership(t *testing.T) {
	m := testModule(t)
	m.LogAuctionObject(nil)
	bad := eventWith(video("<VAST/>"))
	bad.Status = 400
	m.LogAuctionObject(bad)
	bad = eventWith(video("<VAST/>"))
	bad.Account = nil
	m.LogAuctionObject(bad)
	bad = eventWith(video("<VAST/>"))
	bad.Account.ID = "other"
	m.LogAuctionObject(bad)
	bad = eventWith(video("<VAST/>"))
	bad.FinalResponse = nil
	m.LogAuctionObject(bad)
	require.Zero(t, m.Stats()["auctions"])
	m.sample = func() int { return 99 }
	m.cfg.SamplePercent = 1
	m.LogAuctionObject(eventWith(video("<VAST/>")))
	require.Empty(t, m.jobs)
	m.sample = func() int { return 0 }
	extVideo := openrtb2.Bid{Ext: json.RawMessage(`{"prebid":{"type":"video"}}`), AdM: "  <VAST>broken &unknown;"}
	bids := []openrtb2.Bid{{MType: 1, AdM: "<VAST/>"}, {Ext: json.RawMessage("bad")}, {Ext: json.RawMessage(`{"prebid":{"type":"banner"}}`)},
		{Ext: json.RawMessage(strings.Repeat("x", maxExtensionBytes+1))}, video(" "), video("https://example.com/vast"), video("http://example.com/vast"), video(strings.Repeat("x", m.cfg.MaxDocumentBytes+1)), extVideo}
	event := eventWith(bids...)
	m.LogAuctionObject(event)
	require.Len(t, m.jobs, 1)
	job := <-m.jobs
	require.Equal(t, extVideo.AdM, job.xml)
	require.False(t, unsafe.StringData(extVideo.AdM) == unsafe.StringData(job.xml), "queued XML must own its backing bytes")
	event.Response.SeatBid[0].Bid[len(bids)-1].AdM = "modified"
	require.Equal(t, extVideo.AdM, job.xml)
	m.process(job)
	stats := m.Stats()
	require.EqualValues(t, 1, stats["checked"])
	require.EqualValues(t, 1, stats["missing_markup"])
	require.EqualValues(t, 2, stats["url_markup"])
	require.EqualValues(t, 1, stats["oversized_markup"])
	require.EqualValues(t, 1, stats["oversized_extension"])
	require.EqualValues(t, 3, stats["non_video"])
	many := eventWith(make([]openrtb2.Bid, maxScan+1)...)
	many.Response.SeatBid[0].Bid[maxScan] = video("<VAST/>")
	m.LogAuctionObject(many)
	require.Empty(t, m.jobs)
	require.EqualValues(t, 1, m.Stats()["scan_limited"])
	many = eventWith()
	many.Response.SeatBid = make([]openrtb2.SeatBid, maxScan+1)
	many.Response.SeatBid[maxScan].Bid = []openrtb2.Bid{video("<VAST/>")}
	m.LogAuctionObject(many)
	require.EqualValues(t, 2, m.Stats()["scan_limited"])
}

func TestAdmissionIncludesCopyQueueAndInflight(t *testing.T) {
	m := testModule(t)
	m.cfg.QueueSize = 2
	m.jobs = make(chan document, 2)
	m.slots = make(chan struct{}, 2)
	entered, release := make(chan struct{}), make(chan struct{})
	m.client = fakeValidator(func(context.Context, *api.ValidateRequest) (*api.ValidateResponse, error) {
		close(entered)
		<-release
		return goodResponse(), nil
	})
	m.capture("<VAST/>")
	job := <-m.jobs
	finished := make(chan struct{})
	go func() { m.process(job); close(finished) }()
	<-entered
	m.capture("<VAST/>")
	m.capture("<VAST/>")
	require.Len(t, m.slots, 2)
	require.Len(t, m.jobs, 1)
	require.EqualValues(t, 1, m.Stats()["capacity_dropped"])
	m.cancel()
	m.capture("<VAST/>")
	require.Len(t, m.slots, 2)
	close(release)
	<-finished
	require.EqualValues(t, 0, m.Stats()["checked"])
	require.EqualValues(t, 1, m.Stats()["failed"])
	require.Len(t, m.slots, 1)
	m.process(<-m.jobs)
	require.Empty(t, m.slots)
	require.EqualValues(t, 1, m.Stats()["shutdown_dropped"])
}

func TestRateLimitsPrecedeAllocation(t *testing.T) {
	m := testModule(t)
	current := m.refill
	m.now = func() time.Time { return current }
	m.eventTokens = 1
	m.byteTokens = 10
	m.capture("12345")
	m.capture("1")
	require.Len(t, m.jobs, 1)
	require.Len(t, m.slots, 1)
	require.EqualValues(t, 1, m.Stats()["rate_dropped"])
	m.process(<-m.jobs)
	m.eventTokens = 100
	m.byteTokens = 0
	m.capture("12345")
	require.Empty(t, m.slots)
	require.EqualValues(t, 2, m.Stats()["rate_dropped"])
	current = current.Add(time.Second)
	m.capture("12345")
	require.Len(t, m.jobs, 1)
	m.process(<-m.jobs)
	m.stopped.Store(true)
	m.capture("<VAST/>")
	require.Empty(t, m.slots)
	require.EqualValues(t, 1, m.Stats()["shutdown_dropped"])
}

func TestRPCFailureExpiryAndPanic(t *testing.T) {
	cases := []struct {
		name string
		call fakeValidator
		age  time.Duration
		want string
	}{
		{"success", func(context.Context, *api.ValidateRequest) (*api.ValidateResponse, error) { return goodResponse(), nil }, 0, "checked"},
		{"findings", func(context.Context, *api.ValidateRequest) (*api.ValidateResponse, error) {
			r := goodResponse()
			r.Verdict.Summary.Warnings = 1
			return r, nil
		}, 0, "with_findings"},
		{"error", func(context.Context, *api.ValidateRequest) (*api.ValidateResponse, error) {
			return nil, errors.New("contains secret XML")
		}, 0, "failed"},
		{"nil", func(context.Context, *api.ValidateRequest) (*api.ValidateResponse, error) { return nil, nil }, 0, "failed"},
		{"summary_only_unsupported", func(context.Context, *api.ValidateRequest) (*api.ValidateResponse, error) {
			r := goodResponse()
			r.SummaryOnly = false
			return r, nil
		}, 0, "failed"},
		{"no_verdict", func(context.Context, *api.ValidateRequest) (*api.ValidateResponse, error) {
			return &api.ValidateResponse{}, nil
		}, 0, "failed"},
		{"no_summary", func(context.Context, *api.ValidateRequest) (*api.ValidateResponse, error) {
			r := goodResponse()
			r.Verdict.Summary = nil
			return r, nil
		}, 0, "failed"},
		{"no_provenance", func(context.Context, *api.ValidateRequest) (*api.ValidateResponse, error) {
			r := goodResponse()
			r.Verdict.Provenance = nil
			return r, nil
		}, 0, "failed"},
		{"bad_provenance", func(context.Context, *api.ValidateRequest) (*api.ValidateResponse, error) {
			r := goodResponse()
			r.Verdict.Provenance.EngineVersion = ""
			return r, nil
		}, 0, "failed"},
		{"long_provenance", func(context.Context, *api.ValidateRequest) (*api.ValidateResponse, error) {
			r := goodResponse()
			r.Verdict.Provenance.EngineVersion = strings.Repeat("x", 129)
			return r, nil
		}, 0, "failed"},
		{"bad_digest", func(context.Context, *api.ValidateRequest) (*api.ValidateResponse, error) {
			r := goodResponse()
			r.Verdict.Provenance.CatalogDigest = "bad"
			return r, nil
		}, 0, "failed"},
		{"deadline", func(ctx context.Context, _ *api.ValidateRequest) (*api.ValidateResponse, error) {
			<-ctx.Done()
			return goodResponse(), nil
		}, 0, "failed"},
		{"panic", func(context.Context, *api.ValidateRequest) (*api.ValidateResponse, error) {
			panic("contains secret XML")
		}, 0, "failed"},
		{"expired", func(context.Context, *api.ValidateRequest) (*api.ValidateResponse, error) {
			t.Fatal("expired work ran")
			return nil, nil
		}, 2 * time.Second, "expired"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := testModule(t)
			m.cfg.TimeoutMS = 1
			m.client = c.call
			m.slots <- struct{}{}
			m.process(document{xml: "<VAST/>", created: time.Now().Add(-c.age)})
			require.Empty(t, m.slots)
			require.EqualValues(t, 1, m.Stats()[c.want])
		})
	}
}

func TestRaceCaptureAndShutdown(t *testing.T) {
	m := testModule(t)
	go m.run()
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < 20; n++ {
				event := eventWith(video("<VAST>broken</VAST>"))
				m.LogAuctionObject(event)
				event.Response.SeatBid[0].Bid[0].AdM = "changed"
				m.Stats()
			}
		}()
	}
	m.Shutdown()
	wg.Wait()
	<-m.done
	require.Empty(t, m.slots)
	m.LogAuctionObject(eventWith(video("<VAST/>")))
}

type grpcValidator struct {
	api.UnimplementedVastlintServiceServer
	delay    bool
	received chan string
}

func (s grpcValidator) Validate(ctx context.Context, r *api.ValidateRequest) (*api.ValidateResponse, error) {
	if s.delay {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	s.received <- r.Document
	return goodResponse(), nil
}
func TestBackgroundWireAndOutage(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer()
	received := make(chan string, 1)
	api.RegisterVastlintServiceServer(server, grpcValidator{received: received})
	go server.Serve(listener)
	defer server.Stop()
	cfg := enabledConfig()
	cfg.Endpoint = listener.Addr().String()
	m, err := NewModule(cfg)
	require.NoError(t, err)
	defer m.Shutdown()
	m.LogAuctionObject(eventWith(video("<VAST/>")))
	select {
	case xml := <-received:
		require.Equal(t, "<VAST/>", xml)
	case <-time.After(2 * time.Second):
		t.Fatal("not delivered")
	}
	require.Eventually(t, func() bool { return m.Stats()["checked"] == 1 }, time.Second, time.Millisecond)
	server.Stop()
	m.LogAuctionObject(eventWith(video("<VAST/>")))
	require.Eventually(t, func() bool { return m.Stats()["failed"] == 1 }, time.Second, time.Millisecond)
}

func TestDigestFormat(t *testing.T) {
	require.True(t, validDigest("sha256:"+strings.Repeat("a", 64)))
	for _, value := range []string{"", strings.Repeat("a", 64), "sha256:" + strings.Repeat("A", 64), "sha256:" + strings.Repeat("g", 64), "other:" + strings.Repeat("a", 65)} {
		require.False(t, validDigest(value))
	}
}

func TestOnlyFinalResponseMarkupIsCollected(t *testing.T) {
	m := testModule(t)
	event := eventWith(video("<VAST>removed</VAST>"))
	event.FinalResponse = nil
	m.LogAuctionObject(event)
	require.Empty(t, m.jobs)
	require.Zero(t, m.Stats()["auctions"])

	returned := "<VAST>returned</VAST>"
	event.FinalResponse = &openrtb2.BidResponse{SeatBid: []openrtb2.SeatBid{{Bid: []openrtb2.Bid{video(returned)}}}}
	m.client = fakeValidator(func(_ context.Context, r *api.ValidateRequest) (*api.ValidateResponse, error) {
		require.True(t, r.SummaryOnly, "the server must produce a compact reply")
		require.Equal(t, returned, r.Document)
		return goodResponse(), nil
	})
	m.LogAuctionObject(event)
	require.Len(t, m.jobs, 1)
	m.process(<-m.jobs)
	require.EqualValues(t, 1, m.Stats()["checked"])
	require.Equal(t, "<VAST>removed</VAST>", event.Response.SeatBid[0].Bid[0].AdM)
}
