package openrtb2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/buger/jsonparser"
	"github.com/julienschmidt/httprouter"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	analyticsConf "github.com/prebid/prebid-server/analytics/config"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/currency"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/exchange"
	"github.com/prebid/prebid-server/firstpartydata"
	"github.com/prebid/prebid-server/gdpr"
	metricsConfig "github.com/prebid/prebid-server/metrics/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	pbc "github.com/prebid/prebid-server/prebid_cache_client"
	"github.com/prebid/prebid-server/stored_requests/backends/empty_fetcher"
	"github.com/prebid/prebid-server/util/iputil"
)

// In this file we define:
//  - Auxiliary types
//  - Unit test interface implementations such as mocks
//  - Other auxiliary functions that don't make assertions and don't take t *testing.T as parameter
//
// All of the above are useful for this package's unit test framework.

// ----------------------
// test auxiliary types
// ----------------------
const maxSize = 1024 * 256

type testCase struct {
	BidRequest           json.RawMessage   `json:"mockBidRequest"`
	Config               *testConfigValues `json:"config"`
	ExpectedReturnCode   int               `json:"expectedReturnCode,omitempty"`
	ExpectedErrorMessage string            `json:"expectedErrorMessage"`
	ExpectedBidResponse  json.RawMessage   `json:"expectedBidResponse"`
}

type testConfigValues struct {
	AccountRequired     bool                          `json:"accountRequired"`
	AliasJSON           string                        `json:"aliases"`
	BlacklistedAccounts []string                      `json:"blacklistedAccts"`
	BlacklistedApps     []string                      `json:"blacklistedApps"`
	DisabledAdapters    []string                      `json:"disabledAdapters"`
	CurrencyRates       map[string]map[string]float64 `json:"currencyRates"`
	MockBidder          mockBidInfo                   `json:"mockBidder"`
}

// mockBidInfo carries mock bidder server information that will be read from JSON test files
type mockBidInfo struct {
	BidCurrency string  `json:"currency"`
	BidPrice    float64 `json:"price"`
}

type brokenExchange struct{}

func (e *brokenExchange) HoldAuction(ctx context.Context, r exchange.AuctionRequest, debugLog *exchange.DebugLog) (*openrtb2.BidResponse, error) {
	return nil, errors.New("Critical, unrecoverable error.")
}

// Stored Requests
// first below is valid JSON
// second below is identical to first but with extra '}' for invalid JSON
var testStoredRequestData = map[string]json.RawMessage{
	"1": json.RawMessage(`{"id": "{{UUID}}"}`),
	"2": json.RawMessage(`{
		"id": "{{uuid}}",
		"tmax": 500,
		"ext": {
			"prebid": {
				"targeting": {
					"pricegranularity": "low"
				}
			}
		}
	}`),
	"3": json.RawMessage(`{
		"tmax": 500,
				"ext": {
						"prebid": {
								"targeting": {
										"pricegranularity": "low"
								}
						}
				}}
		}`),
	"4": json.RawMessage(`{"id": "{{UUID}}", "cur": ["USD"]}`),
}

// Stored Imp Requests
// first below has valid JSON but doesn't match schema
// second below has invalid JSON (missing comma after rubicon accountId entry) but otherwise matches schema
// third below has valid JSON and matches schema
var testStoredImpData = map[string]json.RawMessage{
	"1": json.RawMessage(`{
			"id": "adUnit1",
			"ext": {
				"appnexus": {
					"placementId": "abc",
					"position": "above",
					"reserve": 0.35
				},
				"rubicon": {
					"accountId": "abc"
				}
			},
			"video":{
				"w":200,
				"h":300
			}
		}`),
	"2": json.RawMessage(`{
			"id": "adUnit1",
			"ext": {
				"appnexus": {
					"placementId": "abc",
					"position": "above",
					"reserve": 0.35
				},
				"rubicon": {
					"accountId": "abc"
				}
			}
		}`),
	"7": json.RawMessage(`{
			"id": "adUnit1",
			"ext": {
				"appnexus": {
					"placementId": 12345678,
					"position": "above",
					"reserve": 0.35
				},
				"rubicon": {
					"accountId": 23456789
					"siteId": 113932,
					"zoneId": 535510
				}
			}
		}`),
	"9": json.RawMessage(`{
"id": "adUnit1",
			"ext": {
				"appnexus": {
					"placementId": 12345678,
					"position": "above",
					"reserve": 0.35
				},
				"rubicon": {
					"accountId": 23456789,
					"siteId": 113932,
					"zoneId": 535510
				}
			}
		}`),
	"10": json.RawMessage(`{
			"ext": {
				"appnexus": {
					"placementId": 12345678,
					"position": "above",
					"reserve": 0.35
				}
			}
		}`),
}

// Incoming requests with stored request IDs
var testStoredRequests = []string{
	`{
		"id": "ThisID",
		"imp": [
			{
				"video":{
					"h":300,
					"w":200
				},
				"ext": {
					"prebid": {
						"storedrequest": {
							"id": "1"
						},
						"options": {
							"echovideoattrs": true
						}
					}
				}
			}
		],
		"ext": {
			"prebid": {
				"cache": {
					"markup": 1
				},
				"targeting": {
				}
			}
		}
	}`,
	`{
		"id": "ThisID",
		"imp": [
			{
				"id": "adUnit2",
				"ext": {
					"prebid": {
						"storedrequest": {
							"id": "1"
						},
						"options": {
							"echovideoattrs": true
						}
					},
					"appnexus": {
						"placementId": "def",
						"trafficSourceCode": "mysite.com",
						"reserve": null
					},
					"rubicon": null
				}
			}
		],
		"ext": {
			"prebid": {
				"cache": {
					"markup": 1
				},
				"targeting": {
				}
			}
		}
	}`,
	`{
		"id": "ThisID",
		"imp": [
			{
				"ext": {
					"prebid": {
						"storedrequest": {
							"id": "2"
						},
						"options": {
							"echovideoattrs": false
						}
					}
				}
			}
		],
		"ext": {
			"prebid": {
				"storedrequest": {
					"id": "2"
				}
			}
		}
	}`,
	`{
		"id": "ThisID",
		"imp": [
			{
				"id": "some-static-imp",
				"video":{
					"mimes":["video/mp4"]
				},
				"ext": {
					"appnexus": {
						"placementId": "abc",
						"position": "below"
					}
				}
			},
			{
				"ext": {
					"prebid": {
						"storedrequest": {
							"id": "1"
						}
					}
				}
			}
		],
		"ext": {
			"prebid": {
				"cache": {
					"markup": 1
				},
				"targeting": {
				}
			}
		}
	}`,
}

// The expected requests after stored request processing
var testFinalRequests = []string{
	`{
		"id": "ThisID",
		"imp": [
			{
				"video":{
					"h":300,
					"w":200
				},
				"ext":{
					"appnexus":{
						"placementId":"abc",
						"position":"above",
						"reserve":0.35
					},
					"prebid":{
						"storedrequest":{
							"id":"1"
						},
					"options":{
						"echovideoattrs":true
					}
				},
				"rubicon":{
					"accountId":"abc"
				}
			},
			"id":"adUnit1"
			}
		],
		"ext": {
			"prebid": {
				"cache": {
					"markup": 1
				},
				"targeting": {
			}
		}
}
	}`,
	`{
		"id": "ThisID",
		"imp": [
			{
				"video":{
					"w":200,
					"h":300
				},
				"ext":{
					"appnexus":{
						"placementId":"def",
						"position":"above",
						"trafficSourceCode":"mysite.com"
					},
					"prebid":{
						"storedrequest":{
							"id":"1"
						},
						"options":{
							"echovideoattrs":true
						}
					}
				},
				"id":"adUnit2"
			}
		],
		"ext": {
			"prebid": {
				"cache": {
					"markup": 1
				},
				"targeting": {
				}
			}
		}
	}`,
	`{
  		"ext": {
  		  "prebid": {
  		    "storedrequest": {
  		      "id": "2"
  		    },
  		    "targeting": {
  		      "pricegranularity": "low"
  		    }
  		  }
  		},
  		"id": "ThisID",
  		"imp": [
  		  {
  		    "ext": {
  		      "appnexus": {
  		        "placementId": "abc",
  		        "position": "above",
  		        "reserve": 0.35
  		      },
  		      "prebid": {
  		        "storedrequest": {
  		          "id": "2"
  		        },
  		        "options":{
					"echovideoattrs":false
				}
  		      },
  		      "rubicon": {
  		        "accountId": "abc"
  		      }
  		    },
  		    "id": "adUnit1"
  		  }
  		],
  		"tmax": 500
	}
`,
	`{
	"id": "ThisID",
	"imp": [
		{
    		"id": "some-static-imp",
    		"video": {
    		  "mimes": [
    		    "video/mp4"
    		  ]
    		},
    		"ext": {
    		  "appnexus": {
    		    "placementId": "abc",
    		    "position": "below"
    		  }
    		}
  		},
  		{
  		  "ext": {
  		    "appnexus": {
  		      "placementId": "abc",
  		      "position": "above",
  		      "reserve": 0.35
  		    },
  		    "prebid": {
  		      "storedrequest": {
  		        "id": "1"
  		      }
  		    },
  		    "rubicon": {
  		      "accountId": "abc"
  		    }
  		  },
  		  "id": "adUnit1",
		  "video":{
				"w":200,
				"h":300
          }
  		}
	],
	"ext": {
		"prebid": {
			"cache": {
				"markup": 1
			},
			"targeting": {
			}
		}
	}
}`,
}

var testStoredImpIds = []string{
	"adUnit1", "adUnit2", "adUnit1", "some-static-imp",
}

var testStoredImps = []string{
	`{
		"id": "adUnit1",
        "ext": {
        	"appnexus": {
        		"placementId": "abc",
        		"position": "above",
        		"reserve": 0.35
        	},
        	"rubicon": {
        		"accountId": "abc"
        	}
        },
		"video":{
        	"w":200,
        	"h":300
		}
	}`,
	`{
		"id": "adUnit1",
        "ext": {
        	"appnexus": {
        		"placementId": "abc",
        		"position": "above",
        		"reserve": 0.35
        	},
        	"rubicon": {
        		"accountId": "abc"
        	}
        },
		"video":{
        	"w":200,
        	"h":300
		}
	}`,
	`{
			"id": "adUnit1",
			"ext": {
				"appnexus": {
					"placementId": "abc",
					"position": "above",
					"reserve": 0.35
				},
				"rubicon": {
					"accountId": "abc"
				}
			}
		}`,
	``,
}

var testBidRequests = []string{
	`{
		"id": "ThisID",
		"app": {
			"id": "123"
		},
		"imp": [
			{
				"video":{
					"h":300,
					"w":200
				},
				"ext": {
					"prebid": {
						"storedrequest": {
							"id": "1"
						},
						"options": {
							"echovideoattrs": true
						}
					}
				}
			}
		],
		"ext": {
			"prebid": {
				"cache": {
					"markup": 1
				},
				"targeting": {
				}
			}
		}
	}`,
	`{
		"id": "ThisID",
		"site": {
			"page": "prebid.org"
		},
		"imp": [
			{
				"id": "adUnit2",
				"ext": {
					"prebid": {
						"storedrequest": {
							"id": "1"
						},
						"options": {
							"echovideoattrs": true
						}
					},
					"appnexus": {
						"placementId": "def",
						"trafficSourceCode": "mysite.com",
						"reserve": null
					},
					"rubicon": null
				}
			}
		],
		"ext": {
			"prebid": {
				"targeting": {
				}
			}
		}
	}`,
	`{
		"id": "ThisID",
		"app": {
			"id": "123"
		},
		"imp": [
			{
				"ext": {
					"prebid": {
						"storedrequest": {
							"id": "2"
						},
						"options": {
							"echovideoattrs": false
						}
					}
				}
			}
		],
		"ext": {
			"prebid": {
				"storedrequest": {
					"id": "2"
				}
			}
		}
	}`,
	`{
		"id": "ThisID",
		"app": {
			"id": "123"
		},
		"imp": [
			{
				"ext": {
					"prebid": {
						"storedrequest": {
							"id": "2"
						},
						"options": {
							"echovideoattrs": false
						}
					}
				}
			}
		],
		"ext": {
			"prebid": {
				"storedrequest": {
					"id": "2"
				}
			}
		}
	}`,
	`{
		"id": "ThisID",
		"app": {
			"id": "123"
		},
		"imp": [
			{
				"ext": {
					"prebid": {
						"storedrequest": {
							"id": "1"
						},
						"options": {
							"echovideoattrs": false
						}
					}
				}
			}
		],
		"ext": {
			"prebid": {
				"storedrequest": {
					"id": "1"
				}
			}
		}
	}`,
	`{
		"id": "ThisID",
		"imp": [{
			"id": "some-impression-id",
			"banner": {
				"format": [{
						"w": 600,
						"h": 500
					},
					{
						"w": 300,
						"h": 600
					}
				]
			},
			"ext": {
				"appnexus": {
					"placementId": 12883451
				}
			}
		}],
		"ext": {
			"prebid": {
				"debug": true,
				"storedrequest": {
					"id": "4"
				}
			}
		},
	  "site": {
		"page": "https://example.com"
	  }
	}`,
}

// ---------------------------------------------------------
// Some interfaces implemented with the purspose of testing
// ---------------------------------------------------------

// mockStoredReqFetcher implements the Fetcher interface
type mockStoredReqFetcher struct {
}

func (cf mockStoredReqFetcher) FetchRequests(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage, errs []error) {
	return testStoredRequestData, testStoredImpData, nil
}

func (cf mockStoredReqFetcher) FetchResponses(ctx context.Context, ids []string) (data map[string]json.RawMessage, errs []error) {
	return nil, nil
}

// mockExchange implements the Exchange interface
type mockExchange struct {
	lastRequest *openrtb2.BidRequest
}

func (m *mockExchange) HoldAuction(ctx context.Context, r exchange.AuctionRequest, debugLog *exchange.DebugLog) (*openrtb2.BidResponse, error) {
	m.lastRequest = r.BidRequest
	return &openrtb2.BidResponse{
		SeatBid: []openrtb2.SeatBid{{
			Bid: []openrtb2.Bid{{
				AdM: "<script></script>",
			}},
		}},
	}, nil
}

// mockExchangeFPD implements the Exchange interface
type mockExchangeFPD struct {
	lastRequest    *openrtb2.BidRequest
	firstPartyData map[openrtb_ext.BidderName]*firstpartydata.ResolvedFirstPartyData
}

func (m *mockExchangeFPD) HoldAuction(ctx context.Context, r exchange.AuctionRequest, debugLog *exchange.DebugLog) (*openrtb2.BidResponse, error) {
	m.lastRequest = r.BidRequest
	m.firstPartyData = r.FirstPartyData
	return &openrtb2.BidResponse{}, nil
}

// hardcodedResponseIPValidator implements the IPValidator interface.
type hardcodedResponseIPValidator struct {
	response bool
}

func (v hardcodedResponseIPValidator) IsValid(net.IP, iputil.IPVersion) bool {
	return v.response
}

// fakeUUIDGenerator implements the UUIDGenerator interface
type fakeUUIDGenerator struct {
	id  string
	err error
}

func (f fakeUUIDGenerator) Generate() (string, error) {
	return f.id, f.err
}

// warningsCheckExchange is a well-behaved exchange which stores all incoming warnings.
// implements the Exchange interface
type warningsCheckExchange struct {
	auctionRequest exchange.AuctionRequest
}

func (e *warningsCheckExchange) HoldAuction(ctx context.Context, r exchange.AuctionRequest, debugLog *exchange.DebugLog) (*openrtb2.BidResponse, error) {
	e.auctionRequest = r
	return nil, nil
}

// nobidExchange is a well-behaved exchange which always bids "no bid".
// implements the Exchange interface
type nobidExchange struct {
	gotRequest *openrtb2.BidRequest
}

func (e *nobidExchange) HoldAuction(ctx context.Context, r exchange.AuctionRequest, debugLog *exchange.DebugLog) (*openrtb2.BidResponse, error) {
	e.gotRequest = r.BidRequest
	return &openrtb2.BidResponse{
		ID:    r.BidRequest.ID,
		BidID: "test bid id",
		NBR:   openrtb2.NoBidReasonCodeUnknownError.Ptr(),
	}, nil
}

// mockCurrencyRatesClient is a mock currency rate server and the rates it returns
// are set in the JSON test file
type mockCurrencyRatesClient struct {
	data currencyInfo
}

type currencyInfo struct {
	Conversions map[string]map[string]float64 `json:"conversions"`
	DataAsOfRaw string                        `json:"dataAsOf"`
}

func (s mockCurrencyRatesClient) handle(w http.ResponseWriter, req *http.Request) {
	s.data.DataAsOfRaw = "2018-09-12"

	// Marshal the response and http write it
	currencyServerJsonResponse, err := json.Marshal(&s.data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(currencyServerJsonResponse)
	return
}

// mockBidderHandler defines the handle function of a a mock bidder service. Its bidder
// name can be set upon instantiation and its bid price and bid currency can be set in the
// JSON test file itself
type mockBidderHandler struct {
	bidInfo    mockBidInfo
	bidderName string
}

func (b mockBidderHandler) bid(w http.ResponseWriter, req *http.Request) {
	// Read request Body
	buf := new(bytes.Buffer)
	buf.ReadFrom(req.Body)

	// Unmarshal exit if error
	var openrtb2Request openrtb2.BidRequest
	if err := json.Unmarshal(buf.Bytes(), &openrtb2Request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var openrtb2ImpExt map[string]json.RawMessage
	if err := json.Unmarshal(openrtb2Request.Imp[0].Ext, &openrtb2ImpExt); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, exists := openrtb2ImpExt["bidder"]
	if !exists {
		http.Error(w, "This request is not meant for this bidder", http.StatusBadRequest)
		return
	}

	// Values we need to build response
	currency := b.bidInfo.BidCurrency
	price := b.bidInfo.BidPrice

	// default values
	if len(currency) == 0 {
		currency = "USD"
	}

	// Create bid service openrtb2.BidResponse with one bid according to JSON test file values
	var serverResponseObject = openrtb2.BidResponse{
		ID:  openrtb2Request.ID,
		Cur: currency,
		SeatBid: []openrtb2.SeatBid{
			{
				Bid: []openrtb2.Bid{
					{
						ID:    b.bidderName + "-bid",
						ImpID: openrtb2Request.Imp[0].ID,
						Price: price,
					},
				},
				Seat: b.bidderName,
			},
		},
	}

	// Marshal the response and http write it
	serverJsonResponse, err := json.Marshal(&serverResponseObject)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(serverJsonResponse)
	return
}

// mockAdapter is a mock impression-splitting adapter
type mockAdapter struct {
	mockServerURL string
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	adapter := &mockAdapter{
		mockServerURL: config.Endpoint,
	}
	return adapter, nil
}

func (a mockAdapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var requests []*adapters.RequestData
	var errors []error

	requestCopy := *request
	for _, imp := range request.Imp {
		requestCopy.Imp = []openrtb2.Imp{imp}

		requestJSON, err := json.Marshal(request)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		requestData := &adapters.RequestData{
			Method: "POST",
			Uri:    a.mockServerURL,
			Body:   requestJSON,
		}
		requests = append(requests, requestData)
	}
	return requests, errors
}

func (a mockAdapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode != http.StatusOK {
		switch responseData.StatusCode {
		case http.StatusNoContent:
			return nil, nil
		case http.StatusBadRequest:
			return nil, []error{&errortypes.BadInput{
				Message: "Unexpected status code: 400. Bad request from publisher. Run with request.debug = 1 for more info.",
			}}
		default:
			return nil, []error{&errortypes.BadServerResponse{
				Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info.", responseData.StatusCode),
			}}
		}
	}

	var publisherResponse openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &publisherResponse); err != nil {
		return nil, []error{err}
	}

	rv := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	rv.Currency = publisherResponse.Cur
	for _, seatBid := range publisherResponse.SeatBid {
		for i, bid := range seatBid.Bid {
			for _, imp := range request.Imp {
				if imp.ID == bid.ImpID {
					b := &adapters.TypedBid{
						Bid:     &seatBid.Bid[i],
						BidType: openrtb_ext.BidTypeBanner,
					}
					rv.Bids = append(rv.Bids, b)
				}
			}
		}
	}
	return rv, nil
}

// ---------------------------------------------------------
// Auxiliary functions that don't make assertions and don't
// take t *testing.T as parameter
// ---------------------------------------------------------
func getBidderInfos(cfg map[string]config.Adapter, biddersNames []openrtb_ext.BidderName) config.BidderInfos {
	biddersInfos := make(config.BidderInfos)
	for _, name := range biddersNames {
		adapterConfig, ok := cfg[string(name)]
		if !ok {
			adapterConfig = config.Adapter{}
		}
		biddersInfos[string(name)] = newBidderInfo(adapterConfig)
	}
	return biddersInfos
}

func newBidderInfo(cfg config.Adapter) config.BidderInfo {
	return config.BidderInfo{
		Enabled: !cfg.Disabled,
	}
}

func getTestFiles(dir string) ([]string, error) {
	var filesToAssert []string

	fileList, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	// Append the path of every file found in `dir` to the `filesToAssert` array
	for _, fileInfo := range fileList {
		filesToAssert = append(filesToAssert, filepath.Join(dir, fileInfo.Name()))
	}

	return filesToAssert, nil
}

func parseTestFile(fileData []byte, testFile string) (testCase, error) {

	parsedTestData := testCase{}
	var err, errEm error

	// Get testCase values
	parsedTestData.BidRequest, _, _, err = jsonparser.Get(fileData, "mockBidRequest")
	if err != nil {
		return parsedTestData, fmt.Errorf("Error jsonparsing root.mockBidRequest from file %s. Desc: %v.", testFile, err)
	}

	// Get testCaseConfig values
	parsedTestData.Config = &testConfigValues{}
	var jsonTestConfig json.RawMessage

	jsonTestConfig, _, _, err = jsonparser.Get(fileData, "config")
	if err == nil {
		if err = json.Unmarshal(jsonTestConfig, parsedTestData.Config); err != nil {
			return parsedTestData, fmt.Errorf("Error unmarshaling root.config from file %s. Desc: %v.", testFile, err)
		}
	}

	// Get the return code we expect PBS to throw back given test's bidRequest and config
	parsedReturnCode, err := jsonparser.GetInt(fileData, "expectedReturnCode")
	if err != nil {
		return parsedTestData, fmt.Errorf("Error jsonparsing root.code from file %s. Desc: %v.", testFile, err)
	}

	// Get both bid response and error message, if any
	parsedTestData.ExpectedBidResponse, _, _, err = jsonparser.Get(fileData, "expectedBidResponse")
	parsedTestData.ExpectedErrorMessage, errEm = jsonparser.GetString(fileData, "expectedErrorMessage")

	if err == nil && errEm == nil {
		return parsedTestData, errors.New("Test case file can't have both a valid expectedBidResponse and a valid expectedErrorMessage, fields are mutually exclusive")
	} else if err != nil && errEm != nil {
		return parsedTestData, errors.New("Test case file should come with either a valid expectedBidResponse or a valid expectedErrorMessage, not both.")
	}

	parsedTestData.ExpectedReturnCode = int(parsedReturnCode)

	return parsedTestData, nil
}

func (tc *testConfigValues) getBlacklistedAppMap() map[string]bool {
	var blacklistedAppMap map[string]bool

	if len(tc.BlacklistedApps) > 0 {
		blacklistedAppMap = make(map[string]bool, len(tc.BlacklistedApps))
		for _, app := range tc.BlacklistedApps {
			blacklistedAppMap[app] = true
		}
	}
	return blacklistedAppMap
}

func (tc *testConfigValues) getBlackListedAccountMap() map[string]bool {
	var blacklistedAccountMap map[string]bool

	if len(tc.BlacklistedAccounts) > 0 {
		blacklistedAccountMap = make(map[string]bool, len(tc.BlacklistedAccounts))
		for _, account := range tc.BlacklistedAccounts {
			blacklistedAccountMap[account] = true
		}
	}
	return blacklistedAccountMap
}

func (tc *testConfigValues) getAdaptersConfigMap() map[string]config.Adapter {
	var adaptersConfig map[string]config.Adapter

	if len(tc.DisabledAdapters) > 0 {
		adaptersConfig = make(map[string]config.Adapter, len(tc.DisabledAdapters))
		for _, adapterName := range tc.DisabledAdapters {
			adaptersConfig[adapterName] = config.Adapter{Disabled: true}
		}
	}
	return adaptersConfig
}

func buildTestEndpoint(test testCase, paramValidator openrtb_ext.BidderParamValidator) (httprouter.Handle, *httptest.Server, *httptest.Server, *httptest.Server, *httptest.Server, error) {
	bidderInfos := getBidderInfos(test.Config.getAdaptersConfigMap(), openrtb_ext.CoreBidderNames())
	bidderMap := exchange.GetActiveBidders(bidderInfos)
	disabledBidders := exchange.GetDisabledBiddersErrorMessages(bidderInfos)

	// Adapter map with mock adapters needed to run JSON test cases
	adapterMap := make(map[openrtb_ext.BidderName]exchange.AdaptedBidder, 0)

	// AppNexus mock bid server and adapter
	appNexusBidder := mockBidderHandler{bidInfo: test.Config.MockBidder, bidderName: "appnexus"}
	appNexusServer := httptest.NewServer(http.HandlerFunc(appNexusBidder.bid))
	appNexusBidderAdapter := mockAdapter{mockServerURL: appNexusServer.URL}
	adapterMap[openrtb_ext.BidderAppnexus] = exchange.AdaptBidder(appNexusBidderAdapter, appNexusServer.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderAppnexus, nil)

	// openX mock bid server and adapter
	openXBidder := mockBidderHandler{bidInfo: test.Config.MockBidder, bidderName: "openx"}
	openXServer := httptest.NewServer(http.HandlerFunc(openXBidder.bid))
	openXBidderAdapter := mockAdapter{mockServerURL: openXServer.URL}
	adapterMap[openrtb_ext.BidderOpenx] = exchange.AdaptBidder(openXBidderAdapter, openXServer.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderOpenx, nil)

	// Rubicon mock bid server and adapter
	rubiconBidder := mockBidderHandler{bidInfo: test.Config.MockBidder, bidderName: "rubicon"}
	rubiconServer := httptest.NewServer(http.HandlerFunc(rubiconBidder.bid))
	rubiconBidderAdapter := mockAdapter{mockServerURL: rubiconServer.URL}
	adapterMap[openrtb_ext.BidderRubicon] = exchange.AdaptBidder(rubiconBidderAdapter, rubiconServer.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderRubicon, nil)

	// Mock prebid Server's currency converter, instantiate and start
	mockCurrencyConversionService := mockCurrencyRatesClient{
		currencyInfo{
			Conversions: test.Config.CurrencyRates,
		},
	}
	mockCurrencyRatesServer := httptest.NewServer(http.HandlerFunc(mockCurrencyConversionService.handle))
	//defer mockCurrencyRatesServer.Close()
	mockCurrencyConverter := currency.NewRateConverter(mockCurrencyRatesServer.Client(), mockCurrencyRatesServer.URL, time.Second)
	mockCurrencyConverter.Run()

	cfg := &config.Configuration{
		MaxRequestSize:     maxSize,
		BlacklistedApps:    test.Config.BlacklistedApps,
		BlacklistedAppMap:  test.Config.getBlacklistedAppMap(),
		BlacklistedAccts:   test.Config.BlacklistedAccounts,
		BlacklistedAcctMap: test.Config.getBlackListedAccountMap(),
		AccountRequired:    test.Config.AccountRequired,
	}

	met := &metricsConfig.NilMetricsEngine{}
	mockFetcher := empty_fetcher.EmptyFetcher{}

	ex := exchange.NewExchange(adapterMap,
		&wellBehavedCache{},
		cfg,
		nil,
		met,
		bidderInfos,
		gdpr.AlwaysAllow{},
		mockCurrencyConverter,
		mockFetcher)

	// Instantiate auction endpoint
	endpoint, err := NewEndpoint(
		fakeUUIDGenerator{},
		ex,
		paramValidator,
		&mockStoredReqFetcher{},
		mockFetcher,
		cfg,
		met,
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		disabledBidders,
		[]byte(test.Config.AliasJSON),
		bidderMap,
		empty_fetcher.EmptyFetcher{})

	return endpoint, appNexusServer, openXServer, rubiconServer, mockCurrencyRatesServer, err
}

type wellBehavedCache struct{}

func (c *wellBehavedCache) GetExtCacheData() (scheme string, host string, path string) {
	return "https", "www.pbcserver.com", "/pbcache/endpoint"
}

func (c *wellBehavedCache) PutJson(ctx context.Context, values []pbc.Cacheable) ([]string, []error) {
	ids := make([]string, len(values))
	for i := 0; i < len(values); i++ {
		ids[i] = strconv.Itoa(i)
	}
	return ids, nil
}

func readFile(t *testing.T, filename string) []byte {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", filename, err)
	}
	return data
}
