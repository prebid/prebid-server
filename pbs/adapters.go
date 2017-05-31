package pbs

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/ssl"
)

type Adapter interface {
	Configure(string, *config.Adapter) // Configure is required
	Name() string                      // Name should be unique
	FamilyName() string                // FamilyName is used for cookies
	GetUsersyncInfo() *UsersyncInfo
	Call(ctx context.Context, req *PBSRequest, bidder *PBSBidder) (PBSBidSlice, error)
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
