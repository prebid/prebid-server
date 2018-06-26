# Starting Cookie Syncs

This endpoint is used during cookie syncs. For technical details, see the
[Cookie Sync developer docs](../developers/cookie-syncs.md).

## POST /cookie_sync

### Sample Request
This returns a set of URLs to enable cookie syncs across bidders. (See Prebid.js documentation?) The request
must supply a JSON object to define the list of bidders that may need to be synced.

```
{
    "bidders": ["appnexus", "rubicon"],
    "gdpr": 1,
    "gdpr_consent": "BONV8oqONXwgmADACHENAO7pqzAAppY"
}
```

`bidders` is optional. If present, it limits the endpoint to return syncs for bidders defined in the list.

`gdpr` is optional. It should be 1 if GDPR is in effect, 0 if not, and omitted if the caller is unsure.

`gdpr_consent` is required if `gdpr` is `1`, and optional otherwise. If present, it should be an [unpadded base64-URL](https://tools.ietf.org/html/rfc4648#page-7) encoded [Vendor Consent String](https://github.com/InteractiveAdvertisingBureau/GDPR-Transparency-and-Consent-Framework/blob/master/Consent%20string%20and%20vendor%20list%20formats%20v1.1%20Final.md#vendor-consent-string-format-).

If `gdpr` is  omitted, callers are still encouraged to send `gdpr_consent` if they have it.
Depending on how the Prebid Server host company has configured their servers, they may or may not require it for cookie syncs.


If the `bidders` field is an empty list, it will not supply any syncs. If the `bidders` field is omitted completely, it will attempt
to sync all bidders.

### Sample Response

This will return a JSON object that will allow the client to request cookie syncs with bidders that still need to be synced:

```
{
    "status": "ok",
    "bidder_status": [
        {
            "bidder": "appnexus",
            "usersync": {
                "url": "someurl.com",
                "type": "redirect",
                "supportCORS": false
            }
        }
    ]
}
```
