package openwrap

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/openrtb/v20/openrtb3"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	mock_cache "github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/cache/mock"
	mock_metrics "github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/metrics/mock"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models/nbr"
	mock_feature "github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/publisherfeature/mock"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

func TestNonBRCodesInHandleAuctionResponseHook(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockFeature := mock_feature.NewMockFeature(ctrl)

	type args struct {
		ctx       context.Context
		moduleCtx hookstage.ModuleInvocationContext
		payload   hookstage.AuctionResponsePayload
	}
	type want struct {
		impBidCtx map[string]models.ImpCtx
	}
	tests := []struct {
		name             string
		args             args
		want             want
		getMetricsEngine func() *mock_metrics.MockMetricsEngine
	}{
		{
			name: "single bid and supportdeal is false",
			args: args{
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							StartTime: time.Now().UnixMilli(),
							ImpBidCtx: map[string]models.ImpCtx{
								"imp1": {},
							},
							PubIDStr: "5890",
						},
					},
				},
				payload: hookstage.AuctionResponsePayload{
					BidResponse: &openrtb2.BidResponse{
						Cur: "USD",
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										Price: 5,
										Ext:   json.RawMessage(`{}`),
									},
								},
								Seat: "pubmatic",
							},
						},
					},
				},
			},
			getMetricsEngine: func() (me *mock_metrics.MockMetricsEngine) {
				mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)
				mockEngine.EXPECT().RecordPublisherResponseTimeStats("5890", gomock.Any())
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("", "5890", "pubmatic")
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
				mockFeature.EXPECT().IsFscApplicable(gomock.Any(), gomock.Any(), gomock.Any()).Return(false)
				return mockEngine
			},
			want: want{
				impBidCtx: map[string]models.ImpCtx{
					"imp1": {
						BidCtx: map[string]models.BidCtx{
							"bid-id-1": {
								BidExt: models.BidExt{
									NetECPM: 5,
									ExtBid: openrtb_ext.ExtBid{
										Prebid: &openrtb_ext.ExtBidPrebid{
											Targeting: map[string]string{},
										},
									},
								},
								EG: 5,
								EN: 5,
							},
						},
					},
				},
			},
		},
		{
			name: "test auction between 3 bids when supportdeal is false and no bid satisfies dealTier",
			args: args{
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							StartTime: time.Now().UnixMilli(),
							ImpBidCtx: map[string]models.ImpCtx{
								"imp1": {},
							},
							PubIDStr: "5890",
						},
					},
				},
				payload: hookstage.AuctionResponsePayload{
					BidResponse: &openrtb2.BidResponse{
						Cur: "USD",
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										Price: 5,
										Ext:   json.RawMessage(`{}`),
									},
								},
								Seat: "pubmatic",
							},
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-2",
										ImpID: "imp1",
										Price: 20,
										Ext:   json.RawMessage(`{}`),
									},
								},
								Seat: "appnexus",
							},
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-3",
										ImpID: "imp1",
										Price: 10,
										Ext:   json.RawMessage(`{}`),
									},
								},
								Seat: "rubicon",
							},
						},
					},
				},
			},
			getMetricsEngine: func() (me *mock_metrics.MockMetricsEngine) {
				mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)
				mockEngine.EXPECT().RecordPublisherResponseTimeStats("5890", gomock.Any())
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("", "5890", "pubmatic")
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("", "5890", "appnexus")
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("", "5890", "rubicon")
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
				mockFeature.EXPECT().IsFscApplicable(gomock.Any(), gomock.Any(), gomock.Any()).Return(false)
				return mockEngine
			},
			want: want{
				impBidCtx: map[string]models.ImpCtx{
					"imp1": {
						BidCtx: map[string]models.BidCtx{
							"bid-id-1": {
								BidExt: models.BidExt{
									Nbr:     nbr.LossBidLostToHigherBid.Ptr(),
									NetECPM: 5,
									ExtBid: openrtb_ext.ExtBid{
										Prebid: &openrtb_ext.ExtBidPrebid{
											Targeting: map[string]string{},
										},
									},
								},
								EG: 5,
								EN: 5,
							},
							"bid-id-2": {
								BidExt: models.BidExt{
									NetECPM: 20,
									ExtBid: openrtb_ext.ExtBid{
										Prebid: &openrtb_ext.ExtBidPrebid{
											Targeting: map[string]string{},
										},
									},
								},
								EG: 20,
								EN: 20,
							},
							"bid-id-3": {
								BidExt: models.BidExt{
									Nbr:     nbr.LossBidLostToHigherBid.Ptr(),
									NetECPM: 10,
									ExtBid: openrtb_ext.ExtBid{
										Prebid: &openrtb_ext.ExtBidPrebid{
											Targeting: map[string]string{},
										},
									},
								},
								EG: 10,
								EN: 10,
							},
						},
					},
				},
			},
		},
		{
			name: "test auction between 3 bids when supportdeal is false and all bids satisfies dealTier",
			args: args{
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							StartTime: time.Now().UnixMilli(),
							ImpBidCtx: map[string]models.ImpCtx{
								"imp1": {},
							},
							PubIDStr: "5890",
						},
					},
				},
				payload: hookstage.AuctionResponsePayload{
					BidResponse: &openrtb2.BidResponse{
						Cur: "USD",
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										Price: 5,
										Ext:   json.RawMessage(`{"prebid":{"dealtiersatisfied":true}}`),
									},
								},
								Seat: "pubmatic",
							},
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-2",
										ImpID: "imp1",
										Price: 20,
										Ext:   json.RawMessage(`{"prebid":{"dealtiersatisfied":true}}`),
									},
								},
								Seat: "appnexus",
							},
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-3",
										ImpID: "imp1",
										Price: 10,
										Ext:   json.RawMessage(`{"prebid":{"dealtiersatisfied":true}}`),
									},
								},
								Seat: "rubicon",
							},
						},
					},
				},
			},
			getMetricsEngine: func() (me *mock_metrics.MockMetricsEngine) {
				mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)
				mockEngine.EXPECT().RecordPublisherResponseTimeStats("5890", gomock.Any())
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("", "5890", "pubmatic")
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("", "5890", "appnexus")
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("", "5890", "rubicon")
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
				mockFeature.EXPECT().IsFscApplicable(gomock.Any(), gomock.Any(), gomock.Any()).Return(false)
				return mockEngine
			},
			want: want{
				impBidCtx: map[string]models.ImpCtx{
					"imp1": {
						BidCtx: map[string]models.BidCtx{
							"bid-id-1": {
								BidExt: models.BidExt{
									Nbr:     nbr.LossBidLostToHigherBid.Ptr(),
									NetECPM: 5,
									ExtBid: openrtb_ext.ExtBid{
										Prebid: &openrtb_ext.ExtBidPrebid{
											DealTierSatisfied: true,
											Targeting:         map[string]string{},
										},
									},
								},
								EG: 5,
								EN: 5,
							},
							"bid-id-2": {
								BidExt: models.BidExt{
									NetECPM: 20,
									ExtBid: openrtb_ext.ExtBid{
										Prebid: &openrtb_ext.ExtBidPrebid{
											DealTierSatisfied: true,
											Targeting:         map[string]string{},
										},
									},
								},
								EG: 20,
								EN: 20,
							},
							"bid-id-3": {
								BidExt: models.BidExt{
									Nbr:     nbr.LossBidLostToHigherBid.Ptr(),
									NetECPM: 10,
									ExtBid: openrtb_ext.ExtBid{
										Prebid: &openrtb_ext.ExtBidPrebid{
											DealTierSatisfied: true,
											Targeting:         map[string]string{},
										},
									},
								},
								EG: 10,
								EN: 10,
							},
						},
					},
				},
			},
		},
		{
			name: "single bid and supportdeal is true",
			args: args{
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							StartTime: time.Now().UnixMilli(),
							ImpBidCtx: map[string]models.ImpCtx{
								"imp1": {},
							},
							PubIDStr:     "5890",
							SupportDeals: true,
						},
					},
				},
				payload: hookstage.AuctionResponsePayload{
					BidResponse: &openrtb2.BidResponse{
						Cur: "USD",
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										Price: 5,
										Ext:   json.RawMessage(`{}`),
									},
								},
								Seat: "pubmatic",
							},
						},
					},
				},
			},
			getMetricsEngine: func() (me *mock_metrics.MockMetricsEngine) {
				mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)
				mockEngine.EXPECT().RecordPublisherResponseTimeStats("5890", gomock.Any())
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("", "5890", "pubmatic")
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
				mockFeature.EXPECT().IsFscApplicable(gomock.Any(), gomock.Any(), gomock.Any()).Return(false)
				return mockEngine
			},
			want: want{
				impBidCtx: map[string]models.ImpCtx{
					"imp1": {
						BidCtx: map[string]models.BidCtx{
							"bid-id-1": {
								BidExt: models.BidExt{
									NetECPM: 5,
									ExtBid: openrtb_ext.ExtBid{
										Prebid: &openrtb_ext.ExtBidPrebid{
											Targeting: map[string]string{},
										},
									},
								},
								EG: 5,
								EN: 5,
							},
						},
					},
				},
			},
		},
		{
			name: "auction between 3 bids when supportdeal is true and no bid satisfies dealTier",
			args: args{
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							StartTime: time.Now().UnixMilli(),
							ImpBidCtx: map[string]models.ImpCtx{
								"imp1": {},
							},
							PubIDStr:     "5890",
							SupportDeals: true,
						},
					},
				},
				payload: hookstage.AuctionResponsePayload{
					BidResponse: &openrtb2.BidResponse{
						Cur: "USD",
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										Price: 20,
										Ext:   json.RawMessage(`{}`),
									},
								},
								Seat: "pubmatic",
							},
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-2",
										ImpID: "imp1",
										Price: 5,
										Ext:   json.RawMessage(`{}`),
									},
								},
								Seat: "appnexus",
							},
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-3",
										ImpID: "imp1",
										Price: 10,
										Ext:   json.RawMessage(`{}`),
									},
								},
								Seat: "rubicon",
							},
						},
					},
				},
			},
			getMetricsEngine: func() (me *mock_metrics.MockMetricsEngine) {
				mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)
				mockEngine.EXPECT().RecordPublisherResponseTimeStats("5890", gomock.Any())
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("", "5890", "pubmatic")
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("", "5890", "appnexus")
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("", "5890", "rubicon")
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
				mockFeature.EXPECT().IsFscApplicable(gomock.Any(), gomock.Any(), gomock.Any()).Return(false)
				return mockEngine
			},
			want: want{
				impBidCtx: map[string]models.ImpCtx{
					"imp1": {
						BidCtx: map[string]models.BidCtx{
							"bid-id-1": {
								BidExt: models.BidExt{
									NetECPM: 20,
									ExtBid: openrtb_ext.ExtBid{
										Prebid: &openrtb_ext.ExtBidPrebid{
											Targeting: map[string]string{},
										},
									},
								},
								EG: 20,
								EN: 20,
							},
							"bid-id-2": {
								BidExt: models.BidExt{
									Nbr:     nbr.LossBidLostToHigherBid.Ptr(),
									NetECPM: 5,
									ExtBid: openrtb_ext.ExtBid{
										Prebid: &openrtb_ext.ExtBidPrebid{
											Targeting: map[string]string{},
										},
									},
								},
								EG: 5,
								EN: 5,
							},
							"bid-id-3": {
								BidExt: models.BidExt{
									Nbr:     nbr.LossBidLostToHigherBid.Ptr(),
									NetECPM: 10,
									ExtBid: openrtb_ext.ExtBid{
										Prebid: &openrtb_ext.ExtBidPrebid{
											Targeting: map[string]string{},
										},
									},
								},
								EG: 10,
								EN: 10,
							},
						},
					},
				},
			},
		},
		{
			name: "auction between 3 bids when supportdeal is true and only middle bid satisfies dealTier",
			args: args{
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							StartTime: time.Now().UnixMilli(),
							ImpBidCtx: map[string]models.ImpCtx{
								"imp1": {},
							},
							PubIDStr:     "5890",
							SupportDeals: true,
						},
					},
				},
				payload: hookstage.AuctionResponsePayload{
					BidResponse: &openrtb2.BidResponse{
						Cur: "USD",
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										Price: 20,
										Ext:   json.RawMessage(`{}`),
									},
								},
								Seat: "pubmatic",
							},
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-2",
										ImpID: "imp1",
										Price: 5,
										Ext:   json.RawMessage(`{"prebid":{"dealtiersatisfied":true}}`),
									},
								},
								Seat: "appnexus",
							},
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-3",
										ImpID: "imp1",
										Price: 10,
										Ext:   json.RawMessage(`{}`),
									},
								},
								Seat: "rubicon",
							},
						},
					},
				},
			},
			getMetricsEngine: func() (me *mock_metrics.MockMetricsEngine) {
				mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)
				mockEngine.EXPECT().RecordPublisherResponseTimeStats("5890", gomock.Any())
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("", "5890", "pubmatic")
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("", "5890", "appnexus")
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("", "5890", "rubicon")
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
				mockFeature.EXPECT().IsFscApplicable(gomock.Any(), gomock.Any(), gomock.Any()).Return(false)
				return mockEngine
			},
			want: want{
				impBidCtx: map[string]models.ImpCtx{
					"imp1": {
						BidCtx: map[string]models.BidCtx{
							"bid-id-1": {
								BidExt: models.BidExt{
									Nbr:     nbr.LossBidLostToDealBid.Ptr(),
									NetECPM: 20,
									ExtBid: openrtb_ext.ExtBid{
										Prebid: &openrtb_ext.ExtBidPrebid{
											Targeting: map[string]string{},
										},
									},
								},
								EG: 20,
								EN: 20,
							},
							"bid-id-2": {
								BidExt: models.BidExt{
									NetECPM: 5,
									ExtBid: openrtb_ext.ExtBid{
										Prebid: &openrtb_ext.ExtBidPrebid{
											DealTierSatisfied: true,
											Targeting:         map[string]string{},
										},
									},
								},
								EG: 5,
								EN: 5,
							},
							"bid-id-3": {
								BidExt: models.BidExt{
									Nbr:     nbr.LossBidLostToDealBid.Ptr(),
									NetECPM: 10,
									ExtBid: openrtb_ext.ExtBid{
										Prebid: &openrtb_ext.ExtBidPrebid{
											Targeting: map[string]string{},
										},
									},
								},
								EG: 10,
								EN: 10,
							},
						},
					},
				},
			},
		},
		{
			name: "auction between 3 bids when supportdeal is true and only last bid satisfies dealTier",
			args: args{
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							StartTime: time.Now().UnixMilli(),
							ImpBidCtx: map[string]models.ImpCtx{
								"imp1": {},
							},
							PubIDStr:     "5890",
							SupportDeals: true,
						},
					},
				},
				payload: hookstage.AuctionResponsePayload{
					BidResponse: &openrtb2.BidResponse{
						Cur: "USD",
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										Price: 20,
										Ext:   json.RawMessage(`{}`),
									},
								},
								Seat: "pubmatic",
							},
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-2",
										ImpID: "imp1",
										Price: 5,
										Ext:   json.RawMessage(`{}`),
									},
								},
								Seat: "appnexus",
							},
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-3",
										ImpID: "imp1",
										Price: 10,
										Ext:   json.RawMessage(`{"prebid":{"dealtiersatisfied":true}}`),
									},
								},
								Seat: "rubicon",
							},
						},
					},
				},
			},
			getMetricsEngine: func() (me *mock_metrics.MockMetricsEngine) {
				mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)
				mockEngine.EXPECT().RecordPublisherResponseTimeStats("5890", gomock.Any())
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("", "5890", "pubmatic")
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("", "5890", "appnexus")
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("", "5890", "rubicon")
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
				mockFeature.EXPECT().IsFscApplicable(gomock.Any(), gomock.Any(), gomock.Any()).Return(false)
				return mockEngine
			},
			want: want{
				impBidCtx: map[string]models.ImpCtx{
					"imp1": {
						BidCtx: map[string]models.BidCtx{
							"bid-id-1": {
								BidExt: models.BidExt{
									Nbr:     nbr.LossBidLostToDealBid.Ptr(),
									NetECPM: 20,
									ExtBid: openrtb_ext.ExtBid{
										Prebid: &openrtb_ext.ExtBidPrebid{
											Targeting: map[string]string{},
										},
									},
								},
								EG: 20,
								EN: 20,
							},
							"bid-id-2": {
								BidExt: models.BidExt{
									NetECPM: 5,
									Nbr:     nbr.LossBidLostToDealBid.Ptr(),
									ExtBid: openrtb_ext.ExtBid{
										Prebid: &openrtb_ext.ExtBidPrebid{
											Targeting: map[string]string{},
										},
									},
								},
								EG: 5,
								EN: 5,
							},
							"bid-id-3": {
								BidExt: models.BidExt{
									ExtBid: openrtb_ext.ExtBid{
										Prebid: &openrtb_ext.ExtBidPrebid{
											DealTierSatisfied: true,
											Targeting:         map[string]string{},
										},
									},
									NetECPM: 10,
								},
								EG: 10,
								EN: 10,
							},
						},
					},
				},
			},
		},
		{
			name: "auction between 3 bids when supportdeal is true and only first bid satisfies dealTier",
			args: args{
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							StartTime: time.Now().UnixMilli(),
							ImpBidCtx: map[string]models.ImpCtx{
								"imp1": {},
							},
							PubIDStr:     "5890",
							SupportDeals: true,
						},
					},
				},
				payload: hookstage.AuctionResponsePayload{
					BidResponse: &openrtb2.BidResponse{
						Cur: "USD",
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										Price: 20,
										Ext:   json.RawMessage(`{"prebid":{"dealtiersatisfied":true}}`),
									},
								},
								Seat: "pubmatic",
							},
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-2",
										ImpID: "imp1",
										Price: 5,
										Ext:   json.RawMessage(`{}`),
									},
								},
								Seat: "appnexus",
							},
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-3",
										ImpID: "imp1",
										Price: 10,
										Ext:   json.RawMessage(`{}`),
									},
								},
								Seat: "rubicon",
							},
						},
					},
				},
			},
			getMetricsEngine: func() (me *mock_metrics.MockMetricsEngine) {
				mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)
				mockEngine.EXPECT().RecordPublisherResponseTimeStats("5890", gomock.Any())
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("", "5890", "pubmatic")
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("", "5890", "appnexus")
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("", "5890", "rubicon")
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
				mockFeature.EXPECT().IsFscApplicable(gomock.Any(), gomock.Any(), gomock.Any()).Return(false)
				return mockEngine
			},
			want: want{
				impBidCtx: map[string]models.ImpCtx{
					"imp1": {
						BidCtx: map[string]models.BidCtx{
							"bid-id-1": {
								BidExt: models.BidExt{
									ExtBid: openrtb_ext.ExtBid{
										Prebid: &openrtb_ext.ExtBidPrebid{
											DealTierSatisfied: true,
											Targeting:         map[string]string{},
										},
									},
									NetECPM: 20,
								},
								EG: 20,
								EN: 20,
							},
							"bid-id-2": {
								BidExt: models.BidExt{
									NetECPM: 5,
									Nbr:     nbr.LossBidLostToDealBid.Ptr(),
									ExtBid: openrtb_ext.ExtBid{
										Prebid: &openrtb_ext.ExtBidPrebid{
											Targeting: map[string]string{},
										},
									},
								},
								EG: 5,
								EN: 5,
							},
							"bid-id-3": {
								BidExt: models.BidExt{
									Nbr:     nbr.LossBidLostToDealBid.Ptr(),
									NetECPM: 10,
									ExtBid: openrtb_ext.ExtBid{
										Prebid: &openrtb_ext.ExtBidPrebid{
											Targeting: map[string]string{},
										},
									},
								},
								EG: 10,
								EN: 10,
							},
						},
					},
				},
			},
		},
		{
			name: "auction between 3 bids when supportdeal is true and all bids satisfies dealTier",
			args: args{
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							StartTime: time.Now().UnixMilli(),
							ImpBidCtx: map[string]models.ImpCtx{
								"imp1": {},
							},
							PubIDStr:     "5890",
							SupportDeals: true,
						},
					},
				},
				payload: hookstage.AuctionResponsePayload{
					BidResponse: &openrtb2.BidResponse{
						Cur: "USD",
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										Price: 20,
										Ext:   json.RawMessage(`{"prebid":{"dealtiersatisfied":true}}`),
									},
								},
								Seat: "pubmatic",
							},
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-2",
										ImpID: "imp1",
										Price: 5,
										Ext:   json.RawMessage(`{"prebid":{"dealtiersatisfied":true}}`),
									},
								},
								Seat: "appnexus",
							},
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-3",
										ImpID: "imp1",
										Price: 10,
										Ext:   json.RawMessage(`{"prebid":{"dealtiersatisfied":true}}`),
									},
								},
								Seat: "rubicon",
							},
						},
					},
				},
			},
			getMetricsEngine: func() (me *mock_metrics.MockMetricsEngine) {
				mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)
				mockEngine.EXPECT().RecordPublisherResponseTimeStats("5890", gomock.Any())
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("", "5890", "pubmatic")
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("", "5890", "appnexus")
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("", "5890", "rubicon")
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
				mockFeature.EXPECT().IsFscApplicable(gomock.Any(), gomock.Any(), gomock.Any()).Return(false)
				return mockEngine
			},
			want: want{
				impBidCtx: map[string]models.ImpCtx{
					"imp1": {
						BidCtx: map[string]models.BidCtx{
							"bid-id-1": {
								BidExt: models.BidExt{
									ExtBid: openrtb_ext.ExtBid{
										Prebid: &openrtb_ext.ExtBidPrebid{
											DealTierSatisfied: true,
											Targeting:         map[string]string{},
										},
									},
									NetECPM: 20,
								},
								EG: 20,
								EN: 20,
							},
							"bid-id-2": {
								BidExt: models.BidExt{
									ExtBid: openrtb_ext.ExtBid{
										Prebid: &openrtb_ext.ExtBidPrebid{
											DealTierSatisfied: true,
											Targeting:         map[string]string{},
										},
									},
									NetECPM: 5,
									Nbr:     nbr.LossBidLostToHigherBid.Ptr(),
								},
								EG: 5,
								EN: 5,
							},
							"bid-id-3": {
								BidExt: models.BidExt{
									ExtBid: openrtb_ext.ExtBid{
										Prebid: &openrtb_ext.ExtBidPrebid{
											DealTierSatisfied: true,
											Targeting:         map[string]string{},
										},
									},
									Nbr:     nbr.LossBidLostToHigherBid.Ptr(),
									NetECPM: 10,
								},
								EG: 10,
								EN: 10,
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := OpenWrap{
				metricEngine: tt.getMetricsEngine(),
				pubFeatures:  mockFeature,
			}
			hookResult, _ := o.handleAuctionResponseHook(tt.args.ctx, tt.args.moduleCtx, tt.args.payload)
			mutations := hookResult.ChangeSet.Mutations()
			assert.NotEmpty(t, mutations, tt.name)
			rctxInterface := hookResult.AnalyticsTags.Activities[0].Results[0].Values["request-ctx"]
			rctx := rctxInterface.(*models.RequestCtx)
			assert.Equal(t, tt.want.impBidCtx, rctx.ImpBidCtx, tt.name)
		})
	}
}

func TestCTAOverlayInHandleAuctionResponseHook(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockFeature := mock_feature.NewMockFeature(ctrl)

	// VAST 3.0 InLine with CreativeExtension name=PubMatic and ctaoverlay JSON
	vastWithCTA := `<VAST version="3.0"><Ad><InLine><AdSystem>Test</AdSystem><AdTitle></AdTitle><Impression></Impression><Creatives><Creative><CreativeExtensions><CreativeExtension name="PubMatic" type="application/json"><![CDATA[{"ctaoverlay":{"delay":0,"pos":1,"ctacopy":"Learn More"}}]]></CreativeExtension></CreativeExtensions></Creative></Creatives></InLine></Ad></VAST>`

	type args struct {
		ctx       context.Context
		moduleCtx hookstage.ModuleInvocationContext
		payload   hookstage.AuctionResponsePayload
	}
	tests := []struct {
		name                   string
		args                   args
		wantCTAOverlayInjected bool // true: bid should have OWSDK[ctaoverlay] set
		wantImpID              string
		wantBidID              string
		getMetricsEngine       func() *mock_metrics.MockMetricsEngine
	}{
		{
			name: "CTA_overlay_injected_for_pubmatic_video_bid_when_imp_requests_it",
			args: args{
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							StartTime:        time.Now().UnixMilli(),
							PubIDStr:         "5890",
							PartnerConfigMap: map[int]map[string]string{1: {models.REVSHARE: "0.5"}},
							ImpBidCtx: map[string]models.ImpCtx{
								"imp1": {
									IsCTAOverlayRequest: true,
									DisplayManagerVer:   "4.10.0",
									Bidders:             map[string]models.PartnerData{"pubmatic": {PartnerID: 1}},
								},
							},
						},
					},
				},
				payload: hookstage.AuctionResponsePayload{
					BidResponse: &openrtb2.BidResponse{
						Cur: "USD",
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "pubmatic",
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-cta-1",
										ImpID: "imp1",
										Price: 2.5,
										AdM:   vastWithCTA,
										Ext:   json.RawMessage(`{"prebid":{"type":"video"}}`),
									},
								},
							},
						},
					},
				},
			},
			wantCTAOverlayInjected: true,
			wantImpID:              "imp1",
			wantBidID:              "bid-cta-1",
			getMetricsEngine: func() *mock_metrics.MockMetricsEngine {
				mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)
				mockEngine.EXPECT().RecordPublisherResponseTimeStats("5890", gomock.Any())
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("", "5890", "pubmatic")
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
				mockFeature.EXPECT().IsFscApplicable(gomock.Any(), gomock.Any(), gomock.Any()).Return(false)
				return mockEngine
			},
		},
		{
			name: "CTA_overlay_not_injected_when_seat_not_pubmatic_core",
			args: args{
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							StartTime:        time.Now().UnixMilli(),
							PubIDStr:         "5890",
							PartnerConfigMap: map[int]map[string]string{2: {models.REVSHARE: "0.5"}},
							ImpBidCtx: map[string]models.ImpCtx{
								"imp1": {
									IsCTAOverlayRequest: true,
									DisplayManagerVer:   "4.10.0",
									Bidders:             map[string]models.PartnerData{"appnexus": {PartnerID: 2}},
								},
							},
						},
					},
				},
				payload: hookstage.AuctionResponsePayload{
					BidResponse: &openrtb2.BidResponse{
						Cur: "USD",
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "appnexus",
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-cta-2",
										ImpID: "imp1",
										Price: 3,
										AdM:   vastWithCTA,
										Ext:   json.RawMessage(`{"prebid":{"type":"video"}}`),
									},
								},
							},
						},
					},
				},
			},
			wantCTAOverlayInjected: false,
			wantImpID:              "imp1",
			wantBidID:              "bid-cta-2",
			getMetricsEngine: func() *mock_metrics.MockMetricsEngine {
				mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)
				mockEngine.EXPECT().RecordPublisherResponseTimeStats("5890", gomock.Any())
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("", "5890", "appnexus")
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
				mockFeature.EXPECT().IsFscApplicable(gomock.Any(), gomock.Any(), gomock.Any()).Return(false)
				return mockEngine
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := OpenWrap{
				metricEngine: tt.getMetricsEngine(),
				pubFeatures:  mockFeature,
			}
			hookResult, _ := o.handleAuctionResponseHook(tt.args.ctx, tt.args.moduleCtx, tt.args.payload)
			mutations := hookResult.ChangeSet.Mutations()
			assert.NotEmpty(t, mutations, tt.name)
			rctxInterface := hookResult.AnalyticsTags.Activities[0].Results[0].Values["request-ctx"]
			rctx := rctxInterface.(*models.RequestCtx)
			impCtx, ok := rctx.ImpBidCtx[tt.wantImpID]
			assert.True(t, ok, "imp %s in ImpBidCtx", tt.wantImpID)
			bidCtx, ok := impCtx.BidCtx[tt.wantBidID]
			assert.True(t, ok, "bid %s in BidCtx", tt.wantBidID)
			if tt.wantCTAOverlayInjected {
				assert.NotNil(t, bidCtx.BidExt.OWSDK, tt.name)
				ctaVal := bidCtx.BidExt.OWSDK[models.CTAOVERLAY]
				assert.NotNil(t, ctaVal, "OWSDK.ctaoverlay should be set")
				raw, ok := ctaVal.(json.RawMessage)
				assert.True(t, ok, "ctaoverlay value is json.RawMessage")
				var ctaMap map[string]interface{}
				assert.NoError(t, json.Unmarshal(raw, &ctaMap), "ctaoverlay JSON must be valid")
				assert.Equal(t, "Learn More", ctaMap["ctacopy"], tt.name)
				assert.Equal(t, float64(1), ctaMap["pos"], tt.name)
				assert.Equal(t, float64(0), ctaMap["delay"], tt.name)
			} else {
				assert.True(t, bidCtx.BidExt.OWSDK == nil || bidCtx.BidExt.OWSDK[models.CTAOVERLAY] == nil,
					"OWSDK.ctaoverlay should not be set for non-PubMatic core partner")
			}
		})
	}
}

func TestPrebidTargetingInHandleAuctionResponseHook(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockFeature := mock_feature.NewMockFeature(ctrl)

	type args struct {
		ctx       context.Context
		moduleCtx hookstage.ModuleInvocationContext
		payload   hookstage.AuctionResponsePayload
	}
	type want struct {
		impBidCtx map[string]models.ImpCtx
	}
	tests := []struct {
		name             string
		args             args
		want             want
		getMetricsEngine func() *mock_metrics.MockMetricsEngine
	}{
		{
			name: "prebid targeting without custom dimensions",
			args: args{
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							StartTime: time.Now().UnixMilli(),
							ImpBidCtx: map[string]models.ImpCtx{
								"imp1": {},
							},
							PubIDStr:         "5890",
							CustomDimensions: map[string]models.CustomDimension{},
						},
					},
				},
				payload: hookstage.AuctionResponsePayload{
					BidResponse: &openrtb2.BidResponse{
						Cur: "USD",
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										Price: 5,
										Ext:   json.RawMessage(`{}`),
									},
								},
								Seat: "pubmatic",
							},
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-2",
										ImpID: "imp1",
										Price: 20,
										Ext:   json.RawMessage(`{"prebid":{"targeting":{}}}`),
									},
								},
								Seat: "appnexus",
							},
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-3",
										ImpID: "imp1",
										Price: 10,
										Ext:   json.RawMessage(`{"prebid":{"targeting":{"key":"val"}}}`),
									},
								},
								Seat: "rubicon",
							},
						},
					},
				},
			},
			getMetricsEngine: func() (me *mock_metrics.MockMetricsEngine) {
				mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)
				mockEngine.EXPECT().RecordPublisherResponseTimeStats("5890", gomock.Any())
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("", "5890", "pubmatic")
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("", "5890", "appnexus")
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("", "5890", "rubicon")
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
				mockFeature.EXPECT().IsFscApplicable(gomock.Any(), gomock.Any(), gomock.Any()).Return(false)
				return mockEngine
			},
			want: want{
				impBidCtx: map[string]models.ImpCtx{
					"imp1": {
						BidCtx: map[string]models.BidCtx{
							"bid-id-1": {
								BidExt: models.BidExt{
									Nbr:     nbr.LossBidLostToHigherBid.Ptr(),
									NetECPM: 5,
									ExtBid: openrtb_ext.ExtBid{
										Prebid: &openrtb_ext.ExtBidPrebid{
											Targeting: map[string]string{},
										},
									},
								},
								EG: 5,
								EN: 5,
							},
							"bid-id-2": {
								BidExt: models.BidExt{
									NetECPM: 20,
									ExtBid: openrtb_ext.ExtBid{
										Prebid: &openrtb_ext.ExtBidPrebid{
											Targeting: map[string]string{},
										},
									},
								},
								EG: 20,
								EN: 20,
							},
							"bid-id-3": {
								BidExt: models.BidExt{
									Nbr:     nbr.LossBidLostToHigherBid.Ptr(),
									NetECPM: 10,
									ExtBid: openrtb_ext.ExtBid{
										Prebid: &openrtb_ext.ExtBidPrebid{
											Targeting: map[string]string{},
										},
									},
								},
								EG: 10,
								EN: 10,
							},
						},
					},
				},
			},
		},
		{
			name: "prebid targeting custom dimensions",
			args: args{
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							StartTime: time.Now().UnixMilli(),
							ImpBidCtx: map[string]models.ImpCtx{
								"imp1": {},
							},
							PubIDStr: "5890",
							CustomDimensions: map[string]models.CustomDimension{
								"traffic": {
									Value:     "email",
									SendToGAM: ptrutil.ToPtr(true),
								},
								"author": {
									Value:     "hemry",
									SendToGAM: ptrutil.ToPtr(false),
								},
								"age": {
									Value: "23",
								},
							},
						},
					},
				},
				payload: hookstage.AuctionResponsePayload{
					BidResponse: &openrtb2.BidResponse{
						Cur: "USD",
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										Price: 5,
										Ext:   json.RawMessage(`{}`),
									},
								},
								Seat: "pubmatic",
							},
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-2",
										ImpID: "imp1",
										Price: 20,
										Ext:   json.RawMessage(`{"prebid":{"targeting":{}}}`),
									},
								},
								Seat: "appnexus",
							},
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-3",
										ImpID: "imp1",
										Price: 10,
										Ext:   json.RawMessage(`{"prebid":{"targeting":{"key":"val"}}}`),
									},
								},
								Seat: "rubicon",
							},
						},
					},
				},
			},
			getMetricsEngine: func() (me *mock_metrics.MockMetricsEngine) {
				mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)
				mockEngine.EXPECT().RecordPublisherResponseTimeStats("5890", gomock.Any())
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("", "5890", "pubmatic")
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("", "5890", "appnexus")
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("", "5890", "rubicon")
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
				mockFeature.EXPECT().IsFscApplicable(gomock.Any(), gomock.Any(), gomock.Any()).Return(false)
				return mockEngine
			},
			want: want{
				impBidCtx: map[string]models.ImpCtx{
					"imp1": {
						BidCtx: map[string]models.BidCtx{
							"bid-id-1": {
								BidExt: models.BidExt{
									Nbr:     nbr.LossBidLostToHigherBid.Ptr(),
									NetECPM: 5,
									ExtBid: openrtb_ext.ExtBid{
										Prebid: &openrtb_ext.ExtBidPrebid{
											Targeting: map[string]string{
												"age":     "23",
												"traffic": "email",
											},
										},
									},
								},
								EG: 5,
								EN: 5,
							},
							"bid-id-2": {
								BidExt: models.BidExt{
									NetECPM: 20,
									ExtBid: openrtb_ext.ExtBid{
										Prebid: &openrtb_ext.ExtBidPrebid{
											Targeting: map[string]string{
												"age":     "23",
												"traffic": "email",
											},
										},
									},
								},
								EG: 20,
								EN: 20,
							},
							"bid-id-3": {
								BidExt: models.BidExt{
									Nbr:     nbr.LossBidLostToHigherBid.Ptr(),
									NetECPM: 10,
									ExtBid: openrtb_ext.ExtBid{
										Prebid: &openrtb_ext.ExtBidPrebid{
											Targeting: map[string]string{
												"age":     "23",
												"traffic": "email",
											},
										},
									},
								},
								EG: 10,
								EN: 10,
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := OpenWrap{
				metricEngine: tt.getMetricsEngine(),
				pubFeatures:  mockFeature,
			}
			hookResult, _ := o.handleAuctionResponseHook(tt.args.ctx, tt.args.moduleCtx, tt.args.payload)
			mutations := hookResult.ChangeSet.Mutations()
			assert.NotEmpty(t, mutations, tt.name)
			rctxInterface := hookResult.AnalyticsTags.Activities[0].Results[0].Values["request-ctx"]
			rctx := rctxInterface.(*models.RequestCtx)
			assert.Equal(t, tt.want.impBidCtx, rctx.ImpBidCtx, tt.name)
		})
	}
}

func TestResetBidIdtoOriginal(t *testing.T) {
	type args struct {
		bidResponse *openrtb2.BidResponse
	}
	tests := []struct {
		name string
		args args
		want *openrtb2.BidResponse
	}{
		{
			name: "Reset Bid Id to original",
			args: args{
				bidResponse: &openrtb2.BidResponse{
					SeatBid: []openrtb2.SeatBid{
						{
							Bid: []openrtb2.Bid{
								{
									ID: "original::generated",
								},
								{
									ID: "original-1::generated-1",
								},
							},
							Seat: "pubmatic",
						},
						{
							Bid: []openrtb2.Bid{
								{
									ID: "original-2::generated-2",
								},
							},
							Seat: "index",
						},
					},
				},
			},
			want: &openrtb2.BidResponse{
				SeatBid: []openrtb2.SeatBid{
					{
						Bid: []openrtb2.Bid{
							{
								ID: "original",
							},
							{
								ID: "original-1",
							},
						},
						Seat: "pubmatic",
					},
					{
						Bid: []openrtb2.Bid{
							{
								ID: "original-2",
							},
						},
						Seat: "index",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetBidIdtoOriginal(tt.args.bidResponse)
			assert.Equal(t, tt.want, tt.args.bidResponse, "Bid Id should reset to original")
		})
	}
}

func TestAuctionResponseHookForEndpointWebS2S(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockCache := mock_cache.NewMockCache(ctrl)
	defer ctrl.Finish()

	type args struct {
		ctx       context.Context
		moduleCtx hookstage.ModuleInvocationContext
		payload   hookstage.AuctionResponsePayload
	}

	type want struct {
		bidResponse *openrtb2.BidResponse
		err         error
	}

	tests := []struct {
		name             string
		args             args
		want             want
		getMetricsEngine func() *mock_metrics.MockMetricsEngine
	}{
		{
			name: "inject_tracker_in_respose_for_WebS2S_endpoint",
			args: args{
				ctx: nil,
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							Endpoint: models.EndpointWebS2S,
							Trackers: map[string]models.OWTracker{
								"bid1": {
									BidType: models.Video,
								},
							},
						},
					},
				},
				payload: hookstage.AuctionResponsePayload{
					BidResponse: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:  "12345",
										AdM: `<VAST version="3.0"><Ad><Wrapper></Wrapper></Ad></VAST>`,
									},
								},
							},
						},
					},
				},
			},
			want: want{
				bidResponse: &openrtb2.BidResponse{
					SeatBid: []openrtb2.SeatBid{
						{
							Bid: []openrtb2.Bid{
								{
									ID:  "12345",
									AdM: "<VAST version=\"3.0\"><Ad><Wrapper></Wrapper></Ad></VAST><div style=\"position:absolute;left:0px;top:0px;visibility:hidden;\"><img src=\"https:?adv=&af=banner&aps=0&au=&bc=&bidid=12345&di=-1&eg=0&en=0&ft=0&iid=&kgpv=&orig=&origbidid=12345&pdvid=0&pid=0&plt=0&pn=&psz=0x0&pubid=0&purl=&sl=1&slot=&ss=1&tst=0\"></div>"},
							},
						},
					},
				},
				err: nil,
			},
			getMetricsEngine: func() *mock_metrics.MockMetricsEngine {
				mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats(gomock.Any(), gomock.Any(), gomock.Any())
				mockEngine.EXPECT().RecordNobidErrPrebidServerResponse(gomock.Any())
				mockEngine.EXPECT().RecordPublisherResponseTimeStats(gomock.Any(), gomock.Any())
				return mockEngine
			},
		},
		{
			name: "inject_tracker_in_respose_and_reset_bidID_to_orignal_for_WebS2S_endpoint",
			args: args{
				ctx: nil,
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							Endpoint: models.EndpointWebS2S,
							Trackers: map[string]models.OWTracker{
								"bid1": {
									BidType: models.Video,
								},
							},
						},
					},
				},
				payload: hookstage.AuctionResponsePayload{
					BidResponse: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:  "12345:: 123422222225",
										AdM: `<VAST version="3.0"><Ad><Wrapper></Wrapper></Ad></VAST>`,
									},
								},
							},
						},
					},
				},
			},
			want: want{
				bidResponse: &openrtb2.BidResponse{
					SeatBid: []openrtb2.SeatBid{
						{
							Bid: []openrtb2.Bid{
								{
									ID:  "12345",
									AdM: "<VAST version=\"3.0\"><Ad><Wrapper></Wrapper></Ad></VAST><div style=\"position:absolute;left:0px;top:0px;visibility:hidden;\"><img src=\"https:?adv=&af=banner&aps=0&au=&bc=&bidid=12345&di=-1&eg=0&en=0&ft=0&iid=&kgpv=&orig=&origbidid=12345&pdvid=0&pid=0&plt=0&pn=&psz=0x0&pubid=0&purl=&sl=1&slot=&ss=1&tst=0\"></div>"},
							},
						},
					},
				},
				err: nil,
			},
			getMetricsEngine: func() *mock_metrics.MockMetricsEngine {
				mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats(gomock.Any(), gomock.Any(), gomock.Any())
				mockEngine.EXPECT().RecordNobidErrPrebidServerResponse(gomock.Any())
				mockEngine.EXPECT().RecordPublisherResponseTimeStats(gomock.Any(), gomock.Any())
				return mockEngine
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := OpenWrap{
				metricEngine: tt.getMetricsEngine(),
				cache:        mockCache,
			}
			hookResult, err := o.handleAuctionResponseHook(tt.args.ctx, tt.args.moduleCtx, tt.args.payload)
			assert.Equal(t, tt.want.err, err, tt.name)
			mutations := hookResult.ChangeSet.Mutations()
			assert.NotEmpty(t, mutations, tt.name)
			for _, mut := range mutations {
				result, err := mut.Apply(tt.args.payload)
				assert.Nil(t, err, tt.name)
				assert.Equal(t, tt.want.bidResponse, result.BidResponse, tt.name)
			}
		})
	}
}

func TestApplyBidExpAndBidExtFromCtx(t *testing.T) {
	const (
		bidID = "bid-1"
		impID = "imp-1"
	)

	bidExt := models.BidExt{
		CreativeType:   "banner",
		NetECPM:        3.6,
		OriginalBidCPM: 4,
		OriginalBidCur: "USD",
	}
	wantExt, err := json.Marshal(bidExt)
	assert.NoError(t, err)

	type args struct {
		rctx        models.RequestCtx
		bidResponse *openrtb2.BidResponse
	}

	tests := []struct {
		name    string
		args    args
		wantExp int64
		wantExt json.RawMessage
	}{
		{
			name: "clears_bid_exp_when_omit_bid_exp_from_tracker",
			args: args{
				rctx: models.RequestCtx{
					Endpoint: models.EndpointVideo,
					ImpBidCtx: map[string]models.ImpCtx{
						impID: {
							BidCtx: map[string]models.BidCtx{
								bidID: {
									BidExt:                bidExt,
									OmitBidExpFromTracker: true,
								},
							},
						},
					},
				},
				bidResponse: &openrtb2.BidResponse{
					SeatBid: []openrtb2.SeatBid{{
						Seat: "pubmatic",
						Bid: []openrtb2.Bid{{
							ID:    bidID,
							ImpID: impID,
							Exp:   300,
						}},
					}},
				},
			},
			wantExp: 0,
			wantExt: wantExt,
		},
		{
			name: "preserves_bid_exp_when_not_omitting_from_tracker",
			args: args{
				rctx: models.RequestCtx{
					Endpoint: models.EndpointVideo,
					ImpBidCtx: map[string]models.ImpCtx{
						impID: {
							BidCtx: map[string]models.BidCtx{
								bidID: {
									BidExt:                bidExt,
									OmitBidExpFromTracker: false,
								},
							},
						},
					},
				},
				bidResponse: &openrtb2.BidResponse{
					SeatBid: []openrtb2.SeatBid{{
						Seat: "pubmatic",
						Bid: []openrtb2.Bid{{
							ID:    bidID,
							ImpID: impID,
							Exp:   300,
						}},
					}},
				},
			},
			wantExp: 300,
			wantExt: wantExt,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applyBidExpAndBidExtFromCtx(tt.args.rctx, tt.args.bidResponse)
			got := tt.args.bidResponse.SeatBid[0].Bid[0]
			assert.Equal(t, tt.wantExp, got.Exp, tt.name)
			assert.JSONEq(t, string(tt.wantExt), string(got.Ext), tt.name)
		})
	}
}

func TestOpenWrapHandleAuctionResponseHook(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockCache := mock_cache.NewMockCache(ctrl)
	defer ctrl.Finish()
	mockFeature := mock_feature.NewMockFeature(ctrl)

	type want struct {
		result      hookstage.HookResult[hookstage.AuctionResponsePayload]
		bidResponse json.RawMessage
		err         error
	}
	type args struct {
		ctx       context.Context
		moduleCtx hookstage.ModuleInvocationContext
		payload   hookstage.AuctionResponsePayload
	}
	tests := []struct {
		name     string
		args     args
		want     want
		doMutate bool
		setup    func() *mock_metrics.MockMetricsEngine
	}{
		{
			name: "empty moduleContext",
			args: args{
				ctx:       nil,
				moduleCtx: hookstage.ModuleInvocationContext{},
				payload:   hookstage.AuctionResponsePayload{},
			},
			doMutate: false,
			want: want{
				result: hookstage.HookResult[hookstage.AuctionResponsePayload]{
					DebugMessages: []string{"error: module-ctx not found in handleAuctionResponseHook()"},
				},
				err: nil,
			},
		},
		{
			name: "empty requestContext",
			args: args{
				ctx: nil,
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": nil,
					},
				},
				payload: hookstage.AuctionResponsePayload{},
			},
			doMutate: false,
			want: want{
				result: hookstage.HookResult[hookstage.AuctionResponsePayload]{
					DebugMessages: []string{"error: request-ctx not found in handleAuctionResponseHook()"},
				},
				err: nil,
			},
		},
		{
			name: "requestContext is not of type RequestCtx",
			args: args{
				ctx: nil,
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": "request-ctx", // request-ctx is not of type RequestCtx
					},
				},
				payload: hookstage.AuctionResponsePayload{},
			},
			doMutate: false,
			want: want{
				result: hookstage.HookResult[hookstage.AuctionResponsePayload]{
					DebugMessages: []string{"error: request-ctx not found in handleAuctionResponseHook()"},
				},
				err: nil,
			},
		},
		{
			name: "requestContext has sshb=1(request should not execute module hook)",
			args: args{
				ctx: nil,
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							Sshb: "1",
						},
					},
				},
				payload: hookstage.AuctionResponsePayload{},
			},
			doMutate: false,
			want: want{
				result: hookstage.HookResult[hookstage.AuctionResponsePayload]{},
				err:    nil,
			},
		},
		{
			name: "empty bidResponse",
			args: args{
				ctx: nil,
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							Sshb:     "0",
							PubID:    5890,
							PubIDStr: "5890",
						},
					},
				},
				payload: hookstage.AuctionResponsePayload{
					BidResponse: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{},
					},
				},
			},
			doMutate: true,
			setup: func() *mock_metrics.MockMetricsEngine {
				mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)
				mockEngine.EXPECT().RecordNobidErrPrebidServerResponse("5890")
				mockEngine.EXPECT().RecordPublisherResponseTimeStats("5890", gomock.Any())
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
				return mockEngine
			},
			want: want{
				result:      hookstage.HookResult[hookstage.AuctionResponsePayload]{},
				err:         nil,
				bidResponse: json.RawMessage(`{"id":"","ext":{"matchedimpression":{}}}`),
			},
		},
		{
			name: "valid bidResponse with banner bids",
			args: args{
				ctx: nil,
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							PubID:    5890,
							PubIDStr: "5890",
							Platform: "web",
							ImpBidCtx: map[string]models.ImpCtx{
								"Div1": {
									Bidders: map[string]models.PartnerData{
										"pubmatic": {
											PartnerID:        123,
											PrebidBidderCode: "pubmatic",
										},
									},
									Video:    &openrtb2.Video{},
									Type:     "video",
									IsBanner: true,
								},
							},
							BidderResponseTimeMillis: map[string]int{},
							SeatNonBids:              map[string][]openrtb_ext.NonBid{},
							ReturnAllBidStatus:       true,
							Debug:                    true,
							ClientConfigFlag:         1,
							PartnerConfigMap: map[int]map[string]string{
								123: {
									models.PARTNER_ID:          "123",
									models.PREBID_PARTNER_NAME: "pubmatic",
									models.BidderCode:          "pubmatic",
									models.SERVER_SIDE_FLAG:    "1",
									models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
									models.TIMEOUT:             "200",
								},
								-1: {
									models.DisplayVersionID: "1",
									"refreshInterval":       "30",
									"rev_share":             "0.5",
								},
							},
						},
					},
				},
				payload: hookstage.AuctionResponsePayload{
					BidResponse: &openrtb2.BidResponse{
						ID: "12345",
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "Div1",
										Price: 5,
										AdM:   "<img src=\"http://ads.pubmatic.com/AdTag/728x90.png\"></img><div style=\"position:absolute;left:0px;top:0px;visibility:hidden;\"><img src=\"https://t.pubmatic.com/wt?adv=&af=banner&aps=0&au=%2F43743431%2FDMDemo&bc=appnexus&bidid=4033c510-6d67-4af6-b53f-682ff1a580c3&di=-1&eg=14&en=14&frv=1.57&ft=0&fv=1.57&iid=429d469d-8cfb-495a-9f0c-5f48aa0ede40&kgpv=&orig=ebay.com&origbidid=718825584&pdvid=1&pid=22503&plt=1&pn=appnexus&psz=728x90&pubid=5890&purl=http%3A%2F%2Febay.com%2Finte%2Fautomation%2Fs2s_activation%2Fbanner-with-gdpr-pubmatic-denied-defaultbidder.html%3Fprofileid%3D22503%26pwtv%3D1%26pwtvc%3D1%26appnexus_banner_fixedbid%3D14%26fixedbid%3D1%26debug%3D1&sl=1&slot=%2F43743431%2FDMDemo&ss=1&tst=1704357774\"></div>",
										Ext:   json.RawMessage(`{"bidtype":0,"deal_channel":1,"dspid":6,"origbidcpm":8,"origbidcur":"USD","ibv":true,"prebid":{"bidid":"bb57a9e3-fdc2-4772-8071-112dd7f50a6a","meta":{"adaptercode":"pubmatic","advertiserId":4098,"agencyId":4098,"demandSource":"6","mediaType":"banner","networkId":6},"targeting":{"hb_bidder_pubmatic":"pubmatic","hb_deal_pubmatic":"PUBDEAL1","hb_pb_pubmatic":"8.00","hb_size_pubmatic":"728x90"},"type":"banner","video":{"duration":0,"primary_category":"","vasttagid":""}},"clicktrackers":["http://clicktracker1.com","http://clicktracker2.com"]}`),
									},
								},
								Seat: "pubmatic",
							},
						},
						Ext: json.RawMessage(`{"responsetimemillis":{"pubmatic":8}}`),
					},
				},
			},
			setup: func() *mock_metrics.MockMetricsEngine {
				mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("web", "5890", "pubmatic")
				mockEngine.EXPECT().RecordPartnerResponseTimeStats("5890", "pubmatic", 8)
				mockEngine.EXPECT().RecordPublisherResponseTimeStats("5890", gomock.Any())
				mockEngine.EXPECT().RecordPublisherPartnerNoCookieStats("5890", gomock.Any()).AnyTimes()
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
				mockFeature.EXPECT().IsFscApplicable(gomock.Any(), gomock.Any(), gomock.Any()).Return(false)
				mockEngine.EXPECT().RecordIBVRequest("5890", gomock.Any())
				return mockEngine
			},
			doMutate: true,
			want: want{
				result: hookstage.HookResult[hookstage.AuctionResponsePayload]{
					DebugMessages: []string{`[{"PubID":5890,"ProfileID":0,"DisplayID":0,"VersionID":0,"DisplayVersionID":0,"SSAuction":0,"SummaryDisable":0,"SSAI":"","PartnerConfigMap":{"-1":{"displayVersionId":"1","refreshInterval":"30","rev_share":"0.5"},"123":{"bidderCode":"pubmatic","kgp":"_AU_@_W_x_H_","partnerId":"123","prebidPartnerName":"pubmatic","serverSideEnabled":"1","timeout":"200"}},"SupportDeals":false,"Platform":"web","LoggerImpressionID":"","ClientConfigFlag":1,"IP":"","TMax":0,"IsTestRequest":0,"ABTestConfig":0,"ABTestConfigApplied":0,"IsCTVRequest":false,"TrackerEndpoint":"","VideoErrorTrackerEndpoint":"","UA":"","Cookies":"","UidCookie":null,"KADUSERCookie":null,"ParsedUidCookie":null,"OriginCookie":"","Debug":true,"Trace":false,"PageURL":"","StartTime":0,"DevicePlatform":0,"Trackers":{"bid-id-1":{"Tracker":{"PubID":5890,"PageURL":"","Timestamp":0,"IID":"","ProfileID":"0","VersionID":"0","SlotID":"","Adunit":"","PartnerInfo":{"PartnerID":"pubmatic","BidderCode":"pubmatic","KGPV":"","GrossECPM":0,"NetECPM":0,"BidID":"bb57a9e3-fdc2-4772-8071-112dd7f50a6a","OrigBidID":"bid-id-1","AdSize":"0x0","AdDuration":0,"Adformat":"banner","ServerSide":1,"Advertiser":"","FloorValue":0,"FloorRuleValue":0,"DealID":"-1"},"RewardedInventory":0,"SURL":"","Platform":0,"SSAI":"","AdPodSlot":0,"TestGroup":0,"Origin":"","FloorSkippedFlag":null,"FloorModelVersion":"","FloorSource":null,"FloorType":0,"CustomDimensions":"","LoggerData":{"KGPSV":"","FloorProvider":"","FloorFetchStatus":null}},"TrackerURL":"https:?adv=\u0026af=banner\u0026aps=0\u0026au=\u0026bc=pubmatic\u0026bidid=bb57a9e3-fdc2-4772-8071-112dd7f50a6a\u0026di=-1\u0026eg=0\u0026en=0\u0026ft=0\u0026iid=\u0026kgpv=\u0026nwid=6\u0026orig=\u0026origbidid=bid-id-1\u0026pdvid=0\u0026pid=0\u0026plt=0\u0026pn=pubmatic\u0026psz=0x0\u0026pubid=5890\u0026purl=\u0026sl=1\u0026slot=\u0026ss=1\u0026tst=0","ErrorURL":"","Price":5,"PriceModel":"CPM","PriceCurrency":""}},"PrebidBidderCode":null,"ImpBidCtx":{"Div1":{"ImpID":"","TagID":"","Div":"","SlotName":"","AdUnitName":"","Secure":0,"BidFloor":0,"BidFloorCur":"","IsRewardInventory":null,"Banner":true,"Video":{"mimes":null},"Native":null,"IncomingSlots":null,"Type":"video","Bidders":{"pubmatic":{"PartnerID":123,"PrebidBidderCode":"pubmatic","MatchedSlot":"","KGP":"","KGPV":"","IsRegex":false,"Params":null,"VASTTagFlag":false,"VASTTagFlags":null}},"NonMapped":null,"NewExt":null,"BidCtx":{"bid-id-1":{"prebid":{"meta":{"adaptercode":"pubmatic","advertiserId":4098,"agencyId":4098,"demandSource":"6","mediaType":"banner","networkId":6},"type":"banner","bidid":"bb57a9e3-fdc2-4772-8071-112dd7f50a6a"},"refreshInterval":30,"crtype":"banner","dspid":6,"netecpm":5,"origbidcpm":8,"origbidcur":"USD","EG":0,"EN":0}},"BannerAdUnitCtx":{"MatchedSlot":"","IsRegex":false,"MatchedRegex":"","SelectedSlotAdUnitConfig":null,"AppliedSlotAdUnitConfig":null,"UsingDefaultConfig":false,"AllowedConnectionTypes":null},"VideoAdUnitCtx":{"MatchedSlot":"","IsRegex":false,"MatchedRegex":"","SelectedSlotAdUnitConfig":null,"AppliedSlotAdUnitConfig":null,"UsingDefaultConfig":false,"AllowedConnectionTypes":null},"BidderError":"","IsAdPodRequest":false}},"Aliases":null,"NewReqExt":null,"ResponseExt":{"responsetimemillis":{"pubmatic":8}},"MarketPlaceBidders":null,"AdapterThrottleMap":null,"AdUnitConfig":null,"Source":"","Origin":"","SendAllBids":false,"WinningBids":{"Div1":{"ID":"bid-id-1","NetEcpm":5,"BidDealTierSatisfied":false,"Nbr":null}},"DroppedBids":null,"DefaultBids":{},"SeatNonBids":{},"BidderResponseTimeMillis":{"pubmatic":8},"Endpoint":"","PubIDStr":"5890","ProfileIDStr":"","MetricsEngine":{},"ReturnAllBidStatus":true,"Sshb":"","DCName":"","CachePutMiss":0,"MatchedImpression":{"pubmatic":0},"CustomDimensions":null}]`},
				},
				err:         nil,
				bidResponse: json.RawMessage(`{"id":"12345","seatbid":[{"bid":[{"id":"bid-id-1","impid":"Div1","price":5,"adm":"\u003cimg src=\"http://ads.pubmatic.com/AdTag/728x90.png\"\u003e\u003c/img\u003e\u003cdiv style=\"position:absolute;left:0px;top:0px;visibility:hidden;\"\u003e\u003cimg src=\"https://t.pubmatic.com/wt?adv=\u0026af=banner\u0026aps=0\u0026au=%2F43743431%2FDMDemo\u0026bc=appnexus\u0026bidid=4033c510-6d67-4af6-b53f-682ff1a580c3\u0026di=-1\u0026eg=14\u0026en=14\u0026frv=1.57\u0026ft=0\u0026fv=1.57\u0026iid=429d469d-8cfb-495a-9f0c-5f48aa0ede40\u0026kgpv=\u0026orig=ebay.com\u0026origbidid=718825584\u0026pdvid=1\u0026pid=22503\u0026plt=1\u0026pn=appnexus\u0026psz=728x90\u0026pubid=5890\u0026purl=http%3A%2F%2Febay.com%2Finte%2Fautomation%2Fs2s_activation%2Fbanner-with-gdpr-pubmatic-denied-defaultbidder.html%3Fprofileid%3D22503%26pwtv%3D1%26pwtvc%3D1%26appnexus_banner_fixedbid%3D14%26fixedbid%3D1%26debug%3D1\u0026sl=1\u0026slot=%2F43743431%2FDMDemo\u0026ss=1\u0026tst=1704357774\"\u003e\u003c/div\u003e\u003cdiv style=\"position:absolute;left:0px;top:0px;visibility:hidden;\"\u003e\u003cimg src=\"https:?adv=\u0026af=banner\u0026aps=0\u0026au=\u0026bc=pubmatic\u0026bidid=bb57a9e3-fdc2-4772-8071-112dd7f50a6a\u0026di=-1\u0026eg=0\u0026en=0\u0026ft=0\u0026iid=\u0026kgpv=\u0026nwid=6\u0026orig=\u0026origbidid=bid-id-1\u0026pdvid=0\u0026pid=0\u0026plt=0\u0026pn=pubmatic\u0026psz=0x0\u0026pubid=5890\u0026purl=\u0026sl=1\u0026slot=\u0026ss=1\u0026tst=0\"\u003e\u003c/div\u003e","ext":{"prebid":{"meta":{"adaptercode":"pubmatic","advertiserId":4098,"agencyId":4098,"demandSource":"6","mediaType":"banner","networkId":6},"type":"banner","bidid":"bb57a9e3-fdc2-4772-8071-112dd7f50a6a"},"refreshInterval":30,"crtype":"banner","dspid":6,"netecpm":5,"origbidcpm":8,"origbidcur":"USD","ibv":true,"clicktrackers":["http://clicktracker1.com","http://clicktracker2.com"]}}],"seat":"pubmatic"}],"ext":{"responsetimemillis":{"pubmatic":8},"matchedimpression":{"pubmatic":0}}}`),
			},
		},
		{
			name: "google_gam_sdk_bidding_preserves_bid_exp_when_bidexp_enf_is_1",
			args: args{
				ctx: nil,
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							PubID:                 5890,
							PubIDStr:              "5890",
							Platform:              models.PLATFORM_APP,
							AppSubIntegrationPath: ptrutil.ToPtr(models.AppSubIntegrationPathIDGoogleAdManagerSDKBidding),
							ImpBidCtx: map[string]models.ImpCtx{
								"Div1": {
									Bidders: map[string]models.PartnerData{
										"pubmatic": {
											PartnerID:        123,
											PrebidBidderCode: "pubmatic",
										},
									},
									Type:     "banner",
									IsBanner: true,
								},
							},
							BidderResponseTimeMillis: map[string]int{},
							SeatNonBids:              map[string][]openrtb_ext.NonBid{},
							PartnerConfigMap: map[int]map[string]string{
								123: {
									models.PARTNER_ID:          "123",
									models.PREBID_PARTNER_NAME: "pubmatic",
									models.BidderCode:          "pubmatic",
									models.SERVER_SIDE_FLAG:    "1",
									models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
									models.TIMEOUT:             "200",
								},
								-1: {
									models.DisplayVersionID: "1",
									"refreshInterval":       "30",
									"rev_share":             "0.5",
								},
							},
						},
					},
				},
				payload: hookstage.AuctionResponsePayload{
					BidResponse: &openrtb2.BidResponse{
						ID: "12345",
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "pubmatic",
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "Div1",
										Price: 5,
										Exp:   300,
										Ext:   json.RawMessage(`{"dspid":6,"origbidcpm":8,"origbidcur":"USD","bidexp_enf":1,"prebid":{"bidid":"bb57a9e3-fdc2-4772-8071-112dd7f50a6a","meta":{"adaptercode":"pubmatic","advertiserId":4098,"agencyId":4098,"demandSource":"6","mediaType":"banner","networkId":6},"type":"banner"}}`),
									},
								},
							},
						},
						Ext: json.RawMessage(`{"responsetimemillis":{"pubmatic":8}}`),
					},
				},
			},
			setup: func() *mock_metrics.MockMetricsEngine {
				mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats(models.PLATFORM_APP, "5890", "pubmatic")
				mockEngine.EXPECT().RecordPartnerResponseTimeStats("5890", "pubmatic", 8)
				mockEngine.EXPECT().RecordPublisherResponseTimeStats("5890", gomock.Any())
				mockEngine.EXPECT().RecordPublisherPartnerNoCookieStats("5890", gomock.Any()).AnyTimes()
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
				mockFeature.EXPECT().IsFscApplicable(gomock.Any(), gomock.Any(), gomock.Any()).Return(false).AnyTimes()
				return mockEngine
			},
			doMutate: true,
			want: want{
				result:      hookstage.HookResult[hookstage.AuctionResponsePayload]{},
				err:         nil,
				bidResponse: json.RawMessage(`{"id":"12345","seatbid":[{"bid":[{"id":"bid-id-1","impid":"Div1","price":5,"adm":"\u003cdiv style=\"position:absolute;left:0px;top:0px;visibility:hidden;\"\u003e\u003cimg src=\"https:?adv=\u0026af=banner\u0026aps=0\u0026asip=14\u0026au=\u0026bc=pubmatic\u0026bexp=300\u0026bexpef=1\u0026bidid=bb57a9e3-fdc2-4772-8071-112dd7f50a6a\u0026di=-1\u0026eg=0\u0026en=0\u0026ft=0\u0026iid=\u0026kgpv=\u0026nwid=6\u0026orig=\u0026origbidid=bid-id-1\u0026pdvid=0\u0026pid=0\u0026plt=0\u0026pn=pubmatic\u0026psz=0x0\u0026pubid=5890\u0026purl=\u0026sl=1\u0026slot=\u0026ss=1\u0026tst=0\"\u003e\u003c/div\u003e","exp":300,"ext":{"prebid":{"meta":{"adaptercode":"pubmatic","advertiserId":4098,"agencyId":4098,"demandSource":"6","mediaType":"banner","networkId":6},"targeting":{"pwtbst":"1","pwtbst_pubmatic":"1","pwtecp":"5.00","pwtecp_pubmatic":"5.00","pwtpid":"pubmatic","pwtpid_pubmatic":"pubmatic","pwtplt":"inapp","pwtplt_pubmatic":"inapp","pwtsid":"bid-id-1","pwtsid_pubmatic":"bid-id-1","pwtsz":"0x0","pwtsz_pubmatic":"0x0"},"type":"banner","bidid":"bb57a9e3-fdc2-4772-8071-112dd7f50a6a"},"refreshInterval":30,"crtype":"banner","dspid":6,"netecpm":5,"origbidcpm":8,"origbidcur":"USD"}}],"seat":"pubmatic"}],"ext":{"responsetimemillis":{"pubmatic":8},"matchedimpression":{"pubmatic":0}}}`),
			},
		},
		{
			name: "valid bidResponse with video bids",
			args: args{
				ctx: nil,
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							PubID:    5890,
							PubIDStr: "5890",
							Platform: "web",
							NewReqExt: &models.RequestExt{
								ExtRequest: openrtb_ext.ExtRequest{
									Prebid: openrtb_ext.ExtRequestPrebid{ExtOWRequestPrebid: openrtb_ext.ExtOWRequestPrebid{GoogleSSUFeatureEnabled: true}},
								},
							},
							ImpBidCtx: map[string]models.ImpCtx{
								"Div1": {
									Bidders: map[string]models.PartnerData{
										"pubmatic": {
											PartnerID:        123,
											PrebidBidderCode: "pubmatic",
										},
									},
									Video: &openrtb2.Video{
										MaxDuration:    20,
										MinDuration:    10,
										SkipAfter:      2,
										Skip:           ptrutil.ToPtr[int8](1),
										SkipMin:        1,
										BAttr:          []adcom1.CreativeAttribute{adcom1.CreativeAttribute(1)},
										PlaybackMethod: []adcom1.PlaybackMethod{adcom1.PlaybackPageLoadSoundOn},
									},
									Type:     "video",
									IsBanner: false,
								},
							},
							PrebidBidderCode: map[string]string{
								"pubmatic": "pubmatic",
							},
							BidderResponseTimeMillis: map[string]int{},
							SeatNonBids:              map[string][]openrtb_ext.NonBid{},
							ReturnAllBidStatus:       true,
							Debug:                    true,
							PartnerConfigMap: map[int]map[string]string{
								123: {
									models.PARTNER_ID:          "123",
									models.PREBID_PARTNER_NAME: "pubmatic",
									models.BidderCode:          "pubmatic",
									models.SERVER_SIDE_FLAG:    "1",
									models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
									models.TIMEOUT:             "200",
								},
								-1: {
									models.DisplayVersionID: "1",
									"refreshInterval":       "30",
									"rev_share":             "0.5",
								},
							},
						},
					},
				},
				payload: hookstage.AuctionResponsePayload{
					BidResponse: &openrtb2.BidResponse{
						ID: "12345",
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "Div1",
										Price: 5,
										AdM:   "<VAST version=\"3.0\"><Ad><Wrapper></Wrapper></Ad></VAST>",
										Ext:   json.RawMessage(`{"bidtype":0,"deal_channel":1,"dspid":6,"mbmfv":4,"origbidcpm":8,"origbidcur":"USD","prebid":{"bidid":"bb57a9e3-fdc2-4772-8071-112dd7f50a6a","meta":{"adaptercode":"pubmatic","advertiserId":4098,"agencyId":4098,"demandSource":"6","mediaType":"banner","networkId":6},"targeting":{"hb_bidder_pubmatic":"pubmatic","hb_deal_pubmatic":"PUBDEAL1","hb_pb_pubmatic":"8.00","hb_size_pubmatic":"728x90"},"type":"video","video":{"duration":0,"primary_category":"","vasttagid":""}},"clicktrackers":["http://clicktracker1.com","http://clicktracker2.com"]}`),
									},
								},
								Seat: "pubmatic",
							},
						},
						Ext: json.RawMessage(`{"responsetimemillis":{"pubmatic":8}}`),
					},
				},
			},
			setup: func() *mock_metrics.MockMetricsEngine {
				mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("web", "5890", "pubmatic")
				mockEngine.EXPECT().RecordPartnerResponseTimeStats("5890", "pubmatic", 8)
				mockEngine.EXPECT().RecordPublisherResponseTimeStats("5890", gomock.Any())
				mockEngine.EXPECT().RecordPublisherPartnerNoCookieStats("5890", gomock.Any()).AnyTimes()
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
				mockFeature.EXPECT().IsFscApplicable(gomock.Any(), gomock.Any(), gomock.Any()).Return(false).AnyTimes()
				return mockEngine
			},
			doMutate: true,
			want: want{
				result:      hookstage.HookResult[hookstage.AuctionResponsePayload]{},
				err:         nil,
				bidResponse: json.RawMessage(`{"id":"12345","seatbid":[{"bid":[{"id":"bid-id-1","impid":"Div1","price":5,"adm":"\u003cVAST version=\"3.0\"\u003e\u003cAd\u003e\u003cWrapper\u003e\u003cImpression\u003e\u003c![CDATA[https:?adv=\u0026af=video\u0026aps=0\u0026au=\u0026bc=pubmatic\u0026bidid=bb57a9e3-fdc2-4772-8071-112dd7f50a6a\u0026di=-1\u0026dur=20\u0026eg=0\u0026en=0\u0026frv=4\u0026ft=0\u0026fv=4\u0026iid=\u0026kgpv=\u0026mbmf=1\u0026nwid=6\u0026orig=\u0026origbidid=bid-id-1\u0026pdvid=0\u0026pid=0\u0026plt=0\u0026pn=pubmatic\u0026psz=0x0\u0026pubid=5890\u0026purl=\u0026sl=1\u0026slot=\u0026ss=1\u0026tst=0]]\u003e\u003c/Impression\u003e\u003cExtensions\u003e\u003cExtension\u003e\u003cPricing model=\"CPM\" currency=\"USD\"\u003e\u003c![CDATA[5]]\u003e\u003c/Pricing\u003e\u003c/Extension\u003e\u003c/Extensions\u003e\u003c/Wrapper\u003e\u003c/Ad\u003e\u003c/VAST\u003e","ext":{"prebid":{"meta":{"adaptercode":"pubmatic","advertiserId":4098,"agencyId":4098,"demandSource":"6","mediaType":"banner","networkId":6},"type":"video","video":{"duration":20,"primary_category":"","vasttagid":""},"bidid":"bb57a9e3-fdc2-4772-8071-112dd7f50a6a"},"refreshInterval":30,"crtype":"video","video":{"minduration":10,"maxduration":20,"skip":1,"skipmin":1,"skipafter":2,"battr":[1],"playbackmethod":[1]},"dspid":6,"netecpm":5,"origbidcpm":8,"origbidcur":"USD"}}],"seat":"pubmatic"}],"ext":{"responsetimemillis":{"pubmatic":8},"matchedimpression":{"pubmatic":0}}}`),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mockEngine *mock_metrics.MockMetricsEngine
			if tt.setup != nil {
				mockEngine = tt.setup()
			}
			m := OpenWrap{
				cache:        mockCache,
				metricEngine: mockEngine,
				pubFeatures:  mockFeature,
			}
			moduleCtx, ok := tt.args.moduleCtx.ModuleContext["rctx"]
			if ok {
				rCtx, ok := moduleCtx.(models.RequestCtx)
				if ok {
					rCtx.MetricsEngine = mockEngine
					tt.args.moduleCtx.ModuleContext["rctx"] = rCtx
				}
			}
			hookResult, err := m.handleAuctionResponseHook(tt.args.ctx, tt.args.moduleCtx, tt.args.payload)
			assert.Equal(t, tt.want.err, err, tt.name)
			if tt.doMutate {
				mutations := hookResult.ChangeSet.Mutations()
				assert.NotEmpty(t, mutations, tt.name)
				for _, mut := range mutations {
					result, err := mut.Apply(tt.args.payload)
					gotBidResponse, _ := json.Marshal(result.BidResponse)
					assert.Nil(t, err, tt.name)
					assert.Equal(t, string(tt.want.bidResponse), string(gotBidResponse), tt.name)
				}
				return
			}
			assert.Equal(t, tt.want.result.DebugMessages, hookResult.DebugMessages, tt.name)
		})
	}
}

func TestAuctionResponseHookForApplovinMax(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockCache := mock_cache.NewMockCache(ctrl)
	mockFeature := mock_feature.NewMockFeature(ctrl)
	defer ctrl.Finish()

	type args struct {
		ctx       context.Context
		moduleCtx hookstage.ModuleInvocationContext
		payload   hookstage.AuctionResponsePayload
	}

	type want struct {
		bidResponse *openrtb2.BidResponse
		err         error
	}

	tests := []struct {
		name             string
		args             args
		want             want
		getMetricsEngine func() *mock_metrics.MockMetricsEngine
	}{
		{
			name: "update_the_bid_response_in_applovin_max_format",
			args: args{
				ctx: nil,
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							Platform: models.PLATFORM_VIDEO,
							Endpoint: models.EndpointAppLovinMax,
							ImpBidCtx: map[string]models.ImpCtx{
								"789": {
									ImpID: "789",
								},
							},
							BidderResponseTimeMillis: map[string]int{},
							Trackers: map[string]models.OWTracker{
								"456": {
									TrackerURL: `Tracker URL`,
									ErrorURL:   `Error URL`,
									Price:      1.2,
								},
							},
						},
					},
				},
				payload: hookstage.AuctionResponsePayload{
					BidResponse: &openrtb2.BidResponse{
						ID:    "123",
						BidID: "456",
						Cur:   "USD",
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "456",
										ImpID: "789",
										Price: 1.0,
										AdM:   `<VAST version='3.0'><Ad id='601364'><InLine><AdSystem>Acudeo Compatible</AdSystem><AdTitle>VAST 2.0 Instream Test 1</AdTitle><Description>VAST 2.0 Instream Test 1</Description><Impression><![CDATA[http://172.16.4.213/AdServer/AdDisplayTrackerServlet?operId=1&pubId=5890&siteId=47163&adId=1405268&adType=13&adServerId=243&kefact=70.000000&kaxefact=70.000000&kadNetFrequecy=0&kadwidth=0&kadheight=0&kadsizeid=97&kltstamp=1529929473&indirectAdId=0&adServerOptimizerId=2&ranreq=0.1&kpbmtpfact=100.000000&dcId=1&tldId=0&passback=0&svr=MADS1107&ekefact=Ad8wW91TCwCmdG0jlfjXn7Tyzh20hnTVx-m5DoNSep-RXGDr&ekaxefact=Ad8wWwRUCwAGir4Zzl1eF0bKiC-qrCV0D0yp_eE7YizB_BQk&ekpbmtpfact=Ad8wWxRUCwD7qgzwwPE2LnS5-Ou19uO5amJl1YT6-XVFvQ41&imprId=48F73E1A-7F23-443D-A53C-30EE6BBF5F7F&oid=48F73E1A-7F23-443D-A53C-30EE6BBF5F7F&crID=creative-1_1_2&ucrid=160175026529250297&campaignId=17050&creativeId=0&pctr=0.000000&wDSPByrId=511&wDspId=6&wbId=0&wrId=0&wAdvID=3170&isRTB=1&rtbId=EBCA079F-8D7C-45B8-B733-92951F670AA1&pmZoneId=zone1&pageURL=www.yahoo.com&lpu=ae.com]]></Impression><Impression>https://dsptracker.com/{PSPM}</Impression><Error><![CDATA[http://172.16.4.213/track?operId=7&p=5890&s=47163&a=1405268&wa=243&ts=1529929473&wc=17050&crId=creative-1_1_2&ucrid=160175026529250297&impid=48F73E1A-7F23-443D-A53C-30EE6BBF5F7F&advertiser_id=3170&ecpm=70.000000&er=[ERRORCODE]]]></Error><Error><![CDATA[https://Errortrack.com?p=1234&er=[ERRORCODE]]]></Error><Creatives><Creative AdID='601364'><Linear skipoffset='20%'><Duration>00:00:04</Duration><VideoClicks><ClickTracking><![CDATA[http://172.16.4.213/track?operId=7&p=5890&s=47163&a=1405268&wa=243&ts=1529929473&wc=17050&crId=creative-1_1_2&ucrid=160175026529250297&impid=48F73E1A-7F23-443D-A53C-30EE6BBF5F7F&advertiser_id=3170&ecpm=70.000000&e=99]]></ClickTracking><ClickThrough>https://www.sample.com</ClickThrough></VideoClicks><MediaFiles><MediaFile delivery='progressive' type='video/mp4' bitrate='500' width='400' height='300' scalable='true' maintainAspectRatio='true'><![CDATA[https://stagingnyc.pubmatic.com:8443/video/Shashank/mediaFileHost/media/mp4-sample-1.mp4]]></MediaFile><MediaFile delivery='progressive' type='video/mp4' bitrate='500' width='400' height='300' scalable='true' maintainAspectRatio='true'><![CDATA[https://stagingnyc.pubmatic.com:8443/video/Shashank/mediaFileHost/media/mp4-sample-2.mp4]]></MediaFile></MediaFiles></Linear></Creative></Creatives></InLine></Ad></VAST>`,
										BURL:  "http://example.com",
										Ext:   json.RawMessage(`{"key":"value"}`),
									},
								},
								Seat: "pubmatic",
							},
						},
						Ext: json.RawMessage(`{"key":"value"}`),
					},
				},
			},
			want: want{
				bidResponse: &openrtb2.BidResponse{
					ID:    "123",
					BidID: "456",
					Cur:   "USD",
					SeatBid: []openrtb2.SeatBid{
						{
							Bid: []openrtb2.Bid{
								{
									ID:    "456",
									ImpID: "789",
									Price: 1.0,
									BURL:  `https:?adv=&af=video&aps=0&au=&bc=pubmatic&bidid=456&di=-1&eg=1&en=1&ft=0&iid=&kgpv=&orig=&origbidid=456&pdvid=0&pid=0&plt=0&pn=pubmatic&psz=0x0&pubid=0&purl=&sl=1&slot=&ss=1&tst=0&owsspburl=http%3A%2F%2Fexample.com`,
									Ext:   json.RawMessage(`{"signaldata":"{\"id\":\"123\",\"seatbid\":[{\"bid\":[{\"id\":\"456\",\"impid\":\"789\",\"price\":1,\"burl\":\"https:?adv=\\u0026af=video\\u0026aps=0\\u0026au=\\u0026bc=pubmatic\\u0026bidid=456\\u0026di=-1\\u0026eg=1\\u0026en=1\\u0026ft=0\\u0026iid=\\u0026kgpv=\\u0026orig=\\u0026origbidid=456\\u0026pdvid=0\\u0026pid=0\\u0026plt=0\\u0026pn=pubmatic\\u0026psz=0x0\\u0026pubid=0\\u0026purl=\\u0026sl=1\\u0026slot=\\u0026ss=1\\u0026tst=0\\u0026owsspburl=http%3A%2F%2Fexample.com\",\"adm\":\"\\u003cVAST version=\\\"3.0\\\"\\u003e\\u003cAd id=\\\"601364\\\"\\u003e\\u003cInLine\\u003e\\u003cAdSystem\\u003e\\u003c![CDATA[Acudeo Compatible]]\\u003e\\u003c/AdSystem\\u003e\\u003cAdTitle\\u003e\\u003c![CDATA[VAST 2.0 Instream Test 1]]\\u003e\\u003c/AdTitle\\u003e\\u003cDescription\\u003e\\u003c![CDATA[VAST 2.0 Instream Test 1]]\\u003e\\u003c/Description\\u003e\\u003cImpression\\u003e\\u003c![CDATA[http://172.16.4.213/AdServer/AdDisplayTrackerServlet?operId=1\\u0026pubId=5890\\u0026siteId=47163\\u0026adId=1405268\\u0026adType=13\\u0026adServerId=243\\u0026kefact=70.000000\\u0026kaxefact=70.000000\\u0026kadNetFrequecy=0\\u0026kadwidth=0\\u0026kadheight=0\\u0026kadsizeid=97\\u0026kltstamp=1529929473\\u0026indirectAdId=0\\u0026adServerOptimizerId=2\\u0026ranreq=0.1\\u0026kpbmtpfact=100.000000\\u0026dcId=1\\u0026tldId=0\\u0026passback=0\\u0026svr=MADS1107\\u0026ekefact=Ad8wW91TCwCmdG0jlfjXn7Tyzh20hnTVx-m5DoNSep-RXGDr\\u0026ekaxefact=Ad8wWwRUCwAGir4Zzl1eF0bKiC-qrCV0D0yp_eE7YizB_BQk\\u0026ekpbmtpfact=Ad8wWxRUCwD7qgzwwPE2LnS5-Ou19uO5amJl1YT6-XVFvQ41\\u0026imprId=48F73E1A-7F23-443D-A53C-30EE6BBF5F7F\\u0026oid=48F73E1A-7F23-443D-A53C-30EE6BBF5F7F\\u0026crID=creative-1_1_2\\u0026ucrid=160175026529250297\\u0026campaignId=17050\\u0026creativeId=0\\u0026pctr=0.000000\\u0026wDSPByrId=511\\u0026wDspId=6\\u0026wbId=0\\u0026wrId=0\\u0026wAdvID=3170\\u0026isRTB=1\\u0026rtbId=EBCA079F-8D7C-45B8-B733-92951F670AA1\\u0026pmZoneId=zone1\\u0026pageURL=www.yahoo.com\\u0026lpu=ae.com]]\\u003e\\u003c/Impression\\u003e\\u003cImpression\\u003e\\u003c![CDATA[https://dsptracker.com/{PSPM}]]\\u003e\\u003c/Impression\\u003e\\u003cError\\u003e\\u003c![CDATA[http://172.16.4.213/track?operId=7\\u0026p=5890\\u0026s=47163\\u0026a=1405268\\u0026wa=243\\u0026ts=1529929473\\u0026wc=17050\\u0026crId=creative-1_1_2\\u0026ucrid=160175026529250297\\u0026impid=48F73E1A-7F23-443D-A53C-30EE6BBF5F7F\\u0026advertiser_id=3170\\u0026ecpm=70.000000\\u0026er=[ERRORCODE]]]\\u003e\\u003c/Error\\u003e\\u003cError\\u003e\\u003c![CDATA[https://Errortrack.com?p=1234\\u0026er=[ERRORCODE]]]\\u003e\\u003c/Error\\u003e\\u003cCreatives\\u003e\\u003cCreative AdID=\\\"601364\\\"\\u003e\\u003cLinear skipoffset=\\\"20%\\\"\\u003e\\u003cDuration\\u003e\\u003c![CDATA[00:00:04]]\\u003e\\u003c/Duration\\u003e\\u003cVideoClicks\\u003e\\u003cClickTracking\\u003e\\u003c![CDATA[http://172.16.4.213/track?operId=7\\u0026p=5890\\u0026s=47163\\u0026a=1405268\\u0026wa=243\\u0026ts=1529929473\\u0026wc=17050\\u0026crId=creative-1_1_2\\u0026ucrid=160175026529250297\\u0026impid=48F73E1A-7F23-443D-A53C-30EE6BBF5F7F\\u0026advertiser_id=3170\\u0026ecpm=70.000000\\u0026e=99]]\\u003e\\u003c/ClickTracking\\u003e\\u003cClickThrough\\u003e\\u003c![CDATA[https://www.sample.com]]\\u003e\\u003c/ClickThrough\\u003e\\u003c/VideoClicks\\u003e\\u003cMediaFiles\\u003e\\u003cMediaFile delivery=\\\"progressive\\\" type=\\\"video/mp4\\\" bitrate=\\\"500\\\" width=\\\"400\\\" height=\\\"300\\\" scalable=\\\"true\\\" maintainAspectRatio=\\\"true\\\"\\u003e\\u003c![CDATA[https://stagingnyc.pubmatic.com:8443/video/Shashank/mediaFileHost/media/mp4-sample-1.mp4]]\\u003e\\u003c/MediaFile\\u003e\\u003cMediaFile delivery=\\\"progressive\\\" type=\\\"video/mp4\\\" bitrate=\\\"500\\\" width=\\\"400\\\" height=\\\"300\\\" scalable=\\\"true\\\" maintainAspectRatio=\\\"true\\\"\\u003e\\u003c![CDATA[https://stagingnyc.pubmatic.com:8443/video/Shashank/mediaFileHost/media/mp4-sample-2.mp4]]\\u003e\\u003c/MediaFile\\u003e\\u003c/MediaFiles\\u003e\\u003c/Linear\\u003e\\u003c/Creative\\u003e\\u003c/Creatives\\u003e\\u003cPricing model=\\\"CPM\\\" currency=\\\"USD\\\"\\u003e\\u003c![CDATA[1]]\\u003e\\u003c/Pricing\\u003e\\u003c/InLine\\u003e\\u003c/Ad\\u003e\\u003c/VAST\\u003e\",\"ext\":{\"prebid\":{},\"crtype\":\"video\",\"netecpm\":1}}],\"seat\":\"pubmatic\"}],\"bidid\":\"456\",\"cur\":\"USD\",\"ext\":{\"matchedimpression\":{}}}"}`),
								},
							},
						},
					},
				},
				err: nil,
			},
			getMetricsEngine: func() *mock_metrics.MockMetricsEngine {
				mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats(gomock.Any(), gomock.Any(), gomock.Any())
				mockEngine.EXPECT().RecordPublisherResponseTimeStats(gomock.Any(), gomock.Any())
				mockFeature.EXPECT().IsFscApplicable(gomock.Any(), gomock.Any(), gomock.Any()).Return(false)
				mockEngine.EXPECT().RecordPartnerResponseTimeStats(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
				mockFeature.EXPECT().GetImpCountingMethodEnabledBidders().Return(map[string]struct{}{})
				mockFeature.EXPECT().GetEnabledPerformanceDSPs().Return(map[int]struct{}{})
				mockFeature.EXPECT().GetInViewEnabledPublishers().Return(map[int]struct{}{})
				return mockEngine
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := OpenWrap{
				metricEngine: tt.getMetricsEngine(),
				cache:        mockCache,
				pubFeatures:  mockFeature,
			}
			hookResult, err := o.handleAuctionResponseHook(tt.args.ctx, tt.args.moduleCtx, tt.args.payload)
			assert.Equal(t, tt.want.err, err, tt.name)
			mutations := hookResult.ChangeSet.Mutations()
			assert.NotEmpty(t, mutations, tt.name)
			for _, mut := range mutations {
				result, err := mut.Apply(tt.args.payload)
				assert.Nil(t, err, tt.name)
				assert.Equal(t, tt.want.bidResponse, result.BidResponse, tt.name)
			}
		})
	}
}
func TestOpenWrapHandleAuctionResponseHookForACT(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockCache := mock_cache.NewMockCache(ctrl)
	defer ctrl.Finish()
	mockFeature := mock_feature.NewMockFeature(ctrl)

	type want struct {
		result      hookstage.HookResult[hookstage.AuctionResponsePayload]
		bidResponse json.RawMessage
		err         error
	}
	type args struct {
		ctx       context.Context
		moduleCtx hookstage.ModuleInvocationContext
		payload   hookstage.AuctionResponsePayload
	}
	tests := []struct {
		name     string
		args     args
		want     want
		doMutate bool
		setup    func() *mock_metrics.MockMetricsEngine
	}{
		{
			name: "valid_bidResponse_with_banner_bids_instl1_isappopenad_not_present_act_disabled",
			args: args{
				ctx: nil,
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							PubID:    5890,
							PubIDStr: "5890",
							Platform: "web",
							ImpBidCtx: map[string]models.ImpCtx{
								"Div1": {
									Bidders: map[string]models.PartnerData{
										"pubmatic": {
											PartnerID:        123,
											PrebidBidderCode: "pubmatic",
										},
									},
									Video:    &openrtb2.Video{},
									Type:     "video",
									IsBanner: true,
									Instl:    1,
								},
							},
							BidderResponseTimeMillis: map[string]int{},
							SeatNonBids:              map[string][]openrtb_ext.NonBid{},
							ReturnAllBidStatus:       true,
							Debug:                    true,
							ClientConfigFlag:         1,
							PartnerConfigMap: map[int]map[string]string{
								123: {
									models.PARTNER_ID:          "123",
									models.PREBID_PARTNER_NAME: "pubmatic",
									models.BidderCode:          "pubmatic",
									models.SERVER_SIDE_FLAG:    "1",
									models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
									models.TIMEOUT:             "200",
								},
								-1: {
									models.DisplayVersionID: "1",
									"refreshInterval":       "30",
									"rev_share":             "0.5",
								},
							},
						},
					},
				},
				payload: hookstage.AuctionResponsePayload{
					BidResponse: &openrtb2.BidResponse{
						ID: "12345",
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "Div1",
										Price: 5,
										AdM:   "<img src=\"http://ads.pubmatic.com/AdTag/728x90.png\"></img><div style=\"position:absolute;left:0px;top:0px;visibility:hidden;\"><img src=\"https://t.pubmatic.com/wt?adv=&af=banner&aps=0&au=%2F43743431%2FDMDemo&bc=appnexus&bidid=4033c510-6d67-4af6-b53f-682ff1a580c3&di=-1&eg=14&en=14&frv=1.57&ft=0&fv=1.57&iid=429d469d-8cfb-495a-9f0c-5f48aa0ede40&kgpv=&orig=ebay.com&origbidid=718825584&pdvid=1&pid=22503&plt=1&pn=appnexus&psz=728x90&pubid=5890&purl=http%3A%2F%2Febay.com%2Finte%2Fautomation%2Fs2s_activation%2Fbanner-with-gdpr-pubmatic-denied-defaultbidder.html%3Fprofileid%3D22503%26pwtv%3D1%26pwtvc%3D1%26appnexus_banner_fixedbid%3D14%26fixedbid%3D1%26debug%3D1&sl=1&slot=%2F43743431%2FDMDemo&ss=1&tst=1704357774\"></div>",
										Ext:   json.RawMessage(`{"bidtype":0,"deal_channel":1,"dspid":6,"origbidcpm":8,"origbidcur":"USD","ibv":true,"prebid":{"bidid":"bb57a9e3-fdc2-4772-8071-112dd7f50a6a","meta":{"adaptercode":"pubmatic","advertiserId":4098,"agencyId":4098,"demandSource":"6","mediaType":"banner","networkId":6},"targeting":{"hb_bidder_pubmatic":"pubmatic","hb_deal_pubmatic":"PUBDEAL1","hb_pb_pubmatic":"8.00","hb_size_pubmatic":"728x90"},"type":"banner","video":{"duration":0,"primary_category":"","vasttagid":""}},"clicktrackers":["http://clicktracker1.com","http://clicktracker2.com"]}`),
									},
								},
								Seat: "pubmatic",
							},
						},
						Ext: json.RawMessage(`{"responsetimemillis":{"pubmatic":8}}`),
					},
				},
			},
			setup: func() *mock_metrics.MockMetricsEngine {
				mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("web", "5890", "pubmatic")
				mockEngine.EXPECT().RecordPartnerResponseTimeStats("5890", "pubmatic", 8)
				mockEngine.EXPECT().RecordPublisherResponseTimeStats("5890", gomock.Any())
				mockEngine.EXPECT().RecordPublisherPartnerNoCookieStats("5890", gomock.Any()).AnyTimes()
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
				mockFeature.EXPECT().IsFscApplicable(gomock.Any(), gomock.Any(), gomock.Any()).Return(false)
				mockFeature.EXPECT().IsActApplicable(gomock.Any(), gomock.Any(), gomock.Any()).Return(false)
				mockEngine.EXPECT().RecordIBVRequest("5890", gomock.Any())
				return mockEngine
			},
			doMutate: true,
			want: want{
				result: hookstage.HookResult[hookstage.AuctionResponsePayload]{
					DebugMessages: []string{`[{"PubID":5890,"ProfileID":0,"DisplayID":0,"VersionID":0,"DisplayVersionID":0,"SSAuction":0,"SummaryDisable":0,"SSAI":"","PartnerConfigMap":{"-1":{"displayVersionId":"1","refreshInterval":"30","rev_share":"0.5"},"123":{"bidderCode":"pubmatic","kgp":"_AU_@_W_x_H_","partnerId":"123","prebidPartnerName":"pubmatic","serverSideEnabled":"1","timeout":"200"}},"SupportDeals":false,"Platform":"web","LoggerImpressionID":"","ClientConfigFlag":1,"IP":"","TMax":0,"IsTestRequest":0,"ABTestConfig":0,"ABTestConfigApplied":0,"IsCTVRequest":false,"TrackerEndpoint":"","VideoErrorTrackerEndpoint":"","UA":"","Cookies":"","UidCookie":null,"KADUSERCookie":null,"ParsedUidCookie":null,"OriginCookie":"","Debug":true,"Trace":false,"PageURL":"","StartTime":0,"DevicePlatform":0,"Trackers":{"bid-id-1":{"Tracker":{"PubID":5890,"PageURL":"","Timestamp":0,"IID":"","ProfileID":"0","VersionID":"0","SlotID":"","Adunit":"","PartnerInfo":{"PartnerID":"pubmatic","BidderCode":"pubmatic","KGPV":"","GrossECPM":0,"NetECPM":0,"BidID":"bb57a9e3-fdc2-4772-8071-112dd7f50a6a","OrigBidID":"bid-id-1","AdSize":"0x0","AdDuration":0,"Adformat":"banner","ServerSide":1,"Advertiser":"","FloorValue":0,"FloorRuleValue":0,"DealID":"-1"},"RewardedInventory":0,"SURL":"","Platform":0,"SSAI":"","AdPodSlot":0,"TestGroup":0,"Origin":"","FloorSkippedFlag":null,"FloorModelVersion":"","FloorSource":null,"FloorType":0,"CustomDimensions":"","LoggerData":{"KGPSV":"","FloorProvider":"","FloorFetchStatus":null}},"TrackerURL":"https:?adv=\u0026af=banner\u0026aps=0\u0026au=\u0026bc=pubmatic\u0026bidid=bb57a9e3-fdc2-4772-8071-112dd7f50a6a\u0026di=-1\u0026eg=0\u0026en=0\u0026ft=0\u0026iid=\u0026kgpv=\u0026nwid=6\u0026orig=\u0026origbidid=bid-id-1\u0026pdvid=0\u0026pid=0\u0026plt=0\u0026pn=pubmatic\u0026psz=0x0\u0026pubid=5890\u0026purl=\u0026sl=1\u0026slot=\u0026ss=1\u0026tst=0","ErrorURL":"","Price":5,"PriceModel":"CPM","PriceCurrency":""}},"PrebidBidderCode":null,"ImpBidCtx":{"Div1":{"ImpID":"","TagID":"","Div":"","SlotName":"","AdUnitName":"","Secure":0,"BidFloor":0,"BidFloorCur":"","IsRewardInventory":null,"Banner":true,"Video":{"mimes":null},"Native":null,"IncomingSlots":null,"Type":"video","Bidders":{"pubmatic":{"PartnerID":123,"PrebidBidderCode":"pubmatic","MatchedSlot":"","KGP":"","KGPV":"","IsRegex":false,"Params":null,"VASTTagFlag":false,"VASTTagFlags":null}},"NonMapped":null,"NewExt":null,"BidCtx":{"bid-id-1":{"prebid":{"meta":{"adaptercode":"pubmatic","advertiserId":4098,"agencyId":4098,"demandSource":"6","mediaType":"banner","networkId":6},"type":"banner","bidid":"bb57a9e3-fdc2-4772-8071-112dd7f50a6a"},"refreshInterval":30,"crtype":"banner","dspid":6,"netecpm":5,"origbidcpm":8,"origbidcur":"USD","EG":0,"EN":0}},"BannerAdUnitCtx":{"MatchedSlot":"","IsRegex":false,"MatchedRegex":"","SelectedSlotAdUnitConfig":null,"AppliedSlotAdUnitConfig":null,"UsingDefaultConfig":false,"AllowedConnectionTypes":null},"VideoAdUnitCtx":{"MatchedSlot":"","IsRegex":false,"MatchedRegex":"","SelectedSlotAdUnitConfig":null,"AppliedSlotAdUnitConfig":null,"UsingDefaultConfig":false,"AllowedConnectionTypes":null},"BidderError":"","IsAdPodRequest":false}},"Aliases":null,"NewReqExt":null,"ResponseExt":{"responsetimemillis":{"pubmatic":8}},"MarketPlaceBidders":null,"AdapterThrottleMap":null,"AdUnitConfig":null,"Source":"","Origin":"","SendAllBids":false,"WinningBids":{"Div1":{"ID":"bid-id-1","NetEcpm":5,"BidDealTierSatisfied":false,"Nbr":null}},"DroppedBids":null,"DefaultBids":{},"SeatNonBids":{},"BidderResponseTimeMillis":{"pubmatic":8},"Endpoint":"","PubIDStr":"5890","ProfileIDStr":"","MetricsEngine":{},"ReturnAllBidStatus":true,"Sshb":"","DCName":"","CachePutMiss":0,"MatchedImpression":{"pubmatic":0},"CustomDimensions":null}]`},
				},
				err:         nil,
				bidResponse: json.RawMessage(`{"id":"12345","seatbid":[{"bid":[{"id":"bid-id-1","impid":"Div1","price":5,"adm":"\u003cimg src=\"http://ads.pubmatic.com/AdTag/728x90.png\"\u003e\u003c/img\u003e\u003cdiv style=\"position:absolute;left:0px;top:0px;visibility:hidden;\"\u003e\u003cimg src=\"https://t.pubmatic.com/wt?adv=\u0026af=banner\u0026aps=0\u0026au=%2F43743431%2FDMDemo\u0026bc=appnexus\u0026bidid=4033c510-6d67-4af6-b53f-682ff1a580c3\u0026di=-1\u0026eg=14\u0026en=14\u0026frv=1.57\u0026ft=0\u0026fv=1.57\u0026iid=429d469d-8cfb-495a-9f0c-5f48aa0ede40\u0026kgpv=\u0026orig=ebay.com\u0026origbidid=718825584\u0026pdvid=1\u0026pid=22503\u0026plt=1\u0026pn=appnexus\u0026psz=728x90\u0026pubid=5890\u0026purl=http%3A%2F%2Febay.com%2Finte%2Fautomation%2Fs2s_activation%2Fbanner-with-gdpr-pubmatic-denied-defaultbidder.html%3Fprofileid%3D22503%26pwtv%3D1%26pwtvc%3D1%26appnexus_banner_fixedbid%3D14%26fixedbid%3D1%26debug%3D1\u0026sl=1\u0026slot=%2F43743431%2FDMDemo\u0026ss=1\u0026tst=1704357774\"\u003e\u003c/div\u003e\u003cdiv style=\"position:absolute;left:0px;top:0px;visibility:hidden;\"\u003e\u003cimg src=\"https:?adv=\u0026af=banner\u0026aps=0\u0026au=\u0026bc=pubmatic\u0026bidid=bb57a9e3-fdc2-4772-8071-112dd7f50a6a\u0026di=-1\u0026eg=0\u0026en=0\u0026ft=0\u0026iid=\u0026kgpv=\u0026nwid=6\u0026orig=\u0026origbidid=bid-id-1\u0026pdvid=0\u0026pid=0\u0026plt=0\u0026pn=pubmatic\u0026psz=0x0\u0026pubid=5890\u0026purl=\u0026sl=1\u0026slot=\u0026ss=1\u0026tst=0\"\u003e\u003c/div\u003e","ext":{"prebid":{"meta":{"adaptercode":"pubmatic","advertiserId":4098,"agencyId":4098,"demandSource":"6","mediaType":"banner","networkId":6},"type":"banner","bidid":"bb57a9e3-fdc2-4772-8071-112dd7f50a6a"},"refreshInterval":30,"crtype":"banner","dspid":6,"netecpm":5,"origbidcpm":8,"origbidcur":"USD","ibv":true,"clicktrackers":["http://clicktracker1.com","http://clicktracker2.com"]}}],"seat":"pubmatic"}],"ext":{"responsetimemillis":{"pubmatic":8},"matchedimpression":{"pubmatic":0}}}`),
			},
		},
		{
			name: "valid_bidResponse_with_banner_bids_instl1_isappopenad_not_present",
			args: args{
				ctx: nil,
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							PubID:    5890,
							PubIDStr: "5890",
							Platform: "web",
							ImpBidCtx: map[string]models.ImpCtx{
								"Div1": {
									Bidders: map[string]models.PartnerData{
										"pubmatic": {
											PartnerID:        123,
											PrebidBidderCode: "pubmatic",
										},
									},
									Video:    &openrtb2.Video{},
									Type:     "video",
									IsBanner: true,
									Instl:    1,
								},
							},
							BidderResponseTimeMillis: map[string]int{},
							SeatNonBids:              map[string][]openrtb_ext.NonBid{},
							ReturnAllBidStatus:       true,
							Debug:                    true,
							ClientConfigFlag:         1,
							PartnerConfigMap: map[int]map[string]string{
								123: {
									models.PARTNER_ID:          "123",
									models.PREBID_PARTNER_NAME: "pubmatic",
									models.BidderCode:          "pubmatic",
									models.SERVER_SIDE_FLAG:    "1",
									models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
									models.TIMEOUT:             "200",
								},
								-1: {
									models.DisplayVersionID: "1",
									"refreshInterval":       "30",
									"rev_share":             "0.5",
								},
							},
						},
					},
				},
				payload: hookstage.AuctionResponsePayload{
					BidResponse: &openrtb2.BidResponse{
						ID: "12345",
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "Div1",
										Price: 5,
										AdM:   "<img src=\"http://ads.pubmatic.com/AdTag/728x90.png\"></img><div style=\"position:absolute;left:0px;top:0px;visibility:hidden;\"><img src=\"https://t.pubmatic.com/wt?adv=&af=banner&aps=0&au=%2F43743431%2FDMDemo&bc=appnexus&bidid=4033c510-6d67-4af6-b53f-682ff1a580c3&di=-1&eg=14&en=14&frv=1.57&ft=0&fv=1.57&iid=429d469d-8cfb-495a-9f0c-5f48aa0ede40&kgpv=&orig=ebay.com&origbidid=718825584&pdvid=1&pid=22503&plt=1&pn=appnexus&psz=728x90&pubid=5890&purl=http%3A%2F%2Febay.com%2Finte%2Fautomation%2Fs2s_activation%2Fbanner-with-gdpr-pubmatic-denied-defaultbidder.html%3Fprofileid%3D22503%26pwtv%3D1%26pwtvc%3D1%26appnexus_banner_fixedbid%3D14%26fixedbid%3D1%26debug%3D1&sl=1&slot=%2F43743431%2FDMDemo&ss=1&tst=1704357774\"></div>",
										Ext:   json.RawMessage(`{"bidtype":0,"deal_channel":1,"dspid":6,"origbidcpm":8,"origbidcur":"USD","ibv":true,"prebid":{"bidid":"bb57a9e3-fdc2-4772-8071-112dd7f50a6a","meta":{"adaptercode":"pubmatic","advertiserId":4098,"agencyId":4098,"demandSource":"6","mediaType":"banner","networkId":6},"targeting":{"hb_bidder_pubmatic":"pubmatic","hb_deal_pubmatic":"PUBDEAL1","hb_pb_pubmatic":"8.00","hb_size_pubmatic":"728x90"},"type":"banner","video":{"duration":0,"primary_category":"","vasttagid":""}},"clicktrackers":["http://clicktracker1.com","http://clicktracker2.com"]}`),
									},
								},
								Seat: "pubmatic",
							},
						},
						Ext: json.RawMessage(`{"responsetimemillis":{"pubmatic":8}}`),
					},
				},
			},
			setup: func() *mock_metrics.MockMetricsEngine {
				mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("web", "5890", "pubmatic")
				mockEngine.EXPECT().RecordPartnerResponseTimeStats("5890", "pubmatic", 8)
				mockEngine.EXPECT().RecordPublisherResponseTimeStats("5890", gomock.Any())
				mockEngine.EXPECT().RecordPublisherPartnerNoCookieStats("5890", gomock.Any()).AnyTimes()
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
				mockFeature.EXPECT().IsFscApplicable(gomock.Any(), gomock.Any(), gomock.Any()).Return(false)
				mockFeature.EXPECT().IsActApplicable(gomock.Any(), gomock.Any(), gomock.Any()).Return(true)
				mockEngine.EXPECT().RecordIBVRequest("5890", gomock.Any())
				return mockEngine
			},
			doMutate: true,
			want: want{
				result: hookstage.HookResult[hookstage.AuctionResponsePayload]{
					DebugMessages: []string{`[{"PubID":5890,"ProfileID":0,"DisplayID":0,"VersionID":0,"DisplayVersionID":0,"SSAuction":0,"SummaryDisable":0,"SSAI":"","PartnerConfigMap":{"-1":{"displayVersionId":"1","refreshInterval":"30","rev_share":"0.5"},"123":{"bidderCode":"pubmatic","kgp":"_AU_@_W_x_H_","partnerId":"123","prebidPartnerName":"pubmatic","serverSideEnabled":"1","timeout":"200"}},"SupportDeals":false,"Platform":"web","LoggerImpressionID":"","ClientConfigFlag":1,"IP":"","TMax":0,"IsTestRequest":0,"ABTestConfig":0,"ABTestConfigApplied":0,"IsCTVRequest":false,"TrackerEndpoint":"","VideoErrorTrackerEndpoint":"","UA":"","Cookies":"","UidCookie":null,"KADUSERCookie":null,"ParsedUidCookie":null,"OriginCookie":"","Debug":true,"Trace":false,"PageURL":"","StartTime":0,"DevicePlatform":0,"Trackers":{"bid-id-1":{"Tracker":{"PubID":5890,"PageURL":"","Timestamp":0,"IID":"","ProfileID":"0","VersionID":"0","SlotID":"","Adunit":"","PartnerInfo":{"PartnerID":"pubmatic","BidderCode":"pubmatic","KGPV":"","GrossECPM":0,"NetECPM":0,"BidID":"bb57a9e3-fdc2-4772-8071-112dd7f50a6a","OrigBidID":"bid-id-1","AdSize":"0x0","AdDuration":0,"Adformat":"banner","ServerSide":1,"Advertiser":"","FloorValue":0,"FloorRuleValue":0,"DealID":"-1"},"RewardedInventory":0,"SURL":"","Platform":0,"SSAI":"","AdPodSlot":0,"TestGroup":0,"Origin":"","FloorSkippedFlag":null,"FloorModelVersion":"","FloorSource":null,"FloorType":0,"CustomDimensions":"","LoggerData":{"KGPSV":"","FloorProvider":"","FloorFetchStatus":null}},"TrackerURL":"https:?adv=\u0026af=banner\u0026aps=0\u0026au=\u0026bc=pubmatic\u0026bidid=bb57a9e3-fdc2-4772-8071-112dd7f50a6a\u0026di=-1\u0026eg=0\u0026en=0\u0026ft=0\u0026iid=\u0026kgpv=\u0026nwid=6\u0026orig=\u0026origbidid=bid-id-1\u0026pdvid=0\u0026pid=0\u0026plt=0\u0026pn=pubmatic\u0026psz=0x0\u0026pubid=5890\u0026purl=\u0026sl=1\u0026slot=\u0026ss=1\u0026tst=0","ErrorURL":"","Price":5,"PriceModel":"CPM","PriceCurrency":""}},"PrebidBidderCode":null,"ImpBidCtx":{"Div1":{"ImpID":"","TagID":"","Div":"","SlotName":"","AdUnitName":"","Secure":0,"BidFloor":0,"BidFloorCur":"","IsRewardInventory":null,"Banner":true,"Video":{"mimes":null},"Native":null,"IncomingSlots":null,"Type":"video","Bidders":{"pubmatic":{"PartnerID":123,"PrebidBidderCode":"pubmatic","MatchedSlot":"","KGP":"","KGPV":"","IsRegex":false,"Params":null,"VASTTagFlag":false,"VASTTagFlags":null}},"NonMapped":null,"NewExt":null,"BidCtx":{"bid-id-1":{"prebid":{"meta":{"adaptercode":"pubmatic","advertiserId":4098,"agencyId":4098,"demandSource":"6","mediaType":"banner","networkId":6},"type":"banner","bidid":"bb57a9e3-fdc2-4772-8071-112dd7f50a6a"},"refreshInterval":30,"crtype":"banner","dspid":6,"netecpm":5,"origbidcpm":8,"origbidcur":"USD","EG":0,"EN":0}},"BannerAdUnitCtx":{"MatchedSlot":"","IsRegex":false,"MatchedRegex":"","SelectedSlotAdUnitConfig":null,"AppliedSlotAdUnitConfig":null,"UsingDefaultConfig":false,"AllowedConnectionTypes":null},"VideoAdUnitCtx":{"MatchedSlot":"","IsRegex":false,"MatchedRegex":"","SelectedSlotAdUnitConfig":null,"AppliedSlotAdUnitConfig":null,"UsingDefaultConfig":false,"AllowedConnectionTypes":null},"BidderError":"","IsAdPodRequest":false}},"Aliases":null,"NewReqExt":null,"ResponseExt":{"responsetimemillis":{"pubmatic":8}},"MarketPlaceBidders":null,"AdapterThrottleMap":null,"AdUnitConfig":null,"Source":"","Origin":"","SendAllBids":false,"WinningBids":{"Div1":{"ID":"bid-id-1","NetEcpm":5,"BidDealTierSatisfied":false,"Nbr":null}},"DroppedBids":null,"DefaultBids":{},"SeatNonBids":{},"BidderResponseTimeMillis":{"pubmatic":8},"Endpoint":"","PubIDStr":"5890","ProfileIDStr":"","MetricsEngine":{},"ReturnAllBidStatus":true,"Sshb":"","DCName":"","CachePutMiss":0,"MatchedImpression":{"pubmatic":0},"CustomDimensions":null}]`},
				},
				err:         nil,
				bidResponse: json.RawMessage(`{"id":"12345","seatbid":[{"bid":[{"id":"bid-id-1","impid":"Div1","price":5,"adm":"\u003cimg src=\"http://ads.pubmatic.com/AdTag/728x90.png\"\u003e\u003c/img\u003e\u003cdiv style=\"position:absolute;left:0px;top:0px;visibility:hidden;\"\u003e\u003cimg src=\"https://t.pubmatic.com/wt?adv=\u0026af=banner\u0026aps=0\u0026au=%2F43743431%2FDMDemo\u0026bc=appnexus\u0026bidid=4033c510-6d67-4af6-b53f-682ff1a580c3\u0026di=-1\u0026eg=14\u0026en=14\u0026frv=1.57\u0026ft=0\u0026fv=1.57\u0026iid=429d469d-8cfb-495a-9f0c-5f48aa0ede40\u0026kgpv=\u0026orig=ebay.com\u0026origbidid=718825584\u0026pdvid=1\u0026pid=22503\u0026plt=1\u0026pn=appnexus\u0026psz=728x90\u0026pubid=5890\u0026purl=http%3A%2F%2Febay.com%2Finte%2Fautomation%2Fs2s_activation%2Fbanner-with-gdpr-pubmatic-denied-defaultbidder.html%3Fprofileid%3D22503%26pwtv%3D1%26pwtvc%3D1%26appnexus_banner_fixedbid%3D14%26fixedbid%3D1%26debug%3D1\u0026sl=1\u0026slot=%2F43743431%2FDMDemo\u0026ss=1\u0026tst=1704357774\"\u003e\u003c/div\u003e\u003cdiv style=\"position:absolute;left:0px;top:0px;visibility:hidden;\"\u003e\u003cimg src=\"https:?adv=\u0026af=banner\u0026aps=0\u0026au=\u0026bc=pubmatic\u0026bidid=bb57a9e3-fdc2-4772-8071-112dd7f50a6a\u0026di=-1\u0026eg=0\u0026en=0\u0026ft=0\u0026iid=\u0026kgpv=\u0026nwid=6\u0026orig=\u0026origbidid=bid-id-1\u0026pdvid=0\u0026pid=0\u0026plt=0\u0026pn=pubmatic\u0026psz=0x0\u0026pubid=5890\u0026purl=\u0026sl=1\u0026slot=\u0026ss=1\u0026tst=0\"\u003e\u003c/div\u003e","ext":{"prebid":{"meta":{"adaptercode":"pubmatic","advertiserId":4098,"agencyId":4098,"demandSource":"6","mediaType":"banner","networkId":6},"type":"banner","bidid":"bb57a9e3-fdc2-4772-8071-112dd7f50a6a"},"refreshInterval":30,"crtype":"banner","dspid":6,"netecpm":5,"origbidcpm":8,"origbidcur":"USD","ibv":true,"clicktrackers":["http://clicktracker1.com","http://clicktracker2.com"],"act":1}}],"seat":"pubmatic"}],"ext":{"responsetimemillis":{"pubmatic":8},"matchedimpression":{"pubmatic":0}}}`),
			},
		},
		{
			name: "valid_bidResponse_with_banner_bids_instl1_isappopenad_present_with_0",
			args: args{
				ctx: nil,
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							PubID:    5890,
							PubIDStr: "5890",
							Platform: "web",
							ImpBidCtx: map[string]models.ImpCtx{
								"Div1": {
									Bidders: map[string]models.PartnerData{
										"pubmatic": {
											PartnerID:        123,
											PrebidBidderCode: "pubmatic",
										},
									},
									Video:       &openrtb2.Video{},
									Type:        "video",
									IsBanner:    true,
									Instl:       1,
									IsAppOpenAd: 0,
								},
							},
							BidderResponseTimeMillis: map[string]int{},
							SeatNonBids:              map[string][]openrtb_ext.NonBid{},
							ReturnAllBidStatus:       true,
							Debug:                    true,
							ClientConfigFlag:         1,
							PartnerConfigMap: map[int]map[string]string{
								123: {
									models.PARTNER_ID:          "123",
									models.PREBID_PARTNER_NAME: "pubmatic",
									models.BidderCode:          "pubmatic",
									models.SERVER_SIDE_FLAG:    "1",
									models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
									models.TIMEOUT:             "200",
								},
								-1: {
									models.DisplayVersionID: "1",
									"refreshInterval":       "30",
									"rev_share":             "0.5",
								},
							},
						},
					},
				},
				payload: hookstage.AuctionResponsePayload{
					BidResponse: &openrtb2.BidResponse{
						ID: "12345",
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "Div1",
										Price: 5,
										AdM:   "<img src=\"http://ads.pubmatic.com/AdTag/728x90.png\"></img><div style=\"position:absolute;left:0px;top:0px;visibility:hidden;\"><img src=\"https://t.pubmatic.com/wt?adv=&af=banner&aps=0&au=%2F43743431%2FDMDemo&bc=appnexus&bidid=4033c510-6d67-4af6-b53f-682ff1a580c3&di=-1&eg=14&en=14&frv=1.57&ft=0&fv=1.57&iid=429d469d-8cfb-495a-9f0c-5f48aa0ede40&kgpv=&orig=ebay.com&origbidid=718825584&pdvid=1&pid=22503&plt=1&pn=appnexus&psz=728x90&pubid=5890&purl=http%3A%2F%2Febay.com%2Finte%2Fautomation%2Fs2s_activation%2Fbanner-with-gdpr-pubmatic-denied-defaultbidder.html%3Fprofileid%3D22503%26pwtv%3D1%26pwtvc%3D1%26appnexus_banner_fixedbid%3D14%26fixedbid%3D1%26debug%3D1&sl=1&slot=%2F43743431%2FDMDemo&ss=1&tst=1704357774\"></div>",
										Ext:   json.RawMessage(`{"bidtype":0,"deal_channel":1,"dspid":6,"origbidcpm":8,"origbidcur":"USD","ibv":true,"prebid":{"bidid":"bb57a9e3-fdc2-4772-8071-112dd7f50a6a","meta":{"adaptercode":"pubmatic","advertiserId":4098,"agencyId":4098,"demandSource":"6","mediaType":"banner","networkId":6},"targeting":{"hb_bidder_pubmatic":"pubmatic","hb_deal_pubmatic":"PUBDEAL1","hb_pb_pubmatic":"8.00","hb_size_pubmatic":"728x90"},"type":"banner","video":{"duration":0,"primary_category":"","vasttagid":""}},"clicktrackers":["http://clicktracker1.com","http://clicktracker2.com"]}`),
									},
								},
								Seat: "pubmatic",
							},
						},
						Ext: json.RawMessage(`{"responsetimemillis":{"pubmatic":8}}`),
					},
				},
			},
			setup: func() *mock_metrics.MockMetricsEngine {
				mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("web", "5890", "pubmatic")
				mockEngine.EXPECT().RecordPartnerResponseTimeStats("5890", "pubmatic", 8)
				mockEngine.EXPECT().RecordPublisherResponseTimeStats("5890", gomock.Any())
				mockEngine.EXPECT().RecordPublisherPartnerNoCookieStats("5890", gomock.Any()).AnyTimes()
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
				mockFeature.EXPECT().IsFscApplicable(gomock.Any(), gomock.Any(), gomock.Any()).Return(false)
				mockFeature.EXPECT().IsActApplicable(gomock.Any(), gomock.Any(), gomock.Any()).Return(true)
				mockEngine.EXPECT().RecordIBVRequest("5890", gomock.Any())
				return mockEngine
			},
			doMutate: true,
			want: want{
				result: hookstage.HookResult[hookstage.AuctionResponsePayload]{
					DebugMessages: []string{`[{"PubID":5890,"ProfileID":0,"DisplayID":0,"VersionID":0,"DisplayVersionID":0,"SSAuction":0,"SummaryDisable":0,"SSAI":"","PartnerConfigMap":{"-1":{"displayVersionId":"1","refreshInterval":"30","rev_share":"0.5"},"123":{"bidderCode":"pubmatic","kgp":"_AU_@_W_x_H_","partnerId":"123","prebidPartnerName":"pubmatic","serverSideEnabled":"1","timeout":"200"}},"SupportDeals":false,"Platform":"web","LoggerImpressionID":"","ClientConfigFlag":1,"IP":"","TMax":0,"IsTestRequest":0,"ABTestConfig":0,"ABTestConfigApplied":0,"IsCTVRequest":false,"TrackerEndpoint":"","VideoErrorTrackerEndpoint":"","UA":"","Cookies":"","UidCookie":null,"KADUSERCookie":null,"ParsedUidCookie":null,"OriginCookie":"","Debug":true,"Trace":false,"PageURL":"","StartTime":0,"DevicePlatform":0,"Trackers":{"bid-id-1":{"Tracker":{"PubID":5890,"PageURL":"","Timestamp":0,"IID":"","ProfileID":"0","VersionID":"0","SlotID":"","Adunit":"","PartnerInfo":{"PartnerID":"pubmatic","BidderCode":"pubmatic","KGPV":"","GrossECPM":0,"NetECPM":0,"BidID":"bb57a9e3-fdc2-4772-8071-112dd7f50a6a","OrigBidID":"bid-id-1","AdSize":"0x0","AdDuration":0,"Adformat":"banner","ServerSide":1,"Advertiser":"","FloorValue":0,"FloorRuleValue":0,"DealID":"-1"},"RewardedInventory":0,"SURL":"","Platform":0,"SSAI":"","AdPodSlot":0,"TestGroup":0,"Origin":"","FloorSkippedFlag":null,"FloorModelVersion":"","FloorSource":null,"FloorType":0,"CustomDimensions":"","LoggerData":{"KGPSV":"","FloorProvider":"","FloorFetchStatus":null}},"TrackerURL":"https:?adv=\u0026af=banner\u0026aps=0\u0026au=\u0026bc=pubmatic\u0026bidid=bb57a9e3-fdc2-4772-8071-112dd7f50a6a\u0026di=-1\u0026eg=0\u0026en=0\u0026ft=0\u0026iid=\u0026kgpv=\u0026nwid=6\u0026orig=\u0026origbidid=bid-id-1\u0026pdvid=0\u0026pid=0\u0026plt=0\u0026pn=pubmatic\u0026psz=0x0\u0026pubid=5890\u0026purl=\u0026sl=1\u0026slot=\u0026ss=1\u0026tst=0","ErrorURL":"","Price":5,"PriceModel":"CPM","PriceCurrency":""}},"PrebidBidderCode":null,"ImpBidCtx":{"Div1":{"ImpID":"","TagID":"","Div":"","SlotName":"","AdUnitName":"","Secure":0,"BidFloor":0,"BidFloorCur":"","IsRewardInventory":null,"Banner":true,"Video":{"mimes":null},"Native":null,"IncomingSlots":null,"Type":"video","Bidders":{"pubmatic":{"PartnerID":123,"PrebidBidderCode":"pubmatic","MatchedSlot":"","KGP":"","KGPV":"","IsRegex":false,"Params":null,"VASTTagFlag":false,"VASTTagFlags":null}},"NonMapped":null,"NewExt":null,"BidCtx":{"bid-id-1":{"prebid":{"meta":{"adaptercode":"pubmatic","advertiserId":4098,"agencyId":4098,"demandSource":"6","mediaType":"banner","networkId":6},"type":"banner","bidid":"bb57a9e3-fdc2-4772-8071-112dd7f50a6a"},"refreshInterval":30,"crtype":"banner","dspid":6,"netecpm":5,"origbidcpm":8,"origbidcur":"USD","EG":0,"EN":0}},"BannerAdUnitCtx":{"MatchedSlot":"","IsRegex":false,"MatchedRegex":"","SelectedSlotAdUnitConfig":null,"AppliedSlotAdUnitConfig":null,"UsingDefaultConfig":false,"AllowedConnectionTypes":null},"VideoAdUnitCtx":{"MatchedSlot":"","IsRegex":false,"MatchedRegex":"","SelectedSlotAdUnitConfig":null,"AppliedSlotAdUnitConfig":null,"UsingDefaultConfig":false,"AllowedConnectionTypes":null},"BidderError":"","IsAdPodRequest":false}},"Aliases":null,"NewReqExt":null,"ResponseExt":{"responsetimemillis":{"pubmatic":8}},"MarketPlaceBidders":null,"AdapterThrottleMap":null,"AdUnitConfig":null,"Source":"","Origin":"","SendAllBids":false,"WinningBids":{"Div1":{"ID":"bid-id-1","NetEcpm":5,"BidDealTierSatisfied":false,"Nbr":null}},"DroppedBids":null,"DefaultBids":{},"SeatNonBids":{},"BidderResponseTimeMillis":{"pubmatic":8},"Endpoint":"","PubIDStr":"5890","ProfileIDStr":"","MetricsEngine":{},"ReturnAllBidStatus":true,"Sshb":"","DCName":"","CachePutMiss":0,"MatchedImpression":{"pubmatic":0},"CustomDimensions":null}]`},
				},
				err:         nil,
				bidResponse: json.RawMessage(`{"id":"12345","seatbid":[{"bid":[{"id":"bid-id-1","impid":"Div1","price":5,"adm":"\u003cimg src=\"http://ads.pubmatic.com/AdTag/728x90.png\"\u003e\u003c/img\u003e\u003cdiv style=\"position:absolute;left:0px;top:0px;visibility:hidden;\"\u003e\u003cimg src=\"https://t.pubmatic.com/wt?adv=\u0026af=banner\u0026aps=0\u0026au=%2F43743431%2FDMDemo\u0026bc=appnexus\u0026bidid=4033c510-6d67-4af6-b53f-682ff1a580c3\u0026di=-1\u0026eg=14\u0026en=14\u0026frv=1.57\u0026ft=0\u0026fv=1.57\u0026iid=429d469d-8cfb-495a-9f0c-5f48aa0ede40\u0026kgpv=\u0026orig=ebay.com\u0026origbidid=718825584\u0026pdvid=1\u0026pid=22503\u0026plt=1\u0026pn=appnexus\u0026psz=728x90\u0026pubid=5890\u0026purl=http%3A%2F%2Febay.com%2Finte%2Fautomation%2Fs2s_activation%2Fbanner-with-gdpr-pubmatic-denied-defaultbidder.html%3Fprofileid%3D22503%26pwtv%3D1%26pwtvc%3D1%26appnexus_banner_fixedbid%3D14%26fixedbid%3D1%26debug%3D1\u0026sl=1\u0026slot=%2F43743431%2FDMDemo\u0026ss=1\u0026tst=1704357774\"\u003e\u003c/div\u003e\u003cdiv style=\"position:absolute;left:0px;top:0px;visibility:hidden;\"\u003e\u003cimg src=\"https:?adv=\u0026af=banner\u0026aps=0\u0026au=\u0026bc=pubmatic\u0026bidid=bb57a9e3-fdc2-4772-8071-112dd7f50a6a\u0026di=-1\u0026eg=0\u0026en=0\u0026ft=0\u0026iid=\u0026kgpv=\u0026nwid=6\u0026orig=\u0026origbidid=bid-id-1\u0026pdvid=0\u0026pid=0\u0026plt=0\u0026pn=pubmatic\u0026psz=0x0\u0026pubid=5890\u0026purl=\u0026sl=1\u0026slot=\u0026ss=1\u0026tst=0\"\u003e\u003c/div\u003e","ext":{"prebid":{"meta":{"adaptercode":"pubmatic","advertiserId":4098,"agencyId":4098,"demandSource":"6","mediaType":"banner","networkId":6},"type":"banner","bidid":"bb57a9e3-fdc2-4772-8071-112dd7f50a6a"},"refreshInterval":30,"crtype":"banner","dspid":6,"netecpm":5,"origbidcpm":8,"origbidcur":"USD","ibv":true,"clicktrackers":["http://clicktracker1.com","http://clicktracker2.com"],"act":1}}],"seat":"pubmatic"}],"ext":{"responsetimemillis":{"pubmatic":8},"matchedimpression":{"pubmatic":0}}}`),
			},
		},
		{
			name: "valid_bidResponse_with_banner_bids_instl1_isappopenad_present_with_1",
			args: args{
				ctx: nil,
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							PubID:    5890,
							PubIDStr: "5890",
							Platform: "web",
							ImpBidCtx: map[string]models.ImpCtx{
								"Div1": {
									Bidders: map[string]models.PartnerData{
										"pubmatic": {
											PartnerID:        123,
											PrebidBidderCode: "pubmatic",
										},
									},
									Video:       &openrtb2.Video{},
									Type:        "video",
									IsBanner:    true,
									Instl:       1,
									IsAppOpenAd: 1,
								},
							},
							BidderResponseTimeMillis: map[string]int{},
							SeatNonBids:              map[string][]openrtb_ext.NonBid{},
							ReturnAllBidStatus:       true,
							Debug:                    true,
							ClientConfigFlag:         1,
							PartnerConfigMap: map[int]map[string]string{
								123: {
									models.PARTNER_ID:          "123",
									models.PREBID_PARTNER_NAME: "pubmatic",
									models.BidderCode:          "pubmatic",
									models.SERVER_SIDE_FLAG:    "1",
									models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
									models.TIMEOUT:             "200",
								},
								-1: {
									models.DisplayVersionID: "1",
									"refreshInterval":       "30",
									"rev_share":             "0.5",
								},
							},
						},
					},
				},
				payload: hookstage.AuctionResponsePayload{
					BidResponse: &openrtb2.BidResponse{
						ID: "12345",
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "Div1",
										Price: 5,
										AdM:   "<img src=\"http://ads.pubmatic.com/AdTag/728x90.png\"></img><div style=\"position:absolute;left:0px;top:0px;visibility:hidden;\"><img src=\"https://t.pubmatic.com/wt?adv=&af=banner&aps=0&au=%2F43743431%2FDMDemo&bc=appnexus&bidid=4033c510-6d67-4af6-b53f-682ff1a580c3&di=-1&eg=14&en=14&frv=1.57&ft=0&fv=1.57&iid=429d469d-8cfb-495a-9f0c-5f48aa0ede40&kgpv=&orig=ebay.com&origbidid=718825584&pdvid=1&pid=22503&plt=1&pn=appnexus&psz=728x90&pubid=5890&purl=http%3A%2F%2Febay.com%2Finte%2Fautomation%2Fs2s_activation%2Fbanner-with-gdpr-pubmatic-denied-defaultbidder.html%3Fprofileid%3D22503%26pwtv%3D1%26pwtvc%3D1%26appnexus_banner_fixedbid%3D14%26fixedbid%3D1%26debug%3D1&sl=1&slot=%2F43743431%2FDMDemo&ss=1&tst=1704357774\"></div>",
										Ext:   json.RawMessage(`{"bidtype":0,"deal_channel":1,"dspid":6,"origbidcpm":8,"origbidcur":"USD","ibv":true,"prebid":{"bidid":"bb57a9e3-fdc2-4772-8071-112dd7f50a6a","meta":{"adaptercode":"pubmatic","advertiserId":4098,"agencyId":4098,"demandSource":"6","mediaType":"banner","networkId":6},"targeting":{"hb_bidder_pubmatic":"pubmatic","hb_deal_pubmatic":"PUBDEAL1","hb_pb_pubmatic":"8.00","hb_size_pubmatic":"728x90"},"type":"banner","video":{"duration":0,"primary_category":"","vasttagid":""}},"clicktrackers":["http://clicktracker1.com","http://clicktracker2.com"]}`),
									},
								},
								Seat: "pubmatic",
							},
						},
						Ext: json.RawMessage(`{"responsetimemillis":{"pubmatic":8}}`),
					},
				},
			},
			setup: func() *mock_metrics.MockMetricsEngine {
				mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("web", "5890", "pubmatic")
				mockEngine.EXPECT().RecordPartnerResponseTimeStats("5890", "pubmatic", 8)
				mockEngine.EXPECT().RecordPublisherResponseTimeStats("5890", gomock.Any())
				mockEngine.EXPECT().RecordPublisherPartnerNoCookieStats("5890", gomock.Any()).AnyTimes()
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
				mockFeature.EXPECT().IsFscApplicable(gomock.Any(), gomock.Any(), gomock.Any()).Return(false)
				mockEngine.EXPECT().RecordIBVRequest("5890", gomock.Any())
				return mockEngine
			},
			doMutate: true,
			want: want{
				result: hookstage.HookResult[hookstage.AuctionResponsePayload]{
					DebugMessages: []string{`[{"PubID":5890,"ProfileID":0,"DisplayID":0,"VersionID":0,"DisplayVersionID":0,"SSAuction":0,"SummaryDisable":0,"SSAI":"","PartnerConfigMap":{"-1":{"displayVersionId":"1","refreshInterval":"30","rev_share":"0.5"},"123":{"bidderCode":"pubmatic","kgp":"_AU_@_W_x_H_","partnerId":"123","prebidPartnerName":"pubmatic","serverSideEnabled":"1","timeout":"200"}},"SupportDeals":false,"Platform":"web","LoggerImpressionID":"","ClientConfigFlag":1,"IP":"","TMax":0,"IsTestRequest":0,"ABTestConfig":0,"ABTestConfigApplied":0,"IsCTVRequest":false,"TrackerEndpoint":"","VideoErrorTrackerEndpoint":"","UA":"","Cookies":"","UidCookie":null,"KADUSERCookie":null,"ParsedUidCookie":null,"OriginCookie":"","Debug":true,"Trace":false,"PageURL":"","StartTime":0,"DevicePlatform":0,"Trackers":{"bid-id-1":{"Tracker":{"PubID":5890,"PageURL":"","Timestamp":0,"IID":"","ProfileID":"0","VersionID":"0","SlotID":"","Adunit":"","PartnerInfo":{"PartnerID":"pubmatic","BidderCode":"pubmatic","KGPV":"","GrossECPM":0,"NetECPM":0,"BidID":"bb57a9e3-fdc2-4772-8071-112dd7f50a6a","OrigBidID":"bid-id-1","AdSize":"0x0","AdDuration":0,"Adformat":"banner","ServerSide":1,"Advertiser":"","FloorValue":0,"FloorRuleValue":0,"DealID":"-1"},"RewardedInventory":0,"SURL":"","Platform":0,"SSAI":"","AdPodSlot":0,"TestGroup":0,"Origin":"","FloorSkippedFlag":null,"FloorModelVersion":"","FloorSource":null,"FloorType":0,"CustomDimensions":"","LoggerData":{"KGPSV":"","FloorProvider":"","FloorFetchStatus":null}},"TrackerURL":"https:?adv=\u0026af=banner\u0026aps=0\u0026au=\u0026bc=pubmatic\u0026bidid=bb57a9e3-fdc2-4772-8071-112dd7f50a6a\u0026di=-1\u0026eg=0\u0026en=0\u0026ft=0\u0026iid=\u0026kgpv=\u0026nwid=6\u0026orig=\u0026origbidid=bid-id-1\u0026pdvid=0\u0026pid=0\u0026plt=0\u0026pn=pubmatic\u0026psz=0x0\u0026pubid=5890\u0026purl=\u0026sl=1\u0026slot=\u0026ss=1\u0026tst=0","ErrorURL":"","Price":5,"PriceModel":"CPM","PriceCurrency":""}},"PrebidBidderCode":null,"ImpBidCtx":{"Div1":{"ImpID":"","TagID":"","Div":"","SlotName":"","AdUnitName":"","Secure":0,"BidFloor":0,"BidFloorCur":"","IsRewardInventory":null,"Banner":true,"Video":{"mimes":null},"Native":null,"IncomingSlots":null,"Type":"video","Bidders":{"pubmatic":{"PartnerID":123,"PrebidBidderCode":"pubmatic","MatchedSlot":"","KGP":"","KGPV":"","IsRegex":false,"Params":null,"VASTTagFlag":false,"VASTTagFlags":null}},"NonMapped":null,"NewExt":null,"BidCtx":{"bid-id-1":{"prebid":{"meta":{"adaptercode":"pubmatic","advertiserId":4098,"agencyId":4098,"demandSource":"6","mediaType":"banner","networkId":6},"type":"banner","bidid":"bb57a9e3-fdc2-4772-8071-112dd7f50a6a"},"refreshInterval":30,"crtype":"banner","dspid":6,"netecpm":5,"origbidcpm":8,"origbidcur":"USD","EG":0,"EN":0}},"BannerAdUnitCtx":{"MatchedSlot":"","IsRegex":false,"MatchedRegex":"","SelectedSlotAdUnitConfig":null,"AppliedSlotAdUnitConfig":null,"UsingDefaultConfig":false,"AllowedConnectionTypes":null},"VideoAdUnitCtx":{"MatchedSlot":"","IsRegex":false,"MatchedRegex":"","SelectedSlotAdUnitConfig":null,"AppliedSlotAdUnitConfig":null,"UsingDefaultConfig":false,"AllowedConnectionTypes":null},"BidderError":"","IsAdPodRequest":false}},"Aliases":null,"NewReqExt":null,"ResponseExt":{"responsetimemillis":{"pubmatic":8}},"MarketPlaceBidders":null,"AdapterThrottleMap":null,"AdUnitConfig":null,"Source":"","Origin":"","SendAllBids":false,"WinningBids":{"Div1":{"ID":"bid-id-1","NetEcpm":5,"BidDealTierSatisfied":false,"Nbr":null}},"DroppedBids":null,"DefaultBids":{},"SeatNonBids":{},"BidderResponseTimeMillis":{"pubmatic":8},"Endpoint":"","PubIDStr":"5890","ProfileIDStr":"","MetricsEngine":{},"ReturnAllBidStatus":true,"Sshb":"","DCName":"","CachePutMiss":0,"MatchedImpression":{"pubmatic":0},"CustomDimensions":null}]`},
				},
				err:         nil,
				bidResponse: json.RawMessage(`{"id":"12345","seatbid":[{"bid":[{"id":"bid-id-1","impid":"Div1","price":5,"adm":"\u003cimg src=\"http://ads.pubmatic.com/AdTag/728x90.png\"\u003e\u003c/img\u003e\u003cdiv style=\"position:absolute;left:0px;top:0px;visibility:hidden;\"\u003e\u003cimg src=\"https://t.pubmatic.com/wt?adv=\u0026af=banner\u0026aps=0\u0026au=%2F43743431%2FDMDemo\u0026bc=appnexus\u0026bidid=4033c510-6d67-4af6-b53f-682ff1a580c3\u0026di=-1\u0026eg=14\u0026en=14\u0026frv=1.57\u0026ft=0\u0026fv=1.57\u0026iid=429d469d-8cfb-495a-9f0c-5f48aa0ede40\u0026kgpv=\u0026orig=ebay.com\u0026origbidid=718825584\u0026pdvid=1\u0026pid=22503\u0026plt=1\u0026pn=appnexus\u0026psz=728x90\u0026pubid=5890\u0026purl=http%3A%2F%2Febay.com%2Finte%2Fautomation%2Fs2s_activation%2Fbanner-with-gdpr-pubmatic-denied-defaultbidder.html%3Fprofileid%3D22503%26pwtv%3D1%26pwtvc%3D1%26appnexus_banner_fixedbid%3D14%26fixedbid%3D1%26debug%3D1\u0026sl=1\u0026slot=%2F43743431%2FDMDemo\u0026ss=1\u0026tst=1704357774\"\u003e\u003c/div\u003e\u003cdiv style=\"position:absolute;left:0px;top:0px;visibility:hidden;\"\u003e\u003cimg src=\"https:?adv=\u0026af=banner\u0026aps=0\u0026au=\u0026bc=pubmatic\u0026bidid=bb57a9e3-fdc2-4772-8071-112dd7f50a6a\u0026di=-1\u0026eg=0\u0026en=0\u0026ft=0\u0026iid=\u0026kgpv=\u0026nwid=6\u0026orig=\u0026origbidid=bid-id-1\u0026pdvid=0\u0026pid=0\u0026plt=0\u0026pn=pubmatic\u0026psz=0x0\u0026pubid=5890\u0026purl=\u0026sl=1\u0026slot=\u0026ss=1\u0026tst=0\"\u003e\u003c/div\u003e","ext":{"prebid":{"meta":{"adaptercode":"pubmatic","advertiserId":4098,"agencyId":4098,"demandSource":"6","mediaType":"banner","networkId":6},"type":"banner","bidid":"bb57a9e3-fdc2-4772-8071-112dd7f50a6a"},"refreshInterval":30,"crtype":"banner","dspid":6,"netecpm":5,"origbidcpm":8,"origbidcur":"USD","ibv":true,"clicktrackers":["http://clicktracker1.com","http://clicktracker2.com"]}}],"seat":"pubmatic"}],"ext":{"responsetimemillis":{"pubmatic":8},"matchedimpression":{"pubmatic":0}}}`),
			},
		},
		{
			name: "valid_bidResponse_with_video_bids_instl_1_isAppOpenAd_0",
			args: args{
				ctx: nil,
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							PubID:    5890,
							PubIDStr: "5890",
							Platform: "web",
							NewReqExt: &models.RequestExt{
								ExtRequest: openrtb_ext.ExtRequest{
									Prebid: openrtb_ext.ExtRequestPrebid{ExtOWRequestPrebid: openrtb_ext.ExtOWRequestPrebid{GoogleSSUFeatureEnabled: true}},
								},
							},
							ImpBidCtx: map[string]models.ImpCtx{
								"Div1": {
									Bidders: map[string]models.PartnerData{
										"pubmatic": {
											PartnerID:        123,
											PrebidBidderCode: "pubmatic",
										},
									},
									Video: &openrtb2.Video{
										MaxDuration:    20,
										MinDuration:    10,
										SkipAfter:      2,
										Skip:           ptrutil.ToPtr[int8](1),
										SkipMin:        1,
										BAttr:          []adcom1.CreativeAttribute{adcom1.CreativeAttribute(1)},
										PlaybackMethod: []adcom1.PlaybackMethod{adcom1.PlaybackPageLoadSoundOn},
									},
									Type:        "video",
									IsBanner:    false,
									Instl:       1,
									IsAppOpenAd: 0,
								},
							},
							PrebidBidderCode: map[string]string{
								"pubmatic": "pubmatic",
							},
							BidderResponseTimeMillis: map[string]int{},
							SeatNonBids:              map[string][]openrtb_ext.NonBid{},
							ReturnAllBidStatus:       true,
							Debug:                    true,
							PartnerConfigMap: map[int]map[string]string{
								123: {
									models.PARTNER_ID:          "123",
									models.PREBID_PARTNER_NAME: "pubmatic",
									models.BidderCode:          "pubmatic",
									models.SERVER_SIDE_FLAG:    "1",
									models.KEY_GEN_PATTERN:     "_AU_@_W_x_H_",
									models.TIMEOUT:             "200",
								},
								-1: {
									models.DisplayVersionID: "1",
									"refreshInterval":       "30",
									"rev_share":             "0.5",
								},
							},
						},
					},
				},
				payload: hookstage.AuctionResponsePayload{
					BidResponse: &openrtb2.BidResponse{
						ID: "12345",
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "Div1",
										Price: 5,
										AdM:   "<VAST version=\"3.0\"><Ad><Wrapper></Wrapper></Ad></VAST>",
										Ext:   json.RawMessage(`{"bidtype":0,"deal_channel":1,"dspid":6,"mbmfv":4,"origbidcpm":8,"origbidcur":"USD","prebid":{"bidid":"bb57a9e3-fdc2-4772-8071-112dd7f50a6a","meta":{"adaptercode":"pubmatic","advertiserId":4098,"agencyId":4098,"demandSource":"6","mediaType":"banner","networkId":6},"targeting":{"hb_bidder_pubmatic":"pubmatic","hb_deal_pubmatic":"PUBDEAL1","hb_pb_pubmatic":"8.00","hb_size_pubmatic":"728x90"},"type":"video","video":{"duration":0,"primary_category":"","vasttagid":""}},"clicktrackers":["http://clicktracker1.com","http://clicktracker2.com"]}`),
									},
								},
								Seat: "pubmatic",
							},
						},
						Ext: json.RawMessage(`{"responsetimemillis":{"pubmatic":8}}`),
					},
				},
			},
			setup: func() *mock_metrics.MockMetricsEngine {
				mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)
				mockEngine.EXPECT().RecordPlatformPublisherPartnerResponseStats("web", "5890", "pubmatic")
				mockEngine.EXPECT().RecordPartnerResponseTimeStats("5890", "pubmatic", 8)
				mockEngine.EXPECT().RecordPublisherResponseTimeStats("5890", gomock.Any())
				mockEngine.EXPECT().RecordPublisherPartnerNoCookieStats("5890", gomock.Any()).AnyTimes()
				mockFeature.EXPECT().IsTBFFeatureEnabled(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
				mockFeature.EXPECT().IsFscApplicable(gomock.Any(), gomock.Any(), gomock.Any()).Return(false).AnyTimes()
				mockFeature.EXPECT().IsActApplicable(gomock.Any(), gomock.Any(), gomock.Any()).Return(true)
				return mockEngine
			},
			doMutate: true,
			want: want{
				result:      hookstage.HookResult[hookstage.AuctionResponsePayload]{},
				err:         nil,
				bidResponse: json.RawMessage(`{"id":"12345","seatbid":[{"bid":[{"id":"bid-id-1","impid":"Div1","price":5,"adm":"\u003cVAST version=\"3.0\"\u003e\u003cAd\u003e\u003cWrapper\u003e\u003cImpression\u003e\u003c![CDATA[https:?adv=\u0026af=video\u0026aps=0\u0026au=\u0026bc=pubmatic\u0026bidid=bb57a9e3-fdc2-4772-8071-112dd7f50a6a\u0026di=-1\u0026dur=20\u0026eg=0\u0026en=0\u0026frv=4\u0026ft=0\u0026fv=4\u0026iid=\u0026kgpv=\u0026mbmf=1\u0026nwid=6\u0026orig=\u0026origbidid=bid-id-1\u0026pdvid=0\u0026pid=0\u0026plt=0\u0026pn=pubmatic\u0026psz=0x0\u0026pubid=5890\u0026purl=\u0026sl=1\u0026slot=\u0026ss=1\u0026tst=0]]\u003e\u003c/Impression\u003e\u003cExtensions\u003e\u003cExtension\u003e\u003cPricing model=\"CPM\" currency=\"USD\"\u003e\u003c![CDATA[5]]\u003e\u003c/Pricing\u003e\u003c/Extension\u003e\u003c/Extensions\u003e\u003c/Wrapper\u003e\u003c/Ad\u003e\u003c/VAST\u003e","ext":{"prebid":{"meta":{"adaptercode":"pubmatic","advertiserId":4098,"agencyId":4098,"demandSource":"6","mediaType":"banner","networkId":6},"type":"video","video":{"duration":20,"primary_category":"","vasttagid":""},"bidid":"bb57a9e3-fdc2-4772-8071-112dd7f50a6a"},"refreshInterval":30,"crtype":"video","video":{"minduration":10,"maxduration":20,"skip":1,"skipmin":1,"skipafter":2,"battr":[1],"playbackmethod":[1]},"dspid":6,"netecpm":5,"origbidcpm":8,"origbidcur":"USD","act":1}}],"seat":"pubmatic"}],"ext":{"responsetimemillis":{"pubmatic":8},"matchedimpression":{"pubmatic":0}}}`),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mockEngine *mock_metrics.MockMetricsEngine
			if tt.setup != nil {
				mockEngine = tt.setup()
			}
			m := OpenWrap{
				cache:        mockCache,
				metricEngine: mockEngine,
				pubFeatures:  mockFeature,
			}
			moduleCtx, ok := tt.args.moduleCtx.ModuleContext["rctx"]
			if ok {
				rCtx, ok := moduleCtx.(models.RequestCtx)
				if ok {
					rCtx.MetricsEngine = mockEngine
					tt.args.moduleCtx.ModuleContext["rctx"] = rCtx
				}
			}
			hookResult, err := m.handleAuctionResponseHook(tt.args.ctx, tt.args.moduleCtx, tt.args.payload)
			assert.Equal(t, tt.want.err, err, tt.name)
			if tt.doMutate {
				mutations := hookResult.ChangeSet.Mutations()
				assert.NotEmpty(t, mutations, tt.name)
				for _, mut := range mutations {
					result, err := mut.Apply(tt.args.payload)
					gotBidResponse, _ := json.Marshal(result.BidResponse)
					assert.Nil(t, err, tt.name)
					assert.Equal(t, string(tt.want.bidResponse), string(gotBidResponse), tt.name)
				}
				return
			}
			assert.Equal(t, tt.want.result.DebugMessages, hookResult.DebugMessages, tt.name)
		})
	}
}

func TestHandleAuctionResponseHook_APSIntegration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockFeature := mock_feature.NewMockFeature(ctrl)
	mockCache := mock_cache.NewMockCache(ctrl)
	mockMetrics := mock_metrics.NewMockMetricsEngine(ctrl)

	module := OpenWrap{
		pubFeatures:  mockFeature,
		cache:        mockCache,
		metricEngine: mockMetrics,
	}

	type args struct {
		ctx       context.Context
		moduleCtx hookstage.ModuleInvocationContext
		payload   hookstage.AuctionResponsePayload
	}
	tests := []struct {
		name             string
		args             args
		wantAPSReject    bool
		wantDebugMessage bool
	}{
		{
			name: "APS endpoint with valid bids should not set reject",
			args: args{
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							StartTime: time.Now().UnixMilli(),
							ImpBidCtx: map[string]models.ImpCtx{
								"imp1": {},
							},
							PubIDStr: "5890",
							Endpoint: models.EndpointAPS,
							APS:      models.APS{Reject: false},
						},
					},
				},
				payload: hookstage.AuctionResponsePayload{
					BidResponse: &openrtb2.BidResponse{
						ID: "test-response",
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "pubmatic",
								Bid: []openrtb2.Bid{
									{
										ID:    "bid1",
										ImpID: "imp1",
										Price: 1.5,
									},
								},
							},
						},
					},
				},
			},
			wantAPSReject:    false,
			wantDebugMessage: false,
		},
		{
			name: "APS endpoint with NBR should set reject",
			args: args{
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							StartTime: time.Now().UnixMilli(),
							ImpBidCtx: map[string]models.ImpCtx{
								"imp1": {},
							},
							PubIDStr: "5890",
							Endpoint: models.EndpointAPS,
							APS:      models.APS{Reject: false},
						},
					},
				},
				payload: hookstage.AuctionResponsePayload{
					BidResponse: &openrtb2.BidResponse{
						ID:  "test-response",
						NBR: ptrutil.ToPtr(openrtb3.NoBidUnknownError),
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "pubmatic",
								Bid:  []openrtb2.Bid{},
							},
						},
					},
				},
			},
			wantAPSReject:    true,
			wantDebugMessage: false,
		},
		{
			name: "APS endpoint with empty bids should set reject",
			args: args{
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							StartTime: time.Now().UnixMilli(),
							ImpBidCtx: map[string]models.ImpCtx{
								"imp1": {},
							},
							PubIDStr: "5890",
							Endpoint: models.EndpointAPS,
							APS:      models.APS{Reject: false},
						},
					},
				},
				payload: hookstage.AuctionResponsePayload{
					BidResponse: &openrtb2.BidResponse{
						ID: "test-response",
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "pubmatic",
								Bid:  []openrtb2.Bid{},
							},
						},
					},
				},
			},
			wantAPSReject:    true,
			wantDebugMessage: false,
		},
		{
			name: "APS endpoint with no seat bids should set reject",
			args: args{
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							StartTime: time.Now().UnixMilli(),
							ImpBidCtx: map[string]models.ImpCtx{
								"imp1": {},
							},
							PubIDStr: "5890",
							Endpoint: models.EndpointAPS,
							APS:      models.APS{Reject: false},
						},
					},
				},
				payload: hookstage.AuctionResponsePayload{
					BidResponse: &openrtb2.BidResponse{
						ID:      "test-response",
						SeatBid: []openrtb2.SeatBid{},
					},
				},
			},
			wantAPSReject:    true,
			wantDebugMessage: false,
		},
		{
			name: "Non-APS endpoint should not be affected by APS logic",
			args: args{
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							StartTime: time.Now().UnixMilli(),
							ImpBidCtx: map[string]models.ImpCtx{
								"imp1": {},
							},
							PubIDStr: "5890",
							Endpoint: models.EndpointWebS2S,
							APS:      models.APS{Reject: false},
						},
					},
				},
				payload: hookstage.AuctionResponsePayload{
					BidResponse: &openrtb2.BidResponse{
						ID: "test-response",
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "pubmatic",
								Bid: []openrtb2.Bid{
									{
										ID:    "bid1",
										ImpID: "imp1",
										Price: 1.5,
									},
								},
							},
						},
					},
				},
			},
			wantAPSReject:    false,
			wantDebugMessage: false,
		},
		{
			name: "APS endpoint with debug enabled should include debug message",
			args: args{
				moduleCtx: hookstage.ModuleInvocationContext{
					ModuleContext: hookstage.ModuleContext{
						"rctx": models.RequestCtx{
							StartTime: time.Now().UnixMilli(),
							ImpBidCtx: map[string]models.ImpCtx{
								"imp1": {},
							},
							PubIDStr: "5890",
							Endpoint: models.EndpointAPS,
							APS:      models.APS{Reject: false},
							Debug:    true,
						},
					},
				},
				payload: hookstage.AuctionResponsePayload{
					BidResponse: &openrtb2.BidResponse{
						ID: "test-response",
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "pubmatic",
								Bid: []openrtb2.Bid{
									{
										ID:    "bid1",
										ImpID: "imp1",
										Price: 1.5,
									},
								},
							},
						},
					},
				},
			},
			wantAPSReject:    false,
			wantDebugMessage: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock expectations
			mockMetrics.EXPECT().RecordPlatformPublisherPartnerResponseStats(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			mockMetrics.EXPECT().RecordPublisherResponseTimeStats(gomock.Any(), gomock.Any()).AnyTimes()
			mockFeature.EXPECT().GetImpCountingMethodEnabledBidders().Return(map[string]struct{}{}).AnyTimes()
			mockFeature.EXPECT().GetEnabledPerformanceDSPs().Return(map[int]struct{}{}).AnyTimes()
			mockFeature.EXPECT().GetInViewEnabledPublishers().Return(map[int]struct{}{}).AnyTimes()
			mockFeature.EXPECT().IsFscApplicable(gomock.Any(), gomock.Any(), gomock.Any()).Return(false).AnyTimes()
			mockMetrics.EXPECT().RecordNobidErrPrebidServerResponse(gomock.Any()).AnyTimes()

			result, err := module.handleAuctionResponseHook(tt.args.ctx, tt.args.moduleCtx, tt.args.payload)

			assert.NoError(t, err, tt.name)
			assert.NotNil(t, result, tt.name)

			// Check that APS reject flag is set correctly in the updated rctx
			updatedRctx := tt.args.moduleCtx.ModuleContext["rctx"].(models.RequestCtx)
			assert.Equal(t, tt.wantAPSReject, updatedRctx.APS.Reject, tt.name)

			// Check debug message presence
			if tt.wantDebugMessage {
				assert.NotEmpty(t, result.DebugMessages, tt.name)
			} else {
				assert.Empty(t, result.DebugMessages, tt.name)
			}

			// Verify analytics tags contain request-ctx
			assert.NotNil(t, result.AnalyticsTags, tt.name)
			assert.Len(t, result.AnalyticsTags.Activities, 1, tt.name)
			assert.Equal(t, "openwrap_request_ctx", result.AnalyticsTags.Activities[0].Name, tt.name)
		})
	}
}
