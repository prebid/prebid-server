package usersync

import (
	"errors"
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/privacy"
	"github.com/stretchr/testify/assert"
)

func TestSyncerBuildError(t *testing.T) {
	err := SyncerBuildError{
		Bidder:    "anyBidder",
		SyncerKey: "anyKey",
		Err:       errors.New("anyError"),
	}
	assert.Equal(t, err.Error(), "cannot create syncer for bidder anyBidder with key anyKey: anyError")
}

func TestBuildSyncers(t *testing.T) {
	var (
		hostConfig              = config.Configuration{ExternalURL: "http://host.com", UserSync: config.UserSync{RedirectURL: "{{.ExternalURL}}/{{.SyncerKey}}/host"}}
		iframeConfig            = &config.SyncerEndpoint{URL: "https://bidder.com/iframe?redirect={{.RedirectURL}}"}
		iframeConfigError       = &config.SyncerEndpoint{URL: "https://bidder.com/iframe?redirect={{xRedirectURL}}"} // Error caused by invalid macro
		infoKeyAPopulated       = config.BidderInfo{Disabled: false, Syncer: &config.Syncer{Key: "a", IFrame: iframeConfig}}
		infoKeyADisabled        = config.BidderInfo{Disabled: true, Syncer: &config.Syncer{Key: "a", IFrame: iframeConfig}}
		infoKeyAEmpty           = config.BidderInfo{Disabled: false, Syncer: &config.Syncer{Key: "a"}}
		infoKeyAError           = config.BidderInfo{Disabled: false, Syncer: &config.Syncer{Key: "a", IFrame: iframeConfigError}}
		infoKeyASupportsOnly    = config.BidderInfo{Disabled: false, Syncer: &config.Syncer{Supports: []string{"iframe"}}}
		infoKeyBPopulated       = config.BidderInfo{Disabled: false, Syncer: &config.Syncer{Key: "b", IFrame: iframeConfig}}
		infoKeyBEmpty           = config.BidderInfo{Disabled: false, Syncer: &config.Syncer{Key: "b"}}
		infoKeyMissingPopulated = config.BidderInfo{Disabled: false, Syncer: &config.Syncer{IFrame: iframeConfig}}
	)

	// NOTE: The hostConfig includes the syncer key in the RedirectURL to distinguish between the syncer keys
	// in these tests. Look carefully at the end of the expected iframe urls to see the syncer key.

	testCases := []struct {
		description         string
		givenConfig         config.Configuration
		givenBidderInfos    config.BidderInfos
		expectedIFramesURLs map[string]string
		expectedErrors      []string
	}{
		{
			description:      "One",
			givenConfig:      hostConfig,
			givenBidderInfos: map[string]config.BidderInfo{"bidder1": infoKeyAPopulated},
			expectedIFramesURLs: map[string]string{
				"bidder1": "https://bidder.com/iframe?redirect=http%3A%2F%2Fhost.com%2Fbidder1%2Fhost",
			},
		},
		{
			description:      "One - Missing Key - Defaults To Bidder Name",
			givenConfig:      hostConfig,
			givenBidderInfos: map[string]config.BidderInfo{"bidder1": infoKeyMissingPopulated},
			expectedIFramesURLs: map[string]string{
				"bidder1": "https://bidder.com/iframe?redirect=http%3A%2F%2Fhost.com%2Fbidder1%2Fhost",
			},
		},
		{
			description:      "One - Syncer Error",
			givenConfig:      hostConfig,
			givenBidderInfos: map[string]config.BidderInfo{"bidder1": infoKeyAError},
			expectedErrors: []string{
				"cannot create syncer for bidder bidder1 with key a: iframe template: bidder1_usersync_url:1: function \"xRedirectURL\" not defined",
			},
		},
		{
			description:      "Many - Different Syncers",
			givenConfig:      hostConfig,
			givenBidderInfos: map[string]config.BidderInfo{"bidder1": infoKeyAPopulated, "bidder2": infoKeyBPopulated},
			expectedIFramesURLs: map[string]string{
				"bidder1": "https://bidder.com/iframe?redirect=http%3A%2F%2Fhost.com%2Fbidder1%2Fhost",
				"bidder2": "https://bidder.com/iframe?redirect=http%3A%2F%2Fhost.com%2Fbidder2%2Fhost",
			},
		},
		{
			description:      "Many - Same Syncers - One Primary",
			givenConfig:      hostConfig,
			givenBidderInfos: map[string]config.BidderInfo{"bidder1": infoKeyAPopulated, "bidder2": infoKeyAEmpty},
			expectedIFramesURLs: map[string]string{
				"bidder1": "https://bidder.com/iframe?redirect=http%3A%2F%2Fhost.com%2Fbidder1%2Fhost",
				"bidder2": "https://bidder.com/iframe?redirect=http%3A%2F%2Fhost.com%2Fbidder2%2Fhost",
			},
		},
		{
			description:      "Many - Same Syncers - Many Primaries",
			givenConfig:      hostConfig,
			givenBidderInfos: map[string]config.BidderInfo{"bidder1": infoKeyAPopulated, "bidder2": infoKeyAPopulated},
			expectedErrors: []string{
				"bidders bidder1, bidder2 define endpoints (iframe and/or redirect) for the same syncer key, but only one bidder is permitted to define endpoints",
			},
		},
		{
			description:      "Many - Same Syncers - Many Primaries - None Populated",
			givenConfig:      hostConfig,
			givenBidderInfos: map[string]config.BidderInfo{"bidder1": infoKeyAEmpty, "bidder2": infoKeyAEmpty},
			expectedErrors: []string{
				"bidders bidder1, bidder2 share the same syncer key, but none define endpoints (iframe and/or redirect)",
			},
		},
		{
			description:      "Many - Sync Error - Bidder Correct",
			givenConfig:      hostConfig,
			givenBidderInfos: map[string]config.BidderInfo{"bidder1": infoKeyAEmpty, "bidder2": infoKeyAError},
			expectedErrors: []string{
				"cannot create syncer for bidder bidder2 with key a: iframe template: bidder1_usersync_url:1: function \"xRedirectURL\" not defined",
				"cannot create syncer for bidder bidder2 with key a: iframe template: bidder2_usersync_url:1: function \"xRedirectURL\" not defined",
			},
		},
		{
			description:      "Many - Empty Syncers Ignored",
			givenConfig:      hostConfig,
			givenBidderInfos: map[string]config.BidderInfo{"bidder1": {}, "bidder2": infoKeyBPopulated},
			expectedIFramesURLs: map[string]string{
				"bidder2": "https://bidder.com/iframe?redirect=http%3A%2F%2Fhost.com%2Fbidder2%2Fhost",
			},
		},
		{
			description:      "Many - Disabled Syncers Ignored",
			givenConfig:      hostConfig,
			givenBidderInfos: map[string]config.BidderInfo{"bidder1": infoKeyADisabled, "bidder2": infoKeyBPopulated},
			expectedIFramesURLs: map[string]string{
				"bidder2": "https://bidder.com/iframe?redirect=http%3A%2F%2Fhost.com%2Fbidder2%2Fhost",
			},
		},
		{
			description:      "Many - Supports Only Syncers Ignored",
			givenConfig:      hostConfig,
			givenBidderInfos: map[string]config.BidderInfo{"bidder1": infoKeyASupportsOnly, "bidder2": infoKeyBPopulated},
			expectedIFramesURLs: map[string]string{
				"bidder2": "https://bidder.com/iframe?redirect=http%3A%2F%2Fhost.com%2Fbidder2%2Fhost",
			},
		},
		{
			description:      "Many - Multiple Errors",
			givenConfig:      hostConfig,
			givenBidderInfos: map[string]config.BidderInfo{"bidder1": infoKeyAError, "bidder2": infoKeyBEmpty},
			expectedErrors: []string{
				"cannot create syncer for bidder bidder1 with key a: iframe template: bidder1_usersync_url:1: function \"xRedirectURL\" not defined",
				"cannot create syncer for bidder bidder2 with key b: at least one endpoint (iframe and/or redirect) is required",
			},
		},
		{
			description:      "ExternalURL Host User Sync Override",
			givenConfig:      config.Configuration{ExternalURL: "http://host.com", UserSync: config.UserSync{ExternalURL: "http://hostoverride.com", RedirectURL: "{{.ExternalURL}}/{{.SyncerKey}}/host"}},
			givenBidderInfos: map[string]config.BidderInfo{"bidder1": infoKeyAPopulated},
			expectedIFramesURLs: map[string]string{
				"bidder1": "https://bidder.com/iframe?redirect=http%3A%2F%2Fhostoverride.com%2Fbidder1%2Fhost",
			},
		},
		{
			description: "parent-and-alias-cannot-have-same-syncer-key",
			givenConfig: config.Configuration{ExternalURL: "http://host.com", UserSync: config.UserSync{ExternalURL: "http://hostoverride.com", RedirectURL: "{{.ExternalURL}}/{{.SyncerKey}}/host"}},
			givenBidderInfos: map[string]config.BidderInfo{
				"bidder1": {Syncer: &config.Syncer{Key: "key", IFrame: iframeConfig}},
				"bidder2": {AliasOf: "bidder1", Syncer: &config.Syncer{Key: "key", IFrame: iframeConfig}},
			},
			expectedErrors: []string{"syncer key of alias bidder bidder2 is same as the syncer key for its parent bidder bidder1"},
		},
	}

	for _, test := range testCases {
		result, errs := BuildSyncers(&test.givenConfig, test.givenBidderInfos)

		if len(test.expectedErrors) == 0 {
			assert.Empty(t, errs, test.description+":err")
			resultRenderedIFrameURLS := map[string]string{}
			for k, v := range result {
				iframeRendered, err := v.GetSync([]SyncType{SyncTypeIFrame}, privacy.Policies{})
				if assert.NoError(t, err, test.description+"key:%s,:iframe_render", k) {
					resultRenderedIFrameURLS[k] = iframeRendered.URL
				}
			}
			assert.Equal(t, test.expectedIFramesURLs, resultRenderedIFrameURLS, test.description+":result")
		} else {
			errMessages := make([]string, 0, len(errs))
			for _, e := range errs {
				errMessages = append(errMessages, e.Error())
			}
			assert.ElementsMatch(t, test.expectedErrors, errMessages, test.description+":err")
			assert.Empty(t, result, test.description+":result")
		}
	}
}

func TestShouldCreateSyncer(t *testing.T) {
	var (
		anySupports = []string{"iframe"}
		anyEndpoint = &config.SyncerEndpoint{}
		anyCORS     = true
	)

	testCases := []struct {
		description string
		given       config.BidderInfo
		expected    bool
	}{
		{
			description: "Enabled, No Syncer",
			given:       config.BidderInfo{Disabled: false, Syncer: nil},
			expected:    false,
		},
		{
			description: "Enabled, Syncer",
			given:       config.BidderInfo{Disabled: false, Syncer: &config.Syncer{Key: "anyKey"}},
			expected:    true,
		},
		{
			description: "Enabled, Syncer - Fully Loaded",
			given:       config.BidderInfo{Disabled: false, Syncer: &config.Syncer{Key: "anyKey", Supports: anySupports, IFrame: anyEndpoint, Redirect: anyEndpoint, SupportCORS: &anyCORS}},
			expected:    true,
		},
		{
			description: "Enabled, Syncer - Only Key",
			given:       config.BidderInfo{Disabled: false, Syncer: &config.Syncer{Key: "anyKey"}},
			expected:    true,
		},
		{
			description: "Enabled, Syncer - Only Supports",
			given:       config.BidderInfo{Disabled: false, Syncer: &config.Syncer{Supports: anySupports}},
			expected:    false,
		},
		{
			description: "Enabled, Syncer - Only IFrame",
			given:       config.BidderInfo{Disabled: false, Syncer: &config.Syncer{IFrame: anyEndpoint}},
			expected:    true,
		},
		{
			description: "Enabled, Syncer - Only Redirect",
			given:       config.BidderInfo{Disabled: false, Syncer: &config.Syncer{Redirect: anyEndpoint}},
			expected:    true,
		},
		{
			description: "Enabled, Syncer - Only SupportCORS",
			given:       config.BidderInfo{Disabled: false, Syncer: &config.Syncer{SupportCORS: &anyCORS}},
			expected:    true,
		},
		{
			description: "Disabled, No Syncer",
			given:       config.BidderInfo{Disabled: true, Syncer: nil},
			expected:    false,
		},
		{
			description: "Disabled, Syncer",
			given:       config.BidderInfo{Disabled: true, Syncer: &config.Syncer{Key: "anyKey"}},
			expected:    false,
		},
		{
			description: "Disabled, Syncer - Fully Loaded",
			given:       config.BidderInfo{Disabled: true, Syncer: &config.Syncer{Key: "anyKey", Supports: anySupports, IFrame: anyEndpoint, Redirect: anyEndpoint, SupportCORS: &anyCORS}},
			expected:    false,
		},
		{
			description: "Disabled, Syncer - Only Key",
			given:       config.BidderInfo{Disabled: true, Syncer: &config.Syncer{Key: "anyKey"}},
			expected:    false,
		},
		{
			description: "Disabled, Syncer - Only Supports",
			given:       config.BidderInfo{Disabled: true, Syncer: &config.Syncer{Supports: anySupports}},
			expected:    false,
		},
		{
			description: "Disabled, Syncer - Only IFrame",
			given:       config.BidderInfo{Disabled: true, Syncer: &config.Syncer{IFrame: anyEndpoint}},
			expected:    false,
		},
		{
			description: "Disabled, Syncer - Only Redirect",
			given:       config.BidderInfo{Disabled: true, Syncer: &config.Syncer{Redirect: anyEndpoint}},
			expected:    false,
		},
		{
			description: "Disabled, Syncer - Only SupportCORS",
			given:       config.BidderInfo{Disabled: true, Syncer: &config.Syncer{SupportCORS: &anyCORS}},
			expected:    false,
		},
	}

	for _, test := range testCases {
		result := shouldCreateSyncer(test.given)
		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestChooseSyncerConfig(t *testing.T) {
	var (
		bidderAPopulated = namedSyncerConfig{name: "bidderA", cfg: config.Syncer{Key: "a", IFrame: &config.SyncerEndpoint{URL: "anyURL"}}}
		bidderAEmpty     = namedSyncerConfig{name: "bidderA", cfg: config.Syncer{}}
		bidderBPopulated = namedSyncerConfig{name: "bidderB", cfg: config.Syncer{Key: "a", IFrame: &config.SyncerEndpoint{URL: "anyURL"}}}
		bidderBEmpty     = namedSyncerConfig{name: "bidderB", cfg: config.Syncer{}}
		syncerCfg        = config.Syncer{Key: "key", Redirect: &config.SyncerEndpoint{RedirectURL: "redirect-url"}}
		parent1          = namedSyncerConfig{name: "parent-1", cfg: syncerCfg, bidderInfo: config.BidderInfo{AliasOf: ""}}
		parent2          = namedSyncerConfig{name: "parent-2", cfg: syncerCfg, bidderInfo: config.BidderInfo{AliasOf: ""}}
		alias1           = namedSyncerConfig{name: "alias-1", cfg: syncerCfg, bidderInfo: config.BidderInfo{AliasOf: "parent-1"}}
		alias2           = namedSyncerConfig{name: "alias-2", cfg: syncerCfg, bidderInfo: config.BidderInfo{AliasOf: "parent-2"}}
		alias3           = namedSyncerConfig{name: "alias-3", cfg: syncerCfg, bidderInfo: config.BidderInfo{AliasOf: "parent-2"}}
	)

	testCases := []struct {
		description    string
		given          []namedSyncerConfig
		expectedConfig namedSyncerConfig
		expectedError  string
	}{
		{
			description:    "One",
			given:          []namedSyncerConfig{bidderAPopulated},
			expectedConfig: bidderAPopulated,
		},
		{
			description:    "Many - Same Key - Unique Configs",
			given:          []namedSyncerConfig{bidderAEmpty, bidderBPopulated},
			expectedConfig: bidderBPopulated,
		},
		{
			description:   "Many - Same Key - Multiple Configs",
			given:         []namedSyncerConfig{bidderAPopulated, bidderBPopulated},
			expectedError: "bidders bidderA, bidderB define endpoints (iframe and/or redirect) for the same syncer key, but only one bidder is permitted to define endpoints",
		},
		{
			description:   "Many - Same Key - No Configs",
			given:         []namedSyncerConfig{bidderAEmpty, bidderBEmpty},
			expectedError: "bidders bidderA, bidderB share the same syncer key, but none define endpoints (iframe and/or redirect)",
		},
		{
			description:    "Many - Same Key - Unique Configs",
			given:          []namedSyncerConfig{bidderAEmpty, bidderBPopulated},
			expectedConfig: bidderBPopulated,
		},
		{
			description:    "alias-can-have-same-key-as-parent",
			given:          []namedSyncerConfig{parent1, alias1},
			expectedConfig: alias1,
		},
		{
			description:   "alias-of-differnt-parent-cannot-have-same-key",
			given:         []namedSyncerConfig{alias1, alias2},
			expectedError: "alias bidders alias-1, alias-2 of different parents defines endpoints (iframe and/or redirect) for the same syncer key, but only one bidder is permitted to define endpoints",
		},
		{
			description:   "non-alias-bidders-cannot-have-same-key",
			given:         []namedSyncerConfig{parent1, parent2},
			expectedError: "bidders parent-1, parent-2 define endpoints (iframe and/or redirect) for the same syncer key, but only one bidder is permitted to define endpoints",
		},
		{
			description:   "non-alias-and-aliases-of-same-parent-cannot-have-same-key",
			given:         []namedSyncerConfig{parent1, alias2, alias3},
			expectedError: "alias bidders alias-2, alias-3 and non-alias bidder parent-1 defines endpoints (iframe and/or redirect) for the same syncer key, but only one bidder is permitted to define endpoints",
		},
	}

	for _, test := range testCases {
		result, err := chooseSyncerConfig(test.given)
		if test.expectedError == "" {
			assert.NoError(t, err, test.description+":err")
			assert.Equal(t, test.expectedConfig, result, test.description+":result")
		} else {
			assert.EqualError(t, err, test.expectedError, test.description+":err")
			assert.Empty(t, result, test.description+":result")
		}
	}
}

func TestGetSyncerKey(t *testing.T) {
	redirectCfg := config.SyncerEndpoint{URL: "redirect-url", UserMacro: "$UID"}
	syncer1 := config.Syncer{Key: "", Redirect: &redirectCfg}
	syncer2 := syncer1
	syncer3 := config.Syncer{Key: "key-1", Redirect: &redirectCfg}
	syncer4 := config.Syncer{Key: "key-1"}

	biddersWithSyncerCfg := config.BidderInfos{
		"parent-1": config.BidderInfo{Syncer: &syncer1},
		"alias-1":  config.BidderInfo{AliasOf: "parent-1", Syncer: &syncer1},
		"alias-2":  config.BidderInfo{AliasOf: "parent-1", Syncer: &syncer2},
		"parent-2": config.BidderInfo{Syncer: &syncer3},
		"alias-3":  config.BidderInfo{AliasOf: "parent-1", Syncer: &syncer3},
		"alias-4":  config.BidderInfo{AliasOf: "parent-2", Syncer: &syncer4},
	}

	tests := []struct {
		name                 string
		biddersWithSyncerCfg map[string]config.BidderInfo
		bidderName           string
		bidderInfo           config.BidderInfo
		expectedErr          string
		expectedSyncerKey    string
	}{
		{
			name:        "alias_bidder_has_no_syncer_config",
			bidderName:  "alias-1",
			bidderInfo:  config.BidderInfo{Syncer: nil},
			expectedErr: "found no syncer config for bidder alias-1",
		},
		{
			name:                 "use_parent_name_as_syncer_key_when_syncer_key_is_empty_and_alias_inherits_parent_syncer_config",
			biddersWithSyncerCfg: biddersWithSyncerCfg,
			bidderName:           "alias-1",
			bidderInfo:           biddersWithSyncerCfg["alias-1"],
			expectedSyncerKey:    "parent-1",
		},
		{
			name:                 "use_alias_name_as_syncer_key_when_syncer_key_is_empty_and_alias_does_not_inherit_parent_syncer_config",
			biddersWithSyncerCfg: biddersWithSyncerCfg,
			bidderName:           "alias-2",
			bidderInfo:           biddersWithSyncerCfg["alias-2"],
			expectedSyncerKey:    "alias-2",
		},
		{
			name:                 "use_syncer_key_when_parent_and_alias_have_different_syncer_config",
			biddersWithSyncerCfg: biddersWithSyncerCfg,
			bidderName:           "alias-3",
			bidderInfo:           biddersWithSyncerCfg["alias-3"],
			expectedSyncerKey:    biddersWithSyncerCfg["alias-3"].Syncer.Key,
		},
		{
			name:                 "use_syncer_key_when_parent_and_alias_have_different_syncer_config",
			biddersWithSyncerCfg: biddersWithSyncerCfg,
			bidderName:           "alias-4",
			bidderInfo:           biddersWithSyncerCfg["alias-4"],
			expectedErr:          "syncer key of alias bidder alias-4 is same as the syncer key for its parent bidder parent-2",
		},
		{
			name:              "use_bidder_name_when_non-alias_bidder_has_no_syncer_key",
			bidderName:        "parent-1",
			bidderInfo:        biddersWithSyncerCfg["parent-1"],
			expectedSyncerKey: "parent-1",
		},
		{
			name:              "use_syncer_key_when_non-alias_bidder_has_defined_syncer_key",
			bidderName:        "parent-2",
			bidderInfo:        biddersWithSyncerCfg["parent-2"],
			expectedSyncerKey: biddersWithSyncerCfg["parent-2"].Syncer.Key,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			syncerKey, err := getSyncerKey(test.biddersWithSyncerCfg, test.bidderName, test.bidderInfo)
			if test.expectedErr != "" {
				assert.Equal(t, test.expectedErr, err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expectedSyncerKey, syncerKey)
			}
		})
	}
}
