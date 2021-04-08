package config

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
	ID            string      `mapstructure:"id" json:"id"`
	Disabled      bool        `mapstructure:"disabled" json:"disabled"`
	CacheTTL      DefaultTTLs `mapstructure:"cache_ttl" json:"cache_ttl"`
	EventsEnabled bool        `mapstructure:"events_enabled" json:"events_enabled"`
	CCPA          AccountCCPA `mapstructure:"ccpa" json:"ccpa"`
	GDPR          AccountGDPR `mapstructure:"gdpr" json:"gdpr"`
	DebugAllow    bool        `mapstructure:"debug_allow" json:"debug_allow"`
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
	Enabled                 *bool              `mapstructure:"enabled" json:"enabled,omitempty"`
	IntegrationEnabled      AccountIntegration `mapstructure:"integration_enabled" json:"integration_enabled"`
	BasicEnforcementVendors []string           `mapstructure:"basic_enforcement_vendors" json:"basic_enforcement_vendors"`
}

// EnabledForIntegrationType indicates whether GDPR is turned on at the account level for the specified integration type
// by using the integration type setting if defined or the general GDPR setting if defined; otherwise it returns nil
func (a *AccountGDPR) EnabledForIntegrationType(integrationType IntegrationType) *bool {

	if integrationEnabled := a.IntegrationEnabled.GetByIntegrationType(integrationType); integrationEnabled != nil {
		return integrationEnabled
	}
	return a.Enabled
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
