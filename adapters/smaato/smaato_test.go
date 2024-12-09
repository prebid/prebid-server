package smaato

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderSmaato, config.Adapter{
		Endpoint: "https://prebid/bidder"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

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
				W:           ptrutil.ToPtr[int64](640),
				H:           ptrutil.ToPtr[int64](360),
				MIMEs:       []string{"video/mp4"},
				MaxDuration: 60,
				Protocols:   []adcom1.MediaCreativeSubtype{2, 3, 5, 6},
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
	headers := http.Header{}
	headers.Add("X-Smt-Adtype", "Video")
	mockedRes := &adapters.ResponseData{
		StatusCode: 200,
		Body:       body,
		Headers:    headers,
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
