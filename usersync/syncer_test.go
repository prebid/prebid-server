package usersync

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestNewSyncer(t *testing.T) {
	var (
		supportCORS      = true
		hostConfig       = config.UserSync{ExternalURL: "http://host.com", RedirectURL: "{{.ExternalURL}}/host"}
		macroValues      = macros.UserSyncTemplateParams{GDPR: "A", GDPRConsent: "B", USPrivacy: "C"}
		iframeConfig     = &config.SyncerEndpoint{URL: "https://bidder.com/iframe?redirect={{.RedirectURL}}"}
		redirectConfig   = &config.SyncerEndpoint{URL: "https://bidder.com/redirect?redirect={{.RedirectURL}}"}
		errParseConfig   = &config.SyncerEndpoint{URL: "{{malformed}}"}
		errInvalidConfig = &config.SyncerEndpoint{URL: "notAURL:{{.RedirectURL}}"}
	)

	testCases := []struct {
		description         string
		givenKey            string
		givenDefault        string
		givenIFrameConfig   *config.SyncerEndpoint
		givenRedirectConfig *config.SyncerEndpoint
		expectedError       string
		expectedDefault     SyncType
		expectedIFrame      string
		expectedRedirect    string
	}{
		{
			description:         "Missing Key",
			givenKey:            "",
			givenDefault:        "iframe",
			givenIFrameConfig:   iframeConfig,
			givenRedirectConfig: nil,
			expectedError:       "key is required",
		},
		{
			description:         "Missing Endpoints",
			givenKey:            "a",
			givenDefault:        "",
			givenIFrameConfig:   nil,
			givenRedirectConfig: nil,
			expectedError:       "at least one endpoint (iframe and/or redirect) is required",
		},
		{
			description:         "Resolve Default Sync Type Error ",
			givenKey:            "a",
			givenDefault:        "",
			givenIFrameConfig:   iframeConfig,
			givenRedirectConfig: redirectConfig,
			expectedError:       "default sync type is required when more then one sync endpoint is configured",
		},
		{
			description:         "IFrame & Redirect Endpoints",
			givenKey:            "a",
			givenDefault:        "iframe",
			givenIFrameConfig:   iframeConfig,
			givenRedirectConfig: redirectConfig,
			expectedDefault:     SyncTypeIFrame,
			expectedIFrame:      "https://bidder.com/iframe?redirect=http%3A%2F%2Fhost.com%2Fhost",
			expectedRedirect:    "https://bidder.com/redirect?redirect=http%3A%2F%2Fhost.com%2Fhost",
		},
		{
			description:         "IFrame - Parse Error",
			givenKey:            "a",
			givenDefault:        "iframe",
			givenIFrameConfig:   errParseConfig,
			givenRedirectConfig: nil,
			expectedError:       "iframe template: a_usersync_url:1: function \"malformed\" not defined",
		},
		{
			description:         "IFrame - Validation Error",
			givenKey:            "a",
			givenDefault:        "iframe",
			givenIFrameConfig:   errInvalidConfig,
			givenRedirectConfig: nil,
			expectedError:       "iframe composed url: \"notAURL:http%3A%2F%2Fhost.com%2Fhost\" is invalid",
		},
		{
			description:         "Redirect - Parse Error",
			givenKey:            "a",
			givenDefault:        "redirect",
			givenIFrameConfig:   nil,
			givenRedirectConfig: errParseConfig,
			expectedError:       "redirect template: a_usersync_url:1: function \"malformed\" not defined",
		},
		{
			description:         "Redirect - Validation Error",
			givenKey:            "a",
			givenDefault:        "redirect",
			givenIFrameConfig:   nil,
			givenRedirectConfig: errInvalidConfig,
			expectedError:       "redirect composed url: \"notAURL:http%3A%2F%2Fhost.com%2Fhost\" is invalid",
		},
	}

	for _, test := range testCases {
		syncerConfig := config.Syncer{
			Key:         test.givenKey,
			SupportCORS: &supportCORS,
			Default:     test.givenDefault,
			IFrame:      test.givenIFrameConfig,
			Redirect:    test.givenRedirectConfig,
		}

		result, err := NewSyncer(hostConfig, syncerConfig)

		if test.expectedError == "" {
			assert.NoError(t, err, test.description+":err")
			if assert.IsType(t, standardSyncer{}, result, test.description+":result_type") {
				result := result.(standardSyncer)
				assert.Equal(t, test.givenKey, result.key, test.description+":key")
				assert.Equal(t, supportCORS, result.supportCORS, test.description+":cors")
				assert.Equal(t, test.expectedDefault, result.defaultSyncType, test.description+":default_sync")

				if test.expectedIFrame == "" {
					assert.Nil(t, result.iframe, test.description+":iframe")
				} else {
					iframeRendered, err := macros.ResolveMacros(result.iframe, macroValues)
					if assert.NoError(t, err, test.description+":iframe_render") {
						assert.Equal(t, test.expectedIFrame, iframeRendered, test.description+":iframe")
					}
				}

				if test.expectedRedirect == "" {
					assert.Nil(t, result.redirect, test.description+":redirect")
				} else {
					redirectRendered, err := macros.ResolveMacros(result.redirect, macroValues)
					if assert.NoError(t, err, test.description+":redirect_render") {
						assert.Equal(t, test.expectedRedirect, redirectRendered, test.description+":redirect")
					}
				}
			}
		} else {
			assert.EqualError(t, err, test.expectedError, test.description+":err")
			assert.Empty(t, result)
		}
	}
}

func TestResolveDefaultSyncType(t *testing.T) {
	anyEndpoint := &config.SyncerEndpoint{}

	testCases := []struct {
		description      string
		givenConfig      config.Syncer
		expectedSyncType SyncType
		expectedError    string
	}{
		{
			description:      "IFrame & Redirect - IFrame Default",
			givenConfig:      config.Syncer{Default: "iframe", IFrame: anyEndpoint, Redirect: anyEndpoint},
			expectedSyncType: SyncTypeIFrame,
		},
		{
			description:      "IFrame & Redirect - Redirect Default",
			givenConfig:      config.Syncer{Default: "redirect", IFrame: anyEndpoint, Redirect: anyEndpoint},
			expectedSyncType: SyncTypeRedirect,
		},
		{
			description:      "IFrame & Redirect - No Default",
			givenConfig:      config.Syncer{Default: "", IFrame: anyEndpoint, Redirect: anyEndpoint},
			expectedSyncType: SyncTypeUnknown,
			expectedError:    "default sync type is required when more then one sync endpoint is configured",
		},
		{
			description:      "IFrame & Redirect - Invalid Default",
			givenConfig:      config.Syncer{Default: "invalid", IFrame: anyEndpoint, Redirect: anyEndpoint},
			expectedSyncType: SyncTypeUnknown,
			expectedError:    "invalid default sync type 'invalid'",
		},
		{
			description:      "IFrame Only - IFrame Default",
			givenConfig:      config.Syncer{Default: "iframe", IFrame: anyEndpoint, Redirect: nil},
			expectedSyncType: SyncTypeIFrame,
		},
		{
			description:      "IFrame Only - No Default",
			givenConfig:      config.Syncer{Default: "", IFrame: anyEndpoint, Redirect: nil},
			expectedSyncType: SyncTypeIFrame,
		},
		{
			description:      "IFrame Only - Redirect Default",
			givenConfig:      config.Syncer{Default: "redirect", IFrame: anyEndpoint, Redirect: nil},
			expectedSyncType: SyncTypeUnknown,
			expectedError:    "default is set to redirect but no redirect endpoint is configured",
		},
		{
			description:      "Redirect Only - Redirect Default",
			givenConfig:      config.Syncer{Default: "redirect", IFrame: nil, Redirect: anyEndpoint},
			expectedSyncType: SyncTypeRedirect,
		},
		{
			description:      "IFrame Only - No Default",
			givenConfig:      config.Syncer{Default: "", IFrame: nil, Redirect: anyEndpoint},
			expectedSyncType: SyncTypeRedirect,
		},
		{
			description:      "IFrame Only - IFrame Default",
			givenConfig:      config.Syncer{Default: "iframe", IFrame: nil, Redirect: anyEndpoint},
			expectedSyncType: SyncTypeUnknown,
			expectedError:    "default is set to iframe but no iframe endpoint is configured",
		},
	}

	for _, test := range testCases {
		result, err := resolveDefaultSyncType(test.givenConfig)

		assert.Equal(t, test.expectedSyncType, result, test.description+":result")
		if test.expectedError == "" {
			assert.NoError(t, err, test.description+":err")
		} else {
			assert.EqualError(t, err, test.expectedError, test.description+":err")
		}
	}
}

func TestBuildTemplate(t *testing.T) {
	var (
		key           = "anyKey"
		syncTypeValue = "x"
		hostConfig    = config.UserSync{ExternalURL: "http://host.com", RedirectURL: "{{.ExternalURL}}/host"}
		macroValues   = macros.UserSyncTemplateParams{GDPR: "A", GDPRConsent: "B", USPrivacy: "C"}
	)

	testCases := []struct {
		description         string
		givenHostConfig     config.UserSync
		givenSyncerEndpoint config.SyncerEndpoint
		expectedError       string
		expectedRendered    string
	}{
		{
			description: "No Composed Macros",
			givenSyncerEndpoint: config.SyncerEndpoint{
				URL: "hasNoComposedMacros,gdpr={{.GDPR}}",
			},
			expectedRendered: "hasNoComposedMacros,gdpr=A",
		},
		{
			description: "All Composed Macros",
			givenSyncerEndpoint: config.SyncerEndpoint{
				URL:         "https://bidder.com/sync?redirect={{.RedirectURL}}",
				RedirectURL: "{{.ExternalURL}}/setuid?bidder={{.SyncerKey}}&f={{.SyncType}}&gdpr={{.GDPR}}&uid={{.UserMacro}}",
				ExternalURL: "http://syncer.com",
				UserMacro:   "$UID$",
			},
			expectedRendered: "https://bidder.com/sync?redirect=http%3A%2F%2Fsyncer.com%2Fsetuid%3Fbidder%3DanyKey%26f%3Dx%26gdpr%3DA%26uid%3D%24UID%24",
		},
		{
			description: "Redirect URL + External URL From Host",
			givenSyncerEndpoint: config.SyncerEndpoint{
				URL: "https://bidder.com/sync?redirect={{.RedirectURL}}",
			},
			expectedRendered: "https://bidder.com/sync?redirect=http%3A%2F%2Fhost.com%2Fhost",
		},
		{
			description: "Redirect URL From Syncer",
			givenSyncerEndpoint: config.SyncerEndpoint{
				URL:         "https://bidder.com/sync?redirect={{.RedirectURL}}",
				RedirectURL: "{{.ExternalURL}}/syncer",
			},
			expectedRendered: "https://bidder.com/sync?redirect=http%3A%2F%2Fhost.com%2Fsyncer",
		},
		{
			description: "External URL From Host",
			givenSyncerEndpoint: config.SyncerEndpoint{
				URL:         "https://bidder.com/sync?redirect={{.RedirectURL}}",
				ExternalURL: "http://syncer.com",
			},
			expectedRendered: "https://bidder.com/sync?redirect=http%3A%2F%2Fsyncer.com%2Fhost",
		},
		{
			description: "Template Parse Error",
			givenSyncerEndpoint: config.SyncerEndpoint{
				URL: "{{malformed}}",
			},
			expectedError: "template: anykey_usersync_url:1: function \"malformed\" not defined",
		},
		{
			description: "User Macro Is Go Template Macro-Like",
			givenSyncerEndpoint: config.SyncerEndpoint{
				URL:         "https://bidder.com/sync?redirect={{.RedirectURL}}",
				RedirectURL: "{{.ExternalURL}}/setuid?bidder={{.SyncerKey}}&f={{.SyncType}}&gdpr={{.GDPR}}&uid={{.UserMacro}}",
				UserMacro:   "{{UID}}",
			},
			expectedRendered: "https://bidder.com/sync?redirect=http%3A%2F%2Fhost.com%2Fsetuid%3Fbidder%3DanyKey%26f%3Dx%26gdpr%3DA%26uid%3D%7B%7BUID%7D%7D",
		},

		// The following tests protect against the "\"." literal character vs the "." character class in regex. Literal
		// value which use {{ }} but do not match Go's naming pattern of {{ .Name }} are escaped.
		{
			description: "Invalid Macro - Redirect URL",
			givenSyncerEndpoint: config.SyncerEndpoint{
				URL: "https://bidder.com/sync?redirect={{xRedirectURL}}",
			},
			expectedError: "template: anykey_usersync_url:1: function \"xRedirectURL\" not defined",
		},
		{
			description: "Macro-Like Literal Value - External URL",
			givenSyncerEndpoint: config.SyncerEndpoint{
				URL:         "https://bidder.com/sync?redirect={{.RedirectURL}}",
				RedirectURL: "{{xExternalURL}}",
			},
			expectedRendered: "https://bidder.com/sync?redirect=%7B%7BxExternalURL%7D%7D",
		},
		{
			description: "Macro-Like Literal Value - Syncer Key",
			givenSyncerEndpoint: config.SyncerEndpoint{
				URL:         "https://bidder.com/sync?redirect={{.RedirectURL}}",
				RedirectURL: "{{xSyncerKey}}",
			},
			expectedRendered: "https://bidder.com/sync?redirect=%7B%7BxSyncerKey%7D%7D",
		},
		{
			description: "Macro-Like Literal Value - Sync Type",
			givenSyncerEndpoint: config.SyncerEndpoint{
				URL:         "https://bidder.com/sync?redirect={{.RedirectURL}}",
				RedirectURL: "{{xSyncType}}",
			},
			expectedRendered: "https://bidder.com/sync?redirect=%7B%7BxSyncType%7D%7D",
		},
		{
			description: "Macro-Like Literal Value - User Macro",
			givenSyncerEndpoint: config.SyncerEndpoint{
				URL:         "https://bidder.com/sync?redirect={{.RedirectURL}}",
				RedirectURL: "{{xUserMacro}}",
			},
			expectedRendered: "https://bidder.com/sync?redirect=%7B%7BxUserMacro%7D%7D",
		},
	}

	for _, test := range testCases {
		result, err := buildTemplate(key, syncTypeValue, hostConfig, test.givenSyncerEndpoint)

		if test.expectedError == "" {
			assert.NoError(t, err, test.description+":err")
			resultRendered, err := macros.ResolveMacros(result, macroValues)
			if assert.NoError(t, err, test.description+":template_render") {
				assert.Equal(t, test.expectedRendered, resultRendered, test.description+":template")
			}
		} else {
			assert.EqualError(t, err, test.expectedError, test.description+":err")
		}
	}
}

func TestEscapeTemplate(t *testing.T) {
	testCases := []struct {
		description string
		given       string
		expected    string
	}{
		{
			description: "Macro Only",
			given:       "{{.Macro}}",
			expected:    "{{.Macro}}",
		},
		{
			description: "Text Only",
			given:       "/a",
			expected:    "%2Fa",
		},
		{
			description: "Start Only",
			given:       "&a{{.Macro1}}",
			expected:    "%26a{{.Macro1}}",
		},
		{
			description: "Middle Only",
			given:       "{{.Macro1}}&a{{.Macro2}}",
			expected:    "{{.Macro1}}%26a{{.Macro2}}",
		},
		{
			description: "End Only",
			given:       "{{.Macro1}}&a",
			expected:    "{{.Macro1}}%26a",
		},
		{
			description: "Start / Middle / End",
			given:       "&a{{.Macro1}}/b{{.Macro2}}&c",
			expected:    "%26a{{.Macro1}}%2Fb{{.Macro2}}%26c",
		},
		{
			description: "Characters In Macro Not Escaped",
			given:       "{{.Macro&}}",
			expected:    "{{.Macro&}}",
		},
		{
			description: "Macro Whitespace Insensitive",
			given:       " &a {{ .Macro1  }} /b ",
			expected:    "+%26a+{{ .Macro1  }}+%2Fb+",
		},
		{
			description: "Double Curly Braces, But Not Macro",
			given:       "{{Macro}}",
			expected:    "%7B%7BMacro%7D%7D",
		},
	}

	for _, test := range testCases {
		result := escapeTemplate(test.given)
		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestValidateTemplate(t *testing.T) {
	testCases := []struct {
		description   string
		given         *template.Template
		expectedError string
	}{
		{
			description:   "Contains Unrecognized Macro",
			given:         template.Must(template.New("test").Parse("invalid:{{.DoesNotExist}}")),
			expectedError: "template: test:1:10: executing \"test\" at <.DoesNotExist>: can't evaluate field DoesNotExist in type macros.UserSyncTemplateParams",
		},
		{
			description:   "Not A Url",
			given:         template.Must(template.New("test").Parse("not-a-url,gdpr:{{.GDPR}},gdprconsent:{{.GDPRConsent}},ccpa:{{.USPrivacy}}")),
			expectedError: "composed url: \"not-a-url,gdpr:anyGDPR,gdprconsent:anyGDPRConsent,ccpa:anyCCPAConsent\" is invalid",
		},
		{
			description:   "Valid",
			given:         template.Must(template.New("test").Parse("http://server.com/sync?gdpr={{.GDPR}}&gdprconsent={{.GDPRConsent}}&ccpa={{.USPrivacy}}")),
			expectedError: "",
		},
	}

	for _, test := range testCases {
		err := validateTemplate(test.given)

		if test.expectedError == "" {
			assert.NoError(t, err, test.description)
		} else {
			assert.EqualError(t, err, test.expectedError, test.description)
		}
	}
}

func TestSyncerKey(t *testing.T) {
	syncer := standardSyncer{key: "a"}
	assert.Equal(t, "a", syncer.Key())
}

func TestSyncerDefaultSyncType(t *testing.T) {
	syncer := standardSyncer{defaultSyncType: SyncTypeRedirect}
	assert.Equal(t, SyncTypeRedirect, syncer.DefaultSyncType())
}

func TestSyncerSupportsType(t *testing.T) {
	endpointTemplate := template.Must(template.New("test").Parse("iframe"))

	testCases := []struct {
		description           string
		givenSyncTypes        []SyncType
		givenIFrameTemplate   *template.Template
		givenRedirectTemplate *template.Template
		expectedResult        bool
	}{
		{
			description:           "All Available - None",
			givenSyncTypes:        []SyncType{},
			givenIFrameTemplate:   endpointTemplate,
			givenRedirectTemplate: endpointTemplate,
			expectedResult:        false,
		},
		{
			description:           "All Available - One",
			givenSyncTypes:        []SyncType{SyncTypeIFrame},
			givenIFrameTemplate:   endpointTemplate,
			givenRedirectTemplate: endpointTemplate,
			expectedResult:        true,
		},
		{
			description:           "All Available - Many",
			givenSyncTypes:        []SyncType{SyncTypeIFrame, SyncTypeRedirect},
			givenIFrameTemplate:   endpointTemplate,
			givenRedirectTemplate: endpointTemplate,
			expectedResult:        true,
		},
		{
			description:           "One Available - None",
			givenSyncTypes:        []SyncType{},
			givenIFrameTemplate:   endpointTemplate,
			givenRedirectTemplate: nil,
			expectedResult:        false,
		},
		{
			description:           "One Available - One - Supported",
			givenSyncTypes:        []SyncType{SyncTypeIFrame},
			givenIFrameTemplate:   endpointTemplate,
			givenRedirectTemplate: nil,
			expectedResult:        true,
		},
		{
			description:           "One Available - One - Not Supported",
			givenSyncTypes:        []SyncType{SyncTypeRedirect},
			givenIFrameTemplate:   endpointTemplate,
			givenRedirectTemplate: nil,
			expectedResult:        false,
		},
		{
			description:           "One Available - Many",
			givenSyncTypes:        []SyncType{SyncTypeIFrame, SyncTypeRedirect},
			givenIFrameTemplate:   endpointTemplate,
			givenRedirectTemplate: nil,
			expectedResult:        true,
		},
	}

	for _, test := range testCases {
		syncer := standardSyncer{
			iframe:   test.givenIFrameTemplate,
			redirect: test.givenRedirectTemplate,
		}
		result := syncer.SupportsType(test.givenSyncTypes)
		assert.Equal(t, test.expectedResult, result, test.description)
	}
}

func TestSyncerFilterSupportedSyncTypes(t *testing.T) {
	endpointTemplate := template.Must(template.New("test").Parse("iframe"))

	testCases := []struct {
		description           string
		givenSyncTypes        []SyncType
		givenIFrameTemplate   *template.Template
		givenRedirectTemplate *template.Template
		expectedSyncTypes     []SyncType
	}{
		{
			description:           "All Available - None",
			givenSyncTypes:        []SyncType{},
			givenIFrameTemplate:   endpointTemplate,
			givenRedirectTemplate: endpointTemplate,
			expectedSyncTypes:     []SyncType{},
		},
		{
			description:           "All Available - One",
			givenSyncTypes:        []SyncType{SyncTypeIFrame},
			givenIFrameTemplate:   endpointTemplate,
			givenRedirectTemplate: endpointTemplate,
			expectedSyncTypes:     []SyncType{SyncTypeIFrame},
		},
		{
			description:           "All Available - Many",
			givenSyncTypes:        []SyncType{SyncTypeIFrame, SyncTypeRedirect},
			givenIFrameTemplate:   endpointTemplate,
			givenRedirectTemplate: endpointTemplate,
			expectedSyncTypes:     []SyncType{SyncTypeIFrame, SyncTypeRedirect},
		},
		{
			description:           "One Available - None",
			givenSyncTypes:        []SyncType{},
			givenIFrameTemplate:   nil,
			givenRedirectTemplate: endpointTemplate,
			expectedSyncTypes:     []SyncType{},
		},
		{
			description:           "One Available - One - Not Supported",
			givenSyncTypes:        []SyncType{SyncTypeIFrame},
			givenIFrameTemplate:   nil,
			givenRedirectTemplate: endpointTemplate,
			expectedSyncTypes:     []SyncType{},
		},
		{
			description:           "One Available - One - Supported",
			givenSyncTypes:        []SyncType{SyncTypeRedirect},
			givenIFrameTemplate:   nil,
			givenRedirectTemplate: endpointTemplate,
			expectedSyncTypes:     []SyncType{SyncTypeRedirect},
		},
		{
			description:           "One Available - Many",
			givenSyncTypes:        []SyncType{SyncTypeIFrame, SyncTypeRedirect},
			givenIFrameTemplate:   nil,
			givenRedirectTemplate: endpointTemplate,
			expectedSyncTypes:     []SyncType{SyncTypeRedirect},
		},
	}

	for _, test := range testCases {
		syncer := standardSyncer{
			iframe:   test.givenIFrameTemplate,
			redirect: test.givenRedirectTemplate,
		}
		result := syncer.filterSupportedSyncTypes(test.givenSyncTypes)
		assert.ElementsMatch(t, test.expectedSyncTypes, result, test.description)
	}
}

func TestSyncerGetSync(t *testing.T) {
	var (
		iframeTemplate    = template.Must(template.New("test").Parse("iframe,gdpr:{{.GDPR}},gdprconsent:{{.GDPRConsent}},ccpa:{{.USPrivacy}}"))
		redirectTemplate  = template.Must(template.New("test").Parse("redirect,gdpr:{{.GDPR}},gdprconsent:{{.GDPRConsent}},ccpa:{{.USPrivacy}}"))
		malformedTemplate = template.Must(template.New("test").Parse("malformed,invalid:{{.DoesNotExist}}"))
	)

	testCases := []struct {
		description          string
		givenSyncer          standardSyncer
		givenSyncTypes       []SyncType
		givenPrivacyPolicies privacy.Policies
		expectedError        string
		expectedSync         Sync
	}{
		{
			description:          "No Sync Types",
			givenSyncer:          standardSyncer{iframe: iframeTemplate, redirect: redirectTemplate},
			givenSyncTypes:       []SyncType{},
			givenPrivacyPolicies: privacy.Policies{GDPR: gdpr.Policy{Signal: "A", Consent: "B"}, CCPA: ccpa.Policy{Consent: "C"}},
			expectedError:        "no sync types provided",
		},
		{
			description:          "IFrame",
			givenSyncer:          standardSyncer{iframe: iframeTemplate, redirect: redirectTemplate},
			givenSyncTypes:       []SyncType{SyncTypeIFrame},
			givenPrivacyPolicies: privacy.Policies{GDPR: gdpr.Policy{Signal: "A", Consent: "B"}, CCPA: ccpa.Policy{Consent: "C"}},
			expectedSync:         Sync{URL: "iframe,gdpr:A,gdprconsent:B,ccpa:C", Type: SyncTypeIFrame, SupportCORS: false},
		},
		{
			description:          "Redirect",
			givenSyncer:          standardSyncer{iframe: iframeTemplate, redirect: redirectTemplate},
			givenSyncTypes:       []SyncType{SyncTypeRedirect},
			givenPrivacyPolicies: privacy.Policies{GDPR: gdpr.Policy{Signal: "A", Consent: "B"}, CCPA: ccpa.Policy{Consent: "C"}},
			expectedSync:         Sync{URL: "redirect,gdpr:A,gdprconsent:B,ccpa:C", Type: SyncTypeRedirect, SupportCORS: false},
		},
		{
			description:          "Macro Error",
			givenSyncer:          standardSyncer{iframe: malformedTemplate},
			givenSyncTypes:       []SyncType{SyncTypeIFrame},
			givenPrivacyPolicies: privacy.Policies{GDPR: gdpr.Policy{Signal: "A", Consent: "B"}, CCPA: ccpa.Policy{Consent: "C"}},
			expectedError:        "template: test:1:20: executing \"test\" at <.DoesNotExist>: can't evaluate field DoesNotExist in type macros.UserSyncTemplateParams",
		},
	}

	for _, test := range testCases {
		result, err := test.givenSyncer.GetSync(test.givenSyncTypes, test.givenPrivacyPolicies)

		if test.expectedError == "" {
			assert.NoError(t, err, test.description+":err")
			assert.Equal(t, test.expectedSync, result, test.description+":sync")
		} else {
			assert.EqualError(t, err, test.expectedError, test.description+":err")
		}
	}
}

func TestSyncerChooseSyncType(t *testing.T) {
	endpointTemplate := template.Must(template.New("test").Parse("iframe"))

	testCases := []struct {
		description           string
		givenSyncTypes        []SyncType
		givenDefaultSyncType  SyncType
		givenIFrameTemplate   *template.Template
		givenRedirectTemplate *template.Template
		expectedError         string
		expectedSyncType      SyncType
	}{
		{
			description:           "None Available - Error",
			givenSyncTypes:        []SyncType{},
			givenDefaultSyncType:  SyncTypeRedirect,
			givenIFrameTemplate:   endpointTemplate,
			givenRedirectTemplate: endpointTemplate,
			expectedError:         "no sync types provided",
		},
		{
			description:           "All Available - Choose Default",
			givenSyncTypes:        []SyncType{SyncTypeIFrame, SyncTypeRedirect},
			givenDefaultSyncType:  SyncTypeRedirect,
			givenIFrameTemplate:   endpointTemplate,
			givenRedirectTemplate: endpointTemplate,
			expectedSyncType:      SyncTypeRedirect,
		},
		{
			description:           "Default Not Available - Choose Other One",
			givenSyncTypes:        []SyncType{SyncTypeIFrame},
			givenDefaultSyncType:  SyncTypeRedirect,
			givenIFrameTemplate:   endpointTemplate,
			givenRedirectTemplate: endpointTemplate,
			expectedSyncType:      SyncTypeIFrame,
		},
		{
			description:           "None Supported - Error",
			givenSyncTypes:        []SyncType{SyncTypeIFrame},
			givenDefaultSyncType:  SyncTypeRedirect,
			givenIFrameTemplate:   nil,
			givenRedirectTemplate: endpointTemplate,
			expectedError:         "no sync types supported",
		},
	}

	for _, test := range testCases {
		syncer := standardSyncer{
			defaultSyncType: test.givenDefaultSyncType,
			iframe:          test.givenIFrameTemplate,
			redirect:        test.givenRedirectTemplate,
		}
		result, err := syncer.chooseSyncType(test.givenSyncTypes)

		if test.expectedError == "" {
			assert.NoError(t, err, test.description+":err")
			assert.Equal(t, test.expectedSyncType, result, test.description+":sync_type")
		} else {
			assert.EqualError(t, err, test.expectedError, test.description+":err")
		}
	}
}

func TestSyncerChooseTemplate(t *testing.T) {
	var (
		iframeTemplate   = template.Must(template.New("test").Parse("iframe"))
		redirectTemplate = template.Must(template.New("test").Parse("redirect"))
	)

	testCases := []struct {
		description      string
		givenSyncType    SyncType
		expectedTemplate *template.Template
	}{
		{
			description:      "IFrame",
			givenSyncType:    SyncTypeIFrame,
			expectedTemplate: iframeTemplate,
		},
		{
			description:      "Redirect",
			givenSyncType:    SyncTypeRedirect,
			expectedTemplate: redirectTemplate,
		},
		{
			description:      "Invalid",
			givenSyncType:    SyncType("invalid"),
			expectedTemplate: nil,
		},
	}

	for _, test := range testCases {
		syncer := standardSyncer{iframe: iframeTemplate, redirect: redirectTemplate}
		result := syncer.chooseTemplate(test.givenSyncType)
		assert.Equal(t, test.expectedTemplate, result, test.description)
	}
}
