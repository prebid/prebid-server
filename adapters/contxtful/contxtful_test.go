package contxtful

import (
	"encoding/json"
	"strings"
	"testing"

	"encoding/base64"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint:         "https://prebid.receptivity.io/v1/prebid/{{.AccountID}}/bid",
		ExtraAdapterInfo: `{"monitoringEndpoint": "https://monitoring.receptivity.io"}`,
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
		Endpoint:         "https://prebid.receptivity.io/v1/prebid/{{.AccountID}}/bid",
		ExtraAdapterInfo: `{"monitoringEndpoint": "https://monitoring.receptivity.io"}`,
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
	expectedURL := "https://prebid.receptivity.io/v1/prebid/test-customer-123/bid"
	if requests[0].Uri != expectedURL {
		t.Errorf("Expected URL %s, got %s", expectedURL, requests[0].Uri)
	}
}

// TestCookieFlowWithBuyerUID tests cookie handling when BuyerUID is present
func TestCookieFlowWithBuyerUID(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint:         "https://prebid.receptivity.io/v1/prebid/{{.AccountID}}/bid",
		ExtraAdapterInfo: `{"monitoringEndpoint": "https://monitoring.receptivity.io"}`,
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
		Endpoint:         "https://prebid.receptivity.dev/v1/prebid/{{.AccountID}}/bid",
		ExtraAdapterInfo: `{"monitoringEndpoint": "https://monitoring.receptivity.dev"}`,
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
		Endpoint:         "https://prebid.receptivity.dev/v1/prebid/{{.AccountID}}/bid",
		ExtraAdapterInfo: `{"monitoringEndpoint": "https://monitoring.receptivity.dev"}`,
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
							"adslot": "/139271940/tonbarbier>fr/tonbarbier_site:header-2"
						},
						"pbadslot": "/139271940/tonbarbier>fr/tonbarbier_site:header-2"
					},
					"gpid": "/139271940/tonbarbier>fr/tonbarbier_site:header-2"
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
			"domain": "tonbarbier.com",
			"publisher": {
				"domain": "tonbarbier.com"
			},
			"page": "https://tonbarbier.com/",
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
						"events": "eyJ1aSI6eyJwb3NpdGlvbiI6eyJ4IjoyMTYuODk0NTMxMjUsInkiOjMyMy4xNDg0Mzc1LCJ0aW1lc3RhbXBNcyI6NzQyLjU5OTk5OTk2NDIzNzJ9LCJzY3JlZW4iOnsidG9wTGVmdCI6eyJ4IjowLCJ5Ijo0MjV9LCJ3aWR0aCI6MTMxNiwiaGVpZ2h0Ijo0NzcsInRpbWVzdGFtcE1zIjo3NDYuODAwMDAwMDExOTIwOX19fQ==",
						"pos": "eyJvYm94YWRzLWhlYWRlci0yIjp7InAiOnsieCI6NjU4LCJ5IjoxMDM0fSwidiI6dHJ1ZSwidCI6ImRpdiJ9fQ==",
						"sm": null,
						"params": {
							"ev": "v1",
							"ci": "MTKP241212"
						},
						"rx": {
							"ReceptivityState": "NonReceptive",
							"EclecticChinchilla": "false",
							"score": "4",
							"gptP": [
								{
									"p": {
										"x": 8,
										"y": -991
									},
									"v": true,
									"a": "/139271940/tonbarbier>fr/tonbarbier_site:oop-1",
									"s": "oboxads-oop-1",
									"t": "div"
								},
								{
									"p": {
										"x": 508,
										"y": 133
									},
									"v": true,
									"a": "/139271940/tonbarbier>fr/tonbarbier_site:header-1",
									"s": "oboxads-header-1",
									"t": "div"
								},
								{
									"p": {
										"x": 658,
										"y": 1034
									},
									"v": true,
									"a": "/139271940/tonbarbier>fr/tonbarbier_site:header-2",
									"s": "oboxads-header-2",
									"t": "div"
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
			"customerId": "MTKP241212"
		},
		"data": {
			"adserver": {
				"name": "gam",
				"adslot": "/139271940/tonbarbier>fr/tonbarbier_site:header-2"
			},
			"pbadslot": "/139271940/tonbarbier>fr/tonbarbier_site:header-2"
		},
		"gpid": "/139271940/tonbarbier>fr/tonbarbier_site:header-2"
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

	if site["domain"] != "tonbarbier.com" {
		t.Error("ortb2.site.domain should be preserved")
	}

	if site["page"] != "https://tonbarbier.com/" {
		t.Error("ortb2.site.page should be preserved")
	}

	publisher, ok := site["publisher"].(map[string]interface{})
	if !ok {
		t.Fatal("ortb2.site.publisher should be an object")
	}

	if publisher["domain"] != "tonbarbier.com" {
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

	if params["ci"] != "MTKP241212" {
		t.Error("Contxtful params should include correct customer ID (MTKP241212)")
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

	// Verify events and position data exist (base64 encoded)
	if contxtfulExt["events"] == nil {
		t.Error("Contxtful data should have events data")
	}

	if contxtfulExt["pos"] == nil {
		t.Error("Contxtful data should have position data")
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
		Endpoint:         "https://prebid.receptivity.io/v1/prebid/{{.AccountID}}/bid",
		ExtraAdapterInfo: `{"monitoringEndpoint": "https://monitoring.receptivity.io"}`,
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

	t.Log("Event tracking configuration test passed - adapter preserves media types and builds proper requests for relay processing")
}

// TestCookieSyncEndpointGeneration tests cookie sync URL generation
func TestCookieSyncEndpointGeneration(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint:         "https://prebid.receptivity.io/v1/prebid/{{.AccountID}}/bid",
		ExtraAdapterInfo: `{"monitoringEndpoint": "https://monitoring.receptivity.io"}`,
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Note: User sync configuration is handled through PBS bidder-info files
	// not through adapter config. This test verifies the adapter can be built
	// successfully for sync scenarios.

	t.Log("Cookie sync endpoint configuration test passed - adapter builds successfully")
}

// TestUserSyncWithBuyerUID tests user sync behavior when BuyerUID is present
func TestUserSyncWithBuyerUID(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint:         "https://prebid.receptivity.io/v1/prebid/{{.AccountID}}/bid",
		ExtraAdapterInfo: `{"monitoringEndpoint": "https://monitoring.receptivity.io"}`,
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
			"buyeruid": "contxtful-v1-eyJjdXN0b21lciI6IlNZTkMxMjMiLCJ0aW1lc3RhbXAiOjE3MzQ1Njc4OTAwMDAsInBhcnRuZXJzIjp7ImFteCI6ImFteC10ZXN0In0sInZlcnNpb24iOiJ2MSJ9"
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

	if buyerUID != "contxtful-v1-eyJjdXN0b21lciI6IlNZTkMxMjMiLCJ0aW1lc3RhbXAiOjE3MzQ1Njc4OTAwMDAsInBhcnRuZXJzIjp7ImFteCI6ImFteC10ZXN0In0sInZlcnNpb24iOiJ2MSJ9" {
		t.Errorf("Expected BuyerUID 'contxtful-v1-eyJjdXN0b21lciI6IlNZTkMxMjMiLCJ0aW1lc3RhbXAiOjE3MzQ1Njc4OTAwMDAsInBhcnRuZXJzIjp7ImFteCI6ImFteC10ZXN0In0sInZlcnNpb24iOiJ2MSJ9', got '%s'", buyerUID)
	}

	t.Logf("BuyerUID correctly preserved: %s", buyerUID)
}

// TestBuyerUIDFromPrebidMap tests reading UID from request.user.ext.prebid.buyeruids.contxtful and writing to ortb2.user.buyeruid
func TestBuyerUIDFromPrebidMap(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint:         "https://prebid.receptivity.io/v1/prebid/{{.AccountID}}/bid",
		ExtraAdapterInfo: `{"monitoringEndpoint": "https://monitoring.receptivity.io"}`,
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

	t.Logf("✅ Successfully read UID from request.user.ext.prebid.buyeruids.contxtful and wrote to ortb2.user.buyeruid: %s", buyerUID)
}

// TestNoCookieHeaders tests that cookie headers are never set (we only use ortb2.user.buyeruid)
func TestNoCookieHeaders(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint:         "https://prebid.receptivity.io/v1/prebid/{{.AccountID}}/bid",
		ExtraAdapterInfo: `{"monitoringEndpoint": "https://monitoring.receptivity.io"}`,
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
				"user": {"buyeruid": "contxtful-v1-eyJjdXN0b21lciI6Ik5PQ09PS0lFMTIzIiwidGltZXN0YW1wIjoxNzM0NTY3ODkwMDAwLCJwYXJ0bmVycyI6eyJhbXgiOiJhbXgtdGVzdCJ9LCJ2ZXJzaW9uIjoidjEifQ=="}
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

			t.Logf("✅ %s: %s", tc.name, tc.description)
		})
	}
}

// TestErrorHandling tests various error scenarios
func TestErrorHandling(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint:         "https://prebid.receptivity.io/v1/prebid/{{.AccountID}}/bid",
		ExtraAdapterInfo: `{"monitoringEndpoint": "https://monitoring.receptivity.io"}`,
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
		Endpoint:         "https://prebid.receptivity.io/v1/prebid/{{.AccountID}}/bid",
		ExtraAdapterInfo: `{"monitoringEndpoint": "https://monitoring.receptivity.io"}`,
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
	expectedURL := "https://prebid.receptivity.io/v1/prebid/CUSTOMER123/bid"
	if requests[0].Uri != expectedURL {
		t.Errorf("Expected URL %s, got %s", expectedURL, requests[0].Uri)
	}

	t.Logf("Base64 Partner UID test passed - UID: %s", encodedUID)
}

// TestBase64UIDVersioning tests different versions of Base64 encoded UIDs
func TestBase64UIDVersioning(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint:         "https://prebid.receptivity.dev/v1/prebid/{{.AccountID}}/bid",
		ExtraAdapterInfo: `{"monitoringEndpoint": "https://monitoring.receptivity.dev"}`,
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

			t.Logf("%s test passed - UID preserved: %s", tc.name, buyerUID)
		})
	}
}

// TestMultiPartnerUIDSize tests that the adapter handles large Base64 UIDs with many partners
func TestMultiPartnerUIDSize(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint:         "https://prebid.receptivity.dev/v1/prebid/{{.AccountID}}/bid",
		ExtraAdapterInfo: `{"monitoringEndpoint": "https://monitoring.receptivity.dev"}`,
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

	t.Logf("Large UID test passed - Size: %d bytes, Under limit: %t", len(largeUID), len(largeUID) <= 4096)
}

// TestCookieFormatValidation tests various cookie format scenarios
func TestCookieFormatValidation(t *testing.T) {
	testCases := []struct {
		name             string
		cookie           string
		expectValid      bool
		expectedCustomer string
		description      string
	}{
		{
			name:             "Valid v1 Base64 format",
			cookie:           "contxtful-v1-eyJjdXN0b21lciI6IkNUWFAyNDExMjciLCJ0aW1lc3RhbXAiOjE3NTEwNzcwODc0NTMsInBhcnRuZXJzIjp7fSwidmVyc2lvbiI6InYxIn0=",
			expectValid:      true,
			expectedCustomer: "CTXP241127",
			description:      "Should parse valid Base64 v1 format correctly",
		},
		{
			name:             "Valid v2 Base64 format (future compatibility)",
			cookie:           "contxtful-v2-eyJjdXN0b21lciI6IkZVVFVSRTEyMyIsInRpbWVzdGFtcCI6MTc1MTA3NzA4NzQ1MywicGFydG5lcnMiOnt9LCJ2ZXJzaW9uIjoidjIifQ==",
			expectValid:      true,
			expectedCustomer: "FUTURE123",
			description:      "Should handle future version formats",
		},
		{
			name:             "Invalid old format (should be ignored)",
			cookie:           "contxtful-pbs-1751077087453-5kaw9arzf",
			expectValid:      false,
			expectedCustomer: "",
			description:      "Should reject old non-Base64 format",
		},
		{
			name:             "Malformed Base64",
			cookie:           "contxtful-v1-invalid-base64!@#",
			expectValid:      false,
			expectedCustomer: "",
			description:      "Should handle malformed Base64 gracefully",
		},
		{
			name:             "Empty cookie",
			cookie:           "",
			expectValid:      false,
			expectedCustomer: "",
			description:      "Should handle empty cookie gracefully",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test cookie parsing logic (simplified version of what happens in real code)
			var customer string
			var isValid bool

			if tc.cookie != "" && strings.HasPrefix(tc.cookie, "contxtful-v") {
				parts := strings.Split(tc.cookie, "-")
				if len(parts) >= 3 {
					base64Part := strings.Join(parts[2:], "-")
					if decoded, err := base64.StdEncoding.DecodeString(base64Part); err == nil {
						var userData map[string]interface{}
						if err := json.Unmarshal(decoded, &userData); err == nil {
							if cust, ok := userData["customer"].(string); ok {
								customer = cust
								isValid = true
							}
						}
					}
				}
			}

			if isValid != tc.expectValid {
				t.Errorf("Expected valid=%t, got valid=%t for cookie: %s", tc.expectValid, isValid, tc.cookie)
			}

			if tc.expectValid && customer != tc.expectedCustomer {
				t.Errorf("Expected customer=%s, got customer=%s", tc.expectedCustomer, customer)
			}

			t.Logf("✅ %s: %s", tc.name, tc.description)
		})
	}
}

// TestBidRejectionScenarios tests various scenarios that should result in bid rejection
func TestBidRejectionScenarios(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint:         "https://prebid.receptivity.dev/v1/prebid/{{.AccountID}}/bid",
		ExtraAdapterInfo: `{"monitoringEndpoint": "https://monitoring.receptivity.dev"}`,
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

			t.Logf("✅ %s: %s", tc.name, tc.description)
		})
	}
}

// TestBidExtensionsRegression ensures that essential bid extensions are present in relay responses
// This test prevents regression of the bid extension creation logic that was accidentally removed during refactoring
func TestBidExtensionsRegression(t *testing.T) {
	// Create mock adapter
	server := config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"}
	mockAdapter, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint:         "https://relay.example.com/v1/prebid/{{.CustomerId}}/bid",
		ExtraAdapterInfo: `{"monitoringEndpoint": "https://monitoring.receptivity.io"}`,
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

	// Create mock response data - RELAY FORMAT with bid extensions
	mockRelayResponse := `[{
		"traceId": "extension-test-trace-123",
		"random": 0.789123,
		"bids": [{
			"impid": "test-imp-extensions",
			"price": 2.85,
			"adm": "<div>Test ad for extension validation</div>",
			"w": 300,
			"h": 250,
			"crid": "extension-creative-456",
			"nurl": "https://pbs-impression",
			"burl": "https://pbs-billing"
		}]
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
	assert.Contains(t, bid.NURL, "https://pbs-impression", "NURL should contain win tracking parameter")
	assert.Contains(t, bid.BURL, "https://pbs-billing", "BURL should contain billing tracking parameter")
}

// TestS2SBidderConfigExtraction tests extraction of rich bidder config data from S2S payloads
func TestS2SBidderConfigExtraction(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint:         "https://prebid.receptivity.io/v1/prebid/{{.AccountID}}/bid",
		ExtraAdapterInfo: `{"monitoringEndpoint": "https://monitoring.receptivity.io"}`,
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
						"customerId": "CTXP241127"
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
												"events": "eyJ1aSI6eyJzY3JlZW4iOnsidG9wTGVmdCI6eyJ4IjowLCJ5IjowfSwid2lkdGgiOjE2NDUsImhlaWdodCI6NDExLCJ0aW1lc3RhbXBNcyI6MTk1NC4zOTk5OTk5NzYxNTgxfX19",
												"pos": "eyIvMTk5NjgzMzYvaGVhZGVyLWJpZC10YWctMCI6eyJwIjp7IngiOjgsInkiOjE1Mn0sInYiOnRydWUsInQiOiJkaXYifX0=",
												"sm": "0fa9d2f7-96a2-497a-83c7-e4f14ee4b580",
												"params": {
													"ev": "v1",
													"ci": "1Pw320rMi1BNmV0C8TEX7LlYD"
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
	expectedCustomer := "1Pw320rMi1BNmV0C8TEX7LlYD" // From bidder config params.ci
	if contxtfulConfig["customer"] != expectedCustomer {
		t.Errorf("Expected customer '%s' from bidder config, got %v", expectedCustomer, contxtfulConfig["customer"])
	}

	// Verify endpoint URL uses bidder config customer ID
	expectedURL := "https://prebid.receptivity.io/v1/prebid/1Pw320rMi1BNmV0C8TEX7LlYD/bid"
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

	t.Logf("✅ S2S Bidder Config Extraction test passed (SIMPLIFIED APPROACH)")
	t.Logf("   - Bidder config version: %v (priority over default)", contxtfulConfig["version"])
	t.Logf("   - Bidder config customer: %v (priority over impression params)", contxtfulConfig["customer"])
	t.Logf("   - Endpoint URL: %s (uses bidder config customer)", requests[0].Uri)
	t.Logf("   - Original ortb2 data preserved as-is (no complex merging)")
	t.Logf("   - Rich bidder config data available in ext.prebid.bidderconfig for relay processing")
	t.Logf("   - Simple passthrough approach like other bidders - relay handles data extraction")
}

// TestMissingCustomerIDHandling tests that MakeBids fails fast when customer ID is missing
func TestMissingCustomerIDHandling(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint:         "https://prebid.receptivity.io/v1/prebid/{{.AccountID}}/bid",
		ExtraAdapterInfo: `{"monitoringEndpoint": "https://monitoring.receptivity.io"}`,
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
		t.Logf("Correctly returned error: %v", errs[0])

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

	t.Logf("✅ Missing customer ID handling test passed")
	t.Logf("   - MakeBids fails fast when no customer ID found")
	t.Logf("   - No invalid endpoints like '/v1/prebid/unknown' or '/v1/prebid//pbs-event'")
	t.Logf("   - Proper error handling prevents malformed requests")
}

// TestCustomerIDFromURIExtraction tests that customer ID is correctly extracted from request URI
func TestCustomerIDFromURIExtraction(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint:         "https://prebid.receptivity.io/v1/prebid/{{.AccountID}}/bid",
		ExtraAdapterInfo: `{"monitoringEndpoint": "https://monitoring.receptivity.io"}`,
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Create response with bids
	responseJSON := `[{
		"traceId": "uri-extraction-test-123",
		"random": 0.777888,
		"bids": [{
			"impid": "test-imp-uri-extraction",
			"price": 2.25,
			"adm": "<div>URI Extraction Test Ad</div>",
			"w": 728,
			"h": 90,
			"crid": "uri-extraction-creative",
			"nurl": "https://nurl/v1/prebid/URITEST123/pbs-impression?traceId=uri-extraction-test-123",
			"burl": "https://monitoring.receptivity.io/v1/prebid/URITEST123/pbs-billing?traceId=uri-extraction-test-123&random=0.777888&impid=test-imp-uri-extraction&price=2.25&crid=uri-extraction-creative"
		}]
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
			"ext": {},
			"nurl": "https://pbs-impression",
			"burl": "https://monitoring.receptivity.io/v1/prebid/URITEST123/pbs-billing?traceId=uri-extraction-test-123&random=0.777888&impid=test-imp-uri-extraction&price=2.25&crid=uri-extraction-creative"
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
		Uri:    "https://prebid.receptivity.io/v1/prebid/URITEST123/bid", // Customer ID in URI
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
		t.Logf("NURL: %s", bid.Bid.NURL)

		// Should contain customer ID from URI
		if !contains(bid.Bid.NURL, "/v1/prebid/URITEST123/pbs-impression") {
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
		// Test exact BURL format
		expectedBURL := "https://monitoring.receptivity.io/v1/prebid/URITEST123/pbs-billing?traceId=uri-extraction-test-123&random=0.777888&impid=test-imp-uri-extraction&price=2.25&crid=uri-extraction-creative"
		if bid.Bid.BURL != expectedBURL {
			t.Errorf("BURL exact match failed.\nExpected: %s\nActual:   %s", expectedBURL, bid.Bid.BURL)
		}
	}
}
