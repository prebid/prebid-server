package aps

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApsFormatDefaultExp(t *testing.T) {
	tests := []struct {
		name    string
		imp     *openrtb2.Imp
		wantExp int64
		wantOK  bool
	}{
		{
			name:    "nil_imp",
			imp:     nil,
			wantExp: 0,
			wantOK:  false,
		},
		{
			name:    "banner_only",
			imp:     &openrtb2.Imp{Banner: &openrtb2.Banner{}},
			wantExp: apsBannerExpSeconds,
			wantOK:  true,
		},
		{
			name:    "video_only",
			imp:     &openrtb2.Imp{Video: &openrtb2.Video{}},
			wantExp: apsVideoExpSeconds,
			wantOK:  true,
		},
		{
			name:    "banner_and_video",
			imp:     &openrtb2.Imp{Banner: &openrtb2.Banner{}, Video: &openrtb2.Video{}},
			wantExp: 0,
			wantOK:  false,
		},
		{
			name:    "neither",
			imp:     &openrtb2.Imp{ID: "1"},
			wantExp: 0,
			wantOK:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotExp, gotOK := apsFormatDefaultExp(tt.imp)
			assert.Equal(t, tt.wantOK, gotOK)
			assert.Equal(t, tt.wantExp, gotExp)
		})
	}
}

func TestSetAPSImpExpIfMissing(t *testing.T) {
	tests := []struct {
		name    string
		imp     *openrtb2.Imp
		wantExp int64
	}{
		{
			name:    "banner_only_sets_600_when_exp_missing",
			imp:     &openrtb2.Imp{ID: "imp-1", Banner: &openrtb2.Banner{}},
			wantExp: 600,
		},
		{
			name:    "video_only_sets_3600_when_exp_missing",
			imp:     &openrtb2.Imp{ID: "imp-1", Video: &openrtb2.Video{}},
			wantExp: 3600,
		},
		{
			name:    "banner_and_video_no_default",
			imp:     &openrtb2.Imp{ID: "imp-1", Banner: &openrtb2.Banner{}, Video: &openrtb2.Video{}},
			wantExp: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setAPSImpExpIfMissing(tt.imp)
			assert.Equal(t, tt.wantExp, tt.imp.Exp)
		})
	}
}

func TestApplyAPSBidExpIfMissing(t *testing.T) {
	rctx := models.RequestCtx{
		ImpBidCtx: map[string]models.ImpCtx{
			"imp-1": {Exp: 600},
			"imp-2": {Exp: 3600},
		},
	}

	bidResponse := &openrtb2.BidResponse{
		SeatBid: []openrtb2.SeatBid{{
			Bid: []openrtb2.Bid{
				{ID: "b1", ImpID: "imp-1", Exp: 0},
				{ID: "b2", ImpID: "imp-2", Exp: 120},
				{ID: "b3", ImpID: "imp-3", Exp: 0},
			},
		}},
	}

	applyAPSBidExpIfMissing(rctx, bidResponse)

	assert.Equal(t, int64(600), bidResponse.SeatBid[0].Bid[0].Exp)
	assert.Equal(t, int64(120), bidResponse.SeatBid[0].Bid[1].Exp)
	assert.Equal(t, int64(0), bidResponse.SeatBid[0].Bid[2].Exp)
}

// syncImpExpFromRequest mirrors before_validation copying outbound imp.exp into ImpBidCtx.
func syncImpExpFromRequest(bidRequest *openrtb2.BidRequest, rctx *models.RequestCtx) {
	if rctx.ImpBidCtx == nil {
		rctx.ImpBidCtx = make(map[string]models.ImpCtx)
	}
	for i := range bidRequest.Imp {
		imp := bidRequest.Imp[i]
		impCtx, ok := rctx.ImpBidCtx[imp.ID]
		if !ok {
			impCtx = models.ImpCtx{ImpID: imp.ID}
		}
		impCtx.Exp = imp.Exp
		rctx.ImpBidCtx[imp.ID] = impCtx
	}
}

func TestAPSExpRequestToResponseFlow(t *testing.T) {
	tests := []struct {
		name        string
		requestBody []byte
		wantImpExp  int64
	}{
		{
			name:        "banner_without_exp_sets_bid_exp_600",
			requestBody: []byte(`{"id":"r1","imp":[{"id":"i1","tagid":"t1","banner":{"w":300,"h":250}}],"app":{"publisher":{"id":"pub"}}}`),
			wantImpExp:  600,
		},
		{
			name:        "s2s_exp_preserved_sets_bid_exp_120",
			requestBody: []byte(`{"id":"r1","imp":[{"id":"i1","tagid":"t1","exp":120,"banner":{"w":300,"h":250}}],"app":{"publisher":{"id":"pub"}}}`),
			wantImpExp:  120,
		},
		{
			name:        "video_only_without_exp_sets_bid_exp_3600",
			requestBody: []byte(`{"id":"r1","imp":[{"id":"i1","tagid":"t1","video":{"mimes":["video/mp4"]}}],"app":{"publisher":{"id":"pub"}}}`),
			wantImpExp:  3600,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modified := NewAPS(nil).ModifyRequestWithAPSParams(tt.requestBody, &models.RequestCtx{})

			var bidRequest openrtb2.BidRequest
			require.NoError(t, json.Unmarshal(modified, &bidRequest))
			require.Len(t, bidRequest.Imp, 1)
			assert.Equal(t, tt.wantImpExp, bidRequest.Imp[0].Exp)

			rctx := models.RequestCtx{
				Endpoint:     models.EndpointAPS,
				PubIDStr:     "pub",
				ProfileIDStr: "1",
				ImpBidCtx:    make(map[string]models.ImpCtx),
			}
			syncImpExpFromRequest(&bidRequest, &rctx)

			br := &openrtb2.BidResponse{
				ID:  "resp",
				Cur: "USD",
				SeatBid: []openrtb2.SeatBid{{
					Bid: []openrtb2.Bid{{
						ID:    "b1",
						ImpID: bidRequest.Imp[0].ID,
						Price: 1.0,
						AdM:   "<html/>",
					}},
				}},
			}

			out := ApplyAPSResponse(rctx, br)
			require.Len(t, out.SeatBid, 1)
			require.Len(t, out.SeatBid[0].Bid, 1)
			assert.Equal(t, tt.wantImpExp, out.SeatBid[0].Bid[0].Exp)
		})
	}
}
