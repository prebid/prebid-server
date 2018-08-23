/*
Package openrtb_ext defines all the input validation for Prebid Server's extensions to the OpenRTB 2.5 spec.

Most of these are defined by simple contract classes.

One notable exception is the bidder params, which have more complex validation rules.
These are validated by a BidderParamValidator, which relies on the json-schemas from
static/bidder-params/{bidder}.json
*/
package openrtb_ext
