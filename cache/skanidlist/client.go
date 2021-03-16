package skanidlist

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/prebid/prebid-server/cache/skanidlist/cfg"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type client struct {
	caches map[openrtb_ext.BidderName]*cache
	mu     *sync.Mutex
}

// Empty skanIDListClient
var skanIDListClient client = client{
	caches: map[openrtb_ext.BidderName]*cache{},
	mu:     new(sync.Mutex),
}

func (c client) makeCache(cfg cfg.Cache) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.caches[cfg.Bidder] == nil {
		c.caches[cfg.Bidder] = newCache(cfg)
	}
}

func cacheClient(bidder openrtb_ext.BidderName) (*cache, error) {
	// Initialize bidder caches at first call
	switch bidder {
	case openrtb_ext.BidderTaurusX:
		if skanIDListClient.caches[bidder] == nil {
			skanIDListClient.makeCache(cfg.TaurusX)
		}
		return skanIDListClient.caches[bidder], nil

	case openrtb_ext.BidderPubmatic:
		if skanIDListClient.caches[bidder] == nil {
			skanIDListClient.makeCache(cfg.Pubmatic)
		}
		return skanIDListClient.caches[bidder], nil
	}

	return nil, errors.New(fmt.Sprintf("bidder (%s) does not support SKAN ID List", bidder))
}

func Update(ctx context.Context, httpClient *http.Client, bidder openrtb_ext.BidderName) {
	if c, err := cacheClient(bidder); err == nil {
		c.update(ctx, httpClient)
	}
}

func Get(bidder openrtb_ext.BidderName) map[string]bool {
	if c, err := cacheClient(bidder); err == nil {
		return c.get()
	}

	return map[string]bool{}
}
