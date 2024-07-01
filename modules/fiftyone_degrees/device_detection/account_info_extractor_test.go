package device_detection

import (
	"encoding/json"
	"github.com/prebid/openrtb/v20/openrtb2"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPublisherIdExtractionFromSiteRequest(t *testing.T) {
	var payload = []byte(`
		{
			"imp": [
			{
				"ext": {
				"prebid": {
					"storedrequest": {
					"id": "p-bid-imp-test-005-banner-320x50"
					},
					"adunitcode": "p-bid-config-test-005"
				},
				"data": {
					"pbadslot": "p-bid-config-test-005"
				}
				},
				"id": "p-bid-config-test-005",
				"banner": {
				"topframe": 1,
				"format": [
					{
					"w": 50,
					"h": 320
					}
				]
				}
			}
			],
			"site": {
			"domain": "prebid.postindustria.com",
			"publisher": {
				"domain": "postindustria.com",
				"id": "p-bid-config-test-005"
			},
			"page": "https://prebid.postindustria.com/playground/page_with_prebid_js.html",
			"ref": "https://prebid.postindustria.com/playground/"
			},
			"device": {
			"w": 345,
			"h": 737,
			"dnt": 0,
			"ua": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
			"language": "en",
			"sua": {
				"source": 2,
				"platform": {
				"brand": "macOS",
				"version": [
					"14",
					"0",
					"0"
				]
				},
				"browsers": [
				{
					"brand": "Not A(Brand",
					"version": [
					"99",
					"0",
					"0",
					"0"
					]
				},
				{
					"brand": "Google Chrome",
					"version": [
					"121",
					"0",
					"6167",
					"184"
					]
				},
				{
					"brand": "Chromium",
					"version": [
					"121",
					"0",
					"6167",
					"184"
					]
				}
				],
				"mobile": 0,
				"model": "",
				"architecture": "arm"
			}
			},
			"user": {
			"data": [
				{
				"ext": {
					"segtax": 601,
					"segclass": "4"
				},
				"segment": [
					{
					"id": "140"
					}
				],
				"name": "topics.authorizedvault.com"
				},
				{
				"ext": {
					"segtax": 601,
					"segclass": "4"
				},
				"segment": [
					{
					"id": "140"
					}
				],
				"name": "pa.openx.net"
				}
			]
			},
			"id": "3732c953-9af6-4fc0-95d6-f4fd45d4f90f",
			"test": 0,
			"ext": {
			"prebid": {
				"auctiontimestamp": 1708629526728,
				"targeting": {
				"includewinners": true,
				"includebidderkeys": false
				},
				"channel": {
				"name": "pbjs",
				"version": "v8.37.0"
				},
				"createtids": false
			}
			}
		}
	`)

	extractor := NewAccountInfoExtractor()
	accountInfo := extractor.Extract(payload)

	assert.Equal(t, accountInfo.Id, "p-bid-config-test-005")
}

func TestPublisherIdExtractionFromMobileRequest(t *testing.T) {
	var payload = []byte(`
		{
			"app": {
			"bundle": "org.prebid.PrebidDemoSwift",
			"ext": {
				"prebid": {
				"source": "prebid-mobile",
				"version": "2.1.6"
				}
			},
			"name": "PrebidDemoSwift",
			"publisher": {
				"id": "p-bid-config-test-005"
			},
			"ver": "1.0"
			},
			"device": {
			"connectiontype": 2,
			"ext": {
				"atts": 0,
				"ifv": "1B8EFA09-FF8F-4123-B07F-7283B50B3870"
			},
			"h": 852,
			"ifa": "00000000-0000-0000-0000-000000000000",
			"language": "en",
			"lmt": 1,
			"make": "Apple",
			"model": "iPhone",
			"os": "iOS",
			"osv": "17.0",
			"pxratio": 3,
			"ua": "Mozilla\/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit\/605.1.15 (KHTML, like Gecko) Mobile\/15E148",
			"w": 393
			},
			"ext": {
			"prebid": {
				"storedrequest": {
				"id": "p-bid-config-test-005"
				},
				"targeting": {
				
				}
			}
			},
			"id": "E17E1E6B-D3C4-4D09-8093-6E473390B717",
			"imp": [
			{
				"banner": {
				"api": [
					5
				],
				"format": [
					{
					"h": 50,
					"w": 320
					}
				]
				},
				"clickbrowser": 1,
				"ext": {
				"prebid": {
					"storedrequest": {
					"id": "p-bid-imp-test-005-banner-320x50"
					}
				}
				},
				"id": "830D74AA-D1F6-497D-8F3D-226BC99EA4FA",
				"instl": 0,
				"secure": 1
			}
			],
			"source": {
			"tid": "2B3E9156-4097-46FA-A58D-48A547BDB5FF"
			}
		}
	`)

	extractor := NewAccountInfoExtractor()
	accountInfo := extractor.Extract(payload)

	assert.Equal(t, accountInfo.Id, "p-bid-config-test-005")
}

func TestEmptyPublisherIdExtraction(t *testing.T) {
	var payload = []byte(`{}`)

	bReq := &openrtb2.BidRequest{}

	err := json.Unmarshal(payload, &bReq)
	assert.NoError(t, err)

	extractor := NewAccountInfoExtractor()
	accountInfo := extractor.Extract(payload)

	assert.Nil(t, accountInfo)
}

func TestExtractionFromEmptyPayload(t *testing.T) {
	extractor := NewAccountInfoExtractor()
	accountInfo := extractor.Extract(nil)

	assert.Nil(t, accountInfo)
}
