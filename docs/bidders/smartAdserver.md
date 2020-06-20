# Smart Adserver Bidder

## Parameters
The `ext.smartadserver` object of impression bid requests supports the following parameters :
- "networkId" - Required. The network identifier you have been provided with.
- "siteId" - Optional. The site identifier from your campaign configuration.
- "pageId" - Optional. The page identifier from your campaign configuration.
- "formatId" - Optional. The format identifier from your campaign configuration.

The network identifier is provided by your Account Manager.
**Note:** The site, page and format identifiers have to all be provided or all empty.

## Examples

Without site/page/format : 
```
	"imp": [{
		"id": "some-impression-id",
		"banner": {
			"format": [{
				"w": 600,
				"h": 500
			}, {
				"w": 300,
				"h": 600
			}]
		},
		"ext": {
			"smartadserver": {
				"networkId": 73
			}
		}
	}]
```

With site/page/format : 

```
	"imp": [{
		"id": "some-impression-id",
		"banner": {
			"format": [{
				"w": 600,
				"h": 500
			}, {
				"w": 300,
				"h": 600
			}]
		},
		"ext": {
			"smartadserver": {
                "networkId": 73
                "siteId": 1,
                "pageId": 2,
                "formatId": 3
			}
		}
	}]
```