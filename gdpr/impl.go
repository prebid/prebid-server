package gdpr

import (
	"context"
	"encoding/base64"

	"github.com/prebid/go-gdpr/consentconstants"
	"github.com/prebid/go-gdpr/vendorconsent"
	"github.com/prebid/go-gdpr/vendorlist"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// This file implements GDPR permissions for the app.
// For more info, see https://github.com/prebid/prebid-server/issues/501
//
// Nothing in this file is exported. Public APIs can be found in gdpr.go

type permissionsImpl struct {
	cfg             config.GDPR
	vendorIDs       map[openrtb_ext.BidderName]uint16
	fetchVendorList func(ctx context.Context, id uint16) (vendorlist.VendorList, error)
}

func (p *permissionsImpl) HostCookiesAllowed(ctx context.Context, consent string) (bool, error) {
	return p.allowSync(ctx, uint16(p.cfg.HostVendorID), consent)
}

func (p *permissionsImpl) BidderSyncAllowed(ctx context.Context, bidder openrtb_ext.BidderName, consent string) (bool, error) {
	id, ok := p.vendorIDs[bidder]
	if ok {
		return p.allowSync(ctx, id, consent)
	}

	if consent == "" {
		return p.cfg.UsersyncIfAmbiguous, nil
	}

	return false, nil
}

func (p *permissionsImpl) allowSync(ctx context.Context, vendorID uint16, consent string) (bool, error) {
	// If we're not given a consent string, respect the preferences in the app config.
	if consent == "" {
		return p.cfg.UsersyncIfAmbiguous, nil
	}

	data, err := base64.RawURLEncoding.DecodeString(consent)
	if err != nil {
		return false, &ErrorMalformedConsent{
			consent: consent,
			cause:   err,
		}
	}

	parsedConsent, err := vendorconsent.Parse([]byte(data))
	if err != nil {
		return false, &ErrorMalformedConsent{
			consent: consent,
			cause:   err,
		}
	}

	vendorList, err := p.fetchVendorList(ctx, parsedConsent.VendorListVersion())
	if err != nil {
		return false, err
	}

	vendor := vendorList.Vendor(vendorID)
	if vendor == nil {
		return false, nil
	}

	if vendor.Purpose(consentconstants.InfoStorageAccess) && parsedConsent.PurposeAllowed(consentconstants.InfoStorageAccess) && parsedConsent.VendorConsent(vendorID) {
		return true, nil
	}

	return false, nil
}

type alwaysAllow struct{}

func (a alwaysAllow) HostCookiesAllowed(ctx context.Context, consent string) (bool, error) {
	return true, nil
}

func (a alwaysAllow) BidderSyncAllowed(ctx context.Context, bidder openrtb_ext.BidderName, consent string) (bool, error) {
	return true, nil
}
