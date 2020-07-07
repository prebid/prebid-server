# Adding a New Bidder

This document describes how to add a new Bidder to Prebid Server. Bidders are responsible for reaching out to your Server to fetch Bids.

**NOTE**: To make everyone's lives easier, Bidders are expected to make Net bids (e.g. "If this ad wins, what will the publisher make?), not Gross ones.
Publishers can correct for Gross bids anyway by setting [Bid Adjustments](../endpoints/openrtb2/auction.md#bid-adjustments) to account for fees.

## Choose a Bidder Name

This name must be unique. Existing BidderNames can be found [here](../../openrtb_ext/bidders.go).

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
- `openrtb_ext/imp_{bidder}.go`: contract classes for your Bidder's params.
- `usersync/usersyncers/{bidder}.go`: A [Usersyncer](../../usersync/usersync.go) which returns cookie sync info for your bidder.
- `usersync/usersyncers/{bidder}_test.go`: Unit tests for your Usersyncer
- `static/bidder-params/{bidder}.json`: A [draft-4 json-schema](https://spacetelescope.github.io/understanding-json-schema/) which [validates your Bidder's params](https://www.jsonschemavalidator.net/).
- `static/bidder-info/{bidder}.yaml`: contains metadata (e.g. contact email, platform & media type support) about the adapter

Bidder implementations may assume that any params have already been validated against the defined json-schema.

### Long form video support
If bidder is going to support long form video make sure bidder has:

| Field          |Type                           |Description                       
|----------------|-------------------------------|-----------------------------|
|bid.bidVideo.PrimaryCategory | string | Category for the bid. Should be able to be translated to Primary ad server format|           
|TypedBid.bid.Cat | []string | Category for the bid. Should be an array with length 1 containing the value in IAB format|            
|TypedBid.BidVideo.Duration | int | AD duration in seconds|
|TypedBid.bid.Price | float | Bid price|

Note: `bid.bidVideo.PrimaryCategory` or `TypedBid.bid.Cat` should be specified.
To learn more about IAB categories, please refer to this convenience link (not the final official definition): [IAB categories](https://adtagmacros.com/list-of-iab-categories-for-advertisement/)

### Timeout notification support
This is an optional feature. If you wish to get timeout notifications when a bid request from PBS times out, you can implement the
`MakeTimeoutNotification` method in your adapter. If you do not wish timeout notification, do not implement the method.

`func (a *Adapter) MakeTimeoutNotification(req *adapters.RequestData) (*adapters.RequestData, []error)`

Here the `RequestData` supplied as an argument is the request returned from `MakeRequests` that timed out. If an adapter generates
multiple requests, and more than one of them times out, then there will be a call to `MakeTimeoutNotification` for each failed
request. The function should then return a `RequestData` object that will be the timeout notification to be sent to the bidder, or a list of errors encountered trying to create the timeout notification request. Timeout notifications will not generate subsequent timeout notifications if they timeout or fail.

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

Bidders should also define an `adapters/{bidder}/{bidder}test/params/race/{mediaType}.json` file for any supported
Media Types (banner, video, audio, or native). These files should contain a JSON object with all the bidder params
(required & optional) which are expected in supporting that video type. This will be used in automated tests which
check for race conditions across Bidders.

### Manual Tests

Build and start your server:

```bash
go build .
./prebid-server
```

Then `POST` an OpenRTB Request to `http://localhost:8000/openrtb2/auction`.

If at least one `request.imp[i].ext.{bidder}` is defined in your Request,
then your bidder should be called.

To test user syncs, [save a UID](../endpoints/setuid.md) using the FamilyName of your Usersyncer.
The next time you use `/openrtb2/auction`, the OpenRTB request sent to your Bidder should have
`BidRequest.User.BuyerUID` with the value you saved.

## Add your Bidder to the Exchange

Add a new [BidderName constant](../../openrtb_ext/bidders.go) for your {bidder}.
Update the [newAdapterMap function](../../exchange/adapter_map.go) to make your Bidder available in [auctions](../endpoints/openrtb2/auction).
Update the [NewSyncerMap function](../../usersync/usersync.go) to make your Bidder available for [usersyncs](../endpoints/setuid.md).

## Contribute

Finally, [Contribute](contributing.md) your Bidder to the project.

## Server requirements

**Note**: In order to be part of the auction, all bids must include:

- An ID
- An ImpID which matches one of the `Imp[i].ID`s from the incoming `BidRequest`
- A positive `Bid.Price`
- A `Bid.CrID` which uniquely identifies the Creative in the bid.

Bids which don't satisfy these standards will be filtered out before Prebid Server responds.
