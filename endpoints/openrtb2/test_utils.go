package openrtb2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/buger/jsonparser"
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/openrtb/v19/openrtb3"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/analytics"
	analyticsConf "github.com/prebid/prebid-server/analytics/config"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/currency"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/exchange"
	"github.com/prebid/prebid-server/experiment/adscert"
	"github.com/prebid/prebid-server/gdpr"
	"github.com/prebid/prebid-server/hooks"
	"github.com/prebid/prebid-server/hooks/hookexecution"
	"github.com/prebid/prebid-server/hooks/hookstage"
	"github.com/prebid/prebid-server/metrics"
	metricsConfig "github.com/prebid/prebid-server/metrics/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	pbc "github.com/prebid/prebid-server/prebid_cache_client"
	"github.com/prebid/prebid-server/stored_requests"
	"github.com/prebid/prebid-server/stored_requests/backends/empty_fetcher"
	"github.com/prebid/prebid-server/util/iputil"
	"github.com/prebid/prebid-server/util/uuidutil"
	jsonpatch "gopkg.in/evanphx/json-patch.v4"
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

const (
	AMP_ENDPOINT = iota
	OPENRTB_ENDPOINT
	VIDEO_ENDPOINT
)

type testCase struct {
	// Common
	endpointType            int
	Description             string            `json:"description"`
	Config                  *testConfigValues `json:"config"`
	BidRequest              json.RawMessage   `json:"mockBidRequest"`
	ExpectedValidatedBidReq json.RawMessage   `json:"expectedValidatedBidRequest"`
	ExpectedReturnCode      int               `json:"expectedReturnCode,omitempty"`
	ExpectedErrorMessage    string            `json:"expectedErrorMessage"`
	Query                   string            `json:"query"`
	planBuilder             hooks.ExecutionPlanBuilder

	// "/openrtb2/auction" endpoint JSON test info
	ExpectedBidResponse json.RawMessage `json:"expectedBidResponse"`

	// "/openrtb2/amp" endpoint JSON test info
	storedRequest       map[string]json.RawMessage `json:"mockAmpStoredRequest"`
	StoredResponse      map[string]json.RawMessage `json:"mockAmpStoredResponse"`
	ExpectedAmpResponse json.RawMessage            `json:"expectedAmpResponse"`
}

type testConfigValues struct {
	AccountRequired     bool                          `json:"accountRequired"`
	AliasJSON           string                        `json:"aliases"`
	BlacklistedAccounts []string                      `json:"blacklistedAccts"`
	BlacklistedApps     []string                      `json:"blacklistedApps"`
	DisabledAdapters    []string                      `json:"disabledAdapters"`
	CurrencyRates       map[string]map[string]float64 `json:"currencyRates"`
	MockBidders         []mockBidderHandler           `json:"mockBidders"`
	RealParamsValidator bool                          `json:"realParamsValidator"`
}

type brokenExchange struct{}

func (e *brokenExchange) HoldAuction(ctx context.Context, r *exchange.AuctionRequest, debugLog *exchange.DebugLog) (*openrtb2.BidResponse, error) {
	return nil, errors.New("Critical, unrecoverable error.")
}

// Stored Requests
var testStoredRequestData = map[string]json.RawMessage{
	// Valid JSON
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
	// Invalid JSON because it comes with an extra closing curly brace '}'
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
	// Valid JSON
	"4": json.RawMessage(`{"id": "ThisID", "cur": ["USD"]}`),

	// Stored Request with Root Ext Passthrough
	"5": json.RawMessage(`{
		"ext": {
			"prebid": {
				"passthrough": {
					"root_ext_passthrough": 20
				}
			}
		}
	}`),
}

// Stored Imp Requests
var testStoredImpData = map[string]json.RawMessage{
	// Has valid JSON and matches schema
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
	// Has valid JSON, matches schema but is missing video object
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
	// Invalid JSON, is missing a coma after the rubicon's "accountId" field
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
	// Valid JSON. Missing video object
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
	// Valid JSON. Missing video object
	"10": json.RawMessage(`{
			"ext": {
				"appnexus": {
					"placementId": 12345678,
					"position": "above",
					"reserve": 0.35
				}
			}
		}`),
	// Stored Imp with Passthrough
	"6": json.RawMessage(`{
		"id": "my-imp-id",
		"ext": {
			"prebid": {
				"passthrough": {
					"imp_passthrough": 30
				}
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
	`{
		"id": "ThisID",
		"imp": [
			{
				"id": "my-imp-id",
				"video":{
					"h":300,
					"w":200
				},
				"ext": {
					"prebid": {
						"storedrequest": {
							"id": "6"
						}
					}
				}
			}
		],
		"ext": {
			"prebid": {
				"storedrequest": {
					"id": "5"
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
	}`,
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
	`{
	"id": "ThisID",
	"imp": [
		{
			"ext":{
			   "prebid":{
				  "passthrough":{
					 "imp_passthrough":30
				  },
				  "storedrequest":{
					 "id":"6"
				  }
			   }
			},
			"id":"my-imp-id",
			"video":{
			   "h":300,
			   "w":200
			}
		 }
	],
	"ext":{
		"prebid":{
		   "passthrough":{
			  "root_ext_passthrough":20
		   },
		   "storedrequest":{
			  "id":"5"
		   }
		}
	 }
}`,
}

var testStoredImpIds = []string{
	"adUnit1", "adUnit2", "adUnit1", "some-static-imp", "my-imp-id",
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
	`{
		"id": "my-imp-id",
		"ext": {
			"prebid": {
				"passthrough": {
					"imp_passthrough": 30
				}
			}
		}
	}`,
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
				"storedrequest": {
					"id": "1"
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
		"site": {
			"page": "prebid.org"
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

func (m *mockExchange) HoldAuction(ctx context.Context, auctionRequest *exchange.AuctionRequest, debugLog *exchange.DebugLog) (*openrtb2.BidResponse, error) {
	r := auctionRequest.BidRequestWrapper
	m.lastRequest = r.BidRequest
	return &openrtb2.BidResponse{
		SeatBid: []openrtb2.SeatBid{{
			Bid: []openrtb2.Bid{{
				AdM: "<script></script>",
			}},
		}},
	}, nil
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

func (e *warningsCheckExchange) HoldAuction(ctx context.Context, r *exchange.AuctionRequest, debugLog *exchange.DebugLog) (*openrtb2.BidResponse, error) {
	e.auctionRequest = *r
	return nil, nil
}

// nobidExchange is a well-behaved exchange which always bids "no bid".
// implements the Exchange interface
type nobidExchange struct {
	gotRequest *openrtb2.BidRequest
}

func (e *nobidExchange) HoldAuction(ctx context.Context, auctionRequest *exchange.AuctionRequest, debugLog *exchange.DebugLog) (*openrtb2.BidResponse, error) {
	r := auctionRequest.BidRequestWrapper
	e.gotRequest = r.BidRequest
	return &openrtb2.BidResponse{
		ID:    r.BidRequest.ID,
		BidID: "test bid id",
		NBR:   openrtb3.NoBidUnknownError.Ptr(),
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

// mockBidderHandler carries mock bidder server information that will be read from JSON test files
// and defines a handle function of a a mock bidder service.
type mockBidderHandler struct {
	BidderName string  `json:"bidderName"`
	Currency   string  `json:"currency"`
	Price      float64 `json:"price"`
	DealID     string  `json:"dealid"`
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

	// Create bid service openrtb2.BidResponse with one bid according to JSON test file values
	var serverResponseObject = openrtb2.BidResponse{
		ID:  openrtb2Request.ID,
		Cur: b.Currency,
		SeatBid: []openrtb2.SeatBid{
			{
				Bid: []openrtb2.Bid{
					{
						ID:     b.BidderName + "-bid",
						ImpID:  openrtb2Request.Imp[0].ID,
						Price:  b.Price,
						DealID: b.DealID,
					},
				},
				Seat: b.BidderName,
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
	Server        config.Server
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	adapter := &mockAdapter{
		mockServerURL: config.Endpoint,
		Server:        server,
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
func getBidderInfos(disabledAdapters []string, biddersNames []openrtb_ext.BidderName) config.BidderInfos {
	biddersInfos := make(config.BidderInfos)
	for _, name := range biddersNames {
		isDisabled := false
		for _, disabledAdapter := range disabledAdapters {
			if string(name) == disabledAdapter {
				isDisabled = true
				break
			}
		}
		biddersInfos[string(name)] = newBidderInfo(isDisabled)
	}
	return biddersInfos
}

func newBidderInfo(isDisabled bool) config.BidderInfo {
	return config.BidderInfo{
		Disabled: isDisabled,
	}
}

func parseTestData(fileData []byte, testFile string) (testCase, error) {

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
		return parsedTestData, fmt.Errorf("Test case %s can't have both a valid expectedBidResponse and a valid expectedErrorMessage, fields are mutually exclusive", testFile)
	} else if err != nil && errEm != nil {
		return parsedTestData, fmt.Errorf("Test case %s should come with either a valid expectedBidResponse or a valid expectedErrorMessage, not both.", testFile)
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

// exchangeTestWrapper is a wrapper that asserts the openrtb2 bid request just before the HoldAuction call
type exchangeTestWrapper struct {
	ex                    exchange.Exchange
	actualValidatedBidReq *openrtb2.BidRequest
}

func (te *exchangeTestWrapper) HoldAuction(ctx context.Context, r *exchange.AuctionRequest, debugLog *exchange.DebugLog) (*openrtb2.BidResponse, error) {

	// rebuild/resync the request in the request wrapper.
	if err := r.BidRequestWrapper.RebuildRequest(); err != nil {
		return nil, err
	}

	// Save the validated bidRequest that we are about to feed HoldAuction
	te.actualValidatedBidReq = r.BidRequestWrapper.BidRequest

	// Call HoldAuction() implementation as written in the exchange package
	return te.ex.HoldAuction(ctx, r, debugLog)
}

// buildTestExchange returns an exchange with mock bidder servers and mock currency conversion server
func buildTestExchange(testCfg *testConfigValues, adapterMap map[openrtb_ext.BidderName]exchange.AdaptedBidder, mockBidServersArray []*httptest.Server, mockCurrencyRatesServer *httptest.Server, bidderInfos config.BidderInfos, cfg *config.Configuration, met metrics.MetricsEngine, mockFetcher stored_requests.CategoryFetcher) (exchange.Exchange, []*httptest.Server) {
	if len(testCfg.MockBidders) == 0 {
		testCfg.MockBidders = append(testCfg.MockBidders, mockBidderHandler{BidderName: "appnexus", Currency: "USD", Price: 0.00})
	}
	for _, mockBidder := range testCfg.MockBidders {
		bidServer := httptest.NewServer(http.HandlerFunc(mockBidder.bid))
		bidderAdapter := mockAdapter{mockServerURL: bidServer.URL}
		bidderName := openrtb_ext.BidderName(mockBidder.BidderName)

		adapterMap[bidderName] = exchange.AdaptBidder(bidderAdapter, bidServer.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, bidderName, nil, "")
		mockBidServersArray = append(mockBidServersArray, bidServer)
	}

	mockCurrencyConverter := currency.NewRateConverter(mockCurrencyRatesServer.Client(), mockCurrencyRatesServer.URL, time.Second)
	mockCurrencyConverter.Run()

	gdprPermsBuilder := fakePermissionsBuilder{
		permissions: &fakePermissions{},
	}.Builder

	testExchange := exchange.NewExchange(adapterMap,
		&wellBehavedCache{},
		cfg,
		nil,
		met,
		bidderInfos,
		gdprPermsBuilder,
		mockCurrencyConverter,
		mockFetcher,
		&adscert.NilSigner{},
	)

	testExchange = &exchangeTestWrapper{
		ex: testExchange,
	}

	return testExchange, mockBidServersArray
}

// buildTestEndpoint instantiates an openrtb2 Auction endpoint designed to test endpoints/openrtb2/auction.go
func buildTestEndpoint(test testCase, cfg *config.Configuration) (httprouter.Handle, *exchangeTestWrapper, []*httptest.Server, *httptest.Server, error) {
	if test.Config == nil {
		test.Config = &testConfigValues{}
	}

	var paramValidator openrtb_ext.BidderParamValidator
	if test.Config.RealParamsValidator {
		var err error
		paramValidator, err = openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
		if err != nil {
			return nil, nil, nil, nil, err
		}
	} else {
		paramValidator = mockBidderParamValidator{}
	}

	bidderInfos := getBidderInfos(test.Config.DisabledAdapters, openrtb_ext.CoreBidderNames())
	bidderMap := exchange.GetActiveBidders(bidderInfos)
	disabledBidders := exchange.GetDisabledBiddersErrorMessages(bidderInfos)
	met := &metricsConfig.NilMetricsEngine{}
	mockFetcher := empty_fetcher.EmptyFetcher{}

	// Adapter map with mock adapters needed to run JSON test cases
	adapterMap := make(map[openrtb_ext.BidderName]exchange.AdaptedBidder, 0)
	mockBidServersArray := make([]*httptest.Server, 0, 3)

	// Mock prebid Server's currency converter, instantiate and start
	mockCurrencyConversionService := mockCurrencyRatesClient{
		currencyInfo{
			Conversions: test.Config.CurrencyRates,
		},
	}
	mockCurrencyRatesServer := httptest.NewServer(http.HandlerFunc(mockCurrencyConversionService.handle))

	testExchange, mockBidServersArray := buildTestExchange(test.Config, adapterMap, mockBidServersArray, mockCurrencyRatesServer, bidderInfos, cfg, met, mockFetcher)

	var storedRequestFetcher stored_requests.Fetcher
	if len(test.storedRequest) > 0 {
		storedRequestFetcher = &mockAmpStoredReqFetcher{test.storedRequest}
	} else {
		storedRequestFetcher = &mockStoredReqFetcher{}
	}

	var storedResponseFetcher stored_requests.Fetcher
	if len(test.StoredResponse) > 0 {
		storedResponseFetcher = &mockAmpStoredResponseFetcher{test.StoredResponse}
	} else {
		storedResponseFetcher = empty_fetcher.EmptyFetcher{}
	}

	var accountFetcher stored_requests.AccountFetcher
	accountFetcher = &mockAccountFetcher{
		data: map[string]json.RawMessage{
			"malformed_acct": json.RawMessage(`{"disabled":"invalid type"}`),
		},
	}

	planBuilder := test.planBuilder
	if planBuilder == nil {
		planBuilder = hooks.EmptyPlanBuilder{}
	}

	var endpointBuilder func(uuidutil.UUIDGenerator, exchange.Exchange, openrtb_ext.BidderParamValidator, stored_requests.Fetcher, stored_requests.AccountFetcher, *config.Configuration, metrics.MetricsEngine, analytics.PBSAnalyticsModule, map[string]string, []byte, map[string]openrtb_ext.BidderName, stored_requests.Fetcher, hooks.ExecutionPlanBuilder) (httprouter.Handle, error)

	switch test.endpointType {
	case AMP_ENDPOINT:
		endpointBuilder = NewAmpEndpoint
	default: //case OPENRTB_ENDPOINT:
		endpointBuilder = NewEndpoint
	}

	endpoint, err := endpointBuilder(
		fakeUUIDGenerator{},
		testExchange,
		paramValidator,
		storedRequestFetcher,
		accountFetcher,
		cfg,
		met,
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		disabledBidders,
		[]byte(test.Config.AliasJSON),
		bidderMap,
		storedResponseFetcher,
		planBuilder,
	)

	return endpoint, testExchange.(*exchangeTestWrapper), mockBidServersArray, mockCurrencyRatesServer, err
}

type mockBidderParamValidator struct{}

func (v mockBidderParamValidator) Validate(name openrtb_ext.BidderName, ext json.RawMessage) error {
	return nil
}
func (v mockBidderParamValidator) Schema(name openrtb_ext.BidderName) string { return "" }

type mockAccountFetcher struct {
	data map[string]json.RawMessage
}

func (af *mockAccountFetcher) FetchAccount(ctx context.Context, defaultAccountJSON json.RawMessage, accountID string) (json.RawMessage, []error) {
	if account, ok := af.data[accountID]; ok {
		return account, nil
	}
	return nil, []error{stored_requests.NotFoundError{ID: accountID, DataType: "Account"}}
}

type mockAmpStoredReqFetcher struct {
	data map[string]json.RawMessage
}

func (cf *mockAmpStoredReqFetcher) FetchRequests(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage, errs []error) {
	return cf.data, nil, nil
}

func (cf *mockAmpStoredReqFetcher) FetchResponses(ctx context.Context, ids []string) (data map[string]json.RawMessage, errs []error) {
	return nil, nil
}

type mockAmpStoredResponseFetcher struct {
	data map[string]json.RawMessage
}

func (cf *mockAmpStoredResponseFetcher) FetchRequests(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage, errs []error) {
	return nil, nil, nil
}

func (cf *mockAmpStoredResponseFetcher) FetchResponses(ctx context.Context, ids []string) (data map[string]json.RawMessage, errs []error) {
	for _, storedResponseID := range ids {
		if storedResponse, exists := cf.data[storedResponseID]; exists {
			// Found. Unescape string before returning
			response, err := strconv.Unquote(string(storedResponse))
			if err != nil {
				return nil, append([]error{}, err)
			}
			cf.data[storedResponseID] = json.RawMessage(response)
			return cf.data, nil
		}
	}
	return nil, nil
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
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", filename, err)
	}
	return data
}

type fakePermissionsBuilder struct {
	permissions gdpr.Permissions
}

func (fpb fakePermissionsBuilder) Builder(gdpr.TCF2ConfigReader, gdpr.RequestInfo) gdpr.Permissions {
	return fpb.permissions
}

type fakePermissions struct {
}

func (p *fakePermissions) HostCookiesAllowed(ctx context.Context) (bool, error) {
	return true, nil
}

func (p *fakePermissions) BidderSyncAllowed(ctx context.Context, bidder openrtb_ext.BidderName) (bool, error) {
	return true, nil
}

func (p *fakePermissions) AuctionActivitiesAllowed(ctx context.Context, bidderCoreName openrtb_ext.BidderName, bidder openrtb_ext.BidderName) (permissions gdpr.AuctionPermissions, err error) {
	return gdpr.AuctionPermissions{
		AllowBidRequest: true,
	}, nil
}

type mockPlanBuilder struct {
	entrypointPlan               hooks.Plan[hookstage.Entrypoint]
	rawAuctionPlan               hooks.Plan[hookstage.RawAuctionRequest]
	processedAuctionPlan         hooks.Plan[hookstage.ProcessedAuctionRequest]
	bidderRequestPlan            hooks.Plan[hookstage.BidderRequest]
	rawBidderResponsePlan        hooks.Plan[hookstage.RawBidderResponse]
	allProcessedBidResponsesPlan hooks.Plan[hookstage.AllProcessedBidResponses]
	auctionResponsePlan          hooks.Plan[hookstage.AuctionResponse]
}

func (m mockPlanBuilder) PlanForEntrypointStage(_ string) hooks.Plan[hookstage.Entrypoint] {
	return m.entrypointPlan
}

func (m mockPlanBuilder) PlanForRawAuctionStage(_ string, _ *config.Account) hooks.Plan[hookstage.RawAuctionRequest] {
	return m.rawAuctionPlan
}

func (m mockPlanBuilder) PlanForProcessedAuctionStage(_ string, _ *config.Account) hooks.Plan[hookstage.ProcessedAuctionRequest] {
	return m.processedAuctionPlan
}

func (m mockPlanBuilder) PlanForBidderRequestStage(_ string, _ *config.Account) hooks.Plan[hookstage.BidderRequest] {
	return m.bidderRequestPlan
}

func (m mockPlanBuilder) PlanForRawBidderResponseStage(_ string, _ *config.Account) hooks.Plan[hookstage.RawBidderResponse] {
	return m.rawBidderResponsePlan
}

func (m mockPlanBuilder) PlanForAllProcessedBidResponsesStage(_ string, _ *config.Account) hooks.Plan[hookstage.AllProcessedBidResponses] {
	return m.allProcessedBidResponsesPlan
}

func (m mockPlanBuilder) PlanForAuctionResponseStage(_ string, _ *config.Account) hooks.Plan[hookstage.AuctionResponse] {
	return m.auctionResponsePlan
}

func makePlan[H any](hook H) hooks.Plan[H] {
	return hooks.Plan[H]{
		{
			Timeout: 5 * time.Millisecond,
			Hooks: []hooks.HookWrapper[H]{
				{
					Module: "foobar",
					Code:   "foo",
					Hook:   hook,
				},
			},
		},
	}
}

type mockRejectionHook struct {
	nbr int
}

func (m mockRejectionHook) HandleEntrypointHook(
	_ context.Context,
	_ hookstage.ModuleInvocationContext,
	_ hookstage.EntrypointPayload,
) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	return hookstage.HookResult[hookstage.EntrypointPayload]{Reject: true, NbrCode: m.nbr}, nil
}

func (m mockRejectionHook) HandleRawAuctionHook(
	_ context.Context,
	_ hookstage.ModuleInvocationContext,
	_ hookstage.RawAuctionRequestPayload,
) (hookstage.HookResult[hookstage.RawAuctionRequestPayload], error) {
	return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{Reject: true, NbrCode: m.nbr}, nil
}

func (m mockRejectionHook) HandleProcessedAuctionHook(
	_ context.Context,
	_ hookstage.ModuleInvocationContext,
	_ hookstage.ProcessedAuctionRequestPayload,
) (hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload], error) {
	return hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload]{Reject: true, NbrCode: m.nbr}, nil
}

func (m mockRejectionHook) HandleBidderRequestHook(
	_ context.Context,
	_ hookstage.ModuleInvocationContext,
	payload hookstage.BidderRequestPayload,
) (hookstage.HookResult[hookstage.BidderRequestPayload], error) {
	result := hookstage.HookResult[hookstage.BidderRequestPayload]{}
	if payload.Bidder == "appnexus" {
		result.Reject = true
		result.NbrCode = m.nbr
	}

	return result, nil
}

func (m mockRejectionHook) HandleRawBidderResponseHook(
	_ context.Context,
	_ hookstage.ModuleInvocationContext,
	payload hookstage.RawBidderResponsePayload,
) (hookstage.HookResult[hookstage.RawBidderResponsePayload], error) {
	result := hookstage.HookResult[hookstage.RawBidderResponsePayload]{}
	if payload.Bidder == "appnexus" {
		result.Reject = true
		result.NbrCode = m.nbr
	}

	return result, nil
}

var entryPointHookUpdateWithErrors = hooks.HookWrapper[hookstage.Entrypoint]{
	Module: "foobar",
	Code:   "foo",
	Hook: mockUpdateHook{
		entrypointHandler: func(
			_ hookstage.ModuleInvocationContext,
			payload hookstage.EntrypointPayload,
		) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
			ch := hookstage.ChangeSet[hookstage.EntrypointPayload]{}
			ch.AddMutation(func(payload hookstage.EntrypointPayload) (hookstage.EntrypointPayload, error) {
				payload.Request.Header.Add("foo", "bar")
				return payload, nil
			}, hookstage.MutationUpdate, "header", "foo")

			return hookstage.HookResult[hookstage.EntrypointPayload]{
				ChangeSet: ch,
				Errors:    []string{"error 1"},
			}, nil
		},
	},
}

var entryPointHookUpdateWithErrorsAndWarnings = hooks.HookWrapper[hookstage.Entrypoint]{
	Module: "foobar",
	Code:   "bar",
	Hook: mockUpdateHook{
		entrypointHandler: func(
			_ hookstage.ModuleInvocationContext,
			payload hookstage.EntrypointPayload,
		) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
			ch := hookstage.ChangeSet[hookstage.EntrypointPayload]{}
			ch.AddMutation(func(payload hookstage.EntrypointPayload) (hookstage.EntrypointPayload, error) {
				params := payload.Request.URL.Query()
				params.Add("foo", "baz")
				payload.Request.URL.RawQuery = params.Encode()
				return payload, nil
			}, hookstage.MutationUpdate, "param", "foo")

			return hookstage.HookResult[hookstage.EntrypointPayload]{
				ChangeSet: ch,
				Errors:    []string{"error 1"},
				Warnings:  []string{"warning 1"},
			}, nil
		},
	},
}

var entryPointHookUpdate = hooks.HookWrapper[hookstage.Entrypoint]{
	Module: "foobar",
	Code:   "baz",
	Hook: mockUpdateHook{
		entrypointHandler: func(
			ctx hookstage.ModuleInvocationContext,
			payload hookstage.EntrypointPayload,
		) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
			result := hookstage.HookResult[hookstage.EntrypointPayload]{}
			if ctx.Endpoint != hookexecution.EndpointAuction {
				result.Warnings = []string{fmt.Sprintf("Endpoint %s is not supported by hook.", ctx.Endpoint)}
				return result, nil
			}

			ch := hookstage.ChangeSet[hookstage.EntrypointPayload]{}
			ch.AddMutation(func(payload hookstage.EntrypointPayload) (hookstage.EntrypointPayload, error) {
				body, err := jsonpatch.MergePatch(payload.Body, []byte(`{"tmax":50}`))
				if err == nil {
					payload.Body = body
				}
				return payload, err
			}, hookstage.MutationUpdate, "body", "tmax")
			ch.AddMutation(func(payload hookstage.EntrypointPayload) (hookstage.EntrypointPayload, error) {
				body, err := jsonpatch.MergePatch(payload.Body, []byte(`{"regs": {"ext": {"gdpr": 1, "us_privacy": "1NYN"}}}`))
				if err == nil {
					payload.Body = body
				}
				return payload, err
			}, hookstage.MutationAdd, "body", "regs", "ext", "us_privacy")
			result.ChangeSet = ch

			return result, nil
		},
	},
}

var rawAuctionHookNone = hooks.HookWrapper[hookstage.RawAuctionRequest]{
	Module: "vendor.module",
	Code:   "foobar",
	Hook:   mockUpdateHook{},
}

type mockUpdateHook struct {
	entrypointHandler func(
		hookstage.ModuleInvocationContext,
		hookstage.EntrypointPayload,
	) (hookstage.HookResult[hookstage.EntrypointPayload], error)
}

func (m mockUpdateHook) HandleEntrypointHook(
	_ context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.EntrypointPayload,
) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	return m.entrypointHandler(miCtx, payload)
}

func (m mockUpdateHook) HandleRawAuctionHook(
	_ context.Context,
	_ hookstage.ModuleInvocationContext,
	_ hookstage.RawAuctionRequestPayload,
) (hookstage.HookResult[hookstage.RawAuctionRequestPayload], error) {
	return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{}, nil
}
