package pubmatic

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/openrtb/v20/openrtb3"
	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/exchange"
	"github.com/prebid/prebid-server/v3/hooks/hookanalytics"
	"github.com/prebid/prebid-server/v3/hooks/hookexecution"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models/nbr"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

func TestGetRequestCtx(t *testing.T) {
	tests := []struct {
		name                 string
		hookExecutionOutcome []hookexecution.StageOutcome
		rctx                 *models.RequestCtx
	}{
		{
			name: "rctx present",
			hookExecutionOutcome: []hookexecution.StageOutcome{
				{
					Groups: []hookexecution.GroupOutcome{
						{
							InvocationResults: []hookexecution.HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{
										Activities: []hookanalytics.Activity{
											{
												Results: []hookanalytics.Result{
													{
														Values: map[string]interface{}{
															"request-ctx": &models.RequestCtx{},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			rctx: &models.RequestCtx{},
		},
		{
			name: "rctx of invalid type",
			hookExecutionOutcome: []hookexecution.StageOutcome{
				{
					Groups: []hookexecution.GroupOutcome{
						{
							InvocationResults: []hookexecution.HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{
										Activities: []hookanalytics.Activity{
											{
												Results: []hookanalytics.Result{
													{
														Values: map[string]interface{}{
															"request-ctx": []string{},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			rctx: nil,
		},
		{
			name: "rctx absent",
			hookExecutionOutcome: []hookexecution.StageOutcome{
				{
					Groups: []hookexecution.GroupOutcome{
						{
							InvocationResults: []hookexecution.HookOutcome{},
						},
					},
				},
			},
			rctx: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rctx := GetRequestCtx(tt.hookExecutionOutcome)
			assert.Equal(t, tt.rctx, rctx, tt.name)
		})
	}
}
func TestConvertNonBidToBid(t *testing.T) {
	tests := []struct {
		name   string
		nonBid openrtb_ext.NonBid
		bid    bidWrapper
	}{
		{
			name: "nonbid to bidwrapper",
			nonBid: openrtb_ext.NonBid{
				StatusCode: int(exchange.ResponseRejectedBelowFloor),
				ImpId:      "imp1",
				Ext: openrtb_ext.ExtNonBid{
					Prebid: openrtb_ext.ExtNonBidPrebid{
						Bid: openrtb_ext.ExtNonBidPrebidBid{
							Price:             10,
							ADomain:           []string{"abc.com"},
							DealID:            "d1",
							OriginalBidCPM:    10,
							OriginalBidCur:    models.USD,
							OriginalBidCPMUSD: 0,
							W:                 10,
							H:                 50,
							DealPriority:      1,
							Video: &openrtb_ext.ExtBidPrebidVideo{
								Duration: 10,
							},
						},
					},
				},
			},
			bid: bidWrapper{
				&openrtb2.Bid{
					ImpID:   "imp1",
					Price:   10,
					ADomain: []string{"abc.com"},
					DealID:  "d1",
					W:       10,
					H:       50,
					Ext:     json.RawMessage(`{"prebid":{"dealpriority":1,"video":{"duration":10,"primary_category":"","vasttagid":""}},"origbidcpm":10,"origbidcur":"USD"}`),
				},
				exchange.ResponseRejectedBelowFloor.Ptr(),
			},
		},
		{
			name: "nonbid to bidwrapper with bundle",
			nonBid: openrtb_ext.NonBid{
				StatusCode: int(exchange.ResponseRejectedBelowFloor),
				ImpId:      "imp1",
				Ext: openrtb_ext.ExtNonBid{
					Prebid: openrtb_ext.ExtNonBidPrebid{
						Bid: openrtb_ext.ExtNonBidPrebidBid{
							Price:             10,
							ADomain:           []string{"abc.com"},
							DealID:            "d1",
							OriginalBidCPM:    10,
							OriginalBidCur:    models.USD,
							OriginalBidCPMUSD: 0,
							W:                 10,
							H:                 50,
							DealPriority:      1,
							Video: &openrtb_ext.ExtBidPrebidVideo{
								Duration: 10,
							},
							Bundle: "dummy_bundle",
						},
					},
				},
			},
			bid: bidWrapper{
				&openrtb2.Bid{
					ImpID:   "imp1",
					Price:   10,
					ADomain: []string{"abc.com"},
					DealID:  "d1",
					W:       10,
					H:       50,
					Bundle:  "dummy_bundle",
					Ext:     json.RawMessage(`{"prebid":{"dealpriority":1,"video":{"duration":10,"primary_category":"","vasttagid":""}},"origbidcpm":10,"origbidcur":"USD"}`),
				},
				exchange.ResponseRejectedBelowFloor.Ptr(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bid := convertNonBidToBidWrapper(&tt.nonBid)
			assert.Equal(t, tt.bid, bid, tt.name)
		})
	}
}
func TestGetDefaultPartnerRecordsByImp(t *testing.T) {
	tests := []struct {
		name     string
		rCtx     *models.RequestCtx
		partners map[string][]PartnerRecord
	}{
		{
			name:     "empty ImpBidCtx",
			rCtx:     &models.RequestCtx{},
			partners: map[string][]PartnerRecord{},
		},
		{
			name: "multiple imps",
			rCtx: &models.RequestCtx{
				ImpBidCtx: map[string]models.ImpCtx{
					"imp1": {},
					"imp2": {},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					PartnerRecord{
						ServerSide:       1,
						DefaultBidStatus: 1,
						PartnerSize:      "0x0",
						DealID:           "-1",
					},
				},
				"imp2": {
					PartnerRecord{
						ServerSide:       1,
						DefaultBidStatus: 1,
						PartnerSize:      "0x0",
						DealID:           "-1",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			partners := getDefaultPartnerRecordsByImp(tt.rCtx)
			assert.Equal(t, len(tt.partners), len(partners), tt.name)
			for ind := range partners {
				// ignore order of elements in slice while comparison
				assert.ElementsMatch(t, partners[ind], tt.partners[ind], tt.name)
			}
		})
	}
}
func TestGetPartnerRecordsByImp(t *testing.T) {
	type args struct {
		ao   analytics.AuctionObject
		rCtx *models.RequestCtx
	}
	tests := []struct {
		name     string
		args     args
		partners map[string][]PartnerRecord
	}{
		{
			name: "adformat for default bid",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										Price: 0,
									},
								},
								Seat: "pubmatic",
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										ExtBid: openrtb_ext.ExtBid{},
									},
								},
							},
							Video: &openrtb2.Video{
								MinDuration: 1,
								MaxDuration: 10,
							},
						},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:        "pubmatic",
						BidderCode:       "pubmatic",
						PartnerSize:      "0x0",
						BidID:            "bid-id-1",
						OrigBidID:        "bid-id-1",
						DealID:           "-1",
						ServerSide:       1,
						OriginalCur:      "USD",
						Adformat:         models.Video,
						DefaultBidStatus: 1,
					},
				},
			},
		},
		{
			name: "adformat for valid bid",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										Price: 10,
										AdM:   "{\"native\":{\"assets\":[{\"id\":1,\"required\":0,\"title\":{\"text\":\"Lexus - Luxury vehicles company\"}},{\"id\":2,\"img\":{\"h\":150,\"url\":\"https://stagingnyc.pubmatic.com:8443//sdk/lexus_logo.png\",\"w\":150},\"required\":0},{\"id\":3,\"img\":{\"h\":428,\"url\":\"https://stagingnyc.pubmatic.com:8443//sdk/28f48244cafa0363b03899f267453fe7%20copy.png\",\"w\":214},\"required\":0},{\"data\":{\"value\":\"Goto PubMatic\"},\"id\":4,\"required\":0},{\"data\":{\"value\":\"Lexus - Luxury vehicles company\"},\"id\":5,\"required\":0},{\"data\":{\"value\":\"4\"},\"id\":6,\"required\":0}],\"imptrackers\":[\"http://phtrack.pubmatic.com/?ts=1496043362&r=84137f17-eefd-4f06-8380-09138dc616e6&i=c35b1240-a0b3-4708-afca-54be95283c61&a=130917&t=9756&au=10002949&p=&c=10014299&o=10002476&wl=10009731&ty=1\"],\"link\":{\"clicktrackers\":[\"http://ct.pubmatic.com/track?ts=1496043362&r=84137f17-eefd-4f06-8380-09138dc616e6&i=c35b1240-a0b3-4708-afca-54be95283c61&a=130917&t=9756&au=10002949&p=&c=10014299&o=10002476&wl=10009731&ty=3&url=\"],\"url\":\"http://www.lexus.com/\"},\"ver\":1}}",
									},
								},
								Seat: "pubmatic",
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										ExtBid:         openrtb_ext.ExtBid{},
										OriginalBidCPM: 10,
									},
								},
							},
							Native: &openrtb2.Native{},
						},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:   "pubmatic",
						BidderCode:  "pubmatic",
						PartnerSize: "0x0",
						BidID:       "bid-id-1",
						OrigBidID:   "bid-id-1",
						DealID:      "-1",
						ServerSide:  1,
						OriginalCur: "USD",
						Adformat:    models.Native,
						NetECPM:     10,
						GrossECPM:   10,
					},
				},
			},
		},
		{
			name: "latency for partner",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										Price: 10,
										AdM:   "{\"native\":{\"assets\":[{\"id\":1,\"required\":0,\"title\":{\"text\":\"Lexus - Luxury vehicles company\"}},{\"id\":2,\"img\":{\"h\":150,\"url\":\"https://stagingnyc.pubmatic.com:8443//sdk/lexus_logo.png\",\"w\":150},\"required\":0},{\"id\":3,\"img\":{\"h\":428,\"url\":\"https://stagingnyc.pubmatic.com:8443//sdk/28f48244cafa0363b03899f267453fe7%20copy.png\",\"w\":214},\"required\":0},{\"data\":{\"value\":\"Goto PubMatic\"},\"id\":4,\"required\":0},{\"data\":{\"value\":\"Lexus - Luxury vehicles company\"},\"id\":5,\"required\":0},{\"data\":{\"value\":\"4\"},\"id\":6,\"required\":0}],\"imptrackers\":[\"http://phtrack.pubmatic.com/?ts=1496043362&r=84137f17-eefd-4f06-8380-09138dc616e6&i=c35b1240-a0b3-4708-afca-54be95283c61&a=130917&t=9756&au=10002949&p=&c=10014299&o=10002476&wl=10009731&ty=1\"],\"link\":{\"clicktrackers\":[\"http://ct.pubmatic.com/track?ts=1496043362&r=84137f17-eefd-4f06-8380-09138dc616e6&i=c35b1240-a0b3-4708-afca-54be95283c61&a=130917&t=9756&au=10002949&p=&c=10014299&o=10002476&wl=10009731&ty=3&url=\"],\"url\":\"http://www.lexus.com/\"},\"ver\":1}}",
									},
								},
								Seat: "pubmatic",
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										ExtBid:         openrtb_ext.ExtBid{},
										OriginalBidCPM: 10,
									},
								},
							},
							Native: &openrtb2.Native{},
						},
					},
					BidderResponseTimeMillis: map[string]int{
						"pubmatic": 20,
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:   "pubmatic",
						BidderCode:  "pubmatic",
						PartnerSize: "0x0",
						BidID:       "bid-id-1",
						OrigBidID:   "bid-id-1",
						DealID:      "-1",
						ServerSide:  1,
						OriginalCur: "USD",
						Adformat:    models.Native,
						NetECPM:     10,
						GrossECPM:   10,
						Latency1:    20,
					},
				},
			},
		},
		{
			name: "matchedimpression for partner",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										Price: 10,
									},
								},
								Seat: "pubmatic",
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										ExtBid:         openrtb_ext.ExtBid{},
										OriginalBidCPM: 10,
									},
								},
							},
						},
					},
					MatchedImpression: map[string]int{
						"pubmatic": 1,
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:         "pubmatic",
						BidderCode:        "pubmatic",
						PartnerSize:       "0x0",
						BidID:             "bid-id-1",
						OrigBidID:         "bid-id-1",
						DealID:            "-1",
						ServerSide:        1,
						OriginalCur:       "USD",
						NetECPM:           10,
						GrossECPM:         10,
						MatchedImpression: 1,
					},
				},
			},
		},
		{
			name: "partnersize for non-video bid",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										Price: 10,
										W:     30,
										H:     50,
									},
								},
								Seat: "pubmatic",
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										ExtBid:         openrtb_ext.ExtBid{},
										OriginalBidCPM: 10,
									},
								},
							},
						},
					},
					Platform: models.PLATFORM_DISPLAY,
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:   "pubmatic",
						BidderCode:  "pubmatic",
						PartnerSize: "30x50",
						BidID:       "bid-id-1",
						OrigBidID:   "bid-id-1",
						DealID:      "-1",
						ServerSide:  1,
						OriginalCur: "USD",
						NetECPM:     10,
						GrossECPM:   10,
					},
				},
			},
		},
		{
			name: "partnersize for video bid",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										Price: 10,
										W:     30,
										H:     50,
									},
								},
								Seat: "pubmatic",
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										ExtBid:         openrtb_ext.ExtBid{},
										OriginalBidCPM: 10,
									},
								},
							},
						},
					},
					Platform: models.PLATFORM_VIDEO,
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:   "pubmatic",
						BidderCode:  "pubmatic",
						PartnerSize: "30x50",
						BidID:       "bid-id-1",
						OrigBidID:   "bid-id-1",
						DealID:      "-1",
						ServerSide:  1,
						OriginalCur: "USD",
						NetECPM:     10,
						GrossECPM:   10,
					},
				},
			},
		},
		{
			name: "dealid present, verify dealid and dealchannel",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:     "bid-id-1",
										ImpID:  "imp1",
										Price:  10,
										DealID: "pubdeal",
									},
								},
								Seat: "pubmatic",
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										ExtBid:         openrtb_ext.ExtBid{},
										OriginalBidCPM: 10,
									},
								},
							},
						},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:   "pubmatic",
						BidderCode:  "pubmatic",
						PartnerSize: "0x0",
						BidID:       "bid-id-1",
						OrigBidID:   "bid-id-1",
						DealID:      "pubdeal",
						DealChannel: "PMP",
						ServerSide:  1,
						OriginalCur: "USD",
						NetECPM:     10,
						GrossECPM:   10,
					},
				},
			},
		},
		{
			name: "log adomain field",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "appnexus",
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										ADomain: []string{
											"http://google.com", "http://yahoo.com",
										},
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										ExtBid: openrtb_ext.ExtBid{
											Prebid: &openrtb_ext.ExtBidPrebid{
												BidId: "prebid-bid-id-1",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:        "appnexus",
						BidderCode:       "appnexus",
						PartnerSize:      "0x0",
						BidID:            "prebid-bid-id-1",
						OrigBidID:        "bid-id-1",
						DealID:           "-1",
						ServerSide:       1,
						OriginalCur:      models.USD,
						ADomain:          "google.com",
						DefaultBidStatus: 1,
					},
				},
			},
		},
		{
			name: "log bundle field if not empty",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "appnexus",
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										ADomain: []string{
											"http://google.com", "http://yahoo.com",
										},
										Bundle: "dummy_bundle",
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										ExtBid: openrtb_ext.ExtBid{
											Prebid: &openrtb_ext.ExtBidPrebid{
												BidId: "prebid-bid-id-1",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:        "appnexus",
						BidderCode:       "appnexus",
						PartnerSize:      "0x0",
						BidID:            "prebid-bid-id-1",
						OrigBidID:        "bid-id-1",
						DealID:           "-1",
						ServerSide:       1,
						OriginalCur:      models.USD,
						ADomain:          "google.com",
						DefaultBidStatus: 1,
						Bundle:           "dummy_bundle",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			partners := getPartnerRecordsByImp(tt.args.ao, tt.args.rCtx)
			assert.Equalf(t, partners, tt.partners, tt.name)
		})
	}
}
func TestGetPartnerRecordsByImpForTracker(t *testing.T) {
	pg, _ := openrtb_ext.NewPriceGranularityFromLegacyID("med")
	type args struct {
		ao   analytics.AuctionObject
		rCtx *models.RequestCtx
	}
	tests := []struct {
		name     string
		args     args
		partners map[string][]PartnerRecord
	}{
		{
			name: "prefer tracker details, avoid computation",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										Price: 10,
									},
								},
								Seat: "pubmatic",
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					Trackers: map[string]models.OWTracker{
						"bid-id-1": {
							IsOMEnabled: true,
							Tracker: models.Tracker{
								PartnerInfo: models.Partner{
									Adformat:               models.Native,
									KGPV:                   "kgpv",
									NetECPM:                10,
									GrossECPM:              12,
									AdSize:                 "15x15",
									FloorValue:             1,
									FloorRuleValue:         2,
									Advertiser:             "sony.com",
									PriceBucket:            "10.00",
									MultiBidMultiFloorFlag: 1,
								},
								LoggerData: models.LoggerData{
									KGPSV: "kgpsv",
								},
							},
						},
					},
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										OriginalBidCPM: 10,
										ExtBid:         openrtb_ext.ExtBid{},
									},
								},
							},
						},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:              "pubmatic",
						BidderCode:             "pubmatic",
						PartnerSize:            "15x15",
						BidID:                  "bid-id-1",
						OrigBidID:              "bid-id-1",
						DealID:                 "-1",
						ServerSide:             1,
						OriginalCur:            "USD",
						Adformat:               models.Native,
						NetECPM:                10,
						GrossECPM:              12,
						FloorValue:             1,
						FloorRuleValue:         2,
						ADomain:                "sony.com",
						KGPV:                   "kgpv",
						KGPSV:                  "kgpsv",
						PriceBucket:            "10.00",
						MultiBidMultiFloorFlag: 1,
						InViewCountingFlag:     1,
					},
				},
			},
		},
		{
			name: "tracker absent, compute data",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:      "bid-id-1",
										ImpID:   "imp1",
										W:       15,
										H:       15,
										Price:   12,
										ADomain: []string{"http://sony.com"},
									},
								},
								Seat: "pubmatic",
							},
						},
					},
					Account: &config.Account{
						BidRounding: config.RoundingModeDown,
					},
				},
				rCtx: &models.RequestCtx{
					Trackers:         map[string]models.OWTracker{},
					PriceGranularity: &pg,
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										OriginalBidCPM: 12,
										ExtBid: openrtb_ext.ExtBid{
											Prebid: &openrtb_ext.ExtBidPrebid{
												Floors: &openrtb_ext.ExtBidPrebidFloors{
													FloorValue:     1,
													FloorRuleValue: 2,
												},
												Type: models.Native,
											},
										},
									},
								},
							},
							Bidders: map[string]models.PartnerData{
								"pubmatic": {
									PrebidBidderCode: "pubmatic",
									PartnerID:        1,
									KGP:              "kgp",
									KGPV:             "kgpv",
								},
							},
							Native: &openrtb2.Native{},
						},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:      "pubmatic",
						BidderCode:     "pubmatic",
						PartnerSize:    "15x15",
						BidID:          "bid-id-1",
						OrigBidID:      "bid-id-1",
						DealID:         "-1",
						ServerSide:     1,
						OriginalCur:    "USD",
						Adformat:       models.Native,
						NetECPM:        12,
						GrossECPM:      12,
						FloorValue:     1,
						FloorRuleValue: 2,
						ADomain:        "sony.com",
						KGPV:           "kgpv",
						KGPSV:          "kgpv",
						PriceBucket:    "12.00",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			partners := getPartnerRecordsByImp(tt.args.ao, tt.args.rCtx)
			assert.Equalf(t, partners, tt.partners, tt.name)
		})
	}
}
func TestGetPartnerRecordsByImpForDroppedBids(t *testing.T) {
	type args struct {
		ao   analytics.AuctionObject
		rCtx *models.RequestCtx
	}
	tests := []struct {
		name     string
		args     args
		partners map[string][]PartnerRecord
	}{
		{
			name: "all bids got dropped",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{},
				},
				rCtx: &models.RequestCtx{
					DroppedBids: map[string][]openrtb2.Bid{
						"pubmatic": {
							{
								ID:    "bid-id-1",
								ImpID: "imp1",
							},
						},
						"appnexus": {
							{
								ID:    "bid-id-2",
								ImpID: "imp1",
							},
						},
					},
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							Bidders: map[string]models.PartnerData{
								"pubmatic": {
									PartnerID:        1,
									PrebidBidderCode: "pubmatic",
								},
								"appnexus": {
									PartnerID:        2,
									PrebidBidderCode: "appnexus",
								},
							},
						},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:        "pubmatic",
						BidderCode:       "pubmatic",
						PartnerSize:      "0x0",
						BidID:            "bid-id-1",
						OrigBidID:        "bid-id-1",
						DealID:           "-1",
						ServerSide:       1,
						OriginalCur:      "USD",
						DefaultBidStatus: 1,
					},
					{
						PartnerID:        "appnexus",
						BidderCode:       "appnexus",
						PartnerSize:      "0x0",
						BidID:            "bid-id-2",
						OrigBidID:        "bid-id-2",
						DealID:           "-1",
						ServerSide:       1,
						OriginalCur:      "USD",
						DefaultBidStatus: 1,
					},
				},
			},
		},
		{
			name: "1 bid got dropped, 1 bid is present in seatbid",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "pubmatic",
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					DroppedBids: map[string][]openrtb2.Bid{
						"appnexus": {
							{
								ID:    "bid-id-2",
								ImpID: "imp1",
							},
						},
					},
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							Bidders: map[string]models.PartnerData{
								"pubmatic": {
									PartnerID:        1,
									PrebidBidderCode: "pubmatic",
								},
								"appnexus": {
									PartnerID:        2,
									PrebidBidderCode: "appnexus",
								},
							},
						},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:        "pubmatic",
						BidderCode:       "pubmatic",
						PartnerSize:      "0x0",
						BidID:            "bid-id-1",
						OrigBidID:        "bid-id-1",
						DealID:           "-1",
						ServerSide:       1,
						OriginalCur:      "USD",
						DefaultBidStatus: 1,
					},
					{
						PartnerID:        "appnexus",
						BidderCode:       "appnexus",
						PartnerSize:      "0x0",
						BidID:            "bid-id-2",
						OrigBidID:        "bid-id-2",
						DealID:           "-1",
						ServerSide:       1,
						OriginalCur:      "USD",
						DefaultBidStatus: 1,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			partners := getPartnerRecordsByImp(tt.args.ao, tt.args.rCtx)
			assert.Equal(t, len(tt.partners), len(partners), tt.name)
			for ind := range partners {
				// ignore order of elements in slice while comparison
				assert.ElementsMatch(t, partners[ind], tt.partners[ind], tt.name)
			}
		})
	}
}
func TestGetPartnerRecordsByImpForDefaultBids(t *testing.T) {
	type args struct {
		ao   analytics.AuctionObject
		rCtx *models.RequestCtx
	}
	tests := []struct {
		name     string
		args     args
		partners map[string][]PartnerRecord
	}{
		{
			name: "no default bid",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										Price: 10,
									},
								},
								Seat: "pubmatic",
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										ExtBid:         openrtb_ext.ExtBid{},
										OriginalBidCPM: 10,
									},
								},
							},
							TagID: "adunit_1234",
						},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:   "pubmatic",
						BidderCode:  "pubmatic",
						PartnerSize: "0x0",
						BidID:       "bid-id-1",
						OrigBidID:   "bid-id-1",
						DealID:      "-1",
						ServerSide:  1,
						OriginalCur: "USD",
						NetECPM:     10,
						GrossECPM:   10,
					},
				},
			},
		},
		{
			name: "partner timeout case, default bid present is seat-bid but absent in seat-non-bid",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										Price: 0,
									},
								},
								Seat: "pubmatic",
							},
						},
					},
					SeatNonBid: []openrtb_ext.SeatNonBid{
						{
							Seat: "appnexus",
							NonBid: []openrtb_ext.NonBid{
								{
									ImpId:      "imp1",
									StatusCode: int(exchange.ResponseRejectedBelowFloor),
									Ext: openrtb_ext.ExtNonBid{
										Prebid: openrtb_ext.ExtNonBidPrebid{
											Bid: openrtb_ext.ExtNonBidPrebidBid{
												ID: "bid-id-2",
											},
										},
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										ExtBid: openrtb_ext.ExtBid{},
										Nbr:    exchange.ErrorTimeout.Ptr(),
									},
								},
							},
						},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:        "appnexus",
						BidderCode:       "appnexus",
						PartnerSize:      "0x0",
						BidID:            "bid-id-2",
						OrigBidID:        "bid-id-2",
						DealID:           "-1",
						ServerSide:       1,
						OriginalCur:      "USD",
						Nbr:              exchange.ResponseRejectedBelowFloor.Ptr(),
						DefaultBidStatus: 1,
					},
					{
						PartnerID:            "pubmatic",
						BidderCode:           "pubmatic",
						PartnerSize:          "0x0",
						BidID:                "bid-id-1",
						OrigBidID:            "bid-id-1",
						DealID:               "-1",
						ServerSide:           1,
						OriginalCur:          "USD",
						NetECPM:              0,
						GrossECPM:            0,
						Nbr:                  exchange.ErrorTimeout.Ptr(),
						PostTimeoutBidStatus: 1,
						DefaultBidStatus:     1,
					},
				},
			},
		},
		{
			name: "floor rejected bid, default bid present in seat-bid and same bid is available in seat-non-bid",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										Price: 0,
									},
								},
								Seat: "pubmatic",
							},
						},
					},
					SeatNonBid: []openrtb_ext.SeatNonBid{
						{
							Seat: "pubmatic",
							NonBid: []openrtb_ext.NonBid{
								{
									ImpId:      "imp1",
									StatusCode: int(exchange.ResponseRejectedBelowFloor),
									Ext: openrtb_ext.ExtNonBid{
										Prebid: openrtb_ext.ExtNonBidPrebid{
											Bid: openrtb_ext.ExtNonBidPrebidBid{
												ID: "bid-id-1",
											},
										},
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										ExtBid: openrtb_ext.ExtBid{},
										Nbr:    exchange.ErrorGeneral.Ptr(),
									},
								},
							},
						},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:        "pubmatic",
						BidderCode:       "pubmatic",
						PartnerSize:      "0x0",
						BidID:            "bid-id-1",
						OrigBidID:        "bid-id-1",
						DealID:           "-1",
						ServerSide:       1,
						OriginalCur:      "USD",
						NetECPM:          0,
						GrossECPM:        0,
						Nbr:              exchange.ResponseRejectedBelowFloor.Ptr(),
						DefaultBidStatus: 1,
					},
				},
			},
		},
		{
			name: "slot not mapped, default bid present is seat-bid but absent in seat-non-bid",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										Price: 0,
									},
								},
								Seat: "pubmatic",
							},
						},
					},
					SeatNonBid: []openrtb_ext.SeatNonBid{},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										ExtBid: openrtb_ext.ExtBid{},
										Nbr:    exchange.ErrorGeneral.Ptr(),
									},
								},
							},
							NonMapped: map[string]struct{}{
								"pubmatic": {},
							},
						},
					},
				},
			},
			partners: map[string][]PartnerRecord{},
		},
		{
			name: "Multi_impression_request_slot_not_mapped_for_imp1_for_appnexus",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										Price: 10,
									},
								},
								Seat: "pubmatic",
							},
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp2",
										Price: 10,
									},
								},
								Seat: "pubmatic",
							},
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-2",
										ImpID: "imp2",
										Price: 20,
									},
								},
								Seat: "appnexus",
							},
						},
					},
					SeatNonBid: []openrtb_ext.SeatNonBid{},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidFloorCur: "USD",
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										ExtBid:         openrtb_ext.ExtBid{},
										OriginalBidCPM: 10,
										OriginalBidCur: "USD",
									},
								},
								"bid-id-2": {
									BidExt: models.BidExt{
										ExtBid: openrtb_ext.ExtBid{},
										Nbr:    exchange.ErrorGeneral.Ptr(),
									},
								},
							},
							NonMapped: map[string]struct{}{
								"appnexus": {},
							},
						},
						"imp2": {
							BidFloorCur: "USD",
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										ExtBid:         openrtb_ext.ExtBid{},
										OriginalBidCPM: 10,
										OriginalBidCur: "USD",
									},
								},
								"bid-id-2": {
									BidExt: models.BidExt{
										ExtBid:         openrtb_ext.ExtBid{},
										OriginalBidCPM: 20,
										OriginalBidCur: "USD",
									},
								},
							},
						},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						NetECPM:     10,
						GrossECPM:   10,
						OriginalCPM: 10,
						OriginalCur: "USD",
						PartnerID:   "pubmatic",
						BidderCode:  "pubmatic",
						PartnerSize: "0x0",
						BidID:       "bid-id-1",
						OrigBidID:   "bid-id-1",
						DealID:      "-1",
						ServerSide:  1,
					},
				},
				"imp2": {
					{
						NetECPM:     10,
						GrossECPM:   10,
						OriginalCPM: 10,
						OriginalCur: "USD",
						PartnerID:   "pubmatic",
						BidderCode:  "pubmatic",
						PartnerSize: "0x0",
						BidID:       "bid-id-1",
						OrigBidID:   "bid-id-1",
						DealID:      "-1",
						ServerSide:  1,
					},
					{
						NetECPM:     20,
						GrossECPM:   20,
						OriginalCPM: 20,
						OriginalCur: "USD",
						PartnerID:   "appnexus",
						BidderCode:  "appnexus",
						PartnerSize: "0x0",
						BidID:       "bid-id-2",
						OrigBidID:   "bid-id-2",
						DealID:      "-1",
						ServerSide:  1,
					},
				},
			},
		},
		{
			name: "slot_not_mapped_for_pubmatic",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										Price: 0,
									},
								},
								Seat: "pubmatic",
							},
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-2",
										ImpID: "imp1",
										Price: 10,
									},
								},
								Seat: "appnexus",
							},
						},
					},
					SeatNonBid: []openrtb_ext.SeatNonBid{},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidFloorCur: "USD",
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										ExtBid: openrtb_ext.ExtBid{},
									},
								},
								"bid-id-2": {
									BidExt: models.BidExt{
										ExtBid:         openrtb_ext.ExtBid{},
										OriginalBidCPM: 10,
										OriginalBidCur: "USD",
									},
								},
							},
							NonMapped: map[string]struct{}{
								"pubmatic": {},
							},
						},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						NetECPM:     10,
						GrossECPM:   10,
						OriginalCPM: 10,
						OriginalCur: "USD",
						PartnerID:   "appnexus",
						BidderCode:  "appnexus",
						PartnerSize: "0x0",
						BidID:       "bid-id-2",
						OrigBidID:   "bid-id-2",
						DealID:      "-1",
						ServerSide:  1,
					},
				},
			},
		},
		{
			name: "partner throttled, default bid present is seat-bid but absent in seat-non-bid",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										Price: 0,
									},
								},
								Seat: "pubmatic",
							},
						},
					},
					SeatNonBid: []openrtb_ext.SeatNonBid{},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										ExtBid: openrtb_ext.ExtBid{},
										Nbr:    exchange.ErrorGeneral.Ptr(),
									},
								},
							},
						},
					},
					AdapterThrottleMap: map[string]struct{}{
						"pubmatic": {},
					},
				},
			},
			partners: map[string][]PartnerRecord{},
		},
		{
			name: "sendAllBids_false_SeatBid_winner_plus_DefaultBids_from_rCtx_logged",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						Cur: "USD",
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-winner",
										ImpID: "imp1",
										Price: 10,
									},
								},
								Seat: "pubmatic",
							},
						},
					},
					SeatNonBid: []openrtb_ext.SeatNonBid{},
				},
				rCtx: &models.RequestCtx{
					SendAllBids: false,
					DefaultBids: map[string]map[string][]openrtb2.Bid{
						"imp1": {
							"appnexus": {
								{
									ID:    "default-uuid-1",
									ImpID: "imp1",
									Price: 0,
									W:     0,
									H:     0,
								},
							},
						},
					},
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							IsBanner: true,
							TagID:    "adunit_1",
							Bidders: map[string]models.PartnerData{
								"pubmatic": {
									PartnerID:        100,
									PrebidBidderCode: "pubmatic",
									KGP:              "pmkgp",
								},
								"appnexus": {
									PartnerID:        501,
									PrebidBidderCode: "appnexus",
									KGP:              "kgp1",
								},
							},
							BidCtx: map[string]models.BidCtx{
								"bid-id-winner": {
									BidExt: models.BidExt{
										ExtBid:         openrtb_ext.ExtBid{},
										OriginalBidCPM: 10,
									},
								},
								"default-uuid-1": {
									BidExt: models.BidExt{
										ExtBid: openrtb_ext.ExtBid{},
									},
								},
							},
						},
					},
					PartnerConfigMap: map[int]map[string]string{
						100: {models.REVSHARE: "0"},
						501: {models.REVSHARE: "0"},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:   "pubmatic",
						BidderCode:  "pubmatic",
						PartnerSize: "0x0",
						BidID:       "bid-id-winner",
						OrigBidID:   "bid-id-winner",
						DealID:      "-1",
						ServerSide:  1,
						OriginalCur: "USD",
						NetECPM:     10,
						GrossECPM:   10,
					},
					{
						PartnerID:        "appnexus",
						BidderCode:       "appnexus",
						PartnerSize:      "0x0",
						Adformat:         models.Banner,
						BidID:            "default-uuid-1",
						OrigBidID:        "default-uuid-1",
						DealID:           "-1",
						ServerSide:       1,
						OriginalCur:      "USD",
						NetECPM:          0,
						GrossECPM:        0,
						DefaultBidStatus: 1,
					},
				},
			},
		},
		{
			name: "sendAllBids_false_placeholder_imp2_only_deduped_against_DefaultBids_imp1_has_winner",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						Cur: "USD",
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-winner",
										ImpID: "imp1",
										Price: 10,
									},
								},
								Seat: "pubmatic",
							},
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "default-uuid-an",
										ImpID: "imp2",
										Price: 0,
										W:     0,
										H:     0,
									},
								},
								Seat: "appnexus",
							},
						},
					},
					SeatNonBid: []openrtb_ext.SeatNonBid{},
				},
				rCtx: &models.RequestCtx{
					SendAllBids: false,
					DefaultBids: map[string]map[string][]openrtb2.Bid{
						"imp2": {
							"appnexus": {
								{
									ID:    "default-uuid-an",
									ImpID: "imp2",
									Price: 0,
									W:     0,
									H:     0,
								},
							},
							"rubicon": {
								{
									ID:    "default-uuid-rb",
									ImpID: "imp2",
									Price: 0,
									W:     0,
									H:     0,
								},
							},
						},
					},
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							IsBanner: true,
							TagID:    "adunit_1",
							Bidders: map[string]models.PartnerData{
								"pubmatic": {
									PartnerID:        100,
									PrebidBidderCode: "pubmatic",
									KGP:              "pmkgp",
								},
							},
							BidCtx: map[string]models.BidCtx{
								"bid-id-winner": {
									BidExt: models.BidExt{
										ExtBid:         openrtb_ext.ExtBid{},
										OriginalBidCPM: 10,
									},
								},
							},
						},
						"imp2": {
							IsBanner: true,
							TagID:    "adunit_2",
							Bidders: map[string]models.PartnerData{
								"appnexus": {
									PartnerID:        501,
									PrebidBidderCode: "appnexus",
									KGP:              "kgp1",
								},
								"rubicon": {
									PartnerID:        502,
									PrebidBidderCode: "rubicon",
									KGP:              "kgp2",
								},
							},
							BidCtx: map[string]models.BidCtx{
								"default-uuid-an": {
									BidExt: models.BidExt{
										ExtBid: openrtb_ext.ExtBid{},
									},
								},
								"default-uuid-rb": {
									BidExt: models.BidExt{
										ExtBid: openrtb_ext.ExtBid{},
									},
								},
							},
						},
					},
					PartnerConfigMap: map[int]map[string]string{
						100: {models.REVSHARE: "0"},
						501: {models.REVSHARE: "0"},
						502: {models.REVSHARE: "0"},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:   "pubmatic",
						BidderCode:  "pubmatic",
						PartnerSize: "0x0",
						BidID:       "bid-id-winner",
						OrigBidID:   "bid-id-winner",
						DealID:      "-1",
						ServerSide:  1,
						OriginalCur: "USD",
						NetECPM:     10,
						GrossECPM:   10,
					},
				},
				"imp2": {
					{
						PartnerID:        "appnexus",
						BidderCode:       "appnexus",
						PartnerSize:      "0x0",
						Adformat:         models.Banner,
						BidID:            "default-uuid-an",
						OrigBidID:        "default-uuid-an",
						DealID:           "-1",
						ServerSide:       1,
						OriginalCur:      "USD",
						NetECPM:          0,
						GrossECPM:        0,
						DefaultBidStatus: 1,
					},
					{
						PartnerID:        "rubicon",
						BidderCode:       "rubicon",
						PartnerSize:      "0x0",
						Adformat:         models.Banner,
						BidID:            "default-uuid-rb",
						OrigBidID:        "default-uuid-rb",
						DealID:           "-1",
						ServerSide:       1,
						OriginalCur:      "USD",
						NetECPM:          0,
						GrossECPM:        0,
						DefaultBidStatus: 1,
					},
				},
			},
		},
		{
			name: "sendAllBids_false_DefaultBids_skipped_when_same_seat_imp_in_SeatNonBid",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						Cur: "USD",
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-winner",
										ImpID: "imp1",
										Price: 10,
									},
								},
								Seat: "pubmatic",
							},
						},
					},
					SeatNonBid: []openrtb_ext.SeatNonBid{
						{
							Seat: "appnexus",
							NonBid: []openrtb_ext.NonBid{
								{
									ImpId:      "imp1",
									StatusCode: int(exchange.ResponseRejectedBelowFloor),
									Ext: openrtb_ext.ExtNonBid{
										Prebid: openrtb_ext.ExtNonBidPrebid{
											Bid: openrtb_ext.ExtNonBidPrebidBid{
												ID: "nbid-1",
											},
										},
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					SendAllBids: false,
					DefaultBids: map[string]map[string][]openrtb2.Bid{
						"imp1": {
							"appnexus": {
								{
									ID:    "default-uuid-1",
									ImpID: "imp1",
									Price: 0,
									W:     0,
									H:     0,
								},
							},
						},
					},
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							IsBanner: true,
							TagID:    "adunit_1",
							Bidders: map[string]models.PartnerData{
								"pubmatic": {
									PartnerID:        100,
									PrebidBidderCode: "pubmatic",
									KGP:              "pmkgp",
								},
								"appnexus": {
									PartnerID:        501,
									PrebidBidderCode: "appnexus",
									KGP:              "kgp1",
								},
							},
							BidCtx: map[string]models.BidCtx{
								"bid-id-winner": {
									BidExt: models.BidExt{
										ExtBid:         openrtb_ext.ExtBid{},
										OriginalBidCPM: 10,
									},
								},
								"default-uuid-1": {
									BidExt: models.BidExt{
										ExtBid: openrtb_ext.ExtBid{},
									},
								},
							},
						},
					},
					PartnerConfigMap: map[int]map[string]string{
						100: {models.REVSHARE: "0"},
						501: {models.REVSHARE: "0"},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:   "pubmatic",
						BidderCode:  "pubmatic",
						PartnerSize: "0x0",
						BidID:       "bid-id-winner",
						OrigBidID:   "bid-id-winner",
						DealID:      "-1",
						ServerSide:  1,
						OriginalCur: "USD",
						NetECPM:     10,
						GrossECPM:   10,
					},
					{
						PartnerID:        "appnexus",
						BidderCode:       "appnexus",
						PartnerSize:      "0x0",
						Adformat:         models.Banner,
						BidID:            "nbid-1",
						OrigBidID:        "nbid-1",
						DealID:           "-1",
						ServerSide:       1,
						OriginalCur:      "USD",
						Nbr:              exchange.ResponseRejectedBelowFloor.Ptr(),
						DefaultBidStatus: 1,
					},
				},
			},
		},
		{
			name: "sendAllBids_true_SeatBid_has_winner_and_other_bid_DefaultBids_map_not_appended_by_logger",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						Cur: "USD",
						SeatBid: []openrtb2.SeatBid{
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-winner",
										ImpID: "imp1",
										Price: 10,
									},
								},
								Seat: "pubmatic",
							},
							{
								Bid: []openrtb2.Bid{
									{
										ID:    "default-uuid-1",
										ImpID: "imp1",
										Price: 0,
										W:     0,
										H:     0,
									},
								},
								Seat: "appnexus",
							},
						},
					},
					SeatNonBid: []openrtb_ext.SeatNonBid{},
				},
				rCtx: &models.RequestCtx{
					SendAllBids: true,
					DefaultBids: map[string]map[string][]openrtb2.Bid{
						"imp1": {
							"appnexus": {
								{
									ID:    "default-uuid-1",
									ImpID: "imp1",
									Price: 0,
									W:     0,
									H:     0,
								},
							},
						},
					},
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							IsBanner: true,
							TagID:    "adunit_1",
							Bidders: map[string]models.PartnerData{
								"pubmatic": {
									PartnerID:        100,
									PrebidBidderCode: "pubmatic",
									KGP:              "pmkgp",
								},
								"appnexus": {
									PartnerID:        501,
									PrebidBidderCode: "appnexus",
									KGP:              "kgp1",
								},
							},
							BidCtx: map[string]models.BidCtx{
								"bid-id-winner": {
									BidExt: models.BidExt{
										ExtBid:         openrtb_ext.ExtBid{},
										OriginalBidCPM: 10,
									},
								},
								"default-uuid-1": {
									BidExt: models.BidExt{
										ExtBid: openrtb_ext.ExtBid{},
									},
								},
							},
						},
					},
					PartnerConfigMap: map[int]map[string]string{
						100: {models.REVSHARE: "0"},
						501: {models.REVSHARE: "0"},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:   "pubmatic",
						BidderCode:  "pubmatic",
						PartnerSize: "0x0",
						BidID:       "bid-id-winner",
						OrigBidID:   "bid-id-winner",
						DealID:      "-1",
						ServerSide:  1,
						OriginalCur: "USD",
						NetECPM:     10,
						GrossECPM:   10,
					},
					{
						PartnerID:        "appnexus",
						BidderCode:       "appnexus",
						PartnerSize:      "0x0",
						Adformat:         models.Banner,
						BidID:            "default-uuid-1",
						OrigBidID:        "default-uuid-1",
						DealID:           "-1",
						ServerSide:       1,
						OriginalCur:      "USD",
						NetECPM:          0,
						GrossECPM:        0,
						DefaultBidStatus: 1,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			partners := getPartnerRecordsByImp(tt.args.ao, tt.args.rCtx)
			assert.Equal(t, len(tt.partners), len(partners), tt.name)
			for ind := range partners {
				// ignore order of elements in slice while comparison
				if !assert.ElementsMatch(t, partners[ind], tt.partners[ind], tt.name) {
					assert.Equal(t, partners[ind], tt.partners[ind], tt.name)
				}
			}
		})
	}
}
func TestGetPartnerRecordsByImpForSeatNonBid(t *testing.T) {
	pg, _ := openrtb_ext.NewPriceGranularityFromLegacyID("med")
	type args struct {
		ao   analytics.AuctionObject
		rCtx *models.RequestCtx
	}
	tests := []struct {
		name     string
		args     args
		partners map[string][]PartnerRecord
	}{
		{
			name: "empty seatnonbids, expect empty partnerRecord",
			args: args{
				ao: analytics.AuctionObject{
					Response:   &openrtb2.BidResponse{},
					SeatNonBid: []openrtb_ext.SeatNonBid{},
				},
				rCtx: &models.RequestCtx{},
			},
			partners: map[string][]PartnerRecord{},
		},
		{
			name: "ImpBidCtx is must to log partner-record in logger",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{},
					SeatNonBid: []openrtb_ext.SeatNonBid{
						{
							Seat: "pubmatic",
							NonBid: []openrtb_ext.NonBid{
								{
									ImpId:      "imp1",
									StatusCode: int(exchange.ResponseRejectedBelowDealFloor),
									Ext: openrtb_ext.ExtNonBid{
										Prebid: openrtb_ext.ExtNonBidPrebid{
											Bid: openrtb_ext.ExtNonBidPrebidBid{
												Price: 10,
											},
										},
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: make(map[string]models.ImpCtx),
				},
			},
			partners: map[string][]PartnerRecord{},
		},
		{
			name: "log rejected non-bid",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{},
					SeatNonBid: []openrtb_ext.SeatNonBid{
						{
							Seat: "appnexus",
							NonBid: []openrtb_ext.NonBid{
								{
									ImpId:      "imp1",
									StatusCode: int(exchange.ResponseRejectedBelowFloor),
									Ext: openrtb_ext.ExtNonBid{
										Prebid: openrtb_ext.ExtNonBidPrebid{
											Bid: openrtb_ext.ExtNonBidPrebidBid{
												Price:          10,
												ID:             "bid-id-1",
												W:              10,
												H:              50,
												OriginalBidCPM: 10,
											},
										},
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							Bidders: map[string]models.PartnerData{
								"appnexus": {
									PartnerID:        1,
									PrebidBidderCode: "appnexus",
									KGP:              "kgp",
									KGPV:             "kgpv",
								},
							},
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										ExtBid: openrtb_ext.ExtBid{},
										Nbr:    exchange.ResponseRejectedBelowFloor.Ptr(),
									},
								},
							},
							BidFloor:    10.5,
							BidFloorCur: "USD",
						},
					},
					PartnerConfigMap: map[int]map[string]string{
						1: {
							"rev_share": "0",
						},
					},
					WinningBids: make(models.WinningBids),
					Platform:    models.PLATFORM_APP,
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:      "appnexus",
						BidderCode:     "appnexus",
						KGPV:           "kgpv",
						KGPSV:          "kgpv",
						PartnerSize:    "10x50",
						GrossECPM:      10,
						NetECPM:        10,
						BidID:          "bid-id-1",
						OrigBidID:      "bid-id-1",
						DealID:         "-1",
						ServerSide:     1,
						OriginalCPM:    0,
						OriginalCur:    models.USD,
						FloorValue:     10.5,
						FloorRuleValue: 10.5,
						Nbr:            exchange.ResponseRejectedBelowFloor.Ptr(),
					},
				},
			},
		},
		{
			name: "log rejected non-bid having bidder_response_currency EUR and request_currency USD",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							Cur: []string{models.USD},
						},
					},
					Response: &openrtb2.BidResponse{Cur: models.USD},
					SeatNonBid: []openrtb_ext.SeatNonBid{
						{
							Seat: "appnexus",
							NonBid: []openrtb_ext.NonBid{
								{
									ImpId:      "imp1",
									StatusCode: int(nbr.LossBidLostInVastUnwrap),
									Ext: openrtb_ext.ExtNonBid{
										Prebid: openrtb_ext.ExtNonBidPrebid{
											Bid: openrtb_ext.ExtNonBidPrebidBid{
												Price:          10,
												ID:             "bid-id-1",
												W:              10,
												H:              50,
												OriginalBidCur: "EUR",
											},
										},
									},
								},
							},
						},
					},
					Account: &config.Account{BidRounding: config.RoundingModeDown},
				},
				rCtx: &models.RequestCtx{
					PriceGranularity: &pg,
					CurrencyConversion: func(from, to string, value float64) (float64, error) {
						if from == "USD" && to == "EUR" {
							return value * 1.2, nil
						}
						if from == "EUR" && to == "USD" {
							return value * 0.8, nil
						}
						return 0, nil
					},
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							Bidders: map[string]models.PartnerData{
								"appnexus": {
									PartnerID:        1,
									PrebidBidderCode: "appnexus",
									KGP:              "kgp",
									KGPV:             "kgpv",
								},
							},
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										ExtBid: openrtb_ext.ExtBid{},
										Nbr:    ptrutil.ToPtr(nbr.LossBidLostInVastUnwrap),
									},
								},
							},
							BidFloor:    10.5,
							BidFloorCur: "USD",
						},
					},
					PartnerConfigMap: map[int]map[string]string{
						1: {
							"rev_share": "0",
						},
					},
					WinningBids: make(models.WinningBids),
					Platform:    models.PLATFORM_APP,
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:      "appnexus",
						BidderCode:     "appnexus",
						KGPV:           "kgpv",
						KGPSV:          "kgpv",
						PartnerSize:    "10x50",
						GrossECPM:      8,
						NetECPM:        8,
						BidID:          "bid-id-1",
						OrigBidID:      "bid-id-1",
						DealID:         "-1",
						ServerSide:     1,
						OriginalCPM:    10,
						OriginalCur:    "EUR",
						FloorValue:     10.5,
						FloorRuleValue: 10.5,
						PriceBucket:    "8.00",
						Nbr:            ptrutil.ToPtr(nbr.LossBidLostInVastUnwrap),
					},
				},
			},
		},
		{
			name: "log rejected non-bid having bidder_response_currency EUR and request_currency USD and having 50% revshare",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							Cur: []string{models.USD},
						},
					},
					Response: &openrtb2.BidResponse{Cur: models.USD},
					SeatNonBid: []openrtb_ext.SeatNonBid{
						{
							Seat: "appnexus",
							NonBid: []openrtb_ext.NonBid{
								{
									ImpId:      "imp1",
									StatusCode: int(nbr.LossBidLostInVastUnwrap),
									Ext: openrtb_ext.ExtNonBid{
										Prebid: openrtb_ext.ExtNonBidPrebid{
											Bid: openrtb_ext.ExtNonBidPrebidBid{
												Price:          10,
												ID:             "bid-id-1",
												W:              10,
												H:              50,
												OriginalBidCur: "EUR",
											},
										},
									},
								},
							},
						},
					},
					Account: &config.Account{BidRounding: config.RoundingModeDown},
				},
				rCtx: &models.RequestCtx{
					PriceGranularity: &pg,
					CurrencyConversion: func(from, to string, value float64) (float64, error) {
						if from == "USD" && to == "EUR" {
							return value * 1.2, nil
						}
						if from == "EUR" && to == "USD" {
							return value * 0.8, nil
						}
						return 0, nil
					},
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							Bidders: map[string]models.PartnerData{
								"appnexus": {
									PartnerID:        1,
									PrebidBidderCode: "appnexus",
									KGP:              "kgp",
									KGPV:             "kgpv",
								},
							},
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										ExtBid: openrtb_ext.ExtBid{},
										Nbr:    ptrutil.ToPtr(nbr.LossBidLostInVastUnwrap),
									},
								},
							},
							BidFloor:    10.5,
							BidFloorCur: "USD",
						},
					},
					PartnerConfigMap: map[int]map[string]string{
						1: {
							"rev_share": "50",
						},
					},
					WinningBids: make(models.WinningBids),
					Platform:    models.PLATFORM_APP,
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:      "appnexus",
						BidderCode:     "appnexus",
						KGPV:           "kgpv",
						KGPSV:          "kgpv",
						PartnerSize:    "10x50",
						GrossECPM:      8,
						NetECPM:        4,
						PriceBucket:    "4.00",
						BidID:          "bid-id-1",
						OrigBidID:      "bid-id-1",
						DealID:         "-1",
						ServerSide:     1,
						OriginalCPM:    10,
						OriginalCur:    "EUR",
						FloorValue:     10.5,
						FloorRuleValue: 10.5,
						Nbr:            ptrutil.ToPtr(nbr.LossBidLostInVastUnwrap),
					},
				},
			},
		},
		{
			name: "log rejected non-bid having response_currency USD and request_currency EUR and having 50% revshare",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							Cur: []string{"EUR"},
						},
					},
					Response: &openrtb2.BidResponse{Cur: "EUR"},
					SeatNonBid: []openrtb_ext.SeatNonBid{
						{
							Seat: "appnexus",
							NonBid: []openrtb_ext.NonBid{
								{
									ImpId:      "imp1",
									StatusCode: int(nbr.LossBidLostInVastUnwrap),
									Ext: openrtb_ext.ExtNonBid{
										Prebid: openrtb_ext.ExtNonBidPrebid{
											Bid: openrtb_ext.ExtNonBidPrebidBid{
												Price:          10,
												ID:             "bid-id-1",
												W:              10,
												H:              50,
												OriginalBidCur: models.USD,
											},
										},
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					CurrencyConversion: func(from, to string, value float64) (float64, error) {
						if from == "USD" && to == "EUR" {
							return value * 1.2, nil
						}
						if from == "EUR" && to == "USD" {
							return value * 0.8, nil
						}
						return value, nil
					},
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							Bidders: map[string]models.PartnerData{
								"appnexus": {
									PartnerID:        1,
									PrebidBidderCode: "appnexus",
									KGP:              "kgp",
									KGPV:             "kgpv",
								},
							},
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										ExtBid: openrtb_ext.ExtBid{},
										Nbr:    ptrutil.ToPtr(nbr.LossBidLostInVastUnwrap),
									},
								},
							},
							BidFloor:    10.5,
							BidFloorCur: "USD",
						},
					},
					PartnerConfigMap: map[int]map[string]string{
						1: {
							"rev_share": "50",
						},
					},
					WinningBids: make(models.WinningBids),
					Platform:    models.PLATFORM_APP,
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:      "appnexus",
						BidderCode:     "appnexus",
						KGPV:           "kgpv",
						KGPSV:          "kgpv",
						PartnerSize:    "10x50",
						GrossECPM:      10,
						NetECPM:        5,
						BidID:          "bid-id-1",
						OrigBidID:      "bid-id-1",
						DealID:         "-1",
						ServerSide:     1,
						OriginalCPM:    10,
						OriginalCur:    "USD",
						FloorValue:     10.5,
						FloorRuleValue: 10.5,
						Nbr:            ptrutil.ToPtr(nbr.LossBidLostInVastUnwrap),
					},
				},
			},
		},
		{
			name: "log from seat-non-bid and seat-bid for Endpoint webs2s: here default/proxy bids are present in seat non-bid",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "pubmatic",
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-2",
										Price: 30,
										W:     10,
										H:     50,
										ImpID: "imp1",
										Ext:   json.RawMessage(`{"origbidcpm":30}`),
									},
								},
							},
						},
					},
					SeatNonBid: []openrtb_ext.SeatNonBid{
						{
							Seat: "appnexus",
							NonBid: []openrtb_ext.NonBid{
								{
									ImpId:      "imp1",
									StatusCode: int(exchange.ResponseRejectedBelowFloor),
									Ext: openrtb_ext.ExtNonBid{
										Prebid: openrtb_ext.ExtNonBidPrebid{
											Bid: openrtb_ext.ExtNonBidPrebidBid{
												Price:          10,
												ID:             "bid-id-1",
												W:              10,
												H:              50,
												OriginalBidCPM: 10,
											},
										},
									},
								},
							},
						},
					},
					Account: &config.Account{BidRounding: config.RoundingModeDown},
				},
				rCtx: &models.RequestCtx{
					PriceGranularity: &pg,
					CurrencyConversion: func(from, to string, value float64) (float64, error) {
						if from == "USD" && to == "EUR" {
							return value * 1.2, nil
						}
						if from == "EUR" && to == "USD" {
							return value * 0.8, nil
						}
						return 0, nil
					},
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							Bidders: map[string]models.PartnerData{
								"appnexus": {
									PartnerID:        1,
									PrebidBidderCode: "appnexus",
									KGP:              "kgp",
									KGPV:             "kgpv",
								},
								"pubmatic": {
									PartnerID:        2,
									PrebidBidderCode: "pubmatic",
									KGP:              "kgp",
									KGPV:             "kgpv",
								},
							},
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										ExtBid: openrtb_ext.ExtBid{},
										Nbr:    exchange.ResponseRejectedBelowFloor.Ptr(),
									},
								},
							},
							BidFloor:    10.5,
							BidFloorCur: "USD",
						},
					},
					PartnerConfigMap: map[int]map[string]string{
						1: {
							"rev_share": "0",
						},
					},
					WinningBids: make(models.WinningBids),
					Platform:    models.PLATFORM_APP,
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:      "appnexus",
						BidderCode:     "appnexus",
						KGPV:           "kgpv",
						KGPSV:          "kgpv",
						PartnerSize:    "10x50",
						GrossECPM:      10,
						NetECPM:        10,
						BidID:          "bid-id-1",
						OrigBidID:      "bid-id-1",
						DealID:         "-1",
						ServerSide:     1,
						OriginalCPM:    0,
						OriginalCur:    models.USD,
						FloorValue:     10.5,
						FloorRuleValue: 10.5,
						PriceBucket:    "10.00",
						Nbr:            exchange.ResponseRejectedBelowFloor.Ptr(),
					},
					{
						PartnerID:      "pubmatic",
						BidderCode:     "pubmatic",
						KGPV:           "kgpv",
						KGPSV:          "kgpv",
						PartnerSize:    "10x50",
						GrossECPM:      30,
						NetECPM:        30,
						BidID:          "bid-id-2",
						OrigBidID:      "bid-id-2",
						DealID:         "-1",
						ServerSide:     1,
						OriginalCPM:    0,
						OriginalCur:    models.USD,
						FloorValue:     10.5,
						FloorRuleValue: 10.5,
						PriceBucket:    "20.00",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			partners := getPartnerRecordsByImp(tt.args.ao, tt.args.rCtx)
			for ind := range partners {
				// ignore order of elements in slice while comparison
				assert.ElementsMatch(t, partners[ind], tt.partners[ind], tt.name)
			}
		})
	}
}
func TestGetPartnerRecordsByImpForSeatNonBidForFloors(t *testing.T) {
	type args struct {
		ao   analytics.AuctionObject
		rCtx *models.RequestCtx
	}
	tests := []struct {
		name     string
		args     args
		partners map[string][]PartnerRecord
	}{
		{
			name: "bid.ext.prebid.floors has high priority than imp.bidfloor",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{},
					SeatNonBid: []openrtb_ext.SeatNonBid{
						{
							Seat: "appnexus",
							NonBid: []openrtb_ext.NonBid{
								{
									ImpId:      "imp1",
									StatusCode: int(exchange.ResponseRejectedBelowFloor),
									Ext: openrtb_ext.ExtNonBid{
										Prebid: openrtb_ext.ExtNonBidPrebid{
											Bid: openrtb_ext.ExtNonBidPrebidBid{
												Price: 10,
												ID:    "bid-id-1",
												Floors: &openrtb_ext.ExtBidPrebidFloors{
													FloorRule:      "*|*|ebay.com",
													FloorRuleValue: 1,
													FloorValue:     1,
													FloorCurrency:  models.USD,
												},
												OriginalBidCPM: 10,
											},
										},
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidFloor:    10.5,
							BidFloorCur: "USD",
						},
					},
					PartnerConfigMap: map[int]map[string]string{
						1: {},
					},
					WinningBids: make(models.WinningBids),
					Platform:    models.PLATFORM_APP,
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:      "appnexus",
						BidderCode:     "appnexus",
						PartnerSize:    "0x0",
						BidID:          "bid-id-1",
						OrigBidID:      "bid-id-1",
						DealID:         "-1",
						GrossECPM:      10,
						NetECPM:        10,
						ServerSide:     1,
						OriginalCPM:    0,
						OriginalCur:    models.USD,
						FloorValue:     1,
						FloorRuleValue: 1,
						Nbr:            exchange.ResponseRejectedBelowFloor.Ptr(),
					},
				},
			},
		},
		{
			name: "bid.ext.prebid.floors can have 0 value",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{},
					SeatNonBid: []openrtb_ext.SeatNonBid{
						{
							Seat: "appnexus",
							NonBid: []openrtb_ext.NonBid{
								{
									ImpId:      "imp1",
									StatusCode: int(exchange.ResponseRejectedBelowFloor),
									Ext: openrtb_ext.ExtNonBid{
										Prebid: openrtb_ext.ExtNonBidPrebid{
											Bid: openrtb_ext.ExtNonBidPrebidBid{
												Price: 10,
												ID:    "bid-id-1",
												Floors: &openrtb_ext.ExtBidPrebidFloors{
													FloorRule:      "*|*|ebay.com",
													FloorRuleValue: 0,
													FloorValue:     0,
													FloorCurrency:  models.USD,
												},
												OriginalBidCPM: 10,
											},
										},
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidFloor:    10.5,
							BidFloorCur: "USD",
						},
					},
					PartnerConfigMap: map[int]map[string]string{
						1: {},
					},
					WinningBids: make(models.WinningBids),
					Platform:    models.PLATFORM_APP,
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:      "appnexus",
						BidderCode:     "appnexus",
						PartnerSize:    "0x0",
						BidID:          "bid-id-1",
						OrigBidID:      "bid-id-1",
						DealID:         "-1",
						GrossECPM:      10,
						NetECPM:        10,
						ServerSide:     1,
						OriginalCPM:    0,
						OriginalCur:    models.USD,
						FloorValue:     0,
						FloorRuleValue: 0,
						Nbr:            exchange.ResponseRejectedBelowFloor.Ptr(),
					},
				},
			},
		},
		{
			name: "bid.ext.prebid.floors.floorRuleValue is 0 then set it to bid.ext.prebid.floors.floorRule",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{},
					SeatNonBid: []openrtb_ext.SeatNonBid{
						{
							Seat: "appnexus",
							NonBid: []openrtb_ext.NonBid{
								{
									ImpId:      "imp1",
									StatusCode: int(exchange.ResponseRejectedBelowFloor),
									Ext: openrtb_ext.ExtNonBid{
										Prebid: openrtb_ext.ExtNonBidPrebid{
											Bid: openrtb_ext.ExtNonBidPrebidBid{
												Price: 10,
												ID:    "bid-id-1",
												Floors: &openrtb_ext.ExtBidPrebidFloors{
													FloorRule:      "*|*|ebay.com",
													FloorRuleValue: 0,
													FloorValue:     10,
													FloorCurrency:  models.USD,
												},
												OriginalBidCPM: 10,
											},
										},
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidFloor:    10.5,
							BidFloorCur: "USD",
						},
					},
					PartnerConfigMap: map[int]map[string]string{
						1: {},
					},
					WinningBids: make(models.WinningBids),
					Platform:    models.PLATFORM_APP,
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:      "appnexus",
						BidderCode:     "appnexus",
						PartnerSize:    "0x0",
						BidID:          "bid-id-1",
						OrigBidID:      "bid-id-1",
						DealID:         "-1",
						GrossECPM:      10,
						NetECPM:        10,
						ServerSide:     1,
						OriginalCPM:    0,
						OriginalCur:    models.USD,
						FloorValue:     10,
						FloorRuleValue: 10,
						Nbr:            exchange.ResponseRejectedBelowFloor.Ptr(),
					},
				},
			},
		},
		{
			name: "bid.ext.prebid.floors not set, fallback to imp.bidfloor",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{},
					SeatNonBid: []openrtb_ext.SeatNonBid{
						{
							Seat: "appnexus",
							NonBid: []openrtb_ext.NonBid{
								{
									ImpId:      "imp1",
									StatusCode: int(exchange.ResponseRejectedBelowFloor),
									Ext: openrtb_ext.ExtNonBid{
										Prebid: openrtb_ext.ExtNonBidPrebid{
											Bid: openrtb_ext.ExtNonBidPrebidBid{
												Price:          10,
												ID:             "bid-id-1",
												OriginalBidCPM: 10,
											},
										},
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidFloor:    10.567,
							BidFloorCur: "USD",
						},
					},
					PartnerConfigMap: map[int]map[string]string{
						1: {},
					},
					WinningBids: make(models.WinningBids),
					Platform:    models.PLATFORM_APP,
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:      "appnexus",
						BidderCode:     "appnexus",
						PartnerSize:    "0x0",
						BidID:          "bid-id-1",
						OrigBidID:      "bid-id-1",
						DealID:         "-1",
						GrossECPM:      10,
						NetECPM:        10,
						ServerSide:     1,
						OriginalCPM:    0,
						OriginalCur:    models.USD,
						FloorValue:     10.57,
						FloorRuleValue: 10.57,
						Nbr:            exchange.ResponseRejectedBelowFloor.Ptr(),
					},
				},
			},
		},
		{
			name: "currency conversion when floor value is set to imp.bidfloor",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{},
					SeatNonBid: []openrtb_ext.SeatNonBid{
						{
							Seat: "appnexus",
							NonBid: []openrtb_ext.NonBid{
								{
									ImpId:      "imp1",
									StatusCode: int(exchange.ResponseRejectedBelowFloor),
									Ext: openrtb_ext.ExtNonBid{
										Prebid: openrtb_ext.ExtNonBidPrebid{
											Bid: openrtb_ext.ExtNonBidPrebidBid{
												Price:          10,
												ID:             "bid-id-1",
												OriginalBidCPM: 10,
											},
										},
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					CurrencyConversion: func(from, to string, value float64) (float64, error) {
						return 1000, nil
					},
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidFloor:    10.567,
							BidFloorCur: "JPY",
						},
					},
					PartnerConfigMap: map[int]map[string]string{
						1: {},
					},
					WinningBids: make(models.WinningBids),
					Platform:    models.PLATFORM_APP,
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:      "appnexus",
						BidderCode:     "appnexus",
						PartnerSize:    "0x0",
						BidID:          "bid-id-1",
						OrigBidID:      "bid-id-1",
						DealID:         "-1",
						GrossECPM:      10,
						NetECPM:        10,
						ServerSide:     1,
						OriginalCPM:    0,
						OriginalCur:    models.USD,
						FloorValue:     1000,
						FloorRuleValue: 1000,
						Nbr:            exchange.ResponseRejectedBelowFloor.Ptr(),
					},
				},
			},
		},
		{
			name: "currency conversion when floor value is set from bid.ext.prebid.floors",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{},
					SeatNonBid: []openrtb_ext.SeatNonBid{
						{
							Seat: "appnexus",
							NonBid: []openrtb_ext.NonBid{
								{
									ImpId:      "imp1",
									StatusCode: int(exchange.ResponseRejectedBelowFloor),
									Ext: openrtb_ext.ExtNonBid{
										Prebid: openrtb_ext.ExtNonBidPrebid{
											Bid: openrtb_ext.ExtNonBidPrebidBid{
												Price: 10,
												ID:    "bid-id-1",
												Floors: &openrtb_ext.ExtBidPrebidFloors{
													FloorRule:      "*|*|ebay.com",
													FloorRuleValue: 1,
													FloorValue:     1,
													FloorCurrency:  "JPY",
												},
												OriginalBidCPM: 10,
											},
										},
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					CurrencyConversion: func(from, to string, value float64) (float64, error) {
						return 0.12, nil
					},
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidFloor:    10.567,
							BidFloorCur: "JPY",
						},
					},
					PartnerConfigMap: map[int]map[string]string{
						1: {},
					},
					WinningBids: make(models.WinningBids),
					Platform:    models.PLATFORM_APP,
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:      "appnexus",
						BidderCode:     "appnexus",
						PartnerSize:    "0x0",
						BidID:          "bid-id-1",
						OrigBidID:      "bid-id-1",
						DealID:         "-1",
						GrossECPM:      10,
						NetECPM:        10,
						ServerSide:     1,
						OriginalCPM:    0,
						OriginalCur:    models.USD,
						FloorValue:     0.12,
						FloorRuleValue: 0.12,
						Nbr:            exchange.ResponseRejectedBelowFloor.Ptr(),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			partners := getPartnerRecordsByImp(tt.args.ao, tt.args.rCtx)
			assert.Equal(t, tt.partners, partners, tt.name)
		})
	}
}
func TestGetPartnerRecordsByImpForReserveredBidders(t *testing.T) {
	type args struct {
		ao   analytics.AuctionObject
		rCtx *models.RequestCtx
	}
	tests := []struct {
		name     string
		args     args
		partners map[string][]PartnerRecord
	}{
		{
			name: "ignore prebid_ctv bidder",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "prebid_ctv",
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{},
				},
			},
			partners: map[string][]PartnerRecord{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			partners := getPartnerRecordsByImp(tt.args.ao, tt.args.rCtx)
			assert.Equal(t, tt.partners, partners, tt.name)
		})
	}
}
func TestGetPartnerRecordsByImpForPostTimeoutBidStatus(t *testing.T) {
	type args struct {
		ao   analytics.AuctionObject
		rCtx *models.RequestCtx
	}
	tests := []struct {
		name     string
		args     args
		partners map[string][]PartnerRecord
	}{
		{
			name: "update 't' when Partner Timed out",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "appnexus",
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										Nbr: exchange.ErrorTimeout.Ptr(),
									},
								},
							},
						},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:            "appnexus",
						BidderCode:           "appnexus",
						PartnerSize:          "0x0",
						BidID:                "bid-id-1",
						OrigBidID:            "bid-id-1",
						DealID:               "-1",
						ServerSide:           1,
						OriginalCur:          models.USD,
						PostTimeoutBidStatus: 1,
						Nbr:                  exchange.ErrorTimeout.Ptr(),
						DefaultBidStatus:     1,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			partners := getPartnerRecordsByImp(tt.args.ao, tt.args.rCtx)
			assert.Equal(t, tt.partners, partners, tt.name)
		})
	}
}
func TestGetPartnerRecordsByImpForBidIDCollisions(t *testing.T) {
	type args struct {
		ao   analytics.AuctionObject
		rCtx *models.RequestCtx
	}
	tests := []struct {
		name     string
		args     args
		partners map[string][]PartnerRecord
	}{
		{
			name: "valid bid, impBidCtx bidID is in bidID::uuid format",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "appnexus",
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										Price: 10,
										Ext:   json.RawMessage(`{"prebid":{"bidid":"uuid"}}`),
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"bid-id-1::uuid": {
									BidExt: models.BidExt{
										ExtBid: openrtb_ext.ExtBid{
											Prebid: &openrtb_ext.ExtBidPrebid{
												DealPriority:      1,
												DealTierSatisfied: true,
												BidId:             "uuid",
											},
										},
										OriginalBidCPM: 10,
									},
								},
							},
						},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:    "appnexus",
						BidderCode:   "appnexus",
						PartnerSize:  "0x0",
						BidID:        "uuid",
						OrigBidID:    "bid-id-1",
						DealID:       "-1",
						ServerSide:   1,
						OriginalCur:  models.USD,
						NetECPM:      10,
						GrossECPM:    10,
						DealPriority: 1,
					},
				},
			},
		},
		{
			name: "valid bid, but json unmarshal fails",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							Cur: []string{},
						},
					},
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "appnexus",
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										Price: 10,
										Ext:   json.RawMessage(`{"prebid":{`),
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"bid-id-1::uuid": {
									BidExt: models.BidExt{
										ExtBid: openrtb_ext.ExtBid{
											Prebid: &openrtb_ext.ExtBidPrebid{
												DealPriority:      1,
												DealTierSatisfied: true,
												BidId:             "uuid",
											},
										},
										OriginalBidCPM: 10,
									},
								},
							},
						},
					},
					CurrencyConversion: func(from, to string, value float64) (float64, error) { return 10, nil },
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:    "appnexus",
						BidderCode:   "appnexus",
						PartnerSize:  "0x0",
						BidID:        "bid-id-1",
						OrigBidID:    "bid-id-1",
						DealID:       "-1",
						ServerSide:   1,
						OriginalCur:  models.USD,
						NetECPM:      10,
						GrossECPM:    10,
						OriginalCPM:  10,
						DealPriority: 0,
					},
				},
			},
		},
		{
			name: "dropped bid, impBidCtx bidID is in bidID::uuid format",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "appnexus",
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										Price: 10,
										Ext:   json.RawMessage(`{"prebid":{"bidid":"uuid"}}`),
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"bid-id-1::uuid": {
									BidExt: models.BidExt{
										ExtBid: openrtb_ext.ExtBid{
											Prebid: &openrtb_ext.ExtBidPrebid{
												DealPriority:      1,
												DealTierSatisfied: true,
											},
										},
										OriginalBidCPM: 10,
										Nbr:            nbr.LossBidLostToHigherBid.Ptr(),
									},
								},
							},
						},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:    "appnexus",
						BidderCode:   "appnexus",
						PartnerSize:  "0x0",
						BidID:        "bid-id-1",
						OrigBidID:    "bid-id-1",
						DealID:       "-1",
						ServerSide:   1,
						OriginalCur:  models.USD,
						NetECPM:      10,
						GrossECPM:    10,
						DealPriority: 1,
						Nbr:          nbr.LossBidLostToHigherBid.Ptr(),
					},
				},
			},
		},
		{
			name: "default bid, impBidCtx bidID is in uuid format",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "appnexus",
								Bid: []openrtb2.Bid{
									{
										ID:    "uuid",
										ImpID: "imp1",
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"uuid": {
									BidExt: models.BidExt{
										ExtBid: openrtb_ext.ExtBid{},
										Nbr:    exchange.ErrorTimeout.Ptr(),
									},
								},
							},
						},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:            "appnexus",
						BidderCode:           "appnexus",
						PartnerSize:          "0x0",
						BidID:                "uuid",
						OrigBidID:            "uuid",
						DealID:               "-1",
						ServerSide:           1,
						OriginalCur:          models.USD,
						Nbr:                  exchange.ErrorTimeout.Ptr(),
						PostTimeoutBidStatus: 1,
						DefaultBidStatus:     1,
					},
				},
			},
		},
		{
			name: "non bid, no bidCtx in impBidCtx",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{},
					SeatNonBid: []openrtb_ext.SeatNonBid{
						{
							Seat: "appnexus",
							NonBid: []openrtb_ext.NonBid{
								{
									ImpId:      "imp1",
									StatusCode: int(nbr.LossBidLostToDealBid),
									Ext: openrtb_ext.ExtNonBid{
										Prebid: openrtb_ext.ExtNonBidPrebid{
											Bid: openrtb_ext.ExtNonBidPrebidBid{
												Price:          10,
												ID:             "bid-id-1",
												BidId:          "uuid",
												OriginalBidCPM: 10,
											},
										},
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:   "appnexus",
						BidderCode:  "appnexus",
						PartnerSize: "0x0",
						BidID:       "uuid",
						OrigBidID:   "bid-id-1",
						DealID:      "-1",
						ServerSide:  1,
						NetECPM:     10,
						GrossECPM:   10,
						OriginalCur: models.USD,
						Nbr:         nbr.LossBidLostToDealBid.Ptr(),
					},
				},
			},
		},
		{
			name: "winning bid contains bidID in bidID::uuid format",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "appnexus",
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										Price: 10,
										Ext:   json.RawMessage(`{"prebid":{"bidid":"uuid"}}`),
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"bid-id-1::uuid": {
									BidExt: models.BidExt{
										ExtBid: openrtb_ext.ExtBid{
											Prebid: &openrtb_ext.ExtBidPrebid{
												DealPriority:      1,
												DealTierSatisfied: true,
												BidId:             "uuid",
											},
										},
										OriginalBidCPM: 10,
									},
								},
							},
						},
					},
					WinningBids: models.WinningBids{
						"imp1": []*models.OwBid{
							{
								ID: "bid-id-1::uuid",
							},
						},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:       "appnexus",
						BidderCode:      "appnexus",
						PartnerSize:     "0x0",
						BidID:           "uuid",
						OrigBidID:       "bid-id-1",
						DealID:          "-1",
						ServerSide:      1,
						OriginalCur:     models.USD,
						NetECPM:         10,
						GrossECPM:       10,
						DealPriority:    1,
						WinningBidStaus: 1,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			partners := getPartnerRecordsByImp(tt.args.ao, tt.args.rCtx)
			assert.Equal(t, tt.partners, partners, tt.name)
		})
	}
}
func TestGetPartnerRecordsByImpForBidExtFailure(t *testing.T) {
	type args struct {
		ao   analytics.AuctionObject
		rCtx *models.RequestCtx
	}
	tests := []struct {
		name     string
		args     args
		partners map[string][]PartnerRecord
	}{
		{
			name: "valid bid, but bid.ext is empty",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{},
					},
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "appnexus",
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										Price: 10,
										Ext:   json.RawMessage(`{}`),
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"bid-id-1::uuid": {
									BidExt: models.BidExt{
										ExtBid: openrtb_ext.ExtBid{
											Prebid: &openrtb_ext.ExtBidPrebid{
												DealPriority:      1,
												DealTierSatisfied: true,
												BidId:             "uuid",
											},
										},
									},
								},
							},
						},
					},
					CurrencyConversion: func(from, to string, value float64) (float64, error) { return 10, nil },
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:    "appnexus",
						BidderCode:   "appnexus",
						PartnerSize:  "0x0",
						BidID:        "bid-id-1",
						OrigBidID:    "bid-id-1",
						DealID:       "-1",
						ServerSide:   1,
						OriginalCur:  models.USD,
						NetECPM:      10,
						GrossECPM:    10,
						OriginalCPM:  10,
						DealPriority: 0,
					},
				},
			},
		},
		{
			name: "dropped bid, bidExt unmarshal fails",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{},
					},
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "appnexus",
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										Price: 10,
										Ext:   json.RawMessage(`{"prebid":{`),
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"bid-id-1::uuid": {
									BidExt: models.BidExt{
										ExtBid: openrtb_ext.ExtBid{
											Prebid: &openrtb_ext.ExtBidPrebid{
												DealPriority:      1,
												DealTierSatisfied: true,
											},
										},
										Nbr: nbr.LossBidLostToHigherBid.Ptr(),
									},
								},
							},
						},
					},
					CurrencyConversion: func(from, to string, value float64) (float64, error) { return 10, nil },
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:    "appnexus",
						BidderCode:   "appnexus",
						PartnerSize:  "0x0",
						BidID:        "bid-id-1",
						OrigBidID:    "bid-id-1",
						DealID:       "-1",
						ServerSide:   1,
						OriginalCur:  models.USD,
						NetECPM:      10,
						GrossECPM:    10,
						OriginalCPM:  10,
						DealPriority: 0,
						Nbr:          nil,
					},
				},
			},
		},
		{
			name: "default bid, bidExt unmarshal fails",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "appnexus",
								Bid: []openrtb2.Bid{
									{
										ID:    "uuid",
										ImpID: "imp1",
										Ext:   json.RawMessage(`{{{`),
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"uuid": {
									BidExt: models.BidExt{
										ExtBid: openrtb_ext.ExtBid{},
										Nbr:    exchange.ErrorTimeout.Ptr(),
									},
								},
							},
						},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:            "appnexus",
						BidderCode:           "appnexus",
						PartnerSize:          "0x0",
						BidID:                "uuid",
						OrigBidID:            "uuid",
						DealID:               "-1",
						ServerSide:           1,
						OriginalCur:          models.USD,
						Nbr:                  exchange.ErrorTimeout.Ptr(),
						PostTimeoutBidStatus: 1,
						DefaultBidStatus:     1,
					},
				},
			},
		},
		{
			name: "non bid, bidExt empty",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{},
					SeatNonBid: []openrtb_ext.SeatNonBid{
						{
							Seat: "appnexus",
							NonBid: []openrtb_ext.NonBid{
								{
									ImpId:      "imp1",
									StatusCode: int(exchange.ResponseRejectedBelowDealFloor),
									Ext:        openrtb_ext.ExtNonBid{},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:   "appnexus",
						BidderCode:  "appnexus",
						PartnerSize: "0x0",
						BidID:       "",
						OrigBidID:   "",
						DealID:      "-1",
						ServerSide:  1,
						NetECPM:     0,
						GrossECPM:   0,
						OriginalCur: models.USD,
						Nbr: func() *openrtb3.NoBidReason {
							a := exchange.ResponseRejectedBelowDealFloor
							return &a
						}(),
						DefaultBidStatus: 1,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			partners := getPartnerRecordsByImp(tt.args.ao, tt.args.rCtx)
			assert.Equal(t, tt.partners, partners, tt.name)
		})
	}
}
func TestGetPartnerRecordsByImpForBidExtPrebidObject(t *testing.T) {
	type args struct {
		ao   analytics.AuctionObject
		rCtx *models.RequestCtx
	}
	tests := []struct {
		name     string
		args     args
		partners map[string][]PartnerRecord
	}{
		{
			name: "log metadata object",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "appnexus",
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										ExtBid: openrtb_ext.ExtBid{
											Prebid: &openrtb_ext.ExtBidPrebid{
												Meta: &openrtb_ext.ExtBidPrebidMeta{
													NetworkID: 100,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:   "appnexus",
						BidderCode:  "appnexus",
						PartnerSize: "0x0",
						BidID:       "bid-id-1",
						OrigBidID:   "bid-id-1",
						DealID:      "-1",
						ServerSide:  1,
						OriginalCur: models.USD,
						MetaData: &MetaData{
							NetworkID: 100,
						},
						DefaultBidStatus: 1,
					},
				},
			},
		},
		{
			name: "dealPriority is 1 but DealTierSatisfied is false",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "appnexus",
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										ExtBid: openrtb_ext.ExtBid{
											Prebid: &openrtb_ext.ExtBidPrebid{
												DealTierSatisfied: false,
												DealPriority:      1,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:        "appnexus",
						BidderCode:       "appnexus",
						PartnerSize:      "0x0",
						BidID:            "bid-id-1",
						OrigBidID:        "bid-id-1",
						DealID:           "-1",
						ServerSide:       1,
						OriginalCur:      models.USD,
						DefaultBidStatus: 1,
					},
				},
			},
		},
		{
			name: "dealPriority is 1 and DealTierSatisfied is true",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "appnexus",
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										ExtBid: openrtb_ext.ExtBid{
											Prebid: &openrtb_ext.ExtBidPrebid{
												DealTierSatisfied: true,
												DealPriority:      1,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:        "appnexus",
						BidderCode:       "appnexus",
						PartnerSize:      "0x0",
						BidID:            "bid-id-1",
						OrigBidID:        "bid-id-1",
						DealID:           "-1",
						ServerSide:       1,
						OriginalCur:      models.USD,
						DealPriority:     1,
						DefaultBidStatus: 1,
					},
				},
			},
		},
		{
			name: "dealPriority is 0 and DealTierSatisfied is true",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "appnexus",
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										ExtBid: openrtb_ext.ExtBid{
											Prebid: &openrtb_ext.ExtBidPrebid{
												DealTierSatisfied: true,
												DealPriority:      0,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:        "appnexus",
						BidderCode:       "appnexus",
						PartnerSize:      "0x0",
						BidID:            "bid-id-1",
						OrigBidID:        "bid-id-1",
						DealID:           "-1",
						ServerSide:       1,
						OriginalCur:      models.USD,
						DealPriority:     0,
						DefaultBidStatus: 1,
					},
				},
			},
		},
		{
			name: "bidExt.Prebid.Video.Duration is 0 ",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "appnexus",
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										ExtBid: openrtb_ext.ExtBid{
											Prebid: &openrtb_ext.ExtBidPrebid{
												Video: &openrtb_ext.ExtBidPrebidVideo{
													Duration: 0,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:        "appnexus",
						BidderCode:       "appnexus",
						PartnerSize:      "0x0",
						BidID:            "bid-id-1",
						OrigBidID:        "bid-id-1",
						DealID:           "-1",
						ServerSide:       1,
						OriginalCur:      models.USD,
						DefaultBidStatus: 1,
					},
				},
			},
		},
		{
			name: "bidExt.Prebid.Video.Duration is valid, log AdDuration",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "appnexus",
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										ExtBid: openrtb_ext.ExtBid{
											Prebid: &openrtb_ext.ExtBidPrebid{
												Video: &openrtb_ext.ExtBidPrebidVideo{
													Duration: 10,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:        "appnexus",
						BidderCode:       "appnexus",
						PartnerSize:      "0x0",
						BidID:            "bid-id-1",
						OrigBidID:        "bid-id-1",
						DealID:           "-1",
						ServerSide:       1,
						OriginalCur:      models.USD,
						AdDuration:       ptrutil.ToPtr(10),
						DefaultBidStatus: 1,
					},
				},
			},
		},
		{
			name: "override bidid by bidExt.Prebid.bidID",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "appnexus",
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										ExtBid: openrtb_ext.ExtBid{
											Prebid: &openrtb_ext.ExtBidPrebid{
												BidId: "prebid-bid-id-1",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:        "appnexus",
						BidderCode:       "appnexus",
						PartnerSize:      "0x0",
						BidID:            "prebid-bid-id-1",
						OrigBidID:        "bid-id-1",
						DealID:           "-1",
						ServerSide:       1,
						OriginalCur:      models.USD,
						DefaultBidStatus: 1,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			partners := getPartnerRecordsByImp(tt.args.ao, tt.args.rCtx)
			assert.Equal(t, tt.partners, partners, tt.name)
		})
	}
}
func TestGetPartnerRecordsByImpForRevShareAndBidCPM(t *testing.T) {
	type args struct {
		ao   analytics.AuctionObject
		rCtx *models.RequestCtx
	}
	tests := []struct {
		name     string
		args     args
		partners map[string][]PartnerRecord
	}{
		{
			name: "origbidcpmusd not present",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "pubmatic",
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										Price: 1.55,
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										OriginalBidCPM: 1.55,
										OriginalBidCur: "USD",
									},
								},
							},
						},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						NetECPM:     1.55,
						GrossECPM:   1.55,
						OriginalCPM: 1.55,
						OriginalCur: "USD",
						PartnerID:   "pubmatic",
						BidderCode:  "pubmatic",
						PartnerSize: "0x0",
						BidID:       "bid-id-1",
						OrigBidID:   "bid-id-1",
						DealID:      "-1",
						ServerSide:  1,
					},
				},
			},
		},
		{
			name: "origbidcpmusd not present and revshare present",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "pubmatic",
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										Price: 100,
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					PartnerConfigMap: map[int]map[string]string{
						1: {
							models.REVSHARE: "10",
						},
					},
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										OriginalBidCPM: 100,
										OriginalBidCur: "USD",
									},
								},
							},
							Bidders: map[string]models.PartnerData{
								"pubmatic": {
									PartnerID:        1,
									PrebidBidderCode: "pubmatic",
								},
							},
						},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						NetECPM:     100,
						GrossECPM:   111.11,
						OriginalCPM: 100,
						OriginalCur: "USD",
						PartnerID:   "pubmatic",
						BidderCode:  "pubmatic",
						PartnerSize: "0x0",
						BidID:       "bid-id-1",
						OrigBidID:   "bid-id-1",
						DealID:      "-1",
						ServerSide:  1,
					},
				},
			},
		},
		{
			name: "origbidcpmusd is present",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "pubmatic",
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										Price: 1.55,
									},
								},
							},
						},
						Cur: "INR",
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										OriginalBidCPM:    125.76829,
										OriginalBidCur:    "INR",
										OriginalBidCPMUSD: 1.76829,
									},
								},
							},
						},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						NetECPM:     1.77,
						GrossECPM:   1.77,
						OriginalCPM: 125.77,
						OriginalCur: "INR",
						PartnerID:   "pubmatic",
						BidderCode:  "pubmatic",
						PartnerSize: "0x0",
						BidID:       "bid-id-1",
						OrigBidID:   "bid-id-1",
						DealID:      "-1",
						ServerSide:  1,
					},
				},
			},
		},
		{
			name: "origbidcpmusd not present for non-USD bids",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "pubmatic",
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										Price: 125.16829,
									},
								},
							},
						},
						Cur: "INR",
					},
				},
				rCtx: &models.RequestCtx{
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										OriginalBidCPM: 125.16829,
										OriginalBidCur: "INR",
									},
									EG: 125.16829,
									EN: 125.16829,
								},
							},
						},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						GrossECPM:   125.17,
						NetECPM:     125.17,
						OriginalCPM: 125.17,
						OriginalCur: "INR",
						PartnerID:   "pubmatic",
						BidderCode:  "pubmatic",
						PartnerSize: "0x0",
						BidID:       "bid-id-1",
						OrigBidID:   "bid-id-1",
						DealID:      "-1",
						ServerSide:  1,
					},
				},
			},
		},
		{
			name: "origbidcpmusd is present, revshare is present",
			args: args{
				ao: analytics.AuctionObject{
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "pubmatic",
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp1",
										Price: 100,
									},
								},
							},
						},
						Cur: "INR",
					},
				},
				rCtx: &models.RequestCtx{
					PartnerConfigMap: map[int]map[string]string{
						1: {
							models.REVSHARE: "10",
						},
					},
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							BidCtx: map[string]models.BidCtx{
								"bid-id-1": {
									BidExt: models.BidExt{
										OriginalBidCPM:    200,
										OriginalBidCur:    "INR",
										OriginalBidCPMUSD: 100,
									},
								},
							},
							Bidders: map[string]models.PartnerData{
								"pubmatic": {
									PrebidBidderCode: "pubmatic",
									PartnerID:        1,
								},
							},
						},
					},
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						NetECPM:     100,
						GrossECPM:   111.11,
						OriginalCPM: 200,
						OriginalCur: "INR",
						PartnerID:   "pubmatic",
						BidderCode:  "pubmatic",
						PartnerSize: "0x0",
						BidID:       "bid-id-1",
						OrigBidID:   "bid-id-1",
						DealID:      "-1",
						ServerSide:  1,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			partners := getPartnerRecordsByImp(tt.args.ao, tt.args.rCtx)
			assert.Equal(t, tt.partners, partners, tt.name)
		})
	}
}
func TestGetPartnerRecordsByImpForMarketPlaceBidders(t *testing.T) {
	type args struct {
		ao   analytics.AuctionObject
		rCtx *models.RequestCtx
	}
	tests := []struct {
		name     string
		args     args
		partners map[string][]PartnerRecord
	}{
		{
			name: "overwrite marketplace bid details",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{},
					},
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "appnexus",
								Bid: []openrtb2.Bid{
									{ID: "bid-id-1", ImpID: "imp1", Price: 1},
								},
							},
							{
								Seat: "pubmatic",
								Bid: []openrtb2.Bid{
									{ID: "bid-id-2", ImpID: "imp1", Price: 2},
								},
							},
							{
								Seat: "groupm",
								Bid: []openrtb2.Bid{
									{ID: "bid-id-3", ImpID: "imp1", Price: 3},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					MarketPlaceBidders: map[string]struct{}{
						"groupm": {},
					},
					ImpBidCtx: map[string]models.ImpCtx{
						"imp1": {
							Bidders: map[string]models.PartnerData{
								"appnexus": {
									KGP:              "apnx_kgp",
									KGPV:             "apnx_kgpv",
									PrebidBidderCode: "appnexus",
								},
								"pubmatic": {
									KGP:              "pubm_kgp",
									KGPV:             "pubm_kgpv",
									PrebidBidderCode: "pubmatic",
								},
								"groupm": {
									KGP:              "gm_kgp",
									KGPV:             "gm_kgpv",
									PrebidBidderCode: "groupm",
								},
							},
						},
					},
					CurrencyConversion: func(from, to string, value float64) (float64, error) { return value, nil },
				},
			},
			partners: map[string][]PartnerRecord{
				"imp1": {
					{
						PartnerID:   "appnexus",
						BidderCode:  "appnexus",
						PartnerSize: "0x0",
						BidID:       "bid-id-1",
						OrigBidID:   "bid-id-1",
						DealID:      "-1",
						ServerSide:  1,
						OriginalCur: models.USD,
						GrossECPM:   1,
						NetECPM:     1,
						OriginalCPM: 1,
						KGPV:        "apnx_kgpv",
						KGPSV:       "apnx_kgpv",
					},
					{
						PartnerID:   "pubmatic",
						BidderCode:  "pubmatic",
						PartnerSize: "0x0",
						BidID:       "bid-id-2",
						OrigBidID:   "bid-id-2",
						DealID:      "-1",
						ServerSide:  1,
						OriginalCur: models.USD,
						GrossECPM:   2,
						NetECPM:     2,
						OriginalCPM: 2,
						KGPV:        "pubm_kgpv",
						KGPSV:       "pubm_kgpv",
					},
					{
						PartnerID:   "pubmatic",
						BidderCode:  "groupm",
						PartnerSize: "0x0",
						BidID:       "bid-id-3",
						OrigBidID:   "bid-id-3",
						DealID:      "-1",
						ServerSide:  1,
						OriginalCur: models.USD,
						GrossECPM:   3,
						NetECPM:     3,
						OriginalCPM: 3,
						KGPV:        "pubm_kgpv",
						KGPSV:       "pubm_kgpv",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			partners := getPartnerRecordsByImp(tt.args.ao, tt.args.rCtx)
			assert.Equal(t, len(tt.partners), len(partners), tt.name)
			for ind := range partners {
				// ignore order of elements in slice while comparison
				assert.ElementsMatch(t, partners[ind], tt.partners[ind], tt.name)
			}
		})
	}
}
func TestGetLogAuctionObjectAsURL(t *testing.T) {

	cfg := ow.cfg
	defer func() {
		ow.cfg = cfg
	}()

	ow.cfg.Endpoint = "http://10.172.141.11/wl"
	ow.cfg.PublicEndpoint = "http://t.pubmatic.com/wl"

	type args struct {
		ao                  analytics.AuctionObject
		rCtx                *models.RequestCtx
		logInfo, forRespExt bool
	}
	type want struct {
		logger string
		header http.Header
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "logger_disabled",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{},
					},
					Response: &openrtb2.BidResponse{},
				},
				rCtx: &models.RequestCtx{
					Endpoint:       models.EndpointV25,
					LoggerDisabled: true,
					PubID:          5890,
				},
				logInfo:    true,
				forRespExt: true,
			},
			want: want{
				logger: "",
				header: nil,
			},
		},
		{
			name: "do not prepare owlogger if pubid is missing",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{},
					},
					Response: &openrtb2.BidResponse{},
				},
				rCtx: &models.RequestCtx{
					Endpoint: models.EndpointV25,
				},
				logInfo:    true,
				forRespExt: true,
			},
			want: want{
				logger: "",
				header: nil,
			},
		},
		{
			name: "do not prepare owlogger if bidrequest is nil",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: nil,
					},
					Response: &openrtb2.BidResponse{},
				},
				rCtx: &models.RequestCtx{
					Endpoint: models.EndpointV25,
				},
				logInfo:    true,
				forRespExt: true,
			},
			want: want{
				logger: "",
				header: nil,
			},
		},
		{
			name: "do not prepare owlogger if bidrequestwrapper is nil",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: nil,
				},
				rCtx: &models.RequestCtx{
					Endpoint: models.EndpointV25,
				},
				logInfo:    true,
				forRespExt: true,
			},
			want: want{
				logger: "",
				header: nil,
			},
		},
		{
			name: "log integration type",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{},
					},
					Response: &openrtb2.BidResponse{},
				},
				rCtx: &models.RequestCtx{
					Endpoint: models.EndpointV25,
					PubID:    5890,
				},
				logInfo:    true,
				forRespExt: true,
			},
			want: want{
				logger: `http://t.pubmatic.com/wl?json={"pubid":5890,"pid":"0","pdvid":"0","sl":1,"dvc":{},"ft":0,"it":"sdk","geo":{}}&pubid=5890`,
				header: http.Header{
					models.USER_AGENT_HEADER: []string{""},
					models.IP_HEADER:         []string{""},
				},
			},
		},
		{
			name: "log consent string",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							User: &openrtb2.User{
								Consent: "any-random-consent-string",
							},
						},
					},
					Response: &openrtb2.BidResponse{},
				},
				rCtx: &models.RequestCtx{
					PubID: 5890,
				},
				logInfo:    true,
				forRespExt: true,
			},
			want: want{
				logger: `http://t.pubmatic.com/wl?json={"pubid":5890,"pid":"0","pdvid":"0","cns":"any-random-consent-string","sl":1,"dvc":{},"ft":0,"geo":{}}&pubid=5890`,
				header: http.Header{
					models.USER_AGENT_HEADER: []string{""},
					models.IP_HEADER:         []string{""},
				},
			},
		},
		{
			name: "log gdpr flag",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							Regs: &openrtb2.Regs{
								GDPR: openrtb2.Int8Ptr(1),
							},
						},
					},
					Response: &openrtb2.BidResponse{},
				},
				rCtx: &models.RequestCtx{
					PubID: 5890,
				},
				logInfo:    true,
				forRespExt: true,
			},
			want: want{
				logger: `http://t.pubmatic.com/wl?json={"pubid":5890,"pid":"0","pdvid":"0","gdpr":1,"sl":1,"dvc":{},"ft":0,"geo":{}}&pubid=5890`,
				header: http.Header{
					models.USER_AGENT_HEADER: []string{""},
					models.IP_HEADER:         []string{""},
				},
			},
		},
		{
			name: "log device platform",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{},
					},
					Response: &openrtb2.BidResponse{},
				},
				rCtx: &models.RequestCtx{
					PubID: 5890,
					DeviceCtx: models.DeviceCtx{
						Platform: models.DevicePlatformMobileAppAndroid,
					},
				},
				logInfo:    true,
				forRespExt: true,
			},
			want: want{
				logger: `http://t.pubmatic.com/wl?json={"pubid":5890,"pid":"0","pdvid":"0","sl":1,"dvc":{"plt":5},"ft":0,"geo":{}}&pubid=5890`,
				header: http.Header{
					models.USER_AGENT_HEADER: []string{""},
					models.IP_HEADER:         []string{""},
				},
			},
		},
		{
			name: "log device IFA Type",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{},
					},
					Response: &openrtb2.BidResponse{},
				},
				rCtx: &models.RequestCtx{
					PubID: 5890,
					DeviceCtx: models.DeviceCtx{
						Platform:  models.DevicePlatformMobileAppAndroid,
						IFATypeID: ptrutil.ToPtr(8),
					},
				},
				logInfo:    true,
				forRespExt: true,
			},
			want: want{
				logger: `http://t.pubmatic.com/wl?json={"pubid":5890,"pid":"0","pdvid":"0","sl":1,"dvc":{"plt":5,"ifty":8},"ft":0,"geo":{}}&pubid=5890`,
				header: http.Header{
					models.USER_AGENT_HEADER: []string{""},
					models.IP_HEADER:         []string{""},
				},
			},
		},
		{
			name: "log_device.ext.atts",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{},
					},
					Response: &openrtb2.BidResponse{},
				},
				rCtx: &models.RequestCtx{
					PubID: 5890,
					DeviceCtx: models.DeviceCtx{
						Ext: func() *models.ExtDevice {
							extDevice := models.ExtDevice{}
							extDevice.UnmarshalJSON([]byte(`{"atts":1}`))
							return &extDevice
						}(),
					},
				},
				logInfo:    true,
				forRespExt: true,
			},
			want: want{
				logger: ow.cfg.PublicEndpoint + `?json={"pubid":5890,"pid":"0","pdvid":"0","sl":1,"dvc":{"atts":1},"ft":0,"geo":{}}&pubid=5890`,
				header: http.Header{
					models.USER_AGENT_HEADER: []string{""},
					models.IP_HEADER:         []string{""},
				},
			},
		},
		{
			name: "log content from site object",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							Site: &openrtb2.Site{
								Content: &openrtb2.Content{
									ID:    "1",
									Title: "Game of thrones",
									Cat:   []string{"IAB-1"},
								},
							},
						},
					},
					Response: &openrtb2.BidResponse{},
				},
				rCtx: &models.RequestCtx{
					PubID: 5890,
				},
				logInfo:    true,
				forRespExt: true,
			},
			want: want{
				logger: `http://t.pubmatic.com/wl?json={"pubid":5890,"pid":"0","pdvid":"0","sl":1,"dvc":{},"ct":{"id":"1","ttl":"Game of thrones","cat":["IAB-1"]},"ft":0,"geo":{}}&pubid=5890`,
				header: http.Header{
					models.USER_AGENT_HEADER: []string{""},
					models.IP_HEADER:         []string{""},
				},
			},
		},
		{
			name: "log content from app object",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							App: &openrtb2.App{
								Content: &openrtb2.Content{
									ID:    "1",
									Title: "Game of thrones",
								},
							},
						},
					},
					Response: &openrtb2.BidResponse{},
				},
				rCtx: &models.RequestCtx{
					PubID: 5890,
				},
				logInfo:    true,
				forRespExt: true,
			},
			want: want{
				logger: `http://t.pubmatic.com/wl?json={"pubid":5890,"pid":"0","pdvid":"0","sl":1,"dvc":{},"ct":{"id":"1","ttl":"Game of thrones"},"ft":0,"geo":{}}&pubid=5890`,
				header: http.Header{
					models.USER_AGENT_HEADER: []string{""},
					models.IP_HEADER:         []string{""},
				},
			},
		},
		{
			name: "log UA and IP in header",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{},
					},
					Response: &openrtb2.BidResponse{},
				},
				rCtx: &models.RequestCtx{
					DeviceCtx:     models.DeviceCtx{UA: "mozilla", IP: "10.10.10.10"},
					KADUSERCookie: &http.Cookie{Name: "uids", Value: "eidsabcd"},
					PubID:         5890,
				},
				logInfo:    true,
				forRespExt: true,
			},
			want: want{
				logger: `http://t.pubmatic.com/wl?json={"pubid":5890,"pid":"0","pdvid":"0","sl":1,"dvc":{},"ft":0,"geo":{}}&pubid=5890`,
				header: http.Header{
					models.USER_AGENT_HEADER: []string{"mozilla"},
					models.IP_HEADER:         []string{"10.10.10.10"},
				},
			},
		},
		{
			name: "loginfo is false",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{},
					},
					Response: &openrtb2.BidResponse{},
				},
				rCtx: &models.RequestCtx{
					PubID: 5890,
				},
				logInfo:    false,
				forRespExt: true,
			},
			want: want{
				logger: ow.cfg.Endpoint + `?json={"pubid":5890,"pid":"0","pdvid":"0","sl":1,"dvc":{},"ft":0,"geo":{}}&pubid=5890`,
				header: http.Header{
					models.USER_AGENT_HEADER: []string{""},
					models.IP_HEADER:         []string{""},
				},
			},
		},
		{
			name: "responseExt.Prebid is nil so floor details not set",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{},
					},
					Response: &openrtb2.BidResponse{
						Ext: json.RawMessage("{}"),
					},
				},
				rCtx: &models.RequestCtx{
					PubID: 5890,
				},
				logInfo:    true,
				forRespExt: true,
			},
			want: want{
				logger: ow.cfg.PublicEndpoint + `?json={"pubid":5890,"pid":"0","pdvid":"0","sl":1,"dvc":{},"ft":0,"geo":{}}&pubid=5890`,
				header: http.Header{
					models.USER_AGENT_HEADER: []string{""},
					models.IP_HEADER:         []string{""},
				},
			},
		},
		{
			name: "set floor details",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{},
					},
					Response: &openrtb2.BidResponse{},
				},
				rCtx: &models.RequestCtx{
					PubID: 5890,
					Trackers: map[string]models.OWTracker{
						"any-bid-id": {
							Tracker: models.Tracker{
								LoggerData: models.LoggerData{
									FloorProvider: "provider-1",
								},
							},
						},
					},
				},
				logInfo:    true,
				forRespExt: true,
			},
			want: want{
				logger: ow.cfg.PublicEndpoint + `?json={"pubid":5890,"pid":"0","pdvid":"0","sl":1,"dvc":{},"ft":0,"fp":"provider-1","geo":{}}&pubid=5890`,
				header: http.Header{
					models.USER_AGENT_HEADER: []string{""},
					models.IP_HEADER:         []string{""},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, header := GetLogAuctionObjectAsURL(tt.args.ao, tt.args.rCtx, tt.args.logInfo, tt.args.forRespExt)
			logger, _ = url.QueryUnescape(logger)
			assert.Equal(t, tt.want.logger, logger, tt.name)
			assert.Equal(t, tt.want.header, header, tt.name)
		})
	}
}
func TestGetLogAuctionObjectAsURLForFloorType(t *testing.T) {
	cfg := ow.cfg
	defer func() {
		ow.cfg = cfg
	}()

	ow.cfg.Endpoint = "http://10.172.141.11/wl"
	ow.cfg.PublicEndpoint = "http://t.pubmatic.com/wl"

	type args struct {
		ao                  analytics.AuctionObject
		rCtx                *models.RequestCtx
		logInfo, forRespExt bool
	}
	type want struct {
		logger string
		header http.Header
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "Floor type should be soft when prebid is nil",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{},
					},
					Response: &openrtb2.BidResponse{},
				},
				rCtx: &models.RequestCtx{
					PubID: 5890,
					NewReqExt: &models.RequestExt{
						ExtRequest: openrtb_ext.ExtRequest{},
					},
				},
				logInfo:    true,
				forRespExt: true,
			},
			want: want{
				logger: `http://t.pubmatic.com/wl?json={"pubid":5890,"pid":"0","pdvid":"0","sl":1,"dvc":{},"ft":0,"geo":{}}&pubid=5890`,
				header: http.Header{
					models.USER_AGENT_HEADER: []string{""},
					models.IP_HEADER:         []string{""},
				},
			},
		},
		{
			name: "Floor type should be soft when prebid.floors is nil",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{},
					},
					Response: &openrtb2.BidResponse{},
				},
				rCtx: &models.RequestCtx{
					PubID: 5890,
					NewReqExt: &models.RequestExt{
						ExtRequest: openrtb_ext.ExtRequest{
							Prebid: openrtb_ext.ExtRequestPrebid{},
						},
					},
				},
				logInfo:    true,
				forRespExt: true,
			},
			want: want{
				logger: `http://t.pubmatic.com/wl?json={"pubid":5890,"pid":"0","pdvid":"0","sl":1,"dvc":{},"ft":0,"geo":{}}&pubid=5890`,
				header: http.Header{
					models.USER_AGENT_HEADER: []string{""},
					models.IP_HEADER:         []string{""},
				},
			},
		},
		{
			name: "Floor type should be soft when prebid.floors is disabled",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{},
					},
					Response: &openrtb2.BidResponse{},
				},
				rCtx: &models.RequestCtx{
					PubID: 5890,
					NewReqExt: &models.RequestExt{
						ExtRequest: openrtb_ext.ExtRequest{
							Prebid: openrtb_ext.ExtRequestPrebid{
								Floors: &openrtb_ext.PriceFloorRules{
									Enabled: ptrutil.ToPtr(false),
									Enforcement: &openrtb_ext.PriceFloorEnforcement{
										EnforcePBS: ptrutil.ToPtr(true),
									},
								},
							},
						},
					},
				},
				logInfo:    true,
				forRespExt: true,
			},
			want: want{
				logger: `http://t.pubmatic.com/wl?json={"pubid":5890,"pid":"0","pdvid":"0","sl":1,"dvc":{},"ft":0,"geo":{}}&pubid=5890`,
				header: http.Header{
					models.USER_AGENT_HEADER: []string{""},
					models.IP_HEADER:         []string{""},
				},
			},
		},
		{
			name: "Floor type should be soft when prebid.floors.enforcement is nil",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{},
					},
					Response: &openrtb2.BidResponse{},
				},
				rCtx: &models.RequestCtx{
					PubID: 5890,
					NewReqExt: &models.RequestExt{
						ExtRequest: openrtb_ext.ExtRequest{
							Prebid: openrtb_ext.ExtRequestPrebid{
								Floors: &openrtb_ext.PriceFloorRules{
									Enabled: ptrutil.ToPtr(true),
								},
							},
						},
					},
				},
				logInfo:    true,
				forRespExt: true,
			},
			want: want{
				logger: `http://t.pubmatic.com/wl?json={"pubid":5890,"pid":"0","pdvid":"0","sl":1,"dvc":{},"ft":0,"geo":{}}&pubid=5890`,
				header: http.Header{
					models.USER_AGENT_HEADER: []string{""},
					models.IP_HEADER:         []string{""},
				},
			},
		},
		{
			name: "Floor type should be soft when prebid.floors.enforcement.enforcepbs is false",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{},
					},
					Response: &openrtb2.BidResponse{},
				},
				rCtx: &models.RequestCtx{
					NewReqExt: &models.RequestExt{
						ExtRequest: openrtb_ext.ExtRequest{
							Prebid: openrtb_ext.ExtRequestPrebid{
								Floors: &openrtb_ext.PriceFloorRules{
									Enabled: ptrutil.ToPtr(true),
									Enforcement: &openrtb_ext.PriceFloorEnforcement{
										EnforcePBS: ptrutil.ToPtr(false),
									},
								},
							},
						},
					},
					PubID: 5890,
				},
				logInfo:    true,
				forRespExt: true,
			},
			want: want{
				logger: `http://t.pubmatic.com/wl?json={"pubid":5890,"pid":"0","pdvid":"0","sl":1,"dvc":{},"ft":0,"geo":{}}&pubid=5890`,
				header: http.Header{
					models.USER_AGENT_HEADER: []string{""},
					models.IP_HEADER:         []string{""},
				},
			},
		},
		{
			name: "Floor type should be hard when prebid.floors.enforcement.enforcepbs is true",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{},
					},
					Response: &openrtb2.BidResponse{},
				},
				rCtx: &models.RequestCtx{
					NewReqExt: &models.RequestExt{
						ExtRequest: openrtb_ext.ExtRequest{
							Prebid: openrtb_ext.ExtRequestPrebid{
								Floors: &openrtb_ext.PriceFloorRules{
									Enabled: ptrutil.ToPtr(true),
									Enforcement: &openrtb_ext.PriceFloorEnforcement{
										EnforcePBS: ptrutil.ToPtr(true),
									},
								},
							},
						},
					},
					PubID: 5890,
				},
				logInfo:    true,
				forRespExt: true,
			},
			want: want{
				logger: `http://t.pubmatic.com/wl?json={"pubid":5890,"pid":"0","pdvid":"0","sl":1,"dvc":{},"ft":1,"geo":{}}&pubid=5890`,
				header: http.Header{
					models.USER_AGENT_HEADER: []string{""},
					models.IP_HEADER:         []string{""},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, header := GetLogAuctionObjectAsURL(tt.args.ao, tt.args.rCtx, tt.args.logInfo, tt.args.forRespExt)
			logger, _ = url.PathUnescape(logger)
			assert.Equal(t, tt.want.logger, logger, tt.name)
			assert.Equal(t, tt.want.header, header, tt.name)
		})
	}
}
func TestGetLogAuctionObjectAsURLForFloorDetailsAndCDS(t *testing.T) {
	cfg := ow.cfg
	uuidFunc := GetUUID
	defer func() {
		ow.cfg = cfg
		GetUUID = uuidFunc
	}()

	GetUUID = func() string { return "uuid" }
	ow.cfg.Endpoint = "http://10.172.141.11/wl"
	ow.cfg.PublicEndpoint = "http://t.pubmatic.com/wl"

	type args struct {
		ao                  analytics.AuctionObject
		rCtx                *models.RequestCtx
		logInfo, forRespExt bool
	}
	type want struct {
		logger string
		header http.Header
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "set floor details from tracker when slots are absent",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{},
					},
					Response: &openrtb2.BidResponse{},
				},
				rCtx: &models.RequestCtx{
					PubID: 5890,
					NewReqExt: &models.RequestExt{
						ExtRequest: openrtb_ext.ExtRequest{},
					},
					Trackers: map[string]models.OWTracker{
						"any-bid": {
							Tracker: models.Tracker{
								FloorSkippedFlag:  ptrutil.ToPtr(1),
								FloorModelVersion: "model-version",
								FloorSource:       ptrutil.ToPtr(2),
								FloorType:         0,
								LoggerData: models.LoggerData{
									FloorProvider:    "provider",
									FloorFetchStatus: ptrutil.ToPtr(3),
								},
								CustomDimensions: "traffic=media;age=23",
							},
						},
					},
				},
				logInfo:    true,
				forRespExt: true,
			},
			want: want{
				logger: `http://t.pubmatic.com/wl?json={"pubid":5890,"pid":"0","pdvid":"0","sl":1,"dvc":{},"fmv":"model-version","fsrc":2,"ft":0,"ffs":3,"fp":"provider","cds":"traffic=media;age=23","geo":{},"fskp":1}&pubid=5890`,
				header: http.Header{
					models.USER_AGENT_HEADER: []string{""},
					models.IP_HEADER:         []string{""},
				},
			},
		},
		{
			name: "set floor details from tracker when slots are present",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							Imp: []openrtb2.Imp{
								{
									ID: "imp-1",
								},
							},
						},
					},
					Response: &openrtb2.BidResponse{},
				},
				rCtx: &models.RequestCtx{
					PubID: 5890,
					NewReqExt: &models.RequestExt{
						ExtRequest: openrtb_ext.ExtRequest{},
					},
					ImpBidCtx: map[string]models.ImpCtx{
						"imp-1": {
							AdUnitName: "au",
							SlotName:   "sn",
						},
					},
					Trackers: map[string]models.OWTracker{
						"any-bid": {
							Tracker: models.Tracker{
								FloorSkippedFlag:  ptrutil.ToPtr(1),
								FloorModelVersion: "model-version",
								FloorSource:       ptrutil.ToPtr(2),
								FloorType:         0,
								LoggerData: models.LoggerData{
									FloorProvider:    "provider",
									FloorFetchStatus: ptrutil.ToPtr(3),
								},
								CustomDimensions: "traffic=media;age=23",
							},
						},
					},
				},
				logInfo:    false,
				forRespExt: true,
			},
			want: want{
				logger: ow.cfg.Endpoint + `?json={"pubid":5890,"pid":"0","pdvid":"0","sl":1,"s":[{"sid":"uuid","sn":"sn","au":"au","ps":[]}],"dvc":{},"fmv":"model-version","fsrc":2,"ft":0,"ffs":3,"fp":"provider","cds":"traffic=media;age=23","geo":{},"fskp":1}&pubid=5890`,
				header: http.Header{
					models.USER_AGENT_HEADER: []string{""},
					models.IP_HEADER:         []string{""},
				},
			},
		},
		{
			name: "set floor details from responseExt and cds from rtcx if tracker details are absent",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							Imp: []openrtb2.Imp{
								{
									ID: "imp-1",
								},
							},
						},
					},
					Response: &openrtb2.BidResponse{},
				},
				rCtx: &models.RequestCtx{
					PubID: 5890,
					NewReqExt: &models.RequestExt{
						ExtRequest: openrtb_ext.ExtRequest{},
					},
					ImpBidCtx: map[string]models.ImpCtx{
						"imp-1": {
							AdUnitName: "au",
							SlotName:   "sn",
						},
					},
					CustomDimensions: map[string]models.CustomDimension{
						"author": {
							Value: "robertshinde",
						},
					},
					ResponseExt: openrtb_ext.ExtBidResponse{
						Prebid: &openrtb_ext.ExtResponsePrebid{
							Floors: &openrtb_ext.PriceFloorRules{
								Skipped:     ptrutil.ToPtr(true),
								FetchStatus: openrtb_ext.FetchError,
								Data: &openrtb_ext.PriceFloorData{
									ModelGroups: []openrtb_ext.PriceFloorModelGroup{
										{
											ModelVersion: "model-version",
										},
									},
									FloorProvider: "provider",
								},
								PriceFloorLocation: openrtb_ext.FetchLocation,
								Enforcement: &openrtb_ext.PriceFloorEnforcement{
									EnforcePBS: ptrutil.ToPtr(true),
								},
							},
						},
					},
				},
				logInfo:    false,
				forRespExt: true,
			},
			want: want{
				logger: ow.cfg.Endpoint + `?json={"pubid":5890,"pid":"0","pdvid":"0","sl":1,"s":[{"sid":"uuid","sn":"sn","au":"au","ps":[]}],"dvc":{},"fmv":"model-version","fsrc":2,"ft":1,"ffs":2,"fp":"provider","cds":"author=robertshinde","geo":{},"fskp":1}&pubid=5890`,
				header: http.Header{
					models.USER_AGENT_HEADER: []string{""},
					models.IP_HEADER:         []string{""},
				},
			},
		},
		{
			name: "set floor value from updated impression if tracker details are absent",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							Imp: []openrtb2.Imp{
								{
									ID:          "imp-1",
									BidFloor:    10.10,
									BidFloorCur: "USD",
								},
							},
						},
					},
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "pubmatic",
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp-1",
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					PubID: 5890,
					NewReqExt: &models.RequestExt{
						ExtRequest: openrtb_ext.ExtRequest{},
					},
					ImpBidCtx: map[string]models.ImpCtx{
						"imp-1": {
							AdUnitName:  "au",
							SlotName:    "sn",
							BidFloor:    2.0,
							BidFloorCur: "USD",
						},
					},
					ResponseExt: openrtb_ext.ExtBidResponse{
						Prebid: &openrtb_ext.ExtResponsePrebid{
							Floors: &openrtb_ext.PriceFloorRules{
								FetchStatus: openrtb_ext.FetchError,
								Data: &openrtb_ext.PriceFloorData{
									ModelGroups: []openrtb_ext.PriceFloorModelGroup{
										{
											ModelVersion: "model-version",
										},
									},
									FloorProvider: "provider1",
								},
								PriceFloorLocation: openrtb_ext.FetchLocation,
								Enforcement: &openrtb_ext.PriceFloorEnforcement{
									EnforcePBS: ptrutil.ToPtr(true),
								},
							},
						},
					},
				},
				logInfo:    false,
				forRespExt: true,
			},
			want: want{
				logger: ow.cfg.Endpoint + `?json={"pubid":5890,"pid":"0","pdvid":"0","sl":1,"s":[{"sid":"uuid","sn":"sn","au":"au","ps":[{"pn":"pubmatic","bc":"pubmatic","kgpv":"","kgpsv":"","psz":"0x0","af":"","eg":0,"en":0,"l1":0,"l2":0,"t":0,"wb":0,"bidid":"bid-id-1","origbidid":"bid-id-1","di":"-1","dc":"","db":1,"ss":1,"mi":0,"ocpm":0,"ocry":"USD","fv":10.1,"frv":10.1}]}],"dvc":{},"fmv":"model-version","fsrc":2,"ft":1,"ffs":2,"fp":"provider1","geo":{}}&pubid=5890`,
				header: http.Header{
					models.USER_AGENT_HEADER: []string{""},
					models.IP_HEADER:         []string{""},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, header := GetLogAuctionObjectAsURL(tt.args.ao, tt.args.rCtx, tt.args.logInfo, tt.args.forRespExt)
			logger, _ = url.PathUnescape(logger)
			assert.Equal(t, tt.want.logger, logger, tt.name)
			assert.Equal(t, tt.want.header, header, tt.name)
		})
	}
}

func TestGetLogAuctionObjectAsURLForProfileMetaData(t *testing.T) {
	cfg := ow.cfg
	uuidFunc := GetUUID
	defer func() {
		ow.cfg = cfg
		GetUUID = uuidFunc
	}()

	GetUUID = func() string { return "uuid" }
	ow.cfg.Endpoint = "http://10.172.141.11/wl"
	ow.cfg.PublicEndpoint = "http://t.pubmatic.com/wl"

	type args struct {
		ao                  analytics.AuctionObject
		rCtx                *models.RequestCtx
		logInfo, forRespExt bool
	}
	type want struct {
		logger string
		header http.Header
	}
	tests := []struct {
		name string
		args args
		want want
	}{

		{
			name: "all profile meta data is present in rctx",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							Imp: []openrtb2.Imp{
								{
									ID:          "imp-1",
									BidFloor:    10.10,
									BidFloorCur: "USD",
								},
							},
						},
					},
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "pubmatic",
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp-1",
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					PubID: 5890,
					NewReqExt: &models.RequestExt{
						ExtRequest: openrtb_ext.ExtRequest{},
					},
					ImpBidCtx: map[string]models.ImpCtx{},
					ResponseExt: openrtb_ext.ExtBidResponse{
						Prebid: &openrtb_ext.ExtResponsePrebid{},
					},
					PartnerConfigMap: map[int]map[string]string{
						-1: {
							"type":               "1",
							"platform":           "in-app",
							"appPlatform":        "5",
							"integrationPath":    "React Native Plugin",
							"subIntegrationPath": "AppLovin Max SDK Bidding",
						},
					},
					ProfileType:           1,
					ProfileTypePlatform:   4,
					AppPlatform:           5,
					AppIntegrationPath:    ptrutil.ToPtr(3),
					AppSubIntegrationPath: ptrutil.ToPtr(8),
				},
				logInfo:    false,
				forRespExt: true,
			},
			want: want{
				logger: ow.cfg.Endpoint + `?json={"pubid":5890,"pid":"0","pdvid":"0","sl":1,"dvc":{},"ft":0,"geo":{},"pt":1,"ptp":4,"ap":5,"aip":3,"asip":8}&pubid=5890`,
				header: http.Header{
					models.USER_AGENT_HEADER: []string{""},
					models.IP_HEADER:         []string{""},
				},
			},
		},
		{
			name: "some profile meta data is not present in rctx",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							Imp: []openrtb2.Imp{
								{
									ID:          "imp-1",
									BidFloor:    10.10,
									BidFloorCur: "USD",
								},
							},
						},
					},
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "pubmatic",
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp-1",
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					PubID: 5890,
					NewReqExt: &models.RequestExt{
						ExtRequest: openrtb_ext.ExtRequest{},
					},
					ImpBidCtx: map[string]models.ImpCtx{},
					ResponseExt: openrtb_ext.ExtBidResponse{
						Prebid: &openrtb_ext.ExtResponsePrebid{},
					},
					PartnerConfigMap: map[int]map[string]string{
						-1: {
							"type":             "1",
							"platform":         "in-app",
							models.AdserverKey: "DFP",
						},
					},
					ProfileType:           1,
					ProfileTypePlatform:   4,
					AppSubIntegrationPath: ptrutil.ToPtr(1),
				},
				logInfo:    false,
				forRespExt: true,
			},
			want: want{
				logger: ow.cfg.Endpoint + `?json={"pubid":5890,"pid":"0","pdvid":"0","sl":1,"dvc":{},"ft":0,"geo":{},"pt":1,"ptp":4,"asip":1}&pubid=5890`,
				header: http.Header{
					models.USER_AGENT_HEADER: []string{""},
					models.IP_HEADER:         []string{""},
				},
			},
		},
		{
			name: "appIntegratioPath and appSubIntegrationPath are nil so it should not be present in the logger",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							Imp: []openrtb2.Imp{
								{
									ID:          "imp-1",
									BidFloor:    10.10,
									BidFloorCur: "USD",
								},
							},
						},
					},
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "pubmatic",
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp-1",
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					PubID: 5890,
					NewReqExt: &models.RequestExt{
						ExtRequest: openrtb_ext.ExtRequest{},
					},
					ImpBidCtx: map[string]models.ImpCtx{},
					ResponseExt: openrtb_ext.ExtBidResponse{
						Prebid: &openrtb_ext.ExtResponsePrebid{},
					},
					PartnerConfigMap: map[int]map[string]string{
						-1: {
							"type":             "1",
							"platform":         "in-app",
							models.AdserverKey: "DFP",
						},
					},
					ProfileType:           1,
					ProfileTypePlatform:   4,
					AppIntegrationPath:    nil,
					AppSubIntegrationPath: nil,
				},
				logInfo:    false,
				forRespExt: true,
			},
			want: want{
				logger: ow.cfg.Endpoint + `?json={"pubid":5890,"pid":"0","pdvid":"0","sl":1,"dvc":{},"ft":0,"geo":{},"pt":1,"ptp":4}&pubid=5890`,
				header: http.Header{
					models.USER_AGENT_HEADER: []string{""},
					models.IP_HEADER:         []string{""},
				},
			},
		},
		{
			name: "appIntegratioPath and appSubIntegrationPath are -1 so it should not be present in the logger",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							Imp: []openrtb2.Imp{
								{
									ID:          "imp-1",
									BidFloor:    10.10,
									BidFloorCur: "USD",
								},
							},
						},
					},
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "pubmatic",
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp-1",
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					PubID: 5890,
					NewReqExt: &models.RequestExt{
						ExtRequest: openrtb_ext.ExtRequest{},
					},
					ImpBidCtx: map[string]models.ImpCtx{},
					ResponseExt: openrtb_ext.ExtBidResponse{
						Prebid: &openrtb_ext.ExtResponsePrebid{},
					},
					PartnerConfigMap: map[int]map[string]string{
						-1: {
							"type":             "1",
							"platform":         "in-app",
							models.AdserverKey: "DFP",
						},
					},
					ProfileType:           1,
					ProfileTypePlatform:   4,
					AppIntegrationPath:    ptrutil.ToPtr(-1),
					AppSubIntegrationPath: ptrutil.ToPtr(-1),
				},
				logInfo:    false,
				forRespExt: true,
			},
			want: want{
				logger: ow.cfg.Endpoint + `?json={"pubid":5890,"pid":"0","pdvid":"0","sl":1,"dvc":{},"ft":0,"geo":{},"pt":1,"ptp":4}&pubid=5890`,
				header: http.Header{
					models.USER_AGENT_HEADER: []string{""},
					models.IP_HEADER:         []string{""},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, header := GetLogAuctionObjectAsURL(tt.args.ao, tt.args.rCtx, tt.args.logInfo, tt.args.forRespExt)
			logger, _ = url.PathUnescape(logger)
			assert.Equal(t, tt.want.logger, logger, tt.name)
			assert.Equal(t, tt.want.header, header, tt.name)
		})
	}
}

func TestSlotRecordsInGetLogAuctionObjectAsURL(t *testing.T) {
	cfg := ow.cfg
	uuidFunc := GetUUID
	defer func() {
		ow.cfg = cfg
		GetUUID = uuidFunc
	}()

	GetUUID = func() string {
		return "sid"
	}

	ow.cfg.Endpoint = "http://10.172.141.11/wl"
	ow.cfg.PublicEndpoint = "http://t.pubmatic.com/wl"

	type args struct {
		ao                  analytics.AuctionObject
		rCtx                *models.RequestCtx
		logInfo, forRespExt bool
	}
	type want struct {
		logger string
		header http.Header
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "req.Imp not mapped in ImpBidCtx",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							Imp: []openrtb2.Imp{
								{
									ID:    "imp1",
									TagID: "tagid",
								},
							},
						},
					},
					Response: &openrtb2.BidResponse{},
				},
				rCtx: &models.RequestCtx{
					Endpoint: models.EndpointV25,
					PubID:    5890,
				},
				logInfo:    false,
				forRespExt: true,
			},
			want: want{
				logger: ow.cfg.Endpoint + `?json={"pubid":5890,"pid":"0","pdvid":"0","sl":1,"dvc":{},"ft":0,"it":"sdk"}&pubid=5890`,
				header: http.Header{
					models.USER_AGENT_HEADER: []string{""},
					models.IP_HEADER:         []string{""},
				},
			},
		},
		{
			name: "multi imps request",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							Imp: []openrtb2.Imp{
								{
									ID:                "imp_1",
									TagID:             "tagid_1",
									DisplayManager:    "pubmatic_sdk",
									DisplayManagerVer: "1.2",
								},
								{
									ID:    "imp_2",
									TagID: "tagid_2",
								},
							},
						},
					},
					Response: &openrtb2.BidResponse{},
				},
				rCtx: &models.RequestCtx{
					PubID:    5890,
					Endpoint: models.EndpointV25,
					ImpBidCtx: map[string]models.ImpCtx{
						"imp_1": {
							SlotName:          "imp_1_tagid_1",
							AdUnitName:        "tagid_1",
							DisplayManager:    "pubmatic_sdk",
							DisplayManagerVer: "1.2",
						},
						"imp_2": {
							AdUnitName: "tagid_2",
							SlotName:   "imp_2_tagid_2",
						},
					},
				},
				logInfo:    false,
				forRespExt: true,
			},
			want: want{
				logger: ow.cfg.Endpoint + `?json={"pubid":5890,"pid":"0","pdvid":"0","sl":1,"s":[{"sid":"sid","sn":"imp_1_tagid_1","au":"tagid_1","ps":[],"dm":"pubmatic_sdk","dmv":"1.2"},{"sid":"sid","sn":"imp_2_tagid_2","au":"tagid_2","ps":[]}],"dvc":{},"ft":0,"it":"sdk"}&pubid=5890`,
				header: http.Header{
					models.USER_AGENT_HEADER: []string{""},
					models.IP_HEADER:         []string{""},
				},
			},
		},
		{
			name: "multi imps request and one request has incomingslots",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							Imp: []openrtb2.Imp{
								{
									ID:    "imp_1",
									TagID: "tagid_1",
								},
								{
									ID:    "imp_2",
									TagID: "tagid_2",
								},
							},
						},
					},
					Response: &openrtb2.BidResponse{},
				},
				rCtx: &models.RequestCtx{
					PubID:    5890,
					Endpoint: models.EndpointV25,
					ImpBidCtx: map[string]models.ImpCtx{
						"imp_1": {
							IncomingSlots:     []string{"0x0v", "100x200"},
							IsRewardInventory: ptrutil.ToPtr(int8(1)),
							SlotName:          "imp_1_tagid_1",
							AdUnitName:        "tagid_1",
						},
						"imp_2": {
							AdUnitName: "tagid_2",
							SlotName:   "imp_2_tagid_2",
						},
					},
				},
				logInfo:    false,
				forRespExt: true,
			},
			want: want{
				logger: ow.cfg.Endpoint + `?json={"pubid":5890,"pid":"0","pdvid":"0","sl":1,"s":[{"sid":"sid","sn":"imp_1_tagid_1","sz":["0x0v","100x200"],"au":"tagid_1","ps":[],"rwrd":1},{"sid":"sid","sn":"imp_2_tagid_2","au":"tagid_2","ps":[]}],"dvc":{},"ft":0,"it":"sdk"}&pubid=5890`,
				header: http.Header{
					models.USER_AGENT_HEADER: []string{""},
					models.IP_HEADER:         []string{""},
				},
			},
		},
		{
			name: "multi imps request and one imp has partner record",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{
							Imp: []openrtb2.Imp{
								{
									ID:    "imp_1",
									TagID: "tagid_1",
								},
								{
									ID:    "imp_2",
									TagID: "tagid_2",
								},
							},
						},
					},
					Response: &openrtb2.BidResponse{
						SeatBid: []openrtb2.SeatBid{
							{
								Seat: "pubmatic",
								Bid: []openrtb2.Bid{
									{
										ID:    "bid-id-1",
										ImpID: "imp_1",
									},
								},
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					PubID:    5890,
					Endpoint: models.EndpointV25,
					ImpBidCtx: map[string]models.ImpCtx{
						"imp_1": {
							IncomingSlots:     []string{"0x0v", "100x200"},
							IsRewardInventory: ptrutil.ToPtr(int8(1)),
							SlotName:          "imp_1_tagid_1",
							AdUnitName:        "tagid_1",
						},
						"imp_2": {
							AdUnitName: "tagid_2",
							SlotName:   "imp_2_tagid_2",
						},
					},
				},
				logInfo:    false,
				forRespExt: true,
			},
			want: want{
				logger: ow.cfg.Endpoint + `?json={"pubid":5890,"pid":"0","pdvid":"0","sl":1,"s":[{"sid":"sid","sn":"imp_1_tagid_1","sz":["0x0v","100x200"],"au":"tagid_1",` +
					`"ps":[{"pn":"pubmatic","bc":"pubmatic","kgpv":"","kgpsv":"","psz":"0x0","af":"","eg":0,"en":0,"l1":0,"l2":0,"t":0,"wb":0,"bidid":"bid-id-1",` +
					`"origbidid":"bid-id-1","di":"-1","dc":"","db":0,"ss":1,"mi":0,"ocpm":0,"ocry":"USD"}],"rwrd":1},{"sid":"sid","sn":"imp_2_tagid_2","au":"tagid_2","ps":[]}],"dvc":{},"ft":0,"it":"sdk"}&pubid=5890`,
				header: http.Header{
					models.USER_AGENT_HEADER: []string{""},
					models.IP_HEADER:         []string{""},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, header := GetLogAuctionObjectAsURL(tt.args.ao, tt.args.rCtx, tt.args.logInfo, tt.args.forRespExt)
			assert.Equal(t, tt.want.header, header, tt.name)
			logger, _ = url.QueryUnescape(logger)
			loggerURL, err := url.Parse(logger)
			if err != nil {
				t.Fail()
			}
			expectedLoggerURL, err := url.Parse(tt.want.logger)
			if err != nil {
				t.Fail()
			}
			assert.Equal(t, expectedLoggerURL.Hostname(), loggerURL.Hostname(), tt.name)
			assert.Equal(t, expectedLoggerURL.Path, loggerURL.Path, tt.name)

			// actualQueryParams := loggerURL.Query()
			// actualJSON := actualQueryParams.Get("json")

			// expectedQueryParams := expectedLoggerURL.Query()
			// expectedJSON := expectedQueryParams.Get("json")

			// fmt.Println(actualJSON)
			// fmt.Println(expectedJSON)
			// assert.JSONEq(t, expectedJSON, actualJSON, tt.name)
		})
	}
}

func Test_getFloorValueFromUpdatedRequest(t *testing.T) {
	type args struct {
		reqWrapper *openrtb_ext.RequestWrapper
		rCtx       *models.RequestCtx
	}
	tests := []struct {
		name string
		args args
		want *models.RequestCtx
	}{
		{
			name: "No floor present in request and in rctx",
			args: args{
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						Imp: []openrtb2.Imp{
							{
								ID:    "imp_1",
								TagID: "tagid_1",
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					PubID:    5890,
					Endpoint: models.EndpointV25,
					ImpBidCtx: map[string]models.ImpCtx{
						"imp_1": {
							AdUnitName: "tagid_1",
						},
					},
				},
			},
			want: &models.RequestCtx{
				PubID:    5890,
				Endpoint: models.EndpointV25,
				ImpBidCtx: map[string]models.ImpCtx{
					"imp_1": {
						AdUnitName: "tagid_1",
					},
				},
			},
		},
		{
			name: "No floor change in request and in rctx",
			args: args{
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						Imp: []openrtb2.Imp{
							{
								ID:          "imp_1",
								TagID:       "tagid_1",
								BidFloor:    10,
								BidFloorCur: "USD",
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					PubID:    5890,
					Endpoint: models.EndpointV25,
					ImpBidCtx: map[string]models.ImpCtx{
						"imp_1": {
							AdUnitName:  "tagid_1",
							BidFloor:    10,
							BidFloorCur: "USD",
						},
					},
				},
			},
			want: &models.RequestCtx{
				PubID:    5890,
				Endpoint: models.EndpointV25,
				ImpBidCtx: map[string]models.ImpCtx{
					"imp_1": {
						AdUnitName:  "tagid_1",
						BidFloor:    10,
						BidFloorCur: "USD",
					},
				},
			},
		},
		{
			name: "floor updated in request",
			args: args{
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						Imp: []openrtb2.Imp{
							{
								ID:          "imp_1",
								TagID:       "tagid_1",
								BidFloor:    20,
								BidFloorCur: "EUR",
							},
						},
					},
				},
				rCtx: &models.RequestCtx{
					PubID:    5890,
					Endpoint: models.EndpointV25,
					ImpBidCtx: map[string]models.ImpCtx{
						"imp_1": {
							AdUnitName:  "tagid_1",
							BidFloor:    10,
							BidFloorCur: "USD",
						},
					},
				},
			},
			want: &models.RequestCtx{
				PubID:    5890,
				Endpoint: models.EndpointV25,
				ImpBidCtx: map[string]models.ImpCtx{
					"imp_1": {
						AdUnitName:  "tagid_1",
						BidFloor:    20,
						BidFloorCur: "EUR",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getFloorValueFromUpdatedRequest(tt.args.reqWrapper, tt.args.rCtx)
			assert.Equal(t, tt.want, tt.args.rCtx, tt.name)
		})
	}
}

func TestGetBidPriceAfterCurrencyConversion(t *testing.T) {
	type args struct {
		price             float64
		requestCurrencies []string
		responseCurrency  string
		currencyConverter func(fromCurrency string, toCurrency string, value float64) (float64, error)
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		{
			name: "Single request currency - successful conversion",
			args: args{
				price:             100.0,
				requestCurrencies: []string{"EUR"},
				responseCurrency:  "USD",
				currencyConverter: func(fromCurrency string, toCurrency string, value float64) (float64, error) {
					if fromCurrency == "USD" && toCurrency == "EUR" {
						return 85.0, nil // Assuming conversion rate USD to EUR is 0.85
					}
					return 0, fmt.Errorf("unsupported conversion")
				},
			},
			want: 85.0,
		},
		{
			name: "Multiple request currencies - first successful conversion",
			args: args{
				price:             100.0,
				requestCurrencies: []string{"EUR", "GBP"},
				responseCurrency:  "USD",
				currencyConverter: func(fromCurrency string, toCurrency string, value float64) (float64, error) {
					if fromCurrency == "USD" && toCurrency == "EUR" {
						return 85.0, nil // Successful conversion to EUR
					}
					return 0, fmt.Errorf("unsupported conversion")
				},
			},
			want: 85.0,
		},
		{
			name: "Multiple request currencies - second successful conversion",
			args: args{
				price:             100.0,
				requestCurrencies: []string{"JPY", "GBP"},
				responseCurrency:  "USD",
				currencyConverter: func(fromCurrency string, toCurrency string, value float64) (float64, error) {
					if fromCurrency == "USD" && toCurrency == "GBP" {
						return 75.0, nil // Successful conversion to GBP
					}
					return 0, fmt.Errorf("unsupported conversion")
				},
			},
			want: 75.0,
		},
		{
			name: "No request currencies provided - default to USD",
			args: args{
				price:             100.0,
				requestCurrencies: []string{},
				responseCurrency:  "USD",
				currencyConverter: func(fromCurrency string, toCurrency string, value float64) (float64, error) {
					if fromCurrency == "USD" && toCurrency == "USD" {
						return 100.0, nil // No conversion needed
					}
					return 0, fmt.Errorf("unsupported conversion")
				},
			},
			want: 100.0,
		},
		{
			name: "Conversion fails for all currencies",
			args: args{
				price:             100.0,
				requestCurrencies: []string{"JPY", "CNY"},
				responseCurrency:  "USD",
				currencyConverter: func(fromCurrency string, toCurrency string, value float64) (float64, error) {
					return 0, fmt.Errorf("conversion failed")
				},
			},
			want: 0.0, // Default to 0 on failure
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetBidPriceAfterCurrencyConversion(tt.args.price, tt.args.requestCurrencies, tt.args.responseCurrency, tt.args.currencyConverter)
			assert.Equal(t, tt.want, got, "mismatched price")
		})
	}
}

func TestGetLogAuctionObjectAsURLForEdsStatus(t *testing.T) {
	cfg := ow.cfg
	defer func() {
		ow.cfg = cfg
	}()

	ow.cfg.Endpoint = "http://10.172.141.11/wl"
	ow.cfg.PublicEndpoint = "http://t.pubmatic.com/wl"

	type args struct {
		ao                  analytics.AuctionObject
		rCtx                *models.RequestCtx
		logInfo, forRespExt bool
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "edsstatus present on request ctx",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{},
					},
					Response: &openrtb2.BidResponse{},
				},
				rCtx: &models.RequestCtx{
					PubID:     5890,
					EdsStatus: ptrutil.ToPtr(1),
				},
				logInfo:    true,
				forRespExt: true,
			},
			want: ow.cfg.PublicEndpoint + `?json={"pubid":5890,"pid":"0","pdvid":"0","sl":1,"dvc":{},"ft":0,"geo":{},"edss":1}&pubid=5890`,
		},
		{
			name: "edsstatus absent from request ctx",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{},
					},
					Response: &openrtb2.BidResponse{},
				},
				rCtx: &models.RequestCtx{
					PubID: 5890,
				},
				logInfo:    true,
				forRespExt: true,
			},
			want: ow.cfg.PublicEndpoint + `?json={"pubid":5890,"pid":"0","pdvid":"0","sl":1,"dvc":{},"ft":0,"geo":{}}&pubid=5890`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := GetLogAuctionObjectAsURL(tt.args.ao, tt.args.rCtx, tt.args.logInfo, tt.args.forRespExt)
			logger, _ = url.QueryUnescape(logger)
			assert.Equal(t, tt.want, logger, tt.name)
		})
	}
}

func TestGetLogAuctionObjectAsURLForVastUnwrap(t *testing.T) {
	cfg := ow.cfg
	defer func() {
		ow.cfg = cfg
	}()

	ow.cfg.Endpoint = "http://10.172.141.11/wl"
	ow.cfg.PublicEndpoint = "http://t.pubmatic.com/wl"

	type args struct {
		ao                  analytics.AuctionObject
		rCtx                *models.RequestCtx
		logInfo, forRespExt bool
	}
	type want struct {
		logger string
		header http.Header
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "VastUnwrapEnabled is true",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{},
					},
					Response: &openrtb2.BidResponse{},
				},
				rCtx: &models.RequestCtx{
					PubID: 5890,
					VastUnWrap: models.VastUnWrap{
						Enabled: true,
					},
				},
				logInfo:    true,
				forRespExt: true,
			},
			want: want{
				logger: ow.cfg.PublicEndpoint + `?json={"pubid":5890,"pid":"0","pdvid":"0","sl":1,"dvc":{},"vu":1,"ft":0,"geo":{}}&pubid=5890`,
				header: http.Header{
					models.USER_AGENT_HEADER: []string{""},
					models.IP_HEADER:         []string{""},
				},
			},
		},
		{
			name: "VastUnwrapEnabled is false",
			args: args{
				ao: analytics.AuctionObject{
					RequestWrapper: &openrtb_ext.RequestWrapper{
						BidRequest: &openrtb2.BidRequest{},
					},
					Response: &openrtb2.BidResponse{},
				},
				rCtx: &models.RequestCtx{
					PubID: 5890,
					VastUnWrap: models.VastUnWrap{
						Enabled: false,
					},
				},
				logInfo:    true,
				forRespExt: true,
			},
			want: want{
				logger: ow.cfg.PublicEndpoint + `?json={"pubid":5890,"pid":"0","pdvid":"0","sl":1,"dvc":{},"ft":0,"geo":{}}&pubid=5890`,
				header: http.Header{
					models.USER_AGENT_HEADER: []string{""},
					models.IP_HEADER:         []string{""},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, header := GetLogAuctionObjectAsURL(tt.args.ao, tt.args.rCtx, tt.args.logInfo, tt.args.forRespExt)
			logger, _ = url.QueryUnescape(logger)
			assert.Equal(t, tt.want.logger, logger, tt.name)
			assert.Equal(t, tt.want.header, header, tt.name)
		})
	}
}
