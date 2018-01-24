# Prebid Server AMP Endpoint

This document describes the behavior of the Prebid Server amp endpoint, including:

- Request/response formats

## AMP RTC
Legacy support for prebid in AMP is being retired. RTC (Real Time Config) in AMP provides a new route
to inject prebid demand into an AMP page. RTC can only send a `GET` request, and only recieve ad 
targeting key/value pairs, so the prebid AMP protocol is a slimmed down version of the OpenRTB2
protocol. This document will only describe the changes from OpenRTB2 support, please refer to that
documentation for anything not specified here.

## `GET /openrtb2/amp?tag_id=<ID>`

This endpoint runs an auction with the OpenRTB 2.5 bid request refernced by the given `tag_id`.
The `tag_id` refernces a stored request that is a full OpenRTB2 request. Obviously there needs to be a
unique `tag_id` for every unique OpenRTB request configuration desired. Once the request is recieved
by prebid server, the stored request is recalled, and processing is passed on to the OpenRTB2 code
to conduct an auction as normal.

## Sample response

The response contains only the targeting key/value pairs produced in the auction. This includes the
key/values for cached ads. The creative delivered in response to the prebid demand must be able to
refernce the cache in order to pull the ad, as AMP/RTC does not allow for transmitting the ad to
the creative through the RTC response. A sample RTC response is shown below:

```
{
	"targeting": {
		"hb_pb": 1.30
		"hb_bidder": "appnexus"
		"hb_cache_host": "prebid-cache.adnexus.com"
		"hb_cache_id": "6768-FB78-9890-7878"
	}
}
```

The key/values are pulled from all the bids in the OpenRTB2 response that have a `cache_id` associated
with them. An uncached ad can never be delivered of course.