package hookstage

import (
	"encoding/json"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAddPrebidBidders(t *testing.T) {
	tests := []struct {
		name           string
		bidRequest     *openrtb2.BidRequest
		allowedBidders []map[string]struct{} // list to allow multiple mutations
		extImpPrebid   *openrtb_ext.ExtImpPrebid
		expectData     openrtb_ext.ExtImpPrebid
	}{

		{
			name:           "none-allowed-bidder-two-imp-bidders-one-mutation",
			bidRequest:     &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "ImpA"}}},
			allowedBidders: []map[string]struct{}{{}},
			extImpPrebid: &openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderA": json.RawMessage(`{"paramA": "valueA"}`),
				"bidderB": json.RawMessage(`{"paramB": "valueB"}`),
			}},
			expectData: openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{}},
		},
		{
			name:           "one-allowed-bidder-one-imp-bidder-one-mutation-match",
			bidRequest:     &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "ImpA"}}},
			allowedBidders: []map[string]struct{}{{"bidderB": {}}},
			extImpPrebid: &openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderB": json.RawMessage(`{"paramB": "valueB"}`),
			}},
			expectData: openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderB": json.RawMessage(`{"paramB": "valueB"}`),
			}},
		},
		{
			name:           "one-allowed-bidder-one-imp-bidder-one-mutation-different",
			bidRequest:     &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "ImpA"}}},
			allowedBidders: []map[string]struct{}{{"bidderA": {}}},
			extImpPrebid: &openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderB": json.RawMessage(`{"paramB": "valueB"}`),
			}},
			expectData: openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{}},
		},
		{
			name:           "one-allowed-bidder-two-imp-bidders-one-mutation",
			bidRequest:     &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "ImpA"}}},
			allowedBidders: []map[string]struct{}{{"bidderB": {}}},
			extImpPrebid: &openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderA": json.RawMessage(`{"paramA": "valueA"}`),
				"bidderB": json.RawMessage(`{"paramB": "valueB"}`),
			}},
			expectData: openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderB": json.RawMessage(`{"paramB": "valueB"}`),
			}},
		},
		{
			name:           "two-allowed-bidders-two-imp-bidders-one-mutation",
			bidRequest:     &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "ImpA"}}},
			allowedBidders: []map[string]struct{}{{"bidderA": {}, "bidderB": {}}},
			extImpPrebid: &openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderA": json.RawMessage(`{"paramA": "valueA"}`),
				"bidderB": json.RawMessage(`{"paramB": "valueB"}`),
			}},
			expectData: openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderA": json.RawMessage(`{"paramA": "valueA"}`),
				"bidderB": json.RawMessage(`{"paramB": "valueB"}`),
			}},
		},
		{
			name:           "two-imp-bidders-two-mutations-override-all",
			bidRequest:     &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "ImpA"}}},
			allowedBidders: []map[string]struct{}{{"bidderA": {}}, {"bidderB": {}}},
			extImpPrebid: &openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderA": json.RawMessage(`{"paramA": "valueA"}`),
				"bidderB": json.RawMessage(`{"paramB": "valueB"}`),
			}},
			expectData: openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			br := &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "ImpA"}}}
			if tt.bidRequest != nil {
				br = tt.bidRequest
			}
			brw := openrtb_ext.RequestWrapper{BidRequest: br}
			impWrapperArr := brw.GetImp()

			if len(impWrapperArr) > 0 {
				impExt, err := brw.GetImp()[0].GetImpExt()
				assert.NoError(t, err)
				impExt.SetPrebid(tt.extImpPrebid)
			}

			payload := ProcessedAuctionRequestPayload{Request: &brw}

			cpar := ChangeSetProcessedAuctionRequest[ProcessedAuctionRequestPayload]{
				changeSet: &ChangeSet[ProcessedAuctionRequestPayload]{},
			}

			for _, allowedBidders := range tt.allowedBidders {
				cpar.Bidders().Add(allowedBidders)
			}

			for _, mut := range cpar.changeSet.Mutations() {
				_, err := mut.Apply(payload)
				assert.NoError(t, err)
			}

			impExtRes, err := payload.Request.GetImp()[0].GetImpExt()
			assert.NoError(t, err)
			assert.Equal(t, &tt.expectData, impExtRes.GetPrebid(), "Bidder data should match expected")

		})
	}

}

func TestDeletePrebidBidders(t *testing.T) {
	tests := []struct {
		name           string
		bidRequest     *openrtb2.BidRequest
		allowedBidders []map[string]struct{} // list to allow multiple mutations
		extImpPrebid   *openrtb_ext.ExtImpPrebid
		expectData     openrtb_ext.ExtImpPrebid
	}{

		{
			name:           "none-allowed-bidder-two-imp-bidders-one-mutation",
			bidRequest:     &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "ImpA"}}},
			allowedBidders: []map[string]struct{}{{}},
			extImpPrebid: &openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderA": json.RawMessage(`{"paramA": "valueA"}`),
				"bidderB": json.RawMessage(`{"paramB": "valueB"}`),
			}},
			expectData: openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderA": json.RawMessage(`{"paramA": "valueA"}`),
				"bidderB": json.RawMessage(`{"paramB": "valueB"}`),
			}},
		},
		{
			name:           "one-allowed-bidder-one-imp-bidder-one-mutation-match",
			bidRequest:     &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "ImpA"}}},
			allowedBidders: []map[string]struct{}{{"bidderB": {}}},
			extImpPrebid: &openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderB": json.RawMessage(`{"paramB": "valueB"}`),
			}},
			expectData: openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{}},
		},
		{
			name:           "one-allowed-bidder-one-imp-bidder-one-mutation-different",
			bidRequest:     &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "ImpA"}}},
			allowedBidders: []map[string]struct{}{{"bidderA": {}}},
			extImpPrebid: &openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderB": json.RawMessage(`{"paramB": "valueB"}`),
			}},
			expectData: openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderB": json.RawMessage(`{"paramB": "valueB"}`),
			}},
		},
		{
			name:           "one-allowed-bidder-two-imp-bidders-one-mutation",
			bidRequest:     &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "ImpA"}}},
			allowedBidders: []map[string]struct{}{{"bidderB": {}}},
			extImpPrebid: &openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderA": json.RawMessage(`{"paramA": "valueA"}`),
				"bidderB": json.RawMessage(`{"paramB": "valueB"}`),
			}},
			expectData: openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderA": json.RawMessage(`{"paramA": "valueA"}`),
			}},
		},
		{
			name:           "two-allowed-bidders-two-imp-bidders-one-mutation",
			bidRequest:     &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "ImpA"}}},
			allowedBidders: []map[string]struct{}{{"bidderA": {}, "bidderB": {}}},
			extImpPrebid: &openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderA": json.RawMessage(`{"paramA": "valueA"}`),
				"bidderB": json.RawMessage(`{"paramB": "valueB"}`),
			}},
			expectData: openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{}},
		},
		{
			name:           "two-imp-bidders-two-mutations-delete-two",
			bidRequest:     &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "ImpA"}}},
			allowedBidders: []map[string]struct{}{{"bidderA": {}}, {"bidderB": {}}},
			extImpPrebid: &openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderA": json.RawMessage(`{"paramA": "valueA"}`),
				"bidderB": json.RawMessage(`{"paramB": "valueB"}`),
				"bidderC": json.RawMessage(`{"paramC": "valueC"}`),
			}},
			expectData: openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderC": json.RawMessage(`{"paramC": "valueC"}`),
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			br := &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "ImpA"}}}
			if tt.bidRequest != nil {
				br = tt.bidRequest
			}
			brw := openrtb_ext.RequestWrapper{BidRequest: br}
			impWrapperArr := brw.GetImp()

			if len(impWrapperArr) > 0 {
				impExt, err := brw.GetImp()[0].GetImpExt()
				assert.NoError(t, err)
				impExt.SetPrebid(tt.extImpPrebid)
			}

			payload := ProcessedAuctionRequestPayload{Request: &brw}

			cpar := ChangeSetProcessedAuctionRequest[ProcessedAuctionRequestPayload]{
				changeSet: &ChangeSet[ProcessedAuctionRequestPayload]{},
			}

			for _, allowedBidders := range tt.allowedBidders {
				cpar.Bidders().Delete(allowedBidders)
			}

			for _, mut := range cpar.changeSet.Mutations() {
				_, err := mut.Apply(payload)
				assert.NoError(t, err)
			}

			impExtRes, err := payload.Request.GetImp()[0].GetImpExt()
			assert.NoError(t, err)
			assert.Equal(t, &tt.expectData, impExtRes.GetPrebid(), "Bidder data should match expected")

		})
	}

}
