package usersync

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"text/template"

	validator "github.com/asaskevich/govalidator"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/privacy"
)

// Syncer represents the user sync configuration for a bidder or a shared set of bidders.
type Syncer interface {
	// Key is the name of the syncer as stored in the user's cookie. This is not necessarily a
	// one-to-one relationship with a bidder.
	Key() string

	// SupportsType returns true if the syncer supports at least one of the specified sync types.
	SupportsType(syncTypes []SyncType) bool

	// GetSync returns a user sync for the user's device to perform, or an error if the none of the
	// sync types are supported or if macro substitution fails.
	GetSync(syncTypes []SyncType, privacyPolicies privacy.Policies) (Sync, error)
}

// Sync represents a user sync for the user's device to perform.
type Sync struct {
	URL         string
	Type        SyncType
	SupportCORS bool
}

type standardSyncer struct {
	key             string
	defaultSyncType SyncType
	iframe          *template.Template
	redirect        *template.Template
	supportCORS     bool
}

const (
	setuidSyncTypeIFrame   = "b"
	setuidSyncTypeRedirect = "i"
)

// NewSyncer creates a new Syncer instance from the provided configuration, or an error if macro
// substition fails or the url specified is invalid.
func NewSyncer(hostConfig config.UserSync, syncerConfig config.Syncer) (Syncer, error) {
	if syncerConfig.IFrame == nil && syncerConfig.Redirect == nil {
		return nil, errors.New("at least one iframe or redirect is required")
	}

	// var defaultSyncType SyncType
	// if syncerConfig.Default == "" {
	// 	// error if more than 1 defined, otherwise choose that one
	// } else {
	// 	// parse. verify it's defined
	// }

	syncer := standardSyncer{
		key:         syncerConfig.Key,
		supportCORS: syncerConfig.SupportCORS,
	}

	// todo: default sync

	if syncerConfig.IFrame != nil {
		var err error
		syncer.iframe, err = composeTemplate(syncerConfig.Key, setuidSyncTypeIFrame, hostConfig, *syncerConfig.IFrame)
		if err != nil {
			return nil, fmt.Errorf("iframe: %v", err)
		}
		if err := validateTemplate(syncer.iframe); err != nil {
			return nil, fmt.Errorf("iframe: %v", err)
		}
	}

	if syncerConfig.Redirect != nil {
		var err error
		syncer.redirect, err = composeTemplate(syncerConfig.Key, setuidSyncTypeRedirect, hostConfig, *syncerConfig.Redirect)
		if err != nil {
			return nil, fmt.Errorf("redirect: %v", err)
		}
		if err := validateTemplate(syncer.redirect); err != nil {
			return nil, fmt.Errorf("redirect: %v", err)
		}
	}

	return syncer, nil
}

var (
	externalHostRegex = regexp.MustCompile(`{{\s*.ExternalURL\s*}}`)
	syncerKeyRegex    = regexp.MustCompile(`{{\s*.SyncerKey\s*}}`)
	syncTypeRegex     = regexp.MustCompile(`{{\s*.SyncType\s*}}`)
	userMacroRegex    = regexp.MustCompile(`{{\s*.UserMacro\s*}}`)
	redirectRegex     = regexp.MustCompile(`{{\s*.RedirectURL\s*}}`)
	macroRegex        = regexp.MustCompile(`{{.*?}}`)
)

func composeTemplate(key, syncTypeValue string, hostConfig config.UserSync, syncerEndpoint config.SyncerEndpoint) (*template.Template, error) {
	redirectTemplate := syncerEndpoint.RedirectURL
	if redirectTemplate == "" {
		redirectTemplate = hostConfig.RedirectURL
	}

	externalURL := syncerEndpoint.ExternalURL
	if externalURL == "" {
		externalURL = hostConfig.ExternalURL
	}

	redirectURL := externalHostRegex.ReplaceAllLiteralString(redirectTemplate, externalURL)
	redirectURL = syncerKeyRegex.ReplaceAllLiteralString(redirectURL, key)
	redirectURL = syncTypeRegex.ReplaceAllLiteralString(redirectURL, syncTypeValue)
	redirectURL = userMacroRegex.ReplaceAllLiteralString(redirectURL, syncerEndpoint.UserMacro)
	redirectURL = escapeTemplate(redirectURL)

	url := redirectRegex.ReplaceAllString(syncerEndpoint.URL, redirectURL)

	templateName := strings.ToLower(key) + "_usersync_url"
	return template.New(templateName).Parse(url)
}

func escapeTemplate(x string) string {
	escaped := strings.Builder{}

	i := 0
	for _, m := range macroRegex.FindAllStringIndex(x, -1) {
		escaped.WriteString(url.QueryEscape(x[i:m[0]]))
		escaped.WriteString(x[m[0]:m[1]])
		i = m[1]
	}
	escaped.WriteString(url.QueryEscape(x[i:]))

	return escaped.String()
}

func validateTemplate(template *template.Template) error {
	testValues := macros.UserSyncTemplateParams{
		GDPR:        "anyGDPR",
		GDPRConsent: "anyGDPRConsent",
		USPrivacy:   "anyCCPAConsent",
	}

	url, err := macros.ResolveMacros(template, testValues)
	if err != nil {
		return err
	}

	if !validator.IsURL(url) || !validator.IsRequestURL(url) {
		return fmt.Errorf("composed url \"%s\" is invalid", url)
	}

	return nil
}

func (s standardSyncer) Key() string {
	return s.key
}

func (s standardSyncer) SupportsType(syncTypes []SyncType) bool {
	supported := s.filterSupportedSyncTypes(syncTypes)
	return len(supported) > 0
}

func (s standardSyncer) filterSupportedSyncTypes(syncTypes []SyncType) []SyncType {
	supported := make([]SyncType, 0, len(syncTypes))
	for _, syncType := range syncTypes {
		switch syncType {
		case SyncTypeIFrame:
			if s.iframe != nil {
				supported = append(supported, SyncTypeIFrame)
			}
		case SyncTypeRedirect:
			if s.redirect != nil {
				supported = append(supported, SyncTypeRedirect)
			}
		}
	}
	return supported
}

func (s standardSyncer) GetSync(syncTypes []SyncType, privacyPolicies privacy.Policies) (Sync, error) {
	syncType, err := s.chooseSyncType(syncTypes)
	if err != nil {
		return Sync{}, err
	}

	syncTemplate := s.chooseTemplate(syncType)

	url, err := macros.ResolveMacros(syncTemplate, macros.UserSyncTemplateParams{
		GDPR:        privacyPolicies.GDPR.Signal,
		GDPRConsent: privacyPolicies.GDPR.Consent,
		USPrivacy:   privacyPolicies.CCPA.Consent,
	})
	if err != nil {
		return Sync{}, err
	}

	sync := Sync{
		URL:         url,
		Type:        syncType,
		SupportCORS: s.supportCORS,
	}
	return sync, nil
}

func (s standardSyncer) chooseSyncType(syncTypes []SyncType) (SyncType, error) {
	if len(syncTypes) == 0 {
		return SyncTypeUnknown, errors.New("no sync types provided")
	}

	supported := s.filterSupportedSyncTypes(syncTypes)
	if len(supported) == 0 {
		return SyncTypeUnknown, errors.New("no sync types supported")
	}

	// prefer default type
	for _, syncType := range supported {
		if syncType == s.defaultSyncType {
			return syncType, nil
		}
	}

	return syncTypes[0], nil
}

func (s standardSyncer) chooseTemplate(syncType SyncType) *template.Template {
	switch syncType {
	case SyncTypeIFrame:
		return s.iframe
	case SyncTypeRedirect:
		return s.redirect
	default:
		return nil
	}
}
