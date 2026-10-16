package vastlint

import (
	"context"
	"encoding/json"
	"github.com/prebid/openrtb/v20/openrtb2"
	"strings"
	"testing"
	"time"

	"github.com/prebid/prebid-server/v4/analytics/vastlint/api"
)

func BenchmarkFinalResponseCapture(b *testing.B) {
	for _, scenario := range []string{"disabled", "unsampled", "capacity_full", "rate_full", "capture_64KiB", "capture_64KiB_prebid_extension", "scan_64_extensions_4KiB"} {
		b.Run(scenario, func(b *testing.B) {
			cfg, _ := normalize(enabledConfig())
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			m := &Module{cfg: cfg, accounts: map[string]struct{}{"host-account": {}}, jobs: make(chan document, cfg.QueueSize), slots: make(chan struct{}, cfg.QueueSize),
				ctx: ctx, cancel: cancel, now: time.Now, sample: func() int { return 0 }, refill: time.Now(), eventTokens: 1e9, byteTokens: 1e12}
			m.client = fakeValidator(func(context.Context, *api.ValidateRequest) (*api.ValidateResponse, error) { return goodResponse(), nil })
			switch scenario {
			case "disabled":
				m.cfg.Enabled = false
			case "unsampled":
				m.sample = func() int { return 99 }
				m.cfg.SamplePercent = 1
			case "capacity_full":
				for i := 0; i < cap(m.slots); i++ {
					m.slots <- struct{}{}
				}
			case "rate_full":
				m.eventTokens = 0
			}
			// Frozen time keeps the benchmark's rate limiter deterministic.
			instant := time.Now()
			m.refill = instant
			m.now = func() time.Time { return instant }
			event := eventWith(video(strings.Repeat("x", 64*1024)))
			if scenario == "capture_64KiB_prebid_extension" {
				event.Response.SeatBid[0].Bid[0].MType = 0
				event.Response.SeatBid[0].Bid[0].Ext = json.RawMessage(`{"prebid":{"type":"video"}}`)
			}
			if scenario == "scan_64_extensions_4KiB" {
				prefix, suffix := `{"prebid":{"type":"banner"},"ignored":"`, `"}`
				ext := json.RawMessage(prefix + strings.Repeat("x", maxExtensionBytes-len(prefix)-len(suffix)) + suffix)
				event = eventWith(make([]openrtb2.Bid, maxScan)...)
				for i := range event.Response.SeatBid[0].Bid {
					event.Response.SeatBid[0].Bid[i].Ext = ext
				}
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if strings.HasPrefix(scenario, "capture_64KiB") {
					m.eventTokens = 100
					m.byteTokens = 1e6
				}
				m.LogAuctionObject(event)
				if strings.HasPrefix(scenario, "capture_64KiB") {
					<-m.jobs
					<-m.slots
				}
			}
		})
	}
}
