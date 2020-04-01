package gdpr

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/glog"
	"github.com/prebid/go-gdpr/api"
	"github.com/prebid/go-gdpr/vendorlist"
	"github.com/prebid/go-gdpr/vendorlist2"
	"github.com/prebid/prebid-server/config"
	"golang.org/x/net/context/ctxhttp"
)

type saveVendors func(uint16, api.VendorList)

// This file provides the vendorlist-fetching function for Prebid Server.
//
// For more info, see https://github.com/prebid/prebid-server/issues/504
//
// Nothing in this file is exported. Public APIs can be found in gdpr.go

func newVendorListFetcher(initCtx context.Context, cfg config.GDPR, client *http.Client, urlMaker func(uint16, uint8) string, TCFVer uint8) func(ctx context.Context, id uint16) (vendorlist.VendorList, error) {
	// These save and load functions can be used to store & retrieve lists from our cache.
	save, load := newVendorListCache()

	withTimeout, cancel := context.WithTimeout(initCtx, cfg.Timeouts.InitTimeout())
	defer cancel()
	populateCache(withTimeout, client, urlMaker, save, TCFVer)

	saveOneSometimes := newOccasionalSaver(cfg.Timeouts.ActiveTimeout(), TCFVer)

	return func(ctx context.Context, id uint16) (vendorlist.VendorList, error) {
		list := load(id)
		if list != nil {
			return list, nil
		}
		saveOneSometimes(ctx, client, urlMaker(id, TCFVer), save)
		list = load(id)
		if list != nil {
			return list, nil
		}
		return nil, fmt.Errorf("gdpr vendor list version %d does not exist, or has not been loaded yet. Try again in a few minutes", id)
	}
}

// populateCache saves all the known versions of the vendor list for future use.
func populateCache(ctx context.Context, client *http.Client, urlMaker func(uint16, uint8) string, saver saveVendors, TCFVer uint8) {
	latestVersion := saveOne(ctx, client, urlMaker(0, TCFVer), saver, TCFVer)

	for i := uint16(1); i < latestVersion; i++ {
		saveOne(ctx, client, urlMaker(i, TCFVer), saver, TCFVer)
	}
}

// Make a URL which can be used to fetch a given version of the Global Vendor List. If the version is 0,
// this will fetch the latest version.
func vendorListURLMaker(version uint16, TCFVer uint8) string {
	if TCFVer == 2 {
		if version == 0 {
			return "https://vendorlist.consensu.org/v2/vendor-list.json"
		}
		return "https://vendorlist.consensu.org/v2/archives/vendor-list-v" + strconv.Itoa(int(version)) + ".json"
	}
	if version == 0 {
		return "https://vendorlist.consensu.org/vendorlist.json"
	}
	return "https://vendorlist.consensu.org/v-" + strconv.Itoa(int(version)) + "/vendorlist.json"
}

// newOccasionalSaver returns a wrapped version of saveOne() which only activates every few minutes.
//
// The goal here is to update quickly when new versions of the VendorList are released, but not wreck
// server performance if a bad CMP starts sending us malformed consent strings that advertize a version
// that doesn't exist yet.
func newOccasionalSaver(timeout time.Duration, TCFVer uint8) func(ctx context.Context, client *http.Client, url string, saver saveVendors) {
	lastSaved := &atomic.Value{}
	lastSaved.Store(time.Time{})

	return func(ctx context.Context, client *http.Client, url string, saver saveVendors) {
		now := time.Now()
		if now.Sub(lastSaved.Load().(time.Time)).Minutes() > 10 {
			withTimeout, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			saveOne(withTimeout, client, url, saver, TCFVer)
			lastSaved.Store(now)
		}
	}
}

func saveOne(ctx context.Context, client *http.Client, url string, saver saveVendors, cTFVer uint8) uint16 {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		glog.Errorf("Failed to build GET %s request. Cookie syncs may be affected: %v", url, err)
		return 0
	}

	resp, err := ctxhttp.Do(ctx, client, req)
	if err != nil {
		glog.Errorf("Error calling GET %s. Cookie syncs may be affected: %v", url, err)
		return 0
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		glog.Errorf("Error reading response body from GET %s. Cookie syncs may be affected: %v", url, err)
		return 0
	}
	if resp.StatusCode != http.StatusOK {
		glog.Errorf("GET %s returned %d. Cookie syncs may be affected.", url, resp.StatusCode)
		return 0
	}
	var newList api.VendorList
	if cTFVer == 2 {
		newList, err = vendorlist2.ParseEagerly(respBody)
	} else {
		newList, err = vendorlist.ParseEagerly(respBody)
	}
	if err != nil {
		glog.Errorf("GET %s returned malformed JSON. Cookie syncs may be affected. Error was %v. Body was %s", url, err, string(respBody))
		return 0
	}

	saver(newList.Version(), newList)
	return newList.Version()
}

func newVendorListCache() (save func(id uint16, list api.VendorList), load func(id uint16) api.VendorList) {
	cache := &sync.Map{}

	save = func(id uint16, list api.VendorList) {
		cache.Store(id, list)
	}
	load = func(id uint16) api.VendorList {
		list, ok := cache.Load(id)
		if ok {
			return list.(vendorlist.VendorList)
		}
		return nil
	}
	return
}
