package openrtb2

import (
	"encoding/json"
	"fmt"
	"github.com/PubMatic-OpenWrap/etree"
	"net/url"
	"strings"
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestGetAdDuration(t *testing.T) {
	var tests = []struct {
		scenario      string
		adDuration    string // actual ad duration. 0 value will be assumed as no ad duration
		maxAdDuration int    // requested max ad duration
		expect        int
	}{
		{"0sec ad duration", "0", 200, 200},
		{"30sec ad duration", "30", 100, 30},
		{"negative ad duration", "-30", 100, 100},
		{"invalid ad duration", "invalid", 80, 80},
		{"ad duration breaking bid.Ext json", `""quote""`, 50, 50},
	}
	for _, test := range tests {
		t.Run(test.scenario, func(t *testing.T) {
			bid := openrtb2.Bid{
				Ext: []byte(`{"prebid" : {"video" : {"duration" : ` + test.adDuration + `}}}`),
			}
			assert.Equal(t, test.expect, getAdDuration(bid, int64(test.maxAdDuration)))
		})
	}
}

func TestAddTargetingKeys(t *testing.T) {
	var tests = []struct {
		scenario string // Testcase scenario
		key      string
		value    string
		bidExt   string
		expect   map[string]string
	}{
		{scenario: "key_not_exists", key: "hb_pb_cat_dur", value: "some_value", bidExt: `{"prebid":{"targeting":{}}}`, expect: map[string]string{"hb_pb_cat_dur": "some_value"}},
		{scenario: "key_already_exists", key: "hb_pb_cat_dur", value: "new_value", bidExt: `{"prebid":{"targeting":{"hb_pb_cat_dur":"old_value"}}}`, expect: map[string]string{"hb_pb_cat_dur": "new_value"}},
	}
	for _, test := range tests {
		t.Run(test.scenario, func(t *testing.T) {
			bid := new(openrtb2.Bid)
			bid.Ext = []byte(test.bidExt)
			key := openrtb_ext.TargetingKey(test.key)
			assert.Nil(t, addTargetingKey(bid, key, test.value))
			extBid := openrtb_ext.ExtBid{}
			json.Unmarshal(bid.Ext, &extBid)
			assert.Equal(t, test.expect, extBid.Prebid.Targeting)
		})
	}
	assert.Equal(t, "Invalid bid", addTargetingKey(nil, openrtb_ext.HbCategoryDurationKey, "some value").Error())
}

func TestAdjustBidIDInVideoEventTrackers(t *testing.T) {
	type args struct {
		modifiedBid *openrtb2.Bid
	}
	type want struct {
		eventURLMap map[string]string
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "replace_with_custom_ctv_bid_id",
			want: want{
				eventURLMap: map[string]string{
					"thirdQuartile": "https://thirdQuartile.com?operId=8&key1=value1&bidid=1-bid_123",
					"complete":      "https://complete.com?operId=8&key1=value1&bidid=1-bid_123&key2=value2",
					"firstQuartile": "https://firstQuartile.com?operId=8&key1=value1&bidid=1-bid_123&key2=value2",
					"midpoint":      "https://midpoint.com?operId=8&key1=value1&bidid=1-bid_123&key2=value2",
					"someevent":     "https://othermacros?bidid=bid_123&abc=pqr",
				},
			},
			args: args{
				modifiedBid: &openrtb2.Bid{
					ID: "1-bid_123",
					AdM: `<VAST  version="3.0">
					<Ad>
						<Wrapper>
							<AdSystem>
								<![CDATA[prebid.org wrapper]]>
							</AdSystem>
							<VASTAdTagURI>
								<![CDATA[https://search.spotxchange.com/vast/2.00/85394?VPI=MP4]]>
							</VASTAdTagURI>
							<Impression>
								<![CDATA[https://imptracker.url]]>
							</Impression>
							<Impression/>
							<Creatives>
								<Creative>
									<Linear>
										<TrackingEvents>
											<Tracking  event="someevent"><![CDATA[https://othermacros?bidid=bid_123&abc=pqr]]></Tracking>
											<Tracking  event="thirdQuartile"><![CDATA[https://thirdQuartile.com?operId=8&key1=value1&bidid=bid_123]]></Tracking>
											<Tracking  event="complete"><![CDATA[https://complete.com?operId=8&key1=value1&bidid=bid_123&key2=value2]]></Tracking>
											<Tracking  event="firstQuartile"><![CDATA[https://firstQuartile.com?operId=8&key1=value1&bidid=bid_123&key2=value2]]></Tracking>
											<Tracking  event="midpoint"><![CDATA[https://midpoint.com?operId=8&key1=value1&bidid=bid_123&key2=value2]]></Tracking>
										</TrackingEvents>
									</Linear>
								</Creative>
							</Creatives>
							<Error>
								<![CDATA[https://error.com]]>
							</Error>
						</Wrapper>
					</Ad>
				</VAST>`,
				},
			},
		},
	}
	for _, test := range tests {
		doc := etree.NewDocument()
		doc.ReadFromString(test.args.modifiedBid.AdM)
		adjustBidIDInVideoEventTrackers(doc, test.args.modifiedBid)
		events := doc.FindElements("VAST/Ad/Wrapper/Creatives/Creative/Linear/TrackingEvents/Tracking")
		for _, event := range events {
			evntName := event.SelectAttr("event").Value
			expectedURL, _ := url.Parse(test.want.eventURLMap[evntName])
			expectedValues := expectedURL.Query()
			actualURL, _ := url.Parse(event.Text())
			actualValues := actualURL.Query()
			for k, ev := range expectedValues {
				av := actualValues[k]
				for i := 0; i < len(ev); i++ {
					assert.Equal(t, ev[i], av[i], fmt.Sprintf("Expected '%v' for '%v' [Event = %v]. but found %v", ev[i], k, evntName, av[i]))
				}
			}

			// check if operId=8 is first param
			if evntName != "someevent" {
				assert.True(t, strings.HasPrefix(actualURL.RawQuery, "operId=8"), "operId=8 must be first query param")
			}
		}
	}
}
