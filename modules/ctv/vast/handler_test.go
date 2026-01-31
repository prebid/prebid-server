package vast

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/modules/ctv/vast/pipeline"
)

func TestVastHandler_GET_Success(t *testing.T) {
	// Create mock bid response with VAST
	mockResp := &openrtb2.BidResponse{
		ID: "test-response",
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "testbidder",
				Bid: []openrtb2.Bid{
					{
						ID:    "bid-123",
						ImpID: "imp1",
						Price: 10.50,
						AdM: `<VAST version="3.0">
  <Ad id="test-ad">
    <InLine>
      <AdSystem>TestBidder</AdSystem>
      <AdTitle>Test Creative</AdTitle>
      <Creatives>
        <Creative>
          <Linear>
            <Duration>00:00:30</Duration>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
</VAST>`,
					},
				},
			},
		},
	}

	// Create test handler with mock response
	handler := &VastHandlerForTesting{
		MockBidResponse: mockResp,
		MockConfig: ReceiverConfig{
			Receiver:           "GAM_SSU",
			VastVersionDefault: "3.0",
			DefaultCurrency:    "USD",
			MaxAdsInPod:        1,
			SelectionStrategy:  "SINGLE",
			AllowSkeletonVast:  true,
			CollisionPolicy:    CollisionPolicyVastWins,
			PlacementRules: PlacementRules{
				PricingPlacement:    PlacementInline,
				AdvertiserPlacement: PlacementInline,
				CategoriesPlacement: PlacementExtensions,
			},
		},
	}

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/vast?imp=imp1", nil)
	w := httptest.NewRecorder()

	// Handle request
	handler.ServeHTTP(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/xml" {
		t.Errorf("Expected Content-Type application/xml, got %s", contentType)
	}

	body := w.Body.String()
	if body == "" {
		t.Fatal("Expected non-empty response body")
	}

	// Verify VAST structure
	if !strings.Contains(body, "<VAST") {
		t.Error("Response should contain <VAST tag")
	}
	if !strings.Contains(body, `version="3.0"`) {
		t.Error("Response should contain version 3.0")
	}
	if !strings.Contains(body, "<Ad id=") {
		t.Error("Response should contain Ad element")
	}
	if !strings.Contains(body, "<InLine>") {
		t.Error("Response should contain InLine element")
	}
	if !strings.Contains(body, "<AdSystem>") {
		t.Error("Response should contain AdSystem element")
	}

	t.Logf("Response XML:\n%s", body)
}

func TestVastHandler_GET_MultipleBids(t *testing.T) {
	// Create mock bid response with multiple bids
	mockResp := &openrtb2.BidResponse{
		ID: "test-response",
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "bidder1",
				Bid: []openrtb2.Bid{
					{
						ID:    "bid-001",
						ImpID: "imp1",
						Price: 15.00,
						AdM:   `<VAST version="3.0"><Ad id="ad1"><InLine><AdSystem>Bidder1</AdSystem><AdTitle>Ad 1</AdTitle></InLine></Ad></VAST>`,
					},
				},
			},
			{
				Seat: "bidder2",
				Bid: []openrtb2.Bid{
					{
						ID:    "bid-002",
						ImpID: "imp1",
						Price: 12.00,
						AdM:   `<VAST version="3.0"><Ad id="ad2"><InLine><AdSystem>Bidder2</AdSystem><AdTitle>Ad 2</AdTitle></InLine></Ad></VAST>`,
					},
				},
			},
			{
				Seat: "bidder3",
				Bid: []openrtb2.Bid{
					{
						ID:    "bid-003",
						ImpID: "imp1",
						Price: 10.00,
						AdM:   `<VAST version="3.0"><Ad id="ad3"><InLine><AdSystem>Bidder3</AdSystem><AdTitle>Ad 3</AdTitle></InLine></Ad></VAST>`,
					},
				},
			},
		},
	}

	// Create test handler configured for TOP_N selection
	handler := &VastHandlerForTesting{
		MockBidResponse: mockResp,
		MockConfig: ReceiverConfig{
			Receiver:           "GAM_SSU",
			VastVersionDefault: "4.0",
			DefaultCurrency:    "USD",
			MaxAdsInPod:        3,
			SelectionStrategy:  "TOP_N",
			AllowSkeletonVast:  true,
			CollisionPolicy:    CollisionPolicyVastWins,
			PlacementRules: PlacementRules{
				PricingPlacement:    PlacementInline,
				AdvertiserPlacement: PlacementInline,
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/vast?imp=imp1", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()

	// Verify VAST 4.0
	if !strings.Contains(body, `version="4.0"`) {
		t.Error("Expected VAST version 4.0")
	}

	// Verify multiple ads with sequences
	if !strings.Contains(body, `sequence="1"`) {
		t.Error("Expected sequence 1")
	}
	if !strings.Contains(body, `sequence="2"`) {
		t.Error("Expected sequence 2")
	}
	if !strings.Contains(body, `sequence="3"`) {
		t.Error("Expected sequence 3")
	}

	// Count number of Ad elements (should be 3)
	adCount := strings.Count(body, "<Ad id=")
	if adCount != 3 {
		t.Errorf("Expected 3 Ad elements, found %d", adCount)
	}

	t.Logf("Response XML:\n%s", body)
}

func TestVastHandler_GET_NoBids(t *testing.T) {
	// Create mock bid response with no bids
	mockResp := &openrtb2.BidResponse{
		ID:      "test-response",
		SeatBid: []openrtb2.SeatBid{},
	}

	handler := &VastHandlerForTesting{
		MockBidResponse: mockResp,
		MockConfig: ReceiverConfig{
			Receiver:           "GAM_SSU",
			VastVersionDefault: "3.0",
			DefaultCurrency:    "USD",
			MaxAdsInPod:        1,
			SelectionStrategy:  "SINGLE",
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/vast?imp=imp1", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()

	// Should return empty VAST
	if !strings.Contains(body, "<VAST") {
		t.Error("Expected VAST structure")
	}
	if !strings.Contains(body, `version="3.0"`) {
		t.Error("Expected version 3.0")
	}

	// Should not contain any Ad elements
	if strings.Contains(body, "<Ad") {
		t.Error("No-bid response should not contain Ad elements")
	}

	t.Logf("No-bid response:\n%s", body)
}

func TestVastHandler_POST_NotAllowed(t *testing.T) {
	handler := &VastHandlerForTesting{
		MockBidResponse: &openrtb2.BidResponse{},
		MockConfig:      ReceiverConfig{},
	}

	req := httptest.NewRequest(http.MethodPost, "/vast", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestVastHandler_DebugHandler(t *testing.T) {
	mockResp := &openrtb2.BidResponse{
		ID: "test-response",
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "testbidder",
				Bid: []openrtb2.Bid{
					{
						ID:    "bid-123",
						ImpID: "imp1",
						Price: 10.50,
						AdM:   `<VAST version="3.0"><Ad id="test"><InLine><AdSystem>Test</AdSystem><AdTitle>Ad</AdTitle></InLine></Ad></VAST>`,
					},
				},
			},
		},
	}

	handler := &VastHandlerForTesting{
		MockBidResponse: mockResp,
		MockConfig: ReceiverConfig{
			Receiver:           "GAM_SSU",
			VastVersionDefault: "3.0",
			SelectionStrategy:  "SINGLE",
			AllowSkeletonVast:  true,
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/vast/debug", nil)
	w := httptest.NewRecorder()

	handler.DebugHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	body := w.Body.String()
	if body == "" {
		t.Fatal("Expected non-empty response body")
	}

	// Verify JSON structure
	if !strings.Contains(body, `"no_ad"`) {
		t.Error("Expected no_ad field in JSON")
	}
	if !strings.Contains(body, `"warnings"`) {
		t.Error("Expected warnings field in JSON")
	}
	if !strings.Contains(body, `"selected_count"`) {
		t.Error("Expected selected_count field in JSON")
	}
	if !strings.Contains(body, `"vast_xml"`) {
		t.Error("Expected vast_xml field in JSON")
	}

	t.Logf("Debug response:\n%s", body)
}

func TestBuildVastFromBidResponse_Integration(t *testing.T) {
	// Test the BuildVastFromBidResponse function directly
	w := int64(640)
	h := int64(480)
	req := &openrtb2.BidRequest{
		ID: "test-req",
		Imp: []openrtb2.Imp{
			{
				ID: "imp1",
				Video: &openrtb2.Video{
					W:           &w,
					H:           &h,
					MinDuration: 15,
					MaxDuration: 30,
				},
			},
		},
	}

	resp := &openrtb2.BidResponse{
		ID: "test-resp",
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "testbidder",
				Bid: []openrtb2.Bid{
					{
						ID:    "bid-abc",
						ImpID: "imp1",
						Price: 12.50,
						AdM:   `<VAST version="3.0"><Ad id="original"><InLine><AdSystem>Original</AdSystem><AdTitle>Test</AdTitle></InLine></Ad></VAST>`,
					},
				},
			},
		},
	}

	cfg := ReceiverConfig{
		Receiver:           "GAM_SSU",
		VastVersionDefault: "3.0",
		DefaultCurrency:    "USD",
		MaxAdsInPod:        1,
		SelectionStrategy:  "SINGLE",
		AllowSkeletonVast:  true,
		CollisionPolicy:    CollisionPolicyVastWins,
		PlacementRules: PlacementRules{
			PricingPlacement:    PlacementInline,
			AdvertiserPlacement: PlacementInline,
		},
	}

	result, err := pipeline.BuildVastFromBidResponse(context.Background(), req, resp, cfg)

	if err != nil {
		t.Fatalf("BuildVastFromBidResponse failed: %v", err)
	}

	if result.NoAd {
		t.Error("Expected NoAd to be false")
	}

	if len(result.VastXML) == 0 {
		t.Error("Expected non-empty VAST XML")
	}

	if len(result.Selected) != 1 {
		t.Errorf("Expected 1 selected bid, got %d", len(result.Selected))
	}

	vastStr := string(result.VastXML)
	if !strings.Contains(vastStr, "<VAST") {
		t.Error("Expected VAST structure")
	}

	if !strings.Contains(vastStr, `version="3.0"`) {
		t.Error("Expected VAST version 3.0")
	}

	// Verify the bid ID is used in the Ad element
	if !strings.Contains(vastStr, "bid-abc") {
		t.Error("Expected bid ID in Ad element")
	}

	t.Logf("VAST Result:\n%s", vastStr)
	t.Logf("Warnings: %v", result.Warnings)
}
