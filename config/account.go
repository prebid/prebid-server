package config

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/prebid/go-gdpr/consentconstants"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/iputil"
)

// ChannelType enumerates the values of integrations Prebid Server can configure for an account
type ChannelType string

// Possible values of channel types Prebid Server can configure for an account
const (
	ChannelAMP   ChannelType = "amp"
	ChannelApp   ChannelType = "app"
	ChannelVideo ChannelType = "video"
	ChannelWeb   ChannelType = "web"
	ChannelDOOH  ChannelType = "dooh"
)

// Account represents a publisher account configuration
type Account struct {
	ID                      string                                      `mapstructure:"id" json:"id"`
	Disabled                bool                                        `mapstructure:"disabled" json:"disabled"`
	CacheTTL                DefaultTTLs                                 `mapstructure:"cache_ttl" json:"cache_ttl"`
	CCPA                    AccountCCPA                                 `mapstructure:"ccpa" json:"ccpa"`
	GDPR                    AccountGDPR                                 `mapstructure:"gdpr" json:"gdpr"`
	DebugAllow              bool                                        `mapstructure:"debug_allow" json:"debug_allow"`
	DefaultIntegration      string                                      `mapstructure:"default_integration" json:"default_integration"`
	CookieSync              CookieSync                                  `mapstructure:"cookie_sync" json:"cookie_sync"`
	Events                  Events                                      `mapstructure:"events" json:"events"` // Don't enable this feature. It is still under developmment - https://github.com/prebid/prebid-server/issues/1725
	TruncateTargetAttribute *int                                        `mapstructure:"truncate_target_attr" json:"truncate_target_attr"`
	AlternateBidderCodes    *openrtb_ext.ExtAlternateBidderCodes        `mapstructure:"alternatebiddercodes" json:"alternatebiddercodes"`
	Hooks                   AccountHooks                                `mapstructure:"hooks" json:"hooks"`
	PriceFloors             AccountPriceFloors                          `mapstructure:"price_floors" json:"price_floors"`
	Validations             Validations                                 `mapstructure:"validations" json:"validations"`
	DefaultBidLimit         int                                         `mapstructure:"default_bid_limit" json:"default_bid_limit"`
	BidAdjustments          *openrtb_ext.ExtRequestPrebidBidAdjustments `mapstructure:"bidadjustments" json:"bidadjustments"`
	Privacy                 AccountPrivacy                              `mapstructure:"privacy" json:"privacy"`
	PreferredMediaType      openrtb_ext.PreferredMediaType              `mapstructure:"preferredmediatype" json:"preferredmediatype"`
	TargetingPrefix         string                                      `mapstructure:"targeting_prefix" json:"targeting_prefix"`
}

// CookieSync represents the account-level defaults for the cookie sync endpoint.
type CookieSync struct {
	DefaultLimit    *int  `mapstructure:"default_limit" json:"default_limit"`
	MaxLimit        *int  `mapstructure:"max_limit" json:"max_limit"`
	DefaultCoopSync *bool `mapstructure:"default_coop_sync" json:"default_coop_sync"`
}

// AccountCCPA represents account-specific CCPA configuration
type AccountCCPA struct {
	Enabled        *bool          `mapstructure:"enabled" json:"enabled,omitempty"`
	ChannelEnabled AccountChannel `mapstructure:"channel_enabled" json:"channel_enabled"`
}

type AccountPriceFloors struct {
	Enabled                bool              `mapstructure:"enabled" json:"enabled"`
	EnforceFloorsRate      int               `mapstructure:"enforce_floors_rate" json:"enforce_floors_rate"`
	AdjustForBidAdjustment bool              `mapstructure:"adjust_for_bid_adjustment" json:"adjust_for_bid_adjustment"`
	EnforceDealFloors      bool              `mapstructure:"enforce_deal_floors" json:"enforce_deal_floors"`
	UseDynamicData         bool              `mapstructure:"use_dynamic_data" json:"use_dynamic_data"`
	MaxRule                int               `mapstructure:"max_rules" json:"max_rules"`
	MaxSchemaDims          int               `mapstructure:"max_schema_dims" json:"max_schema_dims"`
	Fetcher                AccountFloorFetch `mapstructure:"fetch" json:"fetch"`
}

// AccountFloorFetch defines the configuration for dynamic floors fetching.
type AccountFloorFetch struct {
	Enabled       bool   `mapstructure:"enabled" json:"enabled"`
	URL           string `mapstructure:"url" json:"url"`
	Timeout       int    `mapstructure:"timeout_ms" json:"timeout_ms"`
	MaxFileSizeKB int    `mapstructure:"max_file_size_kb" json:"max_file_size_kb"`
	MaxRules      int    `mapstructure:"max_rules" json:"max_rules"`
	MaxAge        int    `mapstructure:"max_age_sec" json:"max_age_sec"`
	Period        int    `mapstructure:"period_sec" json:"period_sec"`
	MaxSchemaDims int    `mapstructure:"max_schema_dims" json:"max_schema_dims"`
	AccountID     string `mapstructure:"accountID" json:"accountID"`
}

func (pf *AccountPriceFloors) validate(errs []error) []error {
	if pf.EnforceFloorsRate < 0 || pf.EnforceFloorsRate > 100 {
		errs = append(errs, fmt.Errorf(`account_defaults.price_floors.enforce_floors_rate should be between 0 and 100`))
	}

	if pf.MaxRule < 0 || pf.MaxRule > math.MaxInt32 {
		errs = append(errs, fmt.Errorf(`account_defaults.price_floors.max_rules should be between 0 and %v`, math.MaxInt32))
	}

	if pf.MaxSchemaDims < 0 || pf.MaxSchemaDims > 20 {
		errs = append(errs, fmt.Errorf(`account_defaults.price_floors.max_schema_dims should be between 0 and 20`))
	}

	if pf.Fetcher.Period > pf.Fetcher.MaxAge {
		errs = append(errs, fmt.Errorf(`account_defaults.price_floors.fetch.period_sec should be less than account_defaults.price_floors.fetch.max_age_sec`))
	}

	if pf.Fetcher.Period < 300 {
		errs = append(errs, fmt.Errorf(`account_defaults.price_floors.fetch.period_sec should not be less than 300 seconds`))
	}

	if pf.Fetcher.MaxAge < 600 {
		errs = append(errs, fmt.Errorf(`account_defaults.price_floors.fetch.max_age_sec should not be less than 600 seconds and greater than maximum integer value`))
	}

	if !(pf.Fetcher.Timeout > 10 && pf.Fetcher.Timeout < 10000) {
		errs = append(errs, fmt.Errorf(`account_defaults.price_floors.fetch.timeout_ms should be between 10 to 10,000 miliseconds`))
	}

	if pf.Fetcher.MaxRules < 0 {
		errs = append(errs, fmt.Errorf(`account_defaults.price_floors.fetch.max_rules should be greater than or equal to 0`))
	}

	if pf.Fetcher.MaxFileSizeKB < 0 {
		errs = append(errs, fmt.Errorf(`account_defaults.price_floors.fetch.max_file_size_kb should be greater than or equal to 0`))
	}

	if !(pf.Fetcher.MaxSchemaDims >= 0 && pf.Fetcher.MaxSchemaDims < 20) {
		errs = append(errs, fmt.Errorf(`account_defaults.price_floors.fetch.max_schema_dims should not be less than 0 and greater than 20`))
	}

	return errs
}

func (pf *AccountPriceFloors) IsAdjustForBidAdjustmentEnabled() bool {
	return pf.AdjustForBidAdjustment
}

// EnabledForChannelType indicates whether CCPA is turned on at the account level for the specified channel type
// by using the channel type setting if defined or the general CCPA setting if defined; otherwise it returns nil
func (a *AccountCCPA) EnabledForChannelType(channelType ChannelType) *bool {
	if channelEnabled := a.ChannelEnabled.GetByChannelType(channelType); channelEnabled != nil {
		return channelEnabled
	}
	return a.Enabled
}

// AccountGDPR represents account-specific GDPR configuration
type AccountGDPR struct {
	Enabled        *bool          `mapstructure:"enabled" json:"enabled,omitempty"`
	ChannelEnabled AccountChannel `mapstructure:"channel_enabled" json:"channel_enabled"`
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
	EEACountries        []string                       `mapstructure:"eea_countries" json:"eea_countries"`
}

// EnabledForChannelType indicates whether GDPR is turned on at the account level for the specified channel type
// by using the channel type setting if defined or the general GDPR setting if defined; otherwise it returns nil.
func (a *AccountGDPR) EnabledForChannelType(channelType ChannelType) *bool {
	if channelEnabled := a.ChannelEnabled.GetByChannelType(channelType); channelEnabled != nil {
		return channelEnabled
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
	if a.PurposeConfigs[purpose].EnforcePurpose == nil {
		return true, false
	}
	return *a.PurposeConfigs[purpose].EnforcePurpose, true
}

// PurposeEnforcementAlgo checks the purpose enforcement algo for a given purpose by first
// looking at the account settings, and if not set there, defaulting to the host configuration.
func (a *AccountGDPR) PurposeEnforcementAlgo(purpose consentconstants.Purpose) (value TCF2EnforcementAlgo, exists bool) {
	var c *AccountGDPRPurpose
	c, exists = a.PurposeConfigs[purpose]

	if exists && (c.EnforceAlgoID == TCF2BasicEnforcement || c.EnforceAlgoID == TCF2FullEnforcement) {
		return c.EnforceAlgoID, true
	}
	return TCF2UndefinedEnforcement, false
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

// PurposeVendorExceptions returns the vendor exception map for a given purpose.
func (a *AccountGDPR) PurposeVendorExceptions(purpose consentconstants.Purpose) (value map[string]struct{}, exists bool) {
	c, exists := a.PurposeConfigs[purpose]

	if exists && c.VendorExceptionMap != nil {
		return c.VendorExceptionMap, true
	}
	return nil, false
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
	EnforceAlgo string `mapstructure:"enforce_algo" json:"enforce_algo,omitempty"`
	// Integer representation of enforcement algo for performance improvement on compares
	EnforceAlgoID  TCF2EnforcementAlgo
	EnforcePurpose *bool `mapstructure:"enforce_purpose" json:"enforce_purpose,omitempty"`
	EnforceVendors *bool `mapstructure:"enforce_vendors" json:"enforce_vendors,omitempty"`
	// Array of vendor exceptions that is used to create the hash table VendorExceptionMap so vendor names can be instantly accessed
	VendorExceptions   []string `mapstructure:"vendor_exceptions" json:"vendor_exceptions"`
	VendorExceptionMap map[string]struct{}
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

// AccountChannel indicates whether a particular privacy policy (GDPR, CCPA) is enabled for each channel type
type AccountChannel struct {
	AMP   *bool `mapstructure:"amp" json:"amp,omitempty"`
	App   *bool `mapstructure:"app" json:"app,omitempty"`
	Video *bool `mapstructure:"video" json:"video,omitempty"`
	Web   *bool `mapstructure:"web" json:"web,omitempty"`
	DOOH  *bool `mapstructure:"dooh" json:"dooh,omitempty"`
}

// GetByChannelType looks up the account integration enabled setting for the specified channel type
func (a *AccountChannel) GetByChannelType(channelType ChannelType) *bool {
	var channelEnabled *bool

	switch channelType {
	case ChannelAMP:
		channelEnabled = a.AMP
	case ChannelApp:
		channelEnabled = a.App
	case ChannelVideo:
		channelEnabled = a.Video
	case ChannelWeb:
		channelEnabled = a.Web
	case ChannelDOOH:
		channelEnabled = a.DOOH
	}

	return channelEnabled
}

// AccountHooks represents account-specific hooks configuration
type AccountHooks struct {
	Modules       AccountModules    `mapstructure:"modules" json:"modules"`
	ExecutionPlan HookExecutionPlan `mapstructure:"execution_plan" json:"execution_plan"`
}

// AccountModules mapping provides account-level module configuration
// format: map[vendor_name]map[module_name]json.RawMessage
type AccountModules map[string]map[string]json.RawMessage

// ModuleConfig returns the account-level module config.
// The id argument must be passed in the form "vendor.module_name",
// otherwise an error is returned.
func (m AccountModules) ModuleConfig(id string) (json.RawMessage, error) {
	ns := strings.SplitN(id, ".", 2)
	if len(ns) < 2 {
		return nil, fmt.Errorf("ID must consist of vendor and module names separated by dot, got: %s", id)
	}

	vendor := ns[0]
	module := ns[1]
	return m[vendor][module], nil
}

type AccountPrivacy struct {
	AllowActivities *AllowActivities `mapstructure:"allowactivities" json:"allowactivities"`
	DSA             *AccountDSA      `mapstructure:"dsa" json:"dsa"`
	IPv6Config      IPv6             `mapstructure:"ipv6" json:"ipv6"`
	IPv4Config      IPv4             `mapstructure:"ipv4" json:"ipv4"`
	PrivacySandbox  PrivacySandbox   `mapstructure:"privacysandbox" json:"privacysandbox"`
}

type PrivacySandbox struct {
	TopicsDomain      string            `mapstructure:"topicsdomain"`
	CookieDeprecation CookieDeprecation `mapstructure:"cookiedeprecation"`
}

type CookieDeprecation struct {
	Enabled bool `mapstructure:"enabled"`
	TTLSec  int  `mapstructure:"ttl_sec"`
}

// AccountDSA represents DSA configuration
type AccountDSA struct {
	Default         string `mapstructure:"default" json:"default"`
	DefaultUnpacked *openrtb_ext.ExtRegsDSA
	GDPROnly        bool `mapstructure:"gdpr_only" json:"gdpr_only"`
}

type IPv6 struct {
	AnonKeepBits int `mapstructure:"anon_keep_bits" json:"anon_keep_bits"`
}

type IPv4 struct {
	AnonKeepBits int `mapstructure:"anon_keep_bits" json:"anon_keep_bits"`
}

func (ip *IPv6) Validate(errs []error) []error {
	if ip.AnonKeepBits > iputil.IPv6BitSize || ip.AnonKeepBits < 0 {
		err := fmt.Errorf("bits cannot exceed %d in ipv6 address, or be less than 0", iputil.IPv6BitSize)
		errs = append(errs, err)
	}
	return errs
}

func (ip *IPv4) Validate(errs []error) []error {
	if ip.AnonKeepBits > iputil.IPv4BitSize || ip.AnonKeepBits < 0 {
		err := fmt.Errorf("bits cannot exceed %d in ipv4 address, or be less than 0", iputil.IPv4BitSize)
		errs = append(errs, err)
	}
	return errs
}
