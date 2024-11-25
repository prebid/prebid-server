package consumable

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderConsumable, config.Adapter{
		Endpoint: "https://e.serverbid.com",
	}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "consumable", bidder)
}

func TestConsumableMakeBidsWithCategoryDuration(t *testing.T) {
	bidder := &adapter{}

	mockedReq := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{
			{
				ID: "1_1",
				Video: &openrtb2.Video{
					W:           ptrutil.ToPtr[int64](640),
					H:           ptrutil.ToPtr[int64](360),
					MIMEs:       []string{"video/mp4"},
					MaxDuration: 60,
					Protocols:   []adcom1.MediaCreativeSubtype{2, 3, 5, 6},
				},
				Ext: json.RawMessage(
					`{
					"prebid": {},
					"bidder": {
						"placementId": "123456"
					}
				}`,
				),
			},
		},
	}
	mockedExtReq := &adapters.RequestData{}
	mockedBidResponse := &openrtb2.BidResponse{
		ID: "test-1",
		SeatBid: []openrtb2.SeatBid{{
			Seat: "Buyer",
			Bid: []openrtb2.Bid{{
				ID:    "1",
				ImpID: "1_1",
				Price: 1.23,
				AdID:  "123",
				Cat:   []string{"IAB18-1"},
				Dur:   30,
				MType: openrtb2.MarkupVideo,
			}},
		}},
	}
	body, _ := json.Marshal(mockedBidResponse)
	mockedRes := &adapters.ResponseData{
		StatusCode: 200,
		Body:       body,
	}

	expectedBidCount := 1
	expectedBidType := openrtb_ext.BidTypeVideo
	expectedBidDuration := 30
	expectedBidCategory := "IAB18-1"
	expectedErrorCount := 0

	bidResponse, errors := bidder.MakeBids(mockedReq, mockedExtReq, mockedRes)

	if len(bidResponse.Bids) != expectedBidCount {
		t.Errorf("should have 1 bid, bids=%v", bidResponse.Bids)
	}
	if bidResponse.Bids[0].BidType != expectedBidType {
		t.Errorf("bid type should be video, bidType=%s", bidResponse.Bids[0].BidType)
	}
	if bidResponse.Bids[0].BidVideo.Duration != expectedBidDuration {
		t.Errorf("video duration should be set")
	}
	if bidResponse.Bids[0].Bid.Cat[0] != expectedBidCategory {
		t.Errorf("bid category should be set")
	}
	if len(errors) != expectedErrorCount {
		t.Errorf("should not have any errors, errors=%v", errors)
	}
}
