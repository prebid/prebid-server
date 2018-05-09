# Cookie Sync Technical Details

This document describes the mechancis of a Prebid Server cookie sync.

## Motivation

Many Bidders track users through Cookies. Since Bidders will generally serve ads from a different domain
than where Prebid Server is hosted, those cookies must be consolidated under the Prebid Server domain so
that they can be sent to each demand source in [/openrtb2/auction](../endpoints/openrtb2/auction.md) calls.

## How to do it?

Start by calling [`/cookie_sync`](../endpoints/cookieSync.md). For each element of `response.bidder_status`,
call `GET element.usersync.url`. That endpoint should respond with a redirect which will complete the cookie sync.

## Mechanics

Bidders who support cookie syncs must implement an endpoint under their domain which accepts
an encoded URI for redirects. For example:

> GET some-bidder-domain.com/usersync-url?redirectUri=www.prebid-domain.com%2Fsetuid%3Fbidder%3Dsomebidder%26uid%3D%24UID

This example endpoint would URL-decode the `redirectUri` param to get `www.prebid-domain.com/setuid?bidder=somebidder&uid=$UID`.
It would then replace the `$UID` macro with the user's ID from their cookie. Supposing this user's ID was "132",
it would then return a redirect to `www.prebid-domain.com/setuid?bidder=somebidder&uid=132`.

Prebid Server would then save this ID mapping of `somebidder: 132` under the cookie at `prebid-domain.com`.

When the client then calls `www.prebid-domain.com/openrtb2/auction`, the ID for `somebidder` will be available in the Cookie.
Prebid Server will then stick this into `request.user.buyeruid` in the OpenRTB request it sends to `somebidder`'s Bidder.
