## GET /bidders/params

This endpoint gets information about all the custom bidders params that Prebid Server supports.

### Returns

A JSON object whose keys are bidder codes, and values are Draft 4 JSON schemas which describe that bidders' params.

For example:

```
{
  "appnexus": { /* A json-schema describing AppNexus' bidder params */ },
  "rubicon": { /* A json-schema describing Rubicon's bidder params */ }
  ... all other bidders will have similar keys & values here ...
}
```

The exact contents of the json-schema values can be found [here](../../../static/bidder-params).

### See also

- [JSON schema homepage](http://json-schema.org/specification-links.html#draft-4)
- [Understanding JSON schema](https://spacetelescope.github.io/understanding-json-schema/)
