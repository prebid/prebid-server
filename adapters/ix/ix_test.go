package ix

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/version"
	"github.com/stretchr/testify/assert"

	"github.com/prebid/openrtb/v19/adcom1"
	"github.com/prebid/openrtb/v19/openrtb2"
)

const endpoint string = "http://host/endpoint"

func TestJsonSamples(t *testing.T) {
	if bidder, err := Builder(openrtb_ext.BidderIx, config.Adapter{Endpoint: endpoint}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"}); err == nil {
		ixBidder := bidder.(*IxAdapter)
		ixBidder.maxRequests = 2
		ixBidder.featuresToRequest = nil
		adapterstest.RunJSONBidderTest(t, "ixtest", bidder)
	} else {
		t.Fatalf("Builder returned unexpected error %v", err)
	}
}

func TestJsonForMultiImpAndSize(t *testing.T) {
	if bidder, err := Builder(openrtb_ext.BidderIx, config.Adapter{Endpoint: endpoint}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"}); err == nil {
		ixBidder := bidder.(*IxAdapter)
		ixBidder.maxRequests = 2
		ixBidder.clientFeatureStatusMap = map[string]FeatureTimestamp{
			"pbs_handle_multi_imp_on_single_req": {
				Activated: true,
				Timestamp: time.Now().Unix(),
			},
		}
		adapterstest.RunJSONBidderTest(t, "ixtestmulti", bidder)
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
				W:           640,
				H:           360,
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
	bidder.maxRequests = 2

	testGppString := "DBACNYA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA~1YNN"

	mockedReq := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{{
			ID: "1_1",
			Video: &openrtb2.Video{
				W:           640,
				H:           360,
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
				Ext: json.RawMessage(`{"prebid":{"channel":{"name":"web","version":"7.20"}},"ixdiag":{"pbsv":"1.880","pbjsv":"7.20"}}`),
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
				Ext: json.RawMessage(`{"prebid":{"server":{"externalurl":"http://localhost:8000","gvlid":0,"datacenter":""}},"ixdiag":{"pbsv":"1.880"}}`),
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
				Ext: json.RawMessage(`{"prebid":null,"ixdiag":{"pbsv":"1.880"}}`),
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
				Ext: json.RawMessage(`{"prebid":null,"ixdiag":{"pbsv":"0.23.1"}}`),
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
				Ext: json.RawMessage(`{"prebid":{"channel":{"name":"web","version":"7.20"}},"ixdiag":{"pbsv":"1.880","pbjsv":"7.20"}}`),
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
				Ext: json.RawMessage(`{"prebid":{"channel":{"name":"web","version":"7.20"}},"ixdiag":{"pbjsv":"7.20"}}`),
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
			version.Ver = test.pbsVersion
			err := BuildIxDiag(test.request)
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

func TestSetFeatureToggles(t *testing.T) {
	testCases := []struct {
		description string
		ext         json.RawMessage
		expected    map[string]FeatureTimestamp
	}{
		{
			description: "nil ext",
			ext:         nil,
			expected:    map[string]FeatureTimestamp{},
		},
		{
			description: "Empty input",
			ext:         json.RawMessage(``),
			expected:    map[string]FeatureTimestamp{},
		},
		{
			description: "valid input with one feature toggle",
			ext:         json.RawMessage(`{"features":{"ft_test_1":{"activated":true}}}`),
			expected: map[string]FeatureTimestamp{
				"ft_test_1": {
					Activated: true,
					Timestamp: time.Now().Unix(),
				},
			},
		},
		{
			description: "valid input with two feature toggles",
			ext:         json.RawMessage(`{"features":{"ft_test_1":{"activated":true},"ft_test_2":{"activated":false}}}`),
			expected: map[string]FeatureTimestamp{
				"ft_test_1": {
					Activated: true,
					Timestamp: time.Now().Unix(),
				},
				"ft_test_2": {
					Activated: false,
					Timestamp: time.Now().Unix(),
				},
			},
		},
		{
			description: "input with one feature toggle, no activated key",
			ext:         json.RawMessage(`{"features":{"ft_test_1":{"exists":true}}}`),
			expected:    map[string]FeatureTimestamp{},
		},
		{
			description: "features not formatted correctly",
			ext:         json.RawMessage(`{"features":"helloworld"}`),
			expected:    map[string]FeatureTimestamp{},
		},
		{
			description: "no features",
			ext:         json.RawMessage(`{"prebid":{"test":"helloworld"}}`),
			expected:    map[string]FeatureTimestamp{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			bidder, _ := Builder(openrtb_ext.BidderIx, config.Adapter{Endpoint: endpoint}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
			ixBidder := bidder.(*IxAdapter)
			setFeatureToggles(ixBidder, &tc.ext)
			assert.Equal(t, tc.expected, ixBidder.clientFeatureStatusMap)
		})
	}
}

func TestGetFeatureToggle(t *testing.T) {
	clientFeatureMap := map[string]FeatureTimestamp{
		"feature1": {
			Activated: true,
			Timestamp: time.Now().Unix(),
		},
		"feature2": {
			Activated: true,
			Timestamp: time.Now().Unix() - 3700,
		},
		"feature3": {
			Activated: false,
			Timestamp: time.Now().Unix(),
		},
	}

	tests := []struct {
		description string
		ftName      string
		expected    bool
	}{
		{"ActivatedFeature", "feature1", true},
		{"ExpiredFeature", "feature2", false},
		{"NotExpiredFeature", "feature3", false},
		{"NonExistentFeature", "nonexistent", false},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			bidder, _ := Builder(openrtb_ext.BidderIx, config.Adapter{Endpoint: endpoint}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
			ixBidder := bidder.(*IxAdapter)
			ixBidder.clientFeatureStatusMap = clientFeatureMap
			result := isFeatureToggleActive(ixBidder, test.ftName)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestRequestFeatureToggles(t *testing.T) {
	type testCase struct {
		name              string
		inputRequest      *openrtb2.BidRequest
		featuresToRequest []string
		expectedExt       json.RawMessage
		initialFeatureMap *FeatureTimestamp
	}

	testCases := []testCase{
		{
			name:              "empty features",
			inputRequest:      &openrtb2.BidRequest{ID: "1"},
			featuresToRequest: []string{},
			expectedExt:       json.RawMessage(nil),
		},
		{
			name:              "no features existing internally, request feature expect false",
			inputRequest:      &openrtb2.BidRequest{ID: "1"},
			featuresToRequest: []string{"ft1"},
			expectedExt:       json.RawMessage(`{"prebid":null,"features":{"ft1":{"activated":false}}}`),
		},
		{
			name:              "feature exists internally and activated",
			inputRequest:      &openrtb2.BidRequest{ID: "1"},
			featuresToRequest: []string{"ft1"},
			expectedExt:       json.RawMessage(`{"prebid":null,"features":{"ft1":{"activated":true}}}`),
			initialFeatureMap: &FeatureTimestamp{Timestamp: time.Now().Unix(), Activated: true},
		},
		{
			name:              "feature exists internally and not activated",
			inputRequest:      &openrtb2.BidRequest{ID: "1"},
			featuresToRequest: []string{"ft1"},
			expectedExt:       json.RawMessage(`{"prebid":null,"features":{"ft1":{"activated":false}}}`),
			initialFeatureMap: &FeatureTimestamp{Timestamp: time.Now().Unix(), Activated: false},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bidder, _ := Builder(openrtb_ext.BidderIx, config.Adapter{Endpoint: endpoint}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
			ixBidder := bidder.(*IxAdapter)
			if tc.initialFeatureMap != nil {
				ixBidder.clientFeatureStatusMap["ft1"] = *tc.initialFeatureMap
			}
			ixBidder.featuresToRequest = tc.featuresToRequest
			requestFeatureToggles(ixBidder, tc.inputRequest)
			assert.Equal(t, tc.expectedExt, tc.inputRequest.Ext)
		})
	}
}

func TestMoveSid(t *testing.T) {
	testCases := []struct {
		description string
		imp         openrtb2.Imp
		expectedExt json.RawMessage
		expectErr   bool
	}{
		{
			description: "valid input with sid",
			imp: openrtb2.Imp{
				Ext: json.RawMessage(`{"bidder":{"sid":"1234"}}`),
			},
			expectedExt: json.RawMessage(`{"bidder":{"sid":"1234"},"sid":"1234"}`),
			expectErr:   false,
		},
		{
			description: "valid input without sid",
			imp: openrtb2.Imp{
				Ext: json.RawMessage(`{"bidder":{"siteId":"1234"}}`),
			},
			expectedExt: json.RawMessage(`{"bidder":{"siteId":"1234"}}`),
			expectErr:   false,
		},
		{
			description: "no ext",
			imp:         openrtb2.Imp{ID: "1"},
			expectedExt: nil,
			expectErr:   false,
		},
		{
			description: "malformed json",
			imp: openrtb2.Imp{
				Ext: json.RawMessage(`"bidder":{"sid":"1234"}`),
			},
			expectedExt: json.RawMessage(`"bidder":{"sid":"1234"}`),
			expectErr:   true,
		},
		{
			description: "malformed bidder json",
			imp: openrtb2.Imp{
				Ext: json.RawMessage(`{"bidder":{"sid":1234}}`),
			},
			expectedExt: json.RawMessage(`{"bidder":{"sid":1234}}`),
			expectErr:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			err := moveSid(&tc.imp)
			assert.Equal(t, tc.expectedExt, tc.imp.Ext)
			if tc.expectErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
