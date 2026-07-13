package tracker

import (
	"encoding/json"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

var rctx = models.RequestCtx{
	PubID:               5890,
	ProfileID:           1234,
	DisplayID:           1,
	DisplayVersionID:    1,
	PageURL:             "abc.com",
	LoggerImpressionID:  "loggerIID",
	SSAI:                "mediatailor",
	Origin:              "publisher.com",
	ABTestConfigApplied: 1,
	DeviceCtx: models.DeviceCtx{
		Platform: 5,
	},
	PrebidBidderCode: map[string]string{
		"pubmatic": "pubmatic",
	},
	MarketPlaceBidders: map[string]struct{}{
		"pubmatic": {},
	},
	CustomDimensions: map[string]models.CustomDimension{"author": {Value: "henry"}},
	ImpBidCtx: map[string]models.ImpCtx{
		"impID-1": {
			TagID:      "adunit-1",
			AdUnitName: "adunit-1",
			SlotName:   "impID-1_adunit-1",
			Bidders: map[string]models.PartnerData{
				"pubmatic": {
					MatchedSlot:      "matchedSlot",
					PrebidBidderCode: "prebidBidderCode",
					KGP:              "_AU_@_W_x_H_",
				},
				"pubmatic2": {
					MatchedSlot:      "matchedSlot2",
					PrebidBidderCode: "prebidBidderCode2",
					KGP:              "_AU_@_W_x_H_",
				},
			},
			BidFloor:    5.5,
			BidFloorCur: "EUR",
			BidCtx: map[string]models.BidCtx{
				"bidID-1": {
					EG: 8.7,
					EN: 8.7,
					BidExt: models.BidExt{
						OriginalBidCPMUSD: 0,
						ExtBid: openrtb_ext.ExtBid{
							Prebid: &openrtb_ext.ExtBidPrebid{
								BidId: "bidID-1",
								Video: &openrtb_ext.ExtBidPrebidVideo{
									Duration: 20,
								},
								Meta: &openrtb_ext.ExtBidPrebidMeta{
									AdapterCode: "pubmatic",
									NetworkID:   123456,
								},
								Floors: &openrtb_ext.ExtBidPrebidFloors{
									FloorRule:      "rule1",
									FloorValue:     6.4,
									FloorRuleValue: 4.4,
								},
								Type: models.Banner,
							},
						},
					},
				},
			},
		},
	},
}

func Test_createTrackers(t *testing.T) {
	startTime := time.Now().Unix()
	type args struct {
		trackers            map[string]models.OWTracker
		rctx                models.RequestCtx
		bidResponse         *openrtb2.BidResponse
		pmMkt               map[string]pubmaticMarketplaceMeta
		globalAccountConfig *config.Account
	}
	tests := []struct {
		name string
		args args
		want map[string]models.OWTracker
	}{
		{
			name: "empty_bidResponse",
			args: args{
				trackers:    map[string]models.OWTracker{},
				bidResponse: &openrtb2.BidResponse{},
			},
			want: map[string]models.OWTracker{},
		},
		{
			name: "response with all details",
			args: args{
				trackers: map[string]models.OWTracker{},
				rctx: func() models.RequestCtx {
					testRctx := rctx
					testRctx.StartTime = startTime
					pg, _ := openrtb_ext.NewPriceGranularityFromLegacyID("med")
					testRctx.PriceGranularity = &pg
					testRctx.DeviceCtx.Ext = func() *models.ExtDevice {
						extDevice := models.ExtDevice{}
						extDevice.UnmarshalJSON([]byte(`{"atts":1}`))
						return &extDevice
					}()
					return testRctx
				}(),
				globalAccountConfig: &config.Account{BidRounding: config.DefaultBidRoundingMode},
				bidResponse: &openrtb2.BidResponse{
					SeatBid: []openrtb2.SeatBid{
						{
							Bid: []openrtb2.Bid{
								{
									ID:      "bidID-1",
									ImpID:   "impID-1",
									Price:   8.7,
									W:       250,
									H:       300,
									ADomain: []string{"domain.com"},
									DealID:  "deal-id-1",
									Ext:     json.RawMessage(`{"prebid":{"meta":{"networkId":123456}}}`),
								},
							},
							Seat: "pubmatic",
						},
					},
					Cur: models.USD,
				},
				pmMkt: map[string]pubmaticMarketplaceMeta{},
			},
			want: map[string]models.OWTracker{
				"bidID-1": {
					Tracker: models.Tracker{
						PubID:     5890,
						PageURL:   "abc.com",
						Timestamp: startTime,
						IID:       "loggerIID",
						ProfileID: "1234",
						VersionID: "1",
						Adunit:    "adunit-1",
						SlotID:    "impID-1_adunit-1",
						PartnerInfo: models.Partner{
							PartnerID:      "prebidBidderCode",
							BidderCode:     "pubmatic",
							KGPV:           "adunit-1@250x300",
							GrossECPM:      8.7,
							NetECPM:        8.7,
							BidID:          "bidID-1",
							OrigBidID:      "bidID-1",
							AdSize:         "250x300",
							AdDuration:     20,
							Adformat:       "banner",
							ServerSide:     1,
							Advertiser:     "domain.com",
							FloorValue:     6.4,
							FloorRuleValue: 4.4,
							DealID:         "deal-id-1",
							NetworkID:      123456,
							PriceBucket:    "8.60",
						},
						Platform:  5,
						SSAI:      "mediatailor",
						AdPodSlot: 0,
						TestGroup: 1,
						Origin:    "publisher.com",
						ImpID:     "impID-1",
						LoggerData: models.LoggerData{
							KGPSV: "adunit-1@250x300",
						},
						CustomDimensions: "author=henry",
						ATTS:             ptrutil.ToPtr(float64(openrtb_ext.IOSAppTrackingStatusRestricted)),
					},
					TrackerURL:    "https:?adv=domain.com&af=banner&aps=0&atts=1&au=adunit-1&bc=pubmatic&bidid=bidID-1&cds=author%3Dhenry&di=deal-id-1&dur=20&eg=8.7&en=8.7&frv=4.4&ft=0&fv=6.4&iid=loggerIID&kgpv=adunit-1%40250x300&nwid=123456&orig=publisher.com&origbidid=bidID-1&pb=8.60&pdvid=1&pid=1234&plt=5&pn=prebidBidderCode&psz=250x300&pubid=5890&purl=abc.com&sl=1&slot=impID-1_adunit-1&ss=1&ssai=mediatailor&tgid=1&tst=" + strconv.FormatInt(startTime, 10),
					Price:         8.7,
					PriceModel:    "CPM",
					PriceCurrency: "USD",
					BidType:       "banner",
				},
			},
		},
		{
			name: "response with all details with alias partner",
			args: args{
				trackers: map[string]models.OWTracker{},
				rctx: func() models.RequestCtx {
					testRctx := rctx
					testRctx.StartTime = startTime
					testRctx.PrebidBidderCode["pubmatic2"] = "pubmatic"
					return testRctx
				}(),
				globalAccountConfig: &config.Account{BidRounding: config.DefaultBidRoundingMode},
				bidResponse: &openrtb2.BidResponse{
					SeatBid: []openrtb2.SeatBid{
						{
							Bid: []openrtb2.Bid{
								{
									ID:      "bidID-1",
									ImpID:   "impID-1",
									Price:   8.7,
									W:       250,
									H:       300,
									ADomain: []string{"domain.com"},
									DealID:  "deal-id-1",
									Ext:     json.RawMessage(`{"prebid":{"meta":{"networkId":123456}}}`),
								},
							},
							Seat: "pubmatic2",
						},
					},
					Cur: models.USD,
				},
				pmMkt: map[string]pubmaticMarketplaceMeta{},
			},
			want: map[string]models.OWTracker{
				"bidID-1": {
					Tracker: models.Tracker{
						PubID:     5890,
						PageURL:   "abc.com",
						Timestamp: startTime,
						IID:       "loggerIID",
						ProfileID: "1234",
						VersionID: "1",
						Adunit:    "adunit-1",
						SlotID:    "impID-1_adunit-1",
						PartnerInfo: models.Partner{
							PartnerID:      "prebidBidderCode2",
							BidderCode:     "pubmatic2",
							KGPV:           "adunit-1@250x300",
							GrossECPM:      8.7,
							NetECPM:        8.7,
							BidID:          "bidID-1",
							OrigBidID:      "bidID-1",
							AdSize:         "250x300",
							AdDuration:     20,
							Adformat:       "banner",
							ServerSide:     1,
							Advertiser:     "domain.com",
							FloorValue:     6.4,
							FloorRuleValue: 4.4,
							DealID:         "deal-id-1",
							NetworkID:      123456,
						},
						Platform:  5,
						SSAI:      "mediatailor",
						AdPodSlot: 0,
						TestGroup: 1,
						Origin:    "publisher.com",
						ImpID:     "impID-1",
						LoggerData: models.LoggerData{
							KGPSV: "adunit-1@250x300",
						},
						CustomDimensions: "author=henry",
					},
					TrackerURL:    "https:?adv=domain.com&af=banner&aps=0&au=adunit-1&bc=pubmatic2&bidid=bidID-1&cds=author%3Dhenry&di=deal-id-1&dur=20&eg=8.7&en=8.7&frv=4.4&ft=0&fv=6.4&iid=loggerIID&kgpv=adunit-1%40250x300&nwid=123456&orig=publisher.com&origbidid=bidID-1&pdvid=1&pid=1234&plt=5&pn=prebidBidderCode2&psz=250x300&pubid=5890&purl=abc.com&sl=1&slot=impID-1_adunit-1&ss=1&ssai=mediatailor&tgid=1&tst=" + strconv.FormatInt(startTime, 10),
					Price:         8.7,
					PriceModel:    "CPM",
					PriceCurrency: "USD",
					BidType:       "banner",
				},
			},
		},
		{
			name: "response with all details and multibid-multifloor feature enabled",
			args: args{
				trackers: map[string]models.OWTracker{},
				rctx: func() models.RequestCtx {
					testRctx := rctx
					testRctx.StartTime = startTime
					pg, _ := openrtb_ext.NewPriceGranularityFromLegacyID("med")
					testRctx.PriceGranularity = &pg
					testRctx.AppLovinMax = models.AppLovinMax{}
					testRctx.ImpBidCtx = map[string]models.ImpCtx{
						"impID-1": {
							TagID:             "adunit-1",
							AdUnitName:        "adunit-1",
							DisplayManager:    "PubMatic_OpenWrap_SDK",
							DisplayManagerVer: "1.2",
							SlotName:          "impID-1_adunit-1",
							Bidders: map[string]models.PartnerData{
								"pubmatic": {
									MatchedSlot:      "matchedSlot",
									PrebidBidderCode: "prebidBidderCode",
									KGP:              "_AU_@_W_x_H_",
								},
								"pubmatic2": {
									MatchedSlot:      "matchedSlot2",
									PrebidBidderCode: "prebidBidderCode2",
									KGP:              "_AU_@_W_x_H_",
								},
							},
							BidFloor:    5.5,
							BidFloorCur: "EUR",
							BidCtx: map[string]models.BidCtx{
								"bidID-1": {
									EG: 8.7,
									EN: 8.7,
									BidExt: models.BidExt{
										OriginalBidCPMUSD: 0,
										ExtBid: openrtb_ext.ExtBid{
											Prebid: &openrtb_ext.ExtBidPrebid{
												BidId: "bidID-1",
												Video: &openrtb_ext.ExtBidPrebidVideo{
													Duration: 20,
												},
												Meta: &openrtb_ext.ExtBidPrebidMeta{
													AdapterCode: "pubmatic",
													NetworkID:   123456,
												},
												Floors: &openrtb_ext.ExtBidPrebidFloors{
													FloorRule:      "rule1",
													FloorValue:     6.4,
													FloorRuleValue: 4.4,
												},
												Type: models.Banner,
											},
										},
										MultiBidMultiFloorValue: 4.5,
									},
								},
							},
						},
					}
					testRctx.DeviceCtx.Ext = func() *models.ExtDevice {
						extDevice := models.ExtDevice{}
						extDevice.UnmarshalJSON([]byte(`{"atts":1}`))
						return &extDevice
					}()
					return testRctx
				}(),
				globalAccountConfig: &config.Account{BidRounding: config.DefaultBidRoundingMode},
				bidResponse: &openrtb2.BidResponse{
					SeatBid: []openrtb2.SeatBid{
						{
							Bid: []openrtb2.Bid{
								{
									ID:      "bidID-1",
									ImpID:   "impID-1",
									Price:   8.7,
									W:       250,
									H:       300,
									ADomain: []string{"domain.com"},
									DealID:  "deal-id-1",
									Ext:     json.RawMessage(`{"prebid":{"meta":{"networkId":123456}}}`),
								},
							},
							Seat: "pubmatic",
						},
					},
					Cur: models.USD,
				},
				pmMkt: map[string]pubmaticMarketplaceMeta{},
			},
			want: map[string]models.OWTracker{
				"bidID-1": {
					Tracker: models.Tracker{
						PubID:     5890,
						PageURL:   "abc.com",
						Timestamp: startTime,
						IID:       "loggerIID",
						ProfileID: "1234",
						VersionID: "1",
						Adunit:    "adunit-1",
						SlotID:    "impID-1_adunit-1",
						PartnerInfo: models.Partner{
							PartnerID:              "prebidBidderCode",
							BidderCode:             "pubmatic",
							KGPV:                   "adunit-1@250x300",
							GrossECPM:              8.7,
							NetECPM:                8.7,
							BidID:                  "bidID-1",
							OrigBidID:              "bidID-1",
							AdSize:                 "250x300",
							AdDuration:             20,
							Adformat:               "banner",
							ServerSide:             1,
							Advertiser:             "domain.com",
							FloorValue:             4.5,
							FloorRuleValue:         4.5,
							DealID:                 "deal-id-1",
							NetworkID:              123456,
							PriceBucket:            "8.60",
							MultiBidMultiFloorFlag: 1,
						},
						Platform:  5,
						SSAI:      "mediatailor",
						AdPodSlot: 0,
						TestGroup: 1,
						Origin:    "publisher.com",
						ImpID:     "impID-1",
						LoggerData: models.LoggerData{
							KGPSV: "adunit-1@250x300",
						},
						CustomDimensions:  "author=henry",
						ATTS:              ptrutil.ToPtr(float64(openrtb_ext.IOSAppTrackingStatusRestricted)),
						DisplayManager:    "PubMatic_OpenWrap_SDK",
						DisplayManagerVer: "1.2",
					},
					TrackerURL:    "https:?adv=domain.com&af=banner&aps=0&atts=1&au=adunit-1&bc=pubmatic&bidid=bidID-1&cds=author%3Dhenry&di=deal-id-1&dm=PubMatic_OpenWrap_SDK&dmv=1.2&dur=20&eg=8.7&en=8.7&frv=4.5&ft=0&fv=4.5&iid=loggerIID&kgpv=adunit-1%40250x300&mbmf=1&nwid=123456&orig=publisher.com&origbidid=bidID-1&pb=8.60&pdvid=1&pid=1234&plt=5&pn=prebidBidderCode&psz=250x300&pubid=5890&purl=abc.com&sl=1&slot=impID-1_adunit-1&ss=1&ssai=mediatailor&tgid=1&tst=" + strconv.FormatInt(startTime, 10),
					Price:         8.7,
					PriceModel:    "CPM",
					PriceCurrency: "USD",
					BidType:       "banner",
				},
			},
		},
		{
			name: "tracker_url_includes_bexp_and_bexpef",
			args: args{
				trackers: map[string]models.OWTracker{},
				rctx: func() models.RequestCtx {
					testRctx := rctx
					testRctx.StartTime = startTime
					pg, _ := openrtb_ext.NewPriceGranularityFromLegacyID("med")
					testRctx.PriceGranularity = &pg
					testRctx.DeviceCtx.Ext = func() *models.ExtDevice {
						extDevice := models.ExtDevice{}
						extDevice.UnmarshalJSON([]byte(`{"atts":1}`))
						return &extDevice
					}()
					newImpBidCtx := make(map[string]models.ImpCtx, len(rctx.ImpBidCtx))
					for impID, imp := range rctx.ImpBidCtx {
						newImp := imp
						newBidCtx := make(map[string]models.BidCtx, len(imp.BidCtx))
						for bidID, bc := range imp.BidCtx {
							newBidCtx[bidID] = bc
						}
						if impID == "impID-1" {
							bc := newBidCtx["bidID-1"]
							bc.BidExt.BidExpEnf = 1
							newBidCtx["bidID-1"] = bc
						}
						newImp.BidCtx = newBidCtx
						newImpBidCtx[impID] = newImp
					}
					testRctx.ImpBidCtx = newImpBidCtx
					return testRctx
				}(),
				globalAccountConfig: &config.Account{BidRounding: config.DefaultBidRoundingMode},
				bidResponse: &openrtb2.BidResponse{
					SeatBid: []openrtb2.SeatBid{
						{
							Bid: []openrtb2.Bid{
								{
									ID:      "bidID-1",
									ImpID:   "impID-1",
									Price:   8.7,
									W:       250,
									H:       300,
									Exp:     300,
									ADomain: []string{"domain.com"},
									DealID:  "deal-id-1",
									Ext:     json.RawMessage(`{"prebid":{"meta":{"networkId":123456}}}`),
								},
							},
							Seat: "pubmatic",
						},
					},
					Cur: models.USD,
				},
				pmMkt: map[string]pubmaticMarketplaceMeta{},
			},
			want: map[string]models.OWTracker{
				"bidID-1": {
					Tracker: models.Tracker{
						PubID:     5890,
						PageURL:   "abc.com",
						Timestamp: startTime,
						IID:       "loggerIID",
						ProfileID: "1234",
						VersionID: "1",
						Adunit:    "adunit-1",
						SlotID:    "impID-1_adunit-1",
						PartnerInfo: models.Partner{
							PartnerID:      "prebidBidderCode",
							BidderCode:     "pubmatic",
							KGPV:           "adunit-1@250x300",
							GrossECPM:      8.7,
							NetECPM:        8.7,
							BidID:          "bidID-1",
							OrigBidID:      "bidID-1",
							AdSize:         "250x300",
							AdDuration:     20,
							Adformat:       "banner",
							ServerSide:     1,
							Advertiser:     "domain.com",
							FloorValue:     6.4,
							FloorRuleValue: 4.4,
							DealID:         "deal-id-1",
							NetworkID:      123456,
							PriceBucket:    "8.60",
						},
						Platform:  5,
						SSAI:      "mediatailor",
						AdPodSlot: 0,
						TestGroup: 1,
						Origin:    "publisher.com",
						ImpID:     "impID-1",
						LoggerData: models.LoggerData{
							KGPSV: "adunit-1@250x300",
						},
						CustomDimensions: "author=henry",
						ATTS:             ptrutil.ToPtr(float64(openrtb_ext.IOSAppTrackingStatusRestricted)),
						BidExp:           300,
						BidExpEnf:        1,
					},
					TrackerURL:    "https:?adv=domain.com&af=banner&aps=0&atts=1&au=adunit-1&bc=pubmatic&bexp=300&bexpef=1&bidid=bidID-1&cds=author%3Dhenry&di=deal-id-1&dur=20&eg=8.7&en=8.7&frv=4.4&ft=0&fv=6.4&iid=loggerIID&kgpv=adunit-1%40250x300&nwid=123456&orig=publisher.com&origbidid=bidID-1&pb=8.60&pdvid=1&pid=1234&plt=5&pn=prebidBidderCode&psz=250x300&pubid=5890&purl=abc.com&sl=1&slot=impID-1_adunit-1&ss=1&ssai=mediatailor&tgid=1&tst=" + strconv.FormatInt(startTime, 10),
					Price:         8.7,
					PriceModel:    "CPM",
					PriceCurrency: "USD",
					BidType:       "banner",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := createTrackers(tt.args.rctx, tt.args.trackers, tt.args.bidResponse, tt.args.pmMkt, tt.args.globalAccountConfig)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConstructTrackerURL(t *testing.T) {
	type args struct {
		rctx    models.RequestCtx
		tracker models.Tracker
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "trackerEndpoint_parsingError",
			args: args{
				rctx: models.RequestCtx{
					TrackerEndpoint: ":www.example.com",
				},
			},
			want: "",
		},
		{
			name: "empty_tracker_details",
			args: args{
				rctx: models.RequestCtx{
					TrackerEndpoint: "//t.pubmatic.com/wt",
					Platform:        models.PLATFORM_DISPLAY,
				},
				tracker: models.Tracker{},
			},
			want: "http://t.pubmatic.com/wt?adv=&af=&aps=0&au=&bc=&bidid=&di=&eg=0&en=0&ft=0&iid=&kgpv=&orig=&origbidid=&pdvid=&pid=&plt=0&pn=&psz=&pubid=0&purl=&sl=1&slot=&ss=0&tst=0",
		},
		{
			name: "platform_amp_with_tracker_details",
			args: args{
				rctx: models.RequestCtx{
					TrackerEndpoint: "//t.pubmatic.com/wt",
					Platform:        models.PLATFORM_AMP,
				},
				tracker: models.Tracker{
					PubID:     12345,
					PageURL:   "www.abc.com",
					IID:       "98765",
					ProfileID: "123",
					VersionID: "1",
					SlotID:    "1234_1234",
					Adunit:    "adunit",
					Platform:  1,
					Origin:    "www.publisher.com",
					TestGroup: 1,
					AdPodSlot: 0,
					PartnerInfo: models.Partner{
						PartnerID:  "AppNexus",
						BidderCode: "AppNexus1",
						BidID:      "6521",
						OrigBidID:  "6521",
						GrossECPM:  4.3,
						NetECPM:    2.5,
						KGPV:       "adunit@300x250",
						AdDuration: 10,
						Adformat:   models.Banner,
						AdSize:     "300x250",
						ServerSide: 1,
						Advertiser: "fb.com",
						DealID:     "420",
					},
				},
			},
			want: "//t.pubmatic.com/wt?adv=fb.com&af=banner&aps=0&au=adunit&bc=AppNexus1&bidid=6521&di=420&dur=10&eg=4.3&en=2.5&ft=0&iid=98765&kgpv=adunit@300x250&orig=www.publisher.com&origbidid=6521&pdvid=1&pid=123&plt=1&pn=AppNexus&psz=300x250&pubid=12345&purl=www.abc.com&sl=1&slot=1234_1234&ss=1&tgid=1&tst=0",
		},
		{
			name: "all_details_with_ssai_in_tracker",
			args: args{
				rctx: models.RequestCtx{
					TrackerEndpoint: "//t.pubmatic.com/wt",
					Platform:        models.PLATFORM_DISPLAY,
				},
				tracker: models.Tracker{
					PubID:             12345,
					PageURL:           "www.abc.com",
					IID:               "98765",
					ProfileID:         "123",
					VersionID:         "1",
					SlotID:            "1234_1234",
					Adunit:            "adunit",
					Platform:          1,
					Origin:            "www.publisher.com",
					TestGroup:         1,
					AdPodSlot:         0,
					FloorSkippedFlag:  ptrutil.ToPtr(0),
					FloorModelVersion: "test version",
					FloorSource:       ptrutil.ToPtr(1),
					FloorType:         1,
					RewardedInventory: 1,
					Secure:            1,
					SSAI:              "mediatailor",
					CustomDimensions:  "traffic=media;age=23",
					LoggerData: models.LoggerData{
						FloorProvider: "PM",
					},
					PartnerInfo: models.Partner{
						PartnerID:          "AppNexus",
						BidderCode:         "AppNexus1",
						BidID:              "6521",
						OrigBidID:          "6521",
						GrossECPM:          4.3,
						NetECPM:            2.5,
						KGPV:               "adunit@300x250",
						AdDuration:         10,
						Adformat:           models.Banner,
						AdSize:             "300x250",
						ServerSide:         1,
						Advertiser:         "fb.com",
						DealID:             "420",
						FloorValue:         4.4,
						FloorRuleValue:     2,
						InViewCountingFlag: 1,
					},
				},
			},
			want: "https://t.pubmatic.com/wt?adv=fb.com&af=banner&aps=0&au=adunit&bc=AppNexus1&bidid=6521&cds=traffic=media;age=23&ctm=1&di=420&dur=10&eg=4.3&en=2.5&fmv=test version&fp=PM&frv=2&fskp=0&fsrc=1&ft=1&fv=4.4&iid=98765&kgpv=adunit@300x250&orig=www.publisher.com&origbidid=6521&pdvid=1&pid=123&plt=1&pn=AppNexus&psz=300x250&pubid=12345&purl=www.abc.com&rwrd=1&sl=1&slot=1234_1234&ss=1&ssai=mediatailor&tgid=1&tst=0",
		},
		{
			name: "all_details_with_secure_enable_in_tracker",
			args: args{
				rctx: models.RequestCtx{
					TrackerEndpoint: "//t.pubmatic.com/wt",
					Platform:        models.PLATFORM_DISPLAY,
					DeviceCtx:       models.DeviceCtx{DerivedCountryCode: "IN"},
				},
				tracker: models.Tracker{
					PubID:             12345,
					PageURL:           "www.abc.com",
					IID:               "98765",
					ProfileID:         "123",
					VersionID:         "1",
					SlotID:            "1234_1234",
					Adunit:            "adunit",
					Platform:          1,
					Origin:            "www.publisher.com",
					TestGroup:         1,
					AdPodSlot:         0,
					FloorSkippedFlag:  ptrutil.ToPtr(0),
					FloorModelVersion: "test version",
					FloorSource:       ptrutil.ToPtr(1),
					FloorType:         1,
					RewardedInventory: 1,
					Secure:            1,
					CustomDimensions:  "traffic=media;age=23",
					PartnerInfo: models.Partner{
						PartnerID:      "AppNexus",
						BidderCode:     "AppNexus1",
						BidID:          "6521",
						OrigBidID:      "6521",
						GrossECPM:      4.3,
						NetECPM:        2.5,
						KGPV:           "adunit@300x250",
						AdDuration:     10,
						Adformat:       models.Banner,
						AdSize:         "300x250",
						ServerSide:     1,
						Advertiser:     "fb.com",
						DealID:         "420",
						FloorValue:     4.4,
						FloorRuleValue: 2,
					},
					CountryCode: "IN",
				},
			},
			want: "https://t.pubmatic.com/wt?adv=fb.com&af=banner&aps=0&au=adunit&bc=AppNexus1&bidid=6521&cc=IN&cds=traffic=media;age=23&di=420&dur=10&eg=4.3&en=2.5&fmv=test version&frv=2&fskp=0&fsrc=1&ft=1&fv=4.4&iid=98765&kgpv=adunit@300x250&orig=www.publisher.com&origbidid=6521&pdvid=1&pid=123&plt=1&pn=AppNexus&psz=300x250&pubid=12345&purl=www.abc.com&rwrd=1&sl=1&slot=1234_1234&ss=1&tgid=1&tst=0",
		},
		{
			name: "all_details_with_RewardInventory_in_tracker",
			args: args{
				rctx: models.RequestCtx{
					TrackerEndpoint: "//t.pubmatic.com/wt",
					Platform:        models.PLATFORM_APP,
				},
				tracker: models.Tracker{
					PubID:             12345,
					PageURL:           "www.abc.com",
					IID:               "98765",
					ProfileID:         "123",
					VersionID:         "1",
					SlotID:            "1234_1234",
					Adunit:            "adunit",
					Platform:          1,
					Origin:            "www.publisher.com",
					TestGroup:         1,
					AdPodSlot:         0,
					FloorSkippedFlag:  ptrutil.ToPtr(0),
					FloorModelVersion: "test version",
					FloorSource:       ptrutil.ToPtr(1),
					FloorType:         1,
					RewardedInventory: 1,
					CustomDimensions:  "traffic=media;age=23",
					PartnerInfo: models.Partner{
						PartnerID:      "AppNexus",
						BidderCode:     "AppNexus1",
						BidID:          "6521",
						OrigBidID:      "6521",
						GrossECPM:      4.3,
						NetECPM:        2.5,
						KGPV:           "adunit@300x250",
						AdDuration:     10,
						Adformat:       models.Banner,
						AdSize:         "300x250",
						ServerSide:     1,
						Advertiser:     "fb.com",
						DealID:         "420",
						FloorValue:     4.4,
						FloorRuleValue: 2,
						PriceBucket:    "2.50",
					},
				},
			},
			want: "//t.pubmatic.com/wt?adv=fb.com&af=banner&aps=0&au=adunit&bc=AppNexus1&bidid=6521&cds=traffic=media;age=23&di=420&dur=10&eg=4.3&en=2.5&fmv=test version&frv=2&fskp=0&fsrc=1&ft=1&fv=4.4&iid=98765&kgpv=adunit@300x250&orig=www.publisher.com&origbidid=6521&pb=2.50&pdvid=1&pid=123&plt=1&pn=AppNexus&psz=300x250&pubid=12345&purl=www.abc.com&rwrd=1&sl=1&slot=1234_1234&ss=1&tgid=1&tst=0",
		},
		{
			name: "all_floors_details_in_tracker",
			args: args{
				rctx: models.RequestCtx{
					TrackerEndpoint: "//t.pubmatic.com/wt",
					Platform:        models.PLATFORM_APP,
				},
				tracker: models.Tracker{
					PubID:             12345,
					PageURL:           "www.abc.com",
					IID:               "98765",
					ProfileID:         "123",
					VersionID:         "1",
					SlotID:            "1234_1234",
					Adunit:            "adunit",
					Platform:          1,
					Origin:            "www.publisher.com",
					TestGroup:         1,
					AdPodSlot:         0,
					FloorSkippedFlag:  ptrutil.ToPtr(0),
					FloorModelVersion: "test version",
					FloorSource:       ptrutil.ToPtr(1),
					FloorType:         1,
					CustomDimensions:  "traffic=media;age=23",
					PartnerInfo: models.Partner{
						PartnerID:      "AppNexus",
						BidderCode:     "AppNexus1",
						BidID:          "6521",
						OrigBidID:      "6521",
						GrossECPM:      4.3,
						NetECPM:        2.5,
						KGPV:           "adunit@300x250",
						AdDuration:     10,
						Adformat:       models.Banner,
						AdSize:         "300x250",
						ServerSide:     1,
						Advertiser:     "fb.com",
						DealID:         "420",
						FloorValue:     4.4,
						FloorRuleValue: 2,
						PriceBucket:    "2.50",
					},
					DisplayManager:    "PubMatic_OpenWrap_SDK",
					DisplayManagerVer: "1.2",
				},
			},
			want: "//t.pubmatic.com/wt?adv=fb.com&af=banner&aps=0&au=adunit&bc=AppNexus1&bidid=6521&cds=traffic=media;age=23&di=420&dm=PubMatic_OpenWrap_SDK&dmv=1.2&dur=10&eg=4.3&en=2.5&fmv=test version&frv=2&fskp=0&fsrc=1&ft=1&fv=4.4&iid=98765&kgpv=adunit@300x250&orig=www.publisher.com&origbidid=6521&pb=2.50&pdvid=1&pid=123&plt=1&pn=AppNexus&psz=300x250&pubid=12345&purl=www.abc.com&sl=1&slot=1234_1234&ss=1&tgid=1&tst=0",
		},
		{
			name: "profileMetadata_details_updated_in_tracker",
			args: args{
				rctx: models.RequestCtx{
					TrackerEndpoint: "//t.pubmatic.com/wt",
					Platform:        models.PLATFORM_APP,
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
			},
			want: "//t.pubmatic.com/wt?adv=&af=&aip=3&ap=5&aps=0&asip=8&au=&bc=&bidid=&di=&eg=0&en=0&ft=0&iid=&kgpv=&orig=&origbidid=&pdvid=&pid=&plt=0&pn=&psz=&pt=1&ptp=4&pubid=0&purl=&sl=1&slot=&ss=0&tst=0",
		},
		{
			name: "all_floors_details_in_tracker_multiBidMultiFloor_enabled",
			args: args{
				rctx: models.RequestCtx{
					TrackerEndpoint: "//t.pubmatic.com/wt",
					Platform:        models.PLATFORM_APP,
				},
				tracker: models.Tracker{
					PubID:             12345,
					PageURL:           "www.abc.com",
					IID:               "98765",
					ProfileID:         "123",
					VersionID:         "1",
					SlotID:            "1234_1234",
					Adunit:            "adunit",
					Platform:          1,
					Origin:            "www.publisher.com",
					TestGroup:         1,
					AdPodSlot:         0,
					FloorSkippedFlag:  ptrutil.ToPtr(0),
					FloorModelVersion: "test version",
					FloorSource:       ptrutil.ToPtr(1),
					FloorType:         1,
					CustomDimensions:  "traffic=media;age=23",
					PartnerInfo: models.Partner{
						PartnerID:              "AppNexus",
						BidderCode:             "AppNexus1",
						BidID:                  "6521",
						OrigBidID:              "6521",
						GrossECPM:              4.3,
						NetECPM:                2.5,
						KGPV:                   "adunit@300x250",
						AdDuration:             10,
						Adformat:               models.Banner,
						AdSize:                 "300x250",
						ServerSide:             1,
						Advertiser:             "fb.com",
						DealID:                 "420",
						FloorValue:             4.5,
						FloorRuleValue:         4.5,
						PriceBucket:            "2.50",
						MultiBidMultiFloorFlag: 1,
					},
				},
			},
			want: "//t.pubmatic.com/wt?adv=fb.com&af=banner&aps=0&au=adunit&bc=AppNexus1&bidid=6521&cds=traffic=media;age=23&di=420&dur=10&eg=4.3&en=2.5&fmv=test version&frv=4.5&fskp=0&fsrc=1&ft=1&fv=4.5&iid=98765&kgpv=adunit@300x250&mbmf=1&orig=www.publisher.com&origbidid=6521&pb=2.50&pdvid=1&pid=123&plt=1&pn=AppNexus&psz=300x250&pubid=12345&purl=www.abc.com&sl=1&slot=1234_1234&ss=1&tgid=1&tst=0",
		},
		{
			name: "all_floors_details_in_tracker_with_NWID",
			args: args{
				rctx: models.RequestCtx{
					TrackerEndpoint: "//t.pubmatic.com/wt",
					Platform:        models.PLATFORM_APP,
				},
				tracker: models.Tracker{
					PubID:             12345,
					PageURL:           "www.abc.com",
					IID:               "98765",
					ProfileID:         "123",
					VersionID:         "1",
					SlotID:            "1234_1234",
					Adunit:            "adunit",
					Platform:          1,
					Origin:            "www.publisher.com",
					TestGroup:         1,
					AdPodSlot:         0,
					FloorSkippedFlag:  ptrutil.ToPtr(0),
					FloorModelVersion: "test version",
					FloorSource:       ptrutil.ToPtr(1),
					FloorType:         1,
					CustomDimensions:  "traffic=media;age=23",
					PartnerInfo: models.Partner{
						PartnerID:      "AppNexus",
						BidderCode:     "AppNexus1",
						BidID:          "6521",
						OrigBidID:      "6521",
						GrossECPM:      4.3,
						NetECPM:        2.5,
						KGPV:           "adunit@300x250",
						AdDuration:     10,
						Adformat:       models.Banner,
						AdSize:         "300x300",
						ServerSide:     1,
						Advertiser:     "fb.com",
						DealID:         "420",
						FloorValue:     4.4,
						FloorRuleValue: 2,
						PriceBucket:    "2.50",
						NetworkID:      987654,
					},
					DisplayManager:    "PubMatic_OpenWrap_SDK",
					DisplayManagerVer: "1.2",
				},
			},
			want: "//t.pubmatic.com/wt?adv=fb.com&af=banner&aps=0&au=adunit&bc=AppNexus1&bidid=6521&cds=traffic=media;age=23&di=420&dm=PubMatic_OpenWrap_SDK&dmv=1.2&dur=10&eg=4.3&en=2.5&fmv=test version&frv=2&fskp=0&fsrc=1&ft=1&fv=4.4&iid=98765&kgpv=adunit@300x250&nwid=987654&orig=www.publisher.com&origbidid=6521&pb=2.50&pdvid=1&pid=123&plt=1&pn=AppNexus&psz=300x300&pubid=12345&purl=www.abc.com&sl=1&slot=1234_1234&ss=1&tgid=1&tst=0",
		},
		{
			name: "tracker_with_vastUnwrap_enabled",
			args: args{
				rctx: models.RequestCtx{
					TrackerEndpoint: "//t.pubmatic.com/wt",
					Platform:        models.PLATFORM_APP,
					VastUnWrap: models.VastUnWrap{
						Enabled: true,
					},
				},
				tracker: models.Tracker{
					PubID:     12345,
					PageURL:   "www.abc.com",
					IID:       "98765",
					ProfileID: "123",
					VersionID: "1",
					SlotID:    "1234_1234",
					Adunit:    "adunit",
					Platform:  1,
					Origin:    "www.publisher.com",
					TestGroup: 1,
					PartnerInfo: models.Partner{
						PartnerID:  "AppNexus",
						BidderCode: "AppNexus1",
						BidID:      "6521",
						OrigBidID:  "6521",
						GrossECPM:  4.3,
						NetECPM:    2.5,
						KGPV:       "adunit@300x250",
						Adformat:   models.Video,
						AdSize:     "300x250",
						ServerSide: 1,
						Advertiser: "fb.com",
					},
					VastUnwrapEnabled: 1,
				},
			},
			want: "//t.pubmatic.com/wt?adv=fb.com&af=video&aps=0&au=adunit&bc=AppNexus1&bidid=6521&di=&eg=4.3&en=2.5&ft=0&iid=98765&kgpv=adunit@300x250&orig=www.publisher.com&origbidid=6521&pdvid=1&pid=123&plt=1&pn=AppNexus&psz=300x250&pubid=12345&purl=www.abc.com&sl=1&slot=1234_1234&ss=1&tgid=1&tst=0&vu=1",
		},
		{
			name: "tracker_with_vastUnwrap_disabled",
			args: args{
				rctx: models.RequestCtx{
					TrackerEndpoint: "//t.pubmatic.com/wt",
					Platform:        models.PLATFORM_APP,
					VastUnWrap: models.VastUnWrap{
						Enabled: false,
					},
				},
				tracker: models.Tracker{
					PubID:     12345,
					PageURL:   "www.abc.com",
					IID:       "98765",
					ProfileID: "123",
					VersionID: "1",
					SlotID:    "1234_1234",
					Adunit:    "adunit",
					Platform:  1,
					Origin:    "www.publisher.com",
					TestGroup: 1,
					PartnerInfo: models.Partner{
						PartnerID:  "AppNexus",
						BidderCode: "AppNexus1",
						BidID:      "6521",
						OrigBidID:  "6521",
						GrossECPM:  4.3,
						NetECPM:    2.5,
						KGPV:       "adunit@300x250",
						Adformat:   models.Video,
						AdSize:     "300x250",
						ServerSide: 1,
						Advertiser: "fb.com",
					},
					VastUnwrapEnabled: 0,
				},
			},
			want: "//t.pubmatic.com/wt?adv=fb.com&af=video&aps=0&au=adunit&bc=AppNexus1&bidid=6521&di=&eg=4.3&en=2.5&ft=0&iid=98765&kgpv=adunit@300x250&orig=www.publisher.com&origbidid=6521&pdvid=1&pid=123&plt=1&pn=AppNexus&psz=300x250&pubid=12345&purl=www.abc.com&sl=1&slot=1234_1234&ss=1&tgid=1&tst=0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trackerUrl := constructTrackerURL(tt.args.rctx, tt.args.tracker)
			decodedTrackerUrl, _ := url.QueryUnescape(trackerUrl)
			assert.Equal(t, tt.want, decodedTrackerUrl, tt.name)
		})
	}
}

func TestConstructVideoErrorURL(t *testing.T) {
	type args struct {
		rctx           models.RequestCtx
		errorURLString string
		bid            openrtb2.Bid
		tracker        models.Tracker
	}
	tests := []struct {
		name   string
		args   args
		want   string
		prefix string
	}{
		{
			name: "empty_urlString",
			args: args{
				rctx:           models.RequestCtx{},
				errorURLString: "",
				bid:            openrtb2.Bid{},
				tracker:        models.Tracker{},
			},
			want:   "",
			prefix: "",
		},
		{
			name: "invalid_urlString_with_parsing_error",
			args: args{
				rctx:           models.RequestCtx{},
				errorURLString: `:invalid_url`,
				bid:            openrtb2.Bid{},
				tracker:        models.Tracker{},
			},
			want:   "",
			prefix: "",
		},
		{
			name: "invalid_urlString_with_parsing",
			args: args{
				rctx:           models.RequestCtx{},
				errorURLString: `invalid_url`,
				bid:            openrtb2.Bid{},
				tracker:        models.Tracker{},
			},
			want:   "",
			prefix: "",
		},
		{
			name: "valid_video_errorUrl",
			args: args{
				rctx: models.RequestCtx{
					Origin: "domain.com:8080",
				},
				errorURLString: `//t.pubmatic.com/wt`,
				bid:            openrtb2.Bid{},
				tracker: models.Tracker{
					PubID:     12345,
					PageURL:   "www.abc.com",
					IID:       "98765",
					ProfileID: "123",
					VersionID: "1",
					SlotID:    "1234_1234",
					Adunit:    "adunit",
					Platform:  1,
					Origin:    "www.publisher.com",
					AdPodSlot: 0,
					SSAI:      "mediatailor",
					PartnerInfo: models.Partner{
						PartnerID:  "AppNexus",
						BidderCode: "AppNexus1",
						BidID:      "6521",
						OrigBidID:  "6521",
						GrossECPM:  4.3,
						NetECPM:    2.5,
						KGPV:       "adunit@300x250",
						AdDuration: 10,
						Adformat:   models.Banner,
						AdSize:     "300x250",
						ServerSide: 1,
						Advertiser: "fb.com",
						DealID:     "420",
					},
				},
			},
			want:   `https://t.pubmatic.com/wt?operId=8&crId=-1&p=12345&pid=123&pn=AppNexus&ts=0&v=1&ier=[ERRORCODE]&bc=AppNexus1&au=adunit&sURL=domain.com%253A8080&ssai=mediatailor&pfi=1&adv=fb.com`,
			prefix: `https://t.pubmatic.com/wt?operId=8`,
		},
		{
			name: "URL_with_Constant_Parameter",
			args: args{
				rctx: models.RequestCtx{
					Origin: "domain.com:8080",
				},
				errorURLString: `//t.pubmatic.com/wt?p1=v1&p2=v2`,
				bid:            openrtb2.Bid{},
				tracker: models.Tracker{
					PubID:     12345,
					PageURL:   "www.abc.com",
					IID:       "98765",
					ProfileID: "123",
					VersionID: "1",
					SlotID:    "1234_1234",
					Adunit:    "adunit",
					Platform:  1,
					Origin:    "www.publisher.com",
					AdPodSlot: 0,
					SSAI:      "mediatailor",
					PartnerInfo: models.Partner{
						PartnerID:  "AppNexus",
						BidderCode: "AppNexus1",
						BidID:      "6521",
						OrigBidID:  "6521",
						GrossECPM:  4.3,
						NetECPM:    2.5,
						KGPV:       "adunit@300x250",
						AdDuration: 10,
						Adformat:   models.Banner,
						AdSize:     "300x250",
						ServerSide: 1,
						Advertiser: "fb.com",
						DealID:     "420",
					},
				},
			},
			want:   `https://t.pubmatic.com/wt?operId=8&p1=v1&p2=v2&crId=-1&p=12345&pid=123&pn=AppNexus&ts=0&v=1&ier=[ERRORCODE]&bc=AppNexus1&au=adunit&sURL=domain.com%253A8080&ssai=mediatailor&pfi=1&adv=fb.com`,
			prefix: `https://t.pubmatic.com/wt?operId=8`,
		},
		{
			name: "Creative_ID_in_bid",
			args: args{
				rctx: models.RequestCtx{
					Origin: "domain.com:8080",
				},
				errorURLString: `//t.pubmatic.com/wt`,
				bid: openrtb2.Bid{
					CrID: "cr123",
				},
				tracker: models.Tracker{
					PubID:     12345,
					PageURL:   "www.abc.com",
					IID:       "98765",
					ProfileID: "123",
					VersionID: "1",
					SlotID:    "1234_1234",
					Adunit:    "adunit",
					Platform:  1,
					Origin:    "www.publisher.com",
					AdPodSlot: 0,
					SSAI:      "mediatailor",
					PartnerInfo: models.Partner{
						PartnerID:  "AppNexus",
						BidderCode: "AppNexus1",
						BidID:      "6521",
						OrigBidID:  "6521",
						GrossECPM:  4.3,
						NetECPM:    2.5,
						KGPV:       "adunit@300x250",
						AdDuration: 10,
						Adformat:   models.Banner,
						AdSize:     "300x250",
						ServerSide: 1,
						Advertiser: "fb.com",
						DealID:     "420",
					},
				},
			},
			want:   `https://t.pubmatic.com/wt?operId=8&crId=cr123&p=12345&pid=123&pn=AppNexus&ts=0&v=1&ier=[ERRORCODE]&bc=AppNexus1&au=adunit&sURL=domain.com%253A8080&ssai=mediatailor&pfi=1&adv=fb.com`,
			prefix: `https://t.pubmatic.com/wt?operId=8`,
		},
		{
			name: "URL_with_Schema",
			args: args{
				rctx: models.RequestCtx{
					Origin: "com.myapp.test",
				},
				errorURLString: `http://t.pubmatic.com/wt?p1=v1&p2=v2`,
				bid:            openrtb2.Bid{},
				tracker: models.Tracker{
					PubID:     12345,
					PageURL:   "www.abc.com",
					IID:       "98765",
					ProfileID: "123",
					VersionID: "1",
					SlotID:    "1234_1234",
					Adunit:    "adunit",
					Platform:  1,
					Origin:    "www.publisher.com",
					AdPodSlot: 0,
					SSAI:      "mediatailor",
					PartnerInfo: models.Partner{
						PartnerID:  "AppNexus",
						BidderCode: "AppNexus1",
						BidID:      "6521",
						OrigBidID:  "6521",
						GrossECPM:  4.3,
						NetECPM:    2.5,
						KGPV:       "adunit@300x250",
						AdDuration: 10,
						Adformat:   models.Banner,
						AdSize:     "300x250",
						ServerSide: 1,
						Advertiser: "fb.com",
						DealID:     "420",
					},
				},
			},
			want:   `https://t.pubmatic.com/wt?operId=8&p1=v1&p2=v2&crId=-1&p=12345&pid=123&pn=AppNexus&ts=0&v=1&ier=[ERRORCODE]&bc=AppNexus1&au=adunit&sURL=com.myapp.test&ssai=mediatailor&pfi=1&adv=fb.com`,
			prefix: `https://t.pubmatic.com/wt?operId=8`,
		},
		{
			name: "TestGroup_Enabled",
			args: args{
				rctx: models.RequestCtx{
					Origin: "com.myapp.test",
				},
				errorURLString: `http://t.pubmatic.com/wt?p1=v1&p2=v2`,
				bid:            openrtb2.Bid{},
				tracker: models.Tracker{
					PubID:     12345,
					PageURL:   "www.abc.com",
					IID:       "98765",
					ProfileID: "123",
					VersionID: "1",
					SlotID:    "1234_1234",
					Adunit:    "adunit",
					Platform:  1,
					Origin:    "www.publisher.com",
					AdPodSlot: 0,
					SSAI:      "mediatailor",
					TestGroup: 1,
					PartnerInfo: models.Partner{
						PartnerID:  "AppNexus",
						BidderCode: "AppNexus1",
						BidID:      "6521",
						OrigBidID:  "6521",
						GrossECPM:  4.3,
						NetECPM:    2.5,
						KGPV:       "adunit@300x250",
						AdDuration: 10,
						Adformat:   models.Banner,
						AdSize:     "300x250",
						ServerSide: 1,
						Advertiser: "fb.com",
						DealID:     "420",
					},
				},
			},
			want:   `https://t.pubmatic.com/wt?operId=8&p1=v1&p2=v2&crId=-1&p=12345&pid=123&pn=AppNexus&ts=0&v=1&ier=[ERRORCODE]&bc=AppNexus1&au=adunit&sURL=com.myapp.test&ssai=mediatailor&pfi=1&adv=fb.com&tgid=1`,
			prefix: `https://t.pubmatic.com/wt?operId=8`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constructedURL := constructVideoErrorURL(tt.args.rctx, tt.args.errorURLString, tt.args.bid, tt.args.tracker)
			if len(constructedURL) > 0 && len(tt.want) > 0 {
				wantURL, _ := url.Parse(constructedURL)
				expectedURL, _ := url.Parse(tt.want)
				if wantURL != nil && expectedURL != nil {
					assert.Equal(t, wantURL.Host, expectedURL.Host)
					assert.Equal(t, wantURL.Query(), expectedURL.Query())
					assert.Contains(t, constructedURL, tt.prefix)
				}
			}
		})
	}
}

func TestCreateTrackers(t *testing.T) {
	startTime := time.Now().Unix()
	type args struct {
		rctx                models.RequestCtx
		bidResponse         *openrtb2.BidResponse
		globalAccountConfig *config.Account
	}
	tests := []struct {
		name string
		args args
		want map[string]models.OWTracker
	}{
		{
			name: "overwrite marketplace bid details",
			args: args{
				rctx: func() models.RequestCtx {
					testRctx := rctx
					testRctx.StartTime = startTime
					return testRctx
				}(),
				bidResponse: &openrtb2.BidResponse{
					SeatBid: []openrtb2.SeatBid{
						{
							Bid: []openrtb2.Bid{
								{
									ID:      "bidID-1",
									ImpID:   "impID-1",
									Price:   8.7,
									W:       250,
									H:       300,
									ADomain: []string{"domain.com"},
									DealID:  "deal-id-1",
									Ext:     json.RawMessage(`{"prebid":{"meta":{"networkId":123456}}}`),
								},
							},
							Seat: "pubmatic",
						},
					},
					Cur: models.USD,
				},
				globalAccountConfig: &config.Account{BidRounding: config.DefaultBidRoundingMode},
			},
			want: map[string]models.OWTracker{
				"bidID-1": {
					Tracker: models.Tracker{
						PubID:            5890,
						PageURL:          "abc.com",
						Timestamp:        startTime,
						IID:              "loggerIID",
						ProfileID:        "1234",
						VersionID:        "1",
						Adunit:           "adunit-1",
						SlotID:           "impID-1_adunit-1",
						CustomDimensions: "author=henry",
						PartnerInfo: models.Partner{
							PartnerID:      "pubmatic",
							BidderCode:     "pubmatic",
							KGPV:           "adunit-1@250x300",
							GrossECPM:      8.7,
							NetECPM:        8.7,
							BidID:          "bidID-1",
							OrigBidID:      "bidID-1",
							AdSize:         "250x300",
							AdDuration:     20,
							Adformat:       "banner",
							ServerSide:     1,
							Advertiser:     "domain.com",
							FloorValue:     6.4,
							FloorRuleValue: 4.4,
							DealID:         "deal-id-1",
							NetworkID:      123456,
						},
						Platform:  5,
						SSAI:      "mediatailor",
						AdPodSlot: 0,
						TestGroup: 1,
						Origin:    "publisher.com",
						ImpID:     "impID-1",
						LoggerData: models.LoggerData{
							KGPSV: "adunit-1@250x300",
						},
					},
					TrackerURL:    "https:?adv=domain.com&af=banner&aps=0&au=adunit-1&bc=pubmatic&bidid=bidID-1&cds=author%3Dhenry&di=deal-id-1&dur=20&eg=8.7&en=8.7&frv=4.4&ft=0&fv=6.4&iid=loggerIID&kgpv=adunit-1%40250x300&nwid=123456&orig=publisher.com&origbidid=bidID-1&pdvid=1&pid=1234&plt=5&pn=pubmatic&psz=250x300&pubid=5890&purl=abc.com&sl=1&slot=impID-1_adunit-1&ss=1&ssai=mediatailor&tgid=1&tst=" + strconv.FormatInt(startTime, 10),
					Price:         8.7,
					PriceModel:    "CPM",
					PriceCurrency: "USD",
					BidType:       "banner",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CreateTrackers(tt.args.rctx, tt.args.bidResponse, tt.args.globalAccountConfig)
			assert.Equal(t, tt.want, got)
		})
	}
}
