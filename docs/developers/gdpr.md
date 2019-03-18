# GDPR Mechanics

Within the framework of [GDPR](https://www.gdpreu.org/), Prebid Server behaves like a [data processor](https://www.gdpreu.org/the-regulation/key-concepts/data-controllers-and-processors/).
[Cookie syncs](./cookie-syncs.md) save the user ID for each Bidder in the cookie, and each Bidder's ID is sent back to that Bidder during the [auction](../endpoints/openrtb2/auction.md).
Prebid Server does not use this ID for any other reason.

## IDs during Auction

The [`/openrtb2/auction`](../endpoints/openrtb2/auction.md#gdpr) endpoint accepts `user.regs.gdpr` and `user.ext.consent` fields,
[as recommended by the IAB](https://iabtechlab.com/wp-content/uploads/2018/02/OpenRTB_Advisory_GDPR_2018-02.pdf).

## IDs during Cookie Syncs

The [`POST /cookie_sync`](../endpoints/cookieSync.md) endpoint accepts `gdpr` and `gdpr_consent` properties in the request body.

If the Prebid Server host company does not have consent to read/write cookies, `/cookie_sync` will return an empty response with no syncs.
Otherwise, it will return a response limited to syncs for Bidders that have consent to read/write cookies.
This limitation is in place for performance reasons; it results in fewer syncs called on the page, and their
sync endpoints will almost certainly read from the cookie anyway.

The [`/setuid`](../endpoints/setuid.md) endpoint accepts `gdpr` and `gdpr_consent` query params. This endpoint
will no-op if the Prebid Server host company does not have consent to read/write cookies.

## Handling the params

For all endpoints, `gdpr` should be `1` if GDPR is in effect, `0` if not, and omitted if the caller isn't sure.
`gdpr_consent` should be an [unpadded base64-URL](https://tools.ietf.org/html/rfc4648#page-7) encoded [Vendor Consent String](https://github.com/InteractiveAdvertisingBureau/GDPR-Transparency-and-Consent-Framework/blob/master/Consent%20string%20and%20vendor%20list%20formats%20v1.1%20Final.md#vendor-consent-string-format-).

`gdpr_consent` is required if `gdpr` is `1` and ignored if `gdpr` is `0`. If `gdpr` is omitted, the Prebid Server
host company can decide whether it behaves like a `1` or `0` through the [app configuration](./configuration.md).
Callers are encouraged to send the `gdpr_consent` param if `gdpr` is omitted.
