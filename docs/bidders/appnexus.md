# Appnexus Bidder

## Using Keywords

The `keywords` [bidder param](../../static/bidder-params/appnexus.json) will only work if
it's enabled for your Account with Appnexus.

**This permission is _distinct_ from the keywords feature used by Prebid.js.**

If you want to enable Appnexus keywords, contact your account manager.

## Display Manager Version

The AppNexus endpoint expects `imp.displaymanagerver` to be populated for mobile app sources
requests, however not all SDKs will populate this field. If the `imp.displaymanagerver` field
is not supplied for an `imp`, but `request.app.ext.prebid.source`
and `request.app.ext.prebid.version` are supplied, the adapter will fill in a value for
`diplaymanagerver`. It will concatonate the two `app` fields as `<source>-<version>` fo fill in
the empty `displaymanagerver` before sending the request to AppNexus.

## Test Request

The following test parameters can be used to verify that Prebid Server is working properly with the 
Appnexus adapter. This example includes an `imp` object with an Appnexus test placement ID and sizes
that would match with the test creative.

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
			"appnexus": {
				"placementId": 13144370
			}
		}
	}]
```