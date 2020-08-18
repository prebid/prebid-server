# Adform Bidder

## Parameters

The `ext.adform` object of impression bid requests supports the following parameters :
- "mid" - Required. Adform placement id.
- "priceType" - Optional. An expected price type (net or gross) of bids.
- "mkv" - Optional. Comma-separated key-value pairs.
- "mkw" - Optional. Comma-separated keywords.
- "minp" - Optional. Minimum CPM price.
- "cdims" - Optional. Comma-separated creative dimentions.
- "url" - Optional. Custom targeting URL.

## Test Request

The following test parameters can be used to verify that Prebid Server is working properly with the 
Adform adapter. This example includes an `imp` object with an Adform test placement id and other available targeting options.

```
	"imp": [{
		"id": "some-impression-id",
		"banner": {
			"format": [{
				"w": 300,
				"h": 250
			}, {
				"w": 300,
				"h": 300
			}]
		},
		"ext": {
			"adform": {
				"mid": 828628,
				"minp": 0.03,
				"cdims": "300x250",
				"mkv": "city:NY"
			}
		}
	}]
```

For any additional information, please contact publishers@adform.com
