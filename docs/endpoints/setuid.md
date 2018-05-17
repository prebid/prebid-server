# Saving User Syncs

This endpoint is used during cookie syncs. For technical details, see the
[Cookie Sync developer docs](../developers/cookie-syncs.md).

## `GET /setuid`

This endpoint saves a UserID for a Bidder in the Cookie. Saved IDs will be recognized for 7 days before being considered "stale" and being re-synced.

### Query Params

- `bidder`: The FamilyName of the [Usersyncer](../../usersync/usersync.go) which is being synced.
- `uid`: The ID which the Bidder uses to recognize this user. If undefined, the UID for `bidder` will be deleted.
- `gdpr`: This should be `1` if GDPR is in effect, `0` if not, and undefined if the caller isn't sure
- `gdpr_consent`: This is required if `gdpr` is one, and optional (but encouraged) otherwise. If present, it should be an [unpadded base64-URL](https://tools.ietf.org/html/rfc4648#page-7) encoded [Vendor Consent String](https://github.com/InteractiveAdvertisingBureau/GDPR-Transparency-and-Consent-Framework/blob/master/Consent%20string%20and%20vendor%20list%20formats%20v1.1%20Final.md#vendor-consent-string-format-).

If the `gdpr` and `gdpr_consent` params are included, this endpoint will _not_ write a cookie unless:

1. The Vendor ID set by the Prebid Server host company has permission to save cookies for that user.
2. The Prebid Server host company did not configure it to run with GDPR support.

If in doubt, contact the company hosting Prebid Server and ask if they're GDPR-ready.

### Sample request

`GET http://prebid.site.com/setuid?bidder=adnxs&uid=12345&gdpr=1&gdpr_consent=BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw`
