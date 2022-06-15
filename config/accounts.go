package config

import (
	"github.com/prebid/go-gdpr/consentconstants"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// IntegrationType enumerates the values of integrations Prebid Server can configure for an account
type IntegrationType string

// Possible values of integration types Prebid Server can configure for an account
const (
	IntegrationTypeAMP   IntegrationType = "amp"
	IntegrationTypeApp   IntegrationType = "app"
	IntegrationTypeVideo IntegrationType = "video"
	IntegrationTypeWeb   IntegrationType = "web"
)

// Account represents a publisher account configuration
type Account struct {
	ID                      string      `mapstructure:"id" json:"id"`
	Disabled                bool        `mapstructure:"disabled" json:"disabled"`
	CacheTTL                DefaultTTLs `mapstructure:"cache_ttl" json:"cache_ttl"`
	EventsEnabled           bool        `mapstructure:"events_enabled" json:"events_enabled"`
	CCPA                    AccountCCPA `mapstructure:"ccpa" json:"ccpa"`
	GDPR                    AccountGDPR `mapstructure:"gdpr" json:"gdpr"`
	DebugAllow              bool        `mapstructure:"debug_allow" json:"debug_allow"`
	DefaultIntegration      string      `mapstructure:"default_integration" json:"default_integration"`
	CookieSync              CookieSync  `mapstructure:"cookie_sync" json:"cookie_sync"`
	Events                  Events      `mapstructure:"events" json:"events"` // Don't enable this feature. It is still under developmment - https://github.com/prebid/prebid-server/issues/1725
	TruncateTargetAttribute *int        `mapstructure:"truncate_target_attr" json:"truncate_target_attr"`
}

// CookieSync represents the account-level defaults for the cookie sync endpoint.
type CookieSync struct {
	DefaultLimit    *int  `mapstructure:"default_limit" json:"default_limit"`
	MaxLimit        *int  `mapstructure:"max_limit" json:"max_limit"`
	DefaultCoopSync *bool `mapstructure:"default_coop_sync" json:"default_coop_sync"`
}

// AccountCCPA represents account-specific CCPA configuration
type AccountCCPA struct {
	Enabled            *bool              `mapstructure:"enabled" json:"enabled,omitempty"`
	IntegrationEnabled AccountIntegration `mapstructure:"integration_enabled" json:"integration_enabled"`
}

// EnabledForIntegrationType indicates whether CCPA is turned on at the account level for the specified integration type
// by using the integration type setting if defined or the general CCPA setting if defined; otherwise it returns nil
func (a *AccountCCPA) EnabledForIntegrationType(integrationType IntegrationType) *bool {
	if integrationEnabled := a.IntegrationEnabled.GetByIntegrationType(integrationType); integrationEnabled != nil {
		return integrationEnabled
	}
	return a.Enabled
}

// AccountGDPR represents account-specific GDPR configuration
type AccountGDPR struct {
	Enabled            *bool              `mapstructure:"enabled" json:"enabled,omitempty"`
	IntegrationEnabled AccountIntegration `mapstructure:"integration_enabled" json:"integration_enabled"`
	// Array of basic enforcement vendors that is used to create the hash table so vendor names can be instantly accessed
	BasicEnforcementVendors    []string `mapstructure:"basic_enforcement_vendors" json:"basic_enforcement_vendors"`
	BasicEnforcementVendorsMap map[string]struct{}
	Purpose1                   AccountGDPRPurpose `mapstructure:"purpose1" json:"purpose1"`
	Purpose2                   AccountGDPRPurpose `mapstructure:"purpose2" json:"purpose2"`
	Purpose3                   AccountGDPRPurpose `mapstructure:"purpose3" json:"purpose3"`
	Purpose4                   AccountGDPRPurpose `mapstructure:"purpose4" json:"purpose4"`
	Purpose5                   AccountGDPRPurpose `mapstructure:"purpose5" json:"purpose5"`
	Purpose6                   AccountGDPRPurpose `mapstructure:"purpose6" json:"purpose6"`
	Purpose7                   AccountGDPRPurpose `mapstructure:"purpose7" json:"purpose7"`
	Purpose8                   AccountGDPRPurpose `mapstructure:"purpose8" json:"purpose8"`
	Purpose9                   AccountGDPRPurpose `mapstructure:"purpose9" json:"purpose9"`
	Purpose10                  AccountGDPRPurpose `mapstructure:"purpose10" json:"purpose10"`
	// Hash table of purpose configs for convenient purpose config lookup
	PurposeConfigs      map[consentconstants.Purpose]*AccountGDPRPurpose
	PurposeOneTreatment AccountGDPRPurposeOneTreatment `mapstructure:"purpose_one_treatment" json:"purpose_one_treatment"`
	SpecialFeature1     AccountGDPRSpecialFeature      `mapstructure:"special_feature1" json:"special_feature1"`
}

// BasicEnforcementVendor checks if the given bidder is considered a basic enforcement vendor which indicates whether
// weak vendor enforcement applies to that bidder.
func (a *AccountGDPR) BasicEnforcementVendor(bidder openrtb_ext.BidderName) (value, exists bool) {
	if a.BasicEnforcementVendorsMap == nil {
		return false, false
	}
	_, found := a.BasicEnforcementVendorsMap[string(bidder)]

	return found, true
}

// EnabledForIntegrationType indicates whether GDPR is turned on at the account level for the specified integration type
// by using the integration type setting if defined or the general GDPR setting if defined; otherwise it returns nil.
func (a *AccountGDPR) EnabledForIntegrationType(integrationType IntegrationType) *bool {
	if integrationEnabled := a.IntegrationEnabled.GetByIntegrationType(integrationType); integrationEnabled != nil {
		return integrationEnabled
	}
	return a.Enabled
}

// FeatureOneEnforced gets the account level feature one enforced setting returning the value and whether or not it
// was set. If not set, a default value of true is returned matching host default behavior.
func (a *AccountGDPR) FeatureOneEnforced() (value, exists bool) {
	if a.SpecialFeature1.Enforce == nil {
		return true, false
	}
	return *a.SpecialFeature1.Enforce, true
}

// FeatureOneVendorException checks if the given bidder is a vendor exception.
func (a *AccountGDPR) FeatureOneVendorException(bidder openrtb_ext.BidderName) (value, exists bool) {
	if a.SpecialFeature1.VendorExceptionMap == nil {
		return false, false
	}
	_, found := a.SpecialFeature1.VendorExceptionMap[bidder]

	return found, true
}

// PurposeEnforced checks if full enforcement is turned on for a given purpose at the account level. It returns the
// enforcement strategy type and whether or not it is set on the account. If not set, a default value of true is
// returned matching host default behavior.
func (a *AccountGDPR) PurposeEnforced(purpose consentconstants.Purpose) (value, exists bool) {
	if a.PurposeConfigs[purpose] == nil {
		return true, false
	}
	if a.PurposeConfigs[purpose].EnforcePurpose == TCF2FullEnforcement {
		return true, true
	}
	if a.PurposeConfigs[purpose].EnforcePurpose == TCF2NoEnforcement {
		return false, true
	}
	return true, false
}

// PurposeEnforcingVendors gets the account level enforce vendors setting for a given purpose returning the value and
// whether or not it is set. If not set, a default value of true is returned matching host default behavior.
func (a *AccountGDPR) PurposeEnforcingVendors(purpose consentconstants.Purpose) (value, exists bool) {
	if a.PurposeConfigs[purpose] == nil {
		return true, false
	}
	if a.PurposeConfigs[purpose].EnforceVendors == nil {
		return true, false
	}
	return *a.PurposeConfigs[purpose].EnforceVendors, true
}

// PurposeVendorException checks if the given bidder is a vendor exception for a given purpose.
func (a *AccountGDPR) PurposeVendorException(purpose consentconstants.Purpose, bidder openrtb_ext.BidderName) (value, exists bool) {
	if a.PurposeConfigs[purpose] == nil {
		return false, false
	}
	if a.PurposeConfigs[purpose].VendorExceptionMap == nil {
		return false, false
	}
	_, found := a.PurposeConfigs[purpose].VendorExceptionMap[bidder]

	return found, true
}

// PurposeOneTreatmentEnabled gets the account level purpose one treatment enabled setting returning the value and
// whether or not it is set. If not set, a default value of true is returned matching host default behavior.
func (a *AccountGDPR) PurposeOneTreatmentEnabled() (value, exists bool) {
	if a.PurposeOneTreatment.Enabled == nil {
		return true, false
	}
	return *a.PurposeOneTreatment.Enabled, true
}

// PurposeOneTreatmentAccessAllowed gets the account level purpose one treatment access allowed setting returning the
// value and whether or not it is set. If not set, a default value of true is returned matching host default behavior.
func (a *AccountGDPR) PurposeOneTreatmentAccessAllowed() (value, exists bool) {
	if a.PurposeOneTreatment.AccessAllowed == nil {
		return true, false
	}
	return *a.PurposeOneTreatment.AccessAllowed, true
}

// AccountGDPRPurpose represents account-specific GDPR purpose configuration
type AccountGDPRPurpose struct {
	EnforcePurpose string `mapstructure:"enforce_purpose" json:"enforce_purpose,omitempty"`
	EnforceVendors *bool  `mapstructure:"enforce_vendors" json:"enforce_vendors,omitempty"`
	// Array of vendor exceptions that is used to create the hash table VendorExceptionMap so vendor names can be instantly accessed
	VendorExceptions   []openrtb_ext.BidderName `mapstructure:"vendor_exceptions" json:"vendor_exceptions"`
	VendorExceptionMap map[openrtb_ext.BidderName]struct{}
}

// AccountGDPRSpecialFeature represents account-specific GDPR special feature configuration
type AccountGDPRSpecialFeature struct {
	Enforce *bool `mapstructure:"enforce" json:"enforce"`
	// Array of vendor exceptions that is used to create the hash table VendorExceptionMap so vendor names can be instantly accessed
	VendorExceptions   []openrtb_ext.BidderName `mapstructure:"vendor_exceptions" json:"vendor_exceptions"`
	VendorExceptionMap map[openrtb_ext.BidderName]struct{}
}

// AccountGDPRPurposeOneTreatment represents account-specific GDPR purpose one treatment configuration
type AccountGDPRPurposeOneTreatment struct {
	Enabled       *bool `mapstructure:"enabled"`
	AccessAllowed *bool `mapstructure:"access_allowed"`
}

// AccountIntegration indicates whether a particular privacy policy (GDPR, CCPA) is enabled for each integration type
type AccountIntegration struct {
	AMP   *bool `mapstructure:"amp" json:"amp,omitempty"`
	App   *bool `mapstructure:"app" json:"app,omitempty"`
	Video *bool `mapstructure:"video" json:"video,omitempty"`
	Web   *bool `mapstructure:"web" json:"web,omitempty"`
}

// GetByIntegrationType looks up the account integration enabled setting for the specified integration type
func (a *AccountIntegration) GetByIntegrationType(integrationType IntegrationType) *bool {
	var integrationEnabled *bool

	switch integrationType {
	case IntegrationTypeAMP:
		integrationEnabled = a.AMP
	case IntegrationTypeApp:
		integrationEnabled = a.App
	case IntegrationTypeVideo:
		integrationEnabled = a.Video
	case IntegrationTypeWeb:
		integrationEnabled = a.Web
	}

	return integrationEnabled
}
