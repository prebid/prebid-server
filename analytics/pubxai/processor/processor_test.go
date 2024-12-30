package processor

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"
	utils "github.com/prebid/prebid-server/v3/analytics/pubxai/utils"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestProcessLogData(t *testing.T) {
	requestData, err := os.ReadFile("./mocks/mock_openrtb_request.json")
	if err != nil {
		panic(err)
	}
	var bidRequest openrtb2.BidRequest
	if err := json.Unmarshal(requestData, &bidRequest); err != nil {
		panic(err)
	}
	winningBidResponseData, err := os.ReadFile("./mocks/mock_openrtb_response_with_winningbid.json")
	if err != nil {
		panic(err)
	}
	var winningBidResponse openrtb2.BidResponse
	if err := json.Unmarshal(winningBidResponseData, &winningBidResponse); err != nil {
		panic(err)
	}
	nonWinningBidResponseData, err := os.ReadFile("./mocks/mock_openrtb_response_without_winningbid.json")
	if err != nil {
		panic(err)
	}
	var nonWinningBidResponse openrtb2.BidResponse
	if err := json.Unmarshal(nonWinningBidResponseData, &nonWinningBidResponse); err != nil {
		panic(err)
	}
	tests := []struct {
		name                string
		logObject           *utils.LogObject
		expectedAuctionBids int
		expectedWinningBids int
	}{
		{
			name:                "NilAuctionObject",
			logObject:           nil,
			expectedAuctionBids: 0,
			expectedWinningBids: 0,
		},
		{
			name:                "NilRequestWrapper",
			logObject:           &utils.LogObject{},
			expectedAuctionBids: 0,
			expectedWinningBids: 0,
		},
		{
			name: "NoImpressions",
			logObject: &utils.LogObject{
				RequestWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{},
				},
			},
			expectedAuctionBids: 0,
			expectedWinningBids: 0,
		},
		{
			name: "UnmarshalExtensionsFailed",
			logObject: &utils.LogObject{
				RequestWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						Imp: []openrtb2.Imp{{ID: "imp1"}},
					},
				},
				Response: &openrtb2.BidResponse{
					SeatBid: []openrtb2.SeatBid{
						{
							Seat: "bidder1",
							Bid:  []openrtb2.Bid{{ImpID: "imp1"}},
						},
					},
				},
				StartTime: time.Now(),
			},
			expectedAuctionBids: 0,
			expectedWinningBids: 0,
		},
		{
			name: "Success",
			logObject: &utils.LogObject{
				RequestWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &bidRequest,
				},
				Response:  &winningBidResponse,
				StartTime: time.Now(),
			},
			expectedAuctionBids: 1,
			expectedWinningBids: 1,
		},
		{
			name: "SuccessWithoutWinningBid",
			logObject: &utils.LogObject{
				RequestWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &bidRequest,
				},
				Response:  &nonWinningBidResponse,
				StartTime: time.Now(),
			},
			expectedAuctionBids: 1,
			expectedWinningBids: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processorService := &ProcessorServiceImpl{}

			auctionBids, winningBids := processorService.ProcessLogData(tt.logObject)
			fmt.Println("name", tt.name)
			// Use zero value if auctionBids is nil
			bidsLength := 0
			if auctionBids != nil {
				bidsLength = len(auctionBids.Bids)
			}
			fmt.Println("auctionBids", bidsLength, tt.expectedAuctionBids)
			fmt.Println("winningBids", len(winningBids), tt.expectedWinningBids)
			assert.Equal(t, tt.expectedAuctionBids, bidsLength)
			assert.Equal(t, tt.expectedWinningBids, len(winningBids))
		})
	}
}
