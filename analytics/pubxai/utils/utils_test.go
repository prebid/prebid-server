package utils

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	useragentutil "github.com/prebid/prebid-server/v3/util/useragentutil"
	"github.com/stretchr/testify/assert"
)

func TestExtractUserIds(t *testing.T) {

	tests := []struct {
		name               string
		requestExt         map[string]interface{}
		expectedUserDetail UserDetail
	}{
		{
			name: "User IDs found",
			requestExt: map[string]interface{}{
				"user": map[string]interface{}{
					"ext": map[string]interface{}{
						"eids": []interface{}{
							map[string]interface{}{"source": "source1"},
							map[string]interface{}{"source": "source2"},
						},
					},
				},
			},
			expectedUserDetail: UserDetail{
				UserIdTypes: []string{"source1", "source2"},
			},
		},
		{
			name: "No user IDs",
			requestExt: map[string]interface{}{
				"user": map[string]interface{}{
					"ext": map[string]interface{}{
						"eids": []interface{}{},
					},
				},
			},
			expectedUserDetail: UserDetail{},
		},
		{
			name:               "Invalid user ext",
			requestExt:         map[string]interface{}{},
			expectedUserDetail: UserDetail{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userDetail := ExtractUserIds(tt.requestExt)
			assert.Equal(t, tt.expectedUserDetail, userDetail)
		})
	}
}

func TestExtractConsentTypes(t *testing.T) {

	tests := []struct {
		name            string
		requestExt      map[string]interface{}
		expectedConsent ConsentDetail
	}{
		{
			name: "Consent types found",
			requestExt: map[string]interface{}{
				"regs": map[string]interface{}{
					"ext": map[string]interface{}{
						"gdpr": 1,
						"ccpa": "1YNN",
					},
				},
			},
			expectedConsent: ConsentDetail{
				ConsentTypes: []string{"gdpr", "ccpa"},
			},
		},
		{
			name: "No consent types",
			requestExt: map[string]interface{}{
				"regs": map[string]interface{}{
					"ext": map[string]interface{}{},
				},
			},
			expectedConsent: ConsentDetail{},
		},
		{
			name:            "Invalid regs ext",
			requestExt:      map[string]interface{}{},
			expectedConsent: ConsentDetail{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualConsent := ExtractConsentTypes(tt.requestExt)
			assert.Equal(t, tt.expectedConsent, actualConsent)
		})
	}
}

func TestExtractDeviceData(t *testing.T) {

	tests := []struct {
		name           string
		requestExt     map[string]interface{}
		expectedDevice DeviceDetail
	}{
		{
			name: "Valid user agent",
			requestExt: map[string]interface{}{
				"device": map[string]interface{}{
					"ua": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
				},
			},
			expectedDevice: DeviceDetail{
				DeviceType: useragentutil.GetDeviceType("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
				DeviceOS:   useragentutil.GetOS("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
				Browser:    useragentutil.GetBrowser("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
			},
		},
		{
			name:           "Invalid device data",
			requestExt:     map[string]interface{}{},
			expectedDevice: DeviceDetail{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualDevice := ExtractDeviceData(tt.requestExt)
			assert.Equal(t, tt.expectedDevice, actualDevice)
		})
	}
}

func TestExtractPageData(t *testing.T) {

	tests := []struct {
		name         string
		requestExt   map[string]interface{}
		expectedPage PageDetail
	}{
		{
			name: "Valid site data with full URL",
			requestExt: map[string]interface{}{
				"site": map[string]interface{}{
					"domain": "example.com",
					"page":   "https://example.com/path?query=1",
				},
			},
			expectedPage: PageDetail{
				Host: "example.com",
				Path: "/path?query=1",
			},
		},
		{
			name: "Invalid site data",
			requestExt: map[string]interface{}{
				"site": map[string]interface{}{
					"domain": "example.com",
				},
			},
			expectedPage: PageDetail{
				Host: "example.com",
				Path: "",
			},
		},
		{
			name:         "No site data",
			requestExt:   map[string]interface{}{},
			expectedPage: PageDetail{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualPage := ExtractPageData(tt.requestExt)
			assert.Equal(t, tt.expectedPage, actualPage)
		})
	}
}

func TestExtractFloorDetail(t *testing.T) {

	tests := []struct {
		name          string
		requestExt    map[string]interface{}
		bidResponse   map[string]interface{}
		expectedFloor FloorDetail
	}{
		{
			name: "Valid floor data",
			requestExt: map[string]interface{}{
				"ext": map[string]interface{}{
					"prebid": map[string]interface{}{
						"floors": map[string]interface{}{
							"data": map[string]interface{}{
								"modelgroups": []interface{}{
									map[string]interface{}{
										"values": map[string]interface{}{
											"*|banner": 1.5,
										},
										"modelversion": "v1",
										"skiprate":     int64(10),
									},
								},
								"floorprovider": "provider",
							},
							"fetchstatus": "fetched",
							"location":    "location",
							"skipped":     true,
						},
					},
				},
				"imp": []interface{}{
					map[string]interface{}{
						"ext": map[string]interface{}{
							"prebid": map[string]interface{}{
								"floors": map[string]interface{}{
									"floorrule":      "*|banner",
									"floorrulevalue": 1.5,
								},
							},
						},
					},
				},
			},
			expectedFloor: FloorDetail{
				FloorProvider: "provider",
				ModelVersion:  "v1",
				SkipRate:      int64(10),
				FetchStatus:   "fetched",
				Location:      "location",
				Skipped:       true,
			},
		},
		{
			name:          "Invalid floor data",
			requestExt:    map[string]interface{}{},
			bidResponse:   map[string]interface{}{},
			expectedFloor: FloorDetail{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualFloor := ExtractFloorDetail(tt.requestExt)
			assert.Equal(t, tt.expectedFloor, actualFloor)
		})
	}
}

func TestExtractAdunitCodes(t *testing.T) {

	var emptyAdunits []string
	tests := []struct {
		name                string
		requestExt          map[string]interface{}
		expectedAdunitCodes []string
	}{
		{
			name: "Valid adunit codes",
			requestExt: map[string]interface{}{
				"imp": []interface{}{
					map[string]interface{}{
						"id": "adunit1",
					},
					map[string]interface{}{
						"id": "adunit2",
					},
				},
			},
			expectedAdunitCodes: []string{"adunit1", "adunit2"},
		},
		{
			name: "No adunit codes",
			requestExt: map[string]interface{}{
				"imp": []interface{}{},
			},
			expectedAdunitCodes: emptyAdunits,
		},
		{
			name:                "Invalid data structure",
			requestExt:          map[string]interface{}{},
			expectedAdunitCodes: emptyAdunits,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualAdunitCodes := ExtractAdunitCodes(tt.requestExt)
			assert.Equal(t, tt.expectedAdunitCodes, actualAdunitCodes)
		})
	}
}

func TestUnmarshalExtensions(t *testing.T) {

	tests := []struct {
		name           string
		logObject      *LogObject
		expectedReqExt map[string]interface{}
		expectedResExt map[string]interface{}
		expectError    bool
	}{
		{
			name: "Valid extensions",
			logObject: &LogObject{
				RequestWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID: "requestId",
					},
				},
				Response: &openrtb2.BidResponse{
					ID: "responseId",
					Ext: json.RawMessage(` {
						"responsetimemillis": {
							"appnexus": 148
						},
						"tmaxrequest": 1000,
						"prebid": {
							"auctiontimestamp": 1707996273474
						}
					}`),
				},
			},
			expectedReqExt: map[string]interface{}{
				"id":  "requestId",
				"imp": nil,
			},
			expectedResExt: map[string]interface{}{
				"responsetimemillis": map[string]interface{}{
					"appnexus": 148.0,
				},
				"tmaxrequest": 1000.0,
				"prebid": map[string]interface{}{
					"auctiontimestamp": 1707996273474.0,
				},
			},
			expectError: false,
		},
		{
			name: "Error Case",
			logObject: &LogObject{
				RequestWrapper: nil,
				Response:       &openrtb2.BidResponse{},
			},
			expectedReqExt: nil,
			expectedResExt: nil,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualReqExt, actualResExt, err := UnmarshalExtensions(tt.logObject)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedReqExt, actualReqExt)
				assert.Equal(t, tt.expectedResExt, actualResExt)
			}
		})
	}
}

func TestProcessBidResponses(t *testing.T) {

	bid := openrtb2.Bid{
		ID:    "bid1",
		ImpID: "imp1",
		CrID:  "creative1",

		Ext: json.RawMessage(`{"prebid": {"targeting": {"hb_pb": "1.00", "hb_bidder": "bidder1", "hb_size": "30x40"}},"origbidcur": "USD","origbidcpm": 1.2}`),
	}

	imp := openrtb2.Imp{
		ID: "imp1",
		Banner: &openrtb2.Banner{
			Format: []openrtb2.Format{
				{W: 10, H: 15},
			},
		},
		Ext: json.RawMessage(`{"prebid": {"bidder": {"bidder1": {"placement_id": 123}}}, "tid": "testtid"}`),
	}

	bidResponse := map[string]interface{}{
		"bidder": "bidder1",
		"bid":    bid,
		"imp":    imp,
	}

	requestExt := map[string]interface{}{}
	responseExt := map[string]interface{}{
		"responsetimemillis": map[string]interface{}{
			"bidder1": 100.0,
		},
	}
	var EmptyFloorData map[string]interface{}
	floorDetail := FloorDetail{
		FloorProvider: "provider1",
		FetchStatus:   "status1",
		Location:      "location1",
		ModelVersion:  "version1",
		SkipRate:      10,
		Skipped:       true,
	}

	tests := []struct {
		name                string
		bidResponses        []map[string]interface{}
		auctionId           string
		startTime           int64
		expectedAuctionBids []Bid
		expectedWinningBids []Bid
	}{
		{
			name:         "Valid bid response",
			bidResponses: []map[string]interface{}{bidResponse},
			auctionId:    "auction1",
			startTime:    time.Now().Unix(),
			expectedAuctionBids: []Bid{
				{
					AdUnitCode:        "",
					BidId:             "bid1",
					GptSlotCode:       "",
					AuctionId:         "auction1",
					BidderCode:        "bidder1",
					Cpm:               1.2,
					CreativeId:        "creative1",
					Currency:          "USD",
					FloorData:         EmptyFloorData,
					NetRevenue:        true,
					RequestTimestamp:  time.Now().Unix(),
					ResponseTimestamp: time.Now().Unix() + 100,
					Status:            "targetingSet",
					StatusMessage:     "Bid available",
					TimeToRespond:     100,
					TransactionId:     "testtid",
					BidType:           2,
					Sizes:             [][]int64{{10, 15}},
				},
			},
			expectedWinningBids: []Bid{
				{
					AdUnitCode:        "",
					BidId:             "bid1",
					GptSlotCode:       "",
					AuctionId:         "auction1",
					BidderCode:        "bidder1",
					Cpm:               1.2,
					CreativeId:        "creative1",
					Currency:          "USD",
					FloorData:         EmptyFloorData,
					NetRevenue:        true,
					RequestTimestamp:  time.Now().Unix(),
					ResponseTimestamp: time.Now().Unix() + 100,
					Status:            "rendered",
					StatusMessage:     "Bid available",
					TimeToRespond:     100,
					TransactionId:     "testtid",
					BidType:           4,
					Sizes: [][]int64{
						{10, 15},
					},
					IsWinningBid:      true,
					PlacementId:       123,
					RenderedSize:      "30x40",
					FloorProvider:     "provider1",
					FloorFetchStatus:  "status1",
					FloorLocation:     "location1",
					FloorModelVersion: "version1",
					FloorSkipRate:     10,
					IsFloorSkipped:    true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualAuctionBids, actualWinningBids := ProcessBidResponses(tt.bidResponses, tt.auctionId, tt.startTime, requestExt, responseExt, floorDetail)

			assert.Equal(t, tt.expectedAuctionBids, actualAuctionBids)
			assert.Equal(t, tt.expectedWinningBids, actualWinningBids)
		})
	}
}

func TestIsWinningBid(t *testing.T) {
	tests := []struct {
		name       string
		bidderName string
		bidExt     map[string]interface{}
		expected   bool
	}{
		{
			name:       "valid bid with matching bidder",
			bidderName: "test_bidder",
			bidExt: map[string]interface{}{
				"prebid": map[string]interface{}{
					"targeting": map[string]interface{}{
						"hb_pb":     "1.50",
						"hb_bidder": "test_bidder",
					},
				},
			},
			expected: true,
		},
		{
			name:       "valid bid with non-matching bidder",
			bidderName: "test_bidder",
			bidExt: map[string]interface{}{
				"prebid": map[string]interface{}{
					"targeting": map[string]interface{}{
						"hb_pb":     "1.50",
						"hb_bidder": "other_bidder",
					},
				},
			},
			expected: false,
		},
		{
			name:       "missing prebid field",
			bidderName: "test_bidder",
			bidExt: map[string]interface{}{
				"other": "value",
			},
			expected: false,
		},
		{
			name:       "missing targeting field",
			bidderName: "test_bidder",
			bidExt: map[string]interface{}{
				"prebid": map[string]interface{}{
					"other": "value",
				},
			},
			expected: false,
		},
		{
			name:       "missing hb_pb field",
			bidderName: "test_bidder",
			bidExt: map[string]interface{}{
				"prebid": map[string]interface{}{
					"targeting": map[string]interface{}{
						"hb_bidder": "test_bidder",
					},
				},
			},
			expected: false,
		},
		{
			name:       "missing hb_bidder field",
			bidderName: "test_bidder",
			bidExt: map[string]interface{}{
				"prebid": map[string]interface{}{
					"targeting": map[string]interface{}{
						"hb_pb": "1.50",
					},
				},
			},
			expected: false,
		},
		{
			name:       "empty hb_pb field",
			bidderName: "test_bidder",
			bidExt: map[string]interface{}{
				"prebid": map[string]interface{}{
					"targeting": map[string]interface{}{
						"hb_pb":     "",
						"hb_bidder": "test_bidder",
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isWinningBid(tt.bidderName, tt.bidExt)
			if result != tt.expected {
				t.Errorf("isWinningBid(%v, %v) = %v; expected %v", tt.bidderName, tt.bidExt, result, tt.expected)
			}
		})
	}
}

func TestExtractFloorData(t *testing.T) {
	tests := []struct {
		name     string
		bidExt   map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "valid bidExt with floors",
			bidExt: map[string]interface{}{
				"prebid": map[string]interface{}{
					"floors": map[string]interface{}{
						"currency":   "USD",
						"floorValue": 1.2,
						"floorRule":  "*|banner",
					},
				},
			},
			expected: map[string]interface{}{
				"currency":   "USD",
				"floorValue": 1.2,
				"floorRule":  "*|banner",
			},
		},
		{
			name: "valid bidExt without floors",
			bidExt: map[string]interface{}{
				"prebid": map[string]interface{}{
					"targeting": map[string]interface{}{
						"hb_pb": "1.50",
					},
				},
			},
			expected: nil,
		},
		{
			name: "invalid bidExt structure",
			bidExt: map[string]interface{}{
				"prebid": map[string]interface{}{
					"other": "value",
				},
			},
			expected: nil,
		},
		{
			name: "empty bidExt",
			bidExt: map[string]interface{}{
				"other": "value",
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFloorData(tt.bidExt)
			if !equal(result, tt.expected) {
				t.Errorf("extractFloorData(%v) = %v; expected %v", tt.bidExt, result, tt.expected)
			}
		})
	}
}

// Helper function to compare two maps
func equal(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}

	for k, v := range a {
		if b[k] != v {
			return false
		}
	}

	return true
}

func TestCreateWinningBidObject(t *testing.T) {
	tests := []struct {
		name        string
		bidObj      Bid
		impExt      map[string]interface{}
		bidExt      map[string]interface{}
		bidderName  string
		floorDetail FloorDetail
		expected    Bid
	}{
		{
			name:   "valid bid with placement_id",
			bidObj: Bid{},
			impExt: map[string]interface{}{
				"prebid": map[string]interface{}{
					"bidder": map[string]interface{}{
						"test_bidder": map[string]interface{}{
							"placement_id": 123.0,
						},
					},
				},
			},
			bidExt: map[string]interface{}{
				"prebid": map[string]interface{}{
					"targeting": map[string]interface{}{
						"hb_size": "300x250",
					},
				},
			},
			bidderName: "test_bidder",
			floorDetail: FloorDetail{
				FloorProvider: "provider",
				FetchStatus:   "fetched",
				Location:      "location",
				ModelVersion:  "v1",
				SkipRate:      1,
				Skipped:       false,
			},
			expected: Bid{
				IsWinningBid:      true,
				BidType:           4,
				Status:            "rendered",
				PlacementId:       123.0,
				RenderedSize:      "300x250",
				FloorProvider:     "provider",
				FloorFetchStatus:  "fetched",
				FloorLocation:     "location",
				FloorModelVersion: "v1",
				FloorSkipRate:     1,
				IsFloorSkipped:    false,
			},
		},
		{
			name:   "missing placement_id",
			bidObj: Bid{},
			impExt: map[string]interface{}{
				"prebid": map[string]interface{}{
					"bidder": map[string]interface{}{
						"test_bidder": map[string]interface{}{
							"other_field": 456.0,
						},
					},
				},
			},
			bidExt: map[string]interface{}{
				"prebid": map[string]interface{}{
					"targeting": map[string]interface{}{
						"hb_size": "728x90",
					},
				},
			},
			bidderName: "test_bidder",
			floorDetail: FloorDetail{
				FloorProvider: "provider",
				FetchStatus:   "fetched",
				Location:      "location",
				ModelVersion:  "v1",
				SkipRate:      2,
				Skipped:       true,
			},
			expected: Bid{
				IsWinningBid:      true,
				BidType:           4,
				Status:            "rendered",
				PlacementId:       0.0,
				RenderedSize:      "728x90",
				FloorProvider:     "provider",
				FloorFetchStatus:  "fetched",
				FloorLocation:     "location",
				FloorModelVersion: "v1",
				FloorSkipRate:     2,
				IsFloorSkipped:    true,
			},
		},
		{
			name:   "invalid placement_id type",
			bidObj: Bid{},
			impExt: map[string]interface{}{
				"prebid": map[string]interface{}{
					"bidder": map[string]interface{}{
						"test_bidder": map[string]interface{}{
							"placement_id": "not_a_number",
						},
					},
				},
			},
			bidExt: map[string]interface{}{
				"prebid": map[string]interface{}{
					"targeting": map[string]interface{}{
						"hb_size": "160x600",
					},
				},
			},
			bidderName: "test_bidder",
			floorDetail: FloorDetail{
				FloorProvider: "provider",
				FetchStatus:   "fetched",
				Location:      "location",
				ModelVersion:  "v2",
				SkipRate:      3,
				Skipped:       false,
			},
			expected: Bid{
				IsWinningBid:      true,
				BidType:           4,
				Status:            "rendered",
				PlacementId:       0.0,
				RenderedSize:      "160x600",
				FloorProvider:     "provider",
				FloorFetchStatus:  "fetched",
				FloorLocation:     "location",
				FloorModelVersion: "v2",
				FloorSkipRate:     3,
				IsFloorSkipped:    false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := createWinningBidObject(tt.bidObj, tt.impExt, tt.bidExt, tt.bidderName, tt.floorDetail)
			if !equalBids(result, tt.expected) {
				t.Errorf("createWinningBidObject(%v, %v, %v, %v, %v) = %v; expected %v", tt.bidObj, tt.impExt, tt.bidExt, tt.bidderName, tt.floorDetail, result, tt.expected)
			}
		})
	}
}

// Helper function to compare two Bid structs
func equalBids(a, b Bid) bool {
	return a.IsWinningBid == b.IsWinningBid &&
		a.BidType == b.BidType &&
		a.Status == b.Status &&
		a.PlacementId == b.PlacementId &&
		a.RenderedSize == b.RenderedSize &&
		a.FloorProvider == b.FloorProvider &&
		a.FloorFetchStatus == b.FloorFetchStatus &&
		a.FloorLocation == b.FloorLocation &&
		a.FloorModelVersion == b.FloorModelVersion &&
		a.FloorSkipRate == b.FloorSkipRate &&
		a.IsFloorSkipped == b.IsFloorSkipped
}
