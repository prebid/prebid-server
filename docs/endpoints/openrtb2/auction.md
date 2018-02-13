# Prebid Server Auction Endpoint

This document describes the behavior of the Prebid Server auction endpoint, including:

- Request/response formats
- OpenRTB extensions
- Debugging and performance tips
- How user syncing works
- Departures from OpenRTB

## `POST /openrtb2/auction`

This endpoint runs an auction with the given OpenRTB 2.5 bid request.

### Sample request

The following is a "hello world" request which fetches the [Prebid sample ad](http://prebid.org/examples/pbjs_demo.html).

```
{
  "id": "some-request-id",
  "imp": [
    {
      "id": "some-impression-id",
      "banner": {
        "format": [
          {
            "w": 300,
            "h": 250
          },
          {
            "w": 300,
            "h": 600
          }
        ]
      },
      "ext": {
        "appnexus": {
          "placementId": 10433394
        }
      }
    }
  ],
  "test": 1,
  "tmax": 500
}
```

### Sample Response

This endpoint will respond with either:

- An OpenRTB 2.5 BidResponse, or
- An HTTP 400 status code if the request is malformed

See below for a "hello world" response.

```
{
  "id": "some-request-id",
  "seatbid": [
    {
      "seat": "appnexus"
      "bid": [
        {
          "id": "4625436751433509010",
          "impid": "some-impression-id",
          "price": 0.5,
          "adm": "<script type=\"application/javascript\">...</script>",
          "adid": "29681110",
          "adomain": [
            "appnexus.com"
          ],
          "iurl": "http://nym1-ib.adnxs.com/cr?id=29681110",
          "cid": "958",
          "crid": "29681110",
          "w": 300,
          "h": 250,
          "ext": {
            "bidder": {
              "appnexus": {
                "brand_id": 1,
                "auction_id": 6127490747252133000,
                "bidder_id": 2
              }
            }
          }
        }
      ]
    }
  ]
}
```

### OpenRTB Extensions

#### Conventions

OpenRTB 2.5 permits exchanges to define their own extensions to any object from the spec.
These fall under the `ext` property of JSON objects.

If `ext` is defined on an object, Prebid Server uses the following conventions:

1. `ext` in "Request objects" uses `ext.prebid` and/or `ext.{anyBidderCode}`.
2. `ext` on "Response objects" uses `ext.prebid` and/or `ext.bidder`.
The only exception here is the top-level `BidResponse`, because it's bidder-independent.

`ext.{anyBidderCode}` and `ext.bidder` extensions are defined by bidders.
`ext.prebid` extensions are defined by Prebid Server.

#### Targeting

Targeting refers to strings which are sent to the adserver to
[make header bidding possible](http://prebid.org/overview/intro.html#how-does-prebid-work).

`request.ext.prebid.targeting` is an optional property which causes Prebid Server
to set these params on the response at `response.seatbid[i].bid[j].ext.prebid.targeting`.

**Request format** (optional param `request.ext.prebid.targeting`)

```
{
  "pricegraularity": "One of ['low', 'med', 'high', 'auto', 'dense']", // Required property.
}
```

**Response format** (returned in `bid.ext.prebid.targeting`)

```
{
  "hb_bidder_{bidderName}": "The seatbid.seat which contains this bid",
  "hb_size_{bidderName}": "A string like '300x250' using bid.w and bid.h for this bid",
  "hb_pb_{bidderName}": "The bid.cpm, rounded down based on the price granularity."
}
```

The winning bid for each `request.imp[i]` will also contain `hb_bidder`, `hb_size`, and `hb_pb`
(with _no_ {bidderName} suffix).

**NOTE**: Targeting keys are limited to 20 characters. If {bidderName} is too long, the returned key
will be truncated to only include the first 20 characters.

#### Bidder Aliases

Requests can define Bidder aliases if they want to refer to a Bidder by a separate name.
This can be used to request bids from the same Bidder with different params. For example:

```
{
  "imp": [
    {
      "id": "some-impression-id",
      "video": {
        "mimes": ["video/mp4"]
      },
      "ext": {
        "appnexus: {
          "placementId": 123
        },
        "districtm": {
          "placementId": 456
        }
      }
    }
  ],
  "ext": {
    "prebid": {
      "aliases": {
        "districtm": "appnexus"
      }
    }
  }
}
```

For all intents and purposes, the alias will be treated as another Bidder. This new Bidder will behave exactly
like the original, except that the Response will contain seprate SeatBids, and any Targeting keys
will be formed using the alias' name.

If an alias overlaps with a core Bidder's name, then the alias will take precedence.
This prevents breaking API changes as new Bidders are added to the project.

For example, if the Request defines an alias like this:

```
{
  "aliases": {
    "appnexus": "rubicon"
  }
}
```

then any `imp.ext.appnexus` params will actually go to the **rubicon** adapter.
It will become impossible to fetch bids from Appnexus within that Request.

#### Bidder Response Times

`response.ext.responsetimemillis.{bidderName}` tells how long each bidder took to respond.
These can help quantify the performance impact of "the slowest bidder."

`response.ext.errors.{bidderName}` contains messages which describe why a request may be "suboptimal".
For example, suppose a `banner` and a `video` impression are offered to a bidder
which only supports `banner`.

In cases like these, the bidder can ignore the `video` impression and bid on the `banner` one.
However, the publisher can improve performance by only offering impressions which the bidder supports.

#### Debugging

`response.ext.debug.httpcalls.{bidder}` will be populated **only if** `request.test` **was set to 1**.

This contains info about every request and response sent by the bidder to its server.
It is only returned on `test` bids for performance reasons, but may be useful during debugging.

#### Stored Requests

`request.imp[i].ext.prebid.storedrequest` incorporates a [Stored Request](../../developers/stored-requests.md) from the server.

A typical `storedrequest` value looks like this:

```
{
  "id": "some-id"
}
```

For more information, see the docs for [Stored Requests](../../developers/stored-requests.md).

### OpenRTB Differences

This section describes the ways in which Prebid Server **breaks** the OpenRTB spec.

#### Allowed Bidders

Prebid Server returns a 400 on requests which define `wseat` or `bseat`.
We may add support for these in the future, if there's compelling need.

Instead, an impression is only offered to a bidder if `bidrequest.imp[i].ext.{bidderName}` exists.

This supports publishers who want to sell different impressions to different bidders.

#### Deprecated Properties

This endpoint returns a 400 if the request contains deprecated properties (e.g. `imp.wmin`, `imp.hmax`).

The error message in the response should describe how to "fix" the request to make it legal.
If the message is unclear, please [log an issue](https://github.com/prebid/prebid-server/issues)
or [submit a pull request](https://github.com/prebid/prebid-server/pulls) to improve it.


### See also

- [The OpenRTB 2.5 spec](https://www.iab.com/wp-content/uploads/2016/03/OpenRTB-API-Specification-Version-2-5-FINAL.pdf)
