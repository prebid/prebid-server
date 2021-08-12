package rubicon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/buger/jsonparser"
	"github.com/stretchr/testify/mock"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/errortypes"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/cache/dummycache"
	"github.com/prebid/prebid-server/pbs"
	"github.com/prebid/prebid-server/usersync"

	"fmt"

	"strings"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/stretchr/testify/assert"
)

type rubiAppendTrackerUrlTestScenario struct {
	source   string
	tracker  string
	expected string
}

type rubiSetNetworkIdTestScenario struct {
	bidExt            *openrtb_ext.ExtBidPrebid
	buyer             string
	expectedNetworkId int64
	isNetworkIdSet    bool
}

type rubiTagInfo struct {
	code              string
	zoneID            int
	bid               float64
	content           string
	adServerTargeting map[string]string
	mediaType         string
}

type rubiBidInfo struct {
	domain             string
	page               string
	accountID          int
	siteID             int
	tags               []rubiTagInfo
	deviceIP           string
	deviceUA           string
	buyerUID           string
	xapiuser           string
	xapipass           string
	delay              time.Duration
	visitorTargeting   string
	inventoryTargeting string
	sdkVersion         string
	sdkPlatform        string
	sdkSource          string
	devicePxRatio      float64
}

var rubidata rubiBidInfo

func DummyRubiconServer(w http.ResponseWriter, r *http.Request) {
	defer func() {
		err := r.Body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var breq openrtb2.BidRequest
	err = json.Unmarshal(body, &breq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(breq.Imp) > 1 {
		http.Error(w, "Rubicon adapter only supports one Imp per request", http.StatusInternalServerError)
		return
	}
	imp := breq.Imp[0]
	var rix rubiconImpExt
	err = json.Unmarshal(imp.Ext, &rix)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	impTargetingString, _ := json.Marshal(&rix.RP.Target)
	if string(impTargetingString) != rubidata.inventoryTargeting {
		http.Error(w, fmt.Sprintf("Inventory FPD targeting '%s' doesn't match '%s'", string(impTargetingString), rubidata.inventoryTargeting), http.StatusInternalServerError)
		return
	}
	if rix.RP.Track.Mint != "prebid" {
		http.Error(w, fmt.Sprintf("Track mint '%s' doesn't match '%s'", rix.RP.Track.Mint, "prebid"), http.StatusInternalServerError)
		return
	}
	mintVersionString := rubidata.sdkSource + "_" + rubidata.sdkPlatform + "_" + rubidata.sdkVersion
	if rix.RP.Track.MintVersion != mintVersionString {
		http.Error(w, fmt.Sprintf("Track mint version '%s' doesn't match '%s'", rix.RP.Track.MintVersion, mintVersionString), http.StatusInternalServerError)
		return
	}

	ix := -1

	for i, tag := range rubidata.tags {
		if rix.RP.ZoneID == tag.zoneID {
			ix = i
		}
	}
	if ix == -1 {
		http.Error(w, fmt.Sprintf("Zone %d not found", rix.RP.ZoneID), http.StatusInternalServerError)
		return
	}

	resp := openrtb2.BidResponse{
		ID:    "test-response-id",
		BidID: "test-bid-id",
		Cur:   "USD",
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "RUBICON",
				Bid:  make([]openrtb2.Bid, 2),
			},
		},
	}

	if imp.Banner != nil {
		var bix rubiconBannerExt
		err = json.Unmarshal(imp.Banner.Ext, &bix)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if bix.RP.SizeID != 15 { // 300x250
			http.Error(w, fmt.Sprintf("Primary size ID isn't 15"), http.StatusInternalServerError)
			return
		}
		if len(bix.RP.AltSizeIDs) != 1 || bix.RP.AltSizeIDs[0] != 10 { // 300x600
			http.Error(w, fmt.Sprintf("Alt size ID isn't 10"), http.StatusInternalServerError)
			return
		}
		if bix.RP.MIME != "text/html" {
			http.Error(w, fmt.Sprintf("MIME isn't text/html"), http.StatusInternalServerError)
			return
		}
	}

	if imp.Video != nil {
		var vix rubiconVideoExt
		err = json.Unmarshal(imp.Video.Ext, &vix)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if len(imp.Video.MIMEs) == 0 {
			http.Error(w, fmt.Sprintf("Empty imp.video.mimes array"), http.StatusInternalServerError)
			return
		}
		if len(imp.Video.Protocols) == 0 {
			http.Error(w, fmt.Sprintf("Empty imp.video.protocols array"), http.StatusInternalServerError)
			return
		}
		for _, protocol := range imp.Video.Protocols {
			if protocol < 1 || protocol > 8 {
				http.Error(w, fmt.Sprintf("Invalid video protocol %d", protocol), http.StatusInternalServerError)
				return
			}
		}
	}

	targeting := "{\"rp\":{\"targeting\":[{\"key\":\"key1\",\"values\":[\"value1\"]},{\"key\":\"key2\",\"values\":[\"value2\"]}]}}"
	rawTargeting := json.RawMessage(targeting)

	resp.SeatBid[0].Bid[0] = openrtb2.Bid{
		ID:    "random-id",
		ImpID: imp.ID,
		Price: rubidata.tags[ix].bid,
		AdM:   rubidata.tags[ix].content,
		Ext:   rawTargeting,
	}

	if breq.Site == nil {
		http.Error(w, fmt.Sprintf("No site object sent"), http.StatusInternalServerError)
		return
	}
	if breq.Site.Domain != rubidata.domain {
		http.Error(w, fmt.Sprintf("Domain '%s' doesn't match '%s", breq.Site.Domain, rubidata.domain), http.StatusInternalServerError)
		return
	}
	if breq.Site.Page != rubidata.page {
		http.Error(w, fmt.Sprintf("Page '%s' doesn't match '%s", breq.Site.Page, rubidata.page), http.StatusInternalServerError)
		return
	}
	var rsx rubiconSiteExt
	err = json.Unmarshal(breq.Site.Ext, &rsx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if rsx.RP.SiteID != rubidata.siteID {
		http.Error(w, fmt.Sprintf("SiteID '%d' doesn't match '%d", rsx.RP.SiteID, rubidata.siteID), http.StatusInternalServerError)
		return
	}
	if breq.Site.Publisher == nil {
		http.Error(w, fmt.Sprintf("No site.publisher object sent"), http.StatusInternalServerError)
		return
	}
	var rpx rubiconPubExt
	err = json.Unmarshal(breq.Site.Publisher.Ext, &rpx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if rpx.RP.AccountID != rubidata.accountID {
		http.Error(w, fmt.Sprintf("AccountID '%d' doesn't match '%d'", rpx.RP.AccountID, rubidata.accountID), http.StatusInternalServerError)
		return
	}
	if breq.Device.UA != rubidata.deviceUA {
		http.Error(w, fmt.Sprintf("UA '%s' doesn't match '%s'", breq.Device.UA, rubidata.deviceUA), http.StatusInternalServerError)
		return
	}
	if breq.Device.IP != rubidata.deviceIP {
		http.Error(w, fmt.Sprintf("IP '%s' doesn't match '%s'", breq.Device.IP, rubidata.deviceIP), http.StatusInternalServerError)
		return
	}
	if breq.Device.PxRatio != rubidata.devicePxRatio {
		http.Error(w, fmt.Sprintf("Pixel ratio '%f' doesn't match '%f'", breq.Device.PxRatio, rubidata.devicePxRatio), http.StatusInternalServerError)
		return
	}
	if breq.User.BuyerUID != rubidata.buyerUID {
		http.Error(w, fmt.Sprintf("User ID '%s' doesn't match '%s'", breq.User.BuyerUID, rubidata.buyerUID), http.StatusInternalServerError)
		return
	}

	var rux rubiconUserExt
	err = json.Unmarshal(breq.User.Ext, &rux)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	userTargetingString, _ := json.Marshal(&rux.RP.Target)
	if string(userTargetingString) != rubidata.visitorTargeting {
		http.Error(w, fmt.Sprintf("User FPD targeting '%s' doesn't match '%s'", string(userTargetingString), rubidata.visitorTargeting), http.StatusInternalServerError)
		return
	}

	if rubidata.delay > 0 {
		<-time.After(rubidata.delay)
	}

	js, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func TestRubiconBasicResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(DummyRubiconServer))
	defer server.Close()

	an, ctx, pbReq := CreatePrebidRequest(server, t)

	bids, err := an.Call(ctx, pbReq, pbReq.Bidders[0])
	assert.Nil(t, err, "Should not have gotten an error: %v", err)
	assert.Equal(t, 2, len(bids), "Received %d bids instead of 3", len(bids))

	for _, bid := range bids {
		matched := false
		for _, tag := range rubidata.tags {
			if bid.AdUnitCode == tag.code {
				matched = true

				assert.Equal(t, "rubicon", bid.BidderCode, "Incorrect BidderCode '%s'", bid.BidderCode)

				assert.Equal(t, tag.bid, bid.Price, "Incorrect bid price '%.2f' expected '%.2f'", bid.Price, tag.bid)

				assert.Equal(t, tag.content, bid.Adm, "Incorrect bid markup '%s' expected '%s'", bid.Adm, tag.content)

				assert.Equal(t, bid.AdServerTargeting, tag.adServerTargeting,
					"Incorrect targeting '%+v' expected '%+v'", bid.AdServerTargeting, tag.adServerTargeting)

				assert.Equal(t, tag.mediaType, bid.CreativeMediaType, "Incorrect media type '%s' expected '%s'", bid.CreativeMediaType, tag.mediaType)
			}
		}
		assert.True(t, matched, "Received bid for unknown ad unit '%s'", bid.AdUnitCode)
	}

	// same test but with request timing out
	rubidata.delay = 20 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	bids, err = an.Call(ctx, pbReq, pbReq.Bidders[0])
	assert.NotNil(t, err, "Should have gotten a timeout error: %v", err)
}

func TestRubiconUserSyncInfo(t *testing.T) {
	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewRubiconLegacyAdapter(&conf, "uri", "xuser", "xpass", "pbs-test-tracker")

	assert.Equal(t, "rubicon", an.Name(), "Name '%s' != 'rubicon'", an.Name())

	assert.False(t, an.SkipNoCookies(), "SkipNoCookies should be false")
}

func getTestSizes() map[int]openrtb2.Format {
	return map[int]openrtb2.Format{
		15: {W: 300, H: 250},
		10: {W: 300, H: 600},
		2:  {W: 728, H: 91},
		9:  {W: 160, H: 600},
		8:  {W: 120, H: 600},
		33: {W: 180, H: 500},
		43: {W: 320, H: 50},
	}
}

func TestParseSizes(t *testing.T) {
	SIZE_ID := getTestSizes()

	sizes := []openrtb2.Format{
		SIZE_ID[10],
		SIZE_ID[15],
	}
	primary, alt, err := parseRubiconSizes(sizes)
	assert.Nil(t, err, "Parsing error: %v", err)
	assert.Equal(t, 15, primary, "Primary %d != 15", primary)
	assert.Equal(t, 1, len(alt), "Alt not len 1")
	assert.Equal(t, 10, alt[0], "Alt not 10: %d", alt[0])

	sizes = []openrtb2.Format{
		{
			W: 1111,
			H: 2222,
		},
		SIZE_ID[15],
	}
	primary, alt, err = parseRubiconSizes(sizes)
	assert.Nil(t, err, "Shouldn't have thrown error for invalid size 1111x1111 since we still have a valid one")
	assert.Equal(t, 15, primary, "Primary %d != 15", primary)
	assert.Equal(t, 0, len(alt), "Alt len %d != 0", len(alt))

	sizes = []openrtb2.Format{
		SIZE_ID[15],
	}
	primary, alt, err = parseRubiconSizes(sizes)
	assert.Nil(t, err, "Parsing error: %v", err)
	assert.Equal(t, 15, primary, "Primary %d != 15", primary)
	assert.Equal(t, 0, len(alt), "Alt len %d != 0", len(alt))

	sizes = []openrtb2.Format{
		{
			W: 1111,
			H: 1222,
		},
	}
	primary, alt, err = parseRubiconSizes(sizes)
	assert.NotNil(t, err, "Parsing error: %v", err)
	assert.Equal(t, 0, primary, "Primary %d != 15", primary)
	assert.Equal(t, 0, len(alt), "Alt len %d != 0", len(alt))
}

func TestMASAlgorithm(t *testing.T) {
	SIZE_ID := getTestSizes()
	type output struct {
		primary int
		alt     []int
		ok      bool
	}
	type testStub struct {
		input  []openrtb2.Format
		output output
	}

	testStubs := []testStub{
		{
			[]openrtb2.Format{
				SIZE_ID[2],
				SIZE_ID[9],
			},
			output{2, []int{9}, false},
		},
		{
			[]openrtb2.Format{

				SIZE_ID[9],
				SIZE_ID[15],
			},
			output{15, []int{9}, false},
		},
		{
			[]openrtb2.Format{
				SIZE_ID[2],
				SIZE_ID[15],
			},
			output{15, []int{2}, false},
		},
		{
			[]openrtb2.Format{
				SIZE_ID[15],
				SIZE_ID[9],
				SIZE_ID[2],
			},
			output{15, []int{2, 9}, false},
		},
		{
			[]openrtb2.Format{
				SIZE_ID[10],
				SIZE_ID[9],
				SIZE_ID[2],
			},
			output{2, []int{10, 9}, false},
		},
		{
			[]openrtb2.Format{
				SIZE_ID[33],
				SIZE_ID[8],
				SIZE_ID[15],
			},
			output{15, []int{33, 8}, false},
		},
		{
			[]openrtb2.Format{
				SIZE_ID[33],
				SIZE_ID[8],
				SIZE_ID[9],
				SIZE_ID[2],
			},
			output{2, []int{33, 8, 9}, false},
		},
		{
			[]openrtb2.Format{
				SIZE_ID[33],
				SIZE_ID[8],
				SIZE_ID[9],
			},
			output{9, []int{33, 8}, false},
		},
		{
			[]openrtb2.Format{
				SIZE_ID[33],
				SIZE_ID[8],
				SIZE_ID[2],
			},
			output{2, []int{33, 8}, false},
		},
		{
			[]openrtb2.Format{
				SIZE_ID[33],
				SIZE_ID[2],
			},
			output{2, []int{33}, false},
		},
		{
			[]openrtb2.Format{
				SIZE_ID[8],
			},
			output{8, []int{}, false},
		},
		{
			[]openrtb2.Format{},
			output{0, []int{}, true},
		},
		{
			[]openrtb2.Format{
				{W: 1111,
					H: 2345,
				},
			},
			output{0, []int{}, true},
		},
	}

	for _, test := range testStubs {
		prim, alt, err := parseRubiconSizes(test.input)

		assert.Equal(t, test.output.primary, prim,
			"Error in parsing rubicon sizes: MAS algorithm fail at primary: testcase %v", test.input)

		assert.Equal(t, len(test.output.alt), len(alt),
			"Error in parsing rubicon sizes: MAS Algorithm fail at alt: testcase %v", test.input)

		assert.False(t, err != nil && !test.output.ok,
			"Error in parsing rubicon sizes: MAS Algorithm fail at throwing error: testcase %v", test.input)

		assert.False(t, err == nil && test.output.ok,
			"Error in parsing rubicon sizes: MAS Algorithm fail at throwing error: testcase %v", test.input)
	}
}

func TestAppendTracker(t *testing.T) {
	testScenarios := []rubiAppendTrackerUrlTestScenario{
		{
			source:   "http://test.url/",
			tracker:  "prebid",
			expected: "http://test.url/?tk_xint=prebid",
		},
		{
			source:   "http://test.url/?hello=true",
			tracker:  "prebid",
			expected: "http://test.url/?hello=true&tk_xint=prebid",
		},
	}

	for _, scenario := range testScenarios {
		res := appendTrackerToUrl(scenario.source, scenario.tracker)
		assert.Equal(t, scenario.expected, res, "Failed to convert '%s' to '%s'", res, scenario.expected)
	}
}

func TestResolveVideoSizeId(t *testing.T) {
	testScenarios := []struct {
		placement   openrtb2.VideoPlacementType
		instl       int8
		impId       string
		expected    int
		expectedErr error
	}{
		{
			placement:   1,
			instl:       1,
			impId:       "impId",
			expected:    201,
			expectedErr: nil,
		},
		{
			placement:   3,
			instl:       1,
			impId:       "impId",
			expected:    203,
			expectedErr: nil,
		},
		{
			placement:   4,
			instl:       1,
			impId:       "impId",
			expected:    202,
			expectedErr: nil,
		},
		{
			placement: 4,
			instl:     3,
			impId:     "impId",
			expectedErr: &errortypes.BadInput{
				Message: "video.size_id can not be resolved in impression with id : impId",
			},
		},
	}

	for _, scenario := range testScenarios {
		res, err := resolveVideoSizeId(scenario.placement, scenario.instl, scenario.impId)
		assert.Equal(t, scenario.expected, res)
		assert.Equal(t, scenario.expectedErr, err)
	}
}

func TestOpenRTBRequestWithDifferentBidFloorAttributes(t *testing.T) {
	testScenarios := []struct {
		bidFloor         float64
		bidFloorCur      string
		setMock          func(m *mock.Mock)
		expectedBidFloor float64
		expectedBidCur   string
		expectedErrors   []error
	}{
		{
			bidFloor:         1,
			bidFloorCur:      "WRONG",
			setMock:          func(m *mock.Mock) { m.On("GetRate", "WRONG", "USD").Return(2.5, errors.New("some error")) },
			expectedBidFloor: 0,
			expectedBidCur:   "",
			expectedErrors: []error{
				&errortypes.BadInput{Message: "Unable to convert provided bid floor currency from WRONG to USD"},
			},
		},
		{
			bidFloor:         1,
			bidFloorCur:      "USD",
			setMock:          func(m *mock.Mock) {},
			expectedBidFloor: 1,
			expectedBidCur:   "USD",
			expectedErrors:   nil,
		},
		{
			bidFloor:         1,
			bidFloorCur:      "EUR",
			setMock:          func(m *mock.Mock) { m.On("GetRate", "EUR", "USD").Return(1.2, nil) },
			expectedBidFloor: 1.2,
			expectedBidCur:   "USD",
			expectedErrors:   nil,
		},
		{
			bidFloor:         0,
			bidFloorCur:      "",
			setMock:          func(m *mock.Mock) {},
			expectedBidFloor: 0,
			expectedBidCur:   "",
			expectedErrors:   nil,
		},
		{
			bidFloor:         -1,
			bidFloorCur:      "CZK",
			setMock:          func(m *mock.Mock) {},
			expectedBidFloor: -1,
			expectedBidCur:   "CZK",
			expectedErrors:   nil,
		},
	}

	for _, scenario := range testScenarios {
		mockConversions := &mockCurrencyConversion{}
		scenario.setMock(&mockConversions.Mock)

		extraRequestInfo := adapters.ExtraRequestInfo{
			CurrencyConversions: mockConversions,
		}

		SIZE_ID := getTestSizes()
		bidder := new(RubiconAdapter)

		request := &openrtb2.BidRequest{
			ID: "test-request-id",
			Imp: []openrtb2.Imp{{
				ID:          "test-imp-id",
				BidFloorCur: scenario.bidFloorCur,
				BidFloor:    scenario.bidFloor,
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{
						SIZE_ID[15],
						SIZE_ID[10],
					},
				},
				Ext: json.RawMessage(`{"bidder": {
										"zoneId": 8394,
										"siteId": 283282,
										"accountId": 7891
                                      }}`),
			}},
			App: &openrtb2.App{
				ID:   "com.test",
				Name: "testApp",
			},
		}

		reqs, errs := bidder.MakeRequests(request, &extraRequestInfo)

		mockConversions.AssertExpectations(t)

		if scenario.expectedErrors == nil {
			rubiconReq := &openrtb2.BidRequest{}
			if err := json.Unmarshal(reqs[0].Body, rubiconReq); err != nil {
				t.Fatalf("Unexpected error while decoding request: %s", err)
			}
			assert.Equal(t, scenario.expectedBidFloor, rubiconReq.Imp[0].BidFloor)
			assert.Equal(t, scenario.expectedBidCur, rubiconReq.Imp[0].BidFloorCur)
		} else {
			assert.Equal(t, scenario.expectedErrors, errs)
		}
	}
}

type mockCurrencyConversion struct {
	mock.Mock
}

func (m mockCurrencyConversion) GetRate(from string, to string) (float64, error) {
	args := m.Called(from, to)
	return args.Get(0).(float64), args.Error(1)
}

func (m mockCurrencyConversion) GetRates() *map[string]map[string]float64 {
	args := m.Called()
	return args.Get(0).(*map[string]map[string]float64)
}

func TestNoContentResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	an, ctx, pbReq := CreatePrebidRequest(server, t)
	bids, err := an.Call(ctx, pbReq, pbReq.Bidders[0])

	assert.Empty(t, bids, "Length of bids should be 0 instead of: %v", len(bids))

	assert.Equal(t, 204, pbReq.Bidders[0].Debug[0].StatusCode,
		"StatusCode should be 204 instead of: %v", pbReq.Bidders[0].Debug[0].StatusCode)

	assert.Nil(t, err, "Should not have gotten an error: %v", err)
}

func TestNotFoundResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	an, ctx, pbReq := CreatePrebidRequest(server, t)
	_, err := an.Call(ctx, pbReq, pbReq.Bidders[0])

	assert.Equal(t, 404, pbReq.Bidders[0].Debug[0].StatusCode,
		"StatusCode should be 404 instead of: %v", pbReq.Bidders[0].Debug[0].StatusCode)

	assert.NotNil(t, err, "Should have gotten an error: %v", err)

	assert.True(t, strings.HasPrefix(err.Error(), "HTTP status 404"),
		"Should start with 'HTTP status' instead of: %v", err.Error())
}

func TestWrongFormatResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("This is text."))
	}))
	defer server.Close()

	an, ctx, pbReq := CreatePrebidRequest(server, t)
	_, err := an.Call(ctx, pbReq, pbReq.Bidders[0])

	assert.Equal(t, 200, pbReq.Bidders[0].Debug[0].StatusCode,
		"StatusCode should be 200 instead of: %v", pbReq.Bidders[0].Debug[0].StatusCode)

	assert.NotNil(t, err, "Should have gotten an error: %v", err)

	assert.True(t, strings.HasPrefix(err.Error(), "invalid character"),
		"Should start with 'invalid character' instead of: %v", err)
}

func TestZeroSeatBidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openrtb2.BidResponse{
			ID:      "test-response-id",
			BidID:   "test-bid-id",
			Cur:     "USD",
			SeatBid: []openrtb2.SeatBid{},
		}
		js, _ := json.Marshal(resp)
		w.Write(js)
	}))
	defer server.Close()

	an, ctx, pbReq := CreatePrebidRequest(server, t)
	bids, err := an.Call(ctx, pbReq, pbReq.Bidders[0])

	assert.Empty(t, bids, "Length of bids should be 0 instead of: %v", len(bids))

	assert.Nil(t, err, "Should not have gotten an error: %v", err)
}

func TestEmptyBidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openrtb2.BidResponse{
			ID:    "test-response-id",
			BidID: "test-bid-id",
			Cur:   "USD",
			SeatBid: []openrtb2.SeatBid{
				{
					Seat: "RUBICON",
					Bid:  make([]openrtb2.Bid, 0),
				},
			},
		}
		js, _ := json.Marshal(resp)
		w.Write(js)
	}))
	defer server.Close()

	an, ctx, pbReq := CreatePrebidRequest(server, t)
	bids, err := an.Call(ctx, pbReq, pbReq.Bidders[0])

	assert.Empty(t, bids, "Length of bids should be 0 instead of: %v", len(bids))

	assert.Nil(t, err, "Should not have gotten an error: %v", err)
}

func TestWrongBidIdResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openrtb2.BidResponse{
			ID:    "test-response-id",
			BidID: "test-bid-id",
			Cur:   "USD",
			SeatBid: []openrtb2.SeatBid{
				{
					Seat: "RUBICON",
					Bid:  make([]openrtb2.Bid, 2),
				},
			},
		}
		resp.SeatBid[0].Bid[0] = openrtb2.Bid{
			ID:    "random-id",
			ImpID: "zma",
			Price: 1.67,
			AdM:   "zma",
			Ext:   json.RawMessage("{\"rp\":{\"targeting\":[{\"key\":\"key1\",\"values\":[\"value1\"]},{\"key\":\"key2\",\"values\":[\"value2\"]}]}}"),
		}
		js, _ := json.Marshal(resp)
		w.Write(js)
	}))
	defer server.Close()

	an, ctx, pbReq := CreatePrebidRequest(server, t)
	bids, err := an.Call(ctx, pbReq, pbReq.Bidders[0])

	assert.Empty(t, bids, "Length of bids should be 0 instead of: %v", len(bids))

	assert.NotNil(t, err, "Should not have gotten an error: %v", err)

	assert.True(t, strings.HasPrefix(err.Error(), "Unknown ad unit code"),
		"Should start with 'Unknown ad unit code' instead of: %v", err)
}

func TestZeroPriceBidResponse(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openrtb2.BidResponse{
			ID:    "test-response-id",
			BidID: "test-bid-id",
			Cur:   "USD",
			SeatBid: []openrtb2.SeatBid{
				{
					Seat: "RUBICON",
					Bid:  make([]openrtb2.Bid, 1),
				},
			},
		}
		resp.SeatBid[0].Bid[0] = openrtb2.Bid{
			ID:    "test-bid-id",
			ImpID: "first-tag",
			Price: 0,
			AdM:   "zma",
			Ext:   json.RawMessage("{\"rp\":{\"targeting\":[{\"key\":\"key1\",\"values\":[\"value1\"]},{\"key\":\"key2\",\"values\":[\"value2\"]}]}}"),
		}
		js, _ := json.Marshal(resp)
		w.Write(js)
	}))
	defer server.Close()

	an, ctx, pbReq := CreatePrebidRequest(server, t)
	b, err := an.Call(ctx, pbReq, pbReq.Bidders[0])

	assert.Nil(t, b, "\n\n\n0 price bids are being included %d, err : %v", len(b), err)
}

func TestDifferentRequest(t *testing.T) {
	SIZE_ID := getTestSizes()
	server := httptest.NewServer(http.HandlerFunc(DummyRubiconServer))
	defer server.Close()

	an, ctx, pbReq := CreatePrebidRequest(server, t)

	// test app not nil
	pbReq.App = &openrtb2.App{
		ID:   "com.test",
		Name: "testApp",
	}

	_, err := an.Call(ctx, pbReq, pbReq.Bidders[0])
	assert.NotNil(t, err, "Should have gotten an error: %v", err)

	// set app back to normal
	pbReq.App = nil

	// test video media type
	pbReq.Bidders[0].AdUnits[0].MediaTypes = []pbs.MediaType{pbs.MEDIA_TYPE_VIDEO}
	_, err = an.Call(ctx, pbReq, pbReq.Bidders[0])
	assert.NotNil(t, err, "Should have gotten an error: %v", err)

	// set media back to normal
	pbReq.Bidders[0].AdUnits[0].MediaTypes = []pbs.MediaType{pbs.MEDIA_TYPE_BANNER}

	// test wrong params
	pbReq.Bidders[0].AdUnits[0].Params = json.RawMessage(fmt.Sprintf("{\"zoneId\": %s, \"siteId\": %d, \"visitor\": %s, \"inventory\": %s}", "zma", rubidata.siteID, rubidata.visitorTargeting, rubidata.inventoryTargeting))
	_, err = an.Call(ctx, pbReq, pbReq.Bidders[0])
	assert.NotNil(t, err, "Should have gotten an error: %v", err)

	// set params back to normal
	pbReq.Bidders[0].AdUnits[0].Params = json.RawMessage(fmt.Sprintf("{\"zoneId\": %d, \"siteId\": %d, \"accountId\": %d, \"visitor\": %s, \"inventory\": %s}", 8394, rubidata.siteID, rubidata.accountID, rubidata.visitorTargeting, rubidata.inventoryTargeting))

	// test invalid size
	pbReq.Bidders[0].AdUnits[0].Sizes = []openrtb2.Format{
		{
			W: 2222,
			H: 333,
		},
	}
	pbReq.Bidders[0].AdUnits[1].Sizes = []openrtb2.Format{
		{
			W: 222,
			H: 3333,
		},
		{
			W: 350,
			H: 270,
		},
	}
	pbReq.Bidders[0].AdUnits = pbReq.Bidders[0].AdUnits[:len(pbReq.Bidders[0].AdUnits)-1]
	b, err := an.Call(ctx, pbReq, pbReq.Bidders[0])
	assert.NotNil(t, err, "Should have gotten an error: %v", err)

	pbReq.Bidders[0].AdUnits[1].Sizes = []openrtb2.Format{
		{
			W: 222,
			H: 3333,
		},
		SIZE_ID[10],
		SIZE_ID[15],
	}
	b, err = an.Call(ctx, pbReq, pbReq.Bidders[0])
	assert.Nil(t, err, "Should have not gotten an error: %v", err)

	assert.Equal(t, 1, len(b),
		"Filtering bids based on ad unit sizes failed. Got %d bids instead of 1, error = %v", len(b), err)
}

func CreatePrebidRequest(server *httptest.Server, t *testing.T) (an *RubiconAdapter, ctx context.Context, pbReq *pbs.PBSRequest) {
	SIZE_ID := getTestSizes()
	rubidata = rubiBidInfo{
		domain:             "nytimes.com",
		page:               "https://www.nytimes.com/2017/05/04/movies/guardians-of-the-galaxy-2-review-chris-pratt.html?hpw&rref=movies&action=click&pgtype=Homepage&module=well-region&region=bottom-well&WT.nav=bottom-well&_r=0",
		accountID:          7891,
		siteID:             283282,
		tags:               make([]rubiTagInfo, 3),
		deviceIP:           "25.91.96.36",
		deviceUA:           "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_4) AppleWebKit/603.1.30 (KHTML, like Gecko) Version/10.1 Safari/603.1.30",
		buyerUID:           "need-an-actual-rp-id",
		visitorTargeting:   "[\"v1\",\"v2\"]",
		inventoryTargeting: "[\"i1\",\"i2\"]",
		sdkVersion:         "2.0.0",
		sdkPlatform:        "iOS",
		sdkSource:          "some-sdk",
		devicePxRatio:      4.0,
	}

	targeting := make(map[string]string, 2)
	targeting["key1"] = "value1"
	targeting["key2"] = "value2"

	rubidata.tags[0] = rubiTagInfo{
		code:              "first-tag",
		zoneID:            8394,
		bid:               1.67,
		adServerTargeting: targeting,
		mediaType:         "banner",
	}
	rubidata.tags[1] = rubiTagInfo{
		code:              "second-tag",
		zoneID:            8395,
		bid:               3.22,
		adServerTargeting: targeting,
		mediaType:         "banner",
	}
	rubidata.tags[2] = rubiTagInfo{
		code:              "video-tag",
		zoneID:            7780,
		bid:               23.12,
		adServerTargeting: targeting,
		mediaType:         "video",
	}

	conf := *adapters.DefaultHTTPAdapterConfig
	an = NewRubiconLegacyAdapter(&conf, "uri", rubidata.xapiuser, rubidata.xapipass, "pbs-test-tracker")
	an.URI = server.URL

	pbin := pbs.PBSRequest{
		AdUnits: make([]pbs.AdUnit, 3),
		Device:  &openrtb2.Device{PxRatio: rubidata.devicePxRatio},
		SDK:     &pbs.SDK{Source: rubidata.sdkSource, Platform: rubidata.sdkPlatform, Version: rubidata.sdkVersion},
	}

	for i, tag := range rubidata.tags {
		pbin.AdUnits[i] = pbs.AdUnit{
			Code:       tag.code,
			MediaTypes: []string{tag.mediaType},
			Sizes: []openrtb2.Format{
				SIZE_ID[10],
				SIZE_ID[15],
			},
			Bids: []pbs.Bids{
				{
					BidderCode: "rubicon",
					BidID:      fmt.Sprintf("random-id-from-pbjs-%d", i),
					Params:     json.RawMessage(fmt.Sprintf("{\"zoneId\": %d, \"siteId\": %d, \"accountId\": %d, \"visitor\": %s, \"inventory\": %s}", tag.zoneID, rubidata.siteID, rubidata.accountID, rubidata.visitorTargeting, rubidata.inventoryTargeting)),
				},
			},
		}
		if tag.mediaType == "video" {
			pbin.AdUnits[i].Video = pbs.PBSVideo{
				Mimes:          []string{"video/mp4"},
				Minduration:    15,
				Maxduration:    30,
				Startdelay:     5,
				Skippable:      0,
				PlaybackMethod: 1,
				Protocols:      []int8{1, 2, 3, 4, 5},
			}
		}
	}

	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(pbin)
	if err != nil {
		t.Fatalf("Json encoding failed: %v", err)
	}

	req := httptest.NewRequest("POST", server.URL, body)
	req.Header.Add("Referer", rubidata.page)
	req.Header.Add("User-Agent", rubidata.deviceUA)
	req.Header.Add("X-Real-IP", rubidata.deviceIP)

	pc := usersync.ParsePBSCookieFromRequest(req, &config.HostCookie{})
	pc.TrySync("rubicon", rubidata.buyerUID)
	fakewriter := httptest.NewRecorder()

	pc.SetCookieOnResponse(fakewriter, false, &config.HostCookie{Domain: ""}, 90*24*time.Hour)
	req.Header.Add("Cookie", fakewriter.Header().Get("Set-Cookie"))

	cacheClient, _ := dummycache.New()
	hcc := config.HostCookie{}

	pbReq, err = pbs.ParsePBSRequest(req, &config.AuctionTimeouts{
		Default: 2000,
		Max:     2000,
	}, cacheClient, &hcc)
	pbReq.IsDebug = true

	assert.Nil(t, err, "ParsePBSRequest failed: %v", err)

	assert.Equal(t, 1, len(pbReq.Bidders),
		"ParsePBSRequest returned %d bidders instead of 1", len(pbReq.Bidders))

	assert.Equal(t, "rubicon", pbReq.Bidders[0].BidderCode,
		"ParsePBSRequest returned invalid bidder")

	ctx = context.TODO()
	return
}

func TestOpenRTBRequest(t *testing.T) {
	SIZE_ID := getTestSizes()
	bidder := new(RubiconAdapter)

	rubidata = rubiBidInfo{
		domain:        "nytimes.com",
		page:          "https://www.nytimes.com/2017/05/04/movies/guardians-of-the-galaxy-2-review-chris-pratt.html?hpw&rref=movies&action=click&pgtype=Homepage&module=well-region&region=bottom-well&WT.nav=bottom-well&_r=0",
		deviceIP:      "25.91.96.36",
		deviceUA:      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_4) AppleWebKit/603.1.30 (KHTML, like Gecko) Version/10.1 Safari/603.1.30",
		buyerUID:      "need-an-actual-rp-id",
		devicePxRatio: 4.0,
	}

	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{{
			ID: "test-imp-banner-id",
			Banner: &openrtb2.Banner{
				Format: []openrtb2.Format{
					SIZE_ID[15],
					SIZE_ID[10],
				},
			},
			Ext: json.RawMessage(`{"bidder": {
				"zoneId": 8394,
				"siteId": 283282,
				"accountId": 7891,
				"inventory": {"key1" : "val1"},
				"visitor": {"key2" : "val2"}
			}}`),
		}, {
			ID: "test-imp-video-id",
			Video: &openrtb2.Video{
				W:           640,
				H:           360,
				MIMEs:       []string{"video/mp4"},
				MinDuration: 15,
				MaxDuration: 30,
			},
			Ext: json.RawMessage(`{"bidder": {
				"zoneId": 7780,
				"siteId": 283282,
				"accountId": 7891,
				"inventory": {"key1" : "val1"},
				"visitor": {"key2" : "val2"},
				"video": {
					"language": "en",
					"playerHeight": 360,
					"playerWidth": 640,
					"size_id": 203,
					"skip": 1,
					"skipdelay": 5
				}
			}}`),
		}},
		App: &openrtb2.App{
			ID:   "com.test",
			Name: "testApp",
		},
		Device: &openrtb2.Device{
			PxRatio: rubidata.devicePxRatio,
		},
		User: &openrtb2.User{
			Ext: json.RawMessage(`{
				"eids": [{
                    "source": "pubcid",
                    "id": "2402fc76-7b39-4f0e-bfc2-060ef7693648"
				}]
            }`),
		},
		Ext: json.RawMessage(`{"prebid": {}}`),
	}

	reqs, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})

	assert.Empty(t, errs, "Got unexpected errors while building HTTP requests: %v", errs)
	assert.Equal(t, 2, len(reqs), "Unexpected number of HTTP requests. Got %d. Expected %d", len(reqs), 2)

	for i := 0; i < len(reqs); i++ {
		httpReq := reqs[i]
		assert.Equal(t, "POST", httpReq.Method, "Expected a POST message. Got %s", httpReq.Method)

		var rpRequest openrtb2.BidRequest
		if err := json.Unmarshal(httpReq.Body, &rpRequest); err != nil {
			t.Fatalf("Failed to unmarshal HTTP request: %v", rpRequest)
		}

		assert.Equal(t, request.ID, rpRequest.ID, "Bad Request ID. Expected %s, Got %s", request.ID, rpRequest.ID)
		assert.Equal(t, 1, len(rpRequest.Imp), "Wrong len(request.Imp). Expected %d, Got %d", len(request.Imp), len(rpRequest.Imp))
		assert.Nil(t, rpRequest.Cur, "Wrong request.Cur. Expected nil, Got %s", rpRequest.Cur)
		assert.Nil(t, rpRequest.Ext, "Wrong request.ext. Expected nil, Got %v", rpRequest.Ext)

		if rpRequest.Imp[0].ID == "test-imp-banner-id" {
			var rpExt rubiconBannerExt
			if err := json.Unmarshal(rpRequest.Imp[0].Ext, &rpExt); err != nil {
				t.Fatal("Error unmarshalling request from the outgoing request.")
			}

			assert.Equal(t, int64(300), rpRequest.Imp[0].Banner.Format[0].W,
				"Banner width does not match. Expected %d, Got %d", 300, rpRequest.Imp[0].Banner.Format[0].W)

			assert.Equal(t, int64(250), rpRequest.Imp[0].Banner.Format[0].H,
				"Banner height does not match. Expected %d, Got %d", 250, rpRequest.Imp[0].Banner.Format[0].H)

			assert.Equal(t, int64(300), rpRequest.Imp[0].Banner.Format[1].W,
				"Banner width does not match. Expected %d, Got %d", 300, rpRequest.Imp[0].Banner.Format[1].W)

			assert.Equal(t, int64(600), rpRequest.Imp[0].Banner.Format[1].H,
				"Banner height does not match. Expected %d, Got %d", 600, rpRequest.Imp[0].Banner.Format[1].H)
		} else if rpRequest.Imp[0].ID == "test-imp-video-id" {
			var rpExt rubiconVideoExt
			if err := json.Unmarshal(rpRequest.Imp[0].Ext, &rpExt); err != nil {
				t.Fatal("Error unmarshalling request from the outgoing request.")
			}

			assert.Equal(t, int64(640), rpRequest.Imp[0].Video.W,
				"Video width does not match. Expected %d, Got %d", 640, rpRequest.Imp[0].Video.W)

			assert.Equal(t, int64(360), rpRequest.Imp[0].Video.H,
				"Video height does not match. Expected %d, Got %d", 360, rpRequest.Imp[0].Video.H)

			assert.Equal(t, "video/mp4", rpRequest.Imp[0].Video.MIMEs[0], "Video MIMEs do not match. Expected %s, Got %s", "video/mp4", rpRequest.Imp[0].Video.MIMEs[0])

			assert.Equal(t, int64(15), rpRequest.Imp[0].Video.MinDuration,
				"Video min duration does not match. Expected %d, Got %d", 15, rpRequest.Imp[0].Video.MinDuration)

			assert.Equal(t, int64(30), rpRequest.Imp[0].Video.MaxDuration,
				"Video max duration does not match. Expected %d, Got %d", 30, rpRequest.Imp[0].Video.MaxDuration)
		}

		assert.NotNil(t, rpRequest.User.Ext, "User.Ext object should not be nil.")

		var userExt rubiconUserExt
		if err := json.Unmarshal(rpRequest.User.Ext, &userExt); err != nil {
			t.Fatal("Error unmarshalling request.user.ext object.")
		}

		assert.NotNil(t, userExt.Eids)
		assert.Equal(t, 1, len(userExt.Eids), "Eids values are not as expected!")
		assert.Contains(t, userExt.Eids, openrtb_ext.ExtUserEid{Source: "pubcid", ID: "2402fc76-7b39-4f0e-bfc2-060ef7693648"})
	}
}

func TestOpenRTBRequestWithBannerImpEvenIfImpHasVideo(t *testing.T) {
	SIZE_ID := getTestSizes()
	bidder := new(RubiconAdapter)

	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{{
			ID: "test-imp-id",
			Banner: &openrtb2.Banner{
				Format: []openrtb2.Format{
					SIZE_ID[15],
					SIZE_ID[10],
				},
			},
			Video: &openrtb2.Video{
				W:     640,
				H:     360,
				MIMEs: []string{"video/mp4"},
			},
			Ext: json.RawMessage(`{"bidder": {
				"zoneId": 8394,
				"siteId": 283282,
				"accountId": 7891,
				"inventory": {"key1" : "val1"},
				"visitor": {"key2" : "val2"}
			}}`),
		}},
		App: &openrtb2.App{
			ID:   "com.test",
			Name: "testApp",
		},
	}

	reqs, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})

	assert.Empty(t, errs, "Got unexpected errors while building HTTP requests: %v", errs)

	assert.Equal(t, 1, len(reqs), "Unexpected number of HTTP requests. Got %d. Expected %d", len(reqs), 1)

	rubiconReq := &openrtb2.BidRequest{}
	if err := json.Unmarshal(reqs[0].Body, rubiconReq); err != nil {
		t.Fatalf("Unexpected error while decoding request: %s", err)
	}

	assert.Equal(t, 1, len(rubiconReq.Imp), "Unexpected number of request impressions. Got %d. Expected %d", len(rubiconReq.Imp), 1)

	assert.Nil(t, rubiconReq.Imp[0].Video, "Unexpected video object in request impression")

	assert.NotNil(t, rubiconReq.Imp[0].Banner, "Banner object must be in request impression")
}

func TestOpenRTBRequestWithImpAndAdSlotIncluded(t *testing.T) {
	SIZE_ID := getTestSizes()
	bidder := new(RubiconAdapter)

	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{{
			ID: "test-imp-id",
			Banner: &openrtb2.Banner{
				Format: []openrtb2.Format{
					SIZE_ID[15],
					SIZE_ID[10],
				},
			},
			Ext: json.RawMessage(`{
				"bidder": {
					"zoneId": 8394,
					"siteId": 283282,
					"accountId": 7891,
					"inventory": {"key1" : "val1"},
					"visitor": {"key2" : "val2"}
				},
				"context": {
					"data": {
						"adslot": "///test-adslot"
					}
				}
			}`),
		}},
		App: &openrtb2.App{
			ID:   "com.test",
			Name: "testApp",
		},
	}

	reqs, _ := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})

	rubiconReq := &openrtb2.BidRequest{}
	if err := json.Unmarshal(reqs[0].Body, rubiconReq); err != nil {
		t.Fatalf("Unexpected error while decoding request: %s", err)
	}

	assert.Equal(t, 1, len(rubiconReq.Imp),
		"Unexpected number of request impressions. Got %d. Expected %d", len(rubiconReq.Imp), 1)

	var rpImpExt rubiconImpExt
	if err := json.Unmarshal(rubiconReq.Imp[0].Ext, &rpImpExt); err != nil {
		t.Fatal("Error unmarshalling imp.ext")
	}

	rubiconExtInventory := make(map[string]interface{})
	if err := json.Unmarshal(rpImpExt.RP.Target, &rubiconExtInventory); err != nil {
		t.Fatal("Error unmarshalling imp.ext.rp.target")
	}

	assert.Equal(t, "test-adslot", rubiconExtInventory["dfp_ad_unit_code"],
		"Unexpected dfp_ad_unit_code: %s", rubiconExtInventory["dfp_ad_unit_code"])
}

func TestOpenRTBRequestWithBadvOverflowed(t *testing.T) {
	SIZE_ID := getTestSizes()
	bidder := new(RubiconAdapter)

	badvOverflowed := make([]string, 100)
	for i := range badvOverflowed {
		badvOverflowed[i] = strconv.Itoa(i)
	}

	request := &openrtb2.BidRequest{
		ID:   "test-request-id",
		BAdv: badvOverflowed,
		Imp: []openrtb2.Imp{{
			ID: "test-imp-id",
			Banner: &openrtb2.Banner{
				Format: []openrtb2.Format{
					SIZE_ID[15],
				},
			},
			Ext: json.RawMessage(`{
				"bidder": {
					"zoneId": 8394,
					"siteId": 283282,
					"accountId": 7891,
					"inventory": {"key1" : "val1"},
					"visitor": {"key2" : "val2"}
				}
			}`),
		}},
		App: &openrtb2.App{
			ID:   "com.test",
			Name: "testApp",
		},
	}

	reqs, _ := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})

	rubiconReq := &openrtb2.BidRequest{}
	if err := json.Unmarshal(reqs[0].Body, rubiconReq); err != nil {
		t.Fatalf("Unexpected error while decoding request: %s", err)
	}

	badvRequest := rubiconReq.BAdv
	assert.Equal(t, badvOverflowed[:50], badvRequest, "Unexpected dfp_ad_unit_code: %s")
}

func TestOpenRTBRequestWithSpecificExtUserEids(t *testing.T) {
	SIZE_ID := getTestSizes()
	bidder := new(RubiconAdapter)

	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{{
			ID: "test-imp-id",
			Banner: &openrtb2.Banner{
				Format: []openrtb2.Format{
					SIZE_ID[15],
					SIZE_ID[10],
				},
			},
			Ext: json.RawMessage(`{"bidder": {
				"zoneId": 8394,
				"siteId": 283282,
				"accountId": 7891
			}}`),
		}},
		App: &openrtb2.App{
			ID:   "com.test",
			Name: "testApp",
		},
		User: &openrtb2.User{
			Ext: json.RawMessage(`{"eids": [
			{
				"source": "pubcid",
				"id": "2402fc76-7b39-4f0e-bfc2-060ef7693648"
			},
			{
				"source": "adserver.org",
				"uids": [{
					"id": "3d50a262-bd8e-4be3-90b8-246291523907",
					"ext": {
						"rtiPartner": "TDID"
					}
				}]
			},
			{
				"source": "liveintent.com",
				"uids": [{
					"id": "T7JiRRvsRAmh88"
				}],
				"ext": {
					"segments": ["999","888"]
				}
			},
			{
				"source": "liveramp.com",
				"uids": [{
					"id": "LIVERAMPID"
				}],
				"ext": {
					"segments": ["111","222"]
				}
			}
			]}`),
		},
	}

	reqs, _ := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})

	rubiconReq := &openrtb2.BidRequest{}
	if err := json.Unmarshal(reqs[0].Body, rubiconReq); err != nil {
		t.Fatalf("Unexpected error while decoding request: %s", err)
	}

	assert.NotNil(t, rubiconReq.User.Ext, "User.Ext object should not be nil.")

	var userExt rubiconUserExt
	if err := json.Unmarshal(rubiconReq.User.Ext, &userExt); err != nil {
		t.Fatal("Error unmarshalling request.user.ext object.")
	}

	assert.NotNil(t, userExt.Eids)
	assert.Equal(t, 4, len(userExt.Eids), "Eids values are not as expected!")

	assert.NotNil(t, userExt.TpID)
	assert.Equal(t, 2, len(userExt.TpID), "TpID values are not as expected!")

	// adserver.org
	assert.Equal(t, "tdid", userExt.TpID[0].Source, "TpID source value is not as expected!")

	// liveintent.com
	assert.Equal(t, "liveintent.com", userExt.TpID[1].Source, "TpID source value is not as expected!")

	// liveramp.com
	assert.Equal(t, "LIVERAMPID", userExt.LiverampIdl, "Liveramp_idl value is not as expected!")

	userExtRPTarget := make(map[string]interface{})
	if err := json.Unmarshal(userExt.RP.Target, &userExtRPTarget); err != nil {
		t.Fatal("Error unmarshalling request.user.ext.rp.target object.")
	}

	assert.Contains(t, userExtRPTarget, "LIseg", "request.user.ext.rp.target value is not as expected!")
	assert.Contains(t, userExtRPTarget["LIseg"], "888", "No segment with 888 as expected!")
	assert.Contains(t, userExtRPTarget["LIseg"], "999", "No segment with 999 as expected!")
}

func TestOpenRTBRequestWithVideoImpEvenIfImpHasBannerButAllRequiredVideoFields(t *testing.T) {
	SIZE_ID := getTestSizes()
	bidder := new(RubiconAdapter)

	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{{
			ID: "test-imp-id",
			Banner: &openrtb2.Banner{
				Format: []openrtb2.Format{
					SIZE_ID[15],
					SIZE_ID[10],
				},
			},
			Video: &openrtb2.Video{
				W:           640,
				H:           360,
				MIMEs:       []string{"video/mp4"},
				Protocols:   []openrtb2.Protocol{openrtb2.ProtocolVAST10},
				MaxDuration: 30,
				Linearity:   1,
				API:         []openrtb2.APIFramework{},
			},
			Ext: json.RawMessage(`{"bidder": {
				"zoneId": 8394,
				"siteId": 283282,
				"accountId": 7891,
				"inventory": {"key1": "val1"},
				"visitor": {"key2": "val2"},
				"video": {"size_id": 1}
			}}`),
		}},
		App: &openrtb2.App{
			ID:   "com.test",
			Name: "testApp",
		},
	}

	reqs, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})

	assert.Empty(t, errs, "Got unexpected errors while building HTTP requests: %v", errs)

	assert.Equal(t, 1, len(reqs), "Unexpected number of HTTP requests. Got %d. Expected %d", len(reqs), 1)

	rubiconReq := &openrtb2.BidRequest{}
	if err := json.Unmarshal(reqs[0].Body, rubiconReq); err != nil {
		t.Fatalf("Unexpected error while decoding request: %s", err)
	}

	assert.Equal(t, 1, len(rubiconReq.Imp),
		"Unexpected number of request impressions. Got %d. Expected %d", len(rubiconReq.Imp), 1)

	assert.Nil(t, rubiconReq.Imp[0].Banner, "Unexpected banner object in request impression")

	assert.NotNil(t, rubiconReq.Imp[0].Video, "Video object must be in request impression")
}

func TestOpenRTBRequestWithVideoImpAndEnabledRewardedInventoryFlag(t *testing.T) {
	bidder := new(RubiconAdapter)

	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{{
			ID: "test-imp-id",
			Video: &openrtb2.Video{
				W:           640,
				H:           360,
				MIMEs:       []string{"video/mp4"},
				Protocols:   []openrtb2.Protocol{openrtb2.ProtocolVAST10},
				MaxDuration: 30,
				Linearity:   1,
				API:         []openrtb2.APIFramework{},
			},
			Ext: json.RawMessage(`{
			"prebid":{
				"is_rewarded_inventory": 1
			},
			"bidder": {
				"video": {"size_id": 1}
			}}`),
		}},
		App: &openrtb2.App{
			ID:   "com.test",
			Name: "testApp",
		},
	}

	reqs, _ := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})

	rubiconReq := &openrtb2.BidRequest{}
	if err := json.Unmarshal(reqs[0].Body, rubiconReq); err != nil {
		t.Fatalf("Unexpected error while decoding request: %s", err)
	}

	videoExt := &rubiconVideoExt{}
	if err := json.Unmarshal(rubiconReq.Imp[0].Video.Ext, &videoExt); err != nil {
		t.Fatal("Error unmarshalling request.imp[i].video.ext object.")
	}

	assert.Equal(t, "rewarded", videoExt.VideoType,
		"Unexpected VideoType. Got %s. Expected %s", videoExt.VideoType, "rewarded")
}

func TestOpenRTBEmptyResponse(t *testing.T) {
	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusNoContent,
	}
	bidder := new(RubiconAdapter)
	bidResponse, errs := bidder.MakeBids(nil, nil, httpResp)

	assert.Nil(t, bidResponse, "Expected empty response")
	assert.Empty(t, errs, "Expected 0 errors. Got %d", len(errs))
}

func TestOpenRTBSurpriseResponse(t *testing.T) {
	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusAccepted,
	}
	bidder := new(RubiconAdapter)
	bidResponse, errs := bidder.MakeBids(nil, nil, httpResp)

	assert.Nil(t, bidResponse, "Expected empty response")

	assert.Equal(t, 1, len(errs), "Expected 1 error. Got %d", len(errs))
}

func TestOpenRTBStandardResponse(t *testing.T) {
	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{{
			ID: "test-imp-id",
			Banner: &openrtb2.Banner{
				Format: []openrtb2.Format{{
					W: 320,
					H: 50,
				}},
			},
			Ext: json.RawMessage(`{"bidder": {
				"accountId": 2763,
				"siteId": 68780,
				"zoneId": 327642
			}}`),
		}},
	}

	requestJson, _ := json.Marshal(request)
	reqData := &adapters.RequestData{
		Method:  "POST",
		Uri:     "test-uri",
		Body:    requestJson,
		Headers: nil,
	}

	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte(`{"id":"test-request-id","seatbid":[{"bid":[{"id":"1234567890","impid":"test-imp-id","price": 2,"crid":"4122982","adm":"some ad","h": 50,"w": 320,"ext":{"bidder":{"rp":{"targeting": {"key": "rpfl_2763", "values":["43_tier0100"]},"mime": "text/html","size_id": 43}}}}]}]}`),
	}

	bidder := new(RubiconAdapter)
	bidResponse, errs := bidder.MakeBids(request, reqData, httpResp)

	assert.NotNil(t, bidResponse, "Expected not empty response")
	assert.Equal(t, 1, len(bidResponse.Bids), "Expected 1 bid. Got %d", len(bidResponse.Bids))

	assert.Empty(t, errs, "Expected 0 errors. Got %d", len(errs))

	assert.Equal(t, openrtb_ext.BidTypeBanner, bidResponse.Bids[0].BidType,
		"Expected a banner bid. Got: %s", bidResponse.Bids[0].BidType)

	theBid := bidResponse.Bids[0].Bid
	assert.Equal(t, "1234567890", theBid.ID, "Bad bid ID. Expected %s, got %s", "1234567890", theBid.ID)
}

func TestOpenRTBResponseOverridePriceFromBidRequest(t *testing.T) {
	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{{
			ID: "test-imp-id",
			Banner: &openrtb2.Banner{
				Format: []openrtb2.Format{{
					W: 320,
					H: 50,
				}},
			},
			Ext: json.RawMessage(`{"bidder": {
				"accountId": 2763,
				"siteId": 68780,
				"zoneId": 327642
			}}`),
		}},
		Ext: json.RawMessage(`{"prebid": {
			"bidders": {
				"rubicon": {
					"debug": {
						"cpmoverride": 10
			}}}}}`),
	}

	requestJson, _ := json.Marshal(request)
	reqData := &adapters.RequestData{
		Method:  "POST",
		Uri:     "test-uri",
		Body:    requestJson,
		Headers: nil,
	}

	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte(`{"id":"test-request-id","seatbid":[{"bid":[{"id":"1234567890","impid":"test-imp-id","price": 2,"crid":"4122982","adm":"some ad","h": 50,"w": 320,"ext":{"bidder":{"rp":{"targeting": {"key": "rpfl_2763", "values":["43_tier0100"]},"mime": "text/html","size_id": 43}}}}]}]}`),
	}

	bidder := new(RubiconAdapter)
	bidResponse, errs := bidder.MakeBids(request, reqData, httpResp)

	assert.Empty(t, errs, "Expected 0 errors. Got %d", len(errs))

	assert.Equal(t, float64(10), bidResponse.Bids[0].Bid.Price,
		"Expected Price 10. Got: %s", bidResponse.Bids[0].Bid.Price)
}

func TestOpenRTBResponseSettingOfNetworkId(t *testing.T) {
	testScenarios := []rubiSetNetworkIdTestScenario{
		{
			bidExt:            nil,
			buyer:             "1",
			expectedNetworkId: 1,
			isNetworkIdSet:    true,
		},
		{
			bidExt:            nil,
			buyer:             "0",
			expectedNetworkId: 0,
			isNetworkIdSet:    false,
		},
		{
			bidExt:            nil,
			buyer:             "-1",
			expectedNetworkId: 0,
			isNetworkIdSet:    false,
		},
		{
			bidExt:            nil,
			buyer:             "1.1",
			expectedNetworkId: 0,
			isNetworkIdSet:    false,
		},
		{
			bidExt:            &openrtb_ext.ExtBidPrebid{},
			buyer:             "2",
			expectedNetworkId: 2,
			isNetworkIdSet:    true,
		},
		{
			bidExt:            &openrtb_ext.ExtBidPrebid{Meta: &openrtb_ext.ExtBidPrebidMeta{}},
			buyer:             "3",
			expectedNetworkId: 3,
			isNetworkIdSet:    true,
		},
		{
			bidExt:            &openrtb_ext.ExtBidPrebid{Meta: &openrtb_ext.ExtBidPrebidMeta{NetworkID: 5}},
			buyer:             "4",
			expectedNetworkId: 4,
			isNetworkIdSet:    true,
		},
		{
			bidExt:            &openrtb_ext.ExtBidPrebid{Meta: &openrtb_ext.ExtBidPrebidMeta{NetworkID: 5}},
			buyer:             "-1",
			expectedNetworkId: 5,
			isNetworkIdSet:    false,
		},
	}

	for _, scenario := range testScenarios {
		request := &openrtb2.BidRequest{
			Imp: []openrtb2.Imp{{
				ID:     "test-imp-id",
				Banner: &openrtb2.Banner{},
			}},
		}

		requestJson, _ := json.Marshal(request)
		reqData := &adapters.RequestData{
			Method:  "POST",
			Uri:     "test-uri",
			Body:    requestJson,
			Headers: nil,
		}

		var givenBidExt json.RawMessage
		if scenario.bidExt != nil {
			marshalledExt, _ := json.Marshal(scenario.bidExt)
			givenBidExt = marshalledExt
		} else {
			givenBidExt = nil
		}
		givenBidResponse := rubiconBidResponse{
			SeatBid: []rubiconSeatBid{{Buyer: scenario.buyer,
				SeatBid: openrtb2.SeatBid{
					Bid: []openrtb2.Bid{{Price: 123.2, ImpID: "test-imp-id", Ext: givenBidExt}}}}},
		}
		body, _ := json.Marshal(&givenBidResponse)
		httpResp := &adapters.ResponseData{
			StatusCode: http.StatusOK,
			Body:       body,
		}

		bidder := new(RubiconAdapter)
		bidResponse, errs := bidder.MakeBids(request, reqData, httpResp)
		assert.Empty(t, errs)
		if scenario.isNetworkIdSet {
			networkdId, err := jsonparser.GetInt(bidResponse.Bids[0].Bid.Ext, "prebid", "meta", "networkId")
			assert.NoError(t, err)
			assert.Equal(t, scenario.expectedNetworkId, networkdId)
		} else {
			assert.Equal(t, bidResponse.Bids[0].Bid.Ext, givenBidExt)
		}
	}
}

func TestOpenRTBResponseOverridePriceFromCorrespondingImp(t *testing.T) {
	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{{
			ID: "test-imp-id",
			Banner: &openrtb2.Banner{
				Format: []openrtb2.Format{{
					W: 320,
					H: 50,
				}},
			},
			Ext: json.RawMessage(`{"bidder": {
				"accountId": 2763,
				"siteId": 68780,
				"zoneId": 327642,
				"debug": {
					"cpmoverride" : 20 
				}
			}}`),
		}},
		Ext: json.RawMessage(`{"prebid": {
			"bidders": {
				"rubicon": {
					"debug": {
						"cpmoverride": 10
			}}}}}`),
	}

	requestJson, _ := json.Marshal(request)
	reqData := &adapters.RequestData{
		Method:  "POST",
		Uri:     "test-uri",
		Body:    requestJson,
		Headers: nil,
	}

	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte(`{"id":"test-request-id","seatbid":[{"bid":[{"id":"1234567890","impid":"test-imp-id","price": 2,"crid":"4122982","adm":"some ad","h": 50,"w": 320,"ext":{"bidder":{"rp":{"targeting": {"key": "rpfl_2763", "values":["43_tier0100"]},"mime": "text/html","size_id": 43}}}}]}]}`),
	}

	bidder := new(RubiconAdapter)
	bidResponse, errs := bidder.MakeBids(request, reqData, httpResp)

	assert.Empty(t, errs, "Expected 0 errors. Got %d", len(errs))

	assert.Equal(t, float64(20), bidResponse.Bids[0].Bid.Price,
		"Expected Price 20. Got: %s", bidResponse.Bids[0].Bid.Price)
}

func TestOpenRTBCopyBidIdFromResponseIfZero(t *testing.T) {
	request := &openrtb2.BidRequest{
		ID:  "test-request-id",
		Imp: []openrtb2.Imp{{}},
	}

	requestJson, _ := json.Marshal(request)
	reqData := &adapters.RequestData{Body: requestJson}

	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte(`{"id":"test-request-id","bidid":"1234567890","seatbid":[{"bid":[{"id":"0","price": 1}]}]}`),
	}

	bidder := new(RubiconAdapter)
	bidResponse, _ := bidder.MakeBids(request, reqData, httpResp)

	theBid := bidResponse.Bids[0].Bid
	assert.Equal(t, "1234567890", theBid.ID, "Bad bid ID. Expected %s, got %s", "1234567890", theBid.ID)
}

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderRubicon, config.Adapter{
		Endpoint: "uri",
		XAPI: config.AdapterXAPI{
			Username: "xuser",
			Password: "xpass",
			Tracker:  "pbs-test-tracker",
		}})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "rubicontest", bidder)
}
