package skanidlist

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/prebid/prebid-server/cache/skanidlist/cfg"
	"github.com/prebid/prebid-server/cache/skanidlist/model"
	"github.com/prebid/prebid-server/openrtb_ext"
	"golang.org/x/net/context/ctxhttp"
)

type cache struct {
	url        string
	successTTL time.Duration
	missTTL    time.Duration
	bidder     openrtb_ext.BidderName

	ids map[string]bool

	mu              *sync.RWMutex
	expiration      int64
	updateRequested bool
}

func newCache(cfg cfg.Cache) *cache {
	c := cache{
		url:        cfg.Url,
		successTTL: time.Duration(1 * time.Hour),
		missTTL:    time.Duration(5 * time.Minute),
		bidder:     cfg.Bidder,

		ids: map[string]bool{},

		mu:              new(sync.RWMutex),
		expiration:      time.Now().UnixNano(),
		updateRequested: false,
	}

	if cfg.BidderSKANID != "" {
		c.ids = map[string]bool{
			cfg.BidderSKANID: true,
		}
	}

	return &c
}

func (c *cache) get() map[string]bool {
	if len(c.url) < 1 {
		// bidder does not support multiple SKAN ID List
		return map[string]bool{}
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.ids
}

func (c *cache) update(ctx context.Context, httpClient *http.Client) {
	chResp := make(chan *response, 1)

	c.mu.Lock()
	if time.Now().UnixNano() > c.expiration && !c.updateRequested {
		c.updateRequested = true

		c.mu.Unlock()

		// get newrelic transaction from context
		txn := newrelic.FromContext(ctx)

		fetchFromServer := func(ctx context.Context, txn *newrelic.Transaction) {
			ctx = newrelic.NewContext(ctx, txn)
			chResp <- c.fetchFromServer(ctx, httpClient)
		}
		go fetchFromServer(ctx, txn.NewGoroutine())

	} else {
		c.mu.Unlock()

		chResp <- &response{
			skanIDList: model.SKANIDList{},
			updated:    false,
			err:        nil,
		}
	}

	resp := <-chResp

	if resp.err != nil {
		// http call returned error, list could not fetched, report NR error
		if txn := newrelic.FromContext(ctx); txn != nil {
			txn.NoticeError(resp.err)
		}

		// update expiration and updateRequested fields to check back in miss TTL time
		c.updateMiss()

		return
	}

	if !resp.updated {
		// update is not required so do not change anything
		return
	}

	c.updateSuccess(resp.skanIDList)
}

func (c *cache) fetchFromServer(ctx context.Context, httpClient *http.Client) *response {
	req, err := http.NewRequest("GET", c.url, nil)
	if err != nil {
		return &response{
			skanIDList: model.SKANIDList{},
			updated:    false,
			err:        errors.New(fmt.Sprintf("error making request for bidder's servers for: %s - %v", c.url, err)),
		}
	}

	req.Header.Set("Accept", "application/json")

	resp, err := ctxhttp.Do(ctx, httpClient, req)
	if err != nil {
		return &response{
			skanIDList: model.SKANIDList{},
			updated:    false,
			err:        errors.New(fmt.Sprintf("error fetching skanidlist from bidder's servers for: %s - %v", c.url, err)),
		}
	}

	if resp.StatusCode != http.StatusOK {
		return &response{
			skanIDList: model.SKANIDList{},
			updated:    false,
			err:        errors.New(fmt.Sprintf("error statuscode (%d) received from bidder's servers for: %s - %v", resp.StatusCode, c.url, err)),
		}
	}

	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return &response{
			skanIDList: model.SKANIDList{},
			updated:    true,
			err:        errors.New(fmt.Sprintf("error reading skanidlist response body for: %s - %v", c.url, err)),
		}
	}

	var skanIDList model.SKANIDList
	err = json.Unmarshal(data, &skanIDList)
	if err != nil {
		return &response{
			skanIDList: model.SKANIDList{},
			updated:    true,
			err:        errors.New(fmt.Sprintf("error unmarshaling response to skanidlist for: %s - %v", c.url, err)),
		}
	}

	return &response{
		skanIDList: skanIDList,
		updated:    true,
		err:        nil,
	}
}

func (c *cache) updateSuccess(skanIDList model.SKANIDList) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ids = extract(skanIDList)
	c.expiration = time.Now().Add(c.successTTL).UnixNano()
	c.updateRequested = false
}

func (c *cache) updateMiss() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.expiration = time.Now().Add(c.missTTL).UnixNano()
	c.updateRequested = false
}
