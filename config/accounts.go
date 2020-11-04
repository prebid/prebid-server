package config

// RequestType : Request type enumeration
type RequestType string

// The request types
const (
	RequestTypeAMP   RequestType = "AMP"
	RequestTypeApp   RequestType = "app"
	RequestTypeVideo RequestType = "video"
	RequestTypeWeb   RequestType = "web"
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

// EnabledForRequestType indicates whether GDPR is turned on at the account level for the specified request type
// by using the request type setting if defined or the general GDPR setting if defined; otherwise it returns nil
func (a *AccountGDPR) EnabledForRequestType(requestType RequestType) *bool {
	var integrationEnabled *bool

	switch requestType {
	case RequestTypeAMP:
		integrationEnabled = a.IntegrationEnabled.AMP
	case RequestTypeApp:
		integrationEnabled = a.IntegrationEnabled.App
	case RequestTypeVideo:
		integrationEnabled = a.IntegrationEnabled.Video
	case RequestTypeWeb:
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

// AccountGDPRIntegration indicates whether GDPR is enabled for each request type
type AccountGDPRIntegration struct {
	AMP   *bool `mapstructure:"amp"   json:"amp,omitempty"`
	App   *bool `mapstructure:"app"   json:"app,omitempty"`
	Video *bool `mapstructure:"video" json:"video,omitempty"`
	Web   *bool `mapstructure:"web"   json:"web,omitempty"`
}
