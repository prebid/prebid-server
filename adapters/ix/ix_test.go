package ix

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
)

const endpoint string = "http://host/endpoint"

func TestJsonSamples(t *testing.T) {
	if bidder, err := Builder(openrtb_ext.BidderIx, config.Adapter{Endpoint: endpoint}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"}); err == nil {
		adapterstest.RunJSONBidderTest(t, "ixtest", bidder)
	} else {
		t.Fatalf("Builder returned unexpected error %v", err)
	}
}

func TestIxMakeBidsWithCategoryDuration(t *testing.T) {
	bidder := &IxAdapter{}

	mockedReq := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{{
			ID: "1_1",
			Video: &openrtb2.Video{
				W:           ptrutil.ToPtr[int64](640),
				H:           ptrutil.ToPtr[int64](360),
				MIMEs:       []string{"video/mp4"},
				MaxDuration: 60,
				Protocols:   []adcom1.MediaCreativeSubtype{2, 3, 5, 6},
			},
			Ext: json.RawMessage(
				`{
					"prebid": {},
					"bidder": {
						"siteID": "123456"
					}
				}`,
			)},
		},
	}
	mockedExtReq := &adapters.RequestData{}
	mockedBidResponse := &openrtb2.BidResponse{
		ID: "test-1",
		SeatBid: []openrtb2.SeatBid{{
			Seat: "Buyer",
			Bid: []openrtb2.Bid{{
				ID:    "1",
				ImpID: "1_1",
				Price: 1.23,
				AdID:  "123",
				Ext: json.RawMessage(
					`{
						"prebid": {
							"video": {
								"duration": 60,
								"primary_category": "IAB18-1"
							}
						}
					}`,
				),
			}},
		}},
	}
	body, _ := json.Marshal(mockedBidResponse)
	mockedRes := &adapters.ResponseData{
		StatusCode: 200,
		Body:       body,
	}

	expectedBidCount := 1
	expectedBidType := openrtb_ext.BidTypeVideo
	expectedBidDuration := 60
	expectedBidCategory := "IAB18-1"
	expectedErrorCount := 0

	bidResponse, errors := bidder.MakeBids(mockedReq, mockedExtReq, mockedRes)

	if len(bidResponse.Bids) != expectedBidCount {
		t.Errorf("should have 1 bid, bids=%v", bidResponse.Bids)
	}
	if bidResponse.Bids[0].BidType != expectedBidType {
		t.Errorf("bid type should be video, bidType=%s", bidResponse.Bids[0].BidType)
	}
	if bidResponse.Bids[0].BidVideo.Duration != expectedBidDuration {
		t.Errorf("video duration should be set")
	}
	if bidResponse.Bids[0].Bid.Cat[0] != expectedBidCategory {
		t.Errorf("bid category should be set")
	}
	if len(errors) != expectedErrorCount {
		t.Errorf("should not have any errors, errors=%v", errors)
	}
}

func TestIxMakeRequestWithGppString(t *testing.T) {
	bidder := &IxAdapter{}

	testGppString := "DBACNYA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA~1YNN"

	mockedReq := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{{
			ID: "1_1",
			Video: &openrtb2.Video{
				W:           ptrutil.ToPtr[int64](640),
				H:           ptrutil.ToPtr[int64](360),
				MIMEs:       []string{"video/mp4"},
				MaxDuration: 60,
				Protocols:   []adcom1.MediaCreativeSubtype{2, 3, 5, 6},
			},
			Ext: json.RawMessage(
				`{
					"prebid": {},
					"bidder": {
						"siteId": "123456"
					}
				}`,
			)},
		},
		Regs: &openrtb2.Regs{
			GPP: testGppString,
		},
	}

	expectedRequestCount := 1
	expectedErrorCount := 0
	var reqInfo *adapters.ExtraRequestInfo

	requests, errors := bidder.MakeRequests(mockedReq, reqInfo)

	if len(requests) != expectedRequestCount {
		t.Errorf("should have 1 request, requests=%v", requests)
	}

	if len(errors) != expectedErrorCount {
		t.Errorf("should not have any errors, errors=%v", errors)
	}

	req := &openrtb2.BidRequest{}
	json.Unmarshal(requests[0].Body, req)

	assert.Equal(t, req.Regs.GPP, testGppString)
}

func TestExtractVersionWithoutCommitHash(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{
			name:     "empty version",
			version:  "",
			expected: "",
		},
		{
			name:     "version with commit hash",
			version:  "1.880-abcdef",
			expected: "1.880",
		},
		{
			name:     "version without commit hash",
			version:  "1.23.4",
			expected: "1.23.4",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, extractVersionWithoutCommitHash(test.version))
		})
	}
}
func TestBuildIxDiag(t *testing.T) {
	testCases := []struct {
		description     string
		request         *openrtb2.BidRequest
		expectedRequest *openrtb2.BidRequest
		expectError     bool
		pbsVersion      string
	}{
		{
			description: "Base Test",
			request: &openrtb2.BidRequest{
				ID:  "1",
				Ext: json.RawMessage(`{"prebid":{"channel":{"name":"web","version":"7.20"}}}`),
			},
			expectedRequest: &openrtb2.BidRequest{
				ID:  "1",
				Ext: json.RawMessage(`{"prebid":{"channel":{"name":"web","version":"7.20"}},"ixdiag":{"pbjsv":"7.20","pbsp":"go","pbsv":"1.880"}}`),
			},
			expectError: false,
			pbsVersion:  "1.880-abcdef",
		},
		{
			description: "Base test for nil channel but non-empty ext prebid payload",
			request: &openrtb2.BidRequest{
				ID:  "1",
				Ext: json.RawMessage(`{"prebid":{"server":{"externalurl":"http://localhost:8000"}}}`),
			},
			expectedRequest: &openrtb2.BidRequest{
				ID:  "1",
				Ext: json.RawMessage(`{"prebid":{"server":{"externalurl":"http://localhost:8000","gvlid":0,"datacenter":""}},"ixdiag":{"pbsp":"go","pbsv":"1.880"}}`),
			},
			expectError: false,
			pbsVersion:  "1.880-abcdef",
		},
		{
			description: "No Ext",
			request: &openrtb2.BidRequest{
				ID: "1",
			},
			expectedRequest: &openrtb2.BidRequest{
				ID:  "1",
				Ext: json.RawMessage(`{"ixdiag":{"pbsp":"go","pbsv":"1.880"}}`),
			},
			expectError: false,
			pbsVersion:  "1.880-abcdef",
		},
		{
			description: "PBS Version Two Hypens",
			request: &openrtb2.BidRequest{
				ID: "1",
			},
			expectedRequest: &openrtb2.BidRequest{
				ID:  "1",
				Ext: json.RawMessage(`{"ixdiag":{"pbsp":"go","pbsv":"0.23.1"}}`),
			},
			expectError: false,
			pbsVersion:  "0.23.1-3-g4ee257d8",
		},
		{
			description: "PBS Version no Hyphen",
			request: &openrtb2.BidRequest{
				ID:  "1",
				Ext: json.RawMessage(`{"prebid":{"channel":{"name":"web","version":"7.20"}}}`),
			},
			expectedRequest: &openrtb2.BidRequest{
				ID:  "1",
				Ext: json.RawMessage(`{"prebid":{"channel":{"name":"web","version":"7.20"}},"ixdiag":{"pbjsv":"7.20","pbsp":"go","pbsv":"1.880"}}`),
			},
			expectError: false,
			pbsVersion:  "1.880",
		},
		{
			description: "PBS Version empty string",
			request: &openrtb2.BidRequest{
				ID:  "1",
				Ext: json.RawMessage(`{"prebid":{"channel":{"name":"web","version":"7.20"}}}`),
			},
			expectedRequest: &openrtb2.BidRequest{
				ID:  "1",
				Ext: json.RawMessage(`{"prebid":{"channel":{"name":"web","version":"7.20"}},"ixdiag":{"pbjsv":"7.20","pbsp":"go","pbsv":"unknown"}}`),
			},
			expectError: false,
			pbsVersion:  "",
		},
		{
			description: "request ixdiag contain other fields that should pass-through along with additional version fields",
			request: &openrtb2.BidRequest{
				ID:  "1",
				Ext: json.RawMessage(`{"prebid":{"channel":{"name":"web","version":"7.20"}},"ixdiag":{"msd":2,"msi":1,"nvin":"123"}}`),
			},
			expectedRequest: &openrtb2.BidRequest{
				ID:  "1",
				Ext: json.RawMessage(`{"prebid":{"channel":{"name":"web","version":"7.20"}},"ixdiag":{"msd":2,"msi":1,"nvin":"123","pbjsv":"7.20","pbsp":"go","pbsv":"unknown"}}`),
			},
			expectError: false,
			pbsVersion:  "",
		},
		{
			description: "Error Test",
			request: &openrtb2.BidRequest{
				ID:  "1",
				Ext: json.RawMessage(`{"prebid":"channel":{"name":"web","version":"7.20"}}}`),
			},
			expectedRequest: &openrtb2.BidRequest{
				ID:  "1",
				Ext: nil,
			},
			expectError: true,
			pbsVersion:  "1.880-abcdef",
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			ixDiagFields := make(map[string]interface{})
			err := setIxDiagIntoExtRequest(test.request, ixDiagFields, test.pbsVersion)
			if test.expectError {
				assert.NotNil(t, err)
			} else {
				assert.Equal(t, test.expectedRequest, test.request)
				assert.Nil(t, err)
			}
		})
	}
}

func TestMakeRequestsErrIxDiag(t *testing.T) {
	bidder := &IxAdapter{}
	req := &openrtb2.BidRequest{
		ID:  "1",
		Ext: json.RawMessage(`{"prebid":"channel":{"name":"web","version":"7.20"}}}`),
	}
	_, errs := bidder.MakeRequests(req, nil)
	assert.Len(t, errs, 1)
}

func TestPABidResponse(t *testing.T) {
	bidder := &IxAdapter{}

	mockedReq := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{{
			ID: "1_1",
			Banner: &openrtb2.Banner{
				Format: []openrtb2.Format{{W: 300, H: 250}},
			},
			Ext: json.RawMessage(
				`{
					"ae": 1,
					"bidder": {
						"siteID": "123456"
					}
				}`,
			)},
		},
	}
	mockedExtReq := &adapters.RequestData{}
	mockedBidResponse := &openrtb2.BidResponse{
		ID: "test-1",
		SeatBid: []openrtb2.SeatBid{{
			Seat: "Buyer",
			Bid: []openrtb2.Bid{{
				ID:    "1",
				ImpID: "1_1",
				Price: 1.23,
				AdID:  "123",
				Ext: json.RawMessage(
					`{
						"prebid": {
							"video": {
								"duration": 60,
								"primary_category": "IAB18-1"
							}
						}
					}`,
				),
			}},
		}},
	}

	testCases := []struct {
		name        string
		ext         json.RawMessage
		expectedLen int
	}{
		{
			name: "properly formatted",
			ext: json.RawMessage(
				`{
					"protectedAudienceAuctionConfigs": [{
						"bidId": "test-imp-id",
						"config": {
							"seller": "https://seller.com",
							"decisionLogicUrl": "https://ssp.com/decision-logic.js",
							"interestGroupBuyers": [
								"https://buyer.com"
							],
							"sellerSignals": {
								"callbackUrl": "https://callbackurl.com"
							},
							"perBuyerSignals": {
								"https://buyer.com": []
							}
						}
					}]
				}`,
			),
			expectedLen: 1,
		},
		{
			name:        "no protected audience auction configs returned",
			ext:         json.RawMessage(`{}`),
			expectedLen: 0,
		},
		{
			name: "no config",
			ext: json.RawMessage(
				`{
					"protectedAudienceAuctionConfigs": [{
						"bidId": "test-imp-id"
					}]
				}`,
			),
			expectedLen: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockedBidResponse.Ext = tc.ext
			body, _ := json.Marshal(mockedBidResponse)
			mockedRes := &adapters.ResponseData{
				StatusCode: 200,
				Body:       body,
			}
			bidResponse, errors := bidder.MakeBids(mockedReq, mockedExtReq, mockedRes)

			assert.Nil(t, errors)
			assert.Len(t, bidResponse.FledgeAuctionConfigs, tc.expectedLen)
		})
	}
}
