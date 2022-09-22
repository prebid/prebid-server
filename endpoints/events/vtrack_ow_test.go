package events

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/beevik/etree"
	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/stretchr/testify/assert"
)

func TestInjectVideoEventTrackers(t *testing.T) {
	type args struct {
		externalURL string
		genbidID    string
		bid         *openrtb2.Bid
		req         *openrtb2.BidRequest
	}
	type want struct {
		eventURLs map[string][]string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "linear_creative",
			args: args{
				externalURL: "http://company.tracker.com?eventId=[EVENT_ID]&appbundle=[DOMAIN]",
				bid: &openrtb2.Bid{
					AdM: `<VAST version="3.0"><Ad><InLine><Creatives><Creative>
					                              <Linear>                      
					                                      <TrackingEvents>
					                                              <Tracking event="firstQuartile"><![CDATA[http://example.com/tracking/firstQuartile?k1=v1&k2=v2]]></Tracking>
					                                              <Tracking event="midpoint">http://example.com/tracking/midpoint</Tracking>
					                                              <Tracking event="thirdQuartile">http://example.com/tracking/thirdQuartile</Tracking>
																  <Tracking event="complete">http://example.com/tracking/complete</Tracking>
																  <Tracking event="start">http://partner.tracking.url</Tracking>
					                                      </TrackingEvents>
					                              </Linear>
					                     </Creative></Creatives></InLine></Ad></VAST>`,
				},
				req: &openrtb2.BidRequest{App: &openrtb2.App{Bundle: "abc"}},
			},
			want: want{
				eventURLs: map[string][]string{
					// "firstQuartile": {"http://example.com/tracking/firstQuartile?k1=v1&k2=v2", "http://company.tracker.com?eventId=1004&appbundle=abc"},
					// "midpoint":      {"http://example.com/tracking/midpoint", "http://company.tracker.com?eventId=1003&appbundle=abc"},
					// "thirdQuartile": {"http://example.com/tracking/thirdQuartile", "http://company.tracker.com?eventId=1005&appbundle=abc"},
					// "complete":      {"http://example.com/tracking/complete", "http://company.tracker.com?eventId=1006&appbundle=abc"},
					"firstQuartile": {"http://example.com/tracking/firstQuartile?k1=v1&k2=v2", "http://company.tracker.com?eventId=4&appbundle=abc"},
					"midpoint":      {"http://example.com/tracking/midpoint", "http://company.tracker.com?eventId=3&appbundle=abc"},
					"thirdQuartile": {"http://example.com/tracking/thirdQuartile", "http://company.tracker.com?eventId=5&appbundle=abc"},
					"complete":      {"http://example.com/tracking/complete", "http://company.tracker.com?eventId=6&appbundle=abc"},
					"start":         {"http://company.tracker.com?eventId=2&appbundle=abc", "http://partner.tracking.url"},
				},
			},
		},
		{
			name: "non_linear_creative",
			args: args{
				externalURL: "http://company.tracker.com?eventId=[EVENT_ID]&appbundle=[DOMAIN]",
				bid: &openrtb2.Bid{ // Adm contains to TrackingEvents tag
					AdM: `<VAST version="3.0"><Ad><InLine><Creatives><Creative>
				<NonLinearAds>
					<TrackingEvents>
					<Tracking event="firstQuartile">http://something.com</Tracking>
					</TrackingEvents>
				</NonLinearAds>
			</Creative></Creatives></InLine></Ad></VAST>`,
				},
				req: &openrtb2.BidRequest{App: &openrtb2.App{Bundle: "abc"}},
			},
			want: want{
				eventURLs: map[string][]string{
					// "firstQuartile": {"http://something.com", "http://company.tracker.com?eventId=1004&appbundle=abc"},
					// "midpoint":      {"http://company.tracker.com?eventId=1003&appbundle=abc"},
					// "thirdQuartile": {"http://company.tracker.com?eventId=1005&appbundle=abc"},
					// "complete":      {"http://company.tracker.com?eventId=1006&appbundle=abc"},
					"firstQuartile": {"http://something.com", "http://company.tracker.com?eventId=4&appbundle=abc"},
					"midpoint":      {"http://company.tracker.com?eventId=3&appbundle=abc"},
					"thirdQuartile": {"http://company.tracker.com?eventId=5&appbundle=abc"},
					"complete":      {"http://company.tracker.com?eventId=6&appbundle=abc"},
					"start":         {"http://company.tracker.com?eventId=2&appbundle=abc"},
				},
			},
		}, {
			name: "no_traker_url_configured", // expect no injection
			args: args{
				externalURL: "",
				bid: &openrtb2.Bid{ // Adm contains to TrackingEvents tag
					AdM: `<VAST version="3.0"><Ad><InLine><Creatives><Creative>
				<Linear>                      
				</Linear>
			</Creative></Creatives></InLine></Ad></VAST>`,
				},
				req: &openrtb2.BidRequest{App: &openrtb2.App{Bundle: "abc"}},
			},
			want: want{
				eventURLs: map[string][]string{},
			},
		},
		{
			name: "wrapper_vast_xml_from_partner", // expect we are injecting trackers inside wrapper
			args: args{
				externalURL: "http://company.tracker.com?eventId=[EVENT_ID]&appbundle=[DOMAIN]",
				bid: &openrtb2.Bid{ // Adm contains to TrackingEvents tag
					AdM: `<VAST version="4.2" xmlns="http://www.iab.com/VAST">
					<Ad id="20011" sequence="1" >
					  <Wrapper followAdditionalWrappers="0" allowMultipleAds="1" fallbackOnNoAd="0">
						<AdSystem version="4.0">iabtechlab</AdSystem>
					  <VASTAdTagURI>http://somevasturl</VASTAdTagURI>
						<Impression id="Impression-ID"><![CDATA[https://example.com/track/impression]]></Impression>
						<Creatives>
						  <Creative id="5480" sequence="1" adId="2447226">
							 <Linear></Linear>
						 </Creative>
				  </Creatives></Wrapper></Ad></VAST>`,
				},
				req: &openrtb2.BidRequest{App: &openrtb2.App{Bundle: "abc"}},
			},
			want: want{
				eventURLs: map[string][]string{
					// "firstQuartile": {"http://company.tracker.com?eventId=firstQuartile&appbundle=abc"},
					// "midpoint":      {"http://company.tracker.com?eventId=midpoint&appbundle=abc"},
					// "thirdQuartile": {"http://company.tracker.com?eventId=thirdQuartile&appbundle=abc"},
					// "complete":      {"http://company.tracker.com?eventId=complete&appbundle=abc"},
					"firstQuartile": {"http://company.tracker.com?eventId=4&appbundle=abc"},
					"midpoint":      {"http://company.tracker.com?eventId=3&appbundle=abc"},
					"thirdQuartile": {"http://company.tracker.com?eventId=5&appbundle=abc"},
					"complete":      {"http://company.tracker.com?eventId=6&appbundle=abc"},
					"start":         {"http://company.tracker.com?eventId=2&appbundle=abc"},
				},
			},
		},
		// {
		// 	name: "vast_tag_uri_response_from_partner",
		// 	args: args{
		// 		externalURL: "http://company.tracker.com?eventId=[EVENT_ID]&appbundle=[DOMAIN]",
		// 		bid: &openrtb2.Bid{ // Adm contains to TrackingEvents tag
		// 			AdM: `<![CDATA[http://hostedvasttag.url&k=v]]>`,
		// 		},
		// 		req: &openrtb2.BidRequest{App: &openrtb2.App{Bundle: "abc"}},
		// 	},
		// 	want: want{
		// 		eventURLs: map[string][]string{
		// 			"firstQuartile": {"http://company.tracker.com?eventId=firstQuartile&appbundle=abc"},
		// 			"midpoint":      {"http://company.tracker.com?eventId=midpoint&appbundle=abc"},
		// 			"thirdQuartile": {"http://company.tracker.com?eventId=thirdQuartile&appbundle=abc"},
		// 			"complete":      {"http://company.tracker.com?eventId=complete&appbundle=abc"},
		// 		},
		// 	},
		// },
		// {
		// 	name: "adm_empty",
		// 	args: args{
		// 		externalURL: "http://company.tracker.com?eventId=[EVENT_ID]&appbundle=[DOMAIN]",
		// 		bid: &openrtb2.Bid{ // Adm contains to TrackingEvents tag
		// 			AdM:  "",
		// 			NURL: "nurl_contents",
		// 		},
		// 		req: &openrtb2.BidRequest{App: &openrtb2.App{Bundle: "abc"}},
		// 	},
		// 	want: want{
		// 		eventURLs: map[string][]string{
		// 			"firstQuartile": {"http://company.tracker.com?eventId=firstQuartile&appbundle=abc"},
		// 			"midpoint":      {"http://company.tracker.com?eventId=midpoint&appbundle=abc"},
		// 			"thirdQuartile": {"http://company.tracker.com?eventId=thirdQuartile&appbundle=abc"},
		// 			"complete":      {"http://company.tracker.com?eventId=complete&appbundle=abc"},
		// 		},
		// 	},
		// },
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			vast := ""
			if nil != tc.args.bid {
				vast = tc.args.bid.AdM // original vast
			}
			// bind this bid id with imp object
			tc.args.req.Imp = []openrtb2.Imp{{ID: "123", Video: &openrtb2.Video{}}}
			tc.args.bid.ImpID = tc.args.req.Imp[0].ID
			accountID := ""
			timestamp := int64(0)
			requestingBidder := "test_bidder"
			bidderCoreName := "test_core_bidder"
			injectedVast, injected, ierr := InjectVideoEventTrackers(tc.args.externalURL, vast, tc.args.bid, tc.args.genbidID, requestingBidder, bidderCoreName, accountID, timestamp, tc.args.req)

			if !injected {
				// expect no change in input vast if tracking events are not injected
				assert.Equal(t, vast, string(injectedVast))
				assert.NotNil(t, ierr)
			} else {
				assert.Nil(t, ierr)
			}
			actualVastDoc := etree.NewDocument()

			err := actualVastDoc.ReadFromBytes(injectedVast)
			if nil != err {
				assert.Fail(t, err.Error())
			}

			// fmt.Println(string(injectedVast))
			actualTrackingEvents := actualVastDoc.FindElements("VAST/Ad/InLine/Creatives/Creative/Linear/TrackingEvents/Tracking")
			actualTrackingEvents = append(actualTrackingEvents, actualVastDoc.FindElements("VAST/Ad/InLine/Creatives/Creative/NonLinearAds/TrackingEvents/Tracking")...)
			actualTrackingEvents = append(actualTrackingEvents, actualVastDoc.FindElements("VAST/Ad/Wrapper/Creatives/Creative/Linear/TrackingEvents/Tracking")...)
			actualTrackingEvents = append(actualTrackingEvents, actualVastDoc.FindElements("VAST/Ad/Wrapper/Creatives/Creative/NonLinearAds/TrackingEvents/Tracking")...)

			totalURLCount := 0
			for event, URLs := range tc.want.eventURLs {

				for _, expectedURL := range URLs {
					present := false
					for _, te := range actualTrackingEvents {
						if te.SelectAttr("event").Value == event && te.Text() == expectedURL {
							present = true
							totalURLCount++
							break // expected URL present. check for next expected URL
						}
					}
					if !present {
						assert.Fail(t, "Expected tracker URL '"+expectedURL+"' is not present")
					}
				}
			}
			// ensure all total of events are injected
			assert.Equal(t, totalURLCount, len(actualTrackingEvents), fmt.Sprintf("Expected '%v' event trackers. But found '%v'", len(tc.want.eventURLs), len(actualTrackingEvents)))

		})
	}
}

func TestGetVideoEventTracking(t *testing.T) {
	type args struct {
		trackerURL       string
		bid              *openrtb2.Bid
		requestingBidder string
		gen_bidid        string
		bidderCoreName   string
		accountId        string
		timestamp        int64
		req              *openrtb2.BidRequest
		doc              *etree.Document
	}
	type want struct {
		trackerURLMap map[string]string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "valid_scenario",
			args: args{
				trackerURL: "http://company.tracker.com?eventId=[EVENT_ID]&appbundle=[DOMAIN]",
				bid:        &openrtb2.Bid{
					// AdM: vastXMLWith2Creatives,
				},
				req: &openrtb2.BidRequest{
					App: &openrtb2.App{
						Bundle: "someappbundle",
					},
					Imp: []openrtb2.Imp{},
				},
			},
			want: want{
				trackerURLMap: map[string]string{
					// "firstQuartile": "http://company.tracker.com?eventId=firstQuartile&appbundle=someappbundle",
					// "midpoint":      "http://company.tracker.com?eventId=midpoint&appbundle=someappbundle",
					// "thirdQuartile": "http://company.tracker.com?eventId=thirdQuartile&appbundle=someappbundle",
					// "complete":      "http://company.tracker.com?eventId=complete&appbundle=someappbundle"},
					"firstQuartile": "http://company.tracker.com?eventId=4&appbundle=someappbundle",
					"midpoint":      "http://company.tracker.com?eventId=3&appbundle=someappbundle",
					"thirdQuartile": "http://company.tracker.com?eventId=5&appbundle=someappbundle",
					"start":         "http://company.tracker.com?eventId=2&appbundle=someappbundle",
					"complete":      "http://company.tracker.com?eventId=6&appbundle=someappbundle"},
			},
		},
		{
			name: "no_macro_value", // expect no replacement
			args: args{
				trackerURL: "http://company.tracker.com?eventId=[EVENT_ID]&appbundle=[DOMAIN]",
				bid:        &openrtb2.Bid{},
				req: &openrtb2.BidRequest{
					App: &openrtb2.App{}, // no app bundle value
					Imp: []openrtb2.Imp{},
				},
			},
			want: want{
				trackerURLMap: map[string]string{
					// "firstQuartile": "http://company.tracker.com?eventId=firstQuartile&appbundle=[DOMAIN]",
					// "midpoint":      "http://company.tracker.com?eventId=midpoint&appbundle=[DOMAIN]",
					// "thirdQuartile": "http://company.tracker.com?eventId=thirdQuartile&appbundle=[DOMAIN]",
					// "complete":      "http://company.tracker.com?eventId=complete&appbundle=[DOMAIN]"},
					"firstQuartile": "http://company.tracker.com?eventId=4&appbundle=",
					"midpoint":      "http://company.tracker.com?eventId=3&appbundle=",
					"thirdQuartile": "http://company.tracker.com?eventId=5&appbundle=",
					"start":         "http://company.tracker.com?eventId=2&appbundle=",
					"complete":      "http://company.tracker.com?eventId=6&appbundle="},
			},
		},
		{
			name: "prefer_company_value_for_standard_macro",
			args: args{
				trackerURL: "http://company.tracker.com?eventId=[EVENT_ID]&appbundle=[DOMAIN]",
				req: &openrtb2.BidRequest{
					App: &openrtb2.App{
						Bundle: "myapp", // do not expect this value
					},
					Imp: []openrtb2.Imp{},
					Ext: []byte(`{"prebid":{
								"macros": {
									"[DOMAIN]": "my_custom_value"
								}
						}}`),
				},
			},
			want: want{
				trackerURLMap: map[string]string{
					// "firstQuartile": "http://company.tracker.com?eventId=firstQuartile&appbundle=my_custom_value",
					// "midpoint":      "http://company.tracker.com?eventId=midpoint&appbundle=my_custom_value",
					// "thirdQuartile": "http://company.tracker.com?eventId=thirdQuartile&appbundle=my_custom_value",
					// "complete":      "http://company.tracker.com?eventId=complete&appbundle=my_custom_value"},
					"firstQuartile": "http://company.tracker.com?eventId=4&appbundle=my_custom_value",
					"midpoint":      "http://company.tracker.com?eventId=3&appbundle=my_custom_value",
					"thirdQuartile": "http://company.tracker.com?eventId=5&appbundle=my_custom_value",
					"start":         "http://company.tracker.com?eventId=2&appbundle=my_custom_value",
					"complete":      "http://company.tracker.com?eventId=6&appbundle=my_custom_value"},
			},
		}, {
			name: "multireplace_macro",
			args: args{
				trackerURL: "http://company.tracker.com?eventId=[EVENT_ID]&appbundle=[DOMAIN]&parameter2=[DOMAIN]",
				req: &openrtb2.BidRequest{
					App: &openrtb2.App{
						Bundle: "myapp123",
					},
					Imp: []openrtb2.Imp{},
				},
			},
			want: want{
				trackerURLMap: map[string]string{
					// "firstQuartile": "http://company.tracker.com?eventId=firstQuartile&appbundle=myapp123&parameter2=myapp123",
					// "midpoint":      "http://company.tracker.com?eventId=midpoint&appbundle=myapp123&parameter2=myapp123",
					// "thirdQuartile": "http://company.tracker.com?eventId=thirdQuartile&appbundle=myapp123&parameter2=myapp123",
					// "complete":      "http://company.tracker.com?eventId=complete&appbundle=myapp123&parameter2=myapp123"},
					"firstQuartile": "http://company.tracker.com?eventId=4&appbundle=myapp123&parameter2=myapp123",
					"midpoint":      "http://company.tracker.com?eventId=3&appbundle=myapp123&parameter2=myapp123",
					"thirdQuartile": "http://company.tracker.com?eventId=5&appbundle=myapp123&parameter2=myapp123",
					"start":         "http://company.tracker.com?eventId=2&appbundle=myapp123&parameter2=myapp123",
					"complete":      "http://company.tracker.com?eventId=6&appbundle=myapp123&parameter2=myapp123"},
			},
		},
		{
			name: "custom_macro_without_prefix_and_suffix",
			args: args{
				trackerURL: "http://company.tracker.com?eventId=[EVENT_ID]&param1=[CUSTOM_MACRO]",
				req: &openrtb2.BidRequest{
					Ext: []byte(`{"prebid":{
							"macros": {
								"CUSTOM_MACRO": "my_custom_value"
							}
					}}`),
					Imp: []openrtb2.Imp{},
				},
			},
			want: want{
				trackerURLMap: map[string]string{
					// "firstQuartile": "http://company.tracker.com?eventId=firstQuartile&param1=[CUSTOM_MACRO]",
					// "midpoint":      "http://company.tracker.com?eventId=midpoint&param1=[CUSTOM_MACRO]",
					// "thirdQuartile": "http://company.tracker.com?eventId=thirdQuartile&param1=[CUSTOM_MACRO]",
					// "complete":      "http://company.tracker.com?eventId=complete&param1=[CUSTOM_MACRO]"},
					"firstQuartile": "http://company.tracker.com?eventId=4&param1=[CUSTOM_MACRO]",
					"midpoint":      "http://company.tracker.com?eventId=3&param1=[CUSTOM_MACRO]",
					"thirdQuartile": "http://company.tracker.com?eventId=5&param1=[CUSTOM_MACRO]",
					"start":         "http://company.tracker.com?eventId=2&param1=[CUSTOM_MACRO]",
					"complete":      "http://company.tracker.com?eventId=6&param1=[CUSTOM_MACRO]"},
			},
		},
		{
			name: "empty_macro",
			args: args{
				trackerURL: "http://company.tracker.com?eventId=[EVENT_ID]&param1=[CUSTOM_MACRO]",
				req: &openrtb2.BidRequest{
					Ext: []byte(`{"prebid":{
							"macros": {
								"": "my_custom_value"
							}
					}}`),
					Imp: []openrtb2.Imp{},
				},
			},
			want: want{
				trackerURLMap: map[string]string{
					// "firstQuartile": "http://company.tracker.com?eventId=firstQuartile&param1=[CUSTOM_MACRO]",
					// "midpoint":      "http://company.tracker.com?eventId=midpoint&param1=[CUSTOM_MACRO]",
					// "thirdQuartile": "http://company.tracker.com?eventId=thirdQuartile&param1=[CUSTOM_MACRO]",
					// "complete":      "http://company.tracker.com?eventId=complete&param1=[CUSTOM_MACRO]"},
					"firstQuartile": "http://company.tracker.com?eventId=4&param1=[CUSTOM_MACRO]",
					"midpoint":      "http://company.tracker.com?eventId=3&param1=[CUSTOM_MACRO]",
					"thirdQuartile": "http://company.tracker.com?eventId=5&param1=[CUSTOM_MACRO]",
					"start":         "http://company.tracker.com?eventId=2&param1=[CUSTOM_MACRO]",
					"complete":      "http://company.tracker.com?eventId=6&param1=[CUSTOM_MACRO]"},
			},
		},
		{
			name: "macro_is_case_sensitive",
			args: args{
				trackerURL: "http://company.tracker.com?eventId=[EVENT_ID]&param1=[CUSTOM_MACRO]",
				req: &openrtb2.BidRequest{
					Ext: []byte(`{"prebid":{
							"macros": {
								"": "my_custom_value"
							}
					}}`),
					Imp: []openrtb2.Imp{},
				},
			},
			want: want{
				trackerURLMap: map[string]string{
					// "firstQuartile": "http://company.tracker.com?eventId=firstQuartile&param1=[CUSTOM_MACRO]",
					// "midpoint":      "http://company.tracker.com?eventId=midpoint&param1=[CUSTOM_MACRO]",
					// "thirdQuartile": "http://company.tracker.com?eventId=thirdQuartile&param1=[CUSTOM_MACRO]",
					// "complete":      "http://company.tracker.com?eventId=complete&param1=[CUSTOM_MACRO]"},
					"firstQuartile": "http://company.tracker.com?eventId=4&param1=[CUSTOM_MACRO]",
					"midpoint":      "http://company.tracker.com?eventId=3&param1=[CUSTOM_MACRO]",
					"thirdQuartile": "http://company.tracker.com?eventId=5&param1=[CUSTOM_MACRO]",
					"start":         "http://company.tracker.com?eventId=2&param1=[CUSTOM_MACRO]",
					"complete":      "http://company.tracker.com?eventId=6&param1=[CUSTOM_MACRO]"},
			},
		},
		{
			name: "empty_tracker_url",
			args: args{trackerURL: "    ", req: &openrtb2.BidRequest{Imp: []openrtb2.Imp{}}},
			want: want{trackerURLMap: make(map[string]string)},
		},
		{
			name: "site_domain_tracker_url",
			args: args{trackerURL: "https://company.tracker.com?operId=8&e=[EVENT_ID]&p=[PBS-ACCOUNT]&pid=[PROFILE_ID]&v=[PROFILE_VERSION]&ts=[UNIX_TIMESTAMP]&pn=[PBS-BIDDER]&advertiser_id=[ADVERTISER_NAME]&sURL=[DOMAIN]&pfi=[PLATFORM]&af=[ADTYPE]&iid=[WRAPPER_IMPRESSION_ID]&pseq=[PODSEQUENCE]&adcnt=[ADCOUNT]&cb=[CACHEBUSTING]&au=[AD_UNIT]&bidid=[PBS-BIDID]",
				req: &openrtb2.BidRequest{Site: &openrtb2.Site{Name: "test", Domain: "www.test.com", Publisher: &openrtb2.Publisher{ID: "5890"}}, Imp: []openrtb2.Imp{}}},
			want: want{
				map[string]string{
					"complete":      "https://company.tracker.com?operId=8&e=6&p=5890&pid=[PROFILE_ID]&v=[PROFILE_VERSION]&ts=[UNIX_TIMESTAMP]&pn=&advertiser_id=&sURL=www.test.com&pfi=[PLATFORM]&af=video&iid=[WRAPPER_IMPRESSION_ID]&pseq=[PODSEQUENCE]&adcnt=[ADCOUNT]&cb=[CACHEBUSTING]&au=&bidid=",
					"firstQuartile": "https://company.tracker.com?operId=8&e=4&p=5890&pid=[PROFILE_ID]&v=[PROFILE_VERSION]&ts=[UNIX_TIMESTAMP]&pn=&advertiser_id=&sURL=www.test.com&pfi=[PLATFORM]&af=video&iid=[WRAPPER_IMPRESSION_ID]&pseq=[PODSEQUENCE]&adcnt=[ADCOUNT]&cb=[CACHEBUSTING]&au=&bidid=",
					"midpoint":      "https://company.tracker.com?operId=8&e=3&p=5890&pid=[PROFILE_ID]&v=[PROFILE_VERSION]&ts=[UNIX_TIMESTAMP]&pn=&advertiser_id=&sURL=www.test.com&pfi=[PLATFORM]&af=video&iid=[WRAPPER_IMPRESSION_ID]&pseq=[PODSEQUENCE]&adcnt=[ADCOUNT]&cb=[CACHEBUSTING]&au=&bidid=",
					"start":         "https://company.tracker.com?operId=8&e=2&p=5890&pid=[PROFILE_ID]&v=[PROFILE_VERSION]&ts=[UNIX_TIMESTAMP]&pn=&advertiser_id=&sURL=www.test.com&pfi=[PLATFORM]&af=video&iid=[WRAPPER_IMPRESSION_ID]&pseq=[PODSEQUENCE]&adcnt=[ADCOUNT]&cb=[CACHEBUSTING]&au=&bidid=",
					"thirdQuartile": "https://company.tracker.com?operId=8&e=5&p=5890&pid=[PROFILE_ID]&v=[PROFILE_VERSION]&ts=[UNIX_TIMESTAMP]&pn=&advertiser_id=&sURL=www.test.com&pfi=[PLATFORM]&af=video&iid=[WRAPPER_IMPRESSION_ID]&pseq=[PODSEQUENCE]&adcnt=[ADCOUNT]&cb=[CACHEBUSTING]&au=&bidid=",
				},
			},
		},
		{
			name: "site_page_tracker_url",
			args: args{trackerURL: "https://company.tracker.com?operId=8&e=[EVENT_ID]&p=[PBS-ACCOUNT]&pid=[PROFILE_ID]&v=[PROFILE_VERSION]&ts=[UNIX_TIMESTAMP]&pn=[PBS-BIDDER]&advertiser_id=[ADVERTISER_NAME]&sURL=[DOMAIN]&pfi=[PLATFORM]&af=[ADTYPE]&iid=[WRAPPER_IMPRESSION_ID]&pseq=[PODSEQUENCE]&adcnt=[ADCOUNT]&cb=[CACHEBUSTING]&au=[AD_UNIT]&bidid=[PBS-BIDID]",
				req: &openrtb2.BidRequest{Site: &openrtb2.Site{Name: "test", Page: "https://www.test.com/", Publisher: &openrtb2.Publisher{ID: "5890"}}, Imp: []openrtb2.Imp{}}},
			want: want{
				map[string]string{
					"complete":      "https://company.tracker.com?operId=8&e=6&p=5890&pid=[PROFILE_ID]&v=[PROFILE_VERSION]&ts=[UNIX_TIMESTAMP]&pn=&advertiser_id=&sURL=www.test.com&pfi=[PLATFORM]&af=video&iid=[WRAPPER_IMPRESSION_ID]&pseq=[PODSEQUENCE]&adcnt=[ADCOUNT]&cb=[CACHEBUSTING]&au=&bidid=",
					"firstQuartile": "https://company.tracker.com?operId=8&e=4&p=5890&pid=[PROFILE_ID]&v=[PROFILE_VERSION]&ts=[UNIX_TIMESTAMP]&pn=&advertiser_id=&sURL=www.test.com&pfi=[PLATFORM]&af=video&iid=[WRAPPER_IMPRESSION_ID]&pseq=[PODSEQUENCE]&adcnt=[ADCOUNT]&cb=[CACHEBUSTING]&au=&bidid=",
					"midpoint":      "https://company.tracker.com?operId=8&e=3&p=5890&pid=[PROFILE_ID]&v=[PROFILE_VERSION]&ts=[UNIX_TIMESTAMP]&pn=&advertiser_id=&sURL=www.test.com&pfi=[PLATFORM]&af=video&iid=[WRAPPER_IMPRESSION_ID]&pseq=[PODSEQUENCE]&adcnt=[ADCOUNT]&cb=[CACHEBUSTING]&au=&bidid=",
					"start":         "https://company.tracker.com?operId=8&e=2&p=5890&pid=[PROFILE_ID]&v=[PROFILE_VERSION]&ts=[UNIX_TIMESTAMP]&pn=&advertiser_id=&sURL=www.test.com&pfi=[PLATFORM]&af=video&iid=[WRAPPER_IMPRESSION_ID]&pseq=[PODSEQUENCE]&adcnt=[ADCOUNT]&cb=[CACHEBUSTING]&au=&bidid=",
					"thirdQuartile": "https://company.tracker.com?operId=8&e=5&p=5890&pid=[PROFILE_ID]&v=[PROFILE_VERSION]&ts=[UNIX_TIMESTAMP]&pn=&advertiser_id=&sURL=www.test.com&pfi=[PLATFORM]&af=video&iid=[WRAPPER_IMPRESSION_ID]&pseq=[PODSEQUENCE]&adcnt=[ADCOUNT]&cb=[CACHEBUSTING]&au=&bidid=",
				},
			},
		},
		{
			name: "all_macros with generated_bidId", // expect encoding for WRAPPER_IMPRESSION_ID macro
			args: args{
				trackerURL: "https://company.tracker.com?operId=8&e=[EVENT_ID]&p=[PBS-ACCOUNT]&pid=[PROFILE_ID]&v=[PROFILE_VERSION]&ts=[UNIX_TIMESTAMP]&pn=[PBS-BIDDER]&advertiser_id=[ADVERTISER_NAME]&sURL=[DOMAIN]&pfi=[PLATFORM]&af=[ADTYPE]&iid=[WRAPPER_IMPRESSION_ID]&pseq=[PODSEQUENCE]&adcnt=[ADCOUNT]&cb=[CACHEBUSTING]&au=[AD_UNIT]&bidid=[PBS-BIDID]&origbidid=[PBS-ORIG_BIDID]&bc=[BIDDER_CODE]",
				req: &openrtb2.BidRequest{
					App: &openrtb2.App{Bundle: "com.someapp.com", Publisher: &openrtb2.Publisher{ID: "5890"}},
					Ext: []byte(`{
						"prebid": {
								"macros": {
									"[PROFILE_ID]": "100",
									"[PROFILE_VERSION]": "2",
									"[UNIX_TIMESTAMP]": "1234567890",
									"[PLATFORM]": "7",
									"[WRAPPER_IMPRESSION_ID]": "abc~!@#$%^&&*()_+{}|:\"<>?[]\\;',./"
								}
						}
					}`),
					Imp: []openrtb2.Imp{
						{TagID: "/testadunit/1", ID: "imp_1"},
					},
				},
				bid:              &openrtb2.Bid{ADomain: []string{"http://a.com/32?k=v", "b.com"}, ImpID: "imp_1", ID: "test_bid_id"},
				gen_bidid:        "random_bid_id",
				requestingBidder: "test_bidder:234",
				bidderCoreName:   "test_core_bidder:234",
			},
			want: want{
				trackerURLMap: map[string]string{
					"firstQuartile": "https://company.tracker.com?operId=8&e=4&p=5890&pid=100&v=2&ts=1234567890&pn=test_core_bidder%3A234&advertiser_id=a.com&sURL=com.someapp.com&pfi=7&af=video&iid=abc~%21%40%23%24%25%5E%26%26%2A%28%29_%2B%7B%7D%7C%3A%22%3C%3E%3F%5B%5D%5C%3B%27%2C.%2F&pseq=[PODSEQUENCE]&adcnt=[ADCOUNT]&cb=[CACHEBUSTING]&au=%2Ftestadunit%2F1&bidid=random_bid_id&origbidid=test_bid_id&bc=test_bidder%3A234",
					"midpoint":      "https://company.tracker.com?operId=8&e=3&p=5890&pid=100&v=2&ts=1234567890&pn=test_core_bidder%3A234&advertiser_id=a.com&sURL=com.someapp.com&pfi=7&af=video&iid=abc~%21%40%23%24%25%5E%26%26%2A%28%29_%2B%7B%7D%7C%3A%22%3C%3E%3F%5B%5D%5C%3B%27%2C.%2F&pseq=[PODSEQUENCE]&adcnt=[ADCOUNT]&cb=[CACHEBUSTING]&au=%2Ftestadunit%2F1&bidid=random_bid_id&origbidid=test_bid_id&bc=test_bidder%3A234",
					"thirdQuartile": "https://company.tracker.com?operId=8&e=5&p=5890&pid=100&v=2&ts=1234567890&pn=test_core_bidder%3A234&advertiser_id=a.com&sURL=com.someapp.com&pfi=7&af=video&iid=abc~%21%40%23%24%25%5E%26%26%2A%28%29_%2B%7B%7D%7C%3A%22%3C%3E%3F%5B%5D%5C%3B%27%2C.%2F&pseq=[PODSEQUENCE]&adcnt=[ADCOUNT]&cb=[CACHEBUSTING]&au=%2Ftestadunit%2F1&bidid=random_bid_id&origbidid=test_bid_id&bc=test_bidder%3A234",
					"complete":      "https://company.tracker.com?operId=8&e=6&p=5890&pid=100&v=2&ts=1234567890&pn=test_core_bidder%3A234&advertiser_id=a.com&sURL=com.someapp.com&pfi=7&af=video&iid=abc~%21%40%23%24%25%5E%26%26%2A%28%29_%2B%7B%7D%7C%3A%22%3C%3E%3F%5B%5D%5C%3B%27%2C.%2F&pseq=[PODSEQUENCE]&adcnt=[ADCOUNT]&cb=[CACHEBUSTING]&au=%2Ftestadunit%2F1&bidid=random_bid_id&origbidid=test_bid_id&bc=test_bidder%3A234",
					"start":         "https://company.tracker.com?operId=8&e=2&p=5890&pid=100&v=2&ts=1234567890&pn=test_core_bidder%3A234&advertiser_id=a.com&sURL=com.someapp.com&pfi=7&af=video&iid=abc~%21%40%23%24%25%5E%26%26%2A%28%29_%2B%7B%7D%7C%3A%22%3C%3E%3F%5B%5D%5C%3B%27%2C.%2F&pseq=[PODSEQUENCE]&adcnt=[ADCOUNT]&cb=[CACHEBUSTING]&au=%2Ftestadunit%2F1&bidid=random_bid_id&origbidid=test_bid_id&bc=test_bidder%3A234"},
			},
		},
		{
			name: "all_macros with empty generated_bidId", // expect encoding for WRAPPER_IMPRESSION_ID macro
			args: args{
				trackerURL: "https://company.tracker.com?operId=8&e=[EVENT_ID]&p=[PBS-ACCOUNT]&pid=[PROFILE_ID]&v=[PROFILE_VERSION]&ts=[UNIX_TIMESTAMP]&pn=[PBS-BIDDER]&advertiser_id=[ADVERTISER_NAME]&sURL=[DOMAIN]&pfi=[PLATFORM]&af=[ADTYPE]&iid=[WRAPPER_IMPRESSION_ID]&pseq=[PODSEQUENCE]&adcnt=[ADCOUNT]&cb=[CACHEBUSTING]&au=[AD_UNIT]&bidid=[PBS-BIDID]&origbidid=[PBS-ORIG_BIDID]&bc=[BIDDER_CODE]",
				req: &openrtb2.BidRequest{
					App: &openrtb2.App{Bundle: "com.someapp.com", Publisher: &openrtb2.Publisher{ID: "5890"}},
					Ext: []byte(`{
						"prebid": {
								"macros": {
									"[PROFILE_ID]": "100",
									"[PROFILE_VERSION]": "2",
									"[UNIX_TIMESTAMP]": "1234567890",
									"[PLATFORM]": "7",
									"[WRAPPER_IMPRESSION_ID]": "abc~!@#$%^&&*()_+{}|:\"<>?[]\\;',./"
								}
						}
					}`),
					Imp: []openrtb2.Imp{
						{TagID: "/testadunit/1", ID: "imp_1"},
					},
				},
				bid:              &openrtb2.Bid{ADomain: []string{"http://a.com/32?k=v", "b.com"}, ImpID: "imp_1", ID: "test_bid_id"},
				gen_bidid:        "",
				requestingBidder: "test_bidder:234",
				bidderCoreName:   "test_core_bidder:234",
			},
			want: want{
				trackerURLMap: map[string]string{
					"firstQuartile": "https://company.tracker.com?operId=8&e=4&p=5890&pid=100&v=2&ts=1234567890&pn=test_core_bidder%3A234&advertiser_id=a.com&sURL=com.someapp.com&pfi=7&af=video&iid=abc~%21%40%23%24%25%5E%26%26%2A%28%29_%2B%7B%7D%7C%3A%22%3C%3E%3F%5B%5D%5C%3B%27%2C.%2F&pseq=[PODSEQUENCE]&adcnt=[ADCOUNT]&cb=[CACHEBUSTING]&au=%2Ftestadunit%2F1&bidid=test_bid_id&origbidid=test_bid_id&bc=test_bidder%3A234",
					"midpoint":      "https://company.tracker.com?operId=8&e=3&p=5890&pid=100&v=2&ts=1234567890&pn=test_core_bidder%3A234&advertiser_id=a.com&sURL=com.someapp.com&pfi=7&af=video&iid=abc~%21%40%23%24%25%5E%26%26%2A%28%29_%2B%7B%7D%7C%3A%22%3C%3E%3F%5B%5D%5C%3B%27%2C.%2F&pseq=[PODSEQUENCE]&adcnt=[ADCOUNT]&cb=[CACHEBUSTING]&au=%2Ftestadunit%2F1&bidid=test_bid_id&origbidid=test_bid_id&bc=test_bidder%3A234",
					"thirdQuartile": "https://company.tracker.com?operId=8&e=5&p=5890&pid=100&v=2&ts=1234567890&pn=test_core_bidder%3A234&advertiser_id=a.com&sURL=com.someapp.com&pfi=7&af=video&iid=abc~%21%40%23%24%25%5E%26%26%2A%28%29_%2B%7B%7D%7C%3A%22%3C%3E%3F%5B%5D%5C%3B%27%2C.%2F&pseq=[PODSEQUENCE]&adcnt=[ADCOUNT]&cb=[CACHEBUSTING]&au=%2Ftestadunit%2F1&bidid=test_bid_id&origbidid=test_bid_id&bc=test_bidder%3A234",
					"complete":      "https://company.tracker.com?operId=8&e=6&p=5890&pid=100&v=2&ts=1234567890&pn=test_core_bidder%3A234&advertiser_id=a.com&sURL=com.someapp.com&pfi=7&af=video&iid=abc~%21%40%23%24%25%5E%26%26%2A%28%29_%2B%7B%7D%7C%3A%22%3C%3E%3F%5B%5D%5C%3B%27%2C.%2F&pseq=[PODSEQUENCE]&adcnt=[ADCOUNT]&cb=[CACHEBUSTING]&au=%2Ftestadunit%2F1&bidid=test_bid_id&origbidid=test_bid_id&bc=test_bidder%3A234",
					"start":         "https://company.tracker.com?operId=8&e=2&p=5890&pid=100&v=2&ts=1234567890&pn=test_core_bidder%3A234&advertiser_id=a.com&sURL=com.someapp.com&pfi=7&af=video&iid=abc~%21%40%23%24%25%5E%26%26%2A%28%29_%2B%7B%7D%7C%3A%22%3C%3E%3F%5B%5D%5C%3B%27%2C.%2F&pseq=[PODSEQUENCE]&adcnt=[ADCOUNT]&cb=[CACHEBUSTING]&au=%2Ftestadunit%2F1&bidid=test_bid_id&origbidid=test_bid_id&bc=test_bidder%3A234"},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			if nil == tc.args.bid {
				tc.args.bid = &openrtb2.Bid{}
			}

			impMap := map[string]*openrtb2.Imp{}

			for _, imp := range tc.args.req.Imp {
				impMap[imp.ID] = &imp
			}

			eventURLMap := GetVideoEventTracking(tc.args.trackerURL, tc.args.bid, tc.args.gen_bidid, tc.args.requestingBidder, tc.args.bidderCoreName, tc.args.accountId, tc.args.timestamp, tc.args.req, tc.args.doc, impMap)

			for event, eurl := range tc.want.trackerURLMap {

				u, _ := url.Parse(eurl)
				expectedValues, _ := url.ParseQuery(u.RawQuery)
				u, _ = url.Parse(eventURLMap[event])
				actualValues, _ := url.ParseQuery(u.RawQuery)
				for k, ev := range expectedValues {
					av := actualValues[k]
					for i := 0; i < len(ev); i++ {
						assert.Equal(t, ev[i], av[i], fmt.Sprintf("Expected '%v' for '%v'. but found %v", ev[i], k, av[i]))
					}
				}

				// error out if extra query params
				if len(expectedValues) != len(actualValues) {
					assert.Equal(t, expectedValues, actualValues, fmt.Sprintf("Expected '%v' query params but found '%v'", len(expectedValues), len(actualValues)))
					break
				}
			}

			// check if new quartile pixels are covered inside test
			assert.Equal(t, tc.want.trackerURLMap, eventURLMap)
		})
	}
}

func TestReplaceMacro(t *testing.T) {
	type args struct {
		trackerURL string
		macro      string
		value      string
	}
	type want struct {
		trackerURL string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{name: "empty_tracker_url", args: args{trackerURL: "", macro: "[TEST]", value: "testme"}, want: want{trackerURL: ""}},
		{name: "tracker_url_with_macro", args: args{trackerURL: "http://something.com?test=[TEST]", macro: "[TEST]", value: "testme"}, want: want{trackerURL: "http://something.com?test=testme"}},
		{name: "tracker_url_with_invalid_macro", args: args{trackerURL: "http://something.com?test=TEST]", macro: "[TEST]", value: "testme"}, want: want{trackerURL: "http://something.com?test=TEST]"}},
		{name: "tracker_url_with_repeating_macro", args: args{trackerURL: "http://something.com?test=[TEST]&test1=[TEST]", macro: "[TEST]", value: "testme"}, want: want{trackerURL: "http://something.com?test=testme&test1=testme"}},
		{name: "empty_macro", args: args{trackerURL: "http://something.com?test=[TEST]", macro: "", value: "testme"}, want: want{trackerURL: "http://something.com?test=[TEST]"}},
		{name: "macro_without_[", args: args{trackerURL: "http://something.com?test=[TEST]", macro: "TEST]", value: "testme"}, want: want{trackerURL: "http://something.com?test=[TEST]"}},
		{name: "macro_without_]", args: args{trackerURL: "http://something.com?test=[TEST]", macro: "[TEST", value: "testme"}, want: want{trackerURL: "http://something.com?test=[TEST]"}},
		{name: "empty_value", args: args{trackerURL: "http://something.com?test=[TEST]", macro: "[TEST]", value: ""}, want: want{trackerURL: "http://something.com?test="}},
		{name: "nested_macro_value", args: args{trackerURL: "http://something.com?test=[TEST]", macro: "[TEST]", value: "[TEST][TEST]"}, want: want{trackerURL: "http://something.com?test=%5BTEST%5D%5BTEST%5D"}},
		{name: "url_as_macro_value", args: args{trackerURL: "http://something.com?test=[TEST]", macro: "[TEST]", value: "http://iamurl.com"}, want: want{trackerURL: "http://something.com?test=http%3A%2F%2Fiamurl.com"}},
		{name: "macro_with_spaces", args: args{trackerURL: "http://something.com?test=[TEST]", macro: "  [TEST]  ", value: "http://iamurl.com"}, want: want{trackerURL: "http://something.com?test=http%3A%2F%2Fiamurl.com"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			trackerURL := replaceMacro(tc.args.trackerURL, tc.args.macro, tc.args.value)
			assert.Equal(t, tc.want.trackerURL, trackerURL)
		})
	}

}
func TestExtractDomain(t *testing.T) {
	testCases := []struct {
		description    string
		url            string
		expectedDomain string
		expectedErr    error
	}{
		{description: "a.com", url: "a.com", expectedDomain: "a.com", expectedErr: nil},
		{description: "a.com/123", url: "a.com/123", expectedDomain: "a.com", expectedErr: nil},
		{description: "http://a.com/123", url: "http://a.com/123", expectedDomain: "a.com", expectedErr: nil},
		{description: "https://a.com/123", url: "https://a.com/123", expectedDomain: "a.com", expectedErr: nil},
		{description: "c.b.a.com", url: "c.b.a.com", expectedDomain: "c.b.a.com", expectedErr: nil},
		{description: "url_encoded_http://c.b.a.com", url: "http%3A%2F%2Fc.b.a.com", expectedDomain: "c.b.a.com", expectedErr: nil},
		{description: "url_encoded_with_www_http://c.b.a.com", url: "http%3A%2F%2Fwww.c.b.a.com", expectedDomain: "c.b.a.com", expectedErr: nil},
	}
	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			domain, err := extractDomain(test.url)
			assert.Equal(t, test.expectedDomain, domain)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}
