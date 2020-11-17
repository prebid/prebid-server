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
	GDPR          AccountGDPR `mapstructure:"gdpr" json:"gdpr"`
}

// AccountGDPR represents account-specific GDPR configuration
type AccountGDPR struct {
	Enabled            *bool                  `mapstructure:"enabled" json:"enabled,omitempty"`
	IntegrationEnabled AccountGDPRIntegration `mapstructure:"integration_enabled" json:"integration_enabled"`
}

// EnabledForIntegrationType indicates whether GDPR is turned on at the account level for the specified integration type
// by using the integration type setting if defined or the general GDPR setting if defined; otherwise it returns nil
func (a *AccountGDPR) EnabledForIntegrationType(integrationType IntegrationType) *bool {
	var integrationEnabled *bool

	switch integrationType {
	case IntegrationTypeAMP:
		integrationEnabled = a.IntegrationEnabled.AMP
	case IntegrationTypeApp:
		integrationEnabled = a.IntegrationEnabled.App
	case IntegrationTypeVideo:
		integrationEnabled = a.IntegrationEnabled.Video
	case IntegrationTypeWeb:
		integrationEnabled = a.IntegrationEnabled.Web
	}

	if integrationEnabled != nil {
		return integrationEnabled
	}
	if a.Enabled != nil {
		return a.Enabled
	}

	return nil
}

// AccountGDPRIntegration indicates whether GDPR is enabled for each integration type
type AccountGDPRIntegration struct {
	AMP   *bool `mapstructure:"amp" json:"amp,omitempty"`
	App   *bool `mapstructure:"app" json:"app,omitempty"`
	Video *bool `mapstructure:"video" json:"video,omitempty"`
	Web   *bool `mapstructure:"web" json:"web,omitempty"`
}
