package hookstage

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
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
			name:       "two-allowed-bidders-two-imp-bidders-one-mutation",
			bidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "ImpA"}}},
			allowedBidders: []map[string]struct{}{
				{"bidderA": {}, "bidderB": {}}, // one mutation for two bidders
			},
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
			name:       "two-imp-bidders-two-mutations-override-all",
			bidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "ImpA"}}},
			allowedBidders: []map[string]struct{}{
				{"bidderA": {}}, // first mutation for bidderA
				{"bidderB": {}}, // second mutation for bidderB
			},
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

type recordedBidderRemoval struct{ impID, bidder, reason string }

type bidderRemovalRecorder []recordedBidderRemoval

func (r *bidderRemovalRecorder) ReportBidderRemoval(impID, bidder, reason string) {
	*r = append(*r, recordedBidderRemoval{impID, bidder, reason})
}

func TestBidderMutationsReportRemovals(t *testing.T) {
	tests := []struct {
		name        string
		addMutation func(ChangeBidders[ProcessedAuctionRequestPayload])
		reporter    bool
		want        []recordedBidderRemoval
	}{
		{
			name: "delete-reports-removed-bidder-and-reason",
			addMutation: func(b ChangeBidders[ProcessedAuctionRequestPayload]) {
				b.DeleteWithReason(map[string]struct{}{"bidderB": {}}, "excludeBidders")
			},
			reporter: true,
			want:     []recordedBidderRemoval{{"imp1", "bidderB", "excludeBidders"}},
		},
		{
			name: "add-reports-bidders-not-allowed",
			addMutation: func(b ChangeBidders[ProcessedAuctionRequestPayload]) {
				b.AddWithReason(map[string]struct{}{"bidderA": {}}, "includeBidders")
			},
			reporter: true,
			want:     []recordedBidderRemoval{{"imp1", "bidderB", "includeBidders"}},
		},
		{
			name: "existing-delete-reports-without-reason",
			addMutation: func(b ChangeBidders[ProcessedAuctionRequestPayload]) {
				b.Delete(map[string]struct{}{"bidderB": {}})
			},
			reporter: true,
			want:     []recordedBidderRemoval{{"imp1", "bidderB", ""}},
		},
		{
			name: "no-reporter-still-removes",
			addMutation: func(b ChangeBidders[ProcessedAuctionRequestPayload]) {
				b.DeleteWithReason(map[string]struct{}{"bidderB": {}}, "excludeBidders")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			wrapper := &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "imp1"}}}}
			impExt, err := wrapper.GetImp()[0].GetImpExt()
			assert.NoError(t, err)
			impExt.SetPrebid(&openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderA": json.RawMessage(`{}`),
				"bidderB": json.RawMessage(`{}`),
			}})

			var removals bidderRemovalRecorder
			payload := ProcessedAuctionRequestPayload{Request: wrapper}
			if test.reporter {
				payload.SetBidderRemovalReporter(&removals)
			}
			changeSet := &ChangeSet[ProcessedAuctionRequestPayload]{}
			test.addMutation(changeSet.ProcessedAuctionRequest().Bidders())

			for _, mutation := range changeSet.Mutations() {
				_, err = mutation.Apply(payload)
				assert.NoError(t, err)
			}

			assert.Equal(t, []recordedBidderRemoval(test.want), []recordedBidderRemoval(removals))
			updatedExt, err := wrapper.GetImp()[0].GetImpExt()
			assert.NoError(t, err)
			assert.Equal(t, map[string]json.RawMessage{"bidderA": json.RawMessage(`{}`)}, updatedExt.GetPrebid().Bidder)
		})
	}
}
