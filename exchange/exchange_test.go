package exchange

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/prebid/prebid-server/stored_requests"
	"github.com/prebid/prebid-server/stored_requests/backends/file_fetcher"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/currencies"
	"github.com/prebid/prebid-server/prebid_cache_client"

	"github.com/buger/jsonparser"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/gdpr"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbsmetrics"
	metricsConf "github.com/prebid/prebid-server/pbsmetrics/config"
	"github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
	"github.com/yudai/gojsondiff"
	"github.com/yudai/gojsondiff/formatter"
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
		Adapters: blankAdapterConfig(openrtb_ext.BidderList()),
	}

	e := NewExchange(server.Client(), nil, cfg, pbsmetrics.NewMetrics(metrics.NewRegistry(), knownAdapters), adapters.ParseBidderInfos("../static/bidder-info", openrtb_ext.BidderList()), gdpr.AlwaysAllow{}, currencies.NewRateConverterDefault()).(*exchange)
	for _, bidderName := range knownAdapters {
		if _, ok := e.adapterMap[bidderName]; !ok {
			t.Errorf("NewExchange produced an Exchange without bidder %s", bidderName)
		}
	}
	if e.cacheTime != time.Duration(cfg.CacheURL.ExpectedTimeMillis)*time.Millisecond {
		t.Errorf("Bad cacheTime. Expected 20 ms, got %s", e.cacheTime.String())
	}
}

// The objective is to get to execute e.buildBidResponse(ctx.Background(), liveA... ) (*openrtb.BidResponse, error)
// and check whether the returned request successfully prints any '&' characters as it should
// To do so, we:
// 	1) Write the endpoint adapter URL with an '&' character into a new config,Configuration struct
// 	   as specified in https://github.com/prebid/prebid-server/issues/465
// 	2) Initialize a new exchange with said configuration
// 	3) Build all the parameters e.buildBidResponse(ctx.Background(), liveA... ) needs including the
// 	   sample request as specified in https://github.com/prebid/prebid-server/issues/465
// 	4) Build a BidResponse struct using exchange.buildBidResponse(ctx.Background(), liveA... )
// 	5) Assert we have no '&' characters in the response that exchange.buildBidResponse returns
func TestCharacterEscape(t *testing.T) {
	/* 1) Adapter with a '& char in its endpoint property 		*/
	/*    https://github.com/prebid/prebid-server/issues/465	*/
	cfg := &config.Configuration{
		Adapters: make(map[string]config.Adapter, 1),
	}
	cfg.Adapters["appnexus"] = config.Adapter{
		Endpoint: "http://ib.adnxs.com/openrtb2?query1&query2", //Note the '&' character in there
	}

	/* 	2) Init new exchange with said configuration			*/
	//Other parameters also needed to create exchange
	handlerNoBidServer := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }
	server := httptest.NewServer(http.HandlerFunc(handlerNoBidServer))
	defer server.Close()

	e := NewExchange(server.Client(), nil, cfg, pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList()), adapters.ParseBidderInfos("../static/bidder-info", openrtb_ext.BidderList()), gdpr.AlwaysAllow{}, currencies.NewRateConverterDefault()).(*exchange)

	/* 	3) Build all the parameters e.buildBidResponse(ctx.Background(), liveA... ) needs */
	//liveAdapters []openrtb_ext.BidderName,
	liveAdapters := make([]openrtb_ext.BidderName, 1)
	liveAdapters[0] = "appnexus"

	//adapterBids map[openrtb_ext.BidderName]*pbsOrtbSeatBid,
	adapterBids := make(map[openrtb_ext.BidderName]*pbsOrtbSeatBid, 1)
	adapterBids["appnexus"] = &pbsOrtbSeatBid{currency: "USD"}

	//An openrtb.BidRequest struct as specified in https://github.com/prebid/prebid-server/issues/465
	bidRequest := &openrtb.BidRequest{
		ID: "some-request-id",
		Imp: []openrtb.Imp{{
			ID:     "some-impression-id",
			Banner: &openrtb.Banner{Format: []openrtb.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
			Ext:    json.RawMessage(`{"appnexus": {"placementId": 10433394}}`),
		}},
		Site:   &openrtb.Site{Page: "prebid.org", Ext: json.RawMessage(`{"amp":0}`)},
		Device: &openrtb.Device{UA: "curl/7.54.0", IP: "::1"},
		AT:     1,
		TMax:   500,
		Ext:    json.RawMessage(`{"id": "some-request-id","site": {"page": "prebid.org"},"imp": [{"id": "some-impression-id","banner": {"format": [{"w": 300,"h": 250},{"w": 300,"h": 600}]},"ext": {"appnexus": {"placementId": 10433394}}}],"tmax": 500}`),
	}

	//resolvedRequest json.RawMessage
	resolvedRequest := json.RawMessage(`{"id": "some-request-id","site": {"page": "prebid.org"},"imp": [{"id": "some-impression-id","banner": {"format": [{"w": 300,"h": 250},{"w": 300,"h": 600}]},"ext": {"appnexus": {"placementId": 10433394}}}],"tmax": 500}`)

	//adapterExtra map[openrtb_ext.BidderName]*seatResponseExtra,
	adapterExtra := make(map[openrtb_ext.BidderName]*seatResponseExtra, 1)
	adapterExtra["appnexus"] = &seatResponseExtra{
		ResponseTimeMillis: 5,
		Errors:             []openrtb_ext.ExtBidderError{{Code: 999, Message: "Post ib.adnxs.com/openrtb2?query1&query2: unsupported protocol scheme \"\""}},
	}

	//errList []error
	var errList []error

	/* 	4) Build bid response 									*/
	bid_resp, err := e.buildBidResponse(context.Background(), liveAdapters, adapterBids, bidRequest, resolvedRequest, adapterExtra, errList)

	/* 	5) Assert we have no errors and one '&' character as we are supposed to 	*/
	if err != nil {
		t.Errorf("exchange.buildBidResponse returned unexpected error: %v", err)
	}
	if len(errList) > 0 {
		t.Errorf("exchange.buildBidResponse returned %d errors", len(errList))
	}
	if bytes.Contains(bid_resp.Ext, []byte("u0026")) {
		t.Errorf("exchange.buildBidResponse() did not correctly print the '&' characters %s", string(bid_resp.Ext))
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
		Adapters: make(map[string]config.Adapter, len(openrtb_ext.BidderMap)),
	}
	for _, bidder := range openrtb_ext.BidderList() {
		cfg.Adapters[strings.ToLower(string(bidder))] = config.Adapter{
			Endpoint: server.URL,
		}
	}
	cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderFacebook))] = config.Adapter{
		Endpoint:   server.URL,
		PlatformID: "abc",
	}
	categoriesFetcher, error := newCategoryFetcher("./test/category-mapping")
	if error != nil {
		t.Errorf("Failed to create a category Fetcher: %v", error)
	}
	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList())
	ex := NewExchange(server.Client(), &wellBehavedCache{}, cfg, theMetrics, adapters.ParseBidderInfos("../static/bidder-info", openrtb_ext.BidderList()), gdpr.AlwaysAllow{}, currencies.NewRateConverterDefault())
	_, err := ex.HoldAuction(context.Background(), newRaceCheckingRequest(t), &emptyUsersync{}, pbsmetrics.Labels{}, &categoriesFetcher)
	if err != nil {
		t.Errorf("HoldAuction returned unexpected error: %v", err)
	}
}

func newCategoryFetcher(directory string) (stored_requests.CategoryFetcher, error) {
	fetcher, err := file_fetcher.NewFileFetcher(directory)
	if err != nil {
		return nil, err
	}
	catfetcher, ok := fetcher.(stored_requests.CategoryFetcher)
	if !ok {
		return nil, fmt.Errorf("Failed to type cast fetcher to CategoryFetcher")
	}
	return catfetcher, nil
}

// newRaceCheckingRequest builds a BidRequest from all the params in the
// adapters/{bidder}/{bidder}test/params/race/*.json files
func newRaceCheckingRequest(t *testing.T) *openrtb.BidRequest {
	dnt := int8(1)
	return &openrtb.BidRequest{
		Site: &openrtb.Site{
			Page:   "www.some.domain.com",
			Domain: "domain.com",
			Publisher: &openrtb.Publisher{
				ID: "some-publisher-id",
			},
		},
		Device: &openrtb.Device{
			UA:       "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/67.0.3396.87 Safari/537.36",
			IFA:      "ifa",
			IP:       "132.173.230.74",
			DNT:      &dnt,
			Language: "EN",
		},
		Source: &openrtb.Source{
			TID: "61018dc9-fa61-4c41-b7dc-f90b9ae80e87",
		},
		User: &openrtb.User{
			ID:       "our-id",
			BuyerUID: "their-id",
			Ext:      json.RawMessage(`{"consent":"BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw","digitrust":{"id":"digi-id","keyv":1,"pref":1}}`),
		},
		Regs: &openrtb.Regs{
			Ext: json.RawMessage(`{"gdpr":1}`),
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

func TestPanicRecovery(t *testing.T) {
	cfg := &config.Configuration{
		CacheURL: config.Cache{
			ExpectedTimeMillis: 20,
		},
		Adapters: blankAdapterConfig(openrtb_ext.BidderList()),
	}

	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList())
	e := NewExchange(&http.Client{}, nil, cfg, theMetrics, adapters.ParseBidderInfos("../static/bidder-info", openrtb_ext.BidderList()), gdpr.AlwaysAllow{}, currencies.NewRateConverterDefault()).(*exchange)
	chBids := make(chan *bidResponseWrapper, 1)
	panicker := func(aName openrtb_ext.BidderName, coreBidder openrtb_ext.BidderName, request *openrtb.BidRequest, bidlabels *pbsmetrics.AdapterLabels, conversions currencies.Conversions) {
		panic("panic!")
	}
	recovered := e.recoverSafely(panicker, chBids)
	apnLabels := pbsmetrics.AdapterLabels{
		Source:      pbsmetrics.DemandWeb,
		RType:       pbsmetrics.ReqTypeORTB2Web,
		Adapter:     openrtb_ext.BidderAppnexus,
		PubID:       "test1",
		Browser:     pbsmetrics.BrowserSafari,
		CookieFlag:  pbsmetrics.CookieFlagYes,
		AdapterBids: pbsmetrics.AdapterBidNone,
	}
	recovered(openrtb_ext.BidderAppnexus, openrtb_ext.BidderAppnexus, nil, &apnLabels, nil)
}

func buildImpExt(t *testing.T, jsonFilename string) json.RawMessage {
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
	return json.RawMessage(toReturn)
}

func TestPanicRecoveryHighLevel(t *testing.T) {
	noBidServer := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	}
	server := httptest.NewServer(http.HandlerFunc(noBidServer))
	defer server.Close()

	cfg := &config.Configuration{
		Adapters: make(map[string]config.Adapter, len(openrtb_ext.BidderMap)),
	}
	for _, bidder := range openrtb_ext.BidderList() {
		cfg.Adapters[strings.ToLower(string(bidder))] = config.Adapter{
			Endpoint: server.URL,
		}
	}
	e := NewExchange(server.Client(), nil, cfg, pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList()), adapters.ParseBidderInfos("../static/bidder-info", openrtb_ext.BidderList()), gdpr.AlwaysAllow{}, currencies.NewRateConverterDefault()).(*exchange)

	e.adapterMap[openrtb_ext.BidderBeachfront] = panicingAdapter{}
	e.adapterMap[openrtb_ext.BidderAppnexus] = panicingAdapter{}

	request := &openrtb.BidRequest{
		Site: &openrtb.Site{
			Page:   "www.some.domain.com",
			Domain: "domain.com",
			Publisher: &openrtb.Publisher{
				ID: "some-publisher-id",
			},
		},
		User: &openrtb.User{
			ID:       "our-id",
			BuyerUID: "their-id",
			Ext:      json.RawMessage(`{"consent":"BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw","digitrust":{"id":"digi-id","keyv":1,"pref":1}}`),
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
		}},
	}

	categoriesFetcher, error := newCategoryFetcher("./test/category-mapping")
	if error != nil {
		t.Errorf("Failed to create a category Fetcher: %v", error)
	}
	_, err := e.HoldAuction(context.Background(), request, &emptyUsersync{}, pbsmetrics.Labels{}, &categoriesFetcher)
	if err != nil {
		t.Errorf("HoldAuction returned unexpected error: %v", err)
	}

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

// TestExchangeJSON executes tests for all the *.json files in exchangetest.
func TestExchangeJSON(t *testing.T) {
	if specFiles, err := ioutil.ReadDir("./exchangetest"); err == nil {
		for _, specFile := range specFiles {
			fileName := "./exchangetest/" + specFile.Name()
			fileDisplayName := "exchange/exchangetest/" + specFile.Name()
			specData, err := loadFile(fileName)
			if err != nil {
				t.Fatalf("Failed to load contents of file %s: %v", fileDisplayName, err)
			}

			runSpec(t, fileDisplayName, specData)
		}
	}
}

// LoadFile reads and parses a file as a test case. If something goes wrong, it returns an error.
func loadFile(filename string) (*exchangeSpec, error) {
	specData, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("Failed to read file %s: %v", filename, err)
	}

	var spec exchangeSpec
	if err := json.Unmarshal(specData, &spec); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal JSON from file: %v", err)
	}

	return &spec, nil
}

func runSpec(t *testing.T, filename string, spec *exchangeSpec) {
	aliases, errs := parseAliases(&spec.IncomingRequest.OrtbRequest)
	if len(errs) != 0 {
		t.Fatalf("%s: Failed to parse aliases", filename)
	}
	ex := newExchangeForTests(t, filename, spec.OutgoingRequests, aliases)
	biddersInAuction := findBiddersInAuction(t, filename, &spec.IncomingRequest.OrtbRequest)
	categoriesFetcher, error := newCategoryFetcher("./test/category-mapping")
	if error != nil {
		t.Errorf("Failed to create a category Fetcher: %v", error)
	}
	bid, err := ex.HoldAuction(context.Background(), &spec.IncomingRequest.OrtbRequest, mockIdFetcher(spec.IncomingRequest.Usersyncs), pbsmetrics.Labels{}, &categoriesFetcher)
	responseTimes := extractResponseTimes(t, filename, bid)
	for _, bidderName := range biddersInAuction {
		if _, ok := responseTimes[bidderName]; !ok {
			t.Errorf("%s: Response JSON missing expected ext.responsetimemillis.%s", filename, bidderName)
		}
	}
	if spec.Response.Bids != nil {
		diffOrtbResponses(t, filename, spec.Response.Bids, bid)
		if err == nil {
			if spec.Response.Error != "" {
				t.Errorf("%s: Exchange did not return expected error: %s", filename, spec.Response.Error)
			}
		} else {
			if err.Error() != spec.Response.Error {
				t.Errorf("%s: Exchange returned different errors. Expected %s, got %s", filename, spec.Response.Error, err.Error())
			}
		}
	}
}

func findBiddersInAuction(t *testing.T, context string, req *openrtb.BidRequest) []string {
	if splitImps, err := splitImps(req.Imp); err != nil {
		t.Errorf("%s: Failed to parse Bidders from request: %v", context, err)
		return nil
	} else {
		bidders := make([]string, 0, len(splitImps))
		for bidderName := range splitImps {
			bidders = append(bidders, bidderName)
		}
		return bidders
	}
}

// extractResponseTimes validates the format of bid.ext.responsetimemillis, and then removes it.
// This is done because the response time will change from run to run, so it's impossible to hardcode a value
// into the JSON. The best we can do is make sure that the property exists.
func extractResponseTimes(t *testing.T, context string, bid *openrtb.BidResponse) map[string]int {
	if data, dataType, _, err := jsonparser.Get(bid.Ext, "responsetimemillis"); err != nil || dataType != jsonparser.Object {
		t.Errorf("%s: Exchange did not return ext.responsetimemillis object: %v", context, err)
		return nil
	} else {
		responseTimes := make(map[string]int)
		if err := json.Unmarshal(data, &responseTimes); err != nil {
			t.Errorf("%s: Failed to unmarshal ext.responsetimemillis into map[string]int: %v", context, err)
			return nil
		}

		// Delete the response times so that they don't appear in the JSON, because they can't be tested reliably anyway.
		// If there's no other ext, just delete it altogether.
		bid.Ext = jsonparser.Delete(bid.Ext, "responsetimemillis")
		if diff, err := gojsondiff.New().Compare(bid.Ext, []byte("{}")); err == nil && !diff.Modified() {
			bid.Ext = nil
		}
		return responseTimes
	}
}

func newExchangeForTests(t *testing.T, filename string, expectations map[string]*bidderSpec, aliases map[string]string) Exchange {
	adapters := make(map[openrtb_ext.BidderName]adaptedBidder)
	for _, bidderName := range openrtb_ext.BidderMap {
		if spec, ok := expectations[string(bidderName)]; ok {
			adapters[bidderName] = &validatingBidder{
				t:             t,
				fileName:      filename,
				bidderName:    string(bidderName),
				expectations:  map[string]*bidderRequest{string(bidderName): spec.ExpectedRequest},
				mockResponses: map[string]bidderResponse{string(bidderName): spec.MockResponse},
			}
		}
	}
	for alias, coreBidder := range aliases {
		if spec, ok := expectations[alias]; ok {
			if bidder, ok := adapters[openrtb_ext.BidderName(coreBidder)]; ok {
				bidder.(*validatingBidder).expectations[alias] = spec.ExpectedRequest
				bidder.(*validatingBidder).mockResponses[alias] = spec.MockResponse
			} else {
				adapters[openrtb_ext.BidderName(coreBidder)] = &validatingBidder{
					t:             t,
					fileName:      filename,
					bidderName:    coreBidder,
					expectations:  map[string]*bidderRequest{coreBidder: spec.ExpectedRequest},
					mockResponses: map[string]bidderResponse{coreBidder: spec.MockResponse},
				}
			}
		}
	}

	return &exchange{
		adapterMap:          adapters,
		me:                  metricsConf.NewMetricsEngine(&config.Configuration{}, openrtb_ext.BidderList()),
		cache:               &wellBehavedCache{},
		cacheTime:           0,
		gDPR:                gdpr.AlwaysAllow{},
		currencyConverter:   currencies.NewRateConverterDefault(),
		UsersyncIfAmbiguous: false,
	}
}

func newExtRequest() openrtb_ext.ExtRequest {
	priceGran := openrtb_ext.PriceGranularity{
		Precision: 2,
		Ranges: []openrtb_ext.GranularityRange{
			{
				Min:       0.0,
				Max:       20.0,
				Increment: 2.0,
			},
		},
	}

	brandCat := openrtb_ext.ExtIncludeBrandCategory{PrimaryAdServer: 1}

	reqExt := openrtb_ext.ExtRequestTargeting{
		PriceGranularity:     priceGran,
		IncludeWinners:       true,
		IncludeBrandCategory: brandCat,
	}

	return openrtb_ext.ExtRequest{
		Prebid: openrtb_ext.ExtRequestPrebid{
			Targeting: &reqExt,
		},
	}
}

func TestCategoryMapping(t *testing.T) {

	categoriesFetcher, error := newCategoryFetcher("./test/category-mapping")
	if error != nil {
		t.Errorf("Failed to create a category Fetcher: %v", error)
	}

	requestExt := newExtRequest()

	targData := &targetData{
		priceGranularity: requestExt.Prebid.Targeting.PriceGranularity,
		includeWinners:   true,
	}
	durationRange := make([]int, 0)
	durationRange = append(durationRange, 15)
	durationRange = append(durationRange, 30)
	durationRange = append(durationRange, 50)
	requestExt.Prebid.Targeting.DurationRangeSec = durationRange

	adapterBids := make(map[openrtb_ext.BidderName]*pbsOrtbSeatBid)

	cats1 := []string{"IAB1-3"}
	cats2 := []string{"IAB1-4"}
	cats3 := []string{"IAB1-1000"}
	cats4 := []string{"IAB1-2000"}
	bid1 := openrtb.Bid{ID: "bid_id1", ImpID: "imp_id1", Price: 10.0000, Cat: cats1, W: 1, H: 1}
	bid2 := openrtb.Bid{ID: "bid_id2", ImpID: "imp_id2", Price: 20.0000, Cat: cats2, W: 1, H: 1}
	bid3 := openrtb.Bid{ID: "bid_id3", ImpID: "imp_id3", Price: 30.0000, Cat: cats3, W: 1, H: 1}
	bid4 := openrtb.Bid{ID: "bid_id4", ImpID: "imp_id4", Price: 40.0000, Cat: cats4, W: 1, H: 1}

	bid1_1 := pbsOrtbBid{&bid1, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}}
	bid1_2 := pbsOrtbBid{&bid2, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 40}}
	bid1_3 := pbsOrtbBid{&bid3, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30, PrimaryCategory: "AdapterOverride"}}
	bid1_4 := pbsOrtbBid{&bid4, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}}

	innerBids := []*pbsOrtbBid{
		&bid1_1,
		&bid1_2,
		&bid1_3,
		&bid1_4,
	}

	seatBid := pbsOrtbSeatBid{innerBids, "USD", nil, nil}
	bidderName1 := openrtb_ext.BidderName("appnexus")

	adapterBids[bidderName1] = &seatBid

	bidCategory, adapterBids, err := applyCategoryMapping(nil, requestExt, adapterBids, categoriesFetcher, targData)

	assert.Equal(t, nil, err, "Category mapping error should be empty")
	assert.Equal(t, "10.00_Electronics_30s", bidCategory["bid_id1"], "Category mapping doesn't match")
	assert.Equal(t, "20.00_Sports_50s", bidCategory["bid_id2"], "Category mapping doesn't match")
	assert.Equal(t, "20.00_AdapterOverride_30s", bidCategory["bid_id3"], "Category mapping override from adapter didn't take")
	assert.Equal(t, 3, len(adapterBids[bidderName1].bids), "Bidders number doesn't match")
	assert.Equal(t, 3, len(bidCategory), "Bidders category mapping doesn't match")
}

func TestCategoryDedupe(t *testing.T) {

	categoriesFetcher, error := newCategoryFetcher("./test/category-mapping")
	if error != nil {
		t.Errorf("Failed to create a category Fetcher: %v", error)
	}

	requestExt := newExtRequest()

	targData := &targetData{
		priceGranularity: requestExt.Prebid.Targeting.PriceGranularity,
		includeWinners:   true,
	}

	adapterBids := make(map[openrtb_ext.BidderName]*pbsOrtbSeatBid)

	cats1 := []string{"IAB1-3"}
	cats2 := []string{"IAB1-4"}
	// bid3 will be same price, category, and duration as bid1 so one of them should get removed
	cats4 := []string{"IAB1-2000"}
	bid1 := openrtb.Bid{ID: "bid_id1", ImpID: "imp_id1", Price: 10.0000, Cat: cats1, W: 1, H: 1}
	bid2 := openrtb.Bid{ID: "bid_id2", ImpID: "imp_id2", Price: 15.0000, Cat: cats2, W: 1, H: 1}
	bid3 := openrtb.Bid{ID: "bid_id3", ImpID: "imp_id3", Price: 10.0000, Cat: cats1, W: 1, H: 1}
	bid4 := openrtb.Bid{ID: "bid_id4", ImpID: "imp_id4", Price: 20.0000, Cat: cats4, W: 1, H: 1}

	bid1_1 := pbsOrtbBid{&bid1, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}}
	bid1_2 := pbsOrtbBid{&bid2, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 50}}
	bid1_3 := pbsOrtbBid{&bid3, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}}
	bid1_4 := pbsOrtbBid{&bid4, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}}

	selectedBids := make(map[string]int)
	expectedCategories := map[string]string{
		"bid_id1": "10.00_Electronics_30s",
		"bid_id2": "14.00_Sports_50s",
		"bid_id3": "10.00_Electronics_30s",
	}

	numIterations := 10

	// Run the function many times, this should be enough for the 50% chance of which bid to remove to remove bid1 sometimes
	// and bid3 others. It's conceivably possible (but highly unlikely) that the same bid get chosen every single time, but
	// if you notice false fails from this test increase numIterations to make it even less likely to happen.
	for i := 0; i < numIterations; i++ {
		innerBids := []*pbsOrtbBid{
			&bid1_1,
			&bid1_2,
			&bid1_3,
			&bid1_4,
		}

		seatBid := pbsOrtbSeatBid{innerBids, "USD", nil, nil}
		bidderName1 := openrtb_ext.BidderName("appnexus")

		adapterBids[bidderName1] = &seatBid

		bidCategory, adapterBids, err := applyCategoryMapping(nil, requestExt, adapterBids, categoriesFetcher, targData)

		assert.Equal(t, nil, err, "Category mapping error should be empty")
		assert.Equal(t, 2, len(adapterBids[bidderName1].bids), "Bidders number doesn't match")
		assert.Equal(t, 2, len(bidCategory), "Bidders category mapping doesn't match")

		for bidId, bidCat := range bidCategory {
			assert.Equal(t, expectedCategories[bidId], bidCat, "Category mapping doesn't match")
			selectedBids[bidId]++
		}
	}

	assert.Equal(t, numIterations, selectedBids["bid_id2"], "Bid 2 did not make it through every time")
	assert.NotEqual(t, numIterations, selectedBids["bid_id1"], "Bid 1 made it through every time")
	assert.NotEqual(t, numIterations, selectedBids["bid_id3"], "Bid 3 made it through every time")
}

type exchangeSpec struct {
	IncomingRequest  exchangeRequest        `json:"incomingRequest"`
	OutgoingRequests map[string]*bidderSpec `json:"outgoingRequests"`
	Response         exchangeResponse       `json:"response,omitempty"`
}

type exchangeRequest struct {
	OrtbRequest openrtb.BidRequest `json:"ortbRequest"`
	Usersyncs   map[string]string  `json:"usersyncs"`
}

type exchangeResponse struct {
	Bids  *openrtb.BidResponse `json:"bids"`
	Error string               `json:"error,omitempty"`
}

type bidderSpec struct {
	ExpectedRequest *bidderRequest `json:"expectRequest"`
	MockResponse    bidderResponse `json:"mockResponse"`
}

type bidderRequest struct {
	OrtbRequest   openrtb.BidRequest `json:"ortbRequest"`
	BidAdjustment float64            `json:"bidAdjustment"`
}

type bidderResponse struct {
	SeatBid *bidderSeatBid `json:"pbsSeatBid,omitempty"`
	Errors  []string       `json:"errors,omitempty"`
}

// bidderSeatBid is basically a subset of pbsOrtbSeatBid from exchange/bidder.go.
// The only real reason I'm not reusing that type is because I don't want people to think that the
// JSON property tags on those types are contracts in prod.
type bidderSeatBid struct {
	Bids []bidderBid `json:"pbsBids,omitempty"`
}

// bidderBid is basically a subset of pbsOrtbBid from exchange/bidder.go.
// See the comment on bidderSeatBid for more info.
type bidderBid struct {
	Bid  *openrtb.Bid `json:"ortbBid,omitempty"`
	Type string       `json:"bidType,omitempty"`
}

type mockIdFetcher map[string]string

func (f mockIdFetcher) GetId(bidder openrtb_ext.BidderName) (id string, ok bool) {
	id, ok = f[string(bidder)]
	return
}

type validatingBidder struct {
	t          *testing.T
	fileName   string
	bidderName string

	// These are maps because they may contain aliases. They should _at least_ contain an entry for bidderName.
	expectations  map[string]*bidderRequest
	mockResponses map[string]bidderResponse
}

func (b *validatingBidder) requestBid(ctx context.Context, request *openrtb.BidRequest, name openrtb_ext.BidderName, bidAdjustment float64, conversions currencies.Conversions) (seatBid *pbsOrtbSeatBid, errs []error) {
	if expectedRequest, ok := b.expectations[string(name)]; ok {
		if expectedRequest != nil {
			if expectedRequest.BidAdjustment != bidAdjustment {
				b.t.Errorf("%s: Bidder %s got wrong bid adjustment. Expected %f, got %f", b.fileName, name, expectedRequest.BidAdjustment, bidAdjustment)
			}
			diffOrtbRequests(b.t, fmt.Sprintf("Request to %s in %s", string(name), b.fileName), &expectedRequest.OrtbRequest, request)
		}
	} else {
		b.t.Errorf("%s: Bidder %s got unexpected request for alias %s. No input assertions.", b.fileName, b.bidderName, name)
	}

	if mockResponse, ok := b.mockResponses[string(name)]; ok {
		if mockResponse.SeatBid != nil {
			bids := make([]*pbsOrtbBid, len(mockResponse.SeatBid.Bids))
			for i := 0; i < len(bids); i++ {
				bids[i] = &pbsOrtbBid{
					bid:     mockResponse.SeatBid.Bids[i].Bid,
					bidType: openrtb_ext.BidType(mockResponse.SeatBid.Bids[i].Type),
				}
			}

			seatBid = &pbsOrtbSeatBid{
				bids: bids,
			}
		}

		for _, err := range mockResponse.Errors {
			errs = append(errs, errors.New(err))
		}
	} else {
		b.t.Errorf("%s: Bidder %s got unexpected request for alias %s. No mock responses.", b.fileName, b.bidderName, name)
	}

	return
}

func diffOrtbRequests(t *testing.T, description string, expected *openrtb.BidRequest, actual *openrtb.BidRequest) {
	t.Helper()
	actualJSON, err := json.Marshal(actual)
	if err != nil {
		t.Fatalf("%s failed to marshal actual BidRequest into JSON. %v", description, err)
	}

	expectedJSON, err := json.Marshal(expected)
	if err != nil {
		t.Fatalf("%s failed to marshal expected BidRequest into JSON. %v", description, err)
	}

	diffJson(t, description, actualJSON, expectedJSON)
}

func diffOrtbResponses(t *testing.T, description string, expected *openrtb.BidResponse, actual *openrtb.BidResponse) {
	t.Helper()
	// The OpenRTB spec is wonky here. Since "bidresponse.seatbid" is an array, order technically matters to any JSON diff or
	// deep equals method. However, for all intents and purposes it really *doesn't* matter. ...so this nasty logic makes compares
	// the seatbids in an order-independent way.
	//
	// Note that the same thing is technically true of the "seatbid[i].bid" array... but since none of our exchange code relies on
	// this implementation detail, I'm cutting a corner and ignoring it here.
	actualSeats := mapifySeatBids(t, description, actual.SeatBid)
	expectedSeats := mapifySeatBids(t, description, expected.SeatBid)
	actualJSON, err := json.Marshal(actualSeats)
	if err != nil {
		t.Fatalf("%s failed to marshal actual BidResponse into JSON. %v", description, err)
	}

	expectedJSON, err := json.Marshal(expectedSeats)
	if err != nil {
		t.Fatalf("%s failed to marshal expected BidResponse into JSON. %v", description, err)
	}

	diffJson(t, description, actualJSON, expectedJSON)
}

func mapifySeatBids(t *testing.T, context string, seatBids []openrtb.SeatBid) map[string]*openrtb.SeatBid {
	seatMap := make(map[string]*openrtb.SeatBid, len(seatBids))
	for i := 0; i < len(seatBids); i++ {
		seatName := seatBids[i].Seat
		if _, ok := seatMap[seatName]; ok {
			t.Fatalf("%s: Contains duplicate Seat: %s", context, seatName)
		} else {
			seatMap[seatName] = &seatBids[i]
		}
	}
	return seatMap
}

// diffJson compares two JSON byte arrays for structural equality. It will produce an error if either
// byte array is not actually JSON.
func diffJson(t *testing.T, description string, actual []byte, expected []byte) {
	t.Helper()
	diff, err := gojsondiff.New().Compare(actual, expected)
	if err != nil {
		t.Fatalf("%s json diff failed. %v", description, err)
	}

	if diff.Modified() {
		var left interface{}
		if err := json.Unmarshal(actual, &left); err != nil {
			t.Fatalf("%s json did not match, but unmarhsalling failed. %v", description, err)
		}
		printer := formatter.NewAsciiFormatter(left, formatter.AsciiFormatterConfig{
			ShowArrayIndex: true,
		})
		output, err := printer.Format(diff)
		if err != nil {
			t.Errorf("%s did not match, but diff formatting failed. %v", description, err)
		} else {
			t.Errorf("%s json did not match expected.\n\n%s", description, output)
		}
	}
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

type wellBehavedCache struct{}

func (c *wellBehavedCache) PutJson(ctx context.Context, values []prebid_cache_client.Cacheable) ([]string, []error) {
	ids := make([]string, len(values))
	for i := 0; i < len(values); i++ {
		ids[i] = strconv.Itoa(i)
	}
	return ids, nil
}

type emptyUsersync struct{}

func (e *emptyUsersync) GetId(bidder openrtb_ext.BidderName) (string, bool) {
	return "", false
}

type mockUsersync struct {
	syncs map[string]string
}

func (e *mockUsersync) GetId(bidder openrtb_ext.BidderName) (id string, exists bool) {
	id, exists = e.syncs[string(bidder)]
	return
}

type panicingAdapter struct{}

func (panicingAdapter) requestBid(ctx context.Context, request *openrtb.BidRequest, name openrtb_ext.BidderName, bidAdjustment float64, conversions currencies.Conversions) (posb *pbsOrtbSeatBid, errs []error) {
	panic("Panic! Panic! The world is ending!")
}
