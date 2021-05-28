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
	URL          string
	Type         SyncType
	SupportsCORS bool
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
	syncer := standardSyncer{
		key: syncerConfig.Key,
	}

	if syncerConfig.IFrame != nil {
		var err error
		syncer.iframe, err = composeTemplate(syncerConfig.Key, setuidSyncTypeIFrame, hostConfig, *syncerConfig.IFrame)
		if err != nil {
			return nil, err
		}
	}

	if syncerConfig.Redirect != nil {
		var err error
		syncer.redirect, err = composeTemplate(syncerConfig.Key, setuidSyncTypeRedirect, hostConfig, *syncerConfig.Redirect)
		if err != nil {
			return nil, err
		}
	}

	return syncer, nil
}

var externalHostRegex = regexp.MustCompile(`{{\s*.ExternalURL\s*}}`)
var syncerKeyRegex = regexp.MustCompile(`{{\s*.SyncerKey\s*}}`)
var syncTypeRegex = regexp.MustCompile(`{{\s*.SyncType\s*}}`)
var userMacroRegex = regexp.MustCompile(`{{\s*.UserMacro\s*}}`)
var redirectRegex = regexp.MustCompile(`{{\s*.RedirectURL\s*}}`)

func composeTemplate(key, syncTypeValue string, hostConfig config.UserSync, syncerEndpoint config.SyncerEndpoint) (*template.Template, error) {
	redirectTemplate := syncerEndpoint.RedirectURL
	if redirectTemplate == "" {
		redirectTemplate = hostConfig.RedirectURL
	}

	externalURL := syncerEndpoint.ExternalURL
	if externalURL == "" {
		externalURL = hostConfig.ExternalURL
	}

	redirectURL := externalHostRegex.ReplaceAllString(redirectTemplate, externalURL)
	redirectURL = syncerKeyRegex.ReplaceAllString(redirectURL, key)
	redirectURL = syncTypeRegex.ReplaceAllString(redirectURL, syncTypeValue)
	redirectURL = userMacroRegex.ReplaceAllString(redirectURL, syncerEndpoint.UserMacro)
	redirectURL = url.PathEscape(redirectURL)

	url := redirectRegex.ReplaceAllString(externalURL, redirectURL)

	templateName := strings.ToLower(key) + "_usersync_url"
	template, err := template.New(templateName).Parse(url)
	if err != nil {
		return nil, err
	}

	if err := validateTemplate(template); err != nil {
		return nil, err
	}

	return template, nil
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
		return fmt.Errorf("composed url %s is invalid", url)
	}

	return nil
}

func (s standardSyncer) Key() string {
	return s.key
}

func (s standardSyncer) SupportsType(syncTypes []SyncType) bool {
	for _, syncType := range syncTypes {
		switch syncType {
		case SyncTypeIFrame:
			if s.iframe != nil {
				return true
			}
		case SyncTypeRedirect:
			if s.redirect != nil {
				return true
			}
		}
	}
	return false
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
		URL:          url,
		Type:         syncType,
		SupportsCORS: s.supportCORS,
	}
	return sync, nil
}

func (s standardSyncer) chooseSyncType(syncTypes []SyncType) (SyncType, error) {
	if len(syncTypes) == 0 {
		return SyncTypeUnknown, errors.New("no sync types provided")
	}

	for _, syncType := range syncTypes {
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
