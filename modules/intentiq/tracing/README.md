# Overview

The `intentiq.tracing` module traces the `/openrtb2/auction` request lifecycle for a
hardcoded set of partners (see `Rules` in `rules.go`) and prints each collected trace packet
as JSON directly to stdout.

For every account whose resolved `Account.ID` matches a rule's `PartnerID`, the module
collects:

1. The incoming `BidRequest` and its timestamp.
2. Each outgoing `BidRequest` sent to a bidder, its timestamp, and the bidder's name.
3. Each incoming `BidResponse` from a bidder, its timestamp, and the bidder's name.
4. The final auction `BidResponse` sent back to the client, and its timestamp.

Tracing for a partner stops (permanently) as soon as either the `Duration` since its first
traced request elapses, or its `TracePacketsAmount` budget - shared across all four packet
types above - is exhausted. The module never mutates the request/response or rejects the
auction; it is a pure observability sidecar, and only acts on the `/openrtb2/auction`
endpoint.
