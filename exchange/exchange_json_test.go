package exchange

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/buger/jsonparser"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/pbsmetrics"
	"github.com/yudai/gojsondiff"
	"github.com/yudai/gojsondiff/formatter"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// DoTests executes tests for all the *.json files in this directory.
func TestExchange(t *testing.T) {
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
	bid, err := ex.HoldAuction(context.Background(), &spec.IncomingRequest.OrtbRequest, mockIdFetcher(spec.IncomingRequest.Usersyncs), pbsmetrics.Labels{})
	extractResponseTimes(t, filename, bid)
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

// extractResponseTimes validates the format of bid.ext.responsetimemillis, and then removes it.
// This is done because the response time will change from run to run, so it's impossible to hardcode a value
// into the JSON. The best we can do is make sure that the property exists.
func extractResponseTimes(t *testing.T, context string, bid *openrtb.BidResponse) {
	if _, dataType, _, err := jsonparser.Get(bid.Ext, "responsetimemillis"); err != nil || dataType != jsonparser.Object {
		t.Errorf("%s: Exchange did not return ext.responsetimemillis object: %v", context, err)
		return
	}
	// Delete the response times so that they don't appear in the JSON, because they can't be tested reliably anyway.
	// If there's no other ext, just delete it altogether.
	bid.Ext = jsonparser.Delete(bid.Ext, "responsetimemillis")
	if diff, err := gojsondiff.New().Compare(bid.Ext, []byte("{}")); err == nil && !diff.Modified() {
		bid.Ext = nil
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
		adapterMap: adapters,
		me:         pbsmetrics.NewMetricsEngine(&config.Configuration{}, openrtb_ext.BidderList()),
		cache:      &wellBehavedCache{},
		cacheTime:  0,
	}
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

func (b *validatingBidder) requestBid(ctx context.Context, request *openrtb.BidRequest, name openrtb_ext.BidderName, bidAdjustment float64) (seatBid *pbsOrtbSeatBid, errs []error) {
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
