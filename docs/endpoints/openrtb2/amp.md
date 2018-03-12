# Prebid Server AMP Endpoint

This document describes the behavior of the Prebid Server AMP endpoint in detail.
For a User's Guide, see the [AMP feature docs](http://prebid.org/dev-docs/show-prebid-ads-on-amp-pages.html).

## `GET /openrtb2/amp?tag_id={ID}`

The `tag_id` ID must reference a [Stored BidRequest](../../developers/stored-requests.md#stored-bidrequests).
For a thorough description of BidRequest JSON, see the [/openrtb2/auction](./auction.md) docs.

Optionally, a param `debug=1` may be set, setting `"test": 1` on the request and resulting in [additional debug output](auction.md#debugging).

The only caveat is that AMP BidRequests must contain an `imp` array with one, and only one, impression object.

All AMP content must be secure, so this endpoint will enforce that request.imp[0].secure = 1. Saves on publishers forgetting to set this.

### Response

A sample response payload looks like this:

```
{
	"targeting": {
		"hb_pb": 1.30
		"hb_bidder": "appnexus"
		"hb_cache_host": "prebid-cache.adnexus.com"
		"hb_uuid": "6768-FB78-9890-7878"
	}
}
```

In [the typical AMP setup](http://prebid.org/dev-docs/show-prebid-ads-on-amp-pages.html),
these targeting params will be sent to DFP.
