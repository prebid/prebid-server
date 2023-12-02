package http

import (
	"testing"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/v2/analytics"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/prebid/prebid-server/v2/util/randomutil"
	"github.com/stretchr/testify/assert"
)

type FakeRandomNumberGenerator struct {
	Number float64
}

func (f FakeRandomNumberGenerator) GenerateInt63() int64 {
	return 0
}

func (f FakeRandomNumberGenerator) GenerateFloat64() float64 {
	return f.Number
}

func TestCreateAuctionFilter(t *testing.T) {
	testCases := []struct {
		name            string
		randomGenerator randomutil.RandomGenerator
		feature         config.AnalyticsFeature
		event           *analytics.AuctionObject
		shouldSample    bool
	}{
		{
			name: "Test with nil event",
			feature: config.AnalyticsFeature{
				SampleRate: 1,
			},
			event:           nil,
			randomGenerator: randomutil.RandomNumberGenerator{},
			shouldSample:    false,
		},
		{
			name:            "Sample everything with 1",
			randomGenerator: randomutil.RandomNumberGenerator{},
			feature: config.AnalyticsFeature{
				SampleRate: 1,
			},
			event:        &analytics.AuctionObject{},
			shouldSample: true,
		},
		{
			name:            "Test with SampleRate 0",
			randomGenerator: randomutil.RandomNumberGenerator{},
			feature: config.AnalyticsFeature{
				SampleRate: 0,
			},
			event:        &analytics.AuctionObject{},
			shouldSample: false,
		},
		{
			name:            "Should not sample when the random number is greater than the sample rate",
			randomGenerator: FakeRandomNumberGenerator{Number: 0.2},
			feature: config.AnalyticsFeature{
				SampleRate: 0.1,
			},
			event:        &analytics.AuctionObject{},
			shouldSample: false,
		},
		{
			name:            "Filter on Account",
			randomGenerator: randomutil.RandomNumberGenerator{},
			feature: config.AnalyticsFeature{
				SampleRate: 1,
				Filter:     "Account.ID == \"123\"",
			},
			event: &analytics.AuctionObject{
				Account: &config.Account{
					ID: "123",
				},
			},
			shouldSample: true,
		},
		{
			name:            "Filter on RequestWrapper.BidRequest.Site.ID",
			randomGenerator: randomutil.RandomNumberGenerator{},
			feature: config.AnalyticsFeature{
				SampleRate: 1,
				Filter:     "RequestWrapper.BidRequest.Site.ID == \"123\"",
			},
			event: &analytics.AuctionObject{
				RequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						ID: "123",
					},
				}},
			},
			shouldSample: true,
		},
		{
			name:            "Filter on RequestWrapper.BidRequest.App.ID",
			randomGenerator: randomutil.RandomNumberGenerator{},
			feature: config.AnalyticsFeature{
				SampleRate: 1,
				Filter:     "RequestWrapper.BidRequest.App.ID == \"123\"",
			},
			event: &analytics.AuctionObject{
				RequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
					App: &openrtb2.App{
						ID: "123",
					},
				}},
			},
			shouldSample: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filter, err := createAuctionFilter(tc.feature, tc.randomGenerator)
			assert.NoError(t, err)

			gotResult := filter(tc.event)
			assert.Equal(t, tc.shouldSample, gotResult)
		})
	}
}

func TestCreateAmpFilter(t *testing.T) {
	testCases := []struct {
		name            string
		randomGenerator randomutil.RandomGenerator
		feature         config.AnalyticsFeature
		event           *analytics.AmpObject
		shouldSample    bool
	}{
		{
			name:            "Test with nil event",
			randomGenerator: randomutil.RandomNumberGenerator{},
			feature: config.AnalyticsFeature{
				SampleRate: 1,
			},
			event:        nil,
			shouldSample: false,
		},
		{
			name:            "Sample everything with 1",
			randomGenerator: randomutil.RandomNumberGenerator{},
			feature: config.AnalyticsFeature{
				SampleRate: 1,
			},
			event:        &analytics.AmpObject{},
			shouldSample: true,
		},
		{
			name:            "Test with SampleRate 0",
			randomGenerator: randomutil.RandomNumberGenerator{},
			feature: config.AnalyticsFeature{
				SampleRate: 0,
			},
			event:        &analytics.AmpObject{},
			shouldSample: false,
		},
		{
			name:            "Should not sample when the random number is greater than the sample rate",
			randomGenerator: FakeRandomNumberGenerator{Number: 0.2},
			feature: config.AnalyticsFeature{
				SampleRate: 0.1,
			},
			event:        &analytics.AmpObject{},
			shouldSample: false,
		},
		{
			name:            "Filter on Account",
			randomGenerator: randomutil.RandomNumberGenerator{},
			feature: config.AnalyticsFeature{
				SampleRate: 1,
				Filter:     "Status == 1",
			},
			event: &analytics.AmpObject{
				Status: 1,
			},
			shouldSample: true,
		},
		{
			name:            "Filter on RequestWrapper.BidRequest.Site.ID",
			randomGenerator: randomutil.RandomNumberGenerator{},
			feature: config.AnalyticsFeature{
				SampleRate: 1,
				Filter:     "RequestWrapper.BidRequest.Site.ID == '123'",
			},
			event: &analytics.AmpObject{
				RequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						ID: "123",
					},
				}},
			},
			shouldSample: true,
		},
		{
			name:            "Filter on RequestWrapper.BidRequest.App.ID",
			randomGenerator: randomutil.RandomNumberGenerator{},
			feature: config.AnalyticsFeature{
				SampleRate: 1,
				Filter:     "RequestWrapper.BidRequest.App.ID == \"123\"",
			},
			event: &analytics.AmpObject{
				RequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
					App: &openrtb2.App{
						ID: "123",
					},
				}},
			},
			shouldSample: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filter, err := createAmpFilter(tc.feature, tc.randomGenerator)
			assert.NoError(t, err)

			gotResult := filter(tc.event)
			assert.Equal(t, tc.shouldSample, gotResult)
		})
	}
}

func TestCreateCookieSyncFilter(t *testing.T) {
	testCases := []struct {
		name            string
		randomGenerator randomutil.RandomGenerator
		feature         config.AnalyticsFeature
		event           *analytics.CookieSyncObject
		shouldSample    bool
	}{
		{
			name:            "Test with nil event",
			randomGenerator: randomutil.RandomNumberGenerator{},
			feature: config.AnalyticsFeature{
				SampleRate: 1,
			},
			event:        nil,
			shouldSample: false,
		},
		{
			name:            "Sample everything with 1",
			randomGenerator: randomutil.RandomNumberGenerator{},
			feature: config.AnalyticsFeature{
				SampleRate: 1,
			},
			event:        &analytics.CookieSyncObject{},
			shouldSample: true,
		},
		{
			name:            "Test with SampleRate 0",
			randomGenerator: randomutil.RandomNumberGenerator{},
			feature: config.AnalyticsFeature{
				SampleRate: 0,
			},
			event:        &analytics.CookieSyncObject{},
			shouldSample: false,
		},
		{
			name:            "Should not sample when the random number is greater than the sample rate",
			randomGenerator: FakeRandomNumberGenerator{Number: 0.2},
			feature: config.AnalyticsFeature{
				SampleRate: 0.1,
			},
			event:        &analytics.CookieSyncObject{},
			shouldSample: false,
		},
		{
			name:            "Filter on Status",
			randomGenerator: randomutil.RandomNumberGenerator{},
			feature: config.AnalyticsFeature{
				SampleRate: 1,
				Filter:     "Status == 1",
			},
			event: &analytics.CookieSyncObject{
				Status: 1,
			},
			shouldSample: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filter, err := createCookieSyncFilter(tc.feature, tc.randomGenerator)
			assert.NoError(t, err)

			gotResult := filter(tc.event)
			assert.Equal(t, tc.shouldSample, gotResult)
		})
	}
}

func TestCreateNotificationFilter(t *testing.T) {
	testCases := []struct {
		name            string
		randomGenerator randomutil.RandomGenerator
		feature         config.AnalyticsFeature
		event           *analytics.NotificationEvent
		shouldSample    bool
	}{
		{
			name:            "Test with nil event",
			randomGenerator: randomutil.RandomNumberGenerator{},
			feature: config.AnalyticsFeature{
				SampleRate: 1,
			},
			event:        nil,
			shouldSample: false,
		},
		{
			name:            "Test with SampleRate 0",
			randomGenerator: randomutil.RandomNumberGenerator{},
			feature: config.AnalyticsFeature{
				SampleRate: 0,
			},
			event:        &analytics.NotificationEvent{},
			shouldSample: false,
		},
		{
			name:            "Sample everything with 1",
			randomGenerator: randomutil.RandomNumberGenerator{},
			feature: config.AnalyticsFeature{
				SampleRate: 1,
			},
			event:        &analytics.NotificationEvent{},
			shouldSample: true,
		},
		{
			name:            "Should not sample when the random number is greater than the sample rate",
			randomGenerator: FakeRandomNumberGenerator{Number: 0.2},
			feature: config.AnalyticsFeature{
				SampleRate: 0.1,
			},
			event:        &analytics.NotificationEvent{},
			shouldSample: false,
		},
		{
			name:            "Filter on Account",
			randomGenerator: randomutil.RandomNumberGenerator{},
			feature: config.AnalyticsFeature{
				SampleRate: 1,
				Filter:     "Account.ID == \"123\"",
			},
			event: &analytics.NotificationEvent{
				Account: &config.Account{
					ID: "123",
				},
			},
			shouldSample: true,
		},
		{
			name:            "Filter on Request.AccountID",
			randomGenerator: randomutil.RandomNumberGenerator{},
			feature: config.AnalyticsFeature{
				SampleRate: 1,
				Filter:     "Request.AccountID == \"123\"",
			},
			event: &analytics.NotificationEvent{
				Request: &analytics.EventRequest{
					AccountID: "123",
				},
			},
			shouldSample: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filter, err := createNotificationFilter(tc.feature, tc.randomGenerator)
			assert.NoError(t, err)

			gotResult := filter(tc.event)
			assert.Equal(t, tc.shouldSample, gotResult)
		})
	}
}

func TestCreateSetUIDFilter(t *testing.T) {
	testCases := []struct {
		name            string
		randomGenerator randomutil.RandomGenerator
		feature         config.AnalyticsFeature
		event           *analytics.SetUIDObject

		shouldSample bool
	}{
		{
			name:            "Test with nil event",
			randomGenerator: randomutil.RandomNumberGenerator{},
			feature: config.AnalyticsFeature{
				SampleRate: 1,
			},
			event:        nil,
			shouldSample: false,
		},
		{
			name:            "Test with SampleRate 0",
			randomGenerator: randomutil.RandomNumberGenerator{},
			feature: config.AnalyticsFeature{
				SampleRate: 0,
			},
			event:        &analytics.SetUIDObject{},
			shouldSample: false,
		},
		{
			name:            "Should not sample when the random number is greater than the sample rate",
			randomGenerator: FakeRandomNumberGenerator{Number: 0.2},
			feature: config.AnalyticsFeature{
				SampleRate: 0.1,
			},
			event:        &analytics.SetUIDObject{},
			shouldSample: false,
		},
		{
			name:            "Sample everything with 1",
			randomGenerator: randomutil.RandomNumberGenerator{},
			feature: config.AnalyticsFeature{
				SampleRate: 1,
			},
			event:        &analytics.SetUIDObject{},
			shouldSample: true,
		},
		{
			name:            "Filter on Bidder",
			randomGenerator: randomutil.RandomNumberGenerator{},
			feature: config.AnalyticsFeature{
				SampleRate: 1,
				Filter:     "Bidder == \"123\"",
			},
			event: &analytics.SetUIDObject{
				Bidder: "123",
			},
			shouldSample: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filter, err := createSetUIDFilter(tc.feature, tc.randomGenerator)
			assert.NoError(t, err)

			gotResult := filter(tc.event)
			assert.Equal(t, tc.shouldSample, gotResult)
		})
	}
}

func TestCreateVideoFilter(t *testing.T) {
	testCases := []struct {
		name            string
		randomGenerator randomutil.RandomGenerator
		feature         config.AnalyticsFeature
		event           *analytics.VideoObject
		shouldSample    bool
	}{
		{
			name:            "Test with nil event",
			randomGenerator: randomutil.RandomNumberGenerator{},
			feature: config.AnalyticsFeature{
				SampleRate: 1,
			},
			event:        nil,
			shouldSample: false,
		},
		{
			name:            "Test with SampleRate 0",
			randomGenerator: randomutil.RandomNumberGenerator{},
			feature: config.AnalyticsFeature{
				SampleRate: 0,
			},
			event:        &analytics.VideoObject{},
			shouldSample: false,
		},
		{
			name:            "Sample everything with 1",
			randomGenerator: randomutil.RandomNumberGenerator{},
			feature: config.AnalyticsFeature{
				SampleRate: 1,
			},
			event:        &analytics.VideoObject{},
			shouldSample: true,
		},
		{
			name:            "Should not sample when the random number is greater than the sample rate",
			randomGenerator: FakeRandomNumberGenerator{Number: 0.2},
			feature: config.AnalyticsFeature{
				SampleRate: 0.1,
			},
			event:        &analytics.VideoObject{},
			shouldSample: false,
		},
		{
			name:            "Filter on RequestWrapper.BidRequest.Site.ID",
			randomGenerator: randomutil.RandomNumberGenerator{},
			feature: config.AnalyticsFeature{
				SampleRate: 1,
				Filter:     "RequestWrapper.BidRequest.Site.ID == \"123\"",
			},
			event: &analytics.VideoObject{
				RequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						ID: "123",
					},
				}},
			},
			shouldSample: true,
		},
		{
			name:            "Filter on VideoRequest.Video.MinDuration",
			randomGenerator: randomutil.RandomNumberGenerator{},
			feature: config.AnalyticsFeature{
				SampleRate: 1,
				Filter:     "VideoRequest.Video.MinDuration > 200",
			},
			event: &analytics.VideoObject{
				VideoRequest: &openrtb_ext.BidRequestVideo{
					Video: &openrtb2.Video{
						MinDuration: 201,
					},
				},
			},
			shouldSample: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filter, err := createVideoFilter(tc.feature, tc.randomGenerator)
			assert.NoError(t, err)

			gotResult := filter(tc.event)
			assert.Equal(t, tc.shouldSample, gotResult)
		})
	}
}
