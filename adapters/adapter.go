package adapters

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/prebid/prebid-server/ssl"
)

type Configuration struct {
	Name        string // required
	Endpoint    string
	Username    string
	Password    string
	UserSyncURL string
}

type HTTPAdapterConfig struct {
	IdleConnTimeout     time.Duration
	MaxConns            int
	MaxIdleConnsPerHost int
	MaxConnsPerHost     int
}

// HTTPAdapter contains an http.Transport and reusable http.Client
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

// NewHTTPAdapter takes a HTTPAdapterConfig and returns a pointer to a HTTPAdapter
func NewHTTPAdapter(c *HTTPAdapterConfig) *HTTPAdapter {
	return &HTTPAdapter{
		Transport: defaultTransport(c),
		Client: &http.Client{
			Transport: defaultTransport(c),
		},
	}
}

// defaultTransport will take a HTTPAdapterConfig and return *http.Transport
func defaultTransport(c *HTTPAdapterConfig) *http.Transport {
	return &http.Transport{
		MaxIdleConns:        c.MaxConns,
		MaxIdleConnsPerHost: c.MaxConnsPerHost,
		IdleConnTimeout:     c.IdleConnTimeout,
		TLSClientConfig:     &tls.Config{RootCAs: ssl.GetRootCAPool()},
	}
}
