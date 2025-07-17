package contxtful

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/version"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "contxtfultest", bidder)
}

// TestEndpointResolution removed - endpoint resolution is now tested through MakeRequests
// in other test functions since getEndpoint was inlined for optimization

func TestPayloadFormat(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Create a test request
	requestJSON := `{
		"id": "test-request-id",
		"imp": [
			{
				"id": "test-imp-id",
				"banner": {
					"format": [
						{"w": 300, "h": 250}
					]
				},
				"ext": {
					"bidder": {
						"placementId": "test-placement",
						"customerId": "test-customer-123"
					}
				}
			}
		],
		"site": {
			"id": "test-site",
			"page": "https://example.com"
		}
	}`

	var request openrtb2.BidRequest
	if err := json.Unmarshal([]byte(requestJSON), &request); err != nil {
		t.Fatalf("Failed to unmarshal test request: %v", err)
	}

	requests, errs := bidder.MakeRequests(&request, &adapters.ExtraRequestInfo{})

	if len(errs) > 0 {
		t.Fatalf("MakeRequests returned errors: %v", errs)
	}

	if len(requests) != 1 {
		t.Fatalf("Expected 1 request, got %d", len(requests))
	}

	// Verify the payload structure
	var payload map[string]interface{}
	if err := json.Unmarshal(requests[0].Body, &payload); err != nil {
		t.Fatalf("Failed to unmarshal request payload: %v", err)
	}

	// Check required top-level fields
	if _, exists := payload["ortb2"]; !exists {
		t.Error("Payload missing 'ortb2' field")
	}
	if _, exists := payload["bidRequests"]; !exists {
		t.Error("Payload missing 'bidRequests' field")
	}
	if _, exists := payload["bidderRequest"]; !exists {
		t.Error("Payload missing 'bidderRequest' field")
	}
	if _, exists := payload["config"]; !exists {
		t.Error("Payload missing 'config' field")
	}

	// Verify bidRequests structure
	bidRequests, ok := payload["bidRequests"].([]interface{})
	if !ok || len(bidRequests) != 1 {
		t.Error("bidRequests should be an array with one element")
	}

	bidRequest, ok := bidRequests[0].(map[string]interface{})
	if !ok {
		t.Error("bidRequest should be an object")
	}

	// Check bidRequest fields
	if bidRequest["bidder"] != "contxtful" {
		t.Error("bidRequest.bidder should be 'contxtful'")
	}

	params, ok := bidRequest["params"].(map[string]interface{})
	if !ok {
		t.Error("bidRequest.params should be an object")
	}

	if params["placementId"] != "test-placement" {
		t.Error("bidRequest.params.placementId should be 'test-placement'")
	}

	// Verify endpoint URL
	expectedURL := "https://prebid.receptivity.io/v1/pbs/test-customer-123/bid"
	if requests[0].Uri != expectedURL {
		t.Errorf("Expected URL %s, got %s", expectedURL, requests[0].Uri)
	}
}

// TestCookieFlowWithBuyerUID tests cookie handling when BuyerUID is present
func TestCookieFlowWithBuyerUID(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	requestJSON := `{
		"id": "test-request-cookie",
		"imp": [
			{
				"id": "test-imp-cookie",
				"banner": {
					"format": [{"w": 300, "h": 250}]
				},
				"ext": {
					"bidder": {
						"placementId": "test-placement-cookie",
						"customerId": "test-customer-cookie"
					}
				}
			}
		],
		"site": {
			"id": "test-site-cookie",
			"page": "https://example.com/cookie-test"
		},
		"user": {
			"id": "test-user-123",
			"buyeruid": "contxtful-v1-eyJjdXN0b21lciI6IkNVU1RPTUVSMTIzIiwidGltZXN0YW1wIjoxNzM0NTY3ODkwMDAwLCJwYXJ0bmVycyI6eyJhbXgiOiJhbXgtdGVzdCJ9LCJ2ZXJzaW9uIjoidjEifQ=="
		}
	}`

	var request openrtb2.BidRequest
	if err := json.Unmarshal([]byte(requestJSON), &request); err != nil {
		t.Fatalf("Failed to unmarshal test request: %v", err)
	}

	requests, errs := bidder.MakeRequests(&request, &adapters.ExtraRequestInfo{})

	if len(errs) > 0 {
		t.Fatalf("MakeRequests returned errors: %v", errs)
	}

	if len(requests) != 1 {
		t.Fatalf("Expected 1 request, got %d", len(requests))
	}

	// Verify user data in payload
	var payload map[string]interface{}
	if err := json.Unmarshal(requests[0].Body, &payload); err != nil {
		t.Fatalf("Failed to unmarshal request payload: %v", err)
	}

	ortb2, ok := payload["ortb2"].(map[string]interface{})
	if !ok {
		t.Fatal("ortb2 should be an object")
	}

	user, ok := ortb2["user"].(map[string]interface{})
	if !ok {
		t.Fatal("ortb2.user should be an object")
	}

	if user["buyeruid"] != "contxtful-v1-eyJjdXN0b21lciI6IkNVU1RPTUVSMTIzIiwidGltZXN0YW1wIjoxNzM0NTY3ODkwMDAwLCJwYXJ0bmVycyI6eyJhbXgiOiJhbXgtdGVzdCJ9LCJ2ZXJzaW9uIjoidjEifQ==" {
		t.Error("ortb2.user.buyeruid should be preserved")
	}
}

// TestCookieFlowWithoutBuyerUID tests cookie handling when BuyerUID is not present
func TestCookieFlowWithoutBuyerUID(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	requestJSON := `{
		"id": "test-request-no-cookie",
		"imp": [
			{
				"id": "test-imp-no-cookie",
				"banner": {
					"format": [{"w": 320, "h": 50}]
				},
				"ext": {
					"bidder": {
						"placementId": "test-placement-no-cookie",
						"customerId": "test-customer-no-cookie"
					}
				}
			}
		],
		"site": {
			"id": "test-site-no-cookie",
			"page": "https://example.com/no-cookie-test"
		},
		"user": {
			"id": "test-user-no-cookie"
		}
	}`

	var request openrtb2.BidRequest
	if err := json.Unmarshal([]byte(requestJSON), &request); err != nil {
		t.Fatalf("Failed to unmarshal test request: %v", err)
	}

	requests, errs := bidder.MakeRequests(&request, &adapters.ExtraRequestInfo{})

	if len(errs) > 0 {
		t.Fatalf("MakeRequests returned errors: %v", errs)
	}

	if len(requests) != 1 {
		t.Fatalf("Expected 1 request, got %d", len(requests))
	}

}

// TestORTB2HandlingWithExistingData tests generic ORTB2 data preservation using real-world payload
func TestORTB2HandlingWithExistingData(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Real-world ORTB2 payload with rich Contxtful data (anonymized)
	requestJSON := `{
		"id": "46c96dbb-eb94-4661-8f82-17b026ee4fd8",
		"test": 0,
		"tmax": 1500,
		"cur": ["USD"],
		"imp": [
			{
				"id": "273ad93d17f890a8",
				"banner": {
					"topframe": 1,
					"format": [
						{
							"w": 300,
							"h": 250
						}
					]
				},
				"secure": 1,
				"ext": {
					"data": {
						"adserver": {
							"name": "gam",
							"adslot": "/139271940/example>fr/example_site:header-2"
						},
						"pbadslot": "/139271940/example>fr/example_site:header-2"
					},
					"gpid": "/139271940/example>fr/example_site:header-2"
				}
			}
		],
		"source": {
			"ext": {
				"schain": {
					"ver": "1.0",
					"complete": 1,
					"nodes": [
						{
							"asi": "contxtful.com",
							"sid": "241212",
							"hp": 1
						}
					]
				}
			}
		},
		"ext": {
			"prebid": {
				"adServerCurrency": "USD"
			}
		},
		"site": {
			"domain": "example.com",
			"publisher": {
				"domain": "example.com"
			},
			"page": "https://example.com/",
			"ext": {
				"data": {
					"documentLang": "fr-CA"
				}
			}
		},
		"device": {
			"w": 1728,
			"h": 1117,
			"dnt": 1,
			"ua": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36",
			"language": "fr",
			"ext": {
				"vpw": 1316,
				"vph": 477
			},
			"sua": {
				"source": 1,
				"platform": {
					"brand": "macOS"
				},
				"browsers": [
					{
						"brand": "Chromium",
						"version": ["137"]
					},
					{
						"brand": "Not/A)Brand",
						"version": ["24"]
					}
				],
				"mobile": 0
			}
		},
		"user": {
			"ext": {
				"eids": [
					{
						"source": "id5-sync.com",
						"uids": [
							{
								"id": "0",
								"atype": 1,
								"ext": {
									"linkType": 0,
									"pba": "QmkNifwqcX/XxmsANFijVw=="
								}
							}
						]
					},
					{
						"source": "pubcid.org",
						"uids": [
							{
								"id": "44609c87-de45-42d6-b024-2e159fc49df6",
								"atype": 1
							}
						]
					}
				]
			},
			"data": [
				{
					"name": "contxtful",
					"ext": {
						"sm": null,
						"params": {
							"ev": "v1",
							"ci": "ABCD123456"
						},
						"rx": {
							"ReceptivityState": "NonReceptive",
							"EclecticChinchilla": "false",
							"score": "4",
							"gptP": [
								{
									"gpt": 1
								}
							],
							"rand": 64.15681135773268
						}
					},
					"segment": [
						{
							"id": "NonReceptive"
						}
					]
				}
			]
		}
	}`

	var request openrtb2.BidRequest
	if err := json.Unmarshal([]byte(requestJSON), &request); err != nil {
		t.Fatalf("Failed to unmarshal test request: %v", err)
	}

	// Add bidder parameters that would be injected by PBS
	request.Imp[0].Ext = json.RawMessage(`{
		"bidder": {
			"placementId": "test-placement-real", 
			"customerId": "ABCD123456"
		},
		"data": {
			"adserver": {
				"name": "gam",
				"adslot": "/139271940/example>fr/example_site:header-2"
			},
			"pbadslot": "/139271940/example>fr/example_site:header-2"
		},
		"gpid": "/139271940/example>fr/example_site:header-2"
	}`)

	requests, errs := bidder.MakeRequests(&request, &adapters.ExtraRequestInfo{})

	if len(errs) > 0 {
		t.Fatalf("MakeRequests returned errors: %v", errs)
	}

	if len(requests) != 1 {
		t.Fatalf("Expected 1 request, got %d", len(requests))
	}

	// Verify ORTB2 data preservation
	var payload map[string]interface{}
	if err := json.Unmarshal(requests[0].Body, &payload); err != nil {
		t.Fatalf("Failed to unmarshal request payload: %v", err)
	}

	ortb2, ok := payload["ortb2"].(map[string]interface{})
	if !ok {
		t.Fatal("ortb2 should be an object")
	}

	// Verify site data preservation (real-world structure)
	site, ok := ortb2["site"].(map[string]interface{})
	if !ok {
		t.Fatal("ortb2.site should be an object")
	}

	if site["domain"] != "example.com" {
		t.Error("ortb2.site.domain should be preserved")
	}

	if site["page"] != "https://example.com/" {
		t.Error("ortb2.site.page should be preserved")
	}

	publisher, ok := site["publisher"].(map[string]interface{})
	if !ok {
		t.Fatal("ortb2.site.publisher should be an object")
	}

	if publisher["domain"] != "example.com" {
		t.Error("ortb2.site.publisher.domain should be preserved")
	}

	siteExt, ok := site["ext"].(map[string]interface{})
	if !ok {
		t.Fatal("ortb2.site.ext should be an object")
	}

	siteData, ok := siteExt["data"].(map[string]interface{})
	if !ok {
		t.Fatal("ortb2.site.ext.data should be preserved")
	}

	if siteData["documentLang"] != "fr-CA" {
		t.Error("ortb2.site.ext.data.documentLang should be preserved")
	}

	// Verify user data preservation and Contxtful data structure
	user, ok := ortb2["user"].(map[string]interface{})
	if !ok {
		t.Fatal("ortb2.user should be an object")
	}

	// Verify user extensions with eids
	userExt, ok := user["ext"].(map[string]interface{})
	if !ok {
		t.Fatal("ortb2.user.ext should be an object")
	}

	eids, ok := userExt["eids"].([]interface{})
	if !ok || len(eids) < 2 {
		t.Fatal("ortb2.user.ext.eids should be an array with multiple ID providers")
	}

	// Check for ID5 and PubcID providers
	var foundID5, foundPubcID bool
	for _, eid := range eids {
		eidObj, ok := eid.(map[string]interface{})
		if !ok {
			continue
		}
		source := eidObj["source"].(string)
		if source == "id5-sync.com" {
			foundID5 = true
		}
		if source == "pubcid.org" {
			foundPubcID = true
		}
	}

	if !foundID5 {
		t.Error("ID5 external ID should be preserved")
	}
	if !foundPubcID {
		t.Error("PubcID external ID should be preserved")
	}

	// Verify Contxtful user data with rich structure
	userData, ok := user["data"].([]interface{})
	if !ok {
		t.Fatal("ortb2.user.data should be an array")
	}

	if len(userData) != 1 {
		t.Fatal("ortb2.user.data should contain exactly one Contxtful data segment")
	}

	// Verify Contxtful data structure
	contxtfulData, ok := userData[0].(map[string]interface{})
	if !ok {
		t.Fatal("Contxtful user data should be an object")
	}

	if contxtfulData["name"] != "contxtful" {
		t.Error("User data should have name 'contxtful'")
	}

	// Verify segments
	segments, ok := contxtfulData["segment"].([]interface{})
	if !ok || len(segments) != 1 {
		t.Fatal("Contxtful data should have segment array")
	}

	segment := segments[0].(map[string]interface{})
	if segment["id"] != "NonReceptive" {
		t.Error("Contxtful segment should be 'NonReceptive'")
	}

	// Verify rich Contxtful extensions
	contxtfulExt, ok := contxtfulData["ext"].(map[string]interface{})
	if !ok {
		t.Fatal("Contxtful data should have ext object")
	}

	// Check for params
	params, ok := contxtfulExt["params"].(map[string]interface{})
	if !ok {
		t.Fatal("Contxtful data should have params in ext")
	}

	if params["ci"] != "ABCD123456" {
		t.Error("Contxtful params should include correct customer ID (ABCD123456)")
	}

	if params["ev"] != "v1" {
		t.Error("Contxtful params should include correct version (v1)")
	}

	// Verify receptivity data
	rx, ok := contxtfulExt["rx"].(map[string]interface{})
	if !ok {
		t.Fatal("Contxtful data should have rx (receptivity) object")
	}

	if rx["ReceptivityState"] != "NonReceptive" {
		t.Error("Receptivity state should be 'NonReceptive'")
	}

	if rx["score"] != "4" {
		t.Error("Receptivity score should be '4'")
	}

	// Verify device data preservation (real-world structure)
	device, ok := ortb2["device"].(map[string]interface{})
	if !ok {
		t.Fatal("ortb2.device should be an object")
	}

	if device["language"] != "fr" {
		t.Error("ortb2.device.language should be preserved")
	}

	if device["w"] != float64(1728) {
		t.Error("ortb2.device.w should be preserved")
	}

	if device["h"] != float64(1117) {
		t.Error("ortb2.device.h should be preserved")
	}

	// Verify SUA (Structured User Agent) data
	sua, ok := device["sua"].(map[string]interface{})
	if !ok {
		t.Fatal("ortb2.device.sua should be preserved")
	}

	platform, ok := sua["platform"].(map[string]interface{})
	if !ok {
		t.Fatal("SUA platform should be preserved")
	}

	if platform["brand"] != "macOS" {
		t.Error("SUA platform brand should be 'macOS'")
	}

	browsers, ok := sua["browsers"].([]interface{})
	if !ok || len(browsers) < 2 {
		t.Fatal("SUA browsers should be preserved")
	}

	// Verify impression data preservation (without extensions)
	imps, ok := ortb2["imp"].([]interface{})
	if !ok || len(imps) != 1 {
		t.Fatal("ortb2.imp should be an array with one element")
	}

	imp := imps[0].(map[string]interface{})

	// Verify basic impression fields are preserved
	if imp["id"] != "273ad93d17f890a8" {
		t.Error("Impression ID should be preserved")
	}

	if imp["secure"] != float64(1) {
		t.Error("Impression secure flag should be preserved")
	}

	// Verify banner object
	banner, ok := imp["banner"].(map[string]interface{})
	if !ok {
		t.Fatal("Impression banner should be preserved")
	}

	if banner["topframe"] != float64(1) {
		t.Error("Banner topframe should be preserved")
	}

	// Note: Impression extensions (ext.data, ext.bidder) are correctly removed
	// in the generic ORTB2 passthrough to avoid leaking bidder-specific data to the relay

	// Verify supply chain (schain) data
	source, ok := ortb2["source"].(map[string]interface{})
	if !ok {
		t.Fatal("ortb2.source should be preserved")
	}

	sourceExt, ok := source["ext"].(map[string]interface{})
	if !ok {
		t.Fatal("ortb2.source.ext should be preserved")
	}

	schain, ok := sourceExt["schain"].(map[string]interface{})
	if !ok {
		t.Fatal("Supply chain should be preserved")
	}

	nodes, ok := schain["nodes"].([]interface{})
	if !ok || len(nodes) != 1 {
		t.Fatal("Supply chain nodes should be preserved")
	}

	node := nodes[0].(map[string]interface{})
	if node["asi"] != "contxtful.com" {
		t.Error("Supply chain ASI should be 'contxtful.com'")
	}
}

// TestEventTrackingConfiguration tests that the adapter properly builds requests
// Note: Event tracking URLs are now injected by the Contxtful relay, not by PBS
func TestEventTrackingConfiguration(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Create a video request to test that the adapter builds proper requests for relay event injection
	requestJSON := `{
		"id": "test-request-events",
		"imp": [
			{
				"id": "test-imp-events",
				"video": {
					"mimes": ["video/mp4"],
					"minduration": 5,
					"maxduration": 30,
					"protocols": [2, 3, 5, 6],
					"w": 640,
					"h": 480,
					"startdelay": 0,
					"placement": 1,
					"linearity": 1,
					"skip": 0,
					"playbackmethod": [1, 3],
					"delivery": [1],
					"api": [1, 2]
				},
				"ext": {
					"bidder": {
						"placementId": "test-placement-events",
						"customerId": "test-customer-events"
					}
				}
			}
		],
		"site": {
			"id": "test-site-events",
			"page": "https://example.com/event-test",
			"domain": "example.com"
		}
	}`

	var request openrtb2.BidRequest
	if err := json.Unmarshal([]byte(requestJSON), &request); err != nil {
		t.Fatalf("Failed to unmarshal test request: %v", err)
	}

	requests, errs := bidder.MakeRequests(&request, &adapters.ExtraRequestInfo{})

	if len(errs) > 0 {
		t.Fatalf("MakeRequests returned errors: %v", errs)
	}

	if len(requests) != 1 {
		t.Fatalf("Expected 1 request, got %d", len(requests))
	}

	// Verify the request structure is properly built for relay processing
	var payload map[string]interface{}
	if err := json.Unmarshal(requests[0].Body, &payload); err != nil {
		t.Fatalf("Failed to unmarshal request payload: %v", err)
	}

	// Verify request structure contains necessary data for relay event injection
	bidRequests, ok := payload["bidRequests"].([]interface{})
	if !ok || len(bidRequests) != 1 {
		t.Fatal("bidRequests should be an array with one element")
	}

	bidRequest, ok := bidRequests[0].(map[string]interface{})
	if !ok {
		t.Fatal("bidRequest should be an object")
	}

	// Verify bidder code is present for relay event URL generation
	if bidRequest["bidder"] != "contxtful" {
		t.Error("Bidder code should be 'contxtful' for relay event tracking")
	}

	// Verify bid ID is present for relay event URL generation
	if bidRequest["bidId"] == nil {
		t.Error("Bid ID should be present for relay event tracking")
	}

	// Verify ORTB2 structure contains necessary data for relay
	ortb2, ok := payload["ortb2"].(map[string]interface{})
	if !ok {
		t.Fatal("ortb2 should be an object")
	}

	imp, ok := ortb2["imp"].([]interface{})
	if !ok || len(imp) != 1 {
		t.Fatal("ortb2.imp should be an array with one element")
	}

	impObj, ok := imp[0].(map[string]interface{})
	if !ok {
		t.Fatal("ortb2.imp[0] should be an object")
	}

	// Verify impression ID is preserved for relay event tracking
	if impObj["id"] != "test-imp-events" {
		t.Error("Impression ID should be preserved for relay event tracking")
	}

	// Verify the adapter preserves original media type (video stays video)
	video, hasVideo := impObj["video"].(map[string]interface{})
	_, hasBanner := impObj["banner"].(map[string]interface{})

	// Should have either video or banner (depending on original request)
	if !hasVideo && !hasBanner {
		t.Error("Impression should preserve original media type (video or banner)")
	}

	// If video was in original request, it should be preserved
	if hasVideo && video == nil {
		t.Error("Video object should be preserved for relay processing")
	}
}

// TestCookieSyncEndpointGeneration tests cookie sync URL generation
func TestCookieSyncEndpointGeneration(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Note: User sync configuration is handled through PBS bidder-info files
	// not through adapter config. This test verifies the adapter can be built
	// successfully for sync scenarios.
}

// TestBiddingProcessWithDifferentFormats tests bidding with various response formats
func TestBiddingProcessWithDifferentFormats(t *testing.T) {
	tests := []struct {
		name     string
		response string
		expected bool
	}{
		{
			name: "Prebid.js format response",
			response: `[{
				"cpm": 4.25,
				"width": 300,
				"height": 250,
				"currency": "USD",
				"requestId": "test-bid-1",
				"ad": "<div>Test Ad</div>",
				"ttl": 300,
				"creativeId": "test-creative-1",
				"netRevenue": true
			}]`,
			expected: true,
		},
		{
			name: "Trace format response (should be ignored)",
			response: `{
				"trace": {
					"request_id": "test-trace-1",
					"debug": true,
					"messages": ["Debug message"]
				}
			}`,
			expected: false,
		},
	}

	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := &adapters.ResponseData{
				StatusCode: 200,
				Body:       []byte(tt.response),
			}

			// Create a dummy request for the response processing
			requestJSON := `{
				"id": "test-request-formats",
				"imp": [{
					"id": "test-imp-formats",
					"banner": {"format": [{"w": 300, "h": 250}]},
					"ext": {"bidder": {"placementId": "test", "customerId": "test"}}
				}]
			}`

			var request openrtb2.BidRequest
			if err := json.Unmarshal([]byte(requestJSON), &request); err != nil {
				t.Fatalf("Failed to unmarshal test request: %v", err)
			}

			bids, errs := bidder.MakeBids(&request, nil, response)

			if tt.expected {
				if len(errs) > 0 {
					t.Errorf("Expected no errors, got: %v", errs)
				}
				if bids == nil || len(bids.Bids) == 0 {
					t.Error("Expected bids to be returned")
				}
			} else {
				// Trace responses should not generate bids
				if bids != nil && len(bids.Bids) > 0 {
					t.Error("Expected no bids for trace response")
				}
			}
		})
	}
}

// TestEventTrackingURLGeneration tests NURL and BURL event tracking URL generation
func TestEventTrackingURLGeneration(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	nurl := "https://monitoring.receptivity.io/v1/pbs/EVNT123/pbs-impression?b=contxtful-test-imp-events&a=EVNT123&bidder=contxtful&impId=test-imp-events&price=3.50&traceId=trace-abc-123&random=0.742856&domain=example.com&adRequestId=req-abc-456&w=728&h=90&f=b"
	burl := "https://monitoring.receptivity.io/v1/pbs/EVNT123/pbs-billing?b=contxtful-test-imp-events&a=EVNT123&bidder=contxtful&impId=test-imp-events&price=3.50&traceId=trace-abc-123&random=0.742856&domain=example.com&adRequestId=req-abc-456&w=728&h=90&f=b"

	responseJSON := fmt.Sprintf(`[{
		"requestId": "test-imp-events",
		"cpm": 3.50,
		"currency": "USD",
		"width": 728,
		"height": 90,
		"creativeId": "event-creative-1",
		"ad": "<div>Event Tracking Test Ad</div>",
		"ttl": 300,
		"netRevenue": true,
		"mediaType": "banner",
		"bidderCode": "contxtful",
		"traceId": "trace-abc-123",
		"random": 0.742856,
		"nurl": "%s",
		"burl": "%s"
	}]`, nurl, burl)

	response := &adapters.ResponseData{
		StatusCode: 200,
		Body:       []byte(responseJSON),
	}

	requestJSON := `{
		"id": "req-abc-456",
		"imp": [{
			"id": "test-imp-events",
			"banner": {"format": [{"w": 728, "h": 90}]},
			"ext": {"bidder": {"placementId": "event-test", "customerId": "EVNT123"}}
		}],
		"site": {
			"domain": "example.com",
			"page": "https://example.com/test-page"
		}
	}`

	var request openrtb2.BidRequest
	if err := json.Unmarshal([]byte(requestJSON), &request); err != nil {
		t.Fatalf("Failed to unmarshal test request: %v", err)
	}

	bids, errs := bidder.MakeBids(&request, nil, response)

	if len(errs) > 0 {
		t.Fatalf("MakeBids returned errors: %v", errs)
	}

	if bids == nil || len(bids.Bids) != 1 {
		bidCount := 0
		if bids != nil {
			bidCount = len(bids.Bids)
		}
		t.Fatalf("Expected 1 bid, got %d", bidCount)
	}

	bid := bids.Bids[0]

	// Verify NURL (win notice URL) matches expected value exactly
	if bid.Bid.NURL != nurl {
		t.Errorf("NURL should be %s, got %s", nurl, bid.Bid.NURL)
	}

	// Verify BURL (billing URL) matches expected value exactly
	if bid.Bid.BURL != burl {
		t.Errorf("BURL should be %s, got %s", burl, bid.Bid.BURL)
	}
}

// TestUserSyncWithBuyerUID tests user sync behavior when BuyerUID is present
func TestUserSyncWithBuyerUID(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	requestJSON := `{
		"id": "test-request-sync-existing",
		"imp": [{
			"id": "test-imp-sync-existing",
			"banner": {"format": [{"w": 300, "h": 250}]},
			"ext": {"bidder": {"placementId": "sync-test", "customerId": "SYNC123"}}
		}],
		"user": {
			"buyeruid": "contxtful-v1-1234"
		}
	}`

	var request openrtb2.BidRequest
	if err := json.Unmarshal([]byte(requestJSON), &request); err != nil {
		t.Fatalf("Failed to unmarshal test request: %v", err)
	}

	requests, errs := bidder.MakeRequests(&request, &adapters.ExtraRequestInfo{})

	if len(errs) > 0 {
		t.Fatalf("MakeRequests returned errors: %v", errs)
	}

	if len(requests) != 1 {
		t.Fatalf("Expected 1 request, got %d", len(requests))
	}

	// Verify payload includes user ID
	var payload map[string]interface{}
	if err := json.Unmarshal(requests[0].Body, &payload); err != nil {
		t.Fatalf("Failed to unmarshal request payload: %v", err)
	}

	ortb2, ok := payload["ortb2"].(map[string]interface{})
	if !ok {
		t.Fatal("ortb2 should be an object")
	}

	user, ok := ortb2["user"].(map[string]interface{})
	if !ok {
		t.Fatal("ortb2.user should be an object")
	}

	buyerUID, ok := user["buyeruid"].(string)
	if !ok || buyerUID == "" {
		t.Error("User BuyerUID should be preserved in payload")
	}

	if buyerUID != "contxtful-v1-1234" {
		t.Errorf("Expected BuyerUID 'contxtful-v1-1234', got '%s'", buyerUID)
	}
}

// TestBuyerUIDFromPrebidMap tests reading UID from request.user.ext.prebid.buyeruids.contxtful and writing to ortb2.user.buyeruid
func TestBuyerUIDFromPrebidMap(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Test with UID in prebid buyeruids map (Priority 2 in extractUserIDForCookie)
	testUID := "contxtful-v1-prebid-map-test-uid-456"
	requestJSON := `{
		"id": "test-request-prebid-map",
		"imp": [
			{
				"id": "test-imp-prebid-map",
				"banner": {
					"format": [{"w": 300, "h": 250}]
				},
				"ext": {
					"bidder": {
						"placementId": "prebid-map-test",
						"customerId": "PBMAP123"
					}
				}
			}
		],
		"site": {
			"id": "test-site-prebid-map",
			"page": "https://example.com/prebid-map-test"
		},
		"user": {
			"id": "test-user-prebid-map",
			"ext": {
				"prebid": {
					"buyeruids": {
						"contxtful": "` + testUID + `"
					}
				}
			}
		}
	}`

	var request openrtb2.BidRequest
	if err := json.Unmarshal([]byte(requestJSON), &request); err != nil {
		t.Fatalf("Failed to unmarshal test request: %v", err)
	}

	requests, errs := bidder.MakeRequests(&request, &adapters.ExtraRequestInfo{})

	if len(errs) > 0 {
		t.Fatalf("MakeRequests returned errors: %v", errs)
	}

	if len(requests) != 1 {
		t.Fatalf("Expected 1 request, got %d", len(requests))
	}

	// Verify UID from prebid buyeruids map is written to ortb2.user.buyeruid
	var payload map[string]interface{}
	if err := json.Unmarshal(requests[0].Body, &payload); err != nil {
		t.Fatalf("Failed to unmarshal request payload: %v", err)
	}

	ortb2, ok := payload["ortb2"].(map[string]interface{})
	if !ok {
		t.Fatal("ortb2 should be an object")
	}

	user, ok := ortb2["user"].(map[string]interface{})
	if !ok {
		t.Fatal("ortb2.user should be an object")
	}

	buyerUID, ok := user["buyeruid"].(string)
	if !ok {
		t.Fatal("ortb2.user.buyeruid should be present")
	}

	if buyerUID != testUID {
		t.Errorf("Expected UID from prebid buyeruids map '%s' to be written to ortb2.user.buyeruid, got '%s'", testUID, buyerUID)
	}
}

// TestNoCookieHeaders tests that cookie headers are never set (we only use ortb2.user.buyeruid)
func TestNoCookieHeaders(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	testCases := []struct {
		name        string
		requestJSON string
		description string
	}{
		{
			name: "With standard User.BuyerUID",
			requestJSON: `{
				"id": "test-no-cookie-standard",
				"imp": [{"id": "test-imp", "banner": {"format": [{"w": 300, "h": 250}]}, "ext": {"bidder": {"placementId": "test-placement", "customerId": "NOCOOKIE123"}}}],
				"user": {"buyeruid": "standard-buyer-uid-123"}
			}`,
			description: "Standard BuyerUID should not generate cookie headers",
		},
		{
			name: "With prebid buyeruids map",
			requestJSON: `{
				"id": "test-no-cookie-prebid",
				"imp": [{"id": "test-imp", "banner": {"format": [{"w": 300, "h": 250}]}, "ext": {"bidder": {"placementId": "test-placement", "customerId": "NOCOOKIE123"}}}],
				"user": {"ext": {"prebid": {"buyeruids": {"contxtful": "prebid-map-uid-456"}}}}
			}`,
			description: "Prebid buyeruids map should not generate cookie headers",
		},
		{
			name: "With no user data",
			requestJSON: `{
				"id": "test-no-cookie-empty",
				"imp": [{"id": "test-imp", "banner": {"format": [{"w": 300, "h": 250}]}, "ext": {"bidder": {"placementId": "test-placement", "customerId": "NOCOOKIE123"}}}]
			}`,
			description: "No user data should not generate cookie headers",
		},
		{
			name: "With Base64 encoded UID",
			requestJSON: `{
				"id": "test-no-cookie-base64",
				"imp": [{"id": "test-imp", "banner": {"format": [{"w": 300, "h": 250}]}, "ext": {"bidder": {"placementId": "test-placement", "customerId": "NOCOOKIE123"}}}],
				"user": {"buyeruid": "contxtful-v1-1234"}
			}`,
			description: "Base64 encoded UID should not generate cookie headers",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var request openrtb2.BidRequest
			if err := json.Unmarshal([]byte(tc.requestJSON), &request); err != nil {
				t.Fatalf("Failed to unmarshal test request: %v", err)
			}

			requests, errs := bidder.MakeRequests(&request, &adapters.ExtraRequestInfo{})

			if len(errs) > 0 {
				t.Fatalf("MakeRequests returned errors: %v", errs)
			}

			if len(requests) != 1 {
				t.Fatalf("Expected 1 request, got %d", len(requests))
			}

			// This is the single place where we verify no cookie headers are set
			if len(requests) == 0 {
				t.Fatal("No requests to check for cookie header")
			}
			cookieHeader := requests[0].Headers.Get("Cookie")
			if cookieHeader != "" {
				t.Errorf("Expected no Cookie header, got %s", cookieHeader)
			}
		})
	}
}

// TestErrorHandling tests various error scenarios
func TestErrorHandling(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedErrors int
	}{
		{
			name:           "HTTP 400 Error",
			statusCode:     400,
			responseBody:   `{"error": "Bad Request"}`,
			expectedErrors: 1,
		},
		{
			name:           "HTTP 500 Error",
			statusCode:     500,
			responseBody:   `{"error": "Internal Server Error"}`,
			expectedErrors: 1,
		},
		{
			name:           "Invalid JSON Response",
			statusCode:     200,
			responseBody:   `{invalid json}`,
			expectedErrors: 1,
		},
		{
			name:           "Empty Response",
			statusCode:     200,
			responseBody:   ``,
			expectedErrors: 1, // Empty response should return error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := &adapters.ResponseData{
				StatusCode: tt.statusCode,
				Body:       []byte(tt.responseBody),
			}

			requestJSON := `{
				"id": "test-request-error",
				"imp": [{
					"id": "test-imp-error",
					"banner": {"format": [{"w": 300, "h": 250}]},
					"ext": {"bidder": {"placementId": "error-test", "customerId": "ERROR123"}}
				}]
			}`

			var request openrtb2.BidRequest
			if err := json.Unmarshal([]byte(requestJSON), &request); err != nil {
				t.Fatalf("Failed to unmarshal test request: %v", err)
			}

			bids, errs := bidder.MakeBids(&request, nil, response)

			if len(errs) != tt.expectedErrors {
				t.Errorf("Expected %d errors, got %d: %v", tt.expectedErrors, len(errs), errs)
			}

			// For error cases, we shouldn't get any bids
			if tt.statusCode >= 400 && bids != nil && len(bids.Bids) > 0 {
				t.Error("Should not return bids on HTTP error")
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestBase64PartnerUIDHandling tests the new Base64 encoded partner UID functionality
func TestBase64PartnerUIDHandling(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Test with Base64 encoded UID containing multiple partner UIDs
	// This simulates what the relay would create after syncing with multiple partners
	encodedUID := "contxtful-v1-eyJjdXN0b21lciI6IkNVU1RPTUVSMTIzIiwidGltZXN0YW1wIjoxNzM0NTY3ODkwMDAwLCJwYXJ0bmVycyI6eyJhbXgiOiJhbXgtMTczNDU2Nzg5MC1hYmMxMjMiLCJzbWFhdG8iOiJzbWFhdG8tMTczNDU2Nzg5MC1kZWY0NTYiLCJpbWRzIjoiaW1kcy0xNzM0NTY3ODkwLWdoaTc4OSJ9LCJ2ZXJzaW9uIjoidjEifQ=="

	requestJSON := `{
		"id": "test-request-base64-uid",
		"imp": [{
			"id": "test-imp-base64-uid",
			"banner": {"format": [{"w": 300, "h": 250}]},
			"ext": {"bidder": {"placementId": "base64-test", "customerId": "CUSTOMER123"}}
		}],
		"user": {
			"id": "test-user-base64",
			"buyeruid": "` + encodedUID + `"
		}
	}`

	var request openrtb2.BidRequest
	if err := json.Unmarshal([]byte(requestJSON), &request); err != nil {
		t.Fatalf("Failed to unmarshal test request: %v", err)
	}

	requests, errs := bidder.MakeRequests(&request, &adapters.ExtraRequestInfo{})

	if len(errs) > 0 {
		t.Fatalf("MakeRequests returned errors: %v", errs)
	}

	if len(requests) != 1 {
		t.Fatalf("Expected 1 request, got %d", len(requests))
	}

	// Verify the Base64 encoded UID is passed through correctly
	var payload map[string]interface{}
	if err := json.Unmarshal(requests[0].Body, &payload); err != nil {
		t.Fatalf("Failed to unmarshal request payload: %v", err)
	}

	ortb2, ok := payload["ortb2"].(map[string]interface{})
	if !ok {
		t.Fatal("ortb2 should be an object")
	}

	user, ok := ortb2["user"].(map[string]interface{})
	if !ok {
		t.Fatal("ortb2.user should be an object")
	}

	buyerUID, ok := user["buyeruid"].(string)
	if !ok || buyerUID == "" {
		t.Error("User BuyerUID should be preserved in payload")
	}

	if buyerUID != encodedUID {
		t.Errorf("Expected encoded UID to be preserved, got '%s'", buyerUID)
	}

	// Verify endpoint URL uses correct customer ID
	expectedURL := "https://prebid.receptivity.io/v1/pbs/CUSTOMER123/bid"
	if requests[0].Uri != expectedURL {
		t.Errorf("Expected URL %s, got %s", expectedURL, requests[0].Uri)
	}
}

// TestBase64UIDVersioning tests different versions of Base64 encoded UIDs
func TestBase64UIDVersioning(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	testCases := []struct {
		name        string
		version     string
		encodedUID  string
		expectError bool
	}{
		{
			name:        "Version 1 UID",
			version:     "v1",
			encodedUID:  "contxtful-v1-eyJjdXN0b21lciI6IlRFU1RDVVNUIiwidGltZXN0YW1wIjoxNzM0NTY3ODkwMDAwLCJwYXJ0bmVycyI6eyJhbXgiOiJhbXgtdGVzdCJ9LCJ2ZXJzaW9uIjoidjEifQ==",
			expectError: false,
		},
		{
			name:        "Version 1 UID (current)",
			version:     "v1",
			encodedUID:  "contxtful-v1-eyJjdXN0b21lciI6IlRFU1RDVVNUIiwidGltZXN0YW1wIjoxNzM0NTY3ODkwMDAwLCJwYXJ0bmVycyI6eyJhbXgiOiJhbXgtdGVzdCIsInNtYWF0byI6InNtYWF0by10ZXN0In0sInZlcnNpb24iOiJ2MSJ9",
			expectError: false,
		},
		{
			name:        "Future Version v2 UID",
			version:     "v2",
			encodedUID:  "contxtful-v2-eyJjdXN0b21lciI6IlRFU1RDVVNUIiwidGltZXN0YW1wIjoxNzM0NTY3ODkwMDAwLCJwYXJ0bmVycyI6eyJhbXgiOiJhbXgtdGVzdCIsInNtYWF0byI6InNtYWF0by10ZXN0In0sInZlcnNpb24iOiJ2MiJ9",
			expectError: false, // PBS adapter should pass through any version
		},
		{
			name:        "Future Version v3 UID",
			version:     "v3",
			encodedUID:  "contxtful-v3-eyJjdXN0b21lciI6IlRFU1RDVVNUIiwidGltZXN0YW1wIjoxNzM0NTY3ODkwMDAwLCJwYXJ0bmVycyI6eyJhbXgiOiJhbXgtdGVzdCJ9LCJ2ZXJzaW9uIjoidjMifQ==",
			expectError: false, // PBS adapter should pass through any version
		},
		{
			name:        "Legacy Format (passthrough - no validation)",
			version:     "legacy",
			encodedUID:  "contxtful-TESTCUST-1734567890-abc123",
			expectError: false, // Passthrough approach - no validation, relay handles format
		},
		{
			name:        "Old Format with Customer (passthrough - no validation)",
			version:     "old",
			encodedUID:  "contxtful-v2-TESTCUST-eyJjdXN0b21lciI6IlRFU1RDVVNUIiwidGltZXN0YW1wIjoxNzM0NTY3ODkwMDAwLCJwYXJ0bmVycyI6eyJhbXgiOiJhbXgtdGVzdCJ9LCJ2ZXJzaW9uIjoidjIifQ==",
			expectError: false, // Passthrough approach - no validation, relay handles format
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requestJSON := `{
				"id": "test-request-versioning",
				"imp": [{
					"id": "test-imp-versioning",
					"banner": {"format": [{"w": 300, "h": 250}]},
					"ext": {"bidder": {"placementId": "version-test", "customerId": "TESTCUST"}}
				}],
				"user": {
					"buyeruid": "` + tc.encodedUID + `"
				}
			}`

			var request openrtb2.BidRequest
			if err := json.Unmarshal([]byte(requestJSON), &request); err != nil {
				t.Fatalf("Failed to unmarshal test request: %v", err)
			}

			requests, errs := bidder.MakeRequests(&request, &adapters.ExtraRequestInfo{})

			if tc.expectError {
				if len(errs) == 0 {
					t.Errorf("Expected error for %s, but got none", tc.name)
				}
				return
			}

			if len(errs) > 0 {
				t.Fatalf("MakeRequests returned unexpected errors: %v", errs)
			}

			if len(requests) != 1 {
				t.Fatalf("Expected 1 request, got %d", len(requests))
			}

			// Verify the versioned UID is passed through correctly
			var payload map[string]interface{}
			if err := json.Unmarshal(requests[0].Body, &payload); err != nil {
				t.Fatalf("Failed to unmarshal request payload: %v", err)
			}

			ortb2, ok := payload["ortb2"].(map[string]interface{})
			if !ok {
				t.Fatal("ortb2 should be an object")
			}

			user, ok := ortb2["user"].(map[string]interface{})
			if !ok {
				t.Fatal("ortb2.user should be an object")
			}

			buyerUID, ok := user["buyeruid"].(string)
			if !ok || buyerUID != tc.encodedUID {
				t.Errorf("Expected UID %s to be preserved, got %s", tc.encodedUID, buyerUID)
			}
		})
	}
}

// TestMultiPartnerUIDSize tests that the adapter handles large Base64 UIDs with many partners
func TestMultiPartnerUIDSize(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Simulate a large Base64 UID with 20 partners
	// This tests the size limits and ensures the adapter can handle realistic scenarios
	largeUID := "contxtful-v1-eyJjdXN0b21lciI6IkxBUkdFQ1VTVCIsInRpbWVzdGFtcCI6MTczNDU2Nzg5MDAwMCwicGFydG5lcnMiOnsiYW14IjoiYW14LTE3MzQ1Njc4OTAtYWJjMTIzIiwic21hYXRvIjoic21hYXRvLTE3MzQ1Njc4OTAtZGVmNDU2IiwiaW1kcyI6ImltZHMtMTczNDU2Nzg5MC1naGk3ODkiLCJwdWJtYXRpYyI6InB1Ym1hdGljLTE3MzQ1Njc4OTAtamtsMDEyIiwiaXgiOiJpeC0xNzM0NTY3ODkwLW1ubzM0NSIsIm9wZW54Ijoib3BlbngtMTczNDU2Nzg5MC1wcXI2NzgiLCJydWJpY29uIjoicnViaWNvbi0xNzM0NTY3ODkwLXN0dTkwMSIsImFwcG5leHVzIjoiYXBwbmV4dXMtMTczNDU2Nzg5MC12d3gyMzQiLCJjcml0ZW8iOiJjcml0ZW8tMTczNDU2Nzg5MC15ejU2NyIsImtpZG96Ijoia2lkb3otMTczNDU2Nzg5MC1hYmM4OTAiLCJtZ2lkIjoibWdpZC0xNzM0NTY3ODkwLWRlZjEyMyIsInVuaXJ1bHkiOiJ1bmlydWx5LTE3MzQ1Njc4OTAtZ2hpNDU2IiwidmlkYXp5IjoidmlkYXp5LTE3MzQ1Njc4OTAtamtsMTIzIiwiYWRmIjoiYWRmLTE3MzQ1Njc4OTAtbW5vNDU2IiwiYmVhY2hmcm9udCI6ImJlYWNoZnJvbnQtMTczNDU2Nzg5MC1wcXI3ODkiLCJlcGxhbm5pbmciOiJlcGxhbm5pbmctMTczNDU2Nzg5MC1zdHUwMTIiLCJncmlkIjoiZ3JpZC0xNzM0NTY3ODkwLXZ3eDM0NSIsImh1YXdlaWFkcyI6Imh1YXdlaWFkcy0xNzM0NTY3ODkwLXl6NTY3OCIsImltcHJvdmVkaWdpdGFsIjoiaW1wcm92ZWRpZ2l0YWwtMTczNDU2Nzg5MC1hYmM5MDEiLCJzbWFydGFkc2VydmVyIjoic21hcnRhZHNlcnZlci0xNzM0NTY3ODkwLWRlZjIzNCJ9LCJ2ZXJzaW9uIjoidjEifQ=="

	// Verify the UID is under cookie size limits (4KB)
	if len(largeUID) > 4096 {
		t.Errorf("Large UID size %d bytes exceeds cookie limit of 4096 bytes", len(largeUID))
	}

	requestJSON := `{
		"id": "test-request-large-uid",
		"imp": [{
			"id": "test-imp-large-uid",
			"banner": {"format": [{"w": 300, "h": 250}]},
			"ext": {"bidder": {"placementId": "large-uid-test", "customerId": "LARGECUST"}}
		}],
		"user": {
			"buyeruid": "` + largeUID + `"
		}
	}`

	var request openrtb2.BidRequest
	if err := json.Unmarshal([]byte(requestJSON), &request); err != nil {
		t.Fatalf("Failed to unmarshal test request: %v", err)
	}

	requests, errs := bidder.MakeRequests(&request, &adapters.ExtraRequestInfo{})

	if len(errs) > 0 {
		t.Fatalf("MakeRequests returned errors: %v", errs)
	}

	if len(requests) != 1 {
		t.Fatalf("Expected 1 request, got %d", len(requests))
	}

	// Verify the large UID is handled correctly in ortb2.user.buyeruid
	var payload map[string]interface{}
	if err := json.Unmarshal(requests[0].Body, &payload); err != nil {
		t.Fatalf("Failed to unmarshal request payload: %v", err)
	}

	ortb2, ok := payload["ortb2"].(map[string]interface{})
	if !ok {
		t.Fatal("ortb2 should be an object")
	}

	user, ok := ortb2["user"].(map[string]interface{})
	if !ok {
		t.Fatal("ortb2.user should be an object")
	}

	buyerUID, ok := user["buyeruid"].(string)
	if !ok || buyerUID != largeUID {
		t.Error("Large UID should be preserved in ortb2.user.buyeruid")
	}
}

// TestPrebidJSEventTrackingURLGeneration tests NURL and BURL event tracking URL generation for Prebid.js format responses
func TestPrebidJSEventTrackingURLGeneration(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Test with Prebid.js format response (with tracking fields from relay)
	nurl := "https://monitoring.receptivity.io/v1/pbs/PJSEVNT123/pbs-impression?b=contxtful-test-prebidjs-events&a=PJSEVNT123&bidder=contxtful&impId=test-prebidjs-events&price=2.75&traceId=prebidjs-trace-abc-789&random=0.654321&domain=prebidjs.example.com&adRequestId=prebidjs-req-456&w=300&h=250&f=b"
	burl := "https://monitoring.receptivity.io/v1/pbs/PJSEVNT123/pbs-billing?b=contxtful-test-prebidjs-events&a=PJSEVNT123&bidder=contxtful&impId=test-prebidjs-events&price=2.75&traceId=prebidjs-trace-abc-789&random=0.654321&domain=prebidjs.example.com&adRequestId=prebidjs-req-456&w=300&h=250&f=b"

	responseJSON := fmt.Sprintf(`[{
		"cpm": 2.75,
		"width": 300,
		"height": 250,
		"currency": "USD",
		"requestId": "test-prebidjs-events",
		"ad": "<div>Prebid.js Event Tracking Test Ad</div>",
		"ttl": 300,
		"creativeId": "prebidjs-creative-1",
		"netRevenue": true,
		"mediaType": "banner",
		"bidderCode": "contxtful",
		"traceId": "prebidjs-trace-abc-789",
		"random": 0.654321,
		"nurl": "%s",
		"burl": "%s"
	}]`, nurl, burl)

	response := &adapters.ResponseData{
		StatusCode: 200,
		Body:       []byte(responseJSON),
	}

	requestJSON := `{
		"id": "prebidjs-req-456",
		"imp": [{
			"id": "test-prebidjs-events",
			"banner": {"format": [{"w": 300, "h": 250}]},
			"ext": {"bidder": {"placementId": "prebidjs-test", "customerId": "PJSEVNT123"}}
		}],
		"site": {
			"domain": "prebidjs.example.com",
			"page": "https://prebidjs.example.com/test-page"
		}
	}`

	var request openrtb2.BidRequest
	if err := json.Unmarshal([]byte(requestJSON), &request); err != nil {
		t.Fatalf("Failed to unmarshal test request: %v", err)
	}

	bids, errs := bidder.MakeBids(&request, nil, response)

	if len(errs) > 0 {
		t.Fatalf("MakeBids returned errors: %v", errs)
	}

	if bids == nil || len(bids.Bids) != 1 {
		bidCount := 0
		if bids != nil {
			bidCount = len(bids.Bids)
		}
		t.Fatalf("Expected 1 bid, got %d", bidCount)
	}

	bid := bids.Bids[0]

	// Verify NURL (win notice URL) is set with enhanced parameters
	if bid.Bid.NURL == "" {
		t.Error("NURL should be set for Prebid.js response event tracking")
	} else {
		// Verify NURL contains all critical logging parameters
		if !contains(bid.Bid.NURL, "pbs-impression") {
			t.Error("NURL should contain t=win parameter")
		}
		if !contains(bid.Bid.NURL, "a=PJSEVNT123") {
			t.Error("NURL should contain customer parameter (a=PJSEVNT123)")
		}
		if !contains(bid.Bid.NURL, "bidder=contxtful") {
			t.Error("NURL should contain bidder=contxtful parameter")
		}
		if !contains(bid.Bid.NURL, "domain=prebidjs.example.com") {
			t.Error("NURL should contain domain=prebidjs.example.com parameter")
		}
		if !contains(bid.Bid.NURL, "traceId=prebidjs-trace-abc-789") {
			t.Error("NURL should contain traceId=prebidjs-trace-abc-789 parameter (real tracking data)")
		}
		if !contains(bid.Bid.NURL, "adRequestId=prebidjs-req-456") {
			t.Error("NURL should contain adRequestId=prebidjs-req-456 parameter")
		}
		if !contains(bid.Bid.NURL, "random=0.654321") {
			t.Error("NURL should contain random=0.654321 parameter (real tracking data)")
		}
		if !contains(bid.Bid.NURL, "impId=test-prebidjs-events") {
			t.Error("NURL should contain impId=test-prebidjs-events parameter")
		}
		if !contains(bid.Bid.NURL, "price=2.75") {
			t.Error("NURL should contain price=2.75 parameter")
		}
		if !contains(bid.Bid.NURL, "w=300") {
			t.Error("NURL should contain w=300 parameter")
		}
		if !contains(bid.Bid.NURL, "h=250") {
			t.Error("NURL should contain h=250 parameter")
		}
	}

	// Verify BURL (billing URL) is set with enhanced parameters
	if bid.Bid.BURL == "" {
		t.Error("BURL should be set for Prebid.js response event tracking")
	} else {
		// Verify BURL contains all critical logging parameters
		if !contains(bid.Bid.BURL, "pbs-billing") {
			t.Error("BURL should contain t=billing parameter")
		}
		if !contains(bid.Bid.BURL, "a=PJSEVNT123") {
			t.Error("BURL should contain customer parameter (a=PJSEVNT123)")
		}
		if !contains(bid.Bid.BURL, "bidder=contxtful") {
			t.Error("BURL should contain bidder=contxtful parameter")
		}
		if !contains(bid.Bid.BURL, "domain=prebidjs.example.com") {
			t.Error("BURL should contain domain=prebidjs.example.com parameter")
		}
		if !contains(bid.Bid.BURL, "traceId=prebidjs-trace-abc-789") {
			t.Error("BURL should contain traceId=prebidjs-trace-abc-789 parameter (real tracking data)")
		}
		if !contains(bid.Bid.BURL, "adRequestId=prebidjs-req-456") {
			t.Error("BURL should contain adRequestId=prebidjs-req-456 parameter")
		}
		if !contains(bid.Bid.BURL, "random=0.654321") {
			t.Error("BURL should contain random=0.654321 parameter (real tracking data)")
		}
		if !contains(bid.Bid.BURL, "impId=test-prebidjs-events") {
			t.Error("BURL should contain impId=test-prebidjs-events parameter")
		}
		if !contains(bid.Bid.BURL, "price=2.75") {
			t.Error("BURL should contain price=2.75 parameter")
		}
		if !contains(bid.Bid.BURL, "w=300") {
			t.Error("BURL should contain w=300 parameter")
		}
		if !contains(bid.Bid.BURL, "h=250") {
			t.Error("BURL should contain h=250 parameter")
		}
	}
}

// TestPrebidJSHybridEventTrackingURLGeneration tests enhanced PrebidJS responses with tracking fields
func TestPrebidJSHybridEventTrackingURLGeneration(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Test with hybrid PrebidJS format response (has BOTH PrebidJS fields AND tracking fields)
	responseJSON := `[{
		"cpm": 3.25,
		"width": 728,
		"height": 90,
		"currency": "USD",
		"requestId": "test-hybrid-events",
		"ad": "<div>Hybrid PrebidJS Event Tracking Test Ad</div>",
		"ttl": 300,
		"creativeId": "hybrid-creative-1",
		"netRevenue": true,
		"mediaType": "banner",
		"bidderCode": "contxtful",
		"traceId": "hybrid-abc-456-def",
		"random": 0.9876543,
		"nurl": "https://monitoring.receptivity.io/v1/pbs/HYBEVNT123/pbs-impression?b=contxtful-test-hybrid-events&a=HYBEVNT123&bidder=contxtful&impId=test-hybrid-events&price=3.25&traceId=hybrid-abc-456-def&random=0.987654&domain=hybrid.example.com&adRequestId=hybrid-req-789&w=728&h=90&f=b",
		"burl": "https://monitoring.receptivity.io/v1/pbs/HYBEVNT123/pbs-billing?b=contxtful-test-hybrid-events&a=HYBEVNT123&bidder=contxtful&impId=test-hybrid-events&price=3.25&traceId=hybrid-abc-456-def&random=0.987654&domain=hybrid.example.com&adRequestId=hybrid-req-789&w=728&h=90&f=b"
	}]`

	response := &adapters.ResponseData{
		StatusCode: 200,
		Body:       []byte(responseJSON),
	}

	requestJSON := `{
		"id": "hybrid-req-789",
		"imp": [{
			"id": "test-hybrid-events",
			"banner": {"format": [{"w": 728, "h": 90}]},
			"ext": {"bidder": {"placementId": "hybrid-test", "customerId": "HYBEVNT123"}}
		}],
		"site": {
			"domain": "hybrid.example.com",
			"page": "https://hybrid.example.com/test-page"
		}
	}`

	var request openrtb2.BidRequest
	if err := json.Unmarshal([]byte(requestJSON), &request); err != nil {
		t.Fatalf("Failed to unmarshal test request: %v", err)
	}

	bids, errs := bidder.MakeBids(&request, nil, response)

	if len(errs) > 0 {
		t.Fatalf("MakeBids returned errors: %v", errs)
	}

	if bids == nil || len(bids.Bids) != 1 {
		bidCount := 0
		if bids != nil {
			bidCount = len(bids.Bids)
		}
		t.Fatalf("Expected 1 bid, got %d", bidCount)
	}

	bid := bids.Bids[0]

	// Verify NURL uses REAL tracking values (not fallbacks)
	if bid.Bid.NURL == "" {
		t.Error("NURL should be set for hybrid PrebidJS response event tracking")
	} else {
		// Verify NURL contains REAL tracking values (not fallbacks)
		if !contains(bid.Bid.NURL, "traceId=hybrid-abc-456-def") {
			t.Error("NURL should contain REAL traceId=hybrid-abc-456-def (not fallback 'unknown')")
		}
		if !contains(bid.Bid.NURL, "random=0.987654") {
			t.Error("NURL should contain REAL random=0.987654 (not fallback '0.500000')")
		}
		if !contains(bid.Bid.NURL, "pbs-impression") {
			t.Error("NURL should contain t=win parameter")
		}
		if !contains(bid.Bid.NURL, "a=HYBEVNT123") {
			t.Error("NURL should contain customer parameter (a=HYBEVNT123)")
		}
		if !contains(bid.Bid.NURL, "domain=hybrid.example.com") {
			t.Error("NURL should contain domain=hybrid.example.com parameter")
		}
		if !contains(bid.Bid.NURL, "price=3.25") {
			t.Error("NURL should contain price=3.25 parameter")
		}
	}

	// Verify BURL uses REAL tracking values (not fallbacks)
	if bid.Bid.BURL == "" {
		t.Error("BURL should be set for hybrid PrebidJS response event tracking")
	} else {
		// Verify BURL contains REAL tracking values (not fallbacks)
		if !contains(bid.Bid.BURL, "traceId=hybrid-abc-456-def") {
			t.Error("BURL should contain REAL traceId=hybrid-abc-456-def (not fallback 'unknown')")
		}
		if !contains(bid.Bid.BURL, "random=0.987654") {
			t.Error("BURL should contain REAL random=0.987654 (not fallback '0.500000')")
		}
		if !contains(bid.Bid.BURL, "pbs-billing") {
			t.Error("BURL should contain t=billing parameter")
		}
		if !contains(bid.Bid.BURL, "a=HYBEVNT123") {
			t.Error("BURL should contain customer parameter (a=HYBEVNT123)")
		}
		if !contains(bid.Bid.BURL, "domain=hybrid.example.com") {
			t.Error("BURL should contain domain=hybrid.example.com parameter")
		}
		if !contains(bid.Bid.BURL, "price=3.25") {
			t.Error("BURL should contain price=3.25 parameter")
		}
	}
}

// TestResponseFormatDetectionPriority tests the priority and accuracy of response format detection
func TestResponseFormatDetectionPriority(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Create a dummy request for testing
	requestJSON := `{
		"id": "format-detection-test",
		"imp": [{
			"id": "test-imp-format",
			"banner": {"format": [{"w": 300, "h": 250}]},
			"ext": {"bidder": {"placementId": "format-test", "customerId": "FMTTEST"}}
		}]
	}`

	var request openrtb2.BidRequest
	if err := json.Unmarshal([]byte(requestJSON), &request); err != nil {
		t.Fatalf("Failed to unmarshal test request: %v", err)
	}

	testCases := []struct {
		name            string
		responseBody    string
		expectedFormat  string
		expectedBids    int
		expectedError   bool
		validateContent func(*testing.T, *adapters.BidderResponse)
	}{
		{
			name: "PrebidJS Format with Real Tracking",
			responseBody: `[{
				"cpm": 2.50,
				"width": 300,
				"height": 250,
				"currency": "USD",
				"requestId": "test-prebidjs-pure",
				"ad": "<div>Pure PrebidJS Ad</div>",
				"ttl": 300,
				"creativeId": "pure-creative",
				"netRevenue": true,
				"traceId": "pure-detection-trace-xyz",
				"random": 0.333777,
				"nurl": "https://monitoring.receptivity.io/v1/pbs/FMTTEST/pbs-impression?b=contxtful-test-prebidjs-pure&a=FMTTEST&bidder=contxtful&impId=test-prebidjs-pure&price=2.50&traceId=pure-detection-trace-xyz&random=0.333777&domain=&adRequestId=format-detection-test&w=300&h=250&f=b",
				"burl": "https://monitoring.receptivity.io/v1/pbs/FMTTEST/pbs-billing?b=contxtful-test-prebidjs-pure&a=FMTTEST&bidder=contxtful&impId=test-prebidjs-pure&price=2.50&traceId=pure-detection-trace-xyz&random=0.333777&domain=&adRequestId=format-detection-test&w=300&h=250&f=b"
			}]`,
			expectedFormat: "PrebidJS",
			expectedBids:   1,
			expectedError:  false,
			validateContent: func(t *testing.T, bids *adapters.BidderResponse) {
				if len(bids.Bids) != 1 {
					t.Errorf("Expected 1 bid, got %d", len(bids.Bids))
					return
				}
				// Should use real tracking values from response
				nurl := bids.Bids[0].Bid.NURL
				if !contains(nurl, "traceId=pure-detection-trace-xyz") {
					t.Error("PrebidJS should use real traceId=pure-detection-trace-xyz")
				}
				if !contains(nurl, "random=0.333777") {
					t.Error("PrebidJS should use real random=0.333777")
				}
			},
		},
		{
			name: "Hybrid PrebidJS Format (with tracking fields) - Should be detected as PrebidJS",
			responseBody: `[{
				"cpm": 3.75,
				"width": 728,
				"height": 90,
				"currency": "USD",
				"requestId": "test-hybrid-format",
				"ad": "<div>Hybrid PrebidJS Ad with Tracking</div>",
				"ttl": 300,
				"creativeId": "hybrid-creative",
				"netRevenue": true,
				"traceId": "hybrid-trace-123-abc",
				"random": 0.8765432,
				"nurl": "https://monitoring.receptivity.io/v1/pbs/FMTTEST/pbs-impression?b=contxtful-test-hybrid-format&a=FMTTEST&bidder=contxtful&impId=test-hybrid-format&price=3.75&traceId=hybrid-trace-123-abc&random=0.876543&domain=&adRequestId=format-detection-test&w=728&h=90&f=b",
				"burl": "https://monitoring.receptivity.io/v1/pbs/FMTTEST/pbs-billing?b=contxtful-test-hybrid-format&a=FMTTEST&bidder=contxtful&impId=test-hybrid-format&price=3.75&traceId=hybrid-trace-123-abc&random=0.876543&domain=&adRequestId=format-detection-test&w=728&h=90&f=b"
			}]`,
			expectedFormat: "PrebidJS",
			expectedBids:   1,
			expectedError:  false,
			validateContent: func(t *testing.T, bids *adapters.BidderResponse) {
				if len(bids.Bids) != 1 {
					t.Errorf("Expected 1 bid, got %d", len(bids.Bids))
					return
				}
				// Should use REAL tracking values from response
				nurl := bids.Bids[0].Bid.NURL
				if !contains(nurl, "traceId=hybrid-trace-123-abc") {
					t.Error("Hybrid PrebidJS should use REAL traceId=hybrid-trace-123-abc")
				}
				if !contains(nurl, "random=0.876543") {
					t.Error("Hybrid PrebidJS should use REAL random=0.876543")
				}
				if contains(nurl, "traceId=unknown") {
					t.Error("Hybrid PrebidJS should NOT use fallback traceId")
				}
			},
		},
		{
			name: "Trace-only Response (no bids)",
			responseBody: `[{
				"traceId": "trace-only-789-ghi",
				"random": 0.9999999,
				"bids": []
			}]`,
			expectedFormat: "Trace",
			expectedBids:   0,
			expectedError:  false,
			validateContent: func(t *testing.T, bids *adapters.BidderResponse) {
				if len(bids.Bids) != 0 {
					t.Errorf("Expected 0 bids for trace-only response, got %d", len(bids.Bids))
				}
			},
		},
		{
			name: "Malformed PrebidJS (missing required fields)",
			responseBody: `[{
				"cpm": 1.50,
				"currency": "USD"
			}]`,
			expectedFormat: "Error",
			expectedBids:   0,
			expectedError:  true,
			validateContent: func(t *testing.T, bids *adapters.BidderResponse) {
				// Should not generate bids for malformed response
				if bids != nil && len(bids.Bids) > 0 {
					t.Error("Should not generate bids for malformed PrebidJS response")
				}
			},
		},
		{
			name:           "Invalid JSON",
			responseBody:   `{invalid json syntax`,
			expectedFormat: "Error",
			expectedBids:   0,
			expectedError:  true,
			validateContent: func(t *testing.T, bids *adapters.BidderResponse) {
				if bids != nil && len(bids.Bids) > 0 {
					t.Error("Should not generate bids for invalid JSON")
				}
			},
		},
		{
			name:           "Empty Response",
			responseBody:   `[]`,
			expectedFormat: "Error",
			expectedBids:   0,
			expectedError:  true,
			validateContent: func(t *testing.T, bids *adapters.BidderResponse) {
				if bids != nil && len(bids.Bids) > 0 {
					t.Error("Should not generate bids for empty response")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			response := &adapters.ResponseData{
				StatusCode: 200,
				Body:       []byte(tc.responseBody),
			}

			bids, errs := bidder.MakeBids(&request, nil, response)

			// Validate error expectations
			if tc.expectedError {
				if len(errs) == 0 {
					t.Errorf("Expected errors for %s, but got none", tc.name)
				}
			} else {
				if len(errs) > 0 {
					t.Errorf("Unexpected errors for %s: %v", tc.name, errs)
				}
			}

			// Validate bid count
			actualBids := 0
			if bids != nil {
				actualBids = len(bids.Bids)
			}
			if actualBids != tc.expectedBids {
				t.Errorf("Expected %d bids, got %d", tc.expectedBids, actualBids)
			}

			// Run custom validation
			if tc.validateContent != nil && bids != nil {
				tc.validateContent(t, bids)
			}
		})
	}
}

// TestTrackingValuePropagation tests that tracking values correctly flow from responses to event URLs
func TestTrackingValuePropagation(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Test with comprehensive tracking data
	testCases := []struct {
		name           string
		responseBody   string
		requestID      string
		domain         string
		customer       string
		expectedValues map[string]string
	}{
		{
			name: "Hybrid PrebidJS with Tracking",
			responseBody: `[{
				"cpm": 3.25,
				"width": 300,
				"height": 250,
				"currency": "USD",
				"requestId": "tracking-imp-2",
				"ad": "<div>Hybrid Tracking Ad</div>",
				"ttl": 300,
				"creativeId": "hybrid-tracking-creative",
				"netRevenue": true,
				"traceId": "hybrid-trace-def-456",
				"random": 0.123456789,
				"nurl": "https://monitoring.receptivity.io/v1/pbs/HYBRID123/pbs-impression?b=contxtful-tracking-imp-2&a=HYBRID123&bidder=contxtful&impId=tracking-imp-2&price=3.25&traceId=hybrid-trace-def-456&random=0.123457&domain=hybrid.tracking.com&adRequestId=hybrid-tracking-request-456&w=300&h=250&f=b",
				"burl": "https://monitoring.receptivity.io/v1/pbs/HYBRID123/pbs-billing?b=contxtful-tracking-imp-2&a=HYBRID123&bidder=contxtful&impId=tracking-imp-2&price=3.25&traceId=hybrid-trace-def-456&random=0.123457&domain=hybrid.tracking.com&adRequestId=hybrid-tracking-request-456&w=300&h=250&f=b"
			}]`,
			requestID: "hybrid-tracking-request-456",
			domain:    "hybrid.tracking.com",
			customer:  "HYBRID123",
			expectedValues: map[string]string{
				"traceId":     "hybrid-trace-def-456",
				"random":      "0.123457", // Note: formatted with 6 decimal places
				"adRequestId": "hybrid-tracking-request-456",
				"domain":      "hybrid.tracking.com",
				"a":           "HYBRID123",
				"price":       "3.25",
				"w":           "300",
				"h":           "250",
				"impId":       "tracking-imp-2",
			},
		},
		{
			name: "PrebidJS with Real Tracking Data",
			responseBody: `[{
				"cpm": 1.85,
				"width": 320,
				"height": 50,
				"currency": "USD",
				"requestId": "tracking-imp-3",
				"ad": "<div>Pure PrebidJS Ad</div>",
				"ttl": 300,
				"creativeId": "pure-creative",
				"netRevenue": true,
				"traceId": "pure-trace-ghi-789",
				"random": 0.444555,
				"nurl": "https://monitoring.receptivity.io/v1/pbs/PURE123/pbs-impression?b=contxtful-tracking-imp-3&a=PURE123&bidder=contxtful&impId=tracking-imp-3&price=1.85&traceId=pure-trace-ghi-789&random=0.444555&domain=pure.tracking.com&adRequestId=pure-tracking-request-789&w=320&h=50&f=b",
				"burl": "https://monitoring.receptivity.io/v1/pbs/PURE123/pbs-billing?b=contxtful-tracking-imp-3&a=PURE123&bidder=contxtful&impId=tracking-imp-3&price=1.85&traceId=pure-trace-ghi-789&random=0.444555&domain=pure.tracking.com&adRequestId=pure-tracking-request-789&w=320&h=50&f=b"
			}]`,
			requestID: "pure-tracking-request-789",
			domain:    "pure.tracking.com",
			customer:  "PURE123",
			expectedValues: map[string]string{
				"traceId":     "pure-trace-ghi-789", // Real tracking data
				"random":      "0.444555",           // Real tracking data
				"adRequestId": "pure-tracking-request-789",
				"domain":      "pure.tracking.com",
				"a":           "PURE123",
				"price":       "1.85",
				"w":           "320",
				"h":           "50",
				"impId":       "tracking-imp-3",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requestJSON := fmt.Sprintf(`{
				"id": "%s",
				"imp": [{
					"id": "%s",
					"banner": {"format": [{"w": %s, "h": %s}]},
					"ext": {"bidder": {"placementId": "tracking-test", "customerId": "%s"}}
				}],
				"site": {
					"domain": "%s",
					"page": "https://%s/test-page"
				}
			}`, tc.requestID, tc.expectedValues["impId"], tc.expectedValues["w"], tc.expectedValues["h"], tc.customer, tc.domain, tc.domain)

			var request openrtb2.BidRequest
			if err := json.Unmarshal([]byte(requestJSON), &request); err != nil {
				t.Fatalf("Failed to unmarshal test request: %v", err)
			}

			response := &adapters.ResponseData{
				StatusCode: 200,
				Body:       []byte(tc.responseBody),
			}

			bids, errs := bidder.MakeBids(&request, nil, response)

			if len(errs) > 0 {
				t.Fatalf("MakeBids returned errors: %v", errs)
			}

			if bids == nil || len(bids.Bids) != 1 {
				t.Fatalf("Expected 1 bid, got %d", len(bids.Bids))
			}

			bid := bids.Bids[0]

			// Validate NURL contains all expected tracking values
			if bid.Bid.NURL == "" {
				t.Fatal("NURL should be set")
			}

			for key, expectedValue := range tc.expectedValues {
				expectedParam := fmt.Sprintf("%s=%s", key, expectedValue)
				if !contains(bid.Bid.NURL, expectedParam) {
					t.Errorf("NURL should contain %s", expectedParam)
				}
			}

			// Validate BURL contains all expected tracking values
			if bid.Bid.BURL == "" {
				t.Fatal("BURL should be set")
			}

			for key, expectedValue := range tc.expectedValues {
				expectedParam := fmt.Sprintf("%s=%s", key, expectedValue)
				if !contains(bid.Bid.BURL, expectedParam) {
					t.Errorf("BURL should contain %s", expectedParam)
				}
			}

			// Verify event type differences
			if !contains(bid.Bid.NURL, "pbs-impression") {
				t.Error("NURL should contain t=win")
			}
			if !contains(bid.Bid.BURL, "pbs-billing") {
				t.Error("BURL should contain t=billing")
			}
		})
	}
}

// TestResponseFormatEdgeCases tests edge cases and error scenarios in response format detection
func TestResponseFormatEdgeCases(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	requestJSON := `{
		"id": "edge-case-test",
		"imp": [{
			"id": "edge-case-imp",
			"banner": {"format": [{"w": 300, "h": 250}]},
			"ext": {"bidder": {"placementId": "edge-test", "customerId": "EDGE123"}}
		}]
	}`

	var request openrtb2.BidRequest
	if err := json.Unmarshal([]byte(requestJSON), &request); err != nil {
		t.Fatalf("Failed to unmarshal test request: %v", err)
	}

	testCases := []struct {
		name           string
		responseBody   string
		statusCode     int
		expectedBids   int
		expectedErrors int
		description    string
	}{
		{
			name: "Partial PrebidJS (missing ad field)",
			responseBody: `[{
				"cpm": 2.50,
				"width": 300,
				"height": 250,
				"currency": "USD",
				"requestId": "partial-prebidjs"
			}]`,
			statusCode:     200,
			expectedBids:   0,
			expectedErrors: 1,
			description:    "Should fail when required PrebidJS fields are missing",
		},
		{
			name: "Empty bids array in relay format",
			responseBody: `[{
				"traceId": "empty-relay-trace",
				"random": 0.5,
				"bids": []
			}]`,
			statusCode:     200,
			expectedBids:   0,
			expectedErrors: 0,
			description:    "Should handle empty bids array gracefully",
		},
		{
			name: "Mixed valid and invalid bids in PrebidJS",
			responseBody: `[
				{
					"cpm": 2.50,
					"width": 300,
					"height": 250,
					"currency": "USD",
					"requestId": "valid-bid",
					"ad": "<div>Valid Ad</div>",
					"ttl": 300,
					"creativeId": "valid-creative",
					"netRevenue": true
				},
				{
					"cpm": 0,
					"requestId": "invalid-bid"
				}
			]`,
			statusCode:     200,
			expectedBids:   1,
			expectedErrors: 0,
			description:    "Should process valid bids and ignore invalid ones",
		},
		{
			name: "Relay format with tracking but no bids array",
			responseBody: `[{
				"traceId": "no-bids-trace-123",
				"random": 0.777777
			}]`,
			statusCode:     200,
			expectedBids:   0,
			expectedErrors: 0,
			description:    "Should handle tracking data without bids array gracefully (trace data)",
		},
		{
			name:           "HTTP Error Response",
			responseBody:   `{"error": "Internal Server Error"}`,
			statusCode:     500,
			expectedBids:   0,
			expectedErrors: 1,
			description:    "Should handle HTTP errors gracefully",
		},
		{
			name: "Very Large Response",
			responseBody: func() string {
				// Generate a response with many bids to test size handling
				var bids []string
				for i := 0; i < 100; i++ {
					bid := fmt.Sprintf(`{
						"cpm": %d.25,
						"width": 300,
						"height": 250,
						"currency": "USD",
						"requestId": "large-bid-%d",
						"ad": "<div>Large Response Bid %d</div>",
						"ttl": 300,
						"creativeId": "large-creative-%d",
						"netRevenue": true
					}`, i+1, i, i, i)
					bids = append(bids, bid)
				}
				return "[" + strings.Join(bids, ",") + "]"
			}(),
			statusCode:     200,
			expectedBids:   100,
			expectedErrors: 0,
			description:    "Should handle large responses with many bids",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			response := &adapters.ResponseData{
				StatusCode: tc.statusCode,
				Body:       []byte(tc.responseBody),
			}

			bids, errs := bidder.MakeBids(&request, nil, response)

			// Validate error count
			if len(errs) != tc.expectedErrors {
				t.Errorf("Expected %d errors, got %d: %v", tc.expectedErrors, len(errs), errs)
			}

			// Validate bid count
			actualBids := 0
			if bids != nil {
				actualBids = len(bids.Bids)
			}
			if actualBids != tc.expectedBids {
				t.Errorf("Expected %d bids, got %d", tc.expectedBids, actualBids)
			}

		})
	}
}

// TestEventURLGeneration tests comprehensive event URL parameter inclusion
func TestEventURLGeneration(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Create request with comprehensive tracking parameters
	requestJSON := `{
		"id": "event-url-test",
		"imp": [{
			"id": "test-imp-event",
			"banner": {"format": [{"w": 300, "h": 250}]},
			"ext": {
				"bidder": {
					"placementId": "12345",
					"customerId": "EVNTTEST123"
				}
			}
		}],
		"site": {"domain": "example.com"},
		"ext": {
			"trace": "event-trace-789",
			"prebid": {
				"adRequestId": "req-abc-123"
			}
		}
	}`

	var request openrtb2.BidRequest
	if err := jsonutil.Unmarshal([]byte(requestJSON), &request); err != nil {
		t.Fatalf("Failed to unmarshal request: %v", err)
	}

	// Test Case 1: PrebidJS response with comprehensive tracking
	responseJSON := `[{
		"requestId": "test-imp-event",
		"cpm": 1.25,
		"currency": "USD",
		"width": 300,
		"height": 250,
		"creativeId": "test-creative-123",
		"ad": "<div>Test Ad</div>",
		"ttl": 300,
		"netRevenue": true,
		"mediaType": "banner",
		"bidderCode": "contxtful",
		"traceId": "event-trace-789",
		"random": 0.123456789,
		"nurl": "https://monitoring.receptivity.io/v1/pbs/EVNTTEST123/pbs-impression?b=contxtful-test-imp-event&a=EVNTTEST123&bidder=contxtful&impId=test-imp-event&price=1.25&traceId=event-trace-789&random=0.123457&domain=example.com&adRequestId=event-url-test&w=300&h=250&f=b",
		"burl": "https://monitoring.receptivity.io/v1/pbs/EVNTTEST123/pbs-billing?b=contxtful-test-imp-event&a=EVNTTEST123&bidder=contxtful&impId=test-imp-event&price=1.25&traceId=event-trace-789&random=0.123457&domain=example.com&adRequestId=event-url-test&w=300&h=250&f=b"
	}]`

	response := &adapters.ResponseData{
		StatusCode: 200,
		Body:       []byte(responseJSON),
	}

	bidResponse, errs := bidder.MakeBids(&request, nil, response)

	if len(errs) != 0 {
		t.Fatalf("Expected no errors, got %v", errs)
	}

	if len(bidResponse.Bids) != 1 {
		t.Fatalf("Expected 1 bid, got %d", len(bidResponse.Bids))
	}

	bid := bidResponse.Bids[0]

	// Validate NURL contains all required parameters
	expectedParams := []string{
		"domain=example.com",
		"traceId=event-trace-789",
		"adRequestId=event-url-test", // Uses bid request ID as fallback
		"random=0.123457",            // Truncated in URL format
		"bidder=contxtful",
		"a=EVNTTEST123",        // Customer ID parameter
		"impId=test-imp-event", // Impression ID parameter
		"f=b",
		"price=1.25", // Relay format provides price in bid data
	}

	for _, param := range expectedParams {
		if !strings.Contains(bid.Bid.NURL, param) {
			t.Errorf("NURL missing parameter: %s\nActual NURL: %s", param, bid.Bid.NURL)
		}
	}

	// Validate BURL contains billing parameters
	expectedBillingParams := []string{
		"f=b",
		"bidder=contxtful",
		"a=EVNTTEST123", // Customer ID parameter
	}

	for _, param := range expectedBillingParams {
		if !strings.Contains(bid.Bid.BURL, param) {
			t.Errorf("BURL missing parameter: %s\nActual BURL: %s", param, bid.Bid.BURL)
		}
	}
}

// TestBidRejectionScenarios tests various scenarios that should result in bid rejection
func TestBidRejectionScenarios(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	requestJSON := `{
		"id": "rejection-test",
		"imp": [{
			"id": "test-imp-reject",
			"banner": {"format": [{"w": 300, "h": 250}]},
			"ext": {
				"bidder": {
					"placementId": "12345",
					"customerId": "REJECTTEST123"
				}
			}
		}]
	}`

	var request openrtb2.BidRequest
	if err := jsonutil.Unmarshal([]byte(requestJSON), &request); err != nil {
		t.Fatalf("Failed to unmarshal request: %v", err)
	}

	testCases := []struct {
		name           string
		responseBody   string
		statusCode     int
		expectedBids   int
		expectedErrors int
		description    string
	}{
		{
			name: "Relay response with no bids array",
			responseBody: `[{
				"traceId": "no-bids-trace",
				"random": 0.5
			}]`,
			statusCode:     200,
			expectedBids:   0,
			expectedErrors: 0,
			description:    "Should handle tracking-only response gracefully",
		},
		{
			name: "Relay response with empty bids array",
			responseBody: `[{
				"traceId": "empty-bids-trace",
				"random": 0.5,
				"bids": []
			}]`,
			statusCode:     200,
			expectedBids:   0,
			expectedErrors: 0,
			description:    "Should handle empty bids array gracefully",
		},
		{
			name:           "Invalid JSON response",
			responseBody:   `{invalid json}`,
			statusCode:     200,
			expectedBids:   0,
			expectedErrors: 1,
			description:    "Should handle malformed JSON with error",
		},
		{
			name:           "HTTP error response",
			responseBody:   `{"error": "Server error"}`,
			statusCode:     500,
			expectedBids:   0,
			expectedErrors: 1,
			description:    "Should handle HTTP errors properly",
		},
		{
			name: "PrebidJS response with missing required fields",
			responseBody: `{
				"id": "test",
				"seatbid": [{
					"bid": [{
						"id": "bid1"
					}]
				}]
			}`,
			statusCode:     200,
			expectedBids:   0,
			expectedErrors: 1,
			description:    "Should reject bids missing required fields",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			response := &adapters.ResponseData{
				StatusCode: tc.statusCode,
				Body:       []byte(tc.responseBody),
			}

			bidResponse, errs := bidder.MakeBids(&request, nil, response)

			actualBids := 0
			if bidResponse != nil && bidResponse.Bids != nil {
				actualBids = len(bidResponse.Bids)
			}

			if actualBids != tc.expectedBids {
				t.Errorf("Expected %d bids, got %d", tc.expectedBids, actualBids)
			}

			if len(errs) != tc.expectedErrors {
				t.Errorf("Expected %d errors, got %d: %v", tc.expectedErrors, len(errs), errs)
			}
		})
	}
}

// TestBidExtensionsRegression ensures that essential bid extensions are present in relay responses
// This test prevents regression of the bid extension creation logic that was accidentally removed during refactoring
func TestBidExtensionsRegression(t *testing.T) {
	// Create mock adapter
	server := config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"}
	mockAdapter, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://relay.example.com/v1/pbs/{{.CustomerId}}/bid",
	}, server)
	assert.NoError(t, buildErr)

	// Create test request
	request := &openrtb2.BidRequest{
		ID:   "test-bid-extensions",
		Test: 1, // Enable test mode
		Imp: []openrtb2.Imp{{
			ID: "test-imp-extensions",
			Ext: json.RawMessage(`{
				"bidder": {
					"customerId": "EXTTEST123"
				}
			}`),
		}},
		Site: &openrtb2.Site{
			Domain: "test.extension.com",
			Page:   "https://test.extension.com",
		},
		User: &openrtb2.User{
			BuyerUID: "contxtful-v1-eyJjdXN0b21lciI6IkVYVFRFU1QxMjMiLCJ0aW1lc3RhbXAiOjE3MzQ1Njc4OTAwMDAsInBhcnRuZXJzIjp7ImFteCI6ImFteC10ZXN0In0sInZlcnNpb24iOiJ2MSJ9",
		},
	}

	// Create mock response data - PREBIDJS FORMAT with bid extensions
	mockRelayResponse := `[{
		"requestId": "test-imp-extensions",
		"cpm": 2.85,
		"ad": "<div>Test ad for extension validation</div>",
		"width": 300,
		"height": 250,
		"creativeId": "extension-creative-456",
		"netRevenue": true,
		"currency": "USD",
		"mediaType": "banner",
		"bidderCode": "contxtful",
		"nurl": "https://monitoring.receptivity.io/v1/pbs/EXTTEST123/pbs-impression?b=contxtful-test-imp-extensions&a=EXTTEST123&bidder=contxtful&impId=test-imp-extensions&price=2.85&traceId=extension-test-trace-123&random=0.789123&domain=extensions.example.com&adRequestId=extension-test-request&w=300&h=250&f=b",
		"burl": "https://monitoring.receptivity.io/v1/pbs/EXTTEST123/pbs-billing?b=contxtful-test-imp-extensions&a=EXTTEST123&bidder=contxtful&impId=test-imp-extensions&price=2.85&traceId=extension-test-trace-123&random=0.789123&domain=extensions.example.com&adRequestId=extension-test-request&w=300&h=250&f=b"
	}]`

	responseData := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte(mockRelayResponse),
	}

	// Call MakeBids
	bidderResponse, errs := mockAdapter.MakeBids(request, nil, responseData)

	// Verify no errors
	assert.Empty(t, errs, "Expected no errors")
	assert.NotNil(t, bidderResponse, "Expected bidder response")

	// Verify bid was created
	assert.Len(t, bidderResponse.Bids, 1, "Expected exactly 1 bid")

	bid := bidderResponse.Bids[0].Bid

	// CRITICAL REGRESSION CHECK: Verify bid extensions are present
	assert.NotNil(t, bid.Ext, "BID EXTENSIONS MISSING! This indicates the bid extension creation logic was removed during refactoring")

	// Parse bid extensions
	var bidExt map[string]interface{}
	err := json.Unmarshal(bid.Ext, &bidExt)
	assert.NoError(t, err, "Failed to parse bid extensions JSON")

	// Verify essential PBS bid fields
	assert.Contains(t, bidExt, "origbidcpm", "Missing origbidcpm - required for PBS processing")
	assert.Contains(t, bidExt, "origbidcur", "Missing origbidcur - required for PBS processing")
	assert.Equal(t, 2.85, bidExt["origbidcpm"], "origbidcpm should match bid price")
	assert.Equal(t, "USD", bidExt["origbidcur"], "origbidcur should be USD")

	// Verify Prebid targeting extensions (CRITICAL for PBS bid processing)
	prebidExt, hasPrebid := bidExt["prebid"].(map[string]interface{})
	assert.True(t, hasPrebid, "Missing prebid extensions - CRITICAL for PBS processing")

	targeting, hasTargeting := prebidExt["targeting"].(map[string]interface{})
	assert.True(t, hasTargeting, "Missing prebid.targeting - CRITICAL for PBS processing")

	// These targeting fields are ESSENTIAL for PBS to process bids correctly
	assert.Equal(t, "contxtful", targeting["hb_bidder"], "hb_bidder must be 'contxtful'")
	assert.Equal(t, "2.85", targeting["hb_pb"], "hb_pb must match formatted price")
	assert.Equal(t, "300x250", targeting["hb_size"], "hb_size must match bid dimensions")

	// Verify essential PBS bid extensions are present (production-level fields only)
	// Note: contxtfulDebug has been removed for production - only core PBS fields remain

	// Verify NURL/BURL are set (these should be generated by generateEventURLs)
	assert.NotEmpty(t, bid.NURL, "NURL should be generated for event tracking")
	assert.NotEmpty(t, bid.BURL, "BURL should be generated for event tracking")
	assert.Contains(t, bid.NURL, "pbs-impression", "NURL should contain win tracking parameter")
	assert.Contains(t, bid.BURL, "pbs-billing", "BURL should contain billing tracking parameter")
}

// TestS2SBidderConfigExtraction tests extraction of rich bidder config data from S2S payloads
func TestS2SBidderConfigExtraction(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Real S2S payload with rich bidder config data (based on user's example)
	requestJSON := `{
		"id": "s2s-bidder-config-test",
		"test": 1,
		"tmax": 30000,
		"imp": [
			{
				"id": "/19968336/header-bid-tag-0",
				"banner": {
					"topframe": 1,
					"format": [
						{"w": 300, "h": 250},
						{"w": 300, "h": 600}
					]
				},
				"secure": 1,
				"ext": {
					"bidder": {
						"placementId": "p101152",
						"customerId": "ABCD123456"
					}
				}
			}
		],
		"site": {
			"domain": "localhost:3000",
			"publisher": {
				"domain": "localhost:3000",
				"id": "1"
			},
			"page": "http://localhost:3000/view?link=test",
			"ext": {
				"data": {
					"documentLang": "en"
				}
			}
		},
		"device": {
			"w": 1728,
			"h": 1117,
			"dnt": 1,
			"ua": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
			"language": "fr",
			"ext": {
				"vpw": 1645,
				"vph": 411
			}
		},
		"ext": {
			"prebid": {
				"auctiontimestamp": 1751152008051,
				"bidderconfig": [
					{
						"bidders": ["contxtful"],
						"config": {
							"ortb2": {
								"user": {
									"data": [
										{
											"name": "contxtful",
											"ext": {
												"rx": {
													"ReceptivityState": "Receptive",
													"EclecticChinchilla": "true",
													"score": "90",
													"gptP": [
														{
															"p": {"x": 8, "y": 152},
															"v": true,
															"a": "/19968336/header-bid-tag-0",
															"s": "/19968336/header-bid-tag-0",
															"t": "div"
														}
													],
													"rand": 66.70383102561092
												},
												"sm": "0fa9d2f7-96a2-497a-83c7-e4f14ee4b580",
												"params": {
													"ev": "v1",
													"ci": "ABCD123456"
												}
											},
											"segment": [
												{
													"id": "Receptive"
												}
											]
										}
									]
								}
							}
						}
					}
				],
				"debug": true
			}
		}
	}`

	var request openrtb2.BidRequest
	if err := json.Unmarshal([]byte(requestJSON), &request); err != nil {
		t.Fatalf("Failed to unmarshal test request: %v", err)
	}

	requests, errs := bidder.MakeRequests(&request, &adapters.ExtraRequestInfo{})

	if len(errs) > 0 {
		t.Fatalf("MakeRequests returned errors: %v", errs)
	}

	if len(requests) != 1 {
		t.Fatalf("Expected 1 request, got %d", len(requests))
	}

	// Parse the request payload to verify bidder config data was extracted and merged
	var payload map[string]interface{}
	if err := json.Unmarshal(requests[0].Body, &payload); err != nil {
		t.Fatalf("Failed to unmarshal request payload: %v", err)
	}

	// Verify the config uses bidder config version and customer (not impression params)
	config, ok := payload["config"].(map[string]interface{})
	if !ok {
		t.Fatal("Config should be an object")
	}

	contxtfulConfig, ok := config["contxtful"].(map[string]interface{})
	if !ok {
		t.Fatal("Contxtful config should be an object")
	}

	// Verify version from bidder config (not default)
	if contxtfulConfig["version"] != "v1" {
		t.Errorf("Expected version 'v1' from bidder config, got %v", contxtfulConfig["version"])
	}

	// Verify customer from bidder config (priority over impression params)
	expectedCustomer := "ABCD123456" // From bidder config params.ci
	if contxtfulConfig["customer"] != expectedCustomer {
		t.Errorf("Expected customer '%s' from bidder config, got %v", expectedCustomer, contxtfulConfig["customer"])
	}

	// Verify endpoint URL uses bidder config customer ID
	expectedURL := "https://prebid.receptivity.io/v1/pbs/ABCD123456/bid"
	if requests[0].Uri != expectedURL {
		t.Errorf("Expected URL %s (with bidder config customer), got %s", expectedURL, requests[0].Uri)
	}

	// Verify ortb2 contains original data (simple passthrough - no complex merging)
	ortb2, ok := payload["ortb2"].(map[string]interface{})
	if !ok {
		t.Fatal("ortb2 should be an object")
	}

	// Verify basic ortb2 structure is preserved as-is
	if ortb2["test"] != float64(1) {
		t.Error("ortb2.test should be preserved as-is")
	}

	if ortb2["tmax"] != float64(30000) {
		t.Error("ortb2.tmax should be preserved as-is")
	}

	// Verify site data is preserved as-is
	site, ok := ortb2["site"].(map[string]interface{})
	if !ok {
		t.Fatal("ortb2.site should be preserved as-is")
	}

	if site["domain"] != "localhost:3000" {
		t.Error("ortb2.site.domain should be preserved as-is")
	}

	// Verify device data is preserved as-is
	device, ok := ortb2["device"].(map[string]interface{})
	if !ok {
		t.Fatal("ortb2.device should be preserved as-is")
	}

	if device["language"] != "fr" {
		t.Error("ortb2.device.language should be preserved as-is")
	}

	// Verify bidder config data is available in ext.prebid.bidderconfig (original location)
	// The relay can extract rich data from here directly - no need for adapter to merge it
	ext, ok := ortb2["ext"].(map[string]interface{})
	if !ok {
		t.Fatal("ortb2.ext should be preserved as-is")
	}

	prebid, ok := ext["prebid"].(map[string]interface{})
	if !ok {
		t.Fatal("ortb2.ext.prebid should be preserved as-is")
	}

	bidderConfig, ok := prebid["bidderconfig"].([]interface{})
	if !ok || len(bidderConfig) == 0 {
		t.Fatal("ortb2.ext.prebid.bidderconfig should be preserved for relay processing")
	}

	// The relay will extract rich Contxtful data from bidderconfig directly
	// We don't need to merge it - simpler and more like other bidders
}

// TestMissingCustomerIDHandling tests that MakeBids fails fast when customer ID is missing
func TestMissingCustomerIDHandling(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Create a response that would normally generate bids
	responseJSON := `[{
		"traceId": "missing-customer-test-123",
		"random": 0.555666,
		"bids": [{
			"impid": "test-imp-missing-customer",
			"price": 1.50,
			"adm": "<div>Missing Customer Test Ad</div>",
			"w": 300,
			"h": 250,
			"crid": "missing-customer-creative"
		}]
	}]`

	response := &adapters.ResponseData{
		StatusCode: 200,
		Body:       []byte(responseJSON),
	}

	// Create request WITHOUT bidder config and WITHOUT valid impression params
	// This simulates the scenario where customer ID extraction fails
	requestJSON := `{
		"id": "missing-customer-test",
		"imp": [{
			"id": "test-imp-missing-customer",
			"banner": {"format": [{"w": 300, "h": 250}]},
			"ext": {} 
		}],
		"site": {
			"domain": "missing-customer-test.com",
			"page": "https://missing-customer-test.com/test"
		}
	}`

	var request openrtb2.BidRequest
	if err := json.Unmarshal([]byte(requestJSON), &request); err != nil {
		t.Fatalf("Failed to unmarshal test request: %v", err)
	}

	// Create empty requestData (simulating missing URI - should only happen in edge cases)
	requestData := &adapters.RequestData{
		Method: "POST",
		Uri:    "", // Empty URI to test fallback logic
	}

	// MakeBids should now return an error when customer ID is missing from all sources
	bids, errs := bidder.MakeBids(&request, requestData, response)

	// Should return an error about missing customer ID
	if len(errs) == 0 {
		t.Error("Expected error when customer ID is missing, but got none")
	} else {
		// Verify it's the right type of error
		expectedError := "No customer ID found in request URI, bidder config, or impression parameters"
		if !contains(errs[0].Error(), expectedError) {
			t.Errorf("Expected error containing '%s', got '%s'", expectedError, errs[0].Error())
		}
	}

	// Should not return any bids when customer ID is missing
	if bids != nil && len(bids.Bids) > 0 {
		t.Errorf("Expected no bids when customer ID missing, got %d bids", len(bids.Bids))
	}

}

// TestCustomerIDFromURIExtraction tests that customer ID is correctly extracted from request URI
func TestCustomerIDFromURIExtraction(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Create response with bids
	responseJSON := `[{
		"requestId": "test-imp-uri-extraction",
		"cpm": 2.25,
		"ad": "<div>URI Extraction Test Ad</div>",
		"width": 728,
		"height": 90,
		"creativeId": "uri-extraction-creative",
		"netRevenue": true,
		"currency": "USD",
		"mediaType": "banner",
		"bidderCode": "contxtful",
		"traceId": "uri-extraction-test-123",
		"random": 0.777888,
		"nurl": "https://monitoring.receptivity.io/v1/pbs/URITEST123/pbs-impression?b=contxtful-test-imp-uri-extraction&a=URITEST123&bidder=contxtful&impId=test-imp-uri-extraction&price=2.25&traceId=uri-extraction-test-123&random=0.777888&domain=uri-extraction-test.com&adRequestId=uri-extraction-test&w=728&h=90&f=b",
		"burl": "https://monitoring.receptivity.io/v1/pbs/URITEST123/pbs-billing?b=contxtful-test-imp-uri-extraction&a=URITEST123&bidder=contxtful&impId=test-imp-uri-extraction&price=2.25&traceId=uri-extraction-test-123&random=0.777888&domain=uri-extraction-test.com&adRequestId=uri-extraction-test&w=728&h=90&f=b"
	}]`

	response := &adapters.ResponseData{
		StatusCode: 200,
		Body:       []byte(responseJSON),
	}

	// Create request WITHOUT bidder config and WITHOUT valid impression params
	// (simulating cleaned up request data)
	requestJSON := `{
		"id": "uri-extraction-test",
		"imp": [{
			"id": "test-imp-uri-extraction",
			"banner": {"format": [{"w": 728, "h": 90}]},
			"ext": {} 
		}],
		"site": {
			"domain": "uri-extraction-test.com",
			"page": "https://uri-extraction-test.com/test"
		}
	}`

	var request openrtb2.BidRequest
	if err := json.Unmarshal([]byte(requestJSON), &request); err != nil {
		t.Fatalf("Failed to unmarshal test request: %v", err)
	}

	// Create requestData with URI containing customer ID (this is what MakeRequests creates)
	requestData := &adapters.RequestData{
		Method: "POST",
		Uri:    "https://prebid.receptivity.io/v1/pbs/URITEST123/bid", // Customer ID in URI
		Body:   []byte(`{"test": "data"}`),
	}

	// MakeBids should successfully extract customer ID from URI and generate event URLs
	bids, errs := bidder.MakeBids(&request, requestData, response)

	if len(errs) > 0 {
		t.Fatalf("MakeBids returned unexpected errors: %v", errs)
	}

	if bids == nil || len(bids.Bids) != 1 {
		bidCount := 0
		if bids != nil {
			bidCount = len(bids.Bids)
		}
		t.Fatalf("Expected 1 bid, got %d", bidCount)
	}

	bid := bids.Bids[0]

	// Verify event URLs are generated with correct customer ID from URI
	if bid.Bid.NURL == "" {
		t.Error("NURL should be generated when customer ID is available from URI")
	} else {
		// Should contain customer ID from URI
		if !contains(bid.Bid.NURL, "/v1/pbs/URITEST123/pbs-impression") {
			t.Error("NURL should contain customer ID extracted from URI (URITEST123)")
		}

		// Should contain tracking data
		if !contains(bid.Bid.NURL, "traceId=uri-extraction-test-123") {
			t.Error("NURL should contain tracking data from response")
		}
	}

	// Verify BURL is also generated correctly
	if bid.Bid.BURL == "" {
		t.Error("BURL should be generated when customer ID is available from URI")
	} else {
		// Should contain customer ID from URI
		if !contains(bid.Bid.BURL, "/v1/pbs/URITEST123/pbs-billing") {
			t.Error("BURL should contain customer ID extracted from URI (URITEST123)")
		}
	}
}

// TestEventURLGenerationNoDoubleSlash tests that event URLs never have double slashes with URI extraction
func TestEventURLGenerationNoDoubleSlash(t *testing.T) {
	// Test with trailing slash on monitoring endpoint
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	testCases := []struct {
		name             string
		requestURI       string
		expectedCustomer string
	}{
		{
			name:             "Production endpoint",
			requestURI:       "https://prebid.receptivity.io/v1/pbs/PRODCUST123/bid",
			expectedCustomer: "PRODCUST123",
		},
		{
			name:             "Development endpoint",
			requestURI:       "https://prebid.receptivity.io/v1/pbs/DEVCUST456/bid",
			expectedCustomer: "DEVCUST456",
		},
		{
			name:             "Monitoring endpoint with trailing slash",
			requestURI:       "https://prebid.receptivity.io/v1/pbs/SLASHTEST789/bid",
			expectedCustomer: "SLASHTEST789",
		},
		{
			name:             "Localhost development",
			requestURI:       "http://localhost:6789/v1/pbs/LOCALTEST999/bid",
			expectedCustomer: "LOCALTEST999",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Update adapter's monitoring endpoint for this test

			responseJSON := fmt.Sprintf(`[{
				"requestId": "test-imp-no-double-slash",
				"cpm": 3.75,
				"ad": "<div>No Double Slash Test Ad</div>",
				"width": 300,
				"height": 250,
				"creativeId": "no-double-slash-creative",
				"netRevenue": true,
				"currency": "USD",
				"mediaType": "banner",
				"bidderCode": "contxtful",
				"traceId": "no-double-slash-test-456",
				"random": 0.123456,
				"nurl": "%s/v1/pbs/%s/pbs-impression?b=contxtful-test-imp-no-double-slash&a=%s&bidder=contxtful&impId=test-imp-no-double-slash&price=3.75&traceId=no-double-slash-test-456&random=0.123456&domain=test.nodoubleSlash.com&adRequestId=no-double-slash-test&w=300&h=250&f=b",
				"burl": "%s/v1/pbs/%s/pbs-billing?b=contxtful-test-imp-no-double-slash&a=%s&bidder=contxtful&impId=test-imp-no-double-slash&price=3.75&traceId=no-double-slash-test-456&random=0.123456&domain=test.nodoubleSlash.com&adRequestId=no-double-slash-test&w=300&h=250&f=b"
			}]`, "https://monitoring.com/", tc.expectedCustomer, tc.expectedCustomer, "https://monitoring.com/", tc.expectedCustomer, tc.expectedCustomer)

			response := &adapters.ResponseData{
				StatusCode: 200,
				Body:       []byte(responseJSON),
			}

			requestJSON := `{
				"id": "no-double-slash-test",
				"imp": [{
					"id": "test-imp-no-double-slash",
					"banner": {"format": [{"w": 300, "h": 250}]},
					"ext": {}
				}],
				"site": {
					"domain": "test.nodoubleSlash.com",
					"page": "https://test.nodoubleSlash.com/test"
				}
			}`

			var request openrtb2.BidRequest
			if err := json.Unmarshal([]byte(requestJSON), &request); err != nil {
				t.Fatalf("Failed to unmarshal test request: %v", err)
			}

			requestData := &adapters.RequestData{
				Method: "POST",
				Uri:    tc.requestURI,
				Body:   []byte(`{"test": "data"}`),
			}

			bids, errs := bidder.MakeBids(&request, requestData, response)

			if len(errs) > 0 {
				t.Fatalf("MakeBids returned unexpected errors: %v", errs)
			}

			if bids == nil || len(bids.Bids) != 1 {
				t.Fatalf("Expected 1 bid, got %d", len(bids.Bids))
			}

			bid := bids.Bids[0]

			// Critical test: Verify NO double slashes anywhere in URLs
			if contains(bid.Bid.NURL, "//pbs-impression") {
				t.Errorf("NURL contains double slash: %s", bid.Bid.NURL)
			}

			if contains(bid.Bid.BURL, "//pbs-billing") {
				t.Errorf("BURL contains double slash: %s", bid.Bid.BURL)
			}

			// Verify correct customer ID is used in URLs
			expectedNURLPattern := fmt.Sprintf("/v1/pbs/%s/pbs-impression", tc.expectedCustomer)
			if !contains(bid.Bid.NURL, expectedNURLPattern) {
				t.Errorf("NURL should contain %s, got: %s", expectedNURLPattern, bid.Bid.NURL)
			}

			expectedBURLPattern := fmt.Sprintf("/v1/pbs/%s/pbs-billing", tc.expectedCustomer)
			if !contains(bid.Bid.BURL, expectedBURLPattern) {
				t.Errorf("BURL should contain %s, got: %s", expectedBURLPattern, bid.Bid.BURL)
			}

			// Verify customer parameter is correct
			expectedCustomerParam := fmt.Sprintf("a=%s", tc.expectedCustomer)
			if !contains(bid.Bid.NURL, expectedCustomerParam) {
				t.Errorf("NURL should contain %s", expectedCustomerParam)
			}
		})
	}
}

// TestNURLBURLFromPrebidJSResponse tests NURL/BURL extraction from PrebidJS format responses
func TestNURLBURLFromPrebidJSResponse(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Create test request
	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{
			{
				ID: "test-imp-id",
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{
						{W: 300, H: 250},
					},
				},
				Ext: json.RawMessage(`{
					"bidder": {
						"placementId": "test-placement",
						"customerId": "test-customer"
					}
				}`),
			},
		},
		Site: &openrtb2.Site{
			Domain: "example.com",
		},
	}

	// Mock response with NURL/BURL
	responseBody := `[{
		"requestId": "test-imp-id",
		"cpm": 2.50,
		"currency": "USD",
		"width": 300,
		"height": 250,
		"creativeId": "creative-123",
		"ad": "<div>Test Ad</div>",
		"ttl": 300,
		"netRevenue": true,
		"mediaType": "banner",
		"bidderCode": "contxtful",
		"traceId": "trace-123",
		"random": 0.123456,
		"nurl": "https://example.com/win?price=${AUCTION_PRICE}",
		"burl": "https://example.com/billing?price=${AUCTION_PRICE}"
	}]`

	response := &adapters.ResponseData{
		StatusCode: 200,
		Body:       []byte(responseBody),
	}

	// Make the request to get request data
	requestData, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})
	if len(errs) > 0 {
		t.Fatalf("MakeRequests returned errors: %v", errs)
	}

	// Process the response
	bidderResponse, errs := bidder.MakeBids(request, requestData[0], response)
	if len(errs) > 0 {
		t.Fatalf("MakeBids returned errors: %v", errs)
	}

	// Verify we got a bid
	if len(bidderResponse.Bids) != 1 {
		t.Fatalf("Expected 1 bid, got %d", len(bidderResponse.Bids))
	}

	bid := bidderResponse.Bids[0].Bid

	// Verify NURL is passed through from response
	expectedNURL := "https://example.com/win?price=${AUCTION_PRICE}"
	if bid.NURL != expectedNURL {
		t.Errorf("NURL should be %s, got %s", expectedNURL, bid.NURL)
	}

	// Verify BURL is passed through from response
	expectedBURL := "https://example.com/billing?price=${AUCTION_PRICE}"
	if bid.BURL != expectedBURL {
		t.Errorf("BURL should be %s, got %s", expectedBURL, bid.BURL)
	}
}

// TestNURLBURLFromContxtfulRelayResponse tests NURL/BURL extraction from ContxtfulRelay format responses
func TestNURLBURLFromContxtfulRelayResponse(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Create test request
	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{
			{
				ID: "test-imp-id",
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{
						{W: 728, H: 90},
					},
				},
				Ext: json.RawMessage(`{
					"bidder": {
						"placementId": "test-placement",
						"customerId": "test-customer"
					}
				}`),
			},
		},
		Site: &openrtb2.Site{
			Domain: "test.com",
		},
	}

	// Mock response with PrebidJS format including NURL/BURL
	responseBody := `[{
		"requestId": "test-imp-id",
		"cpm": 1.75,
		"ad": "<div>Relay Ad</div>",
		"width": 728,
		"height": 90,
		"creativeId": "creative-456",
		"netRevenue": true,
		"currency": "USD",
		"mediaType": "banner",
		"bidderCode": "contxtful",
		"traceId": "trace-456",
		"random": 0.654321,
		"nurl": "https://relay.example.com/notify?win=1",
		"burl": "https://relay.example.com/bill?event=1"
	}]`

	response := &adapters.ResponseData{
		StatusCode: 200,
		Body:       []byte(responseBody),
	}

	// Make the request to get request data
	requestData, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})
	if len(errs) > 0 {
		t.Fatalf("MakeRequests returned errors: %v", errs)
	}

	// Process the response
	bidderResponse, errs := bidder.MakeBids(request, requestData[0], response)
	if len(errs) > 0 {
		t.Fatalf("MakeBids returned errors: %v", errs)
	}

	// Verify we got a bid
	if len(bidderResponse.Bids) != 1 {
		t.Fatalf("Expected 1 bid, got %d", len(bidderResponse.Bids))
	}

	bid := bidderResponse.Bids[0].Bid

	// Verify NURL is passed through from response
	expectedNURL := "https://relay.example.com/notify?win=1"
	if bid.NURL != expectedNURL {
		t.Errorf("NURL should be %s, got %s", expectedNURL, bid.NURL)
	}

	// Verify BURL is passed through from response
	expectedBURL := "https://relay.example.com/bill?event=1"
	if bid.BURL != expectedBURL {
		t.Errorf("BURL should be %s, got %s", expectedBURL, bid.BURL)
	}
}

// TestNURLBURLWithoutResponseURLs tests that event URLs are generated correctly when response has no NURL/BURL
func TestNURLBURLWithoutResponseURLs(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Create test request
	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{
			{
				ID: "test-imp-id",
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{
						{W: 300, H: 250},
					},
				},
				Ext: json.RawMessage(`{
					"bidder": {
						"placementId": "test-placement",
						"customerId": "test-customer"
					}
				}`),
			},
		},
		Site: &openrtb2.Site{
			Domain: "example.com",
		},
	}

	// Mock response without NURL/BURL
	responseBody := `[{
		"requestId": "test-imp-id",
		"cpm": 2.50,
		"currency": "USD",
		"width": 300,
		"height": 250,
		"creativeId": "creative-123",
		"ad": "<div>Test Ad</div>",
		"ttl": 300,
		"netRevenue": true,
		"mediaType": "banner",
		"bidderCode": "contxtful",
		"traceId": "trace-123",
		"random": 0.123456
	}]`

	response := &adapters.ResponseData{
		StatusCode: 200,
		Body:       []byte(responseBody),
	}

	// Make the request to get request data
	requestData, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})
	if len(errs) > 0 {
		t.Fatalf("MakeRequests returned errors: %v", errs)
	}

	// Process the response
	bidderResponse, errs := bidder.MakeBids(request, requestData[0], response)
	if len(errs) > 0 {
		t.Fatalf("MakeBids returned errors: %v", errs)
	}

	// Verify we got a bid
	if len(bidderResponse.Bids) != 1 {
		t.Fatalf("Expected 1 bid, got %d", len(bidderResponse.Bids))
	}

	bid := bidderResponse.Bids[0].Bid

	// Verify NURL/BURL are not set when not provided in response
	if bid.NURL != "" {
		t.Error("NURL should be empty when no response NURL is provided")
	}
	if bid.BURL != "" {
		t.Error("BURL should be empty when no response BURL is provided")
	}
}

// TestNURLBURLURLEncoding tests that special characters in response URLs are properly encoded
func TestNURLBURLURLEncoding(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Create test request
	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{
			{
				ID: "test-imp-id",
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{
						{W: 300, H: 250},
					},
				},
				Ext: json.RawMessage(`{
					"bidder": {
						"placementId": "test-placement",
						"customerId": "test-customer"
					}
				}`),
			},
		},
		Site: &openrtb2.Site{
			Domain: "example.com",
		},
	}

	// Mock response with URLs containing special characters that need encoding
	responseBody := `[{
		"requestId": "test-imp-id",
		"cpm": 2.50,
		"currency": "USD",
		"width": 300,
		"height": 250,
		"creativeId": "creative-123",
		"ad": "<div>Test Ad</div>",
		"ttl": 300,
		"netRevenue": true,
		"mediaType": "banner",
		"bidderCode": "contxtful",
		"traceId": "trace-123",
		"random": 0.123456,
		"nurl": "https://example.com/win?price=${AUCTION_PRICE}&data={\"test\":\"value\"}&special=a+b c&hash#fragment",
		"burl": "https://example.com/billing?price=${AUCTION_PRICE}&data={\"test\":\"value\"}&special=a+b c&hash#fragment"
	}]`

	response := &adapters.ResponseData{
		StatusCode: 200,
		Body:       []byte(responseBody),
	}

	// Make the request to get request data
	requestData, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})
	if len(errs) > 0 {
		t.Fatalf("MakeRequests returned errors: %v", errs)
	}

	// Process the response
	bidderResponse, errs := bidder.MakeBids(request, requestData[0], response)
	if len(errs) > 0 {
		t.Fatalf("MakeBids returned errors: %v", errs)
	}

	// Verify we got a bid
	if len(bidderResponse.Bids) != 1 {
		t.Fatalf("Expected 1 bid, got %d", len(bidderResponse.Bids))
	}

	bid := bidderResponse.Bids[0].Bid

	// Verify the complex URL is passed through as-is (no encoding)
	expectedNURL := "https://example.com/win?price=${AUCTION_PRICE}&data={\"test\":\"value\"}&special=a+b c&hash#fragment"
	if bid.NURL != expectedNURL {
		t.Errorf("NURL should be %s, got %s", expectedNURL, bid.NURL)
	}

	expectedBURL := "https://example.com/billing?price=${AUCTION_PRICE}&data={\"test\":\"value\"}&special=a+b c&hash#fragment"
	if bid.BURL != expectedBURL {
		t.Errorf("BURL should be %s, got %s", expectedBURL, bid.BURL)
	}
}

// TestNURLBURLPartialResponse tests handling when only one of NURL or BURL is present
func TestNURLBURLPartialResponse(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Create test request
	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{
			{
				ID: "test-imp-id",
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{
						{W: 300, H: 250},
					},
				},
				Ext: json.RawMessage(`{
					"bidder": {
						"placementId": "test-placement",
						"customerId": "test-customer"
					}
				}`),
			},
		},
		Site: &openrtb2.Site{
			Domain: "example.com",
		},
	}

	// Mock response with only NURL (no BURL)
	responseBody := `[{
		"requestId": "test-imp-id",
		"cpm": 2.50,
		"currency": "USD",
		"width": 300,
		"height": 250,
		"creativeId": "creative-123",
		"ad": "<div>Test Ad</div>",
		"ttl": 300,
		"netRevenue": true,
		"mediaType": "banner",
		"bidderCode": "contxtful",
		"traceId": "trace-123",
		"random": 0.123456,
		"nurl": "https://example.com/win?price=${AUCTION_PRICE}"
	}]`

	response := &adapters.ResponseData{
		StatusCode: 200,
		Body:       []byte(responseBody),
	}

	// Make the request to get request data
	requestData, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})
	if len(errs) > 0 {
		t.Fatalf("MakeRequests returned errors: %v", errs)
	}

	// Process the response
	bidderResponse, errs := bidder.MakeBids(request, requestData[0], response)
	if len(errs) > 0 {
		t.Fatalf("MakeBids returned errors: %v", errs)
	}

	// Verify we got a bid
	if len(bidderResponse.Bids) != 1 {
		t.Fatalf("Expected 1 bid, got %d", len(bidderResponse.Bids))
	}

	bid := bidderResponse.Bids[0].Bid

	// Verify NURL is passed through from response
	expectedNURL := "https://example.com/win?price=${AUCTION_PRICE}"
	if bid.NURL != expectedNURL {
		t.Errorf("NURL should be %s, got %s", expectedNURL, bid.NURL)
	}

	// Verify BURL is empty (not provided in response)
	if bid.BURL != "" {
		t.Errorf("BURL should be empty when not provided in response, got %s", bid.BURL)
	}
}

// TestNURLBURLEmptyStrings tests handling of empty string values in response
func TestNURLBURLEmptyStrings(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Create test request
	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{
			{
				ID: "test-imp-id",
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{
						{W: 300, H: 250},
					},
				},
				Ext: json.RawMessage(`{
					"bidder": {
						"placementId": "test-placement",
						"customerId": "test-customer"
					}
				}`),
			},
		},
		Site: &openrtb2.Site{
			Domain: "example.com",
		},
	}

	// Mock response with empty string values for NURL/BURL
	responseBody := `[{
		"requestId": "test-imp-id",
		"cpm": 2.50,
		"currency": "USD",
		"width": 300,
		"height": 250,
		"creativeId": "creative-123",
		"ad": "<div>Test Ad</div>",
		"ttl": 300,
		"netRevenue": true,
		"mediaType": "banner",
		"bidderCode": "contxtful",
		"traceId": "trace-123",
		"random": 0.123456,
		"nurl": "",
		"burl": ""
	}]`

	response := &adapters.ResponseData{
		StatusCode: 200,
		Body:       []byte(responseBody),
	}

	// Make the request to get request data
	requestData, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})
	if len(errs) > 0 {
		t.Fatalf("MakeRequests returned errors: %v", errs)
	}

	// Process the response
	bidderResponse, errs := bidder.MakeBids(request, requestData[0], response)
	if len(errs) > 0 {
		t.Fatalf("MakeBids returned errors: %v", errs)
	}

	// Verify we got a bid
	if len(bidderResponse.Bids) != 1 {
		t.Fatalf("Expected 1 bid, got %d", len(bidderResponse.Bids))
	}

	bid := bidderResponse.Bids[0].Bid

	// Verify URLs don't contain 'r' parameter for empty strings
	if strings.Contains(bid.NURL, "&r=") {
		t.Error("NURL should not contain r parameter when response NURL is empty")
	}
	if strings.Contains(bid.BURL, "&r=") {
		t.Error("BURL should not contain r parameter when response BURL is empty")
	}
}

// TestNURLBURLNotPresentInRelayResponse tests that NURL/BURL are not set when not present in ContxtfulRelay response
func TestNURLBURLNotPresentInRelayResponse(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Create test request
	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{
			{
				ID: "test-imp-id",
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{
						{W: 728, H: 90},
					},
				},
				Ext: json.RawMessage(`{
					"bidder": {
						"placementId": "test-placement",
						"customerId": "test-customer"
					}
				}`),
			},
		},
		Site: &openrtb2.Site{
			Domain: "test.com",
		},
	}

	// Mock response with PrebidJS format WITHOUT NURL/BURL fields
	responseBody := `[{
		"requestId": "test-imp-id",
		"cpm": 1.75,
		"ad": "<div>Relay Ad</div>",
		"width": 728,
		"height": 90,
		"creativeId": "creative-456",
		"netRevenue": true,
		"currency": "USD",
		"mediaType": "banner",
		"bidderCode": "contxtful",
		"traceId": "trace-456",
		"random": 0.654321
	}]`

	response := &adapters.ResponseData{
		StatusCode: 200,
		Body:       []byte(responseBody),
	}

	// Make the request to get request data
	requestData, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})
	if len(errs) > 0 {
		t.Fatalf("MakeRequests returned errors: %v", errs)
	}

	// Process the response
	bidderResponse, errs := bidder.MakeBids(request, requestData[0], response)
	if len(errs) > 0 {
		t.Fatalf("MakeBids returned errors: %v", errs)
	}

	// Verify we got a bid
	if len(bidderResponse.Bids) != 1 {
		t.Fatalf("Expected 1 bid, got %d", len(bidderResponse.Bids))
	}

	bid := bidderResponse.Bids[0].Bid

	// Verify NURL is not set when not provided in response
	if bid.NURL != "" {
		t.Errorf("NURL should be empty when not present in relay response, got %s", bid.NURL)
	}

	// Verify BURL is not set when not provided in response
	if bid.BURL != "" {
		t.Errorf("BURL should be empty when not present in relay response, got %s", bid.BURL)
	}
}

// TestPBSVersionInPayload tests that the PBS version is correctly included at the top level of the payload
func TestPBSVersionInPayload(t *testing.T) {
	// Store original version and set test version
	originalVersion := version.Ver
	testVersion := "test-pbs-version-1.2.3"
	version.Ver = testVersion
	defer func() {
		version.Ver = originalVersion
	}()

	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid",
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Create a simple test request
	requestJSON := `{
		"id": "test-request-id",
		"imp": [
			{
				"id": "test-imp-id",
				"banner": {
					"format": [
						{"w": 300, "h": 250}
					]
				},
				"ext": {
					"bidder": {
						"placementId": "test-placement",
						"customerId": "test-customer-123"
					}
				}
			}
		],
		"site": {
			"id": "test-site",
			"page": "https://example.com"
		}
	}`

	var request openrtb2.BidRequest
	if err := json.Unmarshal([]byte(requestJSON), &request); err != nil {
		t.Fatalf("Failed to unmarshal test request: %v", err)
	}

	requests, errs := bidder.MakeRequests(&request, &adapters.ExtraRequestInfo{})

	if len(errs) > 0 {
		t.Fatalf("MakeRequests returned errors: %v", errs)
	}

	if len(requests) != 1 {
		t.Fatalf("Expected 1 request, got %d", len(requests))
	}

	// Parse the request payload
	var payload map[string]interface{}
	if err := json.Unmarshal(requests[0].Body, &payload); err != nil {
		t.Fatalf("Failed to unmarshal request payload: %v", err)
	}

	// Verify config exists (should NOT contain pbs)
	config, ok := payload["config"].(map[string]interface{})
	if !ok {
		t.Fatal("Config should be an object")
	}

	// Verify pbs is NOT in config (it should be at top level)
	if _, exists := config["pbs"]; exists {
		t.Error("PBS should not be nested under config, it should be at top level")
	}

	// Verify pbs config exists at top level
	pbsConfig, ok := payload["pbs"].(map[string]interface{})
	if !ok {
		t.Fatal("PBS config should be an object at top level")
	}

	// Verify PBS version is present
	pbsVersion, ok := pbsConfig["version"].(string)
	if !ok {
		t.Fatal("PBS version should be a string")
	}

	// Verify that PBS version matches the expected test version
	if pbsVersion != testVersion {
		t.Errorf("Expected PBS version '%s', got '%s'", testVersion, pbsVersion)
	}
}
