# SmartRTB Bidder

[SmartRTB](https://smrtb.com/) supports the following parameters to be present in the `ext` object of impression requests:

- "pub_id" type string - Required. Publisher ID assigned to you.
- "zone_id" type string - Optional. Enables mapping for further settings and reporting in the Marketplace UI.
- "force_bid" type bool - Optional. If zone ID is mapped, this may be set to always return fake sample bids (banner, video)

Please contact us to create a new Smart RTB Marketplace account, and for any assistance in configuration.
You may email info@smrtb.com for inquiries.

## Test Request

This sample request is our global test placement and should always return a branded banner bid.

```
	{
        "id": "abc",
        "site": {
          "page": "prebid.org"
        },
       "imp": [{
    		"id": "test",
    		"banner": {
    			"format": [{
    				"w": 300,
    				"h": 250
    			}]
    		},
    		"ext": {
    			"smartrtb": {
                    "pub_id": "test",
                    "zone_id": "N4zTDq3PPEHBIODv7cXK",
                    "force_bid": true
    			}
    		}
    	}]
    }
```
