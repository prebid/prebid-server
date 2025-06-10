package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/openrtb_ext"

	validator "github.com/asaskevich/govalidator"
	"gopkg.in/yaml.v3"
)

// BidderInfos contains a mapping of bidder name to bidder info.
type BidderInfos map[string]BidderInfo

// BidderInfo specifies all configuration for a bidder except for enabled status, endpoint, and extra information.
type BidderInfo struct {
	AliasOf          string `yaml:"aliasOf" mapstructure:"aliasOf"`
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

	// needed for Rubicon
	XAPI AdapterXAPI `yaml:"xapi" mapstructure:"xapi"`

	// needed for Facebook
	PlatformID string `yaml:"platform_id" mapstructure:"platform_id"`
	AppSecret  string `yaml:"app_secret" mapstructure:"app_secret"`
	// EndpointCompression determines, if set, the type of compression the bid request will undergo before being sent to the corresponding bid server
	EndpointCompression string       `yaml:"endpointCompression" mapstructure:"endpointCompression"`
	OpenRTB             *OpenRTBInfo `yaml:"openrtb" mapstructure:"openrtb"`
}

type aliasNillableFields struct {
	Disabled                *bool                 `yaml:"disabled" mapstructure:"disabled"`
	ModifyingVastXmlAllowed *bool                 `yaml:"modifyingVastXmlAllowed" mapstructure:"modifyingVastXmlAllowed"`
	Experiment              *BidderInfoExperiment `yaml:"experiment" mapstructure:"experiment"`
	XAPI                    *AdapterXAPI          `yaml:"xapi" mapstructure:"xapi"`
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
	DOOH *PlatformInfo `yaml:"dooh" mapstructure:"dooh"`
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

// OpenRTBInfo specifies the versions/aspects of openRTB that a bidder supports
// Version is not yet actively supported
// GPPSupported is not yet actively supported
type OpenRTBInfo struct {
	Version              string `yaml:"version" mapstructure:"version"`
	GPPSupported         bool   `yaml:"gpp-supported" mapstructure:"gpp-supported"`
	MultiformatSupported *bool  `yaml:"multiformat-supported" mapstructure:"multiformat-supported"`
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

	// FormatOverride allows a bidder to override their callback type "b" for iframe, "i" for redirect
	FormatOverride string `yaml:"formatOverride" mapstructure:"format_override"`

	// Enabled signifies whether a bidder is enabled/disabled for user sync
	Enabled *bool `yaml:"enabled" mapstructure:"enabled"`

	// SkipWhen allows bidders to specify when they don't want to sync
	SkipWhen *SkipWhen `yaml:"skipwhen" mapstructure:"skipwhen"`
}

type SkipWhen struct {
	GDPR   bool     `yaml:"gdpr" mapstructure:"gdpr"`
	GPPSID []string `yaml:"gpp_sid" mapstructure:"gpp_sid"`
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
//	url: "https://sync.bidderserver.com/usersync?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&gpp={{.GPP}}&gpp_sid={{.GPPSID}}&redirect={{.RedirectURL}}"
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
	//  {{.GPP}}		 - This will be replaced with the "gpp" property sent to /cookie_sync.
	//  {{.GPPSID}}		 - This will be replaced with the "gpp_sid" property sent to /cookie_sync.
	URL string `yaml:"url" mapstructure:"url"`

	// RedirectURL is an endpoint on the host server the user will be redirected to when a user sync
	// request has been completed by the bidder server. The following macros are resolved at application
	// startup:
	//
	//  {{.ExternalURL}} - This will be replaced with the host server's externally reachable http path.
	//  {{.BidderName}}  - This will be replaced with the bidder name.
	//  {{.SyncType}}    - This will be replaced with the sync type, either 'b' for iframe syncs or 'i'
	//                     for redirect/image syncs.
	//  {{.UserMacro}}   - This will be replaced with the bidder server's user id macro.
	//
	// The endpoint on the host server is usually Prebid Server's /setuid endpoint. The default value is:
	// `{{.ExternalURL}}/setuid?bidder={{.SyncerKey}}&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&gpp={{.GPP}}&gpp_sid={{.GPPSID}}&f={{.SyncType}}&uid={{.UserMacro}}`
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

// Defined returns true if at least one field exists, except for the supports field.
func (s *Syncer) Defined() bool {
	if s == nil {
		return false
	}

	return s.Key != "" ||
		s.IFrame != nil ||
		s.Redirect != nil ||
		s.ExternalURL != "" ||
		s.SupportCORS != nil ||
		s.FormatOverride != "" ||
		s.SkipWhen != nil
}

type InfoReader interface {
	Read() (map[string][]byte, error)
}

type InfoReaderFromDisk struct {
	Path string
}

const (
	SyncResponseFormatIFrame   = "b" // b = blank HTML response
	SyncResponseFormatRedirect = "i" // i = image response
)

func (r InfoReaderFromDisk) Read() (map[string][]byte, error) {
	bidderConfigs, err := os.ReadDir(r.Path)
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
		data, err := os.ReadFile(filePath)
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

func processBidderInfos(reader InfoReader, normalizeBidderName openrtb_ext.BidderNameNormalizer) (BidderInfos, error) {
	bidderConfigs, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("error loading bidders data")
	}

	bidderInfos := BidderInfos{}
	aliasNillableFieldsByBidder := map[string]aliasNillableFields{}
	for fileName, data := range bidderConfigs {
		bidderName := strings.Split(fileName, ".")
		if len(bidderName) == 2 && bidderName[1] == "yaml" {
			info := BidderInfo{}
			if err := yaml.Unmarshal(data, &info); err != nil {
				return nil, fmt.Errorf("error parsing config for bidder %s: %v", fileName, err)
			}

			//need to maintain nullable fields from BidderInfo struct into bidderInfoNullableFields
			//to handle the default values in aliases yaml
			if len(info.AliasOf) > 0 {
				aliasFields := aliasNillableFields{}
				if err := yaml.Unmarshal(data, &aliasFields); err != nil {
					return nil, fmt.Errorf("error parsing config for aliased bidder %s: %v", fileName, err)
				}

				//required for CoreBidderNames function to also return aliasBiddernames
				if err := openrtb_ext.SetAliasBidderName(bidderName[0], openrtb_ext.BidderName(info.AliasOf)); err != nil {
					return nil, err
				}

				normalizedBidderName, bidderNameExists := normalizeBidderName(bidderName[0])
				if !bidderNameExists {
					return nil, fmt.Errorf("error parsing config for an alias %s: unknown bidder", fileName)
				}

				aliasNillableFieldsByBidder[string(normalizedBidderName)] = aliasFields
				bidderInfos[string(normalizedBidderName)] = info
			} else {
				normalizedBidderName, bidderNameExists := normalizeBidderName(bidderName[0])
				if !bidderNameExists {
					return nil, fmt.Errorf("error parsing config for bidder %s: unknown bidder", fileName)
				}

				bidderInfos[string(normalizedBidderName)] = info
			}
		}
	}
	return processBidderAliases(aliasNillableFieldsByBidder, bidderInfos)
}

func processBidderAliases(aliasNillableFieldsByBidder map[string]aliasNillableFields, bidderInfos BidderInfos) (BidderInfos, error) {
	for bidderName, alias := range aliasNillableFieldsByBidder {
		aliasBidderInfo, ok := bidderInfos[bidderName]
		if !ok {
			return nil, fmt.Errorf("bidder info not found for an alias: %s", bidderName)
		}
		if err := validateAliases(aliasBidderInfo, bidderInfos, bidderName); err != nil {
			return nil, err
		}

		parentBidderInfo := bidderInfos[aliasBidderInfo.AliasOf]
		if aliasBidderInfo.AppSecret == "" {
			aliasBidderInfo.AppSecret = parentBidderInfo.AppSecret
		}
		if aliasBidderInfo.Capabilities == nil {
			aliasBidderInfo.Capabilities = parentBidderInfo.Capabilities
		}
		if aliasBidderInfo.Debug == nil {
			aliasBidderInfo.Debug = parentBidderInfo.Debug
		}
		if aliasBidderInfo.Endpoint == "" {
			aliasBidderInfo.Endpoint = parentBidderInfo.Endpoint
		}
		if aliasBidderInfo.EndpointCompression == "" {
			aliasBidderInfo.EndpointCompression = parentBidderInfo.EndpointCompression
		}
		if aliasBidderInfo.ExtraAdapterInfo == "" {
			aliasBidderInfo.ExtraAdapterInfo = parentBidderInfo.ExtraAdapterInfo
		}
		if aliasBidderInfo.GVLVendorID == 0 {
			aliasBidderInfo.GVLVendorID = parentBidderInfo.GVLVendorID
		}
		if aliasBidderInfo.Maintainer == nil {
			aliasBidderInfo.Maintainer = parentBidderInfo.Maintainer
		}
		if aliasBidderInfo.OpenRTB == nil {
			aliasBidderInfo.OpenRTB = parentBidderInfo.OpenRTB
		}
		if aliasBidderInfo.PlatformID == "" {
			aliasBidderInfo.PlatformID = parentBidderInfo.PlatformID
		}
		if aliasBidderInfo.Syncer == nil && parentBidderInfo.Syncer.Defined() {
			syncerKey := aliasBidderInfo.AliasOf
			if parentBidderInfo.Syncer.Key != "" {
				syncerKey = parentBidderInfo.Syncer.Key
			}
			syncer := Syncer{Key: syncerKey}
			aliasBidderInfo.Syncer = &syncer
		}
		if alias.Disabled == nil {
			aliasBidderInfo.Disabled = parentBidderInfo.Disabled
		}
		if alias.Experiment == nil {
			aliasBidderInfo.Experiment = parentBidderInfo.Experiment
		}
		if alias.ModifyingVastXmlAllowed == nil {
			aliasBidderInfo.ModifyingVastXmlAllowed = parentBidderInfo.ModifyingVastXmlAllowed
		}
		if alias.XAPI == nil {
			aliasBidderInfo.XAPI = parentBidderInfo.XAPI
		}
		bidderInfos[bidderName] = aliasBidderInfo
	}
	return bidderInfos, nil
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

			if err := validateInfo(bidder, infos, bidderName); err != nil {
				errs = append(errs, err)
			}

			if err := validateSyncer(bidder); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errs
}

func validateAliases(aliasBidderInfo BidderInfo, infos BidderInfos, bidderName string) error {
	if len(aliasBidderInfo.AliasOf) > 0 {
		if parentBidder, ok := infos[aliasBidderInfo.AliasOf]; ok {
			if len(parentBidder.AliasOf) > 0 {
				return fmt.Errorf("bidder: %s cannot be an alias of an alias: %s", aliasBidderInfo.AliasOf, bidderName)
			}
		} else {
			return fmt.Errorf("bidder: %s not found for an alias: %s", aliasBidderInfo.AliasOf, bidderName)
		}
	}
	return nil
}

var testEndpointTemplateParams = macros.EndpointTemplateParams{
	Host:        "anyHost",
	PublisherID: "anyPublisherID",
	AccountID:   "anyAccountID",
	ZoneID:      "anyZoneID",
	SourceId:    "anySourceID",
	AdUnit:      "anyAdUnit",
	MediaType:   "MediaType",
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

func validateInfo(bidder BidderInfo, infos BidderInfos, bidderName string) error {
	if err := validateMaintainer(bidder.Maintainer, bidderName); err != nil {
		return err
	}
	if err := validateCapabilities(bidder.Capabilities, bidderName); err != nil {
		return err
	}
	if len(bidder.AliasOf) > 0 {
		if err := validateAliasCapabilities(bidder, infos, bidderName); err != nil {
			return err
		}
	}
	return nil
}

func validateMaintainer(info *MaintainerInfo, bidderName string) error {
	if info == nil || info.Email == "" {
		return fmt.Errorf("missing required field: maintainer.email for adapter: %s", bidderName)
	}
	return nil
}

func validateAliasCapabilities(aliasBidderInfo BidderInfo, infos BidderInfos, bidderName string) error {
	parentBidder, parentFound := infos[aliasBidderInfo.AliasOf]
	if !parentFound {
		return fmt.Errorf("parent bidder: %s not found for an alias: %s", aliasBidderInfo.AliasOf, bidderName)
	}

	if aliasBidderInfo.Capabilities != nil {
		if parentBidder.Capabilities == nil {
			return fmt.Errorf("capabilities for alias: %s should be a subset of capabilities for parent bidder: %s", bidderName, aliasBidderInfo.AliasOf)
		}

		if (aliasBidderInfo.Capabilities.App != nil && parentBidder.Capabilities.App == nil) ||
			(aliasBidderInfo.Capabilities.Site != nil && parentBidder.Capabilities.Site == nil) ||
			(aliasBidderInfo.Capabilities.DOOH != nil && parentBidder.Capabilities.DOOH == nil) {
			return fmt.Errorf("capabilities for alias: %s should be a subset of capabilities for parent bidder: %s", bidderName, aliasBidderInfo.AliasOf)
		}

		if aliasBidderInfo.Capabilities.Site != nil && parentBidder.Capabilities.Site != nil {
			if err := isAliasPlatformInfoSubsetOfParent(*parentBidder.Capabilities.Site, *aliasBidderInfo.Capabilities.Site, bidderName, aliasBidderInfo.AliasOf); err != nil {
				return err
			}
		}

		if aliasBidderInfo.Capabilities.App != nil && parentBidder.Capabilities.App != nil {
			if err := isAliasPlatformInfoSubsetOfParent(*parentBidder.Capabilities.App, *aliasBidderInfo.Capabilities.App, bidderName, aliasBidderInfo.AliasOf); err != nil {
				return err
			}
		}

		if aliasBidderInfo.Capabilities.DOOH != nil && parentBidder.Capabilities.DOOH != nil {
			if err := isAliasPlatformInfoSubsetOfParent(*parentBidder.Capabilities.DOOH, *aliasBidderInfo.Capabilities.DOOH, bidderName, aliasBidderInfo.AliasOf); err != nil {
				return err
			}
		}
	}

	return nil
}

func isAliasPlatformInfoSubsetOfParent(parentInfo PlatformInfo, aliasInfo PlatformInfo, bidderName string, parentBidderName string) error {
	parentMediaTypes := make(map[openrtb_ext.BidType]struct{})
	for _, info := range parentInfo.MediaTypes {
		parentMediaTypes[info] = struct{}{}
	}

	for _, info := range aliasInfo.MediaTypes {
		if _, found := parentMediaTypes[info]; !found {
			return fmt.Errorf("mediaTypes for alias: %s should be a subset of MediaTypes for parent bidder: %s", bidderName, parentBidderName)
		}
	}

	return nil
}

func validateCapabilities(info *CapabilitiesInfo, bidderName string) error {
	if info == nil {
		return fmt.Errorf("missing required field: capabilities for adapter: %s", bidderName)
	}

	if info.App == nil && info.Site == nil && info.DOOH == nil {
		return fmt.Errorf("at least one of capabilities.site, capabilities.app, or capabilities.dooh must exist for adapter: %s", bidderName)
	}

	if info.App != nil {
		if err := validatePlatformInfo(info.App); err != nil {
			return fmt.Errorf("capabilities.app failed validation: %v for adapter: %s", err, bidderName)
		}
	}

	if info.Site != nil {
		if err := validatePlatformInfo(info.Site); err != nil {
			return fmt.Errorf("capabilities.site failed validation: %v for adapter: %s", err, bidderName)
		}
	}

	if info.DOOH != nil {
		if err := validatePlatformInfo(info.DOOH); err != nil {
			return fmt.Errorf("capabilities.dooh failed validation: %v for adapter: %s", err, bidderName)
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

	if bidderInfo.Syncer.FormatOverride != SyncResponseFormatIFrame && bidderInfo.Syncer.FormatOverride != SyncResponseFormatRedirect && bidderInfo.Syncer.FormatOverride != "" {
		return fmt.Errorf("syncer could not be created, invalid format override value: %s", bidderInfo.Syncer.FormatOverride)
	}

	for _, supports := range bidderInfo.Syncer.Supports {
		if !strings.EqualFold(supports, "iframe") && !strings.EqualFold(supports, "redirect") {
			return fmt.Errorf("syncer could not be created, invalid supported endpoint: %s", supports)
		}
	}

	return nil
}

func applyBidderInfoConfigOverrides(configBidderInfos nillableFieldBidderInfos, fsBidderInfos BidderInfos, normalizeBidderName openrtb_ext.BidderNameNormalizer) (BidderInfos, error) {
	mergedBidderInfos := make(map[string]BidderInfo, len(fsBidderInfos))

	for bidderName, configBidderInfo := range configBidderInfos {
		normalizedBidderName, exists := normalizeBidderName(bidderName)
		if !exists {
			return nil, fmt.Errorf("error setting configuration for bidder %s: unknown bidder", bidderName)
		}
		fsBidderInfo, exists := fsBidderInfos[string(normalizedBidderName)]
		if !exists {
			return nil, fmt.Errorf("error finding configuration for bidder %s: unknown bidder", bidderName)
		}

		mergedBidderInfo := fsBidderInfo
		mergedBidderInfo.Syncer = configBidderInfo.bidderInfo.Syncer.Override(fsBidderInfo.Syncer)
		if len(configBidderInfo.bidderInfo.Endpoint) > 0 {
			mergedBidderInfo.Endpoint = configBidderInfo.bidderInfo.Endpoint
		}
		if len(configBidderInfo.bidderInfo.ExtraAdapterInfo) > 0 {
			mergedBidderInfo.ExtraAdapterInfo = configBidderInfo.bidderInfo.ExtraAdapterInfo
		}
		if configBidderInfo.bidderInfo.Maintainer != nil {
			mergedBidderInfo.Maintainer = configBidderInfo.bidderInfo.Maintainer
		}
		if configBidderInfo.bidderInfo.Capabilities != nil {
			mergedBidderInfo.Capabilities = configBidderInfo.bidderInfo.Capabilities
		}
		if configBidderInfo.bidderInfo.Debug != nil {
			mergedBidderInfo.Debug = configBidderInfo.bidderInfo.Debug
		}
		if configBidderInfo.bidderInfo.GVLVendorID > 0 {
			mergedBidderInfo.GVLVendorID = configBidderInfo.bidderInfo.GVLVendorID
		}
		if configBidderInfo.bidderInfo.XAPI.Username != "" {
			mergedBidderInfo.XAPI.Username = configBidderInfo.bidderInfo.XAPI.Username
		}
		if configBidderInfo.bidderInfo.XAPI.Password != "" {
			mergedBidderInfo.XAPI.Password = configBidderInfo.bidderInfo.XAPI.Password
		}
		if configBidderInfo.bidderInfo.XAPI.Tracker != "" {
			mergedBidderInfo.XAPI.Tracker = configBidderInfo.bidderInfo.XAPI.Tracker
		}
		if configBidderInfo.bidderInfo.PlatformID != "" {
			mergedBidderInfo.PlatformID = configBidderInfo.bidderInfo.PlatformID
		}
		if configBidderInfo.bidderInfo.AppSecret != "" {
			mergedBidderInfo.AppSecret = configBidderInfo.bidderInfo.AppSecret
		}
		if configBidderInfo.nillableFields.Disabled != nil {
			mergedBidderInfo.Disabled = configBidderInfo.bidderInfo.Disabled
		}
		if configBidderInfo.nillableFields.ModifyingVastXmlAllowed != nil {
			mergedBidderInfo.ModifyingVastXmlAllowed = configBidderInfo.bidderInfo.ModifyingVastXmlAllowed
		}
		if configBidderInfo.bidderInfo.Experiment.AdsCert.Enabled {
			mergedBidderInfo.Experiment.AdsCert.Enabled = true
		}
		if configBidderInfo.bidderInfo.EndpointCompression != "" {
			mergedBidderInfo.EndpointCompression = configBidderInfo.bidderInfo.EndpointCompression
		}
		if configBidderInfo.bidderInfo.OpenRTB != nil {
			mergedBidderInfo.OpenRTB = configBidderInfo.bidderInfo.OpenRTB
		}

		mergedBidderInfos[string(normalizedBidderName)] = mergedBidderInfo
	}
	for bidderName, fsBidderInfo := range fsBidderInfos {
		if _, exists := mergedBidderInfos[bidderName]; !exists {
			mergedBidderInfos[bidderName] = fsBidderInfo
		}
	}

	return mergedBidderInfos, nil
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
