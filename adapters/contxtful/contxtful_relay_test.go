package contxtful

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

// TestContxtfulRelay tests that the adapter correctly transforms input OpenRTB requests
// to the expected format for the Contxtful relay endpoint
func TestContxtfulRelay(t *testing.T) {
	// Create a test adapter
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid", // Use the dynamic endpoint template
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Load the test OpenRTB request
	inputRequest := createTestOpenRTBRequest(t)

	// Execute the adapter's request transformation logic
	requestsData, errs := bidder.MakeRequests(inputRequest, &adapters.ExtraRequestInfo{})

	// Verify no errors occurred
	assert.Empty(t, errs, "Expected no errors")
	assert.Len(t, requestsData, 1, "Expected one request")

	// Verify the request URI matches what we expect (should be resolved with customerId)
	expectedEndpoint := "https://prebid.receptivity.io/v1/pbs/test-customer-123/bid"
	assert.Equal(t, expectedEndpoint, requestsData[0].Uri, "Unexpected request URI")
	assert.Equal(t, "POST", requestsData[0].Method, "Unexpected HTTP method")

	// Decode the request body to verify structure
	var requestBody map[string]interface{}
	err := json.Unmarshal(requestsData[0].Body, &requestBody)
	assert.NoError(t, err, "Should unmarshal request body without errors")

	// Verify basic structure of the request
	assert.Contains(t, requestBody, "ortb2", "Request body should contain ortb2 field")
	assert.Equal(t, "1", inputRequest.Imp[0].ID, "Imp ID should be preserved")

	// Verify headers
	assert.Equal(t, "application/json;charset=utf-8", requestsData[0].Headers.Get("Content-Type"), "Content-Type header should be set")
	assert.Equal(t, "application/json", requestsData[0].Headers.Get("Accept"), "Accept header should be set")
}

// createTestOpenRTBRequest creates a test OpenRTB request similar to the example provided
func createTestOpenRTBRequest(t *testing.T) *openrtb2.BidRequest {
	// Sample request based on the structure provided
	rawJson := `{
		"imp": [{
			"ext": {
				"data": {
					"divId": "/19968336/header-bid-tag-0",
					"adg_rtd": {
						"adunit_position": "8x151"
					},
					"placement": "in_article"
				},
				"bidder": {
					"placementId": "p10000001",
					"customerId": "test-customer-123"
				},
				"prebid": {
					"adunitcode": "/19968336/header-bid-tag-0"
				}
			},
			"id": "1",
			"banner": {
				"topframe": 1,
				"format": [{
						"w": 300,
						"h": 250
					},
					{
						"w": 300,
						"h": 600
					}
				]
			},
			"secure": 1
		}],
		"test": 1,
		"app": {
			"id": "test-app-id",
			"bundle": "test.bundle"
		},
		"site": {
			"domain": "prometheus.receptivity.io",
			"publisher": {
				"domain": "receptivity.io",
				"id": "1"
			},
			"page": "https://example.receptivity.io/",
			"ext": {
				"data": {
					"documentLang": "en",
					"adg_rtd": {
						"uid": "4b9f3464-8177-4b6f-9bf5-baf69be33b10",
						"pageviewId": "d3c75c57-3cb5-48d9-b7b6-3ea2e401759b",
						"features": {
							"page_dimensions": "2034x668",
							"viewport_dimensions": "2034x652",
							"user_timestamp": "1746128988",
							"dom_loading": "512"
						},
						"session": {
							"rnd": 0.17883197907537263,
							"pages": 3,
							"new": true,
							"vwSmplg": 0,
							"vwSmplgNxt": 0,
							"expiry": 1746141471202,
							"lastActivityTime": 1746139671202,
							"id": "45811c08-b3f0-4395-9040-a87077325ced"
						}
					}
				}
			}
		},
		"device": {
			"w": 2560,
			"h": 1080,
			"dnt": 1,
			"ua": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36",
			"language": "fr",
			"ext": {
				"vpw": 2034,
				"vph": 344
			},
			"sua": {
				"source": 1,
				"platform": {
					"brand": "macOS"
				},
				"browsers": [{
						"brand": "Chromium",
						"version": ["135"]
					},
					{
						"brand": "Not-A.Brand",
						"version": ["8"]
					}
				],
				"mobile": 0
			}
		},
		"ext": {
			"prebid": {
				"auctiontimestamp": 1746143432301,
				"targeting": {
					"includewinners": true,
					"includebidderkeys": false
				},
				"bidderconfig": [{
					"bidders": ["pubmatic"],
					"config": {
						"ortb2": {
							"user": {
								"data": [{
									"name": "contxtful",
									"ext": {
										"events": "eyJ1aSI6eyJwb3NpdGlvbiI6eyJ4Ijo3Ny43OTY4NzUsInkiOjkyLjI5Mjk2ODc1LCJ0aW1lc3RhbXBNcyI6NDQ3MDQuMzAwMDAwMDExOTJ9LCJzY3JlZW4iOnsidG9wTGVmdCI6eyJ4IjowLCJ5IjowfSwid2lkdGgiOjIwMzQsImhlaWdodCI6MzQ0LCJ0aW1lc3RhbXBNcyI6NDUwNzMuMjAwMDAwMDE3ODh9fX0=",
										"pos": "eyIvMTk5NjgzMzYvaGVhZGVyLWJpZC10YWctMCI6eyJwIjp7IngiOjgsInkiOjE1MX0sInYiOnRydWUsInQiOiJkaXYifSwiLzE5OTY4MzM2L2hlYWRlci1iaWQtdGFnLTEiOnsicCI6eyJ4Ijo4LCJ5Ijo0NzB9LCJ2Ijp0cnVlLCJ0IjoiZGl2In19",
										"sm": "a68e8343-fdcb-4b51-84d6-d4ac8c9b04b8",
										"params": {
											"ev": "v1",
											"ci": "1Pw320rMi1BNmV0C8TEX7LlYD"
										}
									}
								}]
							}
						}
					}
				}],
				"debug": true,
				"channel": {
					"name": "pbjs",
					"version": "v9.39.0-pre"
				},
				"createtids": false
			},
			"tmaxmax": 3000
		},
		"id": "3dc63dbf-dcfa-4645-bf2a-23d32fb759d2",
		"tmax": 1000
	}`

	var req openrtb2.BidRequest
	err := json.Unmarshal([]byte(rawJson), &req)
	assert.NoError(t, err, "Should unmarshal test request without errors")

	return &req
}

// TestContxtfulTransformStructure performs a deeper test of the request transformation structure
func TestContxtfulTransformStructure(t *testing.T) {
	// Create a test adapter
	bidder, buildErr := Builder(openrtb_ext.BidderContxtful, config.Adapter{
		Endpoint: "https://prebid.receptivity.io/v1/pbs/{{.AccountID}}/bid", // Use the dynamic endpoint template
	}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Load the test OpenRTB request
	inputRequest := createTestOpenRTBRequest(t)

	// Execute the adapter's request transformation logic
	requestsData, errs := bidder.MakeRequests(inputRequest, &adapters.ExtraRequestInfo{})

	if errs != nil && len(errs) > 0 {
		t.Fatalf("MakeRequests returned errors: %v", errs)
	}

	// Verify we got a request back
	assert.Len(t, requestsData, 1, "Expected one request")

	// Decode the request body to inspect the structure in detail
	var requestBody map[string]interface{}
	err := json.Unmarshal(requestsData[0].Body, &requestBody)
	assert.NoError(t, err, "Should unmarshal request body without errors")

	// Check for the expected fields in the transformed request
	assert.Contains(t, requestBody, "ortb2", "Request should contain ortb2")

	// Check that ortb2 is a map
	_, ok := requestBody["ortb2"].(map[string]interface{})
	assert.True(t, ok, "ortb2 should be a map")

	// Verify the bidRequests field structure
	assert.Contains(t, requestBody, "bidRequests", "Request should contain bidRequests")
	bidRequests, ok := requestBody["bidRequests"].([]interface{})
	assert.True(t, ok, "bidRequests should be an array")
	assert.GreaterOrEqual(t, len(bidRequests), 1, "bidRequests should have at least one entry")

	// Verify bidRequest contains the expected fields
	bidRequest, ok := bidRequests[0].(map[string]interface{})
	assert.True(t, ok, "bidRequest should be a map")
	assert.Contains(t, bidRequest, "bidder", "bidRequest should contain bidder")
	assert.Contains(t, bidRequest, "params", "bidRequest should contain params")

	// Verify the bidderRequest field is present
	assert.Contains(t, requestBody, "bidderRequest", "Request should contain bidderRequest")
}
