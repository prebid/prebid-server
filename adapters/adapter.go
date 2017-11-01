package adapters

import (
	"context"
	"crypto/tls"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
	"github.com/prebid/prebid-server/ssl"
	"net/http"
	"time"
)

type BidderLeaf interface {
	// MakeHttpRequests makes the HTTP requests which should be made to fetch bids.
	//
	// The requests can be nil (if no external calls are needed), but should not contain nil elements.
	//
	// The errors should contain a list of errors which explain why this bidder's bids will be
	// "subpar" in some way. For example: the request contained ad types which this bidder doesn't support.
	MakeHttpRequests(request *openrtb.BidRequest) ([]*RequestData, []error)

	// MakeBids unpacks the server's response into Bids.
	// This method **should not** close the response body. The caller will fully read and close it so that the
	// connections get reused properly.
	//
	// The bids can be nil (for no bids), but should not contain nil elements.
	//
	// The errors should contain a list of errors which explain why this bidder's bids will be
	// "subpar" in some way. For example: the server response didn't have the expected format.
	MakeBids(request *openrtb.BidRequest, response *ResponseData) ([]*BidData, []error)
}

// RequestData packages together the fields needed to make an http.Request.
//
// This exists so that prebid-server core code can implement its "debug" API uniformly across all adapters.
type RequestData struct {
	Method  string
	Uri     string
	Body    []byte
	Headers http.Header
}

// ResponseData packages together information from the server's http.Response.
//
// This exists so that prebid-server core code can implement its "debug" API uniformly across all adapters.
type ResponseData struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

type BidData struct {
	Bid  *openrtb.Bid
	Type openrtb_ext.BidType
}

// Bidders participate in prebid-server auctions.
type Bidder interface {
	// Bid should return the SeatBid containing all bids used by this bidder.
	//
	// All `Ext` fields from the argument request are generated from contract classes in openrtb_ext.
	// Each bidder may define their own `Ext` format there.
	//
	// All `Ext` fields inside the returned SeatBid must also be generated from the contract classes in openrtb_ext.
	//
	// Bid should still attempt to return a SeatBid, even if some errors occurred. If there are no bids, return nil.
	// Errors will be processed by prebid-server core code and logged or reported to the user as appropriate.
	Bid(ctx context.Context, request *openrtb.BidRequest) *BidderResponse
}

// BidderResponse carries all the data needed for a Bidder's response.
type BidderResponse struct {
	// Bids contains all the bids that the Bidder wants to enter.
	// This can be nil (for no bids), but should not contain nil elements.
	Bids []*openrtb.Bid
	// ServerCalls stores some debugging info.
	// This is only required if the input request.Test was 1.
	ServerCalls []*openrtb_ext.ExtServerCall
	// Errors should contain a list of errors which occurred internally. These should report
	// any conditions which result in "no" or "subpar" bids. For example:
	//
	// 1. The openrtb request needs an ad type which this bidder doesn't support.
	// 2. The auction timed out before all the bids were entered.
	// 3. The remote server returned unexpected input.
	Errors []error
}

// Adapter is a deprecated interface which connects prebid-server to a demand partner.
// PBS is currently being rewritten to use Bidder, and this will be removed after.
// Their primary purpose is to produce bids in response to Auction requests.
//
// For the future, see Bidder.
type Adapter interface {
	// Name uniquely identifies this adapter. This must be identical to the code in Prebid.js,
	// but cannot overlap with any other adapters in prebid-server.
	Name() string
	// FamilyName identifies the space of cookies which this adapter accesses. For example, an adapter
	// using the adnxs.com cookie space should return "adnxs".
	FamilyName() string
	// Determines whether this adapter should get callouts if there is not a synched user ID
	SkipNoCookies() bool
	// GetUsersyncInfo returns the parameters which are needed to do sync users with this bidder.
	// For more information, see http://clearcode.cc/2015/12/cookie-syncing/
	GetUsersyncInfo() *pbs.UsersyncInfo
	// Call produces bids which should be considered, given the auction params.
	//
	// In practice, implementations almost always make one call to an external server here.
	// However, that is not a requirement for satisfying this interface.
	Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error)
}

// HTTPAdapterConfig groups options which control how HTTP requests are made by adapters.
type HTTPAdapterConfig struct {
	// See IdleConnTimeout on https://golang.org/pkg/net/http/#Transport
	IdleConnTimeout time.Duration
	// See MaxIdleConns on https://golang.org/pkg/net/http/#Transport
	MaxConns int
	// See MaxIdleConnsPerHost on https://golang.org/pkg/net/http/#Transport
	MaxConnsPerHost int
}

type HTTPAdapter struct {
	Transport *http.Transport
	Client    *http.Client
}

// DefaultHTTPAdapterConfig is an HTTPAdapterConfig that chooses sensible default values.
var DefaultHTTPAdapterConfig = &HTTPAdapterConfig{
	MaxConns:        50,
	MaxConnsPerHost: 10,
	IdleConnTimeout: 60 * time.Second,
}

// NewHTTPAdapter creates an HTTPAdapter which obeys the rules given by the config, and
// has all the available SSL certs available in the project.
func NewHTTPAdapter(c *HTTPAdapterConfig) *HTTPAdapter {
	ts := &http.Transport{
		MaxIdleConns:        c.MaxConns,
		MaxIdleConnsPerHost: c.MaxConnsPerHost,
		IdleConnTimeout:     c.IdleConnTimeout,
		TLSClientConfig:     &tls.Config{RootCAs: ssl.GetRootCAPool()},
	}

	return &HTTPAdapter{
		Transport: ts,
		Client: &http.Client{
			Transport: ts,
		},
	}
}

// used for callOne (possibly pull all of the shared code here)
type CallOneResult struct {
	StatusCode   int
	ResponseBody string
	Bid          *pbs.PBSBid
	Error        error
}
