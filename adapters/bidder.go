package adapters

import (
	"encoding/base64"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
)

// Bidder describes how to connect to external demand.
type Bidder interface {
	// MakeRequests makes the HTTP requests which should be made to fetch bids.
	//
	// nil return values are acceptable, but nil elements *inside* those slices are not.
	//
	// The errors should contain a list of errors which explain why this bidder's bids will be
	// "subpar" in some way. For example: the request contained ad types which this bidder doesn't support.
	MakeRequests(request *openrtb.BidRequest) ([]*RequestData, []error)

	// MakeBids unpacks the server's response into Bids.
	//
	// The bids can be nil (for no bids), but should not contain nil elements.
	//
	// The errors should contain a list of errors which explain why this bidder's bids will be
	// "subpar" in some way. For example: the server response didn't have the expected format.
	MakeBids(internalRequest *openrtb.BidRequest, externalRequest *RequestData, response *ResponseData) ([]*TypedBid, []error)
}

// TypedBid packages the openrtb.Bid with any bidder-specific information that PBS needs to populate an
// openrtb_ext.ExtBidPrebid.
//
// TypedBid.Bid.Ext will become "response.seatbid[i].bid.ext.bidder" in the final OpenRTB response.
// TypedBid.BidType will become "response.seatbid[i].bid.ext.prebid.type" in the final OpenRTB response.
type TypedBid struct {
	Bid     *openrtb.Bid
	BidType openrtb_ext.BidType
}

// RequestData and ResponseData exist so that prebid-server core code can implement its "debug" functionality
// uniformly across all Bidders.
// It will also let us experiment with valyala/vasthttp vs. net/http without changing every adapter (see #152)

// ResponseData packages together information from the server's http.Response.
type ResponseData struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

// RequestData packages together the fields needed to make an http.Request.
type RequestData struct {
	Method  string
	Uri     string
	Body    []byte
	Headers http.Header
}

// ExtImpBidder can be used by Bidders to unmarshal any request.imp[i].ext.
type ExtImpBidder struct {
	Prebid *openrtb_ext.ExtImpPrebid `json:"prebid"`

	// Bidder contain the bidder-specific extension. Each bidder should unmarshal this using their
	// corresponding openrtb_ext.ExtImp{Bidder} struct.
	//
	// For example, the Appnexus Bidder should unmarshal this with an openrtb_ext.ExtImpAppnexus object.
	//
	// Bidder implementations may safely assume that this JSON has been validated by their
	// static/bidder-params/{bidder}.json file.
	Bidder openrtb.RawJSON `json:"bidder"`
}

func (r *RequestData) SetBasicAuth(username string, password string) {
	r.Headers.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)))
}
