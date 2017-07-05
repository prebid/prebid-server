package adapters

import (
	"context"
	"crypto/tls"
	"github.com/prebid/prebid-server/pbs"
	"github.com/prebid/prebid-server/ssl"
	"net/http"
	"time"
)

type Adapter interface {
	Name() string
	FamilyName() string
	SkipNoCookies() bool
	GetUsersyncInfo() *pbs.UsersyncInfo
	Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error)
}

type HTTPAdapterConfig struct {
	IdleConnTimeout     time.Duration
	MaxConns            int
	MaxIdleConnsPerHost int
	MaxConnsPerHost     int
}

type HTTPAdapter struct {
	Transport *http.Transport
	Client    *http.Client
}

var DefaultHTTPAdapterConfig = &HTTPAdapterConfig{
	MaxConns:            50,
	MaxConnsPerHost:     10,
	MaxIdleConnsPerHost: 3,
	IdleConnTimeout:     60 * time.Second,
}

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
