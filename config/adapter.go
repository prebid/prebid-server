package config

import (
	"fmt"
	"text/template"

	validator "github.com/asaskevich/govalidator"
	"github.com/prebid/prebid-server/macros"
)

type Adapter struct {
	Endpoint string `mapstructure:"endpoint"` // Required
	// UserSyncURL is the URL returned by /cookie_sync for this Bidder. It is _usually_ optional.
	// If not defined, sensible defaults will be derived based on the config.external_url.
	// Note that some Bidders don't have sensible defaults, because their APIs require an ID that will vary
	// from one PBS host to another.
	//
	// For these bidders, there will be a warning logged on startup that usersyncs will not work if you have not
	// defined one in the app config. Check your app logs for more info.
	//
	// This value will be interpreted as a Golang Template. At runtime, the following Template variables will be replaced.
	//
	//   {{.GDPR}}      -- This will be replaced with the "gdpr" property sent to /cookie_sync
	//   {{.Consent}}   -- This will be replaced with the "consent" property sent to /cookie_sync
	//   {{.USPrivacy}} -- This will be replaced with the "us_privacy" property sent to /cookie_sync
	//
	// For more info on templates, see: https://golang.org/pkg/text/template/
	UserSyncURL      string `mapstructure:"usersync_url"`
	Disabled         bool   `mapstructure:"disabled"`
	ExtraAdapterInfo string `mapstructure:"extra_info"`

	// needed for Rubicon
	XAPI AdapterXAPI `mapstructure:"xapi"`

	// needed for Facebook
	PlatformID string `mapstructure:"platform_id"`
	AppSecret  string `mapstructure:"app_secret"`
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
			// Verify that every adapter has a valid endpoint associated with it
			errs = validateAdapterEndpoint(adapter.Endpoint, adapterName, errs)

			// Verify that valid user_sync URLs are specified in the config
			errs = validateAdapterUserSyncURL(adapter.UserSyncURL, adapterName, errs)
		}
	}
	return errs
}

const (
	dummyHost        string = "dummyhost.com"
	dummyPublisherID string = "12"
	dummyAccountID   string = "some_account"
	dummyGDPR        string = "0"
	dummyGDPRConsent string = "someGDPRConsentString"
	dummyCCPA        string = "1NYN"
	dummyZoneID      string = "zone"
)

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
	resolvedEndpoint, err := macros.ResolveMacros(*endpointTemplate, macros.EndpointTemplateParams{
		Host:        dummyHost,
		PublisherID: dummyPublisherID,
		AccountID:   dummyAccountID,
		ZoneID:      dummyZoneID,
	})
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

// validateAdapterUserSyncURL validates an adapter's user sync URL if it is set
func validateAdapterUserSyncURL(userSyncURL string, adapterName string, errs []error) []error {
	if userSyncURL != "" {
		// Create user_sync URL template
		userSyncTemplate, err := template.New("userSyncTemplate").Parse(userSyncURL)
		if err != nil {
			return append(errs, fmt.Errorf("Invalid user sync URL template: %s for adapter: %s. %v", userSyncURL, adapterName, err))
		}
		// Resolve macros (if any) in the user_sync URL
		dummyMacroValues := macros.UserSyncTemplateParams{
			GDPR:        dummyGDPR,
			GDPRConsent: dummyGDPRConsent,
			USPrivacy:   dummyCCPA,
		}
		resolvedUserSyncURL, err := macros.ResolveMacros(*userSyncTemplate, dummyMacroValues)
		if err != nil {
			return append(errs, fmt.Errorf("Unable to resolve user sync URL: %s for adapter: %s. %v", userSyncURL, adapterName, err))
		}
		// Validate the resolved user sync URL
		//
		// Validating using both IsURL and IsRequestURL because IsURL allows relative paths
		// whereas IsRequestURL requires absolute path but fails to check other valid URL
		// format constraints.
		//
		// For example: IsURL will allow "abcd.com" but IsRequestURL won't
		// IsRequestURL will allow "http://http://abcd.com" but IsURL won't
		if !validator.IsURL(resolvedUserSyncURL) || !validator.IsRequestURL(resolvedUserSyncURL) {
			errs = append(errs, fmt.Errorf("The user_sync URL: %s for %s is invalid", resolvedUserSyncURL, adapterName))
		}
	}
	return errs
}
