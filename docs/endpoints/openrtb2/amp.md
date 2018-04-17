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

### Query Parameters

This endpoint supports the following query parameters:

1. `h` - `amp-ad` `height`
2. `w` - `amp-ad` `width`
3. `oh` - `amp-ad` `data-override-height`
4. `ow` - `amp-ad` `data-override-width`
5. `ms` - `amp-ad` `data-multi-size`
6. `curl` - the canonical URL of the page
7. `purl` - the page URL
8. `timeout` - the publisher-specified timeout for the RTC callout
   - A configuration option `amp_timeout_adjustment_ms` may be set to account for estimated latency so that Prebid Server can handle timeouts from adapters and respond to the AMP RTC request before it times out.
9. `debug` - When set to `1`, will set `"test": 1` on outgoing OpenRTB requests and will return additional debug information in the response `ext`.

For more information see [this pull request adding the query params to the Prebid callout](https://github.com/ampproject/amphtml/pull/14155) and [this issue adding support for network-level RTC macros](https://github.com/ampproject/amphtml/issues/12374).
