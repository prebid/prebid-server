# Handling cookie syncs.

## POST /cookie_sync

This returns a set of URLs to enable cookie syncs across bidders. (See Prebid.js documentation?) The request
must supply a JSON object to define the list of bidders that may need to be synced.

```
{
    "uuid": "blahblah",
    "bidders": ["appnexus", "rubicon"]
}
```

The `uuid` field is due to be deprecated, as it is no longer needed. It just gets passed through unused into the response currently.
If the `bidders` field is an empty list, it will not supply any syncs. If the `bidders` field is omitted completely, it will attempt
to sync all bidders.

This will return a JSON object that will allow the client to request cookie syncs with bidders that still need to be synced:

```
{
    "uuid": "blahblah",
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

