# Server Based Default Aliases

Some aliases are common, so it makes some sense to predefine them on the server, rather than adding them to all incoming requests (or all stored requests). We can define the aliases in a file to be read at server startup and injected into all requests.

## Config Options

Two config options are exposed to support this feature.
```
aliases:
    filename : /path/to/aliases.json
    info     : false
```

The `filename` option is the path/filename of a JSON file containing the bidder `aliases` JSON as documented in the [openrtb2 docs](../endpoints/openrtb2/auction.md)
```
{
    "aliases": {
        "districtm": "appnexus"
    }
}
```
This will be JSON merged into the `ext` element of the incomming requests to define common aliases for the server.

The `info` option determines if the alised bidders will be exposed on the `/info` endpoints. If true the alias name will be added to the list returned by
`/info/bidders` and the info JSON for the core bidder will be coppied into `/info/bidder/{biddername}` with the addition of the field 
`"alias_of": "{coreBidder}"` to indicate that it is an aliases, and of which core bidder. Turning the info support on may be useful for hosts
that want to support automation around the `/info` endpoints that will include the predefined aliases.  This config option may be deprecated in a future
version to promote a consistency in the endpoint functionality, depending on the perceived need for the option.


