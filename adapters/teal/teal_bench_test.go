package teal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

// Benchmarks for the Teal adapter hot paths. Run locally with:
//
//	go test -bench=. -benchmem ./adapters/teal/...
//
// Expected callers: prebid-server in the auction loop. MakeRequests is invoked
// once per bid request that includes Teal in its bidder set; MakeBids is
// invoked once per HTTP response. Both are on the request critical path, so
// allocation pressure matters.

// benchPlacement is a stable placement constant used across benches so the
// inner loop compares fairly.
const benchPlacement = "bench-placement-300x250"

// benchBidder constructs a fully-wired adapter once, mirroring how prebid-server
// reuses bidder instances across requests.
func benchBidder(b *testing.B) adapters.Bidder {
	b.Helper()
	bidder, err := Builder(openrtb_ext.BidderTeal,
		config.Adapter{Endpoint: testEndpoint},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
	if err != nil {
		b.Fatalf("Builder failed: %v", err)
	}
	return bidder
}

// benchBannerImp builds a single banner imp with the canonical happy-path ext.
// Each bench builds a fresh imp inside b.ResetTimer() to avoid cross-iteration
// mutation (modifyImp clones, but defensive isolation keeps the bench honest).
func benchBannerImp(b *testing.B, id string) openrtb2.Imp {
	b.Helper()
	ext := map[string]openrtb_ext.ExtImpTeal{
		"bidder": {Account: "bench-account", Placement: strPtrLocal(benchPlacement)},
	}
	raw, err := json.Marshal(ext)
	if err != nil {
		b.Fatalf("ext marshal failed: %v", err)
	}
	return openrtb2.Imp{
		ID:     id,
		Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}},
		Ext:    raw,
	}
}

// strPtrLocal — local *string helper to keep this file self-contained.
// (We can't reuse strPtr from teal_test.go's package because it's the same
// package, but keeping it local-named documents bench-vs-test independence.)
func strPtrLocal(s string) *string { return &s }

// BenchmarkMakeRequests benches the full happy-path of MakeRequests on a
// realistic single-imp banner request. Captures all three mutations + JSON
// marshal of the outbound body.
func BenchmarkMakeRequests(b *testing.B) {
	bidder := benchBidder(b)
	imp := benchBannerImp(b, "bench-imp-banner")
	template := &openrtb2.BidRequest{
		ID:   "bench-request",
		Imp:  []openrtb2.Imp{imp},
		Site: &openrtb2.Site{ID: "bench-site", Publisher: &openrtb2.Publisher{ID: "bench-publisher"}},
	}

	// Snapshot the imp.Ext bytes once so each iteration starts identically —
	// MakeRequests treats imp.Ext as immutable input but we don't want any
	// future change to flake the bench.
	originalExt := append(json.RawMessage(nil), imp.Ext...)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset request state per iteration (cheap: shallow copy of struct
		// + restoring imp.Ext bytes).
		req := *template
		req.Imp = make([]openrtb2.Imp, 1)
		req.Imp[0] = imp
		req.Imp[0].Ext = originalExt

		out, errs := bidder.MakeRequests(&req, &adapters.ExtraRequestInfo{})
		if len(errs) != 0 {
			b.Fatalf("unexpected errors: %v", errs)
		}
		if len(out) != 1 {
			b.Fatalf("expected 1 RequestData, got %d", len(out))
		}
	}
}

// BenchmarkMakeBids benches the full MakeBids on a realistic single-imp banner
// response. Captures status-code check, body unmarshal, and bid-type lookup.
func BenchmarkMakeBids(b *testing.B) {
	bidder := benchBidder(b)
	imp := benchBannerImp(b, "bench-imp-banner")
	request := &openrtb2.BidRequest{Imp: []openrtb2.Imp{imp}}

	body, err := json.Marshal(openrtb2.BidResponse{
		ID:  "bench-resp",
		Cur: "USD",
		SeatBid: []openrtb2.SeatBid{{
			Seat: "teal",
			Bid: []openrtb2.Bid{{
				ID:    "b1",
				ImpID: "bench-imp-banner",
				Price: 1.50,
				W:     300,
				H:     250,
			}},
		}},
	})
	if err != nil {
		b.Fatalf("response marshal failed: %v", err)
	}
	respData := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       body,
	}
	reqData := &adapters.RequestData{
		Method: http.MethodPost,
		Uri:    testEndpoint,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, errs := bidder.MakeBids(request, reqData, respData)
		if len(errs) != 0 {
			b.Fatalf("unexpected errors: %v", errs)
		}
		if out == nil || len(out.Bids) != 1 {
			b.Fatalf("expected 1 bid, got %v", out)
		}
	}
}

// BenchmarkGetBidType benches the imp lookup in a 10-imp request. This is the
// inner loop hot spot when responses contain many bids — getBidType is O(n) in
// imp count per bid and the fallthrough behavior on missing imp matters.
func BenchmarkGetBidType(b *testing.B) {
	imps := make([]openrtb2.Imp, 10)
	for i := range imps {
		imps[i] = openrtb2.Imp{ID: fmt.Sprintf("imp-%d", i)}
		switch i % 4 {
		case 0:
			imps[i].Banner = &openrtb2.Banner{}
		case 1:
			imps[i].Video = &openrtb2.Video{}
		case 2:
			imps[i].Audio = &openrtb2.Audio{}
		case 3:
			imps[i].Native = &openrtb2.Native{}
		}
	}
	// Bid that points at the LAST imp — worst-case linear scan.
	bid := &openrtb2.Bid{ImpID: "imp-9"}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = getBidType(bid, imps)
	}
}
