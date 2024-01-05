package hookexecution

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/hooks/hookstage"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/prebid/prebid-server/v2/privacy"
	"github.com/stretchr/testify/assert"
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
				Request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{ID: "test_user_id"},
				}},
			},
			privacyConfig: getTransmitUFPDActivityConfig("foo", false),
			expectedPayloadData: hookstage.BidderRequestPayload{
				Request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{ID: ""},
				},
				}},
		},
		{
			description: "payload should not change when userFPD is not blocked by activity",
			hookCode:    "foo",
			inPayloadData: hookstage.BidderRequestPayload{
				Request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{ID: "test_user_id"},
				}},
			},
			privacyConfig: getTransmitUFPDActivityConfig("foo", true),
			expectedPayloadData: hookstage.BidderRequestPayload{
				Request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{ID: "test_user_id"},
				}},
			},
		},
	}
	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			//check input payload didn't change
			origInPayloadData := test.inPayloadData
			activityControl := privacy.NewActivityControl(test.privacyConfig)
			newPayload := handleModuleActivities(test.hookCode, activityControl, test.inPayloadData)
			assert.Equal(t, test.expectedPayloadData.Request.BidRequest, newPayload.Request.BidRequest)
			assert.Equal(t, origInPayloadData, test.inPayloadData)
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
				Request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{ID: "test_user_id"},
				}},
			},
			privacyConfig: getTransmitUFPDActivityConfig("foo", false),
			expectedPayloadData: hookstage.ProcessedAuctionRequestPayload{
				Request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{ID: ""},
				}},
			},
		},
		{
			description: "payload should not change when userFPD is not blocked by activity",
			hookCode:    "foo",
			inPayloadData: hookstage.ProcessedAuctionRequestPayload{
				Request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{ID: "test_user_id"},
				}},
			},
			privacyConfig: getTransmitUFPDActivityConfig("foo", true),
			expectedPayloadData: hookstage.ProcessedAuctionRequestPayload{
				Request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{ID: "test_user_id"},
				}},
			},
		},
	}
	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			//check input payload didn't change
			origInPayloadData := test.inPayloadData
			activityControl := privacy.NewActivityControl(test.privacyConfig)
			newPayload := handleModuleActivities(test.hookCode, activityControl, test.inPayloadData)
			assert.Equal(t, test.expectedPayloadData.Request.BidRequest, newPayload.Request.BidRequest)
			assert.Equal(t, origInPayloadData, test.inPayloadData)
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
		t.Run(test.description, func(t *testing.T) {
			//check input payload didn't change
			origInPayloadData := test.inPayloadData
			activityControl := privacy.NewActivityControl(test.privacyConfig)
			newPayload := handleModuleActivities(test.hookCode, activityControl, test.inPayloadData)
			assert.Equal(t, test.expectedPayloadData, newPayload)
			assert.Equal(t, origInPayloadData, test.inPayloadData)
		})
	}
}
