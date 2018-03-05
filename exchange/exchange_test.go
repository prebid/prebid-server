package exchange

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbsmetrics"
	"github.com/rcrowley/go-metrics"
)

func TestNewExchange(t *testing.T) {
	respStatus := 200
	respBody := "{\"bid\":false}"
	server := httptest.NewServer(mockHandler(respStatus, "getBody", respBody))
	defer server.Close()

	knownAdapters := openrtb_ext.BidderList()

	cfg := &config.Configuration{
		CacheURL: config.Cache{
			ExpectedTimeMillis: 20,
		},
	}

	e := NewExchange(server.Client(), nil, cfg, pbsmetrics.NewMetrics(metrics.NewRegistry(), knownAdapters)).(*exchange)
	for _, bidderName := range knownAdapters {
		if _, ok := e.adapterMap[bidderName]; !ok {
			t.Errorf("NewExchange produced an Exchange without bidder %s", bidderName)
		}
	}
	if e.cacheTime != time.Duration(cfg.CacheURL.ExpectedTimeMillis)*time.Millisecond {
		t.Errorf("Bad cacheTime. Expected 20 ms, got %s", e.cacheTime.String())
	}
}

// TestRaceIntegration runs an integration test using all the sample params from
// adapters/{bidder}/{bidder}test/params/race/*.json files.
//
// Its primary goal is to catch race conditions, since parts of the BidRequest passed into MakeBids()
// are shared across many goroutines.
//
// The "known" file names right now are "banner.json" and "video.json". These files should hold params
// which the Bidder would expect on banner or video Imps, respectively.
func TestRaceIntegration(t *testing.T) {
	noBidServer := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	}
	server := httptest.NewServer(http.HandlerFunc(noBidServer))
	defer server.Close()

	cfg := &config.Configuration{
		Adapters: map[string]config.Adapter{
			"facebook": config.Adapter{
				PlatformID: "abc",
			},
		},
	}

	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList())
	ex := NewExchange(server.Client(), &wellBehavedCache{}, cfg, theMetrics)
	_, err := ex.HoldAuction(context.Background(), newRaceCheckingRequest(t), &emptyUsersync{})
	if err != nil {
		t.Errorf("HoldAuction returned unexpected error: %v", err)
	}
}

// newRaceCheckingRequest builds a BidRequest from all the params in the
// adapters/{bidder}/{bidder}test/params/race/*.json files
func newRaceCheckingRequest(t *testing.T) *openrtb.BidRequest {
	return &openrtb.BidRequest{
		Site: &openrtb.Site{
			Page:   "www.some.domain.com",
			Domain: "domain.com",
			Publisher: &openrtb.Publisher{
				ID: "some-publisher-id",
			},
		},
		Imp: []openrtb.Imp{{
			ID: "some-imp-id",
			Banner: &openrtb.Banner{
				Format: []openrtb.Format{{
					W: 300,
					H: 250,
				}, {
					W: 300,
					H: 600,
				}},
			},
			Ext: buildImpExt(t, "banner"),
		}, {
			Video: &openrtb.Video{
				MIMEs:       []string{"video/mp4"},
				MinDuration: 1,
				MaxDuration: 300,
				W:           300,
				H:           600,
			},
			Ext: buildImpExt(t, "video"),
		}},
	}
}

func buildImpExt(t *testing.T, jsonFilename string) openrtb.RawJSON {
	adapterFolders, err := ioutil.ReadDir("../adapters")
	if err != nil {
		t.Fatalf("Failed to open adapters directory: %v", err)
	}
	bidderExts := make(map[string]json.RawMessage, len(openrtb_ext.BidderMap))
	for _, adapterFolder := range adapterFolders {
		if adapterFolder.IsDir() && adapterFolder.Name() != "adapterstest" {
			bidderName := adapterFolder.Name()
			sampleParams := "../adapters/" + bidderName + "/" + bidderName + "test/params/race/" + jsonFilename + ".json"
			// If the file doesn't exist, don't worry about it. I don't think the Go APIs offer a reliable way to check for this.
			fileContents, err := ioutil.ReadFile(sampleParams)
			if err == nil {
				bidderExts[bidderName] = json.RawMessage(fileContents)
			}
		}
	}
	toReturn, err := json.Marshal(bidderExts)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}
	return openrtb.RawJSON(toReturn)
}

func TestHoldAuction(t *testing.T) {
	respStatus := 200
	respBody := "{\"bid\":false}"
	server := httptest.NewServer(mockHandler(respStatus, "getBody", respBody))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	e := NewDummyExchange(server.Client())
	mockAdapterConfig1(e.adapterMap[BidderDummy].(*mockAdapter), "dummy")
	mockAdapterConfig2(e.adapterMap[BidderDummy2].(*mockAdapter), "dummy2")
	mockAdapterConfig3(e.adapterMap[BidderDummy3].(*mockAdapter), "dummy3")

	// Very simple Bid request. The dummy bidders know what to do.
	bidRequest := new(openrtb.BidRequest)
	bidRequest.ID = "This Bid"
	bidRequest.Imp = make([]openrtb.Imp, 2)

	// Need extensions for all the bidders so we know to hold auctions for them.
	impExt := make(map[string]map[string]string)
	impExt["dummy"] = make(map[string]string)
	impExt["dummy2"] = make(map[string]string)
	impExt["dummy3"] = make(map[string]string)
	b, _ := json.Marshal(impExt)
	bidRequest.Imp[0].Ext = b
	bidRequest.Imp[1].Ext = b

	bidResponse, err := e.HoldAuction(ctx, bidRequest, &emptyUsersync{})
	if err != nil {
		t.Errorf("HoldAuction: %s", err.Error())
	}
	bidResponseExt := new(openrtb_ext.ExtBidResponse)
	_ = json.Unmarshal(bidResponse.Ext, bidResponseExt)

	if len(bidResponseExt.ResponseTimeMillis) != 3 {
		t.Errorf("HoldAuction: Did not find 3 response times. Found %d instead", len(bidResponseExt.ResponseTimeMillis))
	}
	if len(bidResponse.SeatBid) != 3 {
		t.Errorf("HoldAuction: Expected 3 SeatBids, found %d instead", len(bidResponse.SeatBid))
	}
	if bidResponse.NBR != nil {
		t.Errorf("HoldAuction: Found invalid auction flag in response: %d", *bidResponse.NBR)
	}
	// Find the indexes of the bidders, as they should be scrambled
	// Set initial value to -1 so we error out if bidders are not found.
	var (
		dummy1 = -1
		dummy3 = -1
	)
	for i, sb := range bidResponse.SeatBid {
		if sb.Seat == "dummy" {
			dummy1 = i
		}
		if sb.Seat == "dummy3" {
			dummy3 = i
		}
	}
	if len(bidResponse.SeatBid[dummy1].Bid) != 2 {
		t.Errorf("HoldAuction: Expected 2 bids from dummy bidder, found %d instead", len(bidResponse.SeatBid[dummy1].Bid))
	}
	if len(bidResponse.SeatBid[dummy3].Bid) != 1 {
		t.Errorf("HoldAuction: Expected 2 bids from dummy bidder, found %d instead", len(bidResponse.SeatBid[dummy1].Bid))
	}

}

func TestGetAllBids(t *testing.T) {
	respStatus := 200
	respBody := "{\"bid\":false}"
	server := httptest.NewServer(mockHandler(respStatus, "getBody", respBody))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	e := NewDummyExchange(server.Client())
	mockAdapterConfig1(e.adapterMap[BidderDummy].(*mockAdapter), "dummy")
	mockAdapterConfig2(e.adapterMap[BidderDummy2].(*mockAdapter), "dummy2")
	mockAdapterConfig3(e.adapterMap[BidderDummy3].(*mockAdapter), "dummy3")

	cleanRequests := map[openrtb_ext.BidderName]*openrtb.BidRequest{
		BidderDummy:  nil,
		BidderDummy2: nil,
		BidderDummy3: nil,
	}
	adapterBids, adapterExtra := e.getAllBids(ctx, cleanRequests, nil, nil)
	if len(adapterBids[BidderDummy].bids) != 2 {
		t.Errorf("GetAllBids failed to get 2 bids from BidderDummy, found %d instead", len(adapterBids[BidderDummy].bids))
	}
	if adapterBids[BidderDummy].bids[0].bid.ID != "1234567890" {
		t.Errorf("GetAllBids failed to get the first bid of BidderDummy")
	}
	if adapterBids[BidderDummy3].bids[0].bid.ID != "MyBid" {
		t.Errorf("GetAllBids failed to get the bid from BidderDummy3")
	}
	if len(adapterExtra) != 3 {
		t.Errorf("GetAllBids failed to return 3 adapterExtra's, got %d instead", len(adapterExtra))
	}

	// Now test with an error condition
	mockAdapterConfigErr1(e.adapterMap[BidderDummy2].(*mockAdapter))
	if len(e.adapterMap[BidderDummy2].(*mockAdapter).errs) != 2 {
		t.Errorf("GetAllBids, Bidder2 adapter error generation failed. Only seeing %d errors", len(e.adapterMap[BidderDummy2].(*mockAdapter).errs))
	}
	adapterBids, adapterExtra = e.getAllBids(ctx, cleanRequests, nil, nil)

	if len(e.adapterMap[BidderDummy2].(*mockAdapter).errs) != 2 {
		t.Errorf("GetAllBids, Bidder2 adapter error generation failed. Only seeing %d errors", len(e.adapterMap[BidderDummy2].(*mockAdapter).errs))
	}
	if len(adapterExtra[BidderDummy2].Errors) != 2 {
		t.Errorf("GetAllBids failed to report 2 errors on Bidder2, found %d errors", len(adapterExtra[BidderDummy2].Errors))
	}
	if len(adapterExtra[BidderDummy].Errors) != 0 {
		t.Errorf("GetAllBids found errors on Bidder1, found %d errors", len(adapterExtra[BidderDummy2].Errors))
	}
	if len(adapterBids[BidderDummy2].bids) != 0 {
		t.Errorf("GetAllBids found bids on Bidder2, found %d bids", len(adapterBids[BidderDummy2].bids))
	}

	// Test with null pointer for bid response
	mockAdapterConfigErr2(e.adapterMap[BidderDummy2].(*mockAdapter))
	adapterBids, adapterExtra = e.getAllBids(ctx, cleanRequests, nil, nil)

	if len(adapterExtra[BidderDummy2].Errors) != 1 {
		t.Errorf("GetAllBids failed to report 1 errors on Bidder2, found %d errors", len(adapterExtra[BidderDummy2].Errors))
	}
	if len(adapterExtra[BidderDummy].Errors) != 0 {
		t.Errorf("GetAllBids found errors on Bidder1, found %d errors", len(adapterExtra[BidderDummy2].Errors))
	}
}

func TestBuildBidResponse(t *testing.T) {
	//  BuildBidResponse(liveAdapters []openrtb_ext.BidderName, adapterBids map[openrtb_ext.BidderName]*adapters.pbsOrtbSeatBid, bidRequest *openrtb.BidRequest, adapterExtra map[openrtb_ext.BidderName]*seatResponseExtra) *openrtb.BidResponse
	respStatus := 200
	respBody := "{\"bid\":false}"
	server := httptest.NewServer(mockHandler(respStatus, "getBody", respBody))
	defer server.Close()

	e := NewDummyExchange(server.Client())
	mockAdapterConfig1(e.adapterMap[BidderDummy].(*mockAdapter), "dummy")
	mockAdapterConfig2(e.adapterMap[BidderDummy2].(*mockAdapter), "dummy2")
	mockAdapterConfig3(e.adapterMap[BidderDummy3].(*mockAdapter), "dummy3")

	// Very simple Bid request. At this point we are just reading these two values
	// Adding targeting to enable targeting tests
	bidReqExt := openrtb_ext.ExtRequest{
		Prebid: openrtb_ext.ExtRequestPrebid{
			Targeting: &openrtb_ext.ExtRequestTargeting{
				PriceGranularity: openrtb_ext.PriceGranularityMedium,
			},
		},
	}
	var bidReqExtRaw openrtb.RawJSON
	bidReqExtRaw, err := json.Marshal(bidReqExt)
	bidRequest := openrtb.BidRequest{
		ID:   "This Bid",
		Test: 0,
		Ext:  bidReqExtRaw,
	}

	liveAdapters := make([]openrtb_ext.BidderName, 3)
	liveAdapters[0] = BidderDummy
	liveAdapters[1] = BidderDummy2
	liveAdapters[2] = BidderDummy3

	adapterBids := make(map[openrtb_ext.BidderName]*pbsOrtbSeatBid)
	adapterExtra := make(map[openrtb_ext.BidderName]*seatResponseExtra)

	var errs1, errs2, errs3 []error
	adapterBids[BidderDummy], errs1 = mockDummyBids1("dummy")
	adapterBids[BidderDummy2], errs2 = mockDummyBids2("dummy2")
	adapterBids[BidderDummy3], errs3 = mockDummyBids3("dummy3")
	adapterExtra[BidderDummy] = &seatResponseExtra{ResponseTimeMillis: 131, Errors: convertErr2Str(errs1)}
	adapterExtra[BidderDummy2] = &seatResponseExtra{ResponseTimeMillis: 97, Errors: convertErr2Str(errs2)}
	adapterExtra[BidderDummy3] = &seatResponseExtra{ResponseTimeMillis: 141, Errors: convertErr2Str(errs3)}

	errList := make([]error, 0, 1)
	targData := &targetData{
		priceGranularity: openrtb_ext.PriceGranularityMedium,
	}
	bidResponse, err := e.buildBidResponse(context.Background(), liveAdapters, adapterBids, &bidRequest, adapterExtra, targData, errList)
	if err != nil {
		t.Errorf("BuildBidResponse: %s", err.Error())
	}
	bidResponseExt := new(openrtb_ext.ExtBidResponse)
	_ = json.Unmarshal(bidResponse.Ext, bidResponseExt)

	if len(bidResponseExt.ResponseTimeMillis) != 3 {
		t.Errorf("BuildBidResponse: Did not find 3 response times. Found %d instead", len(bidResponseExt.ResponseTimeMillis))
	}
	if len(bidResponse.SeatBid) != 3 {
		t.Errorf("BuildBidResponse: Expected 3 SeatBids, found %d instead", len(bidResponse.SeatBid))
	}
	if len(bidResponse.SeatBid) != 3 {
		t.Errorf("BuildBidResponse: Expected 3 SeatBids, found %d instead", len(bidResponse.SeatBid))
	}
	// Find the seat index for BidderDummy
	bidderDummySeat := -1
	for i, seat := range bidResponse.SeatBid {
		if seat.Seat == "dummy" {
			bidderDummySeat = i
		}
	}
	if bidderDummySeat == -1 {
		t.Error("Could not find the SeatBid for BidderDummy!")
	} else {
		bidder1BidExt := make([]openrtb_ext.ExtBid, 2)
		err = json.Unmarshal(bidResponse.SeatBid[bidderDummySeat].Bid[0].Ext, &bidder1BidExt[0])
		if err != nil {
			t.Errorf("Unpacking extensions for bid[0]: %s", err.Error())
		}
		err = json.Unmarshal(bidResponse.SeatBid[bidderDummySeat].Bid[1].Ext, &bidder1BidExt[1])
		if err != nil {
			t.Errorf("Unpacking extensions for bid[1]: %s", err.Error())
		}
		// All tests except for winning bid no longer valid as setting pre bid targeting values moved to exchange/bidder.go
		assertStringValue(t, "bid[0].Targeting[hb_pb_dummy]", "1.30", bidder1BidExt[0].Prebid.Targeting["hb_pb_dummy"])
		assertStringValue(t, "bid[0]Targeting[hb_bidder_dummy]", "dummy", bidder1BidExt[0].Prebid.Targeting["hb_bidder_dummy"])
		assertStringValue(t, "bid[0]Targeting[hb_size_dummy]", "728x90", bidder1BidExt[0].Prebid.Targeting["hb_size_dummy"])
		// This should be the winning bid
		assertStringValue(t, "bid[0].Targeting[hb_pb]", "1.30", bidder1BidExt[0].Prebid.Targeting["hb_pb"])
		_, ok := bidder1BidExt[0].Prebid.Targeting["hb_pb"]
		if !ok {
			t.Errorf("bid[0].Targeting[hb_pb] doesn't exist, but was winning bid.")
		}
		assertStringValue(t, "bid[0]Targeting[hb_bidder]", "dummy", bidder1BidExt[0].Prebid.Targeting["hb_bidder"])
		assertStringValue(t, "bid[0]Targeting[hb_size]", "728x90", bidder1BidExt[0].Prebid.Targeting["hb_size"])
		assertStringValue(t, "bid[1].Targeting[hb_pb_dummy]", "0.70", bidder1BidExt[1].Prebid.Targeting["hb_pb_dummy"])
		assertStringValue(t, "bid[1]Targeting[hb_bidder_dummy]", "dummy", bidder1BidExt[1].Prebid.Targeting["hb_bidder_dummy"])
		assertStringValue(t, "bid[1]Targeting[hb_size_dummy]", "300x250", bidder1BidExt[1].Prebid.Targeting["hb_size_dummy"])
		_, ok = bidder1BidExt[1].Prebid.Targeting["hb_pb"]
		if ok {
			t.Errorf("bid[1].Targeting[hb_pb] exists, but wasn't winning bid. Got \"%s\"", bidder1BidExt[1].Prebid.Targeting["hb_pb"])
		}

	}
	// Now test with an error condition
	adapterBids[BidderDummy2], errs2 = mockDummyBidsErr1()
	adapterExtra[BidderDummy2] = &seatResponseExtra{ResponseTimeMillis: 97, Errors: convertErr2Str(errs2)}

	bidResponse, err = e.buildBidResponse(context.Background(), liveAdapters, adapterBids, &bidRequest, adapterExtra, nil, errList)
	if err != nil {
		t.Errorf("BuildBidResponse: %s", err.Error())
	}
	bidResponseExt = new(openrtb_ext.ExtBidResponse)
	_ = json.Unmarshal(bidResponse.Ext, bidResponseExt)

	// This case we know the order of the adapters, as GetAllBids have not scrambled them
	if len(bidResponse.SeatBid[0].Bid) != 2 {
		t.Errorf("BuildBidResponse: Bidder 1 expected 2 bids, found %d", len(bidResponse.SeatBid[0].Bid))
	}
	if bidResponse.SeatBid[1].Bid[0].ID != "MyBid" {
		t.Errorf("BuildBidResponse: Bidder 3 bid ID not correct. Expected \"MyBid\", found \"%s\"", bidResponse.SeatBid[2].Bid[0].ID)
	}

	// Test with null bid response error
	adapterBids[BidderDummy2], errs2 = mockDummyBidsErr2()
	adapterExtra[BidderDummy2] = &seatResponseExtra{ResponseTimeMillis: 97, Errors: convertErr2Str(errs2)}

	bidResponse, err = e.buildBidResponse(context.Background(), liveAdapters, adapterBids, &bidRequest, adapterExtra, nil, errList)
	if err != nil {
		t.Errorf("BuildBidResponse: %s", err.Error())
	}
	bidResponseExt = new(openrtb_ext.ExtBidResponse)
	_ = json.Unmarshal(bidResponse.Ext, bidResponseExt)

	// This case we know the order of the adapters, as GetAllBids have not scrambled them
	if len(bidResponse.SeatBid[0].Bid) != 2 {
		t.Errorf("BuildBidResponse: Bidder 1 expected 2 bids, found %d", len(bidResponse.SeatBid[0].Bid))
	}
	if bidResponse.SeatBid[1].Bid[0].ID != "MyBid" {
		t.Errorf("BuildBidResponse: Bidder 3 bid ID not correct. Expected \"MyBid\", found \"%s\"", bidResponse.SeatBid[2].Bid[0].ID)
	}
}

var baseRequest = openrtb.BidRequest{
	ID: "foo",
	Imp: []openrtb.Imp{{
		Video: &openrtb.Video{
			MIMEs: []string{"video/mp4"},
		},
	}},
	App: &openrtb.App{},
}

func TestBuyerIdWithoutUser(t *testing.T) {
	req := baseRequest
	runBuyerTest(t, &req, true)
}

func TestBuyerIdWithUser(t *testing.T) {
	req := baseRequest
	req.User = &openrtb.User{
		ID: "abc",
	}

	runBuyerTest(t, &req, true)
}

func TestBuyerIdWithExplicit(t *testing.T) {
	req := baseRequest
	req.User = &openrtb.User{
		ID:       "abc",
		BuyerUID: "def",
	}

	runBuyerTest(t, &req, false)
}

func TestTimeoutComputation(t *testing.T) {
	cacheTimeMillis := 10
	ex := exchange{
		cacheTime: time.Duration(cacheTimeMillis) * time.Millisecond,
	}
	deadline := time.Now()
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	auctionCtx, cancel := ex.makeAuctionContext(ctx, true)
	defer cancel()

	if finalDeadline, ok := auctionCtx.Deadline(); !ok || deadline.Add(-time.Duration(cacheTimeMillis)*time.Millisecond) != finalDeadline {
		t.Errorf("The auction should allocate cacheTime amount of time from the whole request timeout.")
	}
}

func runBuyerTest(t *testing.T, incoming *openrtb.BidRequest, expectBuyeridOverride bool) {
	t.Helper()

	initialId := ""
	if incoming.User != nil {
		initialId = incoming.User.BuyerUID
	}

	overrideId := "apnId"

	incoming.Imp[0].Ext = []byte(`{"appnexus": {}}`)
	syncs := &mockSyncs{
		uids: map[openrtb_ext.BidderName]string{
			openrtb_ext.BidderAppnexus: overrideId,
		},
	}

	bidder := &mockBidder{}

	ex := &exchange{
		adapterMap: map[openrtb_ext.BidderName]adaptedBidder{
			openrtb_ext.BidderAppnexus: bidder,
		},
		m: pbsmetrics.NewBlankMetrics(metrics.NewRegistry(), []openrtb_ext.BidderName{openrtb_ext.BidderAppnexus}),
	}
	ex.HoldAuction(context.Background(), incoming, syncs)

	if bidder.lastRequest == nil {
		t.Fatalf("The Bidder never received a request.")
	}

	if bidder.lastRequest.User == nil {
		t.Fatalf("bidrequest.user was not defined on the Bidder's request.")
	}

	if (expectBuyeridOverride && bidder.lastRequest.User.BuyerUID != overrideId) || (!expectBuyeridOverride && bidder.lastRequest.User.BuyerUID != initialId) {
		t.Errorf("Bidder received bad bidrequest.user.buyeruid. Expected %s, got %s", initialId, bidder.lastRequest.User.BuyerUID)
	}
}

type mockBidder struct {
	lastRequest *openrtb.BidRequest
}

func (b *mockBidder) requestBid(ctx context.Context, request *openrtb.BidRequest, bidderTarg *targetData, name openrtb_ext.BidderName) (*pbsOrtbSeatBid, []error) {
	b.lastRequest = request
	return nil, nil
}

type mockSyncs struct {
	uids map[openrtb_ext.BidderName]string
}

func (m *mockSyncs) GetId(name openrtb_ext.BidderName) (id string, ok bool) {
	id, ok = m.uids[name]
	return
}

func assertStringValue(t *testing.T, object string, expect string, value string) {
	t.Helper()
	if expect != value {
		t.Errorf("Wrong value for %s, expected \"%s\", got \"%s\"", object, expect, value)
	}
}

type mockAdapter struct {
	seatBid *pbsOrtbSeatBid
	errs    []error
}

func (a *mockAdapter) requestBid(ctx context.Context, request *openrtb.BidRequest, targetData *targetData, name openrtb_ext.BidderName) (*pbsOrtbSeatBid, []error) {
	return a.seatBid, a.errs
}

const (
	BidderDummy  openrtb_ext.BidderName = "dummy"
	BidderDummy2 openrtb_ext.BidderName = "dummy2"
	BidderDummy3 openrtb_ext.BidderName = "dummy3"
)

// Tester is responsible for filling bid results into the adapters
func NewDummyExchange(client *http.Client) *exchange {
	e := new(exchange)
	a := new(mockAdapter)
	a.errs = make([]error, 0, 5)

	b := new(mockAdapter)
	b.errs = make([]error, 0, 5)

	c := new(mockAdapter)
	c.errs = make([]error, 0, 5)

	e.adapterMap = map[openrtb_ext.BidderName]adaptedBidder{
		BidderDummy:  a,
		BidderDummy2: b,
		BidderDummy3: c,
	}

	adapterList := []openrtb_ext.BidderName{BidderDummy, BidderDummy2, BidderDummy3}

	e.m = pbsmetrics.NewBlankMetrics(metrics.NewRegistry(), adapterList)
	e.cache = &wellBehavedCache{}
	return e
}

func mockHandler(statusCode int, getBody string, postBody string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		if r.Method == "GET" {
			w.Write([]byte(getBody))
		} else {
			w.Write([]byte(postBody))
		}
	})
}

func mockAdapterConfig1(a *mockAdapter, adapter string) {
	a.seatBid, a.errs = mockDummyBids1(adapter)
}

func mockAdapterConfig2(a *mockAdapter, adapter string) {
	a.seatBid, a.errs = mockDummyBids2(adapter)
}

func mockAdapterConfig3(a *mockAdapter, adapter string) {
	a.seatBid, a.errs = mockDummyBids3(adapter)
}

func mockAdapterConfigErr1(a *mockAdapter) {
	a.seatBid, a.errs = mockDummyBidsErr1()
}

func mockAdapterConfigErr2(a *mockAdapter) {
	a.seatBid, a.errs = mockDummyBidsErr2()
}

func mockDummyBids1(adapter string) (*pbsOrtbSeatBid, []error) {
	var err error
	sb1 := new(pbsOrtbSeatBid)
	sb1.bids = make([]*pbsOrtbBid, 2)
	sb1.bids[0] = new(pbsOrtbBid)
	sb1.bids[1] = new(pbsOrtbBid)
	sb1.bids[0].bid = new(openrtb.Bid)
	sb1.bids[1].bid = new(openrtb.Bid)
	sb1.bids[0].bid.ID = "1234567890"
	sb1.bids[0].bid.W = 728
	sb1.bids[0].bid.H = 90
	sb1.bids[0].bid.Price = 1.34
	sb1.bids[0].bid.ImpID = "1stImp"
	targ := make(map[string]string)
	targ["hb_pb_"+adapter] = "1.30"
	targ["hb_bidder_"+adapter] = adapter
	targ["hb_size_"+adapter] = "728x90"
	sb1.bids[0].bidTargets = targ
	fmt.Println(string(sb1.bids[0].bid.Ext))
	if err != nil {
		fmt.Println("ERROR: Packing ext[0] in mockDummyBids1: " + err.Error())
	}
	sb1.bids[1].bid.ID = "1234567890"
	sb1.bids[1].bid.W = 300
	sb1.bids[1].bid.H = 250
	sb1.bids[1].bid.Price = 0.73
	sb1.bids[1].bid.ImpID = "2ndImp"
	targ = make(map[string]string)
	targ["hb_pb_"+adapter] = "0.70"
	targ["hb_bidder_"+adapter] = adapter
	targ["hb_size_"+adapter] = "300x250"
	sb1.bids[1].bidTargets = targ
	fmt.Println(string(sb1.bids[0].bid.Ext))
	if err != nil {
		fmt.Println("ERROR: Packing ext[0] in mockDummyBids1: " + err.Error())
	}

	errs := make([]error, 0, 5)

	return sb1, errs
}

func mockDummyBids2(adapter string) (*pbsOrtbSeatBid, []error) {
	var err error
	sb1 := new(pbsOrtbSeatBid)
	sb1.bids = make([]*pbsOrtbBid, 2)
	sb1.bids[0] = new(pbsOrtbBid)
	sb1.bids[1] = new(pbsOrtbBid)
	sb1.bids[0].bid = new(openrtb.Bid)
	sb1.bids[1].bid = new(openrtb.Bid)
	sb1.bids[0].bid.ID = "ABC"
	sb1.bids[0].bid.W = 728
	sb1.bids[0].bid.H = 90
	sb1.bids[0].bid.Price = 0.94
	sb1.bids[0].bid.ImpID = "1stImp"
	targ := make(map[string]string)
	targ["hb_pb_"+adapter] = "0.90"
	targ["hb_bidder_"+adapter] = adapter
	targ["hb_size_"+adapter] = "728x90"
	sb1.bids[0].bidTargets = targ
	fmt.Println(string(sb1.bids[0].bid.Ext))
	if err != nil {
		fmt.Println("ERROR: Packing ext[0] in mockDummyBids1: " + err.Error())
	}
	sb1.bids[1].bid.ID = "1234"
	sb1.bids[1].bid.W = 300
	sb1.bids[1].bid.H = 250
	sb1.bids[1].bid.Price = 1.89
	sb1.bids[1].bid.ImpID = "2ndImp"
	targ = make(map[string]string)
	targ["hb_pb_"+adapter] = "1.80"
	targ["hb_bidder_"+adapter] = adapter
	targ["hb_size_"+adapter] = "300x250"
	sb1.bids[1].bidTargets = targ
	fmt.Println(string(sb1.bids[0].bid.Ext))
	if err != nil {
		fmt.Println("ERROR: Packing ext[0] in mockDummyBids1: " + err.Error())
	}

	errs := make([]error, 0, 5)

	return sb1, errs
}
func mockDummyBids3(adapter string) (*pbsOrtbSeatBid, []error) {
	var err error
	sb1 := new(pbsOrtbSeatBid)
	sb1.bids = make([]*pbsOrtbBid, 1)
	sb1.bids[0] = new(pbsOrtbBid)
	sb1.bids[0].bid = new(openrtb.Bid)
	sb1.bids[0].bid.ID = "MyBid"
	sb1.bids[0].bid.W = 728
	sb1.bids[0].bid.H = 90
	sb1.bids[0].bid.Price = 0.34
	sb1.bids[0].bid.ImpID = "1stImp"
	targ := make(map[string]string)
	targ["hb_pb_"+adapter] = "0.30"
	targ["hb_bidder_"+adapter] = adapter
	targ["hb_size_"+adapter] = "728x90"
	sb1.bids[0].bidTargets = targ
	fmt.Println(string(sb1.bids[0].bid.Ext))
	if err != nil {
		fmt.Println("ERROR: Packing ext[0] in mockDummyBids1: " + err.Error())
	}

	errs := make([]error, 0, 5)

	return sb1, errs
}
func mockDummyBidsErr1() (*pbsOrtbSeatBid, []error) {
	sb1 := new(pbsOrtbSeatBid)
	sb1.bids = nil

	errs := make([]error, 0, 5)
	errs = append(errs, errors.New("This was an error"))
	errs = append(errs, errors.New("Another error goes here"))

	return sb1, errs
}
func mockDummyBidsErr2() (*pbsOrtbSeatBid, []error) {
	var sb1 *pbsOrtbSeatBid = nil

	errs := make([]error, 0, 5)
	errs = append(errs, errors.New("This was a FATAL error"))

	return sb1, errs
}

func convertErr2Str(e []error) []string {
	s := make([]string, len(e))
	for i := 0; i < len(e); i++ {
		s[i] = e[i].Error()
	}
	return s
}

type wellBehavedCache struct{}

func (c *wellBehavedCache) PutJson(ctx context.Context, values []json.RawMessage) []string {
	ids := make([]string, len(values))
	for i := 0; i < len(values); i++ {
		ids[i] = strconv.Itoa(i)
	}
	return ids
}
