package adapters

import (
	"context"
	"crypto/tls"
	"github.com/prebid/prebid-server/pbs"
	"github.com/prebid/prebid-server/ssl"
	"net/http"
	"time"
)

// Adapters connect prebid-server to a demand partner. Their primary purpose is to produce bids
// in response to Auction requests.
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
	// Produce bids which should be considered, given the auction params.
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
type callOneResult struct {
	statusCode   int
	responseBody string
	bid          *pbs.PBSBid
	Error        error
}
