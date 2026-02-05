package vast

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/modules/prebid/ctv_vast_enrichment/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock implementations for testing

type mockSelector struct {
	selectFn func(req *openrtb2.BidRequest, resp *openrtb2.BidResponse, cfg ReceiverConfig) ([]SelectedBid, []string, error)
}

func (m *mockSelector) Select(req *openrtb2.BidRequest, resp *openrtb2.BidResponse, cfg ReceiverConfig) ([]SelectedBid, []string, error) {
	if m.selectFn != nil {
		return m.selectFn(req, resp, cfg)
	}
	// Default: select all bids with sequence numbers
	var selected []SelectedBid
	seq := 1
	if resp != nil {
		for _, sb := range resp.SeatBid {
			for _, bid := range sb.Bid {
				adomain := ""
				if len(bid.ADomain) > 0 {
					adomain = bid.ADomain[0]
				}
				selected = append(selected, SelectedBid{
					Bid:      bid,
					Seat:     sb.Seat,
					Sequence: seq,
					Meta: CanonicalMeta{
						BidID:    bid.ID,
						Seat:     sb.Seat,
						Price:    bid.Price,
						Currency: resp.Cur,
						Adomain:  adomain,
						Cats:     bid.Cat,
					},
				})
				seq++
			}
		}
	}
	return selected, nil, nil
}

type mockEnricher struct {
	enrichFn func(ad *model.Ad, meta CanonicalMeta, cfg ReceiverConfig) ([]string, error)
}

func (m *mockEnricher) Enrich(ad *model.Ad, meta CanonicalMeta, cfg ReceiverConfig) ([]string, error) {
	if m.enrichFn != nil {
		return m.enrichFn(ad, meta, cfg)
	}
	// Default: add pricing extension and advertiser
	if ad.InLine != nil {
		ad.InLine.Pricing = &model.Pricing{
			Model:    "CPM",
			Currency: cfg.DefaultCurrency,
			Value:    formatPrice(meta.Price),
		}
		if meta.Adomain != "" {
			ad.InLine.Advertiser = meta.Adomain
		}
		if cfg.Debug {
			if ad.InLine.Extensions == nil {
				ad.InLine.Extensions = &model.Extensions{}
			}
			debugXML := fmt.Sprintf("<BidID>%s</BidID><Seat>%s</Seat><Price>%f</Price>",
				meta.BidID, meta.Seat, meta.Price)
			ad.InLine.Extensions.Extension = append(ad.InLine.Extensions.Extension, model.ExtensionXML{
				Type:     "openrtb",
				InnerXML: debugXML,
			})
		}
	}
	return nil, nil
}

func formatPrice(price float64) string {
	return fmt.Sprintf("%.2f", price)
}

type mockFormatter struct {
	formatFn func(ads []EnrichedAd, cfg ReceiverConfig) ([]byte, []string, error)
}

func (m *mockFormatter) Format(ads []EnrichedAd, cfg ReceiverConfig) ([]byte, []string, error) {
	if m.formatFn != nil {
		return m.formatFn(ads, cfg)
	}
	// Default: build GAM SSU style VAST
	version := cfg.VastVersionDefault
	if version == "" {
		version = "4.0"
	}
	vast := &model.Vast{
		Version: version,
		Ads:     make([]model.Ad, 0, len(ads)),
	}
	for _, ea := range ads {
		ad := *ea.Ad
		ad.ID = ea.Meta.BidID
		ad.Sequence = ea.Sequence
		vast.Ads = append(vast.Ads, ad)
	}
	xml, err := vast.Marshal()
	return xml, nil, err
}

func newTestComponents() (BidSelector, Enricher, Formatter) {
	return &mockSelector{}, &mockEnricher{}, &mockFormatter{}
}

func TestBuildVastFromBidResponse_NoAds(t *testing.T) {
	cfg := DefaultConfig()
	req := &openrtb2.BidRequest{ID: "test-req"}
	resp := &openrtb2.BidResponse{ID: "test-resp"}

	selector, enricher, formatter := newTestComponents()
	result, err := BuildVastFromBidResponse(context.Background(), req, resp, cfg, selector, enricher, formatter)
	require.NoError(t, err)

	assert.True(t, result.NoAd)
	assert.NotEmpty(t, result.VastXML)
	assert.Contains(t, string(result.VastXML), `<VAST version="4.0">`)
	assert.Empty(t, result.Selected)
}

func TestBuildVastFromBidResponse_NilResponse(t *testing.T) {
	cfg := DefaultConfig()
	req := &openrtb2.BidRequest{ID: "test-req"}

	selector, enricher, formatter := newTestComponents()
	result, err := BuildVastFromBidResponse(context.Background(), req, nil, cfg, selector, enricher, formatter)
	require.NoError(t, err)

	assert.True(t, result.NoAd)
	assert.NotEmpty(t, result.VastXML)
}

func TestBuildVastFromBidResponse_SingleBid(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SelectionStrategy = SelectionSingle

	vastXML := `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="4.0">
  <Ad id="ad-123">
    <InLine>
      <AdSystem>TestServer</AdSystem>
      <AdTitle>Test Ad</AdTitle>
      <Creatives>
        <Creative>
          <Linear>
            <Duration>00:00:30</Duration>
            <MediaFiles>
              <MediaFile delivery="progressive" type="video/mp4" width="1920" height="1080">
                <![CDATA[https://example.com/video.mp4]]>
              </MediaFile>
            </MediaFiles>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
</VAST>`

	req := &openrtb2.BidRequest{ID: "test-req"}
	resp := &openrtb2.BidResponse{
		ID: "test-resp",
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "bidder1",
				Bid: []openrtb2.Bid{
					{
						ID:      "bid-1",
						ImpID:   "imp-1",
						Price:   5.0,
						AdM:     vastXML,
						ADomain: []string{"advertiser.com"},
					},
				},
			},
		},
	}

	selector, enricher, formatter := newTestComponents()
	result, err := BuildVastFromBidResponse(context.Background(), req, resp, cfg, selector, enricher, formatter)
	require.NoError(t, err)

	assert.False(t, result.NoAd)
	assert.NotEmpty(t, result.VastXML)
	assert.Len(t, result.Selected, 1)

	xmlStr := string(result.VastXML)
	assert.Contains(t, xmlStr, `<VAST version="4.0">`)
	assert.Contains(t, xmlStr, `<Ad id="bid-1"`)
	assert.Contains(t, xmlStr, "<AdTitle>Test Ad</AdTitle>")
}

func TestBuildVastFromBidResponse_MultipleBids(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SelectionStrategy = SelectionTopN
	cfg.MaxAdsInPod = 3

	makeVAST := func(adID, title string) string {
		return `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="4.0">
  <Ad id="` + adID + `">
    <InLine>
      <AdSystem>TestServer</AdSystem>
      <AdTitle>` + title + `</AdTitle>
      <Creatives>
        <Creative>
          <Linear>
            <Duration>00:00:15</Duration>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
</VAST>`
	}

	req := &openrtb2.BidRequest{ID: "test-req"}
	resp := &openrtb2.BidResponse{
		ID: "test-resp",
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "bidder1",
				Bid: []openrtb2.Bid{
					{ID: "bid-1", ImpID: "imp-1", Price: 10.0, AdM: makeVAST("ad-1", "First Ad")},
					{ID: "bid-2", ImpID: "imp-2", Price: 8.0, AdM: makeVAST("ad-2", "Second Ad")},
					{ID: "bid-3", ImpID: "imp-3", Price: 5.0, AdM: makeVAST("ad-3", "Third Ad")},
				},
			},
		},
	}

	selector, enricher, formatter := newTestComponents()
	result, err := BuildVastFromBidResponse(context.Background(), req, resp, cfg, selector, enricher, formatter)
	require.NoError(t, err)

	assert.False(t, result.NoAd)
	assert.Len(t, result.Selected, 3)

	xmlStr := string(result.VastXML)
	assert.Contains(t, xmlStr, `sequence="1"`)
	assert.Contains(t, xmlStr, `sequence="2"`)
	assert.Contains(t, xmlStr, `sequence="3"`)
}

func TestBuildVastFromBidResponse_SkeletonVast(t *testing.T) {
	cfg := DefaultConfig()
	cfg.AllowSkeletonVast = true
	cfg.SelectionStrategy = SelectionSingle

	req := &openrtb2.BidRequest{ID: "test-req"}
	resp := &openrtb2.BidResponse{
		ID: "test-resp",
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "bidder1",
				Bid: []openrtb2.Bid{
					{
						ID:    "bid-1",
						ImpID: "imp-1",
						Price: 5.0,
						AdM:   "not-valid-vast", // Invalid VAST
					},
				},
			},
		},
	}

	selector, enricher, formatter := newTestComponents()
	result, err := BuildVastFromBidResponse(context.Background(), req, resp, cfg, selector, enricher, formatter)
	require.NoError(t, err)

	// Should succeed with skeleton VAST
	assert.False(t, result.NoAd)
	assert.NotEmpty(t, result.VastXML)
	// Check for skeleton warning
	hasSkeletonWarning := false
	for _, w := range result.Warnings {
		if strings.Contains(strings.ToLower(w), "skeleton") {
			hasSkeletonWarning = true
			break
		}
	}
	assert.True(t, hasSkeletonWarning, "Expected skeleton warning, got: %v", result.Warnings)
}

func TestBuildVastFromBidResponse_InvalidVastNoSkeleton(t *testing.T) {
	cfg := DefaultConfig()
	cfg.AllowSkeletonVast = false // Don't allow skeleton
	cfg.SelectionStrategy = SelectionSingle

	req := &openrtb2.BidRequest{ID: "test-req"}
	resp := &openrtb2.BidResponse{
		ID: "test-resp",
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "bidder1",
				Bid: []openrtb2.Bid{
					{
						ID:    "bid-1",
						ImpID: "imp-1",
						Price: 5.0,
						AdM:   "not-valid-vast",
					},
				},
			},
		},
	}

	selector, enricher, formatter := newTestComponents()
	result, err := BuildVastFromBidResponse(context.Background(), req, resp, cfg, selector, enricher, formatter)
	require.NoError(t, err)

	// Should return no-ad since parse failed and skeleton not allowed
	assert.True(t, result.NoAd)
}

func TestBuildVastFromBidResponse_EnrichmentAddsMetadata(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SelectionStrategy = SelectionSingle
	cfg.Debug = true // Enable debug extensions

	vastXML := `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="4.0">
  <Ad id="ad-123">
    <InLine>
      <AdSystem>TestServer</AdSystem>
      <AdTitle>Test Ad</AdTitle>
      <Creatives>
        <Creative>
          <Linear>
            <MediaFiles>
              <MediaFile delivery="progressive" type="video/mp4" width="1920" height="1080">
                <![CDATA[https://example.com/video.mp4]]>
              </MediaFile>
            </MediaFiles>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
</VAST>`

	req := &openrtb2.BidRequest{ID: "test-req"}
	resp := &openrtb2.BidResponse{
		ID:  "test-resp",
		Cur: "USD",
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "bidder1",
				Bid: []openrtb2.Bid{
					{
						ID:      "bid-enriched",
						ImpID:   "imp-1",
						Price:   7.5,
						AdM:     vastXML,
						ADomain: []string{"advertiser.com"},
						Cat:     []string{"IAB1", "IAB2"},
					},
				},
			},
		},
	}

	selector, enricher, formatter := newTestComponents()
	result, err := BuildVastFromBidResponse(context.Background(), req, resp, cfg, selector, enricher, formatter)
	require.NoError(t, err)
	require.False(t, result.NoAd)

	xmlStr := string(result.VastXML)
	// Check enrichment added pricing
	assert.Contains(t, xmlStr, "<Pricing")
	// Check enrichment added advertiser
	assert.Contains(t, xmlStr, "advertiser.com")
	// Check debug extension
	assert.Contains(t, xmlStr, `type="openrtb"`)
	assert.Contains(t, xmlStr, "<BidID>bid-enriched</BidID>")
}

// HTTP Handler Tests

func TestHandler_MethodNotAllowed(t *testing.T) {
	selector, enricher, formatter := newTestComponents()
	handler := NewHandler().
		WithSelector(selector).
		WithEnricher(enricher).
		WithFormatter(formatter)

	req := httptest.NewRequest(http.MethodPost, "/vast", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestHandler_NotConfigured(t *testing.T) {
	handler := NewHandler() // No selector/enricher/formatter

	req := httptest.NewRequest(http.MethodGet, "/vast", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	body, _ := io.ReadAll(rec.Body)
	assert.Contains(t, string(body), "not properly configured")
}

func TestHandler_NoAuction_ReturnsNoAdVast(t *testing.T) {
	selector, enricher, formatter := newTestComponents()
	handler := NewHandler().
		WithSelector(selector).
		WithEnricher(enricher).
		WithFormatter(formatter)
	// No AuctionFunc set, should return no-ad VAST

	req := httptest.NewRequest(http.MethodGet, "/vast?pod_id=test-pod", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/xml; charset=utf-8", rec.Header().Get("Content-Type"))

	body, _ := io.ReadAll(rec.Body)
	assert.Contains(t, string(body), `<VAST version="4.0">`)
}

func TestHandler_WithMockAuction_ReturnsVast(t *testing.T) {
	vastXML := `<?xml version="1.0" encoding="UTF-8"?>
<VAST version="4.0">
  <Ad id="mock-ad">
    <InLine>
      <AdSystem>MockServer</AdSystem>
      <AdTitle>Mock Ad</AdTitle>
      <Creatives>
        <Creative>
          <Linear>
            <Duration>00:00:15</Duration>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
</VAST>`

	mockAuction := func(ctx context.Context, req *openrtb2.BidRequest) (*openrtb2.BidResponse, error) {
		return &openrtb2.BidResponse{
			ID: "mock-resp",
			SeatBid: []openrtb2.SeatBid{
				{
					Seat: "mock-bidder",
					Bid: []openrtb2.Bid{
						{
							ID:    "mock-bid-1",
							ImpID: "imp-1",
							Price: 3.50,
							AdM:   vastXML,
						},
					},
				},
			},
		}, nil
	}

	selector, enricher, formatter := newTestComponents()
	handler := NewHandler().
		WithSelector(selector).
		WithEnricher(enricher).
		WithFormatter(formatter).
		WithAuctionFunc(mockAuction)

	req := httptest.NewRequest(http.MethodGet, "/vast?pod_id=test-pod", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/xml; charset=utf-8", rec.Header().Get("Content-Type"))

	body, _ := io.ReadAll(rec.Body)
	xmlStr := string(body)
	assert.Contains(t, xmlStr, `<VAST version="4.0">`)
	assert.Contains(t, xmlStr, `<Ad id="mock-bid-1"`)
	assert.Contains(t, xmlStr, "<AdTitle>Mock Ad</AdTitle>")
}

func TestHandler_WithConfig(t *testing.T) {
	cfg := ReceiverConfig{
		Receiver:           ReceiverGAMSSU,
		VastVersionDefault: "3.0",
		DefaultCurrency:    "EUR",
	}

	selector, enricher, formatter := newTestComponents()
	handler := NewHandler().
		WithConfig(cfg).
		WithSelector(selector).
		WithEnricher(enricher).
		WithFormatter(formatter)

	req := httptest.NewRequest(http.MethodGet, "/vast", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	body, _ := io.ReadAll(rec.Body)
	// Should use version 3.0 from config
	assert.Contains(t, string(body), `version="3.0"`)
}

func TestHandler_CacheControlHeader(t *testing.T) {
	selector, enricher, formatter := newTestComponents()
	handler := NewHandler().
		WithSelector(selector).
		WithEnricher(enricher).
		WithFormatter(formatter)

	req := httptest.NewRequest(http.MethodGet, "/vast", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, "no-cache, no-store, must-revalidate", rec.Header().Get("Cache-Control"))
}

func TestHandler_PodIDFromQuery(t *testing.T) {
	var capturedReq *openrtb2.BidRequest

	mockAuction := func(ctx context.Context, req *openrtb2.BidRequest) (*openrtb2.BidResponse, error) {
		capturedReq = req
		return &openrtb2.BidResponse{}, nil
	}

	selector, enricher, formatter := newTestComponents()
	handler := NewHandler().
		WithSelector(selector).
		WithEnricher(enricher).
		WithFormatter(formatter).
		WithAuctionFunc(mockAuction)

	req := httptest.NewRequest(http.MethodGet, "/vast?pod_id=custom-pod-123", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.NotNil(t, capturedReq)
	assert.Equal(t, "custom-pod-123", capturedReq.ID)
}

// Test warnings are captured
func TestBuildVastFromBidResponse_WarningsCollected(t *testing.T) {
	cfg := DefaultConfig()
	cfg.AllowSkeletonVast = true

	// First bid has valid VAST, second has invalid
	validVAST := `<VAST version="4.0"><Ad id="valid"><InLine><AdSystem>Test</AdSystem><Creatives><Creative><Linear><Duration>00:00:15</Duration></Linear></Creative></Creatives></InLine></Ad></VAST>`

	req := &openrtb2.BidRequest{ID: "test-req"}
	resp := &openrtb2.BidResponse{
		ID: "test-resp",
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "bidder1",
				Bid: []openrtb2.Bid{
					{ID: "bid-1", ImpID: "imp-1", Price: 10.0, AdM: validVAST},
					{ID: "bid-2", ImpID: "imp-2", Price: 5.0, AdM: "invalid-vast"},
				},
			},
		},
	}

	selector, enricher, formatter := newTestComponents()
	result, err := BuildVastFromBidResponse(context.Background(), req, resp, cfg, selector, enricher, formatter)
	require.NoError(t, err)

	assert.False(t, result.NoAd)
	// Should have warnings about the invalid VAST using skeleton
	hasSkeletonWarning := false
	for _, w := range result.Warnings {
		if strings.Contains(strings.ToLower(w), "skeleton") {
			hasSkeletonWarning = true
			break
		}
	}
	assert.True(t, hasSkeletonWarning, "Expected skeleton warning in: %v", result.Warnings)
}
