package adapters

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/currency"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// Bidder describes how to connect to external demand.
type Bidder interface {
	// MakeRequests makes the HTTP requests which should be made to fetch bids.
	//
	// Bidder implementations can assume that the incoming BidRequest has:
	//
	//   1. Only {Imp.Type, Platform} combinations which are valid, as defined by the static/bidder-info.{bidder}.yaml file.
	//   2. Imp.Ext of the form {"bidder": params}, where "params" has been validated against the static/bidder-params/{bidder}.json JSON Schema.
	//
	// nil return values are acceptable, but nil elements *inside* those slices are not.
	//
	// The errors should contain a list of errors which explain why this bidder's bids will be
	// "subpar" in some way. For example: the request contained ad types which this bidder doesn't support.
	//
	// If the error is caused by bad user input, return an errortypes.BadInput.
	MakeRequests(request *openrtb2.BidRequest, reqInfo *ExtraRequestInfo) ([]*RequestData, []error)

	// MakeBids unpacks the server's response into Bids.
	//
	// The bids can be nil (for no bids), but should not contain nil elements.
	//
	// The errors should contain a list of errors which explain why this bidder's bids will be
	// "subpar" in some way. For example: the server response didn't have the expected format.
	//
	// If the error was caused by bad user input, return a errortypes.BadInput.
	// If the error was caused by a bad server response, return a errortypes.BadServerResponse
	MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *RequestData, response *ResponseData) (*BidderResponse, []error)
}

// TimeoutBidder is used to identify bidders that support timeout notifications.
type TimeoutBidder interface {
	Bidder

	// MakeTimeoutNotification functions much the same as MakeRequests, except it is fed the bidder request that timed out,
	// and expects that only one notification "request" will be generated. A use case for multiple timeout notifications
	// has not been anticipated.
	//
	// Do note that if MakeRequests returns multiple requests, and more than one of these times out, MakeTimeoutNotice will be called
	// once for each timed out request.
	MakeTimeoutNotification(req *RequestData) (*RequestData, []error)
}

// BidderResponse wraps the server's response with the list of bids and the currency used by the bidder.
//
// Currency declaration is not mandatory but helps to detect an eventual currency mismatch issue.
// From the bid response, the bidder accepts a list of valid currencies for the bid.
// The currency is the same across all bids.
type BidderResponse struct {
	Currency             string
	Bids                 []*TypedBid
	FledgeAuctionConfigs []*openrtb_ext.FledgeAuctionConfig
}

// NewBidderResponseWithBidsCapacity create a new BidderResponse initialising the bids array capacity and the default currency value
// to "USD".
//
// bidsCapacity allows to set initial Bids array capacity.
// By default, currency is USD but this behavior might be subject to change.
func NewBidderResponseWithBidsCapacity(bidsCapacity int) *BidderResponse {
	return &BidderResponse{
		Currency: "USD",
		Bids:     make([]*TypedBid, 0, bidsCapacity),
	}
}

// NewBidderResponse create a new BidderResponse initialising the bids array and the default currency value
// to "USD".
//
// By default, Bids capacity will be set to 0.
// By default, currency is USD but this behavior might be subject to change.
func NewBidderResponse() *BidderResponse {
	return NewBidderResponseWithBidsCapacity(0)
}

// TypedBid packages the openrtb2.Bid with any bidder-specific information that PBS needs to populate an
// openrtb_ext.ExtBidPrebid.
//
// TypedBid.Bid.Ext will become "response.seatbid[i].bid.ext.bidder" in the final OpenRTB response.
// TypedBid.BidMeta will become "response.seatbid[i].bid.ext.prebid.meta" in the final OpenRTB response.
// TypedBid.BidType will become "response.seatbid[i].bid.ext.prebid.type" in the final OpenRTB response.
// TypedBid.BidVideo will become "response.seatbid[i].bid.ext.prebid.video" in the final OpenRTB response.
// TypedBid.DealPriority is optionally provided by adapters and used internally by the exchange to support deal targeted campaigns.
// TypedBid.Seat new seat under which the bid should pe placed. Default is adapter name
type TypedBid struct {
	Bid          *openrtb2.Bid
	BidMeta      *openrtb_ext.ExtBidPrebidMeta
	BidType      openrtb_ext.BidType
	BidVideo     *openrtb_ext.ExtBidPrebidVideo
	DealPriority int
	Seat         openrtb_ext.BidderName
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
	ImpIDs  []string
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
	Bidder json.RawMessage `json:"bidder"`

	AuctionEnvironment openrtb_ext.AuctionEnvironmentType `json:"ae,omitempty"`
}

func (r *RequestData) SetBasicAuth(username string, password string) {
	r.Headers.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)))
}

type ExtraRequestInfo struct {
	PbsEntryPoint              metrics.RequestType
	GlobalPrivacyControlHeader string
	CurrencyConversions        currency.Conversions
	PreferredMediaType         openrtb_ext.BidType
}

func NewExtraRequestInfo(c currency.Conversions) ExtraRequestInfo {
	return ExtraRequestInfo{
		CurrencyConversions: c,
	}
}

// ConvertCurrency converts a given amount from one currency to another, or returns:
//   - Error if the `from` or `to` arguments are malformed or unknown ISO-4217 codes.
//   - ConversionNotFoundError if the conversion mapping is unknown to Prebid Server
//     and not provided in the bid request.
func (r ExtraRequestInfo) ConvertCurrency(value float64, from, to string) (float64, error) {
	if rate, err := r.CurrencyConversions.GetRate(from, to); err == nil {
		return value * rate, nil
	} else {
		return 0, err
	}
}

type Builder func(openrtb_ext.BidderName, config.Adapter, config.Server) (Bidder, error)
