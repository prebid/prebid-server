package config

import (
	"errors"
	"fmt"
	validator "github.com/asaskevich/govalidator"
	"github.com/golang/glog"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/util/sliceutil"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/prebid/prebid-server/openrtb_ext"
	"gopkg.in/yaml.v3"
)

// BidderInfos contains a mapping of bidder name to bidder info.
type BidderInfos map[string]BidderInfo

// BidderInfo specifies all configuration for a bidder except for enabled status, endpoint, and extra information.
type BidderInfo struct {
	Disabled         bool   `yaml:"disabled" mapstructure:"disabled"`
	Endpoint         string `yaml:"endpoint" mapstructure:"endpoint"`
	ExtraAdapterInfo string `yaml:"extra_info" mapstructure:"extra_info"`

	Maintainer              *MaintainerInfo   `yaml:"maintainer" mapstructure:"maintainer"`
	Capabilities            *CapabilitiesInfo `yaml:"capabilities" mapstructure:"capabilities"`
	ModifyingVastXmlAllowed bool              `yaml:"modifyingVastXmlAllowed" mapstructure:"modifyingVastXmlAllowed"`
	Debug                   *DebugInfo        `yaml:"debug" mapstructure:"debug"`
	GVLVendorID             uint16            `yaml:"gvlVendorID" mapstructure:"gvlVendorID"`

	Syncer *Syncer `yaml:"userSync" mapstructure:"userSync"`

	Experiment BidderInfoExperiment `yaml:"experiment" mapstructure:"experiment"`

	// needed for backwards compatibility
	UserSyncURL string `yaml:"usersync_url" mapstructure:"usersync_url"`

	// needed for Rubicon
	XAPI AdapterXAPI `yaml:"xapi" mapstructure:"xapi"`

	// needed for Facebook
	PlatformID string `yaml:"platform_id" mapstructure:"platform_id"`
	AppSecret  string `yaml:"app_secret" mapstructure:"app_secret"`
	// EndpointCompression determines, if set, the type of compression the bid request will undergo before being sent to the corresponding bid server
	EndpointCompression string `yaml:"endpointCompression" mapstructure:"endpointCompression"`
}

// BidderInfoExperiment specifies non-production ready feature config for a bidder
type BidderInfoExperiment struct {
	AdsCert BidderAdsCert `yaml:"adsCert" mapstructure:"adsCert"`
}

// BidderAdsCert enables Call Sign feature for bidder
type BidderAdsCert struct {
	Enabled bool `yaml:"enabled" mapstructure:"enabled"`
}

// MaintainerInfo specifies the support email address for a bidder.
type MaintainerInfo struct {
	Email string `yaml:"email" mapstructure:"email"`
}

// CapabilitiesInfo specifies the supported platforms for a bidder.
type CapabilitiesInfo struct {
	App  *PlatformInfo `yaml:"app" mapstructure:"app"`
	Site *PlatformInfo `yaml:"site" mapstructure:"site"`
}

// PlatformInfo specifies the supported media types for a bidder.
type PlatformInfo struct {
	MediaTypes []openrtb_ext.BidType `yaml:"mediaTypes" mapstructure:"mediaTypes"`
}

// DebugInfo specifies the supported debug options for a bidder.
type DebugInfo struct {
	Allow bool `yaml:"allow" mapstructure:"allow"`
}

type AdapterXAPI struct {
	Username string `yaml:"username" mapstructure:"username"`
	Password string `yaml:"password" mapstructure:"password"`
	Tracker  string `yaml:"tracker" mapstructure:"tracker"`
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
	Supports []string `yaml:"supports" mapstructure:"supports"`

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
//	url: "https://sync.bidderserver.com/usersync?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redirect={{.RedirectURL}}"
//	userMacro: "$UID"
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

func (bi BidderInfo) IsEnabled() bool {
	return !bi.Disabled
}

type InfoReader interface {
	Read() (map[string][]byte, error)
}

type InfoReaderFromDisk struct {
	Path string
}

func (r InfoReaderFromDisk) Read() (map[string][]byte, error) {
	bidderConfigs, err := ioutil.ReadDir(r.Path)
	if err != nil {
		log.Fatal(err)
	}

	bidderInfos := make(map[string][]byte)
	for _, bidderConfig := range bidderConfigs {
		if bidderConfig.IsDir() {
			continue //ignore directories
		}
		fileName := bidderConfig.Name()
		filePath := filepath.Join(r.Path, fileName)
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			return nil, err
		}
		bidderInfos[fileName] = data
	}
	return bidderInfos, nil
}

func LoadBidderInfoFromDisk(path string) (BidderInfos, error) {
	bidderInfoReader := InfoReaderFromDisk{Path: path}
	return LoadBidderInfo(bidderInfoReader)
}

func LoadBidderInfo(reader InfoReader) (BidderInfos, error) {
	return processBidderInfos(reader, openrtb_ext.NormalizeBidderName)
}

func processBidderInfos(reader InfoReader, normalizeBidderName func(string) (openrtb_ext.BidderName, bool)) (BidderInfos, error) {
	bidderConfigs, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("error loading bidders data")
	}

	infos := BidderInfos{}

	for fileName, data := range bidderConfigs {
		bidderName := strings.Split(fileName, ".")
		if len(bidderName) == 2 && bidderName[1] == "yaml" {
			normalizedBidderName, bidderNameExists := normalizeBidderName(bidderName[0])
			if !bidderNameExists {
				return nil, fmt.Errorf("error parsing config for bidder %s: unknown bidder", fileName)
			}
			info := BidderInfo{}
			if err := yaml.Unmarshal(data, &info); err != nil {
				return nil, fmt.Errorf("error parsing config for bidder %s: %v", fileName, err)
			}

			infos[string(normalizedBidderName)] = info
		}
	}
	return infos, nil
}

// ToGVLVendorIDMap transforms a BidderInfos object to a map of bidder names to GVL id.
// Disabled bidders are omitted from the result.
func (infos BidderInfos) ToGVLVendorIDMap() map[openrtb_ext.BidderName]uint16 {
	gvlVendorIds := make(map[openrtb_ext.BidderName]uint16, len(infos))
	for name, info := range infos {
		if info.IsEnabled() && info.GVLVendorID != 0 {
			gvlVendorIds[openrtb_ext.BidderName(name)] = info.GVLVendorID
		}
	}
	return gvlVendorIds
}

// validateBidderInfos validates bidder endpoint, info and syncer data
func (infos BidderInfos) validate(errs []error) []error {
	for bidderName, bidder := range infos {
		if bidder.IsEnabled() {
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
		return errors.New("at least one media type needs to be specified")
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

	for _, supports := range bidderInfo.Syncer.Supports {
		if !strings.EqualFold(supports, "iframe") && !strings.EqualFold(supports, "redirect") {
			return fmt.Errorf("syncer could not be created, invalid supported endpoint: %s", supports)
		}
	}

	return nil
}

func applyBidderInfoConfigOverrides(configBidderInfos BidderInfos, fsBidderInfos BidderInfos, normalizeBidderName func(string) (openrtb_ext.BidderName, bool)) (BidderInfos, error) {
	for bidderName, bidderInfo := range configBidderInfos {
		normalizedBidderName, bidderNameExists := normalizeBidderName(bidderName)
		if !bidderNameExists {
			return nil, fmt.Errorf("error setting configuration for bidder %s: unknown bidder", bidderName)
		}
		if fsBidderCfg, exists := fsBidderInfos[string(normalizedBidderName)]; exists {
			bidderInfo.Syncer = bidderInfo.Syncer.Override(fsBidderCfg.Syncer)

			if bidderInfo.Endpoint == "" && len(fsBidderCfg.Endpoint) > 0 {
				bidderInfo.Endpoint = fsBidderCfg.Endpoint
			}
			if bidderInfo.ExtraAdapterInfo == "" && len(fsBidderCfg.ExtraAdapterInfo) > 0 {
				bidderInfo.ExtraAdapterInfo = fsBidderCfg.ExtraAdapterInfo
			}
			if bidderInfo.Maintainer == nil && fsBidderCfg.Maintainer != nil {
				bidderInfo.Maintainer = fsBidderCfg.Maintainer
			}
			if bidderInfo.Capabilities == nil && fsBidderCfg.Capabilities != nil {
				bidderInfo.Capabilities = fsBidderCfg.Capabilities
			}
			if bidderInfo.Debug == nil && fsBidderCfg.Debug != nil {
				bidderInfo.Debug = fsBidderCfg.Debug
			}
			if bidderInfo.GVLVendorID == 0 && fsBidderCfg.GVLVendorID > 0 {
				bidderInfo.GVLVendorID = fsBidderCfg.GVLVendorID
			}
			if bidderInfo.XAPI.Username == "" && fsBidderCfg.XAPI.Username != "" {
				bidderInfo.XAPI.Username = fsBidderCfg.XAPI.Username
			}
			if bidderInfo.XAPI.Password == "" && fsBidderCfg.XAPI.Password != "" {
				bidderInfo.XAPI.Password = fsBidderCfg.XAPI.Password
			}
			if bidderInfo.XAPI.Tracker == "" && fsBidderCfg.XAPI.Tracker != "" {
				bidderInfo.XAPI.Tracker = fsBidderCfg.XAPI.Tracker
			}
			if bidderInfo.PlatformID == "" && fsBidderCfg.PlatformID != "" {
				bidderInfo.PlatformID = fsBidderCfg.PlatformID
			}
			if bidderInfo.AppSecret == "" && fsBidderCfg.AppSecret != "" {
				bidderInfo.AppSecret = fsBidderCfg.AppSecret
			}
			if bidderInfo.EndpointCompression == "" && fsBidderCfg.EndpointCompression != "" {
				bidderInfo.EndpointCompression = fsBidderCfg.EndpointCompression
			}

			// validate and try to apply the legacy usersync_url configuration in attempt to provide
			// an easier upgrade path. be warned, this will break if the bidder adds a second syncer
			// type and will eventually be removed after we've given hosts enough time to upgrade to
			// the new config.
			if bidderInfo.UserSyncURL != "" {
				if fsBidderCfg.Syncer == nil {
					return nil, fmt.Errorf("adapters.%s.usersync_url cannot be applied, bidder does not define a user sync", strings.ToLower(bidderName))
				}

				endpointsCount := 0
				if bidderInfo.Syncer.IFrame != nil {
					bidderInfo.Syncer.IFrame.URL = bidderInfo.UserSyncURL
					endpointsCount++
				}
				if bidderInfo.Syncer.Redirect != nil {
					bidderInfo.Syncer.Redirect.URL = bidderInfo.UserSyncURL
					endpointsCount++
				}

				// use Supports as a hint if there are no good defaults provided
				if endpointsCount == 0 {
					if sliceutil.ContainsStringIgnoreCase(bidderInfo.Syncer.Supports, "iframe") {
						bidderInfo.Syncer.IFrame = &SyncerEndpoint{URL: bidderInfo.UserSyncURL}
						endpointsCount++
					}
					if sliceutil.ContainsStringIgnoreCase(bidderInfo.Syncer.Supports, "redirect") {
						bidderInfo.Syncer.Redirect = &SyncerEndpoint{URL: bidderInfo.UserSyncURL}
						endpointsCount++
					}
				}

				if endpointsCount == 0 {
					return nil, fmt.Errorf("adapters.%s.usersync_url cannot be applied, bidder does not define user sync endpoints and does not define supported endpoints", strings.ToLower(bidderName))
				}

				// if the bidder defines both an iframe and redirect endpoint, we can't be sure which config value to
				// override, and  it wouldn't be both. this is a fatal configuration error.
				if endpointsCount > 1 {
					return nil, fmt.Errorf("adapters.%s.usersync_url cannot be applied, bidder defines multiple user sync endpoints or supports multiple endpoints", strings.ToLower(bidderName))
				}

				// provide a warning that this compatibility layer is temporary
				glog.Warningf("adapters.%s.usersync_url is deprecated and will be removed in a future version, please update to the latest user sync config values", strings.ToLower(bidderName))
			}

			fsBidderInfos[string(normalizedBidderName)] = bidderInfo
		} else {
			return nil, fmt.Errorf("error finding configuration for bidder %s: unknown bidder", bidderName)
		}
	}
	return fsBidderInfos, nil
}

// Override returns a new Syncer object where values in the original are replaced by non-empty/non-default
// values in the override, except for the Supports field which may not be overridden. No changes are made
// to the original or override Syncer.
func (s *Syncer) Override(original *Syncer) *Syncer {
	if s == nil && original == nil {
		return nil
	}

	var copy Syncer
	if original != nil {
		copy = *original
	}

	if s == nil {
		return &copy
	}

	if s.Key != "" {
		copy.Key = s.Key
	}

	if original == nil {
		copy.IFrame = s.IFrame.Override(nil)
		copy.Redirect = s.Redirect.Override(nil)
	} else {
		copy.IFrame = s.IFrame.Override(original.IFrame)
		copy.Redirect = s.Redirect.Override(original.Redirect)
	}

	if s.ExternalURL != "" {
		copy.ExternalURL = s.ExternalURL
	}

	if s.SupportCORS != nil {
		copy.SupportCORS = s.SupportCORS
	}

	return &copy
}

// Override returns a new SyncerEndpoint object where values in the original are replaced by non-empty/non-default
// values in the override. No changes are made to the original or override SyncerEndpoint.
func (s *SyncerEndpoint) Override(original *SyncerEndpoint) *SyncerEndpoint {
	if s == nil && original == nil {
		return nil
	}

	var copy SyncerEndpoint
	if original != nil {
		copy = *original
	}

	if s == nil {
		return &copy
	}

	if s.URL != "" {
		copy.URL = s.URL
	}

	if s.RedirectURL != "" {
		copy.RedirectURL = s.RedirectURL
	}

	if s.ExternalURL != "" {
		copy.ExternalURL = s.ExternalURL
	}

	if s.UserMacro != "" {
		copy.UserMacro = s.UserMacro
	}

	return &copy
}
