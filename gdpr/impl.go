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
	"github.com/prebid/go-gdpr/consentconstants"
	"github.com/prebid/go-gdpr/vendorconsent"
	"github.com/prebid/go-gdpr/vendorlist"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"golang.org/x/net/context/ctxhttp"
)

type permissionsImpl struct {
	cfg             config.GDPR
	vendorIDs       map[openrtb_ext.BidderName]uint16
	fetchVendorList func(ctx context.Context, id uint16) (vendorlist.VendorList, error)
}

func (p *permissionsImpl) HostCookiesAllowed(ctx context.Context, consent string) (bool, error) {
	// If we're not given a consent string, respect the preferences in the app config.
	if consent == "" {
		return p.cfg.UsersyncIfAmbiguous, nil
	}

	parsedConsent, err := vendorconsent.Parse([]byte(consent))
	if err != nil {
		return false, err
	}

	vendorList, err := p.fetchVendorList(ctx, parsedConsent.VendorListVersion())
	if err != nil {
		return false, err
	}

	// Config validation makes uint16 conversion safe here
	return hasPermissions(parsedConsent, vendorList, uint16(p.cfg.HostVendorID), consentconstants.InfoStorageAccess), nil
}

func hasPermissions(consent vendorconsent.VendorConsents, vendorList vendorlist.VendorList, vendorID uint16, purpose consentconstants.Purpose) bool {
	vendor := vendorList.Vendor(vendorID)
	if vendor == nil {
		return false
	}
	if vendor.LegitimateInterest(purpose) {
		return true
	}

	// If the host declared writing cookies to be a "normal" purpose, only do the sync if the user consented to it.
	if vendor.Purpose(purpose) && consent.VendorConsent(vendorID) {
		return true
	}

	return false
}

func (p *permissionsImpl) BidderSyncAllowed(ctx context.Context, bidder openrtb_ext.BidderName, consent string) (bool, error) {
	// If we're not given a consent string, respect the preferences in the app config.
	if consent == "" {
		return p.cfg.UsersyncIfAmbiguous, nil
	}

	// If the bidder isn't part of the GDPR global vendor list yet, defer to the publisher's preferences.
	id, ok := p.vendorIDs[bidder]
	if !ok {
		return p.cfg.UsersyncIfAmbiguous, nil
	}

	parsedConsent, err := vendorconsent.Parse([]byte(consent))
	if err != nil {
		return false, err
	}

	vendorList, err := p.fetchVendorList(ctx, parsedConsent.VendorListVersion())
	if err != nil {
		return false, err
	}

	return hasPermissions(parsedConsent, vendorList, id, consentconstants.AdSelectionDeliveryReporting), nil
}

func newVendorListFetcher(initContext context.Context, client *http.Client) func(ctx context.Context, id uint16) (vendorlist.VendorList, error) {
	// These save and load functions can be used to store & retrieve lists from our cache.
	save, load := newVendorListCache()
	populateCache(initContext, client, save)

	saveOneSometimes := newOccasionalSaver()

	return func(ctx context.Context, id uint16) (vendorlist.VendorList, error) {
		list := load(id)
		if list != nil {
			return list, nil
		}
		saveOneSometimes(ctx, client, "https://vendorlist.consensu.org/v-"+strconv.Itoa(int(id))+"/vendorlist.json", save)
		list = load(id)
		if list != nil {
			return list, nil
		}
		return nil, fmt.Errorf("gdpr vendor list version %d does not exist, or has not been loaded yet. Try again in a few minutes", id)
	}
}

// populateCache saves all the known versions of the vendor list for future use.
func populateCache(ctx context.Context, client *http.Client, saver func(id uint16, list vendorlist.VendorList)) {
	latestVersion := saveOne(ctx, client, "https://vendorlist.consensu.org/vendorlist.json", saver)

	for i := 1; i < latestVersion; i++ {
		saveOne(ctx, client, "https://vendorlist.consensu.org/v-"+strconv.Itoa(i)+"/vendorlist.json", saver)
	}
}

// newOccasionalSaver returns a wrapped version of saveOne() which only activates every few minutes.
//
// The goal here is to update quickly when new versions of the VendorList are released, but not wreck
// server performance if a bad CMP starts sending us malformed consent strings that advertize a version
// that doesn't exist yet.
func newOccasionalSaver() func(ctx context.Context, client *http.Client, url string, saver func(id uint16, list vendorlist.VendorList)) {
	lastSaved := &atomic.Value{}
	lastSaved.Store(time.Now())

	return func(ctx context.Context, client *http.Client, url string, saver func(id uint16, list vendorlist.VendorList)) {
		now := time.Now()
		if now.Sub(lastSaved.Load().(time.Time)).Minutes() > 10 {
			saveOne(ctx, client, url, saver)
			lastSaved.Store(now)
		}
	}
}

func saveOne(ctx context.Context, client *http.Client, url string, saver func(id uint16, list vendorlist.VendorList)) int {
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

	newList, err := vendorlist.ParseEagerly(respBody)
	if err != nil {
		glog.Errorf("GET %s returned malformed JSON. Cookie syncs may be affected. Error was %v. Body was %s", url, err, string(respBody))
		return 0
	}

	saver(newList.Version(), newList)
	return int(newList.Version())
}

func newVendorListCache() (save func(id uint16, list vendorlist.VendorList), load func(id uint16) vendorlist.VendorList) {
	cache := &sync.Map{}

	save = func(id uint16, list vendorlist.VendorList) {
		cache.Store(id, list)
	}
	load = func(id uint16) vendorlist.VendorList {
		list, ok := cache.Load(id)
		if ok {
			return list.(vendorlist.VendorList)
		}
		return nil
	}
	return
}
