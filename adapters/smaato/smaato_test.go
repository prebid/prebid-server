package smaato

import (
	"encoding/json"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderSmaato, config.Adapter{
		Endpoint: "https://prebid/bidder"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapter, _ := bidder.(*adapter)
	assert.NotNil(t, adapter.clock)
	adapter.clock = &mockTime{time: time.Date(2021, 6, 25, 10, 00, 0, 0, time.UTC)}

	adapterstest.RunJSONBidderTest(t, "smaatotest", bidder)
}

func TestVideoWithCategoryAndDuration(t *testing.T) {
	bidder := &adapter{}

	mockedReq := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{{
			ID: "1_1",
			Video: &openrtb2.Video{
				W:           640,
				H:           360,
				MIMEs:       []string{"video/mp4"},
				MaxDuration: 60,
				Protocols:   []openrtb2.Protocol{2, 3, 5, 6},
			},
			Ext: json.RawMessage(
				`{
					"bidder": {
						"publisherId": "12345"
						"adbreakId": "4123456"
					}
				}`,
			)},
		},
	}
	mockedExtReq := &adapters.RequestData{}
	mockedBidResponse := &openrtb2.BidResponse{
		ID: "some-id",
		SeatBid: []openrtb2.SeatBid{{
			Seat: "some-seat",
			Bid: []openrtb2.Bid{{
				ID:    "6906aae8-7f74-4edd-9a4f-f49379a3cadd",
				ImpID: "1_1",
				Price: 0.01,
				AdM:   "<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"no\"?><VAST version=\"2.0\"></VAST>",
				Cat:   []string{"IAB1"},
				Ext: json.RawMessage(
					`{
						"duration": 5
					}`,
				),
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
	expectedBidDuration := 5
	expectedBidCategory := "IAB1"
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
	if bidResponse.Bids[0].BidVideo.PrimaryCategory != expectedBidCategory {
		t.Errorf("bid category should be set")
	}
	if bidResponse.Bids[0].Bid.Cat[0] != expectedBidCategory {
		t.Errorf("bid category should be set")
	}
	if len(errors) != expectedErrorCount {
		t.Errorf("should not have any errors, errors=%v", errors)
	}
}

type mockTime struct {
	time time.Time
}

func (mt *mockTime) Now() time.Time {
	return mt.time
}
