# Adding a New Bidder

This document describes how to add a new Bidder to Prebid Server.

## Choose a Bidder Name

This name must be unique. Existing names can be found in [the Adapter Map](../../exchange/adapter_map.go).

Throughout the rest of this document, substitute `{bidder}` with the name you've chosen.

## Define your Bidder Params

Bidders may define their own APIs for Publishers pass custom values. It is _strongly encouraged_ that these not
duplicate values already present in the [OpenRTB 2.5 spec](https://www.iab.com/wp-content/uploads/2016/03/OpenRTB-API-Specification-Version-2-5-FINAL.pdf).

Publishers will send values for these parameters in `request.imp[i].ext.{bidder}` of
[the Auction endpoint](../endpoints/openrtb2/auction.md). Prebid Server will preprocess these so that
your bidder will access them at `request.imp[i].ext.bidder`--regardless of what your `{bidder}` name is.

## Implement your Bidder

Bidder implementations are scattered throughout several files.

- `adapters/{bidder}/{bidder}.go`: contains an implementation of [the Bidder interface](../../adapters/bidder.go).
- `adapters/{bidder}/info.yaml`: contains contact info for the adapter's maintainer.
- `openrtb_ext/imp_{bidder}.go`: contract classes for your Bidder's params.
- `static/bidder-params/{bidder}.json`: A [draft-4 json-schema](https://spacetelescope.github.io/understanding-json-schema/) which [validates your Bidder's params](https://www.jsonschemavalidator.net/).

Bidder implementations may assume that any params have already been validated against the defined json-schema.

## Test Your Bidder

### Automated Tests

Bidder tests live in two files:

- `adapters/{bidder}/{bidder}_test.go`: contains unit tests for your Bidder implementation.
- `adapters/{bidder}/params_test.go`: contains unit tests for your Bidder's JSON Schema params.

Since most Bidders communicate through HTTP using JSON bodies, you should
use the [JSON-test utilities](../../adapters/adapterstest/test_json.go).
This comes with several benefits, which are described in the source code docs.

If your HTTP requests don't use JSON, you'll need to write your tests in the code.
We expect to see at least 90% code coverage on each Bidder.

### Manual Tests

Build and start your server:

```bash
go build .
./prebid-server
```

Then `POST` an OpenRTB Request to `http://localhost:8000/openrtb2/auction`.

If at least one `request.imp[i].ext.{bidder}` is defined in your Request,
then your bidder should be called.

## Add your Bidder to the Exchange

Update the [the adapter map](../../exchange/adapter_map.go) with your Bidder.
This will also require a new [BidderName constant](../../openrtb_ext/bidders.go) for your Bidder.

## Contribute

Finally, [Contribute](contributing.md) your Bidder to the project.
