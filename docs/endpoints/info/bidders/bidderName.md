# Prebid Server Bidders

## `GET /info/bidders/{bidderName}`

This endpoint returns some metadata about the Bidder whose name is `{bidderName}`.
Legal values for `{bidderName}` can be retrieved from the [/info/bidders](../bidders.md) endpoint.

### Sample Response

This endpoint returns JSON like:

```
{
  "maintainer": {
    "email": "info@prebid.org"
  },
  "capabilities": {
    "app": {
      "mediaTypes": [
        "banner",
        "native"
      ]
    },
    "site": {
      "mediaTypes": [
        "banner",
        "video",
        "native"
      ]
    }
  }
}
```

The fields hold the following information:

- `maintainer.email`: A contact email for the Bidder's maintainer. In general, Bidder bugs should be logged as [issues](https://github.com/prebid/prebid-server/issues)... but this contact email may be useful in case of emergency.
- `capabilities.app.mediaTypes`: A list of media types this Bidder supports from Mobile Apps.
- `capabilities.site.mediaTypes`: A list of media types this Bidder supports from Web pages.

If `capabilities.app` or `capabilities.site` do not exist, then this Bidder does not support that platform.
OpenRTB Requests which define a `request.app` or `request.site` property will fail if a
`request.imp[i].ext.{bidderName}` exists for a Bidder which doesn't support them.
