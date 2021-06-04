package usersync

import (
	"strings"
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/privacy"
	"github.com/stretchr/testify/assert"
)

func TestBuildSyncers(t *testing.T) {
	var (
		hostConfig              = config.UserSync{ExternalURL: "http://host.com", RedirectURL: "{{.ExternalURL}}/{{.SyncerKey}}/host"}
		iframeConfig            = &config.SyncerEndpoint{URL: "https://bidder.com/iframe?redirect={{.RedirectURL}}"}
		infoKeyAPopulated       = config.BidderInfo{Syncer: &config.Syncer{Key: "a", IFrame: iframeConfig}}
		infoKeyAEmpty           = config.BidderInfo{Syncer: &config.Syncer{Key: "a"}}
		infoKeyAError           = config.BidderInfo{Syncer: &config.Syncer{Key: "a", Default: "redirect", IFrame: iframeConfig}} // Error caused by invalid default sync type
		infoKeyBPopulated       = config.BidderInfo{Syncer: &config.Syncer{Key: "b", IFrame: iframeConfig}}
		infoKeyBEmpty           = config.BidderInfo{Syncer: &config.Syncer{Key: "b"}}
		infoKeyMissingPopulated = config.BidderInfo{Syncer: &config.Syncer{IFrame: iframeConfig}}
	)

	// NOTE: The hostConfig includes the syncer key in the RedirectURL to distinguish between the syncer keys
	// in these tests. Look carefully at the end of the expected iframe urls to see the syncer key.

	testCases := []struct {
		description           string
		givenBidderInfos      config.BidderInfos
		expectedIFramesURLs   map[string]string
		expectedErrorHeader   string
		expectedErrorSegments []string
	}{
		{
			description:      "One",
			givenBidderInfos: map[string]config.BidderInfo{"bidder1": infoKeyAPopulated},
			expectedIFramesURLs: map[string]string{
				"bidder1": "https://bidder.com/iframe?redirect=http%3A%2F%2Fhost.com%2Fa%2Fhost",
			},
		},
		{
			description:      "One - Missing Key - Defaults To Bidder Name",
			givenBidderInfos: map[string]config.BidderInfo{"bidder1": infoKeyMissingPopulated},
			expectedIFramesURLs: map[string]string{
				"bidder1": "https://bidder.com/iframe?redirect=http%3A%2F%2Fhost.com%2Fbidder1%2Fhost",
			},
		},
		{
			description:         "One - Syncer Error",
			givenBidderInfos:    map[string]config.BidderInfo{"bidder1": infoKeyAError},
			expectedErrorHeader: "user sync (1 error)",
			expectedErrorSegments: []string{
				"cannot create syncer for bidder bidder1 with key a. default is set to redirect but no redirect endpoint is configured\n",
			},
		},
		{
			description:      "Many - Different Syncers",
			givenBidderInfos: map[string]config.BidderInfo{"bidder1": infoKeyAPopulated, "bidder2": infoKeyBPopulated},
			expectedIFramesURLs: map[string]string{
				"bidder1": "https://bidder.com/iframe?redirect=http%3A%2F%2Fhost.com%2Fa%2Fhost",
				"bidder2": "https://bidder.com/iframe?redirect=http%3A%2F%2Fhost.com%2Fb%2Fhost",
			},
		},
		{
			description:      "Many - Same Syncers - One Primary",
			givenBidderInfos: map[string]config.BidderInfo{"bidder1": infoKeyAPopulated, "bidder2": infoKeyAEmpty},
			expectedIFramesURLs: map[string]string{
				"bidder1": "https://bidder.com/iframe?redirect=http%3A%2F%2Fhost.com%2Fa%2Fhost",
				"bidder2": "https://bidder.com/iframe?redirect=http%3A%2F%2Fhost.com%2Fa%2Fhost",
			},
		},
		{
			description:         "Many - Same Syncers - Many Primaries",
			givenBidderInfos:    map[string]config.BidderInfo{"bidder1": infoKeyAPopulated, "bidder2": infoKeyAPopulated},
			expectedErrorHeader: "user sync (1 error)",
			expectedErrorSegments: []string{
				"bidders bidder1, bidder2 define endpoints (iframe and/or redirect) for the same syncer key, but only one bidder is permitted to define endpoints\n",
			},
		},
		{
			description:         "Many - Sync Error - Bidder Correct",
			givenBidderInfos:    map[string]config.BidderInfo{"bidder1": infoKeyAEmpty, "bidder2": infoKeyAError},
			expectedErrorHeader: "user sync (1 error)",
			expectedErrorSegments: []string{
				"cannot create syncer for bidder bidder2 with key a. default is set to redirect but no redirect endpoint is configured\n",
			},
		},
		{
			description:      "Many - Empty Syncers Ignored",
			givenBidderInfos: map[string]config.BidderInfo{"bidder1": {}, "bidder2": infoKeyBPopulated},
			expectedIFramesURLs: map[string]string{
				"bidder2": "https://bidder.com/iframe?redirect=http%3A%2F%2Fhost.com%2Fb%2Fhost",
			},
		},
		{
			description:         "Many - Multiple Errors",
			givenBidderInfos:    map[string]config.BidderInfo{"bidder1": infoKeyAError, "bidder2": infoKeyBEmpty},
			expectedErrorHeader: "user sync (2 errors)",
			expectedErrorSegments: []string{
				"cannot create syncer for bidder bidder1 with key a. default is set to redirect but no redirect endpoint is configured\n",
				"cannot create syncer for bidder bidder2 with key b. at least one endpoint (iframe and/or redirect) is required\n",
			},
		},
	}

	for _, test := range testCases {
		result, err := BuildSyncers(hostConfig, test.givenBidderInfos)

		if test.expectedErrorHeader == "" {
			assert.NoError(t, err, test.description+":err")
			resultRenderedIFrameURLS := map[string]string{}
			for k, v := range result {
				iframeRendered, err := v.GetSync([]SyncType{SyncTypeIFrame}, privacy.Policies{})
				if assert.NoError(t, err, test.description+"key:%s,:iframe_render", k) {
					resultRenderedIFrameURLS[k] = iframeRendered.URL
				}
			}
			assert.Equal(t, test.expectedIFramesURLs, resultRenderedIFrameURLS, test.description+":result")
		} else {
			errMessage := err.Error()
			assert.True(t, strings.HasPrefix(errMessage, test.expectedErrorHeader), test.description+":err")
			for _, s := range test.expectedErrorSegments {
				assert.Contains(t, errMessage, s, test.description+":err")
			}
			assert.Empty(t, result, test.description+":result")
		}
	}
}

func TestChooseSyncerConfig(t *testing.T) {
	var (
		bidderAPopulated = namedSyncerConfig{name: "bidderA", cfg: config.Syncer{Key: "a", IFrame: &config.SyncerEndpoint{URL: "anyURL"}}}
		bidderAEmpty     = namedSyncerConfig{name: "bidderA", cfg: config.Syncer{}}
		bidderBPopulated = namedSyncerConfig{name: "bidderB", cfg: config.Syncer{Key: "a", IFrame: &config.SyncerEndpoint{URL: "anyURL"}}}
		bidderBEmpty     = namedSyncerConfig{name: "bidderB", cfg: config.Syncer{}}
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
