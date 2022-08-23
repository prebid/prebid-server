package config

import (
	"errors"
	"fmt"
	validator "github.com/asaskevich/govalidator"
	"github.com/prebid/prebid-server/macros"
	"io/ioutil"
	"log"
	"strings"
	"text/template"

	"github.com/prebid/prebid-server/openrtb_ext"
	"gopkg.in/yaml.v3"
)

// BidderInfos contains a mapping of bidder name to bidder info.
type BidderInfos map[string]BidderInfo

// BidderInfo specifies all configuration for a bidder except for enabled status, endpoint, and extra information.
type BidderInfo struct {
	Disabled         bool   `mapstructure:"disabled"`
	Endpoint         string `mapstructure:"endpoint"`
	ExtraAdapterInfo string `mapstructure:"extra_info"`

	Maintainer              *MaintainerInfo   `yaml:"maintainer"`
	Capabilities            *CapabilitiesInfo `yaml:"capabilities"`
	ModifyingVastXmlAllowed bool              `yaml:"modifyingVastXmlAllowed"`
	Debug                   *DebugInfo        `yaml:"debug"`
	GVLVendorID             uint16            `yaml:"gvlVendorID"`

	Syncer *Syncer `yaml:"userSync"`

	Experiment BidderInfoExperiment `yaml:"experiment"`

	// needed for backwards compatibility
	UserSyncURL string `mapstructure:"usersync_url"`

	// needed for Rubicon
	XAPI AdapterXAPI `mapstructure:"xapi"`

	// needed for Facebook
	PlatformID string `mapstructure:"platform_id"`
	AppSecret  string `mapstructure:"app_secret"`
}

// BidderInfoExperiment specifies non-production ready feature config for a bidder
type BidderInfoExperiment struct {
	AdsCert BidderAdsCert `yaml:"adsCert"`
}

// BidderAdsCert enables Call Sign feature for bidder
type BidderAdsCert struct {
	Enabled bool `yaml:"enabled"`
}

// MaintainerInfo specifies the support email address for a bidder.
type MaintainerInfo struct {
	Email string `yaml:"email"`
}

// CapabilitiesInfo specifies the supported platforms for a bidder.
type CapabilitiesInfo struct {
	App  *PlatformInfo `yaml:"app"`
	Site *PlatformInfo `yaml:"site"`
}

// PlatformInfo specifies the supported media types for a bidder.
type PlatformInfo struct {
	MediaTypes []openrtb_ext.BidType `yaml:"mediaTypes"`
}

// DebugInfo specifies the supported debug options for a bidder.
type DebugInfo struct {
	Allow bool `yaml:"allow"`
}

type AdapterXAPI struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Tracker  string `mapstructure:"tracker"`
}

// Syncer specifies the user sync settings for a bidder. This struct is shared by the account config,
// so it needs to have both yaml and mapstructure mappings.
type Syncer struct {
	// Key is used as the record key for the user sync cookie. We recommend using the bidder name
	// as the key for consistency, but that is not enforced as a requirement.
	Key string `yaml:"key" mapstructure:"key"`

	// Supports allows bidders to specify which user sync endpoints they support but which don't have
	// good defaults. Host companies should contact the bidder for the endpoint configuration. Hosts
	// may not override this value.
	Supports []string `yaml:"supports"`

	// IFrame configures an iframe endpoint for user syncing.
	IFrame *SyncerEndpoint `yaml:"iframe" mapstructure:"iframe"`

	// Redirect configures an redirect endpoint for user syncing. This is also known as an image
	// endpoint in the Prebid.js project.
	Redirect *SyncerEndpoint `yaml:"redirect" mapstructure:"redirect"`

	// ExternalURL is available as a macro to the RedirectURL template.
	ExternalURL string `yaml:"externalUrl" mapstructure:"external_url"`

	// SupportCORS identifies if CORS is supported for the user syncing endpoints.
	SupportCORS *bool `yaml:"supportCors" mapstructure:"support_cors"`
}

// SyncerEndpoint specifies the configuration of the URL returned by the /cookie_sync endpoint
// for a specific bidder. Bidders must specify at least one endpoint configuration to be eligible
// for selection during a user sync request.
//
// URL is the only required field, although we highly recommend to use the available macros to
// make the configuration readable and maintainable. User sync urls include a redirect url back to
// Prebid Server which is url escaped and can be very diffcult for humans to read.
//
// In most cases, bidders will specify a URL with a `{{.RedirectURL}}` macro for the call back to
// Prebid Server and a UserMacro which the bidder server will replace with the user's id. Example:
//
//  url: "https://sync.bidderserver.com/usersync?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redirect={{.RedirectURL}}"
//  userMacro: "$UID"
//
// Prebid Server is configured with a default RedirectURL template matching the /setuid call. This
// may be overridden for all bidders with the `user_sync.redirect_url` host configuration or for a
// specific bidder with the RedirectURL value in this struct.
type SyncerEndpoint struct {
	// URL is the endpoint on the bidder server the user will be redirected to when a user sync is
	// requested. The following macros are resolved at application startup:
	//
	//  {{.RedirectURL}} - This will be replaced with a redirect url generated using the RedirectURL
	//                     template and url escaped for safe inclusion in any part of the URL.
	//
	// The following macros are specific to individual requests and are resolved at runtime using the
	// Go template engine. For more information on Go templates, see: https://golang.org/pkg/text/template/
	//
	//  {{.GDPR}}        - This will be replaced with the "gdpr" property sent to /cookie_sync.
	//  {{.Consent}}     - This will be replaced with the "consent" property sent to /cookie_sync.
	//  {{.USPrivacy}}   - This will be replaced with the "us_privacy" property sent to /cookie_sync.
	URL string `yaml:"url" mapstructure:"url"`

	// RedirectURL is an endpoint on the host server the user will be redirected to when a user sync
	// request has been completed by the bidder server. The following macros are resolved at application
	// startup:
	//
	//  {{.ExternalURL}} - This will be replaced with the host server's externally reachable http path.
	//  {{.SyncerKey}}   - This will be replaced with the syncer key.
	//  {{.SyncType}}    - This will be replaced with the sync type, either 'b' for iframe syncs or 'i'
	//                     for redirect/image syncs.
	//  {{.UserMacro}}   - This will be replaced with the bidder server's user id macro.
	//
	// The endpoint on the host server is usually Prebid Server's /setuid endpoint. The default value is:
	// `{{.ExternalURL}}/setuid?bidder={{.SyncerKey}}&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&f={{.SyncType}}&uid={{.UserMacro}}`
	RedirectURL string `yaml:"redirectUrl" mapstructure:"redirect_url"`

	// ExternalURL is available as a macro to the RedirectURL template. If not specified, either the syncer configuration
	// value or the host configuration value is used.
	ExternalURL string `yaml:"externalUrl" mapstructure:"external_url"`

	// UserMacro is available as a macro to the RedirectURL template. This value is specific to the bidder server
	// and has no default.
	UserMacro string `yaml:"userMacro" mapstructure:"user_macro"`
}

func ProcessBidderInfos(path string) (BidderInfos, []error) {
	errs := make([]error, 0)
	bidderInfos, err := LoadBidderInfoFromDisk(path)
	if err != nil {
		return nil, append(errs, fmt.Errorf("Unable to load bidderconfigs %v", err))
	}
	errs = validateBidderInfos(bidderInfos)

	return bidderInfos, errs
}

// LoadBidderInfoFromDisk parses all static/bidder-info/{bidder}.yaml files from the file system.
func LoadBidderInfoFromDisk(path string) (BidderInfos, error) {
	return loadBidderInfo(path)
}

func loadBidderInfo(path string) (BidderInfos, error) {
	bidderConfigs, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}

	infos := BidderInfos{}

	for _, bidderConfig := range bidderConfigs {
		if bidderConfig.IsDir() {
			continue //or throw an error?
		}
		fileName := bidderConfig.Name()
		filePath := fmt.Sprintf("%v/%v", path, fileName)
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			return nil, err
		}

		info := BidderInfo{}
		if err := yaml.Unmarshal(data, &info); err != nil {
			return nil, fmt.Errorf("error parsing yaml for bidder %s: %v", fileName, err)
		}

		bidderName := strings.Split(fileName, ".yaml")
		infos[(bidderName[0])] = info
	}

	return infos, nil
}

// ToGVLVendorIDMap transforms a BidderInfos object to a map of bidder names to GVL id. Disabled
// bidders are omitted from the result.
func (infos BidderInfos) ToGVLVendorIDMap() map[openrtb_ext.BidderName]uint16 {
	m := make(map[openrtb_ext.BidderName]uint16, len(infos))
	for name, info := range infos {
		if !info.Disabled && info.GVLVendorID != 0 {
			m[openrtb_ext.BidderName(name)] = info.GVLVendorID
		}
	}
	return m
}

// vlidateBidderInfos validates bidder endpoint, info and syncer data
func validateBidderInfos(bidderInfos BidderInfos) []error {
	errs := make([]error, 0, 0)
	for bidderName, bidder := range bidderInfos {
		if !bidder.Disabled {
			errs = validateAdapterEndpoint(bidder.Endpoint, bidderName, errs)

			validateInfoErr := validateInfo(bidder, bidderName)
			if validateInfoErr != nil {
				errs = append(errs, validateInfoErr)
			}

			validateSyncerErr := validateSyncer(bidder)
			if validateSyncerErr != nil {
				errs = append(errs, validateSyncerErr)
			}
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
func validateAdapterEndpoint(endpoint string, bidderName string, errs []error) []error {
	if endpoint == "" {
		return append(errs, fmt.Errorf("There's no default endpoint available for %s. Calls to this bidder/exchange will fail. "+
			"Please set adapters.%s.endpoint in your app config", bidderName, bidderName))
	}

	// Create endpoint template
	endpointTemplate, err := template.New("endpointTemplate").Parse(endpoint)
	if err != nil {
		return append(errs, fmt.Errorf("Invalid endpoint template: %s for adapter: %s. %v", endpoint, bidderName, err))
	}
	// Resolve macros (if any) in the endpoint URL
	resolvedEndpoint, err := macros.ResolveMacros(endpointTemplate, testEndpointTemplateParams)
	if err != nil {
		return append(errs, fmt.Errorf("Unable to resolve endpoint: %s for adapter: %s. %v", endpoint, bidderName, err))
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
		errs = append(errs, fmt.Errorf("The endpoint: %s for %s is not a valid URL", resolvedEndpoint, bidderName))
	}
	return errs
}

func validateInfo(info BidderInfo, bidderName string) error {
	if err := validateMaintainer(info.Maintainer, bidderName); err != nil {
		return err
	}
	if err := validateCapabilities(info.Capabilities, bidderName); err != nil {
		return err
	}

	return nil
}

func validateMaintainer(info *MaintainerInfo, bidderName string) error {
	if info == nil || info.Email == "" {
		return fmt.Errorf("missing required field: maintainer.email for adapter: %s", bidderName)
	}
	return nil
}

func validateCapabilities(info *CapabilitiesInfo, bidderName string) error {
	if info == nil {
		return fmt.Errorf("missing required field: capabilities for adapter: %s", bidderName)
	}

	if info.App == nil && info.Site == nil {
		return fmt.Errorf("at least one of capabilities.site or capabilities.app must exist for adapter: %s", bidderName)
	}

	if info.App != nil {
		if err := validatePlatformInfo(info.App); err != nil {
			return fmt.Errorf("capabilities.app failed validation: %v for adapter: %s", err, bidderName)
		}
	}

	if info.Site != nil {
		if err := validatePlatformInfo(info.Site); err != nil {
			return fmt.Errorf("capabilities.site failed validation: %v, for adapter: %s", err, bidderName)
		}
	}
	return nil
}

func validatePlatformInfo(info *PlatformInfo) error {
	if len(info.MediaTypes) == 0 {
		return errors.New("mediaTypes should be an array with at least one string element")
	}

	for index, mediaType := range info.MediaTypes {
		if mediaType != "banner" && mediaType != "video" && mediaType != "native" && mediaType != "audio" {
			return fmt.Errorf("unrecognized media type at index %d: %s", index, mediaType)
		}
	}

	return nil
}

func validateSyncer(bidderInfo BidderInfo) error {
	if bidderInfo.Syncer == nil {
		return nil
	}

	for _, v := range bidderInfo.Syncer.Supports {
		if !strings.EqualFold(v, "iframe") && !strings.EqualFold(v, "redirect") {
			return fmt.Errorf("syncer could not be created, invalid supported endpoint: %s", v)
		}
	}

	return nil
}
