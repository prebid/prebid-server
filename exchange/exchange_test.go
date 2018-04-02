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
	_, err := ex.HoldAuction(context.Background(), newRaceCheckingRequest(t), &emptyUsersync{}, pbsmetrics.Labels{})
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

type mockAdapter struct {
	seatBid *pbsOrtbSeatBid
	errs    []error
}

func (a *mockAdapter) requestBid(ctx context.Context, request *openrtb.BidRequest, name openrtb_ext.BidderName, bidAdjustment float64) (*pbsOrtbSeatBid, []error) {
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

	// adapterList := []openrtb_ext.BidderName{BidderDummy, BidderDummy2, BidderDummy3}

	e.me = &pbsmetrics.DummyMetricsEngine{}
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
	sb1.bids[0].bid.CrID = "12"
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
	sb1.bids[1].bid.CrID = "34"
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
	sb1.bids[0].bid.CrID = "123"
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
	sb1.bids[1].bid.CrID = "456"
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
	sb1.bids[0].bid.CrID = "135"
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
