package hookexecution

import (
	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/hooks"
	"github.com/prebid/prebid-server/hooks/hookstage"
	"github.com/prebid/prebid-server/privacy"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHandleModuleActivitiesBidderRequestPayload(t *testing.T) {

	testCases := []struct {
		description         string
		hookCode            string
		privacyConfig       *config.AccountPrivacy
		inPayloadData       hookstage.BidderRequestPayload
		expectedPayloadData hookstage.BidderRequestPayload
	}{
		{
			description: "payload should change when userFPD is blocked by activity",
			hookCode:    "foo",
			inPayloadData: hookstage.BidderRequestPayload{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{ID: "test_user_id"},
				},
			},
			privacyConfig: getTransmitUFPDActivityConfig("foo", false),
			expectedPayloadData: hookstage.BidderRequestPayload{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{ID: ""},
				},
			},
		},
		{
			description: "payload should not change when userFPD is not blocked by activity",
			hookCode:    "foo",
			inPayloadData: hookstage.BidderRequestPayload{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{ID: "test_user_id"},
				},
			},
			privacyConfig: getTransmitUFPDActivityConfig("foo", true),
			expectedPayloadData: hookstage.BidderRequestPayload{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{ID: "test_user_id"},
				},
			},
		},
	}
	for _, test := range testCases {
		activityControl, err := privacy.NewActivityControl(test.privacyConfig)
		assert.NoError(t, err)
		t.Run(test.description, func(t *testing.T) {
			newPayload := runHandleModuleActivities(test.hookCode, activityControl, test.inPayloadData)
			assert.Equal(t, test.expectedPayloadData, newPayload)
		})
	}
}

func TestHandleModuleActivitiesProcessedAuctionRequestPayload(t *testing.T) {

	testCases := []struct {
		description         string
		hookCode            string
		privacyConfig       *config.AccountPrivacy
		inPayloadData       hookstage.ProcessedAuctionRequestPayload
		expectedPayloadData hookstage.ProcessedAuctionRequestPayload
	}{
		{
			description: "payload should change when userFPD is blocked by activity",
			hookCode:    "foo",
			inPayloadData: hookstage.ProcessedAuctionRequestPayload{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{ID: "test_user_id"},
				},
			},
			privacyConfig: getTransmitUFPDActivityConfig("foo", false),
			expectedPayloadData: hookstage.ProcessedAuctionRequestPayload{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{ID: ""},
				},
			},
		},
		{
			description: "payload should not change when userFPD is not blocked by activity",
			hookCode:    "foo",
			inPayloadData: hookstage.ProcessedAuctionRequestPayload{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{ID: "test_user_id"},
				},
			},
			privacyConfig: getTransmitUFPDActivityConfig("foo", true),
			expectedPayloadData: hookstage.ProcessedAuctionRequestPayload{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{ID: "test_user_id"},
				},
			},
		},
	}
	for _, test := range testCases {
		activityControl, err := privacy.NewActivityControl(test.privacyConfig)
		assert.NoError(t, err)
		t.Run(test.description, func(t *testing.T) {
			newPayload := runHandleModuleActivities(test.hookCode, activityControl, test.inPayloadData)
			assert.Equal(t, test.expectedPayloadData, newPayload)
		})
	}
}

func TestHandleModuleActivitiesNoBidderRequestPayload(t *testing.T) {

	testCases := []struct {
		description         string
		hookCode            string
		privacyConfig       *config.AccountPrivacy
		inPayloadData       hookstage.RawAuctionRequestPayload
		expectedPayloadData hookstage.RawAuctionRequestPayload
	}{
		{
			description:         "payload should change when userFPD is blocked by activity",
			hookCode:            "foo",
			inPayloadData:       hookstage.RawAuctionRequestPayload{},
			privacyConfig:       getTransmitUFPDActivityConfig("foo", false),
			expectedPayloadData: hookstage.RawAuctionRequestPayload{},
		},
		{
			description:         "payload should not change when userFPD is not blocked by activity",
			hookCode:            "foo",
			inPayloadData:       hookstage.RawAuctionRequestPayload{},
			privacyConfig:       getTransmitUFPDActivityConfig("foo", true),
			expectedPayloadData: hookstage.RawAuctionRequestPayload{},
		},
	}
	for _, test := range testCases {
		activityControl, err := privacy.NewActivityControl(test.privacyConfig)
		assert.NoError(t, err)
		t.Run(test.description, func(t *testing.T) {
			newPayload := runHandleModuleActivities(test.hookCode, activityControl, test.inPayloadData)
			assert.Equal(t, test.expectedPayloadData, newPayload)
		})
	}
}

func runHandleModuleActivities[P any](hookCode string, activityControl privacy.ActivityControl, payload P) P {
	hook := []hooks.HookWrapper[hookstage.BidderRequestPayload]{
		{Module: "foobar", Code: hookCode},
	}
	newPayload := handleModuleActivities(hook[0], activityControl, payload)
	return newPayload
}
