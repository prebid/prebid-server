package config

import (
	"fmt"
	"text/template"

	validator "github.com/asaskevich/govalidator"
	"github.com/prebid/prebid-server/macros"
)

type Adapter struct {
	Disabled         bool    `mapstructure:"disabled"`
	Endpoint         string  `mapstructure:"endpoint"`
	ExtraAdapterInfo string  `mapstructure:"extra_info"`
	Syncer           *Syncer `mapstructure:"usersync"`

	// needed for backwards compatibility
	UserSyncURL string `mapstructure:"usersync_url"`

	// needed for Rubicon
	XAPI AdapterXAPI `mapstructure:"xapi"`

	// needed for Facebook
	PlatformID string `mapstructure:"platform_id"`
	AppSecret  string `mapstructure:"app_secret"`

	// needed for commerce partners
	ComParams AdapterCommerce `mapstructure:"commerceparams"`
}

type AdapterXAPI struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Tracker  string `mapstructure:"tracker"`
}

// validateAdapters validates adapter's endpoint and user sync URL
func validateAdapters(adapterMap map[string]Adapter, errs []error) []error {
	for adapterName, adapter := range adapterMap {
		if !adapter.Disabled {
			errs = validateAdapterEndpoint(adapter.Endpoint, adapterName, errs)
		}
	}
	return errs
}

var testEndpointTemplateParams = macros.EndpointTemplateParams{
	Host:        "anyHost",
	PublisherID: "anyPublisherID",
	AccountID:   "anyAccountID",
	ZoneID:      "anyZoneID",
	SourceId:    "anySourceID",
	AdUnit:      "anyAdUnit",
}

// validateAdapterEndpoint makes sure that an adapter has a valid endpoint
// associated with it
func validateAdapterEndpoint(endpoint string, adapterName string, errs []error) []error {
	if endpoint == "" {
		return append(errs, fmt.Errorf("There's no default endpoint available for %s. Calls to this bidder/exchange will fail. "+
			"Please set adapters.%s.endpoint in your app config", adapterName, adapterName))
	}

	// Create endpoint template
	endpointTemplate, err := template.New("endpointTemplate").Parse(endpoint)
	if err != nil {
		return append(errs, fmt.Errorf("Invalid endpoint template: %s for adapter: %s. %v", endpoint, adapterName, err))
	}
	// Resolve macros (if any) in the endpoint URL
	resolvedEndpoint, err := macros.ResolveMacros(endpointTemplate, testEndpointTemplateParams)
	if err != nil {
		return append(errs, fmt.Errorf("Unable to resolve endpoint: %s for adapter: %s. %v", endpoint, adapterName, err))
	}
	// Validate the resolved endpoint
	//
	// Validating using both IsURL and IsRequestURL because IsURL allows relative paths
	// whereas IsRequestURL requires absolute path but fails to check other valid URL
	// format constraints.
	//
	// For example: IsURL will allow "abcd.com" but IsRequestURL won't
	// IsRequestURL will allow "http://http://abcd.com" but IsURL won't
	if !validator.IsURL(resolvedEndpoint) || !validator.IsRequestURL(resolvedEndpoint) {
		errs = append(errs, fmt.Errorf("The endpoint: %s for %s is not a valid URL", resolvedEndpoint, adapterName))
	}
	return errs
}
