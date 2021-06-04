package config

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/prebid/prebid-server/openrtb_ext"
	"gopkg.in/yaml.v2"
)

// BidderInfos contains a mapping of bidder name to bidder info.
type BidderInfos map[string]BidderInfo

// BidderInfo specifies all configuration for a bidder except for enabled status, endpoint, and extra information.
type BidderInfo struct {
	Enabled                 bool              // copied from adapter config for convenience. to be refactored.
	Maintainer              *MaintainerInfo   `yaml:"maintainer"`
	Capabilities            *CapabilitiesInfo `yaml:"capabilities"`
	ModifyingVastXmlAllowed bool              `yaml:"modifyingVastXmlAllowed"`
	Debug                   *DebugInfo        `yaml:"debug"`
	GVLVendorID             uint16            `yaml:"gvlVendorID"`
	Syncer                  *Syncer           `yaml:"userSync"`
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

// Syncer specifies the user sync settings for a bidder. This struct is shared by the account config,
// so it needs to have both yaml and mapstructure mappings.
type Syncer struct {
	// Key is used as the record key for the user sync cookie. We recommend using the bidder name
	// as the key for consistency, but that is not enforced as a requirement. Each bidder must
	// have a unique key.
	Key string `yaml:"key" mapstructure:"key"`

	// Default identifies which endpoint is preferred if both are allowed by the publisher. This is
	// only required if there is more than one endpoint configured for the bidder. Valid values are
	// `iframe` and `redirect`.
	Default string `yaml:"default" mapstructure:"default"`

	// IFrame configures an iframe endpoint for user syncing.
	IFrame *SyncerEndpoint `yaml:"iframe" mapstructure:"iframe"`

	// Redirect configures an redirect endpoint for user syncing. This is also known as an image
	// endpoint in the Prebid.js project.
	Redirect *SyncerEndpoint `yaml:"redirect" mapstructure:"redirect"`

	// SupportCORS identifies if CORS is supported for user syncing.
	SupportCORS *bool `yaml:"supportCors" mapstructure:"support_cors"`
}

func (s *Syncer) ApplyTo(v *Syncer) *Syncer {
	if s.Empty() {
		return v
	}

	if v == nil {
		v = &Syncer{}
	}

	if s.Key != "" {
		v.Key = s.Key
	}

	if s.Default != "" {
		v.Key = s.Default
	}

	v.IFrame = s.IFrame.ApplyTo(v.IFrame)

	v.Redirect = s.IFrame.ApplyTo(v.Redirect)

	if s.SupportCORS != nil {
		v.SupportCORS = s.SupportCORS
	}

	return v
}

func (s *Syncer) Empty() bool {
	if s == nil {
		return true
	}
	return s.Key == "" && s.Default == "" && s.IFrame.Empty() && s.Redirect.Empty() && s.SupportCORS == nil
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

	// ExternalURL is available as a macro to the RedirectURL template. If not specified, the host configuration
	// value is used.
	ExternalURL string `yaml:"externalUrl" mapstructure:"external_url"`

	// UserMacro is available as a macro to the RedirectURL template. This value is specific to the bidder server
	// and has no default.
	UserMacro string `yaml:"userMacro" mapstructure:"user_macro"`
}

func (s *SyncerEndpoint) ApplyTo(v *SyncerEndpoint) *SyncerEndpoint {
	if s.Empty() {
		return v
	}

	if v == nil {
		v = &SyncerEndpoint{}
	}

	if s.URL != "" {
		v.URL = s.URL
	}

	if s.RedirectURL != "" {
		v.RedirectURL = s.RedirectURL
	}

	if s.ExternalURL != "" {
		v.ExternalURL = s.ExternalURL
	}

	if s.UserMacro != "" {
		v.UserMacro = s.UserMacro
	}

	return v
}

func (s *SyncerEndpoint) Empty() bool {
	if s == nil {
		return true
	}
	return s.URL == "" && s.RedirectURL == "" && s.ExternalURL == "" && s.UserMacro == ""
}

// LoadBidderInfoFromDisk parses all static/bidder-info/{bidder}.yaml files from the file system.
func LoadBidderInfoFromDisk(path string, adapterConfigs map[string]Adapter, bidders []string) (BidderInfos, error) {
	reader := infoReaderFromDisk{path}
	return loadBidderInfo(reader, adapterConfigs, bidders)
}

func loadBidderInfo(r infoReader, adapterConfigs map[string]Adapter, bidders []string) (BidderInfos, error) {
	infos := BidderInfos{}

	for _, bidder := range bidders {
		data, err := r.Read(bidder)
		if err != nil {
			return nil, err
		}

		info := BidderInfo{}
		if err := yaml.Unmarshal(data, &info); err != nil {
			return nil, fmt.Errorf("error parsing yaml for bidder %s: %v", bidder, err)
		}

		info.Enabled = isEnabledByConfig(adapterConfigs, bidder)
		infos[bidder] = info
	}

	return infos, nil
}

func isEnabledByConfig(adapterConfigs map[string]Adapter, bidderName string) bool {
	a, ok := adapterConfigs[strings.ToLower(bidderName)]
	return ok && !a.Disabled
}

type infoReader interface {
	Read(bidder string) ([]byte, error)
}

type infoReaderFromDisk struct {
	path string
}

func (r infoReaderFromDisk) Read(bidder string) ([]byte, error) {
	path := fmt.Sprintf("%v/%v.yaml", r.path, bidder)
	return ioutil.ReadFile(path)
}

// ToGVLVendorIDMap transforms a BidderInfos object to a map of bidder names to GVL id. Disabled
// bidders are omitted from the result.
func (infos BidderInfos) ToGVLVendorIDMap() map[openrtb_ext.BidderName]uint16 {
	m := make(map[openrtb_ext.BidderName]uint16, len(infos))
	for name, info := range infos {
		if info.Enabled && info.GVLVendorID != 0 {
			m[openrtb_ext.BidderName(name)] = info.GVLVendorID
		}
	}
	return m
}
