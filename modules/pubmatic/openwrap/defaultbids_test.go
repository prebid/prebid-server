package openwrap

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/openrtb/v20/openrtb3"
	"github.com/prebid/prebid-server/v3/currency"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/exchange"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/config"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/geodb"
	metrics "github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/metrics"
	mock_metrics "github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/metrics/mock"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/profilemetadata"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/publisherfeature"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/unwrap"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/uuidutil"
	"github.com/stretchr/testify/assert"
)

const fakeUuid = "30470a14-2949-4110-abce-b62d57304ad5"

type TestUUIDGenerator struct{}

func (TestUUIDGenerator) Generate() (string, error) {
	return fakeUuid, nil
}

func TestGetNonBRCodeFromBidRespExt(t *testing.T) {
	type args struct {
		bidder         string
		bidResponseExt openrtb_ext.ExtBidResponse
	}
	tests := []struct {
		name string
		args args
		nbr  *openrtb3.NoBidReason
	}{
		{
			name: "bidResponseExt.Errors_is_empty",
			args: args{
				bidder: "pubmatic",
				bidResponseExt: openrtb_ext.ExtBidResponse{
					Errors: nil,
				},
			},
			nbr: openrtb3.NoBidUnknownError.Ptr(),
		},
		{
			name: "invalid_partner_err",
			args: args{
				bidder: "pubmatic",
				bidResponseExt: openrtb_ext.ExtBidResponse{
					Errors: map[openrtb_ext.BidderName][]openrtb_ext.ExtBidderMessage{
						"pubmatic": {
							{
								Code: 0,
							},
						},
					},
				},
			},
			nbr: exchange.ErrorGeneral.Ptr(),
		},
		{
			name: "unknown_partner_err",
			args: args{
				bidder: "pubmatic",
				bidResponseExt: openrtb_ext.ExtBidResponse{
					Errors: map[openrtb_ext.BidderName][]openrtb_ext.ExtBidderMessage{
						"pubmatic": {
							{
								Code: errortypes.UnknownErrorCode,
							},
						},
					},
				},
			},
			nbr: exchange.ErrorGeneral.Ptr(),
		},
		{
			name: "partner_timeout_err",
			args: args{
				bidder: "pubmatic",
				bidResponseExt: openrtb_ext.ExtBidResponse{
					Errors: map[openrtb_ext.BidderName][]openrtb_ext.ExtBidderMessage{
						"pubmatic": {
							{
								Code: errortypes.TimeoutErrorCode,
							},
						},
					},
				},
			},
			nbr: exchange.ErrorTimeout.Ptr(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nbr := getNonBRCodeFromBidRespExt(tt.args.bidder, tt.args.bidResponseExt)
			assert.Equal(t, tt.nbr, nbr, tt.name)
		})
	}
}

func TestOpenWrap_addDefaultBids(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)
	type fields struct {
		cfg             config.Config
		rateConvertor   *currency.RateConverter
		metricEngine    metrics.MetricsEngine
		geoInfoFetcher  geodb.Geography
		pubFeatures     publisherfeature.Feature
		unwrap          unwrap.Unwrap
		profileMetaData profilemetadata.ProfileMetaData
		uuidGenerator   uuidutil.UUIDGenerator
	}
	type args struct {
		rctx           *models.RequestCtx
		bidResponse    *openrtb2.BidResponse
		bidResponseExt openrtb_ext.ExtBidResponse
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		setup  func()
		want   map[string]map[string][]openrtb2.Bid
	}{
		{
			name: "EndpointWebS2S do not add default bids for slot-not-mapped and partner-throttled",
			fields: fields{
				metricEngine:  mockEngine,
				uuidGenerator: TestUUIDGenerator{},
			},
			args: args{
				rctx: &models.RequestCtx{
					Endpoint: models.EndpointWebS2S,
					ImpBidCtx: map[string]models.ImpCtx{
						"imp-1": {
							Bidders: map[string]models.PartnerData{
								"pubmatic": {
									PrebidBidderCode: "pubmatic",
								},
								"openx": {
									PrebidBidderCode: "openx",
								},
							},
							NonMapped: map[string]struct{}{
								"appnexus": {},
							},
							BidCtx: map[string]models.BidCtx{
								"pubmatic-bid-1": {
									BidExt: models.BidExt{},
								},
							},
						},
					},
					AdapterThrottleMap: map[string]struct{}{
						"rubicon": {},
					},
				},
				bidResponse: &openrtb2.BidResponse{
					ID: "bid-1",
					SeatBid: []openrtb2.SeatBid{
						{
							Seat: "pubmatic",
							Bid: []openrtb2.Bid{
								{
									ID:    "pubmatic-bid-1",
									ImpID: "imp-1",
									Price: 1.0,
								},
							},
						},
					},
				},
			},
			setup: func() {
				mockEngine.EXPECT().RecordPartnerResponseErrors(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			},
			want: map[string]map[string][]openrtb2.Bid{
				"imp-1": {
					"openx": {
						{
							ID:    "30470a14-2949-4110-abce-b62d57304ad5",
							ImpID: "imp-1",
							Ext:   []byte(`{}`),
						},
					},
				},
			},
		},
		{
			name: "one_bidder_in_SeatBid_two_in_DroppedBids_only_remaining_bidder_gets_default",
			fields: fields{
				metricEngine:  mockEngine,
				uuidGenerator: TestUUIDGenerator{},
			},
			args: args{
				rctx: &models.RequestCtx{
					Endpoint: models.EndpointWebS2S,
					ImpBidCtx: map[string]models.ImpCtx{
						"imp-1": {
							Bidders: map[string]models.PartnerData{
								"pubmatic": {
									PrebidBidderCode: "pubmatic",
								},
								"openx": {
									PrebidBidderCode: "openx",
								},
								"appnexus": {
									PrebidBidderCode: "appnexus",
								},
								"rubicon": {
									PrebidBidderCode: "rubicon",
								},
							},
							BidCtx: map[string]models.BidCtx{},
						},
					},
					DroppedBids: map[string][]openrtb2.Bid{
						"openx": {
							{
								ID:    "openx-dropped-1",
								ImpID: "imp-1",
								Price: 0.5,
							},
						},
						"appnexus": {
							{
								ID:    "appnexus-dropped-1",
								ImpID: "imp-1",
								Price: 0.4,
							},
						},
					},
				},
				bidResponse: &openrtb2.BidResponse{
					ID: "bid-1",
					SeatBid: []openrtb2.SeatBid{
						{
							Seat: "pubmatic",
							Bid: []openrtb2.Bid{
								{
									ID:    "pubmatic-winning-1",
									ImpID: "imp-1",
									Price: 2.0,
								},
							},
						},
					},
				},
				bidResponseExt: openrtb_ext.ExtBidResponse{},
			},
			setup: func() {
				mockEngine.EXPECT().RecordPartnerResponseErrors(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			},
			want: map[string]map[string][]openrtb2.Bid{
				"imp-1": {
					"rubicon": {
						{
							ID:    "30470a14-2949-4110-abce-b62d57304ad5",
							ImpID: "imp-1",
							Ext:   []byte(`{}`),
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			m := &OpenWrap{
				cfg:             tt.fields.cfg,
				metricEngine:    tt.fields.metricEngine,
				rateConvertor:   tt.fields.rateConvertor,
				geoInfoFetcher:  tt.fields.geoInfoFetcher,
				pubFeatures:     tt.fields.pubFeatures,
				unwrap:          tt.fields.unwrap,
				profileMetaData: tt.fields.profileMetaData,
				uuidGenerator:   tt.fields.uuidGenerator,
			}
			got := m.addDefaultBids(tt.args.rctx, tt.args.bidResponse, tt.args.bidResponseExt)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOpenWrap_addDefaultBidsForMultiFloorsConfig(t *testing.T) {
	type fields struct {
		cfg             config.Config
		metricEngine    metrics.MetricsEngine
		rateConvertor   *currency.RateConverter
		geoInfoFetcher  geodb.Geography
		pubFeatures     publisherfeature.Feature
		unwrap          unwrap.Unwrap
		profileMetaData profilemetadata.ProfileMetaData
		uuidGenerator   uuidutil.UUIDGenerator
	}
	type args struct {
		rctx           *models.RequestCtx
		bidResponse    *openrtb2.BidResponse
		bidResponseExt openrtb_ext.ExtBidResponse
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   map[string]map[string][]openrtb2.Bid
	}{
		{
			name: "request is other than applovinmax",
			args: args{
				rctx: &models.RequestCtx{
					Endpoint:    models.EndpointWebS2S,
					DefaultBids: map[string]map[string][]openrtb2.Bid{},
				},
				bidResponse: &openrtb2.BidResponse{
					ID:      "bid-1",
					SeatBid: []openrtb2.SeatBid{},
				},
			},
			want: map[string]map[string][]openrtb2.Bid{},
		},
		{
			name: "request is applovinmax but the multi-floors config is not enabled from DB",
			args: args{
				rctx: &models.RequestCtx{
					Endpoint:    models.EndpointAppLovinMax,
					DefaultBids: map[string]map[string][]openrtb2.Bid{},
					MultiFloors: nil,
				},
				bidResponse: &openrtb2.BidResponse{
					ID: "bid-1",
				},
			},
			want: map[string]map[string][]openrtb2.Bid{},
		},
		{
			name: "mulit-floors config have three floors and no bids in the response",
			args: args{
				rctx: &models.RequestCtx{
					Endpoint: models.EndpointAppLovinMax,
					DefaultBids: map[string]map[string][]openrtb2.Bid{
						"test-impID-1": {
							"pubmatic": {
								{
									ID:    "dbbsdhkldks1234",
									ImpID: "test-impID-1",
									Ext:   []byte(`{}`),
								},
							},
						},
					},
					MultiFloors: map[string]*models.MultiFloors{
						"test-impID-1": {Tier1: 1.1, Tier2: 2.1, Tier3: 3.1},
					},
					ImpBidCtx: map[string]models.ImpCtx{
						"test-impID-1": {
							TagID: "adunit-1",
							Bidders: map[string]models.PartnerData{
								"pubmatic": {
									PrebidBidderCode: "pubmatic",
								},
							},
							BidCtx: map[string]models.BidCtx{},
						},
					},
					PrebidBidderCode: map[string]string{
						"pubmatic": "pubmatic",
					},
				},
				bidResponse: &openrtb2.BidResponse{
					ID: "bid-1",
				},
			},
			fields: fields{
				uuidGenerator: TestUUIDGenerator{},
			},
			want: map[string]map[string][]openrtb2.Bid{
				"test-impID-1": {
					"pubmatic": {
						{
							ID:    "30470a14-2949-4110-abce-b62d57304ad5",
							ImpID: "test-impID-1",
						},
						{
							ID:    "30470a14-2949-4110-abce-b62d57304ad5",
							ImpID: "test-impID-1",
						},
						{
							ID:    "30470a14-2949-4110-abce-b62d57304ad5",
							ImpID: "test-impID-1",
						},
					},
				},
			},
		},
		{
			name: "mulit-floors config do not have adunit configured and no bids in the response",
			args: args{
				rctx: &models.RequestCtx{
					Endpoint: models.EndpointAppLovinMax,
					DefaultBids: map[string]map[string][]openrtb2.Bid{
						"test-impID-1": {
							"pubmatic": {
								{
									ID:    "dbbsdhkldks1234",
									ImpID: "test-impID-1",
									Ext:   []byte(`{}`),
								},
							},
						},
					},
					MultiFloors: map[string]*models.MultiFloors{},
					ImpBidCtx: map[string]models.ImpCtx{
						"test-impID-1": {
							TagID: "adunit-2",
							Bidders: map[string]models.PartnerData{
								"pubmatic": {
									PrebidBidderCode: "pubmatic",
								},
							},
							BidCtx: map[string]models.BidCtx{},
						},
					},
					PrebidBidderCode: map[string]string{
						"pubmatic": "pubmatic",
					},
				},
				bidResponse: &openrtb2.BidResponse{
					ID: "bid-1",
				},
			},
			fields: fields{
				uuidGenerator: TestUUIDGenerator{},
			},
			want: map[string]map[string][]openrtb2.Bid{
				"test-impID-1": {
					"pubmatic": {
						{
							ID:    "dbbsdhkldks1234",
							ImpID: "test-impID-1",
							Ext:   []byte(`{}`),
						},
					},
				},
			},
		},
		{
			name: "mulit-floors config have adunit configured but no floor in config and no bids in the response",
			args: args{
				rctx: &models.RequestCtx{
					Endpoint: models.EndpointAppLovinMax,
					DefaultBids: map[string]map[string][]openrtb2.Bid{
						"test-impID-1": {
							"pubmatic": {
								{
									ID:    "dbbsdhkldks1234",
									ImpID: "test-impID-1",
									Ext:   []byte(`{}`),
								},
							},
						},
					},
					MultiFloors: map[string]*models.MultiFloors{},
					ImpBidCtx: map[string]models.ImpCtx{
						"test-impID-1": {
							TagID: "adunit-1",
							Bidders: map[string]models.PartnerData{
								"pubmatic": {
									PrebidBidderCode: "pubmatic",
								},
							},
							BidCtx: map[string]models.BidCtx{},
						},
					},
					PrebidBidderCode: map[string]string{
						"pubmatic": "pubmatic",
					},
				},
				bidResponse: &openrtb2.BidResponse{
					ID: "bid-1",
				},
			},
			fields: fields{
				uuidGenerator: TestUUIDGenerator{},
			},
			want: map[string]map[string][]openrtb2.Bid{
				"test-impID-1": {
					"pubmatic": {
						{
							ID:    "dbbsdhkldks1234",
							ImpID: "test-impID-1",
							Ext:   []byte(`{}`),
						},
					},
				},
			},
		},
		{
			name: "mulit-floors config have three floors and only one bid in the response",
			args: args{
				rctx: &models.RequestCtx{
					Endpoint:    models.EndpointAppLovinMax,
					DefaultBids: map[string]map[string][]openrtb2.Bid{},
					MultiFloors: map[string]*models.MultiFloors{
						"test-impID-1": {Tier1: 1.1, Tier2: 2.1, Tier3: 3.1},
					},
					ImpBidCtx: map[string]models.ImpCtx{
						"test-impID-1": {
							TagID: "adunit-1",
							Bidders: map[string]models.PartnerData{
								"pubmatic": {
									PrebidBidderCode: "pubmatic",
								},
							},
							BidCtx: map[string]models.BidCtx{
								"pubmatic-bid-1": {
									BidExt: models.BidExt{
										MultiBidMultiFloorValue: 1.1,
									},
								},
							},
						},
					},
					PrebidBidderCode: map[string]string{
						"pubmatic": "pubmatic",
					},
				},
				bidResponse: &openrtb2.BidResponse{
					ID: "bid-1",
					SeatBid: []openrtb2.SeatBid{
						{
							Seat: "pubmatic",
							Bid: []openrtb2.Bid{
								{
									ID:    "pubmatic-bid-1",
									ImpID: "test-impID-1",
									Price: 1.5,
									Ext:   []byte(`{"mbmfv":1.1}`),
								},
							},
						},
					},
				},
			},
			fields: fields{
				uuidGenerator: TestUUIDGenerator{},
			},
			want: map[string]map[string][]openrtb2.Bid{
				"test-impID-1": {
					"pubmatic": {
						{
							ID:    "30470a14-2949-4110-abce-b62d57304ad5",
							ImpID: "test-impID-1",
						},
						{
							ID:    "30470a14-2949-4110-abce-b62d57304ad5",
							ImpID: "test-impID-1",
						},
					},
				},
			},
		},
		{
			name: "mulit-floors config have three floors and all three bid in the response",
			args: args{
				rctx: &models.RequestCtx{
					Endpoint:    models.EndpointAppLovinMax,
					DefaultBids: map[string]map[string][]openrtb2.Bid{},
					MultiFloors: map[string]*models.MultiFloors{
						"test-impID-1": {Tier1: 1.1, Tier2: 2.1, Tier3: 3.1},
					},
					ImpBidCtx: map[string]models.ImpCtx{
						"test-impID-1": {
							TagID: "adunit-1",
							Bidders: map[string]models.PartnerData{
								"pubmatic": {
									PrebidBidderCode: "pubmatic",
								},
							},
							BidCtx: map[string]models.BidCtx{
								"pubmatic-bid-1": {
									BidExt: models.BidExt{
										MultiBidMultiFloorValue: 1.1,
									},
								},
								"pubmatic-bid-2": {
									BidExt: models.BidExt{
										MultiBidMultiFloorValue: 2.1,
									},
								},
								"pubmatic-bid-3": {
									BidExt: models.BidExt{
										MultiBidMultiFloorValue: 3.1,
									},
								},
							},
						},
					},
					PrebidBidderCode: map[string]string{
						"pubmatic": "pubmatic",
					},
				},
				bidResponse: &openrtb2.BidResponse{
					ID: "bid-1",
					SeatBid: []openrtb2.SeatBid{
						{
							Seat: "pubmatic",
							Bid: []openrtb2.Bid{
								{
									ID:    "pubmatic-bid-1",
									ImpID: "test-impID-1",
									Price: 1.5,
									Ext:   []byte(`{"mbmfv":1.1}`),
								},
								{
									ID:    "pubmatic-bid-2",
									ImpID: "test-impID-1",
									Price: 2.5,
									Ext:   []byte(`{"mbmfv":2.1}`),
								},
								{
									ID:    "pubmatic-bid-3",
									ImpID: "test-impID-1",
									Price: 3.5,
								},
							},
						},
					},
				},
			},
			fields: fields{
				uuidGenerator: TestUUIDGenerator{},
			},
			want: map[string]map[string][]openrtb2.Bid{
				"test-impID-1": {},
			},
		},
		{
			name: "mulit-floors config have three floors and only one bid in the response for both partner pubmatic and pubmatic_1123",
			args: args{
				rctx: &models.RequestCtx{
					Endpoint:    models.EndpointAppLovinMax,
					DefaultBids: map[string]map[string][]openrtb2.Bid{},
					MultiFloors: map[string]*models.MultiFloors{
						"test-impID-1": {Tier1: 1.1, Tier2: 2.1, Tier3: 3.1},
					},
					ImpBidCtx: map[string]models.ImpCtx{
						"test-impID-1": {
							TagID: "adunit-1",
							Bidders: map[string]models.PartnerData{
								"pubmatic": {
									PrebidBidderCode: "pubmatic",
								},
								"pubmatic_1123": {
									PrebidBidderCode: "pubmatic",
								},
							},
							BidCtx: map[string]models.BidCtx{
								"pubmatic-bid-1": {
									BidExt: models.BidExt{
										MultiBidMultiFloorValue: 1.1,
									},
								},
								"pubmatic-bid-2": {
									BidExt: models.BidExt{
										MultiBidMultiFloorValue: 1.1,
									},
								},
							},
						},
					},
					PrebidBidderCode: map[string]string{
						"pubmatic_1123": "pubmatic",
						"pubmatic":      "pubmatic",
					},
				},
				bidResponse: &openrtb2.BidResponse{
					ID: "bid-1",
					SeatBid: []openrtb2.SeatBid{
						{
							Seat: "pubmatic",
							Bid: []openrtb2.Bid{
								{
									ID:    "pubmatic-bid-1",
									ImpID: "test-impID-1",
									Price: 1.5,
									Ext:   []byte(`{"mbmfv":1.1}`),
								},
							},
						},
						{
							Seat: "pubmatic_1123",
							Bid: []openrtb2.Bid{
								{
									ID:    "pubmatic-bid-2",
									ImpID: "test-impID-1",
									Price: 1.6,
									Ext:   []byte(`{"mbmfv":1.1}`),
								},
							},
						},
					},
				},
			},
			fields: fields{
				uuidGenerator: TestUUIDGenerator{},
			},
			want: map[string]map[string][]openrtb2.Bid{
				"test-impID-1": {
					"pubmatic": {
						{
							ID:    "30470a14-2949-4110-abce-b62d57304ad5",
							ImpID: "test-impID-1",
						},
						{
							ID:    "30470a14-2949-4110-abce-b62d57304ad5",
							ImpID: "test-impID-1",
						},
					},
					"pubmatic_1123": {
						{
							ID:    "30470a14-2949-4110-abce-b62d57304ad5",
							ImpID: "test-impID-1",
						},
						{
							ID:    "30470a14-2949-4110-abce-b62d57304ad5",
							ImpID: "test-impID-1",
						},
					},
				},
			},
		},
		{
			name: "mulit-floors config have three floors and no bid in the response for both partner pubmatic and pubmatic_1123",
			args: args{
				rctx: &models.RequestCtx{
					Endpoint:    models.EndpointAppLovinMax,
					DefaultBids: map[string]map[string][]openrtb2.Bid{},
					MultiFloors: map[string]*models.MultiFloors{
						"test-impID-1": {Tier1: 1.1, Tier2: 2.1, Tier3: 3.1},
					},
					ImpBidCtx: map[string]models.ImpCtx{
						"test-impID-1": {
							TagID: "adunit-1",
							Bidders: map[string]models.PartnerData{
								"pubmatic": {
									PrebidBidderCode: "pubmatic",
								},
								"pubmatic_1123": {
									PrebidBidderCode: "pubmatic",
								},
							},
							BidCtx: map[string]models.BidCtx{},
						},
					},
					PrebidBidderCode: map[string]string{
						"pubmatic_1123": "pubmatic",
						"pubmatic":      "pubmatic",
					},
				},
				bidResponse: &openrtb2.BidResponse{
					ID:      "bid-1",
					SeatBid: []openrtb2.SeatBid{},
				},
			},
			fields: fields{
				uuidGenerator: TestUUIDGenerator{},
			},
			want: map[string]map[string][]openrtb2.Bid{
				"test-impID-1": {
					"pubmatic": {
						{
							ID:    "30470a14-2949-4110-abce-b62d57304ad5",
							ImpID: "test-impID-1",
						},
						{
							ID:    "30470a14-2949-4110-abce-b62d57304ad5",
							ImpID: "test-impID-1",
						},
						{
							ID:    "30470a14-2949-4110-abce-b62d57304ad5",
							ImpID: "test-impID-1",
						},
					},
					"pubmatic_1123": {
						{
							ID:    "30470a14-2949-4110-abce-b62d57304ad5",
							ImpID: "test-impID-1",
						},
						{
							ID:    "30470a14-2949-4110-abce-b62d57304ad5",
							ImpID: "test-impID-1",
						},
						{
							ID:    "30470a14-2949-4110-abce-b62d57304ad5",
							ImpID: "test-impID-1",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &OpenWrap{
				cfg:             tt.fields.cfg,
				metricEngine:    tt.fields.metricEngine,
				rateConvertor:   tt.fields.rateConvertor,
				geoInfoFetcher:  tt.fields.geoInfoFetcher,
				pubFeatures:     tt.fields.pubFeatures,
				unwrap:          tt.fields.unwrap,
				profileMetaData: tt.fields.profileMetaData,
				uuidGenerator:   tt.fields.uuidGenerator,
			}
			got := m.addDefaultBidsForMultiFloorsConfig(tt.args.rctx, tt.args.bidResponse, tt.args.bidResponseExt)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOpenWrap_applyDefaultBids(t *testing.T) {
	m := &OpenWrap{}
	type args struct {
		rctx        models.RequestCtx
		bidResponse *openrtb2.BidResponse
	}
	tests := []struct {
		name string
		args args
		want *openrtb2.BidResponse
	}{
		{
			name: "sendAllBids_true_appends_SeatBid_for_default_bids_when_no_matching_seat",
			args: args{
				rctx: models.RequestCtx{
					SendAllBids: true,
					DefaultBids: map[string]map[string][]openrtb2.Bid{
						"imp-1": {
							"openx": {
								{ID: "def-openx-1", ImpID: "imp-1", Price: 0},
							},
						},
					},
				},
				bidResponse: &openrtb2.BidResponse{
					ID: "resp-1",
					SeatBid: []openrtb2.SeatBid{
						{
							Seat: "pubmatic",
							Bid: []openrtb2.Bid{
								{ID: "win-pm-1", ImpID: "imp-1", Price: 1.5},
							},
						},
					},
				},
			},
			want: &openrtb2.BidResponse{
				ID: "resp-1",
				SeatBid: []openrtb2.SeatBid{
					{
						Seat: "pubmatic",
						Bid: []openrtb2.Bid{
							{ID: "win-pm-1", ImpID: "imp-1", Price: 1.5},
						},
					},
					{
						Seat: "openx",
						Bid: []openrtb2.Bid{
							{ID: "def-openx-1", ImpID: "imp-1", Price: 0},
						},
					},
				},
			},
		},
		{
			name: "sendAllBids_false_does_not_append_when_imp_has_any_bid_pubmatic_covers_imp1",
			args: args{
				rctx: models.RequestCtx{
					SendAllBids: false,
					DefaultBids: map[string]map[string][]openrtb2.Bid{
						"imp-1": {
							"openx": {
								{ID: "def-openx-1", ImpID: "imp-1", Price: 0},
							},
						},
					},
				},
				bidResponse: &openrtb2.BidResponse{
					ID: "resp-1",
					SeatBid: []openrtb2.SeatBid{
						{
							Seat: "pubmatic",
							Bid: []openrtb2.Bid{
								{ID: "win-pm-1", ImpID: "imp-1", Price: 1.5},
							},
						},
					},
				},
			},
			want: &openrtb2.BidResponse{
				ID: "resp-1",
				SeatBid: []openrtb2.SeatBid{
					{
						Seat: "pubmatic",
						Bid: []openrtb2.Bid{
							{ID: "win-pm-1", ImpID: "imp-1", Price: 1.5},
						},
					},
				},
			},
		},
		{
			name: "sendAllBids_false_does_not_append_when_imp_has_any_bid_slot_not_mapped_defaults_unused",
			args: args{
				rctx: models.RequestCtx{
					SendAllBids: false,
					ImpBidCtx: map[string]models.ImpCtx{
						"imp-1": {
							NonMapped: map[string]struct{}{
								"appnexus": {},
							},
						},
					},
					DefaultBids: map[string]map[string][]openrtb2.Bid{
						"imp-1": {
							"appnexus": {
								{ID: "def-appnexus-1", ImpID: "imp-1", Price: 0},
							},
						},
					},
				},
				bidResponse: &openrtb2.BidResponse{
					ID: "resp-1",
					SeatBid: []openrtb2.SeatBid{
						{
							Seat: "pubmatic",
							Bid: []openrtb2.Bid{
								{ID: "win-pm-1", ImpID: "imp-1", Price: 1.5},
							},
						},
					},
				},
			},
			want: &openrtb2.BidResponse{
				ID: "resp-1",
				SeatBid: []openrtb2.SeatBid{
					{
						Seat: "pubmatic",
						Bid: []openrtb2.Bid{
							{ID: "win-pm-1", ImpID: "imp-1", Price: 1.5},
						},
					},
				},
			},
		},
		{
			name: "sendAllBids_false_empty_SeatBid_single_bidder_default_per_imp",
			args: args{
				rctx: models.RequestCtx{
					SendAllBids: false,
					ImpBidCtx: map[string]models.ImpCtx{
						"imp-1": {
							NonMapped: map[string]struct{}{
								"pubmatic": {},
							},
						},
					},
					DefaultBids: map[string]map[string][]openrtb2.Bid{
						"imp-1": {
							"pubmatic": {
								{ID: "def-pm", ImpID: "imp-1", Price: 0},
							},
						},
					},
				},
				bidResponse: &openrtb2.BidResponse{
					ID:      "resp-1",
					SeatBid: []openrtb2.SeatBid{},
				},
			},
			want: &openrtb2.BidResponse{
				ID: "resp-1",
				SeatBid: []openrtb2.SeatBid{
					{
						Seat: models.BidderPubMatic,
						Bid: []openrtb2.Bid{
							{ID: "def-pm", ImpID: "imp-1", Price: 0},
						},
					},
				},
			},
		},
		{
			name: "sendAllBids_false_empty_SeatBid_one_SeatBid_per_imp_sorted_impid",
			args: args{
				rctx: models.RequestCtx{
					SendAllBids: false,
					ImpBidCtx: map[string]models.ImpCtx{
						"imp-1": {
							NonMapped: map[string]struct{}{
								"pubmatic": {},
							},
						},
						"imp-2": {
							NonMapped: map[string]struct{}{
								"pubmatic": {},
							},
						},
					},
					DefaultBids: map[string]map[string][]openrtb2.Bid{
						"imp-2": {
							"pubmatic": {
								{ID: "def-pm-2", ImpID: "imp-2", Price: 0},
							},
						},
						"imp-1": {
							"pubmatic": {
								{ID: "def-pm-1", ImpID: "imp-1", Price: 0},
							},
						},
					},
				},
				bidResponse: &openrtb2.BidResponse{
					ID:      "resp-1",
					SeatBid: []openrtb2.SeatBid{},
				},
			},
			want: &openrtb2.BidResponse{
				ID: "resp-1",
				SeatBid: []openrtb2.SeatBid{
					{
						Seat: models.BidderPubMatic,
						Bid: []openrtb2.Bid{
							{ID: "def-pm-1", ImpID: "imp-1", Price: 0},
						},
					},
					{
						Seat: models.BidderPubMatic,
						Bid: []openrtb2.Bid{
							{ID: "def-pm-2", ImpID: "imp-2", Price: 0},
						},
					},
				},
			},
		},
		{
			name: "sendAllBids_false_empty_SeatBid_different_first_bidder_per_imp",
			args: args{
				rctx: models.RequestCtx{
					SendAllBids: false,
					ImpBidCtx: map[string]models.ImpCtx{
						"imp-1": {
							NonMapped: map[string]struct{}{
								"appnexus": {},
							},
						},
						"imp-2": {
							NonMapped: map[string]struct{}{
								"rubicon": {},
							},
						},
					},
					DefaultBids: map[string]map[string][]openrtb2.Bid{
						"imp-1": {
							"appnexus": {
								{ID: "def-an-1", ImpID: "imp-1", Price: 0},
							},
						},
						"imp-2": {
							"rubicon": {
								{ID: "def-rb-2", ImpID: "imp-2", Price: 0},
							},
						},
					},
				},
				bidResponse: &openrtb2.BidResponse{
					ID:      "resp-1",
					SeatBid: []openrtb2.SeatBid{},
				},
			},
			want: &openrtb2.BidResponse{
				ID: "resp-1",
				SeatBid: []openrtb2.SeatBid{
					{
						Seat: "appnexus",
						Bid: []openrtb2.Bid{
							{ID: "def-an-1", ImpID: "imp-1", Price: 0},
						},
					},
					{
						Seat: "rubicon",
						Bid: []openrtb2.Bid{
							{ID: "def-rb-2", ImpID: "imp-2", Price: 0},
						},
					},
				},
			},
		},
		{
			name: "sendAllBids_false_empty_SeatBid_three_imps_one_bidder_each_distinct_seat",
			args: args{
				rctx: models.RequestCtx{
					SendAllBids: false,
					ImpBidCtx: map[string]models.ImpCtx{
						"imp-1": {
							NonMapped: map[string]struct{}{
								"appnexus": {},
							},
						},
						"imp-2": {
							NonMapped: map[string]struct{}{
								"rubicon": {},
							},
						},
						"imp-3": {
							NonMapped: map[string]struct{}{
								"openx": {},
							},
						},
					},
					DefaultBids: map[string]map[string][]openrtb2.Bid{
						"imp-3": {
							"openx": {
								{ID: "def-ox-3", ImpID: "imp-3", Price: 0},
							},
						},
						"imp-1": {
							"appnexus": {
								{ID: "def-an-1", ImpID: "imp-1", Price: 0},
							},
						},
						"imp-2": {
							"rubicon": {
								{ID: "def-rb-2", ImpID: "imp-2", Price: 0},
							},
						},
					},
				},
				bidResponse: &openrtb2.BidResponse{
					ID:      "resp-1",
					SeatBid: []openrtb2.SeatBid{},
				},
			},
			want: &openrtb2.BidResponse{
				ID: "resp-1",
				SeatBid: []openrtb2.SeatBid{
					{
						Seat: "appnexus",
						Bid: []openrtb2.Bid{
							{ID: "def-an-1", ImpID: "imp-1", Price: 0},
						},
					},
					{
						Seat: "rubicon",
						Bid: []openrtb2.Bid{
							{ID: "def-rb-2", ImpID: "imp-2", Price: 0},
						},
					},
					{
						Seat: "openx",
						Bid: []openrtb2.Bid{
							{ID: "def-ox-3", ImpID: "imp-3", Price: 0},
						},
					},
				},
			},
		},
		{
			name: "sendAllBids_false_pubmatic_imp1_win_appends_one_default_SeatBid_for_imp2_only",
			args: args{
				rctx: models.RequestCtx{
					SendAllBids: false,
					ImpBidCtx: map[string]models.ImpCtx{
						"imp-1": {
							NonMapped: map[string]struct{}{
								"appnexus": {},
								"rubicon":  {},
								"openx":    {},
							},
						},
						"imp-2": {
							NonMapped: map[string]struct{}{
								"appnexus": {},
							},
						},
					},
					DefaultBids: map[string]map[string][]openrtb2.Bid{
						"imp-1": {
							"appnexus": {
								{ID: "def-an-1", ImpID: "imp-1", Price: 0},
							},
							"rubicon": {
								{ID: "def-rb-1", ImpID: "imp-1", Price: 0},
							},
							"openx": {
								{ID: "def-ox-1", ImpID: "imp-1", Price: 0},
							},
						},
						"imp-2": {
							"appnexus": {
								{ID: "def-an-2", ImpID: "imp-2", Price: 0},
							},
						},
					},
				},
				bidResponse: &openrtb2.BidResponse{
					ID:      "resp-saf-false-2imp-placeholder",
					SeatBid: []openrtb2.SeatBid{{Seat: "pubmatic", Bid: []openrtb2.Bid{{ID: "win-pm-1", ImpID: "imp-1", Price: 1.5}}}},
				},
			},
			want: &openrtb2.BidResponse{
				ID: "resp-saf-false-2imp-placeholder",
				SeatBid: []openrtb2.SeatBid{
					{
						Seat: "pubmatic",
						Bid: []openrtb2.Bid{
							{ID: "win-pm-1", ImpID: "imp-1", Price: 1.5},
						},
					},
					{
						Seat: "appnexus",
						Bid: []openrtb2.Bid{
							{ID: "def-an-2", ImpID: "imp-2", Price: 0},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			br := tt.args.bidResponse
			got, err := m.applyDefaultBids(tt.args.rctx, br)
			assert.NoError(t, err)

			if !tt.args.rctx.SendAllBids {
				assert.ElementsMatch(t, tt.want.SeatBid, got.SeatBid)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAppendDefaultSeatBids(t *testing.T) {
	type args struct {
		rctx models.RequestCtx
		resp openrtb2.BidResponse
	}
	tests := []struct {
		name string
		args args
		want openrtb2.BidResponse
	}{
		{
			name: "empty_DefaultBids_noop",
			args: args{
				rctx: models.RequestCtx{},
				resp: openrtb2.BidResponse{
					SeatBid: []openrtb2.SeatBid{{Seat: "pubmatic", Bid: []openrtb2.Bid{{ID: "w", ImpID: "imp-1", Price: 1}}}},
				},
			},
			want: openrtb2.BidResponse{
				SeatBid: []openrtb2.SeatBid{{Seat: "pubmatic", Bid: []openrtb2.Bid{{ID: "w", ImpID: "imp-1", Price: 1}}}},
			},
		},
		{
			name: "adds_one_placeholder_when_SeatBid_empty",
			args: args{
				rctx: models.RequestCtx{
					DefaultBids: map[string]map[string][]openrtb2.Bid{
						"imp-1": {"pubmatic": {{ID: "def-pm", ImpID: "imp-1", Price: 0, W: 0, H: 0}}},
					},
				},
				resp: openrtb2.BidResponse{SeatBid: []openrtb2.SeatBid{}},
			},
			want: openrtb2.BidResponse{
				SeatBid: []openrtb2.SeatBid{
					{
						Seat: "pubmatic",
						Bid:  []openrtb2.Bid{{ID: "def-pm", ImpID: "imp-1", Price: 0, W: 0, H: 0}},
					},
				},
			},
		},
		{
			name: "skips_impression_that_already_has_bid_in_SeatBid",
			args: args{
				rctx: models.RequestCtx{
					DefaultBids: map[string]map[string][]openrtb2.Bid{
						"imp-1": {"openx": {{ID: "def-ox", ImpID: "imp-1", Price: 0, W: 0, H: 0}}},
					},
				},
				resp: openrtb2.BidResponse{
					SeatBid: []openrtb2.SeatBid{
						{Seat: "pubmatic", Bid: []openrtb2.Bid{{ID: "win", ImpID: "imp-1", Price: 2}}},
					},
				},
			},
			want: openrtb2.BidResponse{
				SeatBid: []openrtb2.SeatBid{
					{Seat: "pubmatic", Bid: []openrtb2.Bid{{ID: "win", ImpID: "imp-1", Price: 2}}},
				},
			},
		},
		{
			name: "placeholder_only_for_imp_without_bid_multi_imp_DefaultBids",
			args: args{
				rctx: models.RequestCtx{
					DefaultBids: map[string]map[string][]openrtb2.Bid{
						"imp-1": {"openx": {{ID: "def-ox-1", ImpID: "imp-1", Price: 0, W: 0, H: 0}}},
						"imp-2": {"appnexus": {{ID: "def-an-2", ImpID: "imp-2", Price: 0, W: 0, H: 0}}},
					},
				},
				resp: openrtb2.BidResponse{
					SeatBid: []openrtb2.SeatBid{
						{Seat: "pubmatic", Bid: []openrtb2.Bid{{ID: "win-1", ImpID: "imp-1", Price: 1.5}}},
					},
				},
			},
			want: openrtb2.BidResponse{
				SeatBid: []openrtb2.SeatBid{
					{Seat: "pubmatic", Bid: []openrtb2.Bid{{ID: "win-1", ImpID: "imp-1", Price: 1.5}}},
					{
						Seat: "appnexus",
						Bid:  []openrtb2.Bid{{ID: "def-an-2", ImpID: "imp-2", Price: 0, W: 0, H: 0}},
					},
				},
			},
		},
		{
			name: "appends_only_bids0_for_chosen_seat",
			args: args{
				rctx: models.RequestCtx{
					DefaultBids: map[string]map[string][]openrtb2.Bid{
						"imp-1": {"pubmatic": {
							{ID: "def-a", ImpID: "imp-1", Price: 0, W: 0, H: 0},
							{ID: "def-b", ImpID: "imp-1", Price: 0, W: 0, H: 0},
						}},
					},
				},
				resp: openrtb2.BidResponse{SeatBid: []openrtb2.SeatBid{}},
			},
			want: openrtb2.BidResponse{
				SeatBid: []openrtb2.SeatBid{
					{
						Seat: "pubmatic",
						Bid:  []openrtb2.Bid{{ID: "def-a", ImpID: "imp-1", Price: 0, W: 0, H: 0}},
					},
				},
			},
		},
		{
			name: "nil_inner_map_no_panic_skips_imp",
			args: args{
				rctx: models.RequestCtx{
					DefaultBids: map[string]map[string][]openrtb2.Bid{
						"imp-1": nil,
						"imp-2": {"z": {{ID: "d2", ImpID: "imp-2", Price: 0, W: 0, H: 0}}},
					},
				},
				resp: openrtb2.BidResponse{SeatBid: []openrtb2.SeatBid{}},
			},
			want: openrtb2.BidResponse{
				SeatBid: []openrtb2.SeatBid{
					{
						Seat: "z",
						Bid:  []openrtb2.Bid{{ID: "d2", ImpID: "imp-2", Price: 0, W: 0, H: 0}},
					},
				},
			},
		},
		{
			name: "skips_seat_with_empty_bid_slice",
			args: args{
				rctx: models.RequestCtx{
					DefaultBids: map[string]map[string][]openrtb2.Bid{
						"imp-1": {
							"ghost": {},
							"openx": {{ID: "def-ox", ImpID: "imp-1", Price: 0, W: 0, H: 0}},
						},
					},
				},
				resp: openrtb2.BidResponse{SeatBid: []openrtb2.SeatBid{}},
			},
			want: openrtb2.BidResponse{
				SeatBid: []openrtb2.SeatBid{
					{
						Seat: "openx",
						Bid:  []openrtb2.Bid{{ID: "def-ox", ImpID: "imp-1", Price: 0, W: 0, H: 0}},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appendDefaultSeatBids(tt.args.rctx, &tt.args.resp)
			assert.Equal(t, tt.want, tt.args.resp)
		})
	}
}
