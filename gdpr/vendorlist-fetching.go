package gdpr

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/glog"
	"github.com/prebid/go-gdpr/api"
	"github.com/prebid/go-gdpr/vendorlist"
	"github.com/prebid/go-gdpr/vendorlist2"
	"github.com/prebid/prebid-server/v3/config"
	"golang.org/x/net/context/ctxhttp"
)

type saveVendors func(uint16, uint16, api.VendorList)
type VendorListFetcher func(ctx context.Context, specVersion uint16, listVersion uint16) (vendorlist.VendorList, error)

// This file provides the vendorlist-fetching function for Prebid Server.
//
// For more info, see https://github.com/prebid/prebid-server/issues/504
//
// Nothing in this file is exported. Public APIs can be found in gdpr.go

func NewVendorListFetcher(initCtx context.Context, cfg config.GDPR, client *http.Client, urlMaker func(uint16, uint16) string) VendorListFetcher {
	cacheSave, cacheLoad := newVendorListCache()

	preloadContext, cancel := context.WithTimeout(initCtx, cfg.Timeouts.InitTimeout())
	defer cancel()
	preloadCache(preloadContext, client, urlMaker, cacheSave)

	saveOneRateLimited := newOccasionalSaver(cfg.Timeouts.ActiveTimeout())
	return func(ctx context.Context, specVersion, listVersion uint16) (vendorlist.VendorList, error) {
		// Attempt To Load From Cache
		if list := cacheLoad(specVersion, listVersion); list != nil {
			return list, nil
		}

		// Attempt To Download
		// - May not add to cache immediately.
		saveOneRateLimited(ctx, client, urlMaker(specVersion, listVersion), cacheSave)

		// Attempt To Load From Cache Again
		// - May have been added by the call to saveOneRateLimited.
		if list := cacheLoad(specVersion, listVersion); list != nil {
			return list, nil
		}

		// Give Up
		return nil, makeVendorListNotFoundError(specVersion, listVersion)
	}
}

func makeVendorListNotFoundError(specVersion, listVersion uint16) error {
	return fmt.Errorf("gdpr vendor list spec version %d list version %d does not exist, or has not been loaded yet. Try again in a few minutes", specVersion, listVersion)
}

// preloadCache saves all the known versions of the vendor list for future use.
func preloadCache(ctx context.Context, client *http.Client, urlMaker func(uint16, uint16) string, saver saveVendors) {
	versions := [2]struct {
		specVersion      uint16
		firstListVersion uint16
	}{
		{
			specVersion:      2,
			firstListVersion: 2, // The GVL for TCF2 has no vendors defined in its first version. It's very unlikely to be used, so don't preload it.
		},
		{
			specVersion:      3,
			firstListVersion: 1,
		},
	}
	for _, v := range versions {
		latestVersion := saveOne(ctx, client, urlMaker(v.specVersion, 0), saver)

		for i := v.firstListVersion; i < latestVersion; i++ {
			saveOne(ctx, client, urlMaker(v.specVersion, i), saver)
		}
	}
}

// Make a URL which can be used to fetch a given version of the Global Vendor List. If the version is 0,
// this will fetch the latest version.
func VendorListURLMaker(specVersion, listVersion uint16) string {
	if listVersion == 0 {
		return "https://vendor-list.consensu.org/v" + strconv.Itoa(int(specVersion)) + "/vendor-list.json"
	}
	return "https://vendor-list.consensu.org/v" + strconv.Itoa(int(specVersion)) + "/archives/vendor-list-v" + strconv.Itoa(int(listVersion)) + ".json"
}

// newOccasionalSaver returns a wrapped version of saveOne() which only activates every few minutes.
//
// The goal here is to update quickly when new versions of the VendorList are released, but not wreck
// server performance if a bad CMP starts sending us malformed consent strings that advertize a version
// that doesn't exist yet.
func newOccasionalSaver(timeout time.Duration) func(ctx context.Context, client *http.Client, url string, saver saveVendors) {
	lastSaved := &atomic.Value{}
	lastSaved.Store(time.Time{})

	return func(ctx context.Context, client *http.Client, url string, saver saveVendors) {
		now := time.Now()
		timeSinceLastSave := now.Sub(lastSaved.Load().(time.Time))

		if timeSinceLastSave.Minutes() > 10 {
			withTimeout, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			saveOne(withTimeout, client, url, saver)
			lastSaved.Store(now)
		}
	}
}

func saveOne(ctx context.Context, client *http.Client, url string, saver saveVendors) uint16 {
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

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		glog.Errorf("Error reading response body from GET %s. Cookie syncs may be affected: %v", url, err)
		return 0
	}
	if resp.StatusCode != http.StatusOK {
		glog.Errorf("GET %s returned %d. Cookie syncs may be affected.", url, resp.StatusCode)
		return 0
	}
	var newList api.VendorList
	newList, err = vendorlist2.ParseEagerly(respBody)
	if err != nil {
		glog.Errorf("GET %s returned malformed JSON. Cookie syncs may be affected. Error was %v. Body was %s", url, err, string(respBody))
		return 0
	}

	saver(newList.SpecVersion(), newList.Version(), newList)
	return newList.Version()
}

func newVendorListCache() (save func(specVersion, listVersion uint16, list api.VendorList), load func(specVersion, listVersion uint16) api.VendorList) {
	cache := &sync.Map{}

	save = func(specVersion uint16, listVersion uint16, list api.VendorList) {
		key := fmt.Sprint(specVersion) + "-" + fmt.Sprint(listVersion)
		cache.Store(key, list)
	}

	load = func(specVersion, listVersion uint16) api.VendorList {
		key := fmt.Sprint(specVersion) + "-" + fmt.Sprint(listVersion)
		list, ok := cache.Load(key)
		if ok {
			return list.(vendorlist.VendorList)
		}
		return nil
	}
	return
}
