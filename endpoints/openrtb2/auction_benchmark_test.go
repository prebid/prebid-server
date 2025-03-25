package openrtb2

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	analyticsBuild "github.com/prebid/prebid-server/v3/analytics/build"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/currency"
	"github.com/prebid/prebid-server/v3/exchange"
	"github.com/prebid/prebid-server/v3/experiment/adscert"
	"github.com/prebid/prebid-server/v3/hooks"
	"github.com/prebid/prebid-server/v3/macros"
	metricsConfig "github.com/prebid/prebid-server/v3/metrics/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/ortb"
	"github.com/prebid/prebid-server/v3/stored_requests/backends/empty_fetcher"
	"github.com/prebid/prebid-server/v3/usersync"
)

// benchmarkTestServer returns the header bidding test ad. This response was scraped from a real appnexus server response.
func benchmarkTestServer(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`{"id":"some-request-id","seatbid":[{"bid":[{"id":"4625436751433509010","impid":"my-imp-id","price":0.5,"adm":"\u003cscript type=\"application/javascript\" src=\"http://nym1-ib.adnxs.com/ab?e=wqT_3QKABqAAAwAAAwDWAAUBCM-OiNAFELuV09Pqi86EVRj6t-7QyLin_REqLQkAAAECCOA_EQEHNAAA4D8ZAAAAgOtR4D8hERIAKREJoDDy5vwEOL4HQL4HSAJQ1suTDljhgEhgAGiRQHixhQSAAQGKAQNVU0SSBQbwUpgBrAKgAfoBqAEBsAEAuAECwAEDyAEC0AEA2AEA4AEB8AEAigI6dWYoJ2EnLCA0OTQ0NzIsIDE1MTAwODIzODMpO3VmKCdyJywgMjk2ODExMTAsMh4A8JySAvkBIVR6WGNkQWk2MEljRUVOYkxrdzRZQUNEaGdFZ3dBRGdBUUFSSXZnZFE4dWI4QkZnQVlQX19fXzhQYUFCd0FYZ0JnQUVCaUFFQmtBRUJtQUVCb0FFQnFBRURzQUVBdVFFcGk0aURBQURnUDhFQktZdUlnd0FBNERfSkFTZlJKRUdtbi00XzJRRUFBQUFBQUFEd1AtQUJBUFVCBQ8oSmdDQUtBQ0FMVUMFEARMMAkI8ExNQUNBY2dDQWRBQ0FkZ0NBZUFDQU9nQ0FQZ0NBSUFEQVpBREFKZ0RBYWdEdXRDSEJMb0RDVTVaVFRJNk16STNOdy4umgItITh3aENuZzb8ALg0WUJJSUFRb0FEb0pUbGxOTWpvek1qYzPYAugH4ALH0wHyAhAKBkFEVl9JRBIGNCV1HPICEQoGQ1BHARMcBzE5Nzc5MzMBJwgFQ1AFE_B-ODUxMzU5NIADAYgDAZADAJgDFKADAaoDAMADrALIAwDYAwDgAwDoAwD4AwCABACSBAkvb3BlbnJ0YjKYBACoBACyBAwIABAAGAAgADAAOAC4BADABADIBADSBAlOWU0yOjMyNzfaBAIIAeAEAPAE1suTDogFAZgFAKAF_____wUDXAGqBQ9zb21lLXJlcXVlc3QtaWTABQDJBUmbTPA_0gUJCQAAAAAAAAAA2AUB4AUB\u0026s=61dc0e8770543def5a3a77b4589830d1274b26f1\u0026test=1\u0026pp=${AUCTION_PRICE}\u0026\"\u003e\u003c/script\u003e","adid":"29681110","adomain":["appnexus.com"],"iurl":"http://nym1-ib.adnxs.com/cr?id=29681110","cid":"958","crid":"29681110","w":300,"h":250,"ext":{"bidder":{"appnexus":{"brand_id":1,"auction_id":6127490747252132539,"bidder_id":2}}}}],"seat":"appnexus"}],"ext":{"debug":{"httpcalls":{"appnexus":[{"uri":"http://ib.adnxs.com/openrtb2","requestbody":"{\"id\":\"some-request-id\",\"imp\":[{\"id\":\"my-imp-id\",\"banner\":{\"format\":[{\"w\":300,\"h\":250},{\"w\":300,\"h\":600}]},\"ext\":{\"appnexus\":{\"placement_id\":12883451}}}],\"test\":1,\"tmax\":500}","responsebody":"{\"id\":\"some-request-id\",\"seatbid\":[{\"bid\":[{\"id\":\"4625436751433509010\",\"impid\":\"my-imp-id\",\"price\": 0.500000,\"adid\":\"29681110\",\"adm\":\"\u003cscript type=\\\"application/javascript\\\" src=\\\"http://nym1-ib.adnxs.com/ab?e=wqT_3QKABqAAAwAAAwDWAAUBCM-OiNAFELuV09Pqi86EVRj6t-7QyLin_REqLQkAAAECCOA_EQEHNAAA4D8ZAAAAgOtR4D8hERIAKREJoDDy5vwEOL4HQL4HSAJQ1suTDljhgEhgAGiRQHixhQSAAQGKAQNVU0SSBQbwUpgBrAKgAfoBqAEBsAEAuAECwAEDyAEC0AEA2AEA4AEB8AEAigI6dWYoJ2EnLCA0OTQ0NzIsIDE1MTAwODIzODMpO3VmKCdyJywgMjk2ODExMTAsMh4A8JySAvkBIVR6WGNkQWk2MEljRUVOYkxrdzRZQUNEaGdFZ3dBRGdBUUFSSXZnZFE4dWI4QkZnQVlQX19fXzhQYUFCd0FYZ0JnQUVCaUFFQmtBRUJtQUVCb0FFQnFBRURzQUVBdVFFcGk0aURBQURnUDhFQktZdUlnd0FBNERfSkFTZlJKRUdtbi00XzJRRUFBQUFBQUFEd1AtQUJBUFVCBQ8oSmdDQUtBQ0FMVUMFEARMMAkI8ExNQUNBY2dDQWRBQ0FkZ0NBZUFDQU9nQ0FQZ0NBSUFEQVpBREFKZ0RBYWdEdXRDSEJMb0RDVTVaVFRJNk16STNOdy4umgItITh3aENuZzb8ALg0WUJJSUFRb0FEb0pUbGxOTWpvek1qYzPYAugH4ALH0wHyAhAKBkFEVl9JRBIGNCV1HPICEQoGQ1BHARMcBzE5Nzc5MzMBJwgFQ1AFE_B-ODUxMzU5NIADAYgDAZADAJgDFKADAaoDAMADrALIAwDYAwDgAwDoAwD4AwCABACSBAkvb3BlbnJ0YjKYBACoBACyBAwIABAAGAAgADAAOAC4BADABADIBADSBAlOWU0yOjMyNzfaBAIIAeAEAPAE1suTDogFAZgFAKAF_____wUDXAGqBQ9zb21lLXJlcXVlc3QtaWTABQDJBUmbTPA_0gUJCQAAAAAAAAAA2AUB4AUB\u0026s=61dc0e8770543def5a3a77b4589830d1274b26f1\u0026test=1\u0026pp=${AUCTION_PRICE}\u0026\\\"\u003e\u003c/script\u003e\",\"adomain\":[\"appnexus.com\"],\"iurl\":\"http://nym1-ib.adnxs.com/cr?id=29681110\",\"cid\":\"958\",\"crid\":\"29681110\",\"h\": 250,\"w\": 300,\"ext\":{\"appnexus\":{\"brand_id\": 1,\"auction_id\": 6127490747252132539,\"bidder_id\": 2}}}],\"seat\":\"958\"}],\"bidid\":\"8271358638249766712\",\"cur\":\"USD\"}","status":200}]}},"responsetimemillis":{"appnexus":42}}}`))
}

// benchmarkBuildTestRequest returns a request which fetches the header bidding test ad.
func benchmarkBuildTestRequest() *http.Request {
	request, _ := http.NewRequest("POST", "/openrtb2/auction", strings.NewReader(`{
  "id": "some-request-id",
  "imp": [
    {
      "id": "my-imp-id",
      "banner": {
    	"format": [
    	  {
    	    "w": 300,
    	    "h": 250
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
    }
  ],
  "test": 1,
  "tmax": 500
}`))
	return request
}

// BenchmarkOpenrtbEndpoint measures the performance of the endpoint, mocking out the external server dependency.
func BenchmarkOpenrtbEndpoint(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(benchmarkTestServer))
	defer server.Close()

	var infos = make(config.BidderInfos, 0)
	infos["appnexus"] = config.BidderInfo{Capabilities: &config.CapabilitiesInfo{Site: &config.PlatformInfo{MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner}}}}
	paramValidator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		return
	}
	requestValidator := ortb.NewRequestValidator(nil, map[string]string{}, paramValidator)

	nilMetrics := &metricsConfig.NilMetricsEngine{}

	adapters, singleFormatBidders, adaptersErr := exchange.BuildAdapters(server.Client(), &config.Configuration{}, infos, nilMetrics)
	if adaptersErr != nil {
		b.Fatal("unable to build adapters")
	}

	gdprPermsBuilder := fakePermissionsBuilder{
		permissions: &fakePermissions{},
	}.Builder

	exchange := exchange.NewExchange(
		adapters,
		nil,
		&config.Configuration{},
		requestValidator,
		map[string]usersync.Syncer{},
		nilMetrics,
		infos,
		gdprPermsBuilder,
		currency.NewRateConverter(&http.Client{}, "", time.Duration(0)),
		empty_fetcher.EmptyFetcher{},
		&adscert.NilSigner{},
		macros.NewStringIndexBasedReplacer(),
		nil,
		singleFormatBidders,
	)

	endpoint, _ := NewEndpoint(
		fakeUUIDGenerator{},
		exchange,
		requestValidator,
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		nilMetrics,
		analyticsBuild.New(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		nil,
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
		nil,
	)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		endpoint(httptest.NewRecorder(), benchmarkBuildTestRequest(), nil)
	}
}

// BenchmarkValidWholeExemplary benchmarks the process that results of hitting the `openrtb2/auction` with
// the different JSON bid requests found in the `sample-requests/valid-whole/exemplary/` directory. As
// especified in said file, we expect this bid request to succeed with a 200 status code.
func BenchmarkValidWholeExemplary(b *testing.B) {
	var benchInput = []string{
		"sample-requests/valid-whole/exemplary/all-ext.json",
		"sample-requests/valid-whole/exemplary/interstitial-no-size.json",
		"sample-requests/valid-whole/exemplary/prebid-test-ad.json",
		"sample-requests/valid-whole/exemplary/simple.json",
		"sample-requests/valid-whole/exemplary/skadn.json",
	}

	for _, testFile := range benchInput {
		b.Run(fmt.Sprintf("input_file_%s", testFile), func(b *testing.B) {
			b.StopTimer()
			// Set up
			fileData, err := os.ReadFile(testFile)
			if err != nil {
				b.Fatalf("unable to read file %s", testFile)
			}
			test, err := parseTestData(fileData, testFile)
			if err != nil {
				b.Fatal(err.Error())
			}
			test.endpointType = OPENRTB_ENDPOINT

			cfg := &config.Configuration{
				MaxRequestSize:    maxSize,
				BlockedApps:       test.Config.BlockedApps,
				BlockedAppsLookup: test.Config.getBlockedAppLookup(),
				AccountRequired:   test.Config.AccountRequired,
			}

			auctionEndpointHandler, _, mockBidServers, mockCurrencyRatesServer, err := buildTestEndpoint(test, cfg)
			if err != nil {
				b.Fatal(err.Error())
			}
			request := httptest.NewRequest("POST", "/openrtb2/auction", bytes.NewReader(test.BidRequest))
			recorder := httptest.NewRecorder()

			// Run benchmark
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				b.StartTimer()
				auctionEndpointHandler(recorder, request, nil) //Request comes from the unmarshalled mockBidRequest
				b.StopTimer()
			}
			for _, mockBidServer := range mockBidServers {
				mockBidServer.Close()
			}
			mockCurrencyRatesServer.Close()
		})
	}
}
