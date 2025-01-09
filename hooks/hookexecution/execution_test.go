package hookexecution

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/privacy"
	"github.com/stretchr/testify/assert"
)

const (
	testIpv6                = "1111:2222:3333:4444:5555:6666:7777:8888"
	testIPv6Scrubbed        = "1111:2222::"
	testIPv6ScrubbedDefault = "1111:2222:3333:4400::"
	testIPv6ScrubBytes      = 32
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
		{
			description: "payload should change when transmitPreciseGeo is blocked by activity",
			hookCode:    "foo",
			inPayloadData: hookstage.BidderRequestPayload{
				Request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{IPv6: testIpv6},
				}},
			},
			privacyConfig: getTransmitPreciseGeoActivityConfig("foo", false),
			expectedPayloadData: hookstage.BidderRequestPayload{
				Request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{IPv6: testIPv6ScrubbedDefault},
				},
				}},
		},
		{
			description: "payload should not change when transmitPreciseGeo is not blocked by activity",
			hookCode:    "foo",
			inPayloadData: hookstage.BidderRequestPayload{
				Request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{IPv6: testIpv6},
				}},
			},
			privacyConfig: getTransmitPreciseGeoActivityConfig("foo", true),
			expectedPayloadData: hookstage.BidderRequestPayload{
				Request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{IPv6: testIpv6},
				}},
			},
		},
	}
	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			//check input payload didn't change
			origInPayloadData := test.inPayloadData
			activityControl := privacy.NewActivityControl(test.privacyConfig)
			newPayload := handleModuleActivities(test.hookCode, activityControl, test.inPayloadData, nil)
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

		{
			description: "payload should change when transmitPreciseGeo is blocked by activity",
			hookCode:    "foo",
			inPayloadData: hookstage.ProcessedAuctionRequestPayload{
				Request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{IPv6: testIpv6},
				}},
			},
			privacyConfig: getTransmitPreciseGeoActivityConfig("foo", false),
			expectedPayloadData: hookstage.ProcessedAuctionRequestPayload{
				Request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{IPv6: testIPv6Scrubbed},
				}},
			},
		},
		{
			description: "payload should not change when transmitPreciseGeo is not blocked by activity",
			hookCode:    "foo",
			inPayloadData: hookstage.ProcessedAuctionRequestPayload{
				Request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{IPv6: testIpv6},
				}},
			},
			privacyConfig: getTransmitPreciseGeoActivityConfig("foo", true),
			expectedPayloadData: hookstage.ProcessedAuctionRequestPayload{
				Request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{IPv6: testIpv6},
				}},
			},
		},
	}
	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			//check input payload didn't change
			origInPayloadData := test.inPayloadData
			activityControl := privacy.NewActivityControl(test.privacyConfig)
			account := &config.Account{Privacy: config.AccountPrivacy{IPv6Config: config.IPv6{AnonKeepBits: testIPv6ScrubBytes}}}
			newPayload := handleModuleActivities(test.hookCode, activityControl, test.inPayloadData, account)
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
			description:         "payload should not change when userFPD is blocked by activity",
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
		{
			description:         "payload should not change when transmitPreciseGeo is blocked by activity",
			hookCode:            "foo",
			inPayloadData:       hookstage.RawAuctionRequestPayload{},
			privacyConfig:       getTransmitPreciseGeoActivityConfig("foo", false),
			expectedPayloadData: hookstage.RawAuctionRequestPayload{},
		},
		{
			description:         "payload should not change when transmitPreciseGeo is not blocked by activity",
			hookCode:            "foo",
			inPayloadData:       hookstage.RawAuctionRequestPayload{},
			privacyConfig:       getTransmitPreciseGeoActivityConfig("foo", true),
			expectedPayloadData: hookstage.RawAuctionRequestPayload{},
		},
	}
	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			//check input payload didn't change
			origInPayloadData := test.inPayloadData
			activityControl := privacy.NewActivityControl(test.privacyConfig)
			newPayload := handleModuleActivities(test.hookCode, activityControl, test.inPayloadData, &config.Account{})
			assert.Equal(t, test.expectedPayloadData, newPayload)
			assert.Equal(t, origInPayloadData, test.inPayloadData)
		})
	}
}
