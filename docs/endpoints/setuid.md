# Saving User Syncs

This endpoint is used by bidders to sync user IDs with Prebid Server.
If a user runs an [auction](./openrtb2/auction.md) _without_ specifying `request.user.buyeruid`,
then Prebid Server will set it to the uid saved here before forwarding the request to the Bidder.

## `GET /setuid`

This endpoint can be used to save UserIDs for a Bidder. These UIDs will be saved in a Cookie,
so they will not translate across Prebid Server instances hosted on different domains.

Saved IDs will be recognized for 7 days before being considered "stale" and being re-synced.

### Query Params

- `bidder`: The FamilyName of the [Usersyncer](../../usersync/usersync.go) which is being synced.
- `uid`: The User's ID in the given domain.

### Sample request

`GET http://prebid.site.com/setuid?bidder=adnxs&uid=12345`