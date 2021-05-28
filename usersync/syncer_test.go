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

// NewSyncer

func TestComposeTemplate(t *testing.T) {
	var (
		key           = "anyKey"
		syncTypeValue = "x"
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
			description:     "No Composed Macros - Validated Legacy Overrides",
			givenHostConfig: config.UserSync{ExternalURL: "externalURL", RedirectURL: "redirectURL"},
			givenSyncerEndpoint: config.SyncerEndpoint{
				URL: "hasNoMacros,gdpr={{.GDPR}}",
			},
			expectedRendered: "hasNoMacros,gdpr=A",
		},
		{
			description:     "All Composed Macros",
			givenHostConfig: config.UserSync{ExternalURL: "externalURL", RedirectURL: "redirectURL"},
			givenSyncerEndpoint: config.SyncerEndpoint{
				URL:         "https://bidder.com/sync?redirect={{.RedirectURL}}",
				RedirectURL: "{{.ExternalURL}}/setuid?bidder={{.SyncerKey}}&f={{.SyncType}}&gdpr={{.GDPR}}&uid={{.UserMacro}}",
				ExternalURL: "http://host.com",
				UserMacro:   "$UID$",
			},
			expectedRendered: "https://bidder.com/sync?redirect=http%3A%2F%2Fhost.com%2Fsetuid%3Fbidder%3DanyKey%26f%3Dx%26gdpr%3DA%26uid%3D%24UID%24",
		},
	}

	for _, test := range testCases {
		result, err := composeTemplate(key, syncTypeValue, test.givenHostConfig, test.givenSyncerEndpoint)

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
			description: "Just Macro",
			given:       "{{.Macro}}",
			expected:    "{{.Macro}}",
		},
		{
			description: "Just Text",
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
			description: "Characters In Macros Not Escaped",
			given:       "{{.Macro&}}",
			expected:    "{{.Macro&}}",
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
			expectedError: "composed url \"not-a-url,gdpr:anyGDPR,gdprconsent:anyGDPRConsent,ccpa:anyCCPAConsent\" is invalid",
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
	syncer := standardSyncer{key: "foo"}
	assert.Equal(t, "foo", syncer.Key())
}

func TestSyncerSupportsType(t *testing.T) {
	endpointTemplate := template.Must(template.New("test").Parse("iframe"))

	testCases := []struct {
		description           string
		givenSyncTypes        []SyncType
		givenIFrameTemplate   *template.Template
		givenRedirectTemplate *template.Template
		expected              bool
	}{
		{
			description:           "All Available - None",
			givenSyncTypes:        []SyncType{},
			givenIFrameTemplate:   endpointTemplate,
			givenRedirectTemplate: endpointTemplate,
			expected:              false,
		},
		{
			description:           "All Available - One",
			givenSyncTypes:        []SyncType{SyncTypeIFrame},
			givenIFrameTemplate:   endpointTemplate,
			givenRedirectTemplate: endpointTemplate,
			expected:              true,
		},
		{
			description:           "All Available - Many",
			givenSyncTypes:        []SyncType{SyncTypeIFrame, SyncTypeRedirect},
			givenIFrameTemplate:   endpointTemplate,
			givenRedirectTemplate: endpointTemplate,
			expected:              true,
		},
		{
			description:           "One Available - None",
			givenSyncTypes:        []SyncType{},
			givenIFrameTemplate:   endpointTemplate,
			givenRedirectTemplate: nil,
			expected:              false,
		},
		{
			description:           "One Available - One - Supported",
			givenSyncTypes:        []SyncType{SyncTypeIFrame},
			givenIFrameTemplate:   endpointTemplate,
			givenRedirectTemplate: nil,
			expected:              true,
		},
		{
			description:           "One Available - One - Not Supported",
			givenSyncTypes:        []SyncType{SyncTypeRedirect},
			givenIFrameTemplate:   endpointTemplate,
			givenRedirectTemplate: nil,
			expected:              false,
		},
		{
			description:           "One Available - Many",
			givenSyncTypes:        []SyncType{SyncTypeIFrame, SyncTypeRedirect},
			givenIFrameTemplate:   endpointTemplate,
			givenRedirectTemplate: nil,
			expected:              true,
		},
	}

	for _, test := range testCases {
		syncer := standardSyncer{
			iframe:   test.givenIFrameTemplate,
			redirect: test.givenRedirectTemplate,
		}
		result := syncer.SupportsType(test.givenSyncTypes)
		assert.Equal(t, test.expected, result, test.description)
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
			description:          "Resolve Macros Error",
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
			givenSyncType:    SyncType(42),
			expectedTemplate: nil,
		},
	}

	for _, test := range testCases {
		syncer := standardSyncer{iframe: iframeTemplate, redirect: redirectTemplate}
		result := syncer.chooseTemplate(test.givenSyncType)
		assert.Equal(t, test.expectedTemplate, result, test.description)
	}
}
