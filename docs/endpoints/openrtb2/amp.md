# Prebid Server AMP Endpoint

This document describes the behavior of the Prebid Server AMP endpoint in detail.
For a User's Guide, see the [AMP feature docs](http://prebid.org/dev-docs/show-prebid-ads-on-amp-pages.html).

## `GET /openrtb2/amp?tag_id={ID}`

The `tag_id` ID must reference a [Stored BidRequest](../../developers/stored-requests.md#stored-bidrequests).
For a thorough description of BidRequest JSON, see the [/openrtb2/auction](./auction.md) docs.

To be compatible with AMP, this endpoint behaves slightly different from normal `/openrtb2/auction` requests.

1. The Stored `request.imp` data must have exactly one element.
2. `request.imp[0].secure` will be always be set to `1`, because AMP requires all content to be `https`.
3. AMP query params will overwrite parts of your Stored Request. For details, see the Query Params section.

### Request

Valid Stored Requests for AMP pages must contain an `imp` array with exactly one element.  It is not necessary to include a `tmax` field in the Stored Request, as Prebid Server will always use the smaller of the AMP default timeout (1000ms) and the value passed via the `timeoutMillis` field of the `amp-ad.rtc-config`.

An example Stored Request is given below:

```
{
    "id": "some-request-id",
    "site": {
        "page": "prebid.org"
    },
    "ext": {
        "prebid": {
            "targeting": {
                "pricegranularity": {  // This is equivalent to the deprecated "pricegranularity": "medium"
                    "precision": 2,
                    "ranges": [{
                        "max": 20.00,
                        "increment": 0.10
                    }]
                }
            }
        }
    },
    "imp": [
        {
            "id": "some-impression-id",
            "banner": {}, // The sizes are defined is set by your AMP tag query params
            "ext": {
                "appnexus": {
                    // Insert parameters here
                },
                "rubicon": {
                    // Insert parameters here
                }
            }
        }
    ]
}
```

### Response

A sample response payload looks like this:

```
{
    "targeting": {
        "hb_bidder": "appnexus",
        "hb_bidder_appnexus": "appnexus",
        "hb_cache_id": "420d7329-30e8-4c4e-8eaa-fe937172e4e0",
        "hb_cache_id_appnexus": "420d7329-30e8-4c4e-8eaa-fe937172e4e0",
        "hb_pb": "0.50",
        "hb_pb_appnexus": "0.50",
        "hb_size": "300x250",
        "hb_size_appnexus": "300x250"
    }
    "errors": {
        "openx":[
            {
               "code": 1, 
               "message": "The request exceeded the timeout allocated"
            }
        ]
    }
}
```

In [the typical AMP setup](http://prebid.org/dev-docs/show-prebid-ads-on-amp-pages.html),
these targeting params will be sent to DFP.

Note that "errors" will only appear if there were any errors generated. They are identical to the "errors" field in the response.ext of the OpenRTB endpoint.

### Query Parameters

This endpoint supports the following query parameters:

1. `h` - `amp-ad` `height`
2. `w` - `amp-ad` `width`
3. `oh` - `amp-ad` `data-override-height`
4. `ow` - `amp-ad` `data-override-width`
5. `ms` - `amp-ad` `data-multi-size`
6. `curl` - the canonical URL of the page
7. `timeout` - the publisher-specified timeout for the RTC callout
   - A configuration option `amp_timeout_adjustment_ms` may be set to account for estimated latency so that Prebid Server can handle timeouts from adapters and respond to the AMP RTC request before it times out.
8. `debug` - When set to `1`, the respones will contain extra info for debugging.

For information on how these get from AMP into this endpoint, see [this pull request adding the query params to the Prebid callout](https://github.com/ampproject/amphtml/pull/14155) and [this issue adding support for network-level RTC macros](https://github.com/ampproject/amphtml/issues/12374).

If present, these will override parts of your Stored Request.

1. `ow`, `oh`, `w`, `h`, and/or `ms` will be used to set `request.imp[0].banner.format` if `request.imp[0].banner` is present.
2. `curl` will be used to set `request.site.page`
3. `timeout` will generally be used to set `request.tmax`. However, the Prebid Server host can [configure](../../developers/configuration.md) their deploy to reduce this timeout for technical reasons.
4. `debug` will be used to set `request.test`, causing the `response.debug` to have extra debugging info in it.

### Resolving Sizes

We strive to return ads with sizes which are valid for the `amp-ad` on your page. This logic intends to
track the logic used by `doubleclick` when resolving sizes used to fetch ads from their ad server.

Specifically:

1. If `ow` and `oh` exist, `request.imp[0].banner.format` will be a single element with `w: ow` and `h: oh`
2. If `ow` and `h` exist, `request.imp[0].banner.format` will be a single element with `w: ow` and `h: h`
3. If `oh` and `w` exist, `request.imp[0].banner.format` will be a single element with `w: w` and `h: oh`
4. If `ms` exists, `request.imp[0].banner.format` will contain an element for every size it uses.
5. If `w` and `h` exist, `request.imp[0].banner.format` will be a single element with `w: w` and `h: h`
6. If `w` _or_ `h` exist, it will be used to override _one_ of the dimensions inside each element of `request.imp[0].banner.format`
7. If none of these exist then the Stored Request values for `request.imp[0].banner.format` will be used without modification.
