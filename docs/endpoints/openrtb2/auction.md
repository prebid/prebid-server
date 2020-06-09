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

This is a sample OpenRTB 2.5 bid request for a Xandr (formerly AppNexus) test placement. Please note, the Xandr Ad Server will only
respond with a bid if the "test" field is set to 1.

```
{
  "id": "some-request-id",
  "test": 1,
  "site": {
    "page": "prebid.org"
  },
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
        "placementId": 12883451
      }
    }
  }],
  "tmax": 500
}
```

Additional examples can be found in [endpoints/openrtb2/sample-requests/valid-whole](../../../endpoints/openrtb2/sample-requests/valid-whole).

### Sample Response

This endpoint will respond with either:

- An OpenRTB 2.5 bid response, or
- HTTP 400 if the request is malformed, or
- HTTP 503 if the account or app specified in the request is blacklisted

This is the corresponding response to the above sample OpenRTB 2.5 bid request, with the `ext.debug` field removed and the `seatbid.bid.adm` field simplified.

```
{
  "id": "some-request-id",
  "seatbid": [{
    "seat": "appnexus",
    "bid": [{
      "id": "145556724130495288",
      "impid": "some-impression-id",
      "price": 0.01,
      "adm": "<script type=\"application/javascript\">...</script>",
      "adid": "107987536",
      "adomain": [
        "appnexus.com"
      ],
      "iurl": "https://nym1-ib.adnxs.com/cr?id=107987536",
      "cid": "3532",
      "crid": "107987536",
      "w": 600,
      "h": 500,
      "ext": {
        "prebid": {
          "type": "banner",
          "video": {
            "duration": 0,
            "primary_category": ""
          }
        },
        "bidder": {
          "appnexus": {
            "brand_id": 1,
            "auction_id": 7311907164510136364,
            "bidder_id": 2,
            "bid_ad_type": 0
          }
        }
      }
    }]
  }],
  "cur": "USD",
  "ext": {
    "responsetimemillis": {
      "appnexus": 10
    },
    "tmaxrequest": 500
  }
}
```

### OpenRTB Extensions

#### Conventions

OpenRTB 2.5 permits exchanges to define their own extensions to any object from the spec.
These fall under the `ext` field of JSON objects.

If `ext` is defined on an object, Prebid Server uses the following conventions:

1. `ext` in "request objects" uses `ext.prebid` and/or `ext.{anyBidderCode}`.
2. `ext` on "response objects" uses `ext.prebid` and/or `ext.bidder`.
The only exception here is the top-level `BidResponse`, because it's bidder-independent.

`ext.{anyBidderCode}` and `ext.bidder` extensions are defined by bidders.
`ext.prebid` extensions are defined by Prebid Server.

Exceptions are made for extensions with "standard" recommendations:

- `request.user.ext.digitrust` -- To support Digitrust
- `request.regs.ext.gdpr` and `request.user.ext.consent` -- To support GDPR
- `request.regs.us_privacy` -- To support CCPA
- `request.site.ext.amp` -- To identify AMP as the request source
- `request.app.ext.source` and `request.app.ext.version` -- To support identifying the displaymanager/SDK in mobile apps. If given, we expect these to be strings.

#### Bid Adjustments

Bidders [are encouraged](../../developers/add-new-bidder.md) to make Net bids. However, there's no way for Prebid to enforce this.
If you find that some bidders use Gross bids, publishers can adjust for it with `request.ext.prebid.bidadjustmentfactors`:

```
{
  "ext": {
    "prebid": {
      "bidadjustmentfactors": {
        "appnexus": 0.8,
        "rubicon": 0.7
      }
    }
  }
}
```

This may also be useful for publishers who want to account for different discrepancies with different bidders.

#### Targeting

Targeting refers to strings which are sent to the adserver to
[make header bidding possible](http://prebid.org/overview/intro.html#how-does-prebid-work).

`request.ext.prebid.targeting` is an optional property which causes Prebid Server
to set these params on the response at `response.seatbid[i].bid[j].ext.prebid.targeting`.

**Request format** (optional param `request.ext.prebid.targeting`)

```
{
  "ext": {
    "prebid": {
      "targeting": {
        "pricegranularity": {
          "precision": 2,
          "ranges": [{
            "max": 20.00,
            "increment": 0.10 // This is equivalent to the deprecated "pricegranularity": "medium"
          }]
        },
        "includewinners": false, // Optional param defaulting to true
        "includebidderkeys": false // Optional param defaulting to true
      }
    }
  }
}
```
The list of price granularity ranges must be given in order of increasing `max` values. If `precision` is omitted, it will default to `2`. The minimum of a range will be 0 or the previous `max`. Any cmp above the largest `max` will go in the `max` pricebucket.

For backwards compatibility the following strings will also be allowed as price granularity definitions. There is no guarantee that these will be honored in the future. "One of ['low', 'med', 'high', 'auto', 'dense']" See [price granularity definitions](http://prebid.org/prebid-mobile/adops-price-granularity.html)

One of "includewinners" or "includebidderkeys" must be true (both default to true if unset). If both were false, then no targeting keys would be set, which is better configured by omitting targeting altogether.

MediaType PriceGranularity (PBS-Java only) - when a single OpenRTB request contains multiple impressions with different mediatypes, or a single impression supports multiple formats, the different mediatypes may need different price granularities. If `mediatypepricegranularity` is present, `pricegranularity` would only be used for any mediatypes not specified. 

```
{
  "ext": {
    "prebid": {
      "targeting": {
        "mediatypepricegranularity": {
          "banner": {
            "ranges": [
              {"max": 20, "increment": 0.5}
            ]
          },
          "video": {
            "ranges": [
              {"max": 10, "increment": 1},
              {"max": 20, "increment": 2},
              {"max": 50, "increment": 5}
            ]
          }
        }
      },
      "includewinners": true
    }
  }
}
```

**Response format** (returned in `bid.ext.prebid.targeting`)

```
{
  "seatbid": [{
    "bid": [{
      ...
      "ext": {
        "prebid": {
          "targeting": {
            "hb_bidder_{bidderName}": "The seatbid.seat which contains this bid",
            "hb_size_{bidderName}": "A string like '300x250' using bid.w and bid.h for this bid",
            "hb_pb_{bidderName}": "The bid.cpm, rounded down based on the price granularity."
          }
        }
      }
    }]
  }]
}
```

The winning bid for each `request.imp[i]` will also contain `hb_bidder`, `hb_size`, and `hb_pb`
(with _no_ {bidderName} suffix). To prevent these keys, set `request.ext.prebid.targeting.includeWinners` to false.

**NOTE**: Targeting keys are limited to 20 characters. If {bidderName} is too long, the returned key
will be truncated to only include the first 20 characters.

#### Cookie syncs

Each Bidder should receive their own ID in the `request.user.buyeruid` property.
Prebid Server has three ways to populate this field. In order of priority:

1. If the request payload contains `request.user.buyeruid`, then that value will be sent to all Bidders.
In most cases, this is probably a bad idea.

2. The request payload can store a `buyeruid` for each Bidder by defining `request.user.ext.prebid.buyeruids` like so:

```
{
  "user": {
    "ext": {
      "prebid": {
        "buyeruids": {
          "appnexus": "some-appnexus-id",
          "rubicon": "some-rubicon-id"
        }
      }
    }
  }
}
```

Prebid Server's core logic will preprocess the request so that each Bidder sees their own value in the `request.user.buyeruid` field.

3. Prebid Server will use its Cookie to map IDs for each Bidder.

If you're using [Prebid.js](https://github.com/prebid/Prebid.js), this is happening automatically.

If you're using another client, you can populate the Cookie of the Prebid Server host with User IDs
for each Bidder by using the `/cookie_sync` endpoint, and calling the URLs that it returns in the response.

#### Native Request

For each native request, the `assets` object's `id` field must not be defined. Prebid Server will set this automatically, using the index of the asset in the array as the ID.


#### Bidder Aliases

Requests can define Bidder aliases if they want to refer to a Bidder by a separate name.
This can be used to request bids from the same Bidder with different params. For example:

```
{
  "imp": [{
    "id": "some-impression-id",
    "video": {
      "mimes": ["video/mp4"]
    },
    "ext": {
      "appnexus": {
        "placementId": 123
      },
      "districtm": {
        "placementId": 456
      }
    }
  }],
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
like the original, except that the Response will contain separate SeatBids, and any Targeting keys
will be formed using the alias' name.

If an alias overlaps with a core Bidder's name, then the alias will take precedence.
This prevents breaking API changes as new Bidders are added to the project.

For example, if the Request defines an alias like this:

```
  "aliases": {
    "appnexus": "rubicon"
  }
```

then any `imp.ext.appnexus` params will actually go to the **rubicon** adapter.
It will become impossible to fetch bids from AppNexus within that Request.

#### Bidder Response Times

`response.ext.responsetimemillis.{bidderName}` tells how long each bidder took to respond.
These can help quantify the performance impact of "the slowest bidder."

#### Bidder Errors

`response.ext.errors.{bidderName}` contains messages which describe why a request may be "suboptimal".
For example, suppose a `banner` and a `video` impression are offered to a bidder
which only supports `banner`.

In cases like these, the bidder can ignore the `video` impression and bid on the `banner` one.
However, the publisher can improve performance by only offering impressions which the bidder supports.

For example, a request may return this in `response.ext`

```
{
  "ext": {
    "errors": {
      "appnexus": [{
        "code": 2,
        "message": "A hybrid Banner/Audio Imp was offered, but Appnexus doesn't support Audio."
      }],
      "rubicon": [{
        "code": 1,
        "message": "The request exceeded the timeout allocated"
      }]
    }
  }
}
```

The codes currently defined are:

```
0   NoErrorCode
1   TimeoutCode
2   BadInputCode
3   BadServerResponseCode
999 UnknownErrorCode
```

#### Debugging

`response.ext.debug.httpcalls.{bidder}` will be populated **only if** `request.test` **was set to 1**.

This contains info about every request and response sent by the bidder to its server.
It is only returned on `test` bids for performance reasons, but may be useful during debugging.

`response.ext.debug.resolvedrequest` will be populated **only if** `request.test` **was set to 1**.

This contains the request after the resolution of stored requests and implicit information (e.g. site domain, device user agent).

#### Stored Requests

`request.imp[i].ext.prebid.storedrequest` incorporates a [Stored Request](../../developers/stored-requests.md) from the server.

A typical `storedrequest` value looks like this:

```
{
  "imp": [{
    "ext": {
      "prebid": {
        "storedrequest": {
          "id": "some-id"
        }
      }
    }
  }]
}
```

For more information, see the docs for [Stored Requests](../../developers/stored-requests.md).

#### Cache bids

Bids can be temporarily cached on the server by sending the following data as `request.ext.prebid.cache`:

```
{
  "ext": {
    "prebid": {
      "cache": {
        "bids": {},
        "vastxml": {}
      }
    }
  }
}
```

Both `bids` and `vastxml` are optional, but one of the two is required if you want to cache bids. This property will have no effect
unless `request.ext.prebid.targeting` is also set in the request.

If `bids` is present, Prebid Server will make a _best effort_ to include these extra
`bid.ext.prebid.targeting` keys:

- `hb_cache_id`: On the highest overall Bid in each Imp.
- `hb_cache_id_{bidderName}`: On the highest Bid from {bidderName} in each Imp.

Clients _should not assume_ that these keys will exist, just because they were requested, though.
If they exist, the value will be a UUID which can be used to fetch Bid JSON from [Prebid Cache](https://github.com/prebid/prebid-cache).
They may not exist if the host company's cache is full, having connection problems, or other issues like that.

If `vastxml` is present, PBS will try to add analogous keys `hb_uuid` and `hb_uuid_{bidderName}`.
In addition to the caveats above, these will exist _only if the relevant Bids are for Video_.
If they exist, the values can be used to fetch the bid's VAST XML from Prebid Cache directly.

These options are mainly intended for certain limited Prebid Mobile setups, where bids cannot be cached client-side.

#### GDPR

Prebid Server supports the IAB's GDPR recommendations, which can be found [here](https://iabtechlab.com/wp-content/uploads/2018/02/OpenRTB_Advisory_GDPR_2018-02.pdf).

This adds two optional properties:

- `request.user.ext.consent`: Is the consent string required by the IAB standards.
- `request.regs.ext.gdpr`: Is 0 if the caller believes that the user is *not* under GDPR, 1 if the user *is* under GDPR, and undefined if we're not certain.

These fields will be forwarded to each Bidder, so they can decide how to process them.

#### Interstitial support
Additional support for interstitials is enabled through the addition of two fields to the request:
device.ext.prebid.interstitial.minwidthperc and device.ext.interstial.minheightperc
The values will be numbers that indicate the minimum allowed size for the ad, as a percentage of the base side. For example, a width of 600 and "minwidthperc": 60 would allow ads with widths from 360 to 600 pixels inclusive.

Example:
```
{
  "imp": [{
    ...
    "banner": {
      ...
    }
    "instl": 1,
    ...
  }]
  "device": {
    ...
    "h": 640,
    "w": 320,
    "ext": {
      "prebid": {
        "interstitial": {
          "minwidthperc": 60,
          "minheightperc": 60
        }
      }
    }
  }
}
```

PBS receiving a request for an interstitial imp and these parameters set, it will rewrite the format object within the interstitial imp. If the format array's first object is a size, PBS will take it as the max size for the interstitial. If that size is 1x1, it will look up the device's size and use that as the max size. If the format is not present, it will also use the device size as the max size. (1x1 support so that you don't have to omit the format object to use the device size)
PBS with interstitial support will come preconfigured with a list of common ad sizes. Preferentially organized by weighing the larger and more common sizes first. But no guarantees to the ordering will be made. PBS will generate a new format list for the interstitial imp by traversing this list and picking the first 10 sizes that fall within the imp's max size and minimum percentage size. There will be no attempt to favor aspect ratios closer to the original size's aspect ratio. The limit of 10 is enforced to ensure we don't overload bidders with an overlong list. All the interstitial parameters will still be passed to the bidders, so they may recognize them and use their own size matching algorithms if they prefer.

#### Currency Support

To set the desired 'ad server currency', use the standard OpenRTB `cur` attribute. Note that Prebid Server only looks at the first currency in the array.

```
    "cur": ["USD"]
```

If you want or need to define currency conversion rates (e.g. for currencies that your Prebid Server doesn't support),
define ext.prebid.currency.rates. (Currently supported in PBS-Java only)

```
"ext": {
  "prebid": {
	  "currency": {
		  "rates": {
			  "USD": { "UAH": 24.47, "ETB": 32.04 }
		  }
	  }
  }
}
```

If it exists, a rate defined in ext.prebid.currency.rates has the highest priority.
If a currency rate doesn't exist in the request, the external file will be used.

#### Supply Chain Support


Basic supply chains are passed to Prebid Server on `source.ext.schain` and passed through to bid adapters. Prebid Server does not currently offer the ability to add a node to the supply chain.

Bidder-specific schains (PBS-Java only):

```
ext.prebid.schains: [
   { bidders: ["bidderA"], schain: { SCHAIN OBJECT 1}},
   { bidders: ["*"], schain: { SCHAIN OBJECT 2}}
]
```
In this scenario, Prebid Server sends the first schain object to `bidderA` and the second schain object to everyone else.

If there's already an source.ext.schain and a bidder is named in ext.prebid.schains (or covered by the wildcard condition), ext.prebid.schains takes precedent.

#### Rewarded Video (PBS-Java only)

Rewarded video is a way to incentivize users to watch ads by giving them 'points' for viewing an ad. A Prebid Server
client can declare a given adunit as eligible for rewards by declaring `imp.ext.prebid.is_rewarded_inventory:1`.

#### Stored Responses (PBS-Java only)

While testing SDK and video integrations, it's important, but often difficult, to get consistent responses back from bidders that cover a range of scenarios like different CPM values, deals, etc. Prebid Server supports a debugging workflow in two ways:

- a stored-auction-response that covers multiple bidder responses
- multiple stored-bid-responses at the bidder adapter level

**Single Stored Auction Response ID**

When a storedauctionresponse ID is specified:

- the rest of the ext.prebid block is irrelevant and ignored
- nothing is sent to any bidder adapter for that imp
- the response retrieved from the stored-response-id is assumed to be the entire contents of the seatbid object corresponding to that impression.

This request:
```
{
  "test":1,
  "tmax":500,
  "id": "test-auction-id",
  "app": { ... },
  "ext": {
      "prebid": {
             "targeting": {},
             "cache": { "bids": {} }
       }
  },
  "imp": [
    {
      "id": "a",
      "ext": { "prebid": { "storedauctionresponse": { "id": "1111111111" } } }
    },
    {
      "id": "b",
      "ext": { "prebid": { "storedauctionresponse": { "id": "22222222222" } } }
    }
  ]
}
```

Will result in this response, assuming that the ids exist in the appropriate DB table read by Prebid Server:
```
{
    "id": "test-auction-id",
    "seatbid": [
        {
             // BidderA bids from storedauctionresponse=1111111111
             // BidderA bids from storedauctionresponse=22222222
        },
       {
             // BidderB bids from storedauctionresponse=1111111111
             // BidderB bids from storedauctionresponse=22222222
       }
  ]
}
```

**Multiple Stored Bid Response IDs**

In contrast to what's outlined above, this approach lets some real auctions take place while some bidders have test responses that still exercise bidder code. For example, this request:

```
{
  "test":1,
  "tmax":500,
  "id": "test-auction-id",
  "app": { ... },
  "ext": {
      "prebid": {
             "targeting": {},
             "cache": { "bids": {} }
       }
  },
  "imp": [
    {
      "id": "a",
      "ext": {
          "prebid": {
            "storedbidresponse": [
                  { "bidder": "BidderA", "id": "333333" },
                  { "bidder": "BidderB", "id": "444444" },
             ]
           } 
      }
    },
    {
      "id": "b",
      "ext": {
          "prebid": {
            "storedbidresponse": [
                  { "bidder": "BidderA", "id": "5555555" },
                  { "bidder": "BidderB", "id": "6666666" },
             ]
           } 
      }
    }
  ]
}
```
Could result in this response:

```
{
    "id": "test-auction-id",
    "seatbid": [
        {
             "bid": [
             // contents of storedbidresponse=3333333 as parsed by bidderA adapter
             // contents of storedbidresponse=5555555 as parsed by bidderA adapter
             ]
        },
       {
             // contents of storedbidresponse=4444444 as parsed by bidderB adapter
             // contents of storedbidresponse=6666666 as parsed by bidderB adapter
       }
  ]
}
```

Setting up the storedresponse DB entries is the responsibility of each Prebid Server host company.

See Prebid.org troubleshooting pages for how to utilize this feature within the context of the browser.


#### User IDs (PBS-Java only)

Prebid Server adapters can support the [Prebid.js User ID modules](http://prebid.org/dev-docs/modules/userId.html) by reading the following extensions and passing them through to their server endpoints:

```
{
    "user": {
        "ext": {
            "eids": [{
                "source": "adserver.org",
                "uids": [{
                    "id": "111111111111",
                    "ext": {
                        "rtiPartner": "TDID"
                    }
                }]
            },
            {
                "source": "pubcommon",
                "id":"11111111"
            }
            ],
            "digitrust": {
                "id": "11111111111",
                "keyv": 4
            }
        }
    }
}
```

#### First Party Data Support (PBS-Java only)

This is the Prebid Server version of the Prebid.js First Party Data feature. It's a standard way for the page (or app) to supply first party data and control which bidders have access to it.

It specifies where in the OpenRTB request non-standard attributes should be passed. For example:

```
{
    "ext": {
       "prebid": {
           "data": { "bidders": [ "rubicon", "appnexus" ] }  // these are the bidders allowed to see protected data
       }
    },
    "site": {
         "keywords": "",
         "search": "",
         "ext": {
             data: { GLOBAL CONTEXT DATA } // only seen by bidders named in ext.prebid.data.bidders[]
         }
    },
    "user": {
        "keywords": "", 
        "gender": "", 
        "yob": 1999, 
        "geo": {},
        "ext": {
            data: { GLOBAL USER DATA }  // only seen by bidders named in ext.prebid.data.bidders[]
        }
    },
    "imp": [
        "ext": {
            "context": {
                "keywords": "",
                "search": "",
                "data": { ADUNIT SPECFIC CONTEXT DATA }  // can be seen by all bidders
            }
         }
    ]
```

Prebid Server enforces the data permissioning

So before passing the values to the bidder adapters, core will:

1. check for ext.prebid.data.bidders
1. if it exists, store it locally, but remove it from the OpenRTB before being sent to the adapters
1. As the OpenRTB request is being sent to each adapter:
    1. if ext.prebid.data.bidders exists in the original request, and this bidder is on the list then copy site.ext.data, app.ext.data, and user.ext.data to their bidder request -- otherwise don't copy those blocks
    1. copy other objects as normal

Each adapter must be coded to read the values from these locations and pass it to their endpoints appropriately.

### OpenRTB Ambiguities

This section describes the ways in which Prebid Server **implements** OpenRTB spec ambiguous parts.

- `request.cur`: If `request.cur` is not specified in the bid request, Prebid Server will consider it as being `USD` whereas OpenRTB spec doesn't mention any default currency for bid request.
```request.cur: ['USD'] // Default value if not set```


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

#### Determining Bid Security (http/https)

In the OpenRTB spec, `request.imp[i].secure` says:

> Flag to indicate if the impression requires secure HTTPS URL creative assets and markup,
> where 0 = non-secure, 1 = secure. If omitted, the secure state is unknown, but non-secure
> HTTP support can be assumed.

In Prebid Server, an `https` request which does not define `secure` will be forwarded to Bidders with a `1`.
Publishers who run `https` sites and want insecure ads can still set this to `0` explicitly.

### See also

- [The OpenRTB 2.5 spec](https://www.iab.com/wp-content/uploads/2016/03/OpenRTB-API-Specification-Version-2-5-FINAL.pdf)
