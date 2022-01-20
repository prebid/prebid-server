package gdpr

import (
	"github.com/prebid/go-gdpr/consentconstants"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// Need to figure out how to create the VendorExceptionMap for the account config
//	- can we cache it?
//  - should we just create it here for now? would rather not
//

// OPTIONS:
// 1) construct final config on instantiation
//	- could be done by copying hostConfig and overwriting applicable fields
//  - could be done by creating a new object
// 2) compute needed config every time we try to access config info
// 3) lazily compute needed config and memoize results
type TCF2ConfigReader interface { //TODO: change to TCF2Config
	BasicEnforcementVendor(openrtb_ext.BidderName) bool
	Enabled() bool
	FeatureOneEnforced() bool
	FeatureOneVendorException(openrtb_ext.BidderName) bool
	IntegrationEnabled(config.IntegrationType) bool
	PurposeEnforced(consentconstants.Purpose) bool
	PurposeEnforcingVendors(consentconstants.Purpose) bool
	PurposeVendorException(consentconstants.Purpose, openrtb_ext.BidderName) bool
	PurposeOneTreatmentEnabled() bool
	PurposeOneTreatmentAccessAllowed() bool
}

type TCF2Config struct { //TODO: rename so no conflict - doesn't need to be exported. maybe just config or tcf2Config?
	HostConfig    config.TCF2
	AccountConfig config.AccountGDPR
}

func NewTCF2ConfigReader(hostConfig config.TCF2, accountConfig config.AccountGDPR) TCF2ConfigReader {
	return &TCF2Config{
		HostConfig:    hostConfig,
		AccountConfig: accountConfig,
	}
}

func (tc *TCF2Config) Enabled() bool {
	return tc.HostConfig.Enabled
}

func (tc *TCF2Config) IntegrationEnabled(integrationType config.IntegrationType) bool {
	if accountEnabled := tc.AccountConfig.EnabledForIntegrationType(integrationType); accountEnabled != nil {
		return *accountEnabled
	}
	return tc.HostConfig.Enabled
}

// PurposeEnforced checks if full enforcement is turned on for a given purpose. With full enforcement enabled, the
// GDPR full enforcement algorithm will execute for that purpose determining legal basis; otherwise it's skipped.
func (tc *TCF2Config) PurposeEnforced(purpose consentconstants.Purpose) bool {
	if value, exists := tc.AccountConfig.PurposeEnforced(purpose); exists {
		return value
	}

	value := tc.HostConfig.PurposeEnforced(purpose)
	return value
}

// PurposeEnforcingVendors checks if enforcing vendors is turned on for a given purpose. With enforcing vendors
// enabled, the GDPR full enforcement algorithm considers the GVL when determining legal basis; otherwise it's skipped.
func (tc *TCF2Config) PurposeEnforcingVendors(purpose consentconstants.Purpose) bool {
	if value, exists := tc.AccountConfig.PurposeEnforcingVendors(purpose); exists {
		return value
	}

	value := tc.HostConfig.PurposeEnforcingVendors(purpose)
	return value
}

// PurposeVendorException checks if the specified bidder is considered a vendor exception for a given purpose. If a bidder is a
// vendor exception, the GDPR full enforcement algorithm will bypass the legal basis calculation assuming the request is valid
// and there isn't "deny all" publisher restriction
func (tc *TCF2Config) PurposeVendorException(purpose consentconstants.Purpose, bidder openrtb_ext.BidderName) bool {
	if value, exists := tc.AccountConfig.PurposeVendorException(purpose, bidder); exists {
		return value
	}
	value := tc.HostConfig.PurposeVendorException(purpose, bidder)
	return value
}

// FeatureOneEnforced checks if special feature one is enforced. If it is enforced, geo may be used to determine...TODO
func (tc *TCF2Config) FeatureOneEnforced() bool {
	if value, exists := tc.AccountConfig.FeatureOneEnforced(); exists {
		return value
	}
	value := tc.HostConfig.FeatureOneEnforced()
	return value
}

// FeatureOneVendorException checks if the specified bidder is considered a vendor exception for special feature one. If a bider
// is a vendor exception...TODO
func (tc *TCF2Config) FeatureOneVendorException(bidder openrtb_ext.BidderName) bool {
	if value, exists := tc.AccountConfig.FeatureOneVendorException(bidder); exists {
		return value
	}
	value := tc.HostConfig.FeatureOneVendorException(bidder)
	return value
}

// PurposeOneTreatmentEnabled...TODO
func (tc *TCF2Config) PurposeOneTreatmentEnabled() bool {
	if value, exists := tc.AccountConfig.PurposeOneTreatmentEnabled(); exists {
		return value
	}
	value := tc.HostConfig.PurposeOneTreatmentEnabled()
	return value
}

// PurposeOneTreatmentAccessAllowed...TODO
func (tc *TCF2Config) PurposeOneTreatmentAccessAllowed() bool {
	if value, exists := tc.AccountConfig.PurposeOneTreatmentAccessAllowed(); exists {
		return value
	}
	value := tc.HostConfig.PurposeOneTreatmentAccessAllowed()
	return value
}

// BasicEnforcementVendor...TODO...weakVendorEnforcement
func (tc *TCF2Config) BasicEnforcementVendor(bidder openrtb_ext.BidderName) bool {
	if value, exists := tc.AccountConfig.BasicEnforcementVendor(bidder); exists {
		return value
	}
	return false
}
