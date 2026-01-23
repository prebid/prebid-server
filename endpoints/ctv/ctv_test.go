package ctv

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/exchange"
	"github.com/prebid/prebid-server/v3/logger"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/ortb"
	"github.com/prebid/prebid-server/v3/stored_responses"
	"github.com/stretchr/testify/mock"
)

// Mock implementations for testing

type mockUUIDGenerator struct{}

func (m *mockUUIDGenerator) Generate() (string, error) {
	return "test-request-id", nil
}

type mockExchange struct {
	response *exchange.AuctionResponse
	err      error
}

func (m *mockExchange) HoldAuction(ctx context.Context, r *exchange.AuctionRequest, debugLog *exchange.DebugLog) (*exchange.AuctionResponse, error) {
	return m.response, m.err
}

type mockValidator struct{}

func (m *mockValidator) ValidateImp(imp *openrtb_ext.ImpWrapper, cfg ortb.ValidationConfig, index int, aliases map[string]string, hasStoredAuctionResponses bool, storedBidResponses stored_responses.ImpBidderStoredResp) []error {
	return nil
}

type mockFetcher struct{}

func (m *mockFetcher) FetchRequests(ctx context.Context, ids []string, impIDs []string) (map[string]json.RawMessage, map[string]json.RawMessage, []error) {
	return nil, nil, nil
}

func (m *mockFetcher) FetchResponses(ctx context.Context, ids []string) (map[string]json.RawMessage, []error) {
	return nil, nil
}

type mockAccountFetcher struct{}

func (m *mockAccountFetcher) FetchAccount(ctx context.Context, defaultAccountJSON json.RawMessage, accountID string) (json.RawMessage, []error) {
	return nil, nil
}

func TestCTVEndpoint_Success(t *testing.T) {
	// Create mock auction response with a winning bid
	auctionResponse := &exchange.AuctionResponse{
		BidResponse: &openrtb2.BidResponse{
			ID:  "test-response",
			Cur: "USD",
			SeatBid: []openrtb2.SeatBid{
				{
					Seat: "testbidder",
					Bid: []openrtb2.Bid{
						{
							ID:      "bid1",
							ImpID:   "1",
							Price:   5.50,
							ADomain: []string{"example.com"},
							AdM:     `<VAST version="3.0"><Ad id="test"><InLine><AdSystem>Test</AdSystem><AdTitle>Test Ad</AdTitle><Impression>http://example.com/imp</Impression><Creatives><Creative><Linear><Duration>00:00:30</Duration></Linear></Creative></Creatives></InLine></Ad></VAST>`,
						},
					},
				},
			},
		},
	}

	mockMetrics := &metrics.MetricsEngineMock{}
	mockMetrics.On("RecordRequest", mock.Anything).Return()
	mockMetrics.On("RecordRequestTime", mock.Anything, mock.Anything).Return()

	deps := &CTVEndpointDeps{
		uuidGenerator:    &mockUUIDGenerator{},
		ex:               &mockExchange{response: auctionResponse, err: nil},
		requestValidator: &mockValidator{},
		storedReqFetcher: &mockFetcher{},
		accounts:         &mockAccountFetcher{},
		cfg: &config.Configuration{
			TmaxDefault: 1000,
		},
		metricsEngine: mockMetrics,
		logger:        logger.NewGlogLogger(),
	}

	// Create test request
	req := httptest.NewRequest("GET", "/ctv/vast?publisher_id=test-pub", nil)
	w := httptest.NewRecorder()

	// Handle request
	deps.HandleVast(w, req, nil)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/xml") {
		t.Errorf("Expected Content-Type application/xml, got %s", contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, "<VAST") {
		t.Error("Expected VAST XML in response")
	}

	if !strings.Contains(body, "version=") {
		t.Error("Expected VAST version in response")
	}
}

func TestCTVEndpoint_NoBids(t *testing.T) {
	// Empty auction response
	auctionResponse := &exchange.AuctionResponse{
		BidResponse: &openrtb2.BidResponse{
			ID:      "test-response",
			SeatBid: []openrtb2.SeatBid{},
		},
	}

	mockMetrics := &metrics.MetricsEngineMock{}
	mockMetrics.On("RecordRequest", mock.Anything).Return()
	mockMetrics.On("RecordRequestTime", mock.Anything, mock.Anything).Return()

	deps := &CTVEndpointDeps{
		uuidGenerator:    &mockUUIDGenerator{},
		ex:               &mockExchange{response: auctionResponse, err: nil},
		requestValidator: &mockValidator{},
		storedReqFetcher: &mockFetcher{},
		accounts:         &mockAccountFetcher{},
		cfg: &config.Configuration{
			TmaxDefault: 1000,
		},
		metricsEngine: mockMetrics,
		logger:        logger.NewGlogLogger(),
	}

	req := httptest.NewRequest("GET", "/ctv/vast?publisher_id=test-pub", nil)
	w := httptest.NewRecorder()

	deps.HandleVast(w, req, nil)

	// Should still return 200 with empty VAST
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "<VAST") {
		t.Error("Expected VAST XML in response")
	}

	// Empty VAST should not have Ad elements
	if strings.Contains(body, "<Ad") {
		t.Error("Expected no Ad elements in empty VAST")
	}
}

func TestCTVEndpoint_AuctionError(t *testing.T) {
	mockMetrics := &metrics.MetricsEngineMock{}
	mockMetrics.On("RecordRequest", mock.Anything).Return()
	mockMetrics.On("RecordRequestTime", mock.Anything, mock.Anything).Return()

	deps := &CTVEndpointDeps{
		uuidGenerator:    &mockUUIDGenerator{},
		ex:               &mockExchange{response: nil, err: errors.New("auction error")},
		requestValidator: &mockValidator{},
		storedReqFetcher: &mockFetcher{},
		accounts:         &mockAccountFetcher{},
		cfg: &config.Configuration{
			TmaxDefault: 1000,
		},
		metricsEngine: mockMetrics,
		logger:        logger.NewGlogLogger(),
	}

	req := httptest.NewRequest("GET", "/ctv/vast?publisher_id=test-pub", nil)
	w := httptest.NewRecorder()

	deps.HandleVast(w, req, nil)

	// Should return empty VAST on error
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "<VAST") {
		t.Error("Expected VAST XML in response even on error")
	}
}

func TestParseQueryParams(t *testing.T) {
	deps := &CTVEndpointDeps{}

	tests := []struct {
		name     string
		url      string
		validate func(*testing.T, *CTVQueryParams)
	}{
		{
			name: "Basic params",
			url:  "/ctv/vast?publisher_id=pub123&width=1920&height=1080",
			validate: func(t *testing.T, params *CTVQueryParams) {
				if params.PublisherID != "pub123" {
					t.Errorf("Expected publisher_id pub123, got %s", params.PublisherID)
				}
				if params.Width != 1920 {
					t.Errorf("Expected width 1920, got %d", params.Width)
				}
				if params.Height != 1080 {
					t.Errorf("Expected height 1080, got %d", params.Height)
				}
			},
		},
		{
			name: "Duration params",
			url:  "/ctv/vast?publisher_id=pub123&min_duration=15&max_duration=30",
			validate: func(t *testing.T, params *CTVQueryParams) {
				if params.MinDuration != 15 {
					t.Errorf("Expected min_duration 15, got %d", params.MinDuration)
				}
				if params.MaxDuration != 30 {
					t.Errorf("Expected max_duration 30, got %d", params.MaxDuration)
				}
			},
		},
		{
			name: "Debug flag",
			url:  "/ctv/vast?publisher_id=pub123&debug=1",
			validate: func(t *testing.T, params *CTVQueryParams) {
				if !params.DebugEnabled {
					t.Error("Expected debug to be enabled")
				}
			},
		},
		{
			name: "Custom macros",
			url:  "/ctv/vast?publisher_id=pub123&custom_macro=value1&another=value2",
			validate: func(t *testing.T, params *CTVQueryParams) {
				if len(params.Macros) < 2 {
					t.Errorf("Expected at least 2 macros, got %d", len(params.Macros))
				}
				if params.Macros["custom_macro"] != "value1" {
					t.Errorf("Expected custom_macro=value1, got %s", params.Macros["custom_macro"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			params, err := deps.parseQueryParams(req)
			if err != nil {
				t.Fatalf("parseQueryParams failed: %v", err)
			}
			tt.validate(t, params)
		})
	}
}

func TestBuildBidRequest(t *testing.T) {
	deps := &CTVEndpointDeps{
		uuidGenerator: &mockUUIDGenerator{},
		cfg: &config.Configuration{
			TmaxDefault: 1000,
		},
	}

	params := &CTVQueryParams{
		PublisherID: "test-pub",
		Width:       1920,
		Height:      1080,
		MinDuration: 15,
		MaxDuration: 30,
	}

	vastConfig := config.CTVVastDefaults()

	bidRequest, err := deps.buildBidRequest(context.Background(), params, vastConfig)
	if err != nil {
		t.Fatalf("buildBidRequest failed: %v", err)
	}

	if bidRequest.ID != "test-request-id" {
		t.Errorf("Expected request ID test-request-id, got %s", bidRequest.ID)
	}

	if len(bidRequest.Imp) != 1 {
		t.Fatalf("Expected 1 impression, got %d", len(bidRequest.Imp))
	}

	imp := bidRequest.Imp[0]
	if imp.Video == nil {
		t.Fatal("Expected video impression")
	}

	if *imp.Video.W != 1920 {
		t.Errorf("Expected width 1920, got %d", *imp.Video.W)
	}

	if *imp.Video.H != 1080 {
		t.Errorf("Expected height 1080, got %d", *imp.Video.H)
	}

	if imp.Video.MinDuration != 15 {
		t.Errorf("Expected min duration 15, got %d", imp.Video.MinDuration)
	}

	if imp.Video.MaxDuration != 30 {
		t.Errorf("Expected max duration 30, got %d", imp.Video.MaxDuration)
	}

	if bidRequest.Site == nil || bidRequest.Site.Publisher == nil {
		t.Fatal("Expected publisher in site")
	}

	if bidRequest.Site.Publisher.ID != "test-pub" {
		t.Errorf("Expected publisher ID test-pub, got %s", bidRequest.Site.Publisher.ID)
	}
}

func TestIsReservedParam(t *testing.T) {
	tests := []struct {
		param    string
		expected bool
	}{
		{"publisher_id", true},
		{"width", true},
		{"height", true},
		{"debug", true},
		{"custom_macro", false},
		{"anything_else", false},
	}

	for _, tt := range tests {
		t.Run(tt.param, func(t *testing.T) {
			result := isReservedParam(tt.param)
			if result != tt.expected {
				t.Errorf("isReservedParam(%s) = %v; expected %v", tt.param, result, tt.expected)
			}
		})
	}
}

func TestBuildEnricherConfig(t *testing.T) {
	deps := &CTVEndpointDeps{}

	vastConfig := config.CTVVastDefaults()
	vastConfig.CollisionPolicy = "OPENRTB_WINS"
	vastConfig.DefaultCurrency = "EUR"
	vastConfig.IncludeDebugIDs = true
	vastConfig.PlacementRules.Price = "EXTENSIONS"

	enrichConfig := deps.buildEnricherConfig(vastConfig)

	if string(enrichConfig.CollisionPolicy) != "OPENRTB_WINS" {
		t.Errorf("Expected collision policy OPENRTB_WINS, got %s", enrichConfig.CollisionPolicy)
	}

	if enrichConfig.DefaultCurrency != "EUR" {
		t.Errorf("Expected currency EUR, got %s", enrichConfig.DefaultCurrency)
	}

	if !enrichConfig.IncludeDebugIDs {
		t.Error("Expected IncludeDebugIDs to be true")
	}

	if string(enrichConfig.PlacementRules.Price) != "EXTENSIONS" {
		t.Errorf("Expected price placement EXTENSIONS, got %s", enrichConfig.PlacementRules.Price)
	}
}
