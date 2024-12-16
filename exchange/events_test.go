package exchange

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/exchange/entities"
	"github.com/prebid/prebid-server/v3/injector"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func Test_eventsData_makeBidExtEvents(t *testing.T) {
	type args struct {
		enabledForAccount bool
		enabledForRequest bool
		bidType           openrtb_ext.BidType
		generatedBidId    string
	}
	tests := []struct {
		name string
		args args
		want *openrtb_ext.ExtBidPrebidEvents
	}{
		{
			name: "banner: events enabled for request, disabled for account",
			args: args{enabledForAccount: false, enabledForRequest: true, bidType: openrtb_ext.BidTypeBanner, generatedBidId: ""},
			want: &openrtb_ext.ExtBidPrebidEvents{
				Win: "http://localhost/event?t=win&b=BID-1&a=123456&bidder=openx&ts=1234567890",
				Imp: "http://localhost/event?t=imp&b=BID-1&a=123456&bidder=openx&ts=1234567890",
			},
		},
		{
			name: "banner: events enabled for account, disabled for request",
			args: args{enabledForAccount: true, enabledForRequest: false, bidType: openrtb_ext.BidTypeBanner, generatedBidId: ""},
			want: &openrtb_ext.ExtBidPrebidEvents{
				Win: "http://localhost/event?t=win&b=BID-1&a=123456&bidder=openx&ts=1234567890",
				Imp: "http://localhost/event?t=imp&b=BID-1&a=123456&bidder=openx&ts=1234567890",
			},
		},
		{
			name: "banner: events disabled for account and request",
			args: args{enabledForAccount: false, enabledForRequest: false, bidType: openrtb_ext.BidTypeBanner, generatedBidId: ""},
			want: nil,
		},
		{
			name: "video: events enabled for account and request",
			args: args{enabledForAccount: true, enabledForRequest: true, bidType: openrtb_ext.BidTypeVideo, generatedBidId: ""},
			want: nil,
		},
		{
			name: "video: events disabled for account and request",
			args: args{enabledForAccount: false, enabledForRequest: false, bidType: openrtb_ext.BidTypeVideo, generatedBidId: ""},
			want: nil,
		},
		{
			name: "banner: use generated bid id",
			args: args{enabledForAccount: false, enabledForRequest: true, bidType: openrtb_ext.BidTypeBanner, generatedBidId: "randomId"},
			want: &openrtb_ext.ExtBidPrebidEvents{
				Win: "http://localhost/event?t=win&b=randomId&a=123456&bidder=openx&ts=1234567890",
				Imp: "http://localhost/event?t=imp&b=randomId&a=123456&bidder=openx&ts=1234567890",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evData := &eventTracking{
				enabledForAccount:  tt.args.enabledForAccount,
				enabledForRequest:  tt.args.enabledForRequest,
				accountID:          "123456",
				auctionTimestampMs: 1234567890,
				externalURL:        "http://localhost",
			}
			bid := &entities.PbsOrtbBid{Bid: &openrtb2.Bid{ID: "BID-1"}, BidType: tt.args.bidType, GeneratedBidID: tt.args.generatedBidId}
			assert.Equal(t, tt.want, evData.makeBidExtEvents(bid, openrtb_ext.BidderOpenx))
		})
	}
}

func Test_eventsData_modifyBidJSON(t *testing.T) {
	type args struct {
		enabledForAccount bool
		enabledForRequest bool
		bidType           openrtb_ext.BidType
		generatedBidId    string
	}
	tests := []struct {
		name      string
		args      args
		jsonBytes []byte
		want      []byte
	}{
		{
			name:      "banner: events enabled for request, disabled for account",
			args:      args{enabledForAccount: false, enabledForRequest: true, bidType: openrtb_ext.BidTypeBanner, generatedBidId: ""},
			jsonBytes: []byte(`{"ID": "something"}`),
			want:      []byte(`{"ID": "something", "wurl": "http://localhost/event?t=win&b=BID-1&a=123456&bidder=openx&ts=1234567890"}`),
		},
		{
			name:      "banner: events enabled for account, disabled for request",
			args:      args{enabledForAccount: true, enabledForRequest: false, bidType: openrtb_ext.BidTypeBanner, generatedBidId: ""},
			jsonBytes: []byte(`{"ID": "something"}`),
			want:      []byte(`{"ID": "something", "wurl": "http://localhost/event?t=win&b=BID-1&a=123456&bidder=openx&ts=1234567890"}`),
		},
		{
			name:      "banner: events disabled for account and request",
			args:      args{enabledForAccount: false, enabledForRequest: false, bidType: openrtb_ext.BidTypeBanner, generatedBidId: ""},
			jsonBytes: []byte(`{"ID": "something"}`),
			want:      []byte(`{"ID": "something"}`),
		},
		{
			name:      "video: events disabled for account and request",
			args:      args{enabledForAccount: false, enabledForRequest: false, bidType: openrtb_ext.BidTypeVideo, generatedBidId: ""},
			jsonBytes: []byte(`{"ID": "something"}`),
			want:      []byte(`{"ID": "something"}`),
		},
		{
			name:      "video: events enabled for account and request",
			args:      args{enabledForAccount: true, enabledForRequest: true, bidType: openrtb_ext.BidTypeVideo, generatedBidId: ""},
			jsonBytes: []byte(`{"ID": "something"}`),
			want:      []byte(`{"ID": "something"}`),
		},
		{
			name:      "banner: broken json expected to fail patching",
			args:      args{enabledForAccount: false, enabledForRequest: true, bidType: openrtb_ext.BidTypeBanner, generatedBidId: ""},
			jsonBytes: []byte(`broken json`),
			want:      nil,
		},
		{
			name:      "banner: generate bid id enabled",
			args:      args{enabledForAccount: false, enabledForRequest: true, bidType: openrtb_ext.BidTypeBanner, generatedBidId: "randomID"},
			jsonBytes: []byte(`{"ID": "something"}`),
			want:      []byte(`{"ID": "something", "wurl":"http://localhost/event?t=win&b=randomID&a=123456&bidder=openx&ts=1234567890"}`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evData := &eventTracking{
				enabledForAccount:  tt.args.enabledForAccount,
				enabledForRequest:  tt.args.enabledForRequest,
				accountID:          "123456",
				auctionTimestampMs: 1234567890,
				externalURL:        "http://localhost",
			}
			bid := &entities.PbsOrtbBid{Bid: &openrtb2.Bid{ID: "BID-1"}, BidType: tt.args.bidType, GeneratedBidID: tt.args.generatedBidId}
			modifiedJSON, err := evData.modifyBidJSON(bid, openrtb_ext.BidderOpenx, tt.jsonBytes)
			if tt.want != nil {
				assert.NoError(t, err, "Unexpected error")
				assert.JSONEq(t, string(tt.want), string(modifiedJSON))
			} else {
				assert.Error(t, err)
				assert.Equal(t, string(tt.jsonBytes), string(modifiedJSON), "Expected original json on failure to modify")
			}
		})
	}
}

func Test_isEventAllowed(t *testing.T) {
	type args struct {
		enabledForAccount bool
		enabledForRequest bool
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "enabled for account",
			args: args{enabledForAccount: true, enabledForRequest: false},
			want: true,
		},
		{
			name: "enabled for request",
			args: args{enabledForAccount: false, enabledForRequest: true},
			want: true,
		},
		{
			name: "disabled for account and request",
			args: args{enabledForAccount: false, enabledForRequest: false},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evData := &eventTracking{
				enabledForAccount: tt.args.enabledForAccount,
				enabledForRequest: tt.args.enabledForRequest,
			}
			isEventAllowed := evData.isEventAllowed()
			assert.Equal(t, tt.want, isEventAllowed)
		})
	}
}

func TestConvertToVastEvent(t *testing.T) {
	tests := []struct {
		name     string
		input    config.Events
		expected injector.VASTEvents
	}{
		{
			name: "Error_event",
			input: config.Events{
				DefaultURL: "http://default.url",
				VASTEvents: []config.VASTEvent{
					{
						CreateElement:     config.ErrorVASTElement,
						URLs:              []string{"http://error.url"},
						ExcludeDefaultURL: true,
					},
				},
			},
			expected: injector.VASTEvents{
				TrackingEvents: make(map[string][]string),
				Errors:         []string{"http://error.url"},
			},
		},
		{
			name: "NonLinearTracking_event",
			input: config.Events{
				DefaultURL: "http://default.url",
				VASTEvents: []config.VASTEvent{
					{
						CreateElement:     config.NonLinearClickTrackingVASTElement,
						URLs:              []string{"http://tracking.url"},
						ExcludeDefaultURL: true,
					},
				},
			},
			expected: injector.VASTEvents{
				TrackingEvents:         make(map[string][]string),
				NonLinearClickTracking: []string{"http://tracking.url"},
			},
		},
		{
			name: "CompanionClickThrough_event",
			input: config.Events{
				DefaultURL: "http://default.url",
				VASTEvents: []config.VASTEvent{
					{
						CreateElement:     config.CompanionClickThroughVASTElement,
						URLs:              []string{"http://tracking.url"},
						ExcludeDefaultURL: false,
					},
				},
			},
			expected: injector.VASTEvents{
				TrackingEvents:        make(map[string][]string),
				CompanionClickThrough: []string{"http://tracking.url", "http://default.url"},
			},
		},
		{
			name: "Tracking_event",
			input: config.Events{
				DefaultURL: "http://default.url",
				VASTEvents: []config.VASTEvent{
					{
						CreateElement: config.TrackingVASTElement,
						Type:          "start",
						URLs:          []string{"http://tracking.url"},
					},
				},
			},
			expected: injector.VASTEvents{
				TrackingEvents: map[string][]string{
					"start": {"http://tracking.url", "http://default.url"},
				},
			},
		},
		{
			name: "Clicktracking_event",
			input: config.Events{
				DefaultURL: "http://default.url",
				VASTEvents: []config.VASTEvent{
					{
						CreateElement: config.ClickTrackingVASTElement,
						URLs:          []string{"http://clicktracking.url"},
					},
				},
			},
			expected: injector.VASTEvents{
				TrackingEvents: make(map[string][]string),
				VideoClicks:    []string{"http://clicktracking.url", "http://default.url"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToVastEvent(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAppendURLs(t *testing.T) {
	tests := []struct {
		name         string
		urls         []string
		event        config.VASTEvent
		defaultURL   string
		expectedURLs []string
	}{
		{
			name: "urls_is_nil",
			urls: nil,
			event: config.VASTEvent{
				URLs:              []string{},
				ExcludeDefaultURL: false,
			},
			defaultURL:   "http://default.url",
			expectedURLs: []string{"http://default.url"},
		},
		{
			name: "events.url_is_nil",
			urls: []string{},
			event: config.VASTEvent{
				URLs:              nil,
				ExcludeDefaultURL: false,
			},
			defaultURL:   "http://default.url",
			expectedURLs: []string{"http://default.url"},
		},
		{
			name: "No_URLs_in_event,_include_default_URL",
			urls: []string{},
			event: config.VASTEvent{
				URLs:              []string{},
				ExcludeDefaultURL: false,
			},
			defaultURL:   "http://default.url",
			expectedURLs: []string{"http://default.url"},
		},
		{
			name: "No_URLs_in_event,_exclude_default_URL",
			urls: []string{},
			event: config.VASTEvent{
				URLs:              []string{},
				ExcludeDefaultURL: true,
			},
			defaultURL:   "http://default.url",
			expectedURLs: []string{},
		},
		{
			name: "URLs_in_event,_include_default_URL",
			urls: []string{},
			event: config.VASTEvent{
				URLs:              []string{"http://event.url"},
				ExcludeDefaultURL: false,
			},
			defaultURL:   "http://default.url",
			expectedURLs: []string{"http://event.url", "http://default.url"},
		},
		{
			name: "URLs_in_event,_exclude_default_URL",
			urls: []string{},
			event: config.VASTEvent{
				URLs:              []string{"http://event.url"},
				ExcludeDefaultURL: true,
			},
			defaultURL:   "http://default.url",
			expectedURLs: []string{"http://event.url"},
		},
		{
			name: "Existing_URLs,_URLs_in_event,_include_default_URL",
			urls: []string{"http://existing.url"},
			event: config.VASTEvent{
				URLs:              []string{"http://event.url"},
				ExcludeDefaultURL: false,
			},
			defaultURL:   "http://default.url",
			expectedURLs: []string{"http://existing.url", "http://event.url", "http://default.url"},
		},
		{
			name: "Existing_URLs,_URLs_in_event,_exclude_default_URL",
			urls: []string{"http://existing.url"},
			event: config.VASTEvent{
				URLs:              []string{"http://event.url"},
				ExcludeDefaultURL: true,
			},
			defaultURL:   "http://default.url",
			expectedURLs: []string{"http://existing.url", "http://event.url"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := appendURLs(tt.urls, tt.event, tt.defaultURL)
			assert.Equal(t, tt.expectedURLs, result)
		})
	}
}

func TestModifyBidVAST(t *testing.T) {
	tests := []struct {
		name           string
		ev             *eventTracking
		pbsBid         *entities.PbsOrtbBid
		bidderName     openrtb_ext.BidderName
		expectedPbsBid *entities.PbsOrtbBid
	}{
		{
			name: "Non-video bid type",
			ev: &eventTracking{
				externalURL:        "http://example.com",
				accountID:          "account1",
				auctionTimestampMs: 1234567890,
				integrationType:    "integration1",
				macroProvider: macros.NewProvider(&openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{},
				}),
				events: injector.VASTEvents{
					TrackingEvents:        make(map[string][]string),
					CompanionClickThrough: []string{"http://tracking.url", "http://default.url"},
				},
			},
			pbsBid: &entities.PbsOrtbBid{
				BidType: openrtb_ext.BidTypeBanner,
				Bid: &openrtb2.Bid{
					ID:  "bid1",
					AdM: "<Ad></Ad>",
				},
			},
			bidderName: "bidder1",
			expectedPbsBid: &entities.PbsOrtbBid{
				BidType: openrtb_ext.BidTypeBanner,
				Bid: &openrtb2.Bid{
					ID:  "bid1",
					AdM: "<Ad></Ad>",
				},
			},
		},
		{
			name: "Video bid type with No AdM",
			ev: &eventTracking{
				externalURL:        "http://example.com",
				accountID:          "account1",
				auctionTimestampMs: 1234567890,
				integrationType:    "integration1",
				macroProvider: macros.NewProvider(&openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{},
				}),
				events: injector.VASTEvents{
					TrackingEvents:        make(map[string][]string),
					CompanionClickThrough: []string{"http://tracking.url", "http://default.url"},
				},
			},
			pbsBid: &entities.PbsOrtbBid{
				BidType: openrtb_ext.BidTypeVideo,
				Bid: &openrtb2.Bid{
					ID:   "bid2",
					NURL: "http://nurl.com",
					AdM:  "",
				},
			},
			bidderName: "bidder2",
			expectedPbsBid: &entities.PbsOrtbBid{
				BidType: openrtb_ext.BidTypeVideo,
				Bid: &openrtb2.Bid{
					ID:   "bid2",
					NURL: "http://nurl.com",
					AdM:  "<VAST version=\"3.0\"><Ad><Wrapper><AdSystem>prebid.org wrapper</AdSystem><VASTAdTagURI><![CDATA[http://nurl.com]]></VASTAdTagURI><Impression><![CDATA[http://example.com/event?t=imp&b=bid2&a=account1&bidder=bidder2&f=b&int=integration1&ts=1234567890]]></Impression><Creatives></Creatives></Wrapper></Ad></VAST>",
				},
			},
		},
		{
			name: "Video bid type with AdM",
			ev: &eventTracking{
				externalURL:        "http://example.com",
				accountID:          "account1",
				auctionTimestampMs: 1234567890,
				integrationType:    "integration1",
				macroProvider: macros.NewProvider(&openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{},
				}),
				events: injector.VASTEvents{
					TrackingEvents:        make(map[string][]string),
					CompanionClickThrough: []string{"http://tracking.url", "http://default.url"},
				},
			},
			pbsBid: &entities.PbsOrtbBid{
				BidType: openrtb_ext.BidTypeVideo,
				Bid: &openrtb2.Bid{
					ID:   "bid2",
					NURL: "http://nurl.com",
					AdM:  `<VAST version="4.0" xmlns="http://www.iab.com/VAST"><Ad id="20011" sequence="1" conditionalAd="false"><Wrapper followAdditionalWrappers="0" allowMultipleAds="1" fallbackOnNoAd="0"><AdSystem version="4.0">iabtechlab</AdSystem><Error>http://example.com/error</Error><Impression id="Impression-ID">http://example.com/track/impression</Impression><Creatives><Creative id="5480" sequence="1" adId="2447226"><CompanionAds><Companion id="1232" width="100" height="150" assetWidth="250" assetHeight="200" expandedWidth="350" expandedHeight="250" apiFramework="VPAID" adSlotID="3214" pxratio="1400"><StaticResource creativeType="image/png"><![CDATA[https://www.iab.com/wp-content/uploads/2014/09/iab-tech-lab-6-644x290.png]]></StaticResource><CompanionClickThrough><![CDATA[https://iabtechlab.com]]></CompanionClickThrough></Companion></CompanionAds></Creative></Creatives><VASTAdTagURI><![CDATA[https://raw.githubusercontent.com/InteractiveAdvertisingBureau/VAST_Samples/master/VAST%204.0%20Samples/Inline_Companion_Tag-test.xml]]></VASTAdTagURI></Wrapper></Ad></VAST>`,
				},
			},
			bidderName: "bidder2",
			expectedPbsBid: &entities.PbsOrtbBid{
				BidType: openrtb_ext.BidTypeVideo,
				Bid: &openrtb2.Bid{
					ID:   "bid2",
					NURL: "http://nurl.com",
					AdM:  "<VAST version=\"4.0\" xmlns=\"http://www.iab.com/VAST\"><Ad id=\"20011\" sequence=\"1\" conditionalAd=\"false\"><Wrapper followAdditionalWrappers=\"0\" allowMultipleAds=\"1\" fallbackOnNoAd=\"0\"><AdSystem version=\"4.0\"><![CDATA[iabtechlab]]></AdSystem><Error><![CDATA[http://example.com/error]]></Error><Impression id=\"Impression-ID\"><![CDATA[http://example.com/track/impression]]></Impression><Impression><![CDATA[http://example.com/event?t=imp&b=bid2&a=account1&bidder=bidder2&f=b&int=integration1&ts=1234567890]]></Impression><Creatives><Creative id=\"5480\" sequence=\"1\" adId=\"2447226\"><CompanionAds><Companion id=\"1232\" width=\"100\" height=\"150\" assetWidth=\"250\" assetHeight=\"200\" expandedWidth=\"350\" expandedHeight=\"250\" apiFramework=\"VPAID\" adSlotID=\"3214\" pxratio=\"1400\"><StaticResource creativeType=\"image/png\"><![CDATA[https://www.iab.com/wp-content/uploads/2014/09/iab-tech-lab-6-644x290.png]]></StaticResource><CompanionClickThrough><![CDATA[https://iabtechlab.com]]></CompanionClickThrough><CompanionClickThrough><![CDATA[http://tracking.url]]></CompanionClickThrough><CompanionClickThrough><![CDATA[http://default.url]]></CompanionClickThrough></Companion></CompanionAds></Creative></Creatives><VASTAdTagURI><![CDATA[https://raw.githubusercontent.com/InteractiveAdvertisingBureau/VAST_Samples/master/VAST%204.0%20Samples/Inline_Companion_Tag-test.xml]]></VASTAdTagURI></Wrapper></Ad></VAST>",
				},
			},
		},
		{
			name: "Video bid type with AdM with generatedBIdID set",
			ev: &eventTracking{
				externalURL:        "http://example.com",
				accountID:          "account1",
				auctionTimestampMs: 1234567890,
				integrationType:    "integration1",
				macroProvider: macros.NewProvider(&openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{},
				}),
				events: injector.VASTEvents{
					TrackingEvents:        make(map[string][]string),
					CompanionClickThrough: []string{"http://tracking.url", "http://default.url"},
				},
			},
			pbsBid: &entities.PbsOrtbBid{
				BidType: openrtb_ext.BidTypeVideo,
				Bid: &openrtb2.Bid{
					ID:   "bid2",
					NURL: "http://nurl.com",
					AdM:  `<VAST version="4.0" xmlns="http://www.iab.com/VAST"><Ad id="20011" sequence="1" conditionalAd="false"><Wrapper followAdditionalWrappers="0" allowMultipleAds="1" fallbackOnNoAd="0"><AdSystem version="4.0">iabtechlab</AdSystem><Error>http://example.com/error</Error><Impression id="Impression-ID">http://example.com/track/impression</Impression><Creatives><Creative id="5480" sequence="1" adId="2447226"><CompanionAds><Companion id="1232" width="100" height="150" assetWidth="250" assetHeight="200" expandedWidth="350" expandedHeight="250" apiFramework="VPAID" adSlotID="3214" pxratio="1400"><StaticResource creativeType="image/png"><![CDATA[https://www.iab.com/wp-content/uploads/2014/09/iab-tech-lab-6-644x290.png]]></StaticResource><CompanionClickThrough><![CDATA[https://iabtechlab.com]]></CompanionClickThrough></Companion></CompanionAds></Creative></Creatives><VASTAdTagURI><![CDATA[https://raw.githubusercontent.com/InteractiveAdvertisingBureau/VAST_Samples/master/VAST%204.0%20Samples/Inline_Companion_Tag-test.xml]]></VASTAdTagURI></Wrapper></Ad></VAST>`,
				},
				GeneratedBidID: "generatedBidID",
			},
			bidderName: "bidder2",
			expectedPbsBid: &entities.PbsOrtbBid{
				BidType: openrtb_ext.BidTypeVideo,
				Bid: &openrtb2.Bid{
					ID:   "bid2",
					NURL: "http://nurl.com",
					AdM:  "<VAST version=\"4.0\" xmlns=\"http://www.iab.com/VAST\"><Ad id=\"20011\" sequence=\"1\" conditionalAd=\"false\"><Wrapper followAdditionalWrappers=\"0\" allowMultipleAds=\"1\" fallbackOnNoAd=\"0\"><AdSystem version=\"4.0\"><![CDATA[iabtechlab]]></AdSystem><Error><![CDATA[http://example.com/error]]></Error><Impression id=\"Impression-ID\"><![CDATA[http://example.com/track/impression]]></Impression><Impression><![CDATA[http://example.com/event?t=imp&b=generatedBidID&a=account1&bidder=bidder2&f=b&int=integration1&ts=1234567890]]></Impression><Creatives><Creative id=\"5480\" sequence=\"1\" adId=\"2447226\"><CompanionAds><Companion id=\"1232\" width=\"100\" height=\"150\" assetWidth=\"250\" assetHeight=\"200\" expandedWidth=\"350\" expandedHeight=\"250\" apiFramework=\"VPAID\" adSlotID=\"3214\" pxratio=\"1400\"><StaticResource creativeType=\"image/png\"><![CDATA[https://www.iab.com/wp-content/uploads/2014/09/iab-tech-lab-6-644x290.png]]></StaticResource><CompanionClickThrough><![CDATA[https://iabtechlab.com]]></CompanionClickThrough><CompanionClickThrough><![CDATA[http://tracking.url]]></CompanionClickThrough><CompanionClickThrough><![CDATA[http://default.url]]></CompanionClickThrough></Companion></CompanionAds></Creative></Creatives><VASTAdTagURI><![CDATA[https://raw.githubusercontent.com/InteractiveAdvertisingBureau/VAST_Samples/master/VAST%204.0%20Samples/Inline_Companion_Tag-test.xml]]></VASTAdTagURI></Wrapper></Ad></VAST>",
				},
				GeneratedBidID: "generatedBidID",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.ev.modifyBidVAST(tt.pbsBid, tt.bidderName)
			assert.Equal(t, tt.expectedPbsBid, tt.pbsBid)
		})
	}
}
