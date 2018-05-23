# GDPR Mechanics

Within [GDPR](https://www.gdpreu.org/), Prebid Server behaves like a [data processor](https://www.gdpreu.org/the-regulation/key-concepts/data-controllers-and-processors/).
[Cookie syncs](./cookie-syncs.md) save the user ID for each Bidder in the cookie, and each Bidder's ID is sent back to it during the [auction](../endpoints/openrtb2/auction.md).
Prebid Server does not use this ID for any other reason.

## IDs during Auction

The [`/openrtb2/auction`](../endpoints/openrtb2/auction.md#gdpr) endpoint accepts `user.regs.gdpr` and `user.ext.consent` fields,
just like the [IAB recommends](https://iabtechlab.com/wp-content/uploads/2018/02/OpenRTB_Advisory_GDPR_2018-02.pdf).

## IDs during Cookie Syncs

The [`POST /cookie_sync`](../endpoints/cookieSync.md) endpoint accepts `gdpr` and `gdpr_consent` properties in the request body.

If the Prebid Server host company does not have consent to read/write cookies, it will return an empty response with no syncs.
Otherwise, it will limit the syncs to Bidders _which have consent to read/write cookies_. We do this for performance, because
it results in fewer syncs called on the page, and their sync endpoints will almost certainly read from the cookie anyway.

The [`/setuid`](../endpoints/setuid.md) endpoint accepts `gdpr` and `gdpr_consent` query params. This endpoint
will no-op if the Prebid Server host company does not have consent to read/write cookies.
