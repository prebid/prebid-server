package hookexecution

import (
	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/hooks/hookstage"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTransmitUFPDMutationUser(t *testing.T) {
	testBidderReqPayload := hookstage.BidderRequestPayload{
		BidRequest: &openrtb2.BidRequest{
			ID:     "ID1",
			User:   &openrtb2.User{ID: "UserId1"},
			Device: &openrtb2.Device{IFA: "DeviceIFA"},
		},
		Bidder: "BidderA",
	}

	resultPayload, err := transmitUFPDMutationUser(testBidderReqPayload)
	assert.NoError(t, err)
	finalResultPayload, err := transmitUFPDMutationDevice(resultPayload)
	assert.NoError(t, err)

	assert.Equal(t, testBidderReqPayload.GetBidderRequestPayload().ID, "ID1")
	assert.Equal(t, testBidderReqPayload.GetBidderRequestPayload().User.ID, "UserId1")
	assert.Equal(t, testBidderReqPayload.GetBidderRequestPayload().Device.IFA, "DeviceIFA")
	assert.Equal(t, finalResultPayload.GetBidderRequestPayload().ID, "ID1")
	assert.Equal(t, finalResultPayload.GetBidderRequestPayload().User.ID, "")
	assert.Equal(t, finalResultPayload.GetBidderRequestPayload().Device.IFA, "")
}
