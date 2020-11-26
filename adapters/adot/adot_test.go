package adot

import (
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"strings"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "adottest", NewAdotAdapter("https://dsp.adotmob.com/headerbidding/bidrequest"))
}

var jsonBidReq = []byte(`{
							"id": "reqId",
							"imp": [
								{
									"id": "impId",
									"banner": {
										"format": [
											{
												"w": 320,
												"h": 480
											}
										],
										"w": 320,
										"h": 480
									},
									"tagid": "ee234aac-114",
									"bidfloorcur": "EUR",
									"ext": {
										"adot": {
											"parallax": true
										}
									}
								}
							],
							"app": {
								"id": "0",
								"name": "test-adot-integration",
								"domain": "www.geev.com",
								"cat": [
									"IAB1"
								],
								"publisher": {
									"id": "1",
									"name": "GEEV"
								}
							},
							"device": {
								"ua": "Mozilla/5.0 (Linux; Android 8.1.0; A11_Y Build/O11019; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/75.0.3770.101 Mobile Safari/537.36",
								"geo": {
									"lat": 48.8566,
									"lon": 2.35222,
									"type": 2,
									"country": "France",
									"zip": "75004"
								},
								"ip": "::1",
								"devicetype": 4,
								"make": "Vmobile",
								"model": "A11_Y",
								"os": "Android",
								"osv": "27",
								"language": "français",
								"carrier": "WIFI",
								"connectiontype": 2,
								"ifa": "AD0D3F0B-A407-70DA-AD07-3E2A5E4AD073",
								"ext": {
									"is_app": 1
								}
							},
							"user": {
								"id": "AD0D3F0B-A407-70DA-AD07-3E2A5E4AD073",
								"buyeruid": "2432"
							},
							"at": 2,
							"cur": [
								"EUR",
								"USD"
							],
							"regs": {
								"ext": {
									"gdpr": 0
								}
							},
							"ext": {}
						}`,
)

// Test properties of Adapter interface
func TestAdotUrl(t *testing.T) {
	adotAdapter := NewAdotAdapter("someUrl")

	if strings.Compare(adotAdapter.endpoint, "someUrl") == 1 {
		t.Errorf("The endpoint should be the same as " + adotAdapter.endpoint)
	}
}

// Test the reqyest with the parallax parameter
func TestRequestWithParallax(t *testing.T) {
	var bidReq *openrtb.BidRequest
	if err := json.Unmarshal(jsonBidReq, &bidReq); err != nil {
		fmt.Println("error: ", err.Error())
	}

	reqJSON, err := json.Marshal(bidReq)
	if err != nil {
		t.Errorf("The request should not be the same, because their is a parallax param in ext.")
	}

	adotJson := addParallaxIfNecessary(reqJSON)
	stringReqJSON := string(adotJson)

	if stringReqJSON == string(reqJSON) {
		t.Errorf("The request should not be the same, because their is a parallax param in ext.")
	}

	if strings.Count(stringReqJSON, "parallax: true") == 2 {
		t.Errorf("The parallax was not well add in the request")
	}
}

// Test the request without the parallax parameter
func TestRequestWithoutParallax(t *testing.T) {
	stringBidReq := strings.Replace(string(jsonBidReq), "\"parallax\": true", "", -1)
	jsonReq := []byte(stringBidReq)

	reqJSON := addParallaxIfNecessary(jsonReq)

	if strings.Contains(string(reqJSON), "parallax") {
		t.Errorf("The request should not contains parallax param " + string(reqJSON))
	}
}
