package gdpr

import (
	"github.com/prebid/go-gdpr/consentconstants"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// TCF2ConfigReader is an interface to access TCF2 configurations
type TCF2ConfigReader interface {
	BasicEnforcementVendor(openrtb_ext.BidderName) bool
	FeatureOneEnforced() bool
	FeatureOneVendorException(openrtb_ext.BidderName) bool
	IntegrationEnabled(config.IntegrationType) bool
	IsEnabled() bool
	PurposeEnforced(consentconstants.Purpose) bool
	PurposeEnforcingVendors(consentconstants.Purpose) bool
	PurposeVendorException(consentconstants.Purpose, openrtb_ext.BidderName) bool
	PurposeOneTreatmentEnabled() bool
	PurposeOneTreatmentAccessAllowed() bool
}

type TCF2ConfigBuilder func(hostConfig config.TCF2, accountConfig config.AccountGDPR) TCF2ConfigReader

type tcf2Config struct {
	HostConfig    config.TCF2
	AccountConfig config.AccountGDPR
}

// NewTCF2Config creates an instance of tcf2Config which implements the TCF2ConfigReader interface
func NewTCF2Config(hostConfig config.TCF2, accountConfig config.AccountGDPR) TCF2ConfigReader {
	return &tcf2Config{
		HostConfig:    hostConfig,
		AccountConfig: accountConfig,
	}
}

// IsEnabled indicates if TCF2 is enabled
func (tc *tcf2Config) IsEnabled() bool {
	return tc.HostConfig.Enabled
}

// IntegrationEnabled checks if a given integration type is enabled at the account level. If it is not set at the
// account level, the host TCF2 enabled flag is used to determine if the integration type is enabled.
func (tc *tcf2Config) IntegrationEnabled(integrationType config.IntegrationType) bool {
	if accountEnabled := tc.AccountConfig.EnabledForIntegrationType(integrationType); accountEnabled != nil {
		return *accountEnabled
	}
	return tc.HostConfig.Enabled
}

// PurposeEnforced checks if full enforcement is turned on for a given purpose by first looking at the account
// settings, and if not set there, defaulting to the host configuration. With full enforcement enabled, the
// GDPR full enforcement algorithm will execute for that purpose determining legal basis; otherwise it's skipped.
func (tc *tcf2Config) PurposeEnforced(purpose consentconstants.Purpose) bool {
	if value, exists := tc.AccountConfig.PurposeEnforced(purpose); exists {
		return value
	}

	value := tc.HostConfig.PurposeEnforced(purpose)
	return value
}

// PurposeEnforcingVendors checks if enforcing vendors is turned on for a given purpose by first looking at the
// account settings, and if not set there, defaulting to the host configuration. With enforcing vendors enabled,
// the GDPR full enforcement algorithm considers the GVL when determining legal basis; otherwise it's skipped.
func (tc *tcf2Config) PurposeEnforcingVendors(purpose consentconstants.Purpose) bool {
	if value, exists := tc.AccountConfig.PurposeEnforcingVendors(purpose); exists {
		return value
	}

	value := tc.HostConfig.PurposeEnforcingVendors(purpose)
	return value
}

// PurposeVendorException checks if the specified bidder is considered a vendor exception for a given purpose by first
// looking at the account settings, and if not set there, defaulting to the host configuration. If a bidder is a vendor
// exception, the GDPR full enforcement algorithm will bypass the legal basis calculation assuming the request is valid
// and there isn't a "deny all" publisher restriction
func (tc *tcf2Config) PurposeVendorException(purpose consentconstants.Purpose, bidder openrtb_ext.BidderName) bool {
	if value, exists := tc.AccountConfig.PurposeVendorException(purpose, bidder); exists {
		return value
	}
	value := tc.HostConfig.PurposeVendorException(purpose, bidder)
	return value
}

// FeatureOneEnforced checks if special feature one is enforced by first looking at the account settings, and if not
// set there, defaulting to the host configuration. If it is enforced, PBS will determine whether geo information
// may be passed through in the bid request.
func (tc *tcf2Config) FeatureOneEnforced() bool {
	if value, exists := tc.AccountConfig.FeatureOneEnforced(); exists {
		return value
	}
	value := tc.HostConfig.FeatureOneEnforced()
	return value
}

// FeatureOneVendorException checks if the specified bidder is considered a vendor exception for special feature one
// by first looking at the account settings, and if not set there, defaulting to the host configuration. If a bidder
// is a vendor exception, PBS will bypass the pass geo calculation passing the geo information in the bid request.
func (tc *tcf2Config) FeatureOneVendorException(bidder openrtb_ext.BidderName) bool {
	if value, exists := tc.AccountConfig.FeatureOneVendorException(bidder); exists {
		return value
	}
	value := tc.HostConfig.FeatureOneVendorException(bidder)
	return value
}

// PurposeOneTreatmentEnabled checks if purpose one treatment is enabled by first looking at the account settings, and
// if not set there, defaulting to the host configuration.
func (tc *tcf2Config) PurposeOneTreatmentEnabled() bool {
	if value, exists := tc.AccountConfig.PurposeOneTreatmentEnabled(); exists {
		return value
	}
	value := tc.HostConfig.PurposeOneTreatmentEnabled()
	return value
}

// PurposeOneTreatmentAccessAllowed checks if purpose one treatment access is allowed by first looking at the account
// settings, and if not set there, defaulting to the host configuration.
func (tc *tcf2Config) PurposeOneTreatmentAccessAllowed() bool {
	if value, exists := tc.AccountConfig.PurposeOneTreatmentAccessAllowed(); exists {
		return value
	}
	value := tc.HostConfig.PurposeOneTreatmentAccessAllowed()
	return value
}

// BasicEnforcementVendor checks if the given bidder is considered a basic enforcement vendor by looking at the account
// settings, and if not set there, defaulting to false. If set, the legal basis calculation for the bidder only considers
// consent to the purpose, not the vendor. The idea is that the publisher trusts this vendor to enforce the
// appropriate rules on their own. This only comes into play when enforceVendors is true as it lists those vendors that
// are exempt for vendor enforcement.
func (tc *tcf2Config) BasicEnforcementVendor(bidder openrtb_ext.BidderName) bool {
	if value, exists := tc.AccountConfig.BasicEnforcementVendor(bidder); exists {
		return value
	}
	return false
}
