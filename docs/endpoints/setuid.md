# Saving User Syncs

This endpoint is used during cookie syncs. For technical details, see the
[Cookie Sync developer docs](../developers/cookie-syncs.md).

## `GET /setuid`

This endpoint saves a UserID for a Bidder in the Cookie. Saved IDs will be recognized for 7 days before being considered "stale" and being re-synced.

### Query Params

- `bidder`: The FamilyName of the [Usersyncer](../../usersync/usersync.go) which is being synced.
- `uid`: The ID which the Bidder uses to recognize this user.

### Sample request

`GET http://prebid.site.com/setuid?bidder=adnxs&uid=12345`
