package openrtb2

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/beevik/etree"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/endpoints/openrtb2/ctv/constant"
	"github.com/prebid/prebid-server/endpoints/openrtb2/ctv/types"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

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

func TestFilterImpsVastTagsByDuration(t *testing.T) {
	type inputParams struct {
		request          *openrtb2.BidRequest
		generatedRequest *openrtb2.BidRequest
		impData          []*types.ImpData
	}

	type output struct {
		reqs        openrtb2.BidRequest
		blockedTags []map[string][]string
	}

	tt := []struct {
		testName       string
		input          inputParams
		expectedOutput output
	}{
		{
			testName: "test_single_impression_single_vast_partner_with_no_excluded_tags",
			input: inputParams{
				request: &openrtb2.BidRequest{
					Imp: []openrtb2.Imp{
						{ID: "imp1", Ext: []byte(`{"openx_vast_bidder":{"tags":[{"dur":35,"tagid":"openx_35"},{"dur":25,"tagid":"openx_25"},{"dur":20,"tagid":"openx_20"}]}}`)},
					},
				},
				generatedRequest: &openrtb2.BidRequest{
					Imp: []openrtb2.Imp{
						{ID: "imp1_1", Video: &openrtb2.Video{MinDuration: 10, MaxDuration: 10}},
						{ID: "imp1_2", Video: &openrtb2.Video{MinDuration: 10, MaxDuration: 20}},
						{ID: "imp1_3", Video: &openrtb2.Video{MinDuration: 25, MaxDuration: 35}},
					},
				},
				impData: []*types.ImpData{},
			},
			expectedOutput: output{
				reqs: openrtb2.BidRequest{
					Imp: []openrtb2.Imp{
						{ID: "imp1_1", Video: &openrtb2.Video{MinDuration: 10, MaxDuration: 10}, Ext: []byte(`{}`)},
						{ID: "imp1_2", Video: &openrtb2.Video{MinDuration: 10, MaxDuration: 20}, Ext: []byte(`{"openx_vast_bidder":{"tags":[{"dur":20,"tagid":"openx_20"}]}}`)},
						{ID: "imp1_3", Video: &openrtb2.Video{MinDuration: 25, MaxDuration: 35}, Ext: []byte(`{"openx_vast_bidder":{"tags":[{"dur":35,"tagid":"openx_35"},{"dur":25,"tagid":"openx_25"}]}}`)},
					},
				},
				blockedTags: []map[string][]string{},
			},
		},
		{
			testName: "test_single_impression_single_vast_partner_with_excluded_tags",
			input: inputParams{
				request: &openrtb2.BidRequest{
					Imp: []openrtb2.Imp{
						{ID: "imp1", Ext: []byte(`{"openx_vast_bidder":{"tags":[{"dur":35,"tagid":"openx_35"},{"dur":25,"tagid":"openx_25"},{"dur":20,"tagid":"openx_20"}]}}`)},
					},
				},
				generatedRequest: &openrtb2.BidRequest{
					Imp: []openrtb2.Imp{
						{ID: "imp1_1", Video: &openrtb2.Video{MinDuration: 10, MaxDuration: 10}},
						{ID: "imp1_2", Video: &openrtb2.Video{MinDuration: 10, MaxDuration: 20}},
						{ID: "imp1_3", Video: &openrtb2.Video{MinDuration: 25, MaxDuration: 30}},
					},
				},
				impData: []*types.ImpData{
					{ImpID: "imp1"},
				},
			},
			expectedOutput: output{
				reqs: openrtb2.BidRequest{
					Imp: []openrtb2.Imp{
						{ID: "imp1_1", Video: &openrtb2.Video{MinDuration: 10, MaxDuration: 10}, Ext: []byte(`{}`)},
						{ID: "imp1_2", Video: &openrtb2.Video{MinDuration: 10, MaxDuration: 20}, Ext: []byte(`{"openx_vast_bidder":{"tags":[{"dur":20,"tagid":"openx_20"}]}}`)},
						{ID: "imp1_3", Video: &openrtb2.Video{MinDuration: 25, MaxDuration: 30}, Ext: []byte(`{"openx_vast_bidder":{"tags":[{"dur":25,"tagid":"openx_25"}]}}`)},
					},
				},
				blockedTags: []map[string][]string{
					{"openx_vast_bidder": []string{"openx_35"}},
				},
			},
		},
		{
			testName: "test_single_impression_multiple_vast_partners_no_exclusions",
			input: inputParams{
				request: &openrtb2.BidRequest{
					Imp: []openrtb2.Imp{
						{ID: "imp1", Ext: []byte(`{"spotx_vast_bidder":{"tags":[{"dur":15,"tagid":"spotx_15"},{"dur":25,"tagid":"spotx_25"},{"dur":30,"tagid":"spotx_30"}]},"openx_vast_bidder":{"tags":[{"dur":35,"tagid":"openx_35"},{"dur":25,"tagid":"openx_25"},{"dur":20,"tagid":"openx_20"}]}}`)},
					},
				},
				generatedRequest: &openrtb2.BidRequest{
					Imp: []openrtb2.Imp{
						{ID: "imp1_1", Video: &openrtb2.Video{MinDuration: 10, MaxDuration: 10}},
						{ID: "imp1_2", Video: &openrtb2.Video{MinDuration: 10, MaxDuration: 20}},
						{ID: "imp1_3", Video: &openrtb2.Video{MinDuration: 25, MaxDuration: 30}},
					},
				},
				impData: []*types.ImpData{},
			},
			expectedOutput: output{
				reqs: openrtb2.BidRequest{
					Imp: []openrtb2.Imp{
						{ID: "imp1_1", Video: &openrtb2.Video{MinDuration: 10, MaxDuration: 10}, Ext: []byte(`{}`)},
						{ID: "imp1_2", Video: &openrtb2.Video{MinDuration: 10, MaxDuration: 20}, Ext: []byte(`{"openx_vast_bidder":{"tags":[{"dur":20,"tagid":"openx_20"}]},"spotx_vast_bidder":{"tags":[{"dur":15,"tagid":"spotx_15"}]}}`)},
						{ID: "imp1_3", Video: &openrtb2.Video{MinDuration: 25, MaxDuration: 30}, Ext: []byte(`{"openx_vast_bidder":{"tags":[{"dur":25,"tagid":"openx_25"}]},"spotx_vast_bidder":{"tags":[{"dur":25,"tagid":"spotx_25"},{"dur":30,"tagid":"spotx_30"}]}}`)},
					},
				},
				blockedTags: []map[string][]string{},
			},
		},
		{
			testName: "test_single_impression_multiple_vast_partners_with_exclusions",
			input: inputParams{
				request: &openrtb2.BidRequest{
					Imp: []openrtb2.Imp{
						{ID: "imp1", Ext: []byte(`{"spotx_vast_bidder":{"tags":[{"dur":15,"tagid":"spotx_15"},{"dur":25,"tagid":"spotx_25"},{"dur":35,"tagid":"spotx_35"}]},"openx_vast_bidder":{"tags":[{"dur":35,"tagid":"openx_35"},{"dur":25,"tagid":"openx_25"},{"dur":40,"tagid":"openx_40"}]}}`)},
					},
				},
				generatedRequest: &openrtb2.BidRequest{
					Imp: []openrtb2.Imp{
						{ID: "imp1_1", Video: &openrtb2.Video{MinDuration: 10, MaxDuration: 10}},
						{ID: "imp1_2", Video: &openrtb2.Video{MinDuration: 10, MaxDuration: 20}},
						{ID: "imp1_3", Video: &openrtb2.Video{MinDuration: 25, MaxDuration: 30}},
					},
				},
				impData: []*types.ImpData{
					{ImpID: "imp1"},
				},
			},
			expectedOutput: output{
				reqs: openrtb2.BidRequest{
					Imp: []openrtb2.Imp{
						{ID: "imp1_1", Video: &openrtb2.Video{MinDuration: 10, MaxDuration: 10}, Ext: []byte(`{}`)},
						{ID: "imp1_2", Video: &openrtb2.Video{MinDuration: 10, MaxDuration: 20}, Ext: []byte(`{"spotx_vast_bidder":{"tags":[{"dur":15,"tagid":"spotx_15"}]}}`)},
						{ID: "imp1_3", Video: &openrtb2.Video{MinDuration: 25, MaxDuration: 30}, Ext: []byte(`{"openx_vast_bidder":{"tags":[{"dur":25,"tagid":"openx_25"}]},"spotx_vast_bidder":{"tags":[{"dur":25,"tagid":"spotx_25"}]}}`)},
					},
				},
				blockedTags: []map[string][]string{
					{"openx_vast_bidder": []string{"openx_35", "openx_40"}, "spotx_vast_bidder": []string{"spotx_35"}},
				},
			},
		},
		{
			testName: "test_multi_impression_multi_partner_no_exclusions",
			input: inputParams{
				request: &openrtb2.BidRequest{
					Imp: []openrtb2.Imp{
						{ID: "imp1", Ext: []byte(`{"openx_vast_bidder":{"tags":[{"dur":35,"tagid":"openx_35"},{"dur":25,"tagid":"openx_25"},{"dur":20,"tagid":"openx_20"}]}}`)},
						{ID: "imp2", Ext: []byte(`{"spotx_vast_bidder":{"tags":[{"dur":30,"tagid":"spotx_30"},{"dur":40,"tagid":"spotx_40"}]}}`)},
					},
				},
				generatedRequest: &openrtb2.BidRequest{
					Imp: []openrtb2.Imp{
						{ID: "imp1_1", Video: &openrtb2.Video{MinDuration: 10, MaxDuration: 10}},
						{ID: "imp1_2", Video: &openrtb2.Video{MinDuration: 10, MaxDuration: 20}},
						{ID: "imp1_3", Video: &openrtb2.Video{MinDuration: 25, MaxDuration: 30}},
						{ID: "imp2_1", Video: &openrtb2.Video{MinDuration: 5, MaxDuration: 30}},
					},
				},
				impData: nil,
			},
			expectedOutput: output{
				reqs: openrtb2.BidRequest{
					Imp: []openrtb2.Imp{
						{ID: "imp1_1", Video: &openrtb2.Video{MinDuration: 10, MaxDuration: 10}, Ext: []byte(`{}`)},
						{ID: "imp1_2", Video: &openrtb2.Video{MinDuration: 10, MaxDuration: 20}, Ext: []byte(`{"openx_vast_bidder":{"tags":[{"dur":20,"tagid":"openx_20"}]}}`)},
						{ID: "imp1_3", Video: &openrtb2.Video{MinDuration: 25, MaxDuration: 30}, Ext: []byte(`{"openx_vast_bidder":{"tags":[{"dur":25,"tagid":"openx_25"}]}}`)},
						{ID: "imp2_1", Video: &openrtb2.Video{MinDuration: 5, MaxDuration: 30}, Ext: []byte(`{"spotx_vast_bidder":{"tags":[{"dur":30,"tagid":"spotx_30"}]}}`)},
					},
				},
				blockedTags: nil,
			},
		},
		{
			testName: "test_multi_impression_multi_partner_with_exclusions",
			input: inputParams{
				request: &openrtb2.BidRequest{
					Imp: []openrtb2.Imp{
						{ID: "imp1", Ext: []byte(`{"openx_vast_bidder":{"tags":[{"dur":35,"tagid":"openx_35"},{"dur":25,"tagid":"openx_25"},{"dur":20,"tagid":"openx_20"}]}}`)},
						{ID: "imp2", Ext: []byte(`{"spotx_vast_bidder":{"tags":[{"dur":30,"tagid":"spotx_30"},{"dur":40,"tagid":"spotx_40"}]}}`)},
					},
				},
				generatedRequest: &openrtb2.BidRequest{
					Imp: []openrtb2.Imp{
						{ID: "imp1_1", Video: &openrtb2.Video{MinDuration: 10, MaxDuration: 10}},
						{ID: "imp1_2", Video: &openrtb2.Video{MinDuration: 10, MaxDuration: 20}},
						{ID: "imp1_3", Video: &openrtb2.Video{MinDuration: 25, MaxDuration: 30}},
						{ID: "imp2_1", Video: &openrtb2.Video{MinDuration: 5, MaxDuration: 30}},
					},
				},
				impData: []*types.ImpData{
					{ImpID: "imp1"},
					{ImpID: "imp2"},
				},
			},
			expectedOutput: output{
				reqs: openrtb2.BidRequest{
					Imp: []openrtb2.Imp{
						{ID: "imp1_1", Video: &openrtb2.Video{MinDuration: 10, MaxDuration: 10}, Ext: []byte(`{}`)},
						{ID: "imp1_2", Video: &openrtb2.Video{MinDuration: 10, MaxDuration: 20}, Ext: []byte(`{"openx_vast_bidder":{"tags":[{"dur":20,"tagid":"openx_20"}]}}`)},
						{ID: "imp1_3", Video: &openrtb2.Video{MinDuration: 25, MaxDuration: 30}, Ext: []byte(`{"openx_vast_bidder":{"tags":[{"dur":25,"tagid":"openx_25"}]}}`)},
						{ID: "imp2_1", Video: &openrtb2.Video{MinDuration: 5, MaxDuration: 30}, Ext: []byte(`{"spotx_vast_bidder":{"tags":[{"dur":30,"tagid":"spotx_30"}]}}`)},
					},
				},
				blockedTags: []map[string][]string{
					{"openx_vast_bidder": []string{"openx_35"}},
					{"spotx_vast_bidder": []string{"spotx_40"}},
				},
			},
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()

			deps := ctvEndpointDeps{request: tc.input.request, impData: tc.input.impData}
			deps.readImpExtensionsAndTags()

			outputBids := tc.input.generatedRequest
			deps.filterImpsVastTagsByDuration(outputBids)

			assert.Equal(t, tc.expectedOutput.reqs, *outputBids, "Expected length of impressions array was %d but actual was %d", tc.expectedOutput.reqs, outputBids)

			for i, datum := range deps.impData {
				assert.Equal(t, tc.expectedOutput.blockedTags[i], datum.BlockedVASTTags, "Expected and actual impData was different")
			}
		})
	}
}

func TestGetBidDuration(t *testing.T) {
	type args struct {
		bid             *openrtb2.Bid
		reqExt          *openrtb_ext.ExtRequestAdPod
		config          []*types.ImpAdPodConfig
		defaultDuration int64
	}
	type want struct {
		duration int64
		status   constant.BidStatus
	}
	var tests = []struct {
		name   string
		args   args
		want   want
		expect int
	}{
		{
			name: "nil_bid_ext",
			args: args{
				bid:             &openrtb2.Bid{},
				reqExt:          nil,
				config:          nil,
				defaultDuration: 100,
			},
			want: want{
				duration: 100,
				status:   constant.StatusOK,
			},
		},
		{
			name: "use_default_duration",
			args: args{
				bid: &openrtb2.Bid{
					Ext: json.RawMessage(`{"tmp":123}`),
				},
				reqExt:          nil,
				config:          nil,
				defaultDuration: 100,
			},
			want: want{
				duration: 100,
				status:   constant.StatusOK,
			},
		},
		{
			name: "invalid_duration_in_bid_ext",
			args: args{
				bid: &openrtb2.Bid{
					Ext: json.RawMessage(`{"prebid":{"video":{"duration":"invalid"}}}`),
				},
				reqExt:          nil,
				config:          nil,
				defaultDuration: 100,
			},
			want: want{
				duration: 100,
				status:   constant.StatusOK,
			},
		},
		{
			name: "0sec_duration_in_bid_ext",
			args: args{
				bid: &openrtb2.Bid{
					Ext: json.RawMessage(`{"prebid":{"video":{"duration":0}}}`),
				},
				reqExt:          nil,
				config:          nil,
				defaultDuration: 100,
			},
			want: want{
				duration: 100,
				status:   constant.StatusOK,
			},
		},
		{
			name: "negative_duration_in_bid_ext",
			args: args{
				bid: &openrtb2.Bid{
					Ext: json.RawMessage(`{"prebid":{"video":{"duration":-30}}}`),
				},
				reqExt:          nil,
				config:          nil,
				defaultDuration: 100,
			},
			want: want{
				duration: 100,
				status:   constant.StatusOK,
			},
		},
		{
			name: "30sec_duration_in_bid_ext",
			args: args{
				bid: &openrtb2.Bid{
					Ext: json.RawMessage(`{"prebid":{"video":{"duration":30}}}`),
				},
				reqExt:          nil,
				config:          nil,
				defaultDuration: 100,
			},
			want: want{
				duration: 30,
				status:   constant.StatusOK,
			},
		},
		{
			name: "duration_matching_empty",
			args: args{
				bid: &openrtb2.Bid{
					Ext: json.RawMessage(`{"prebid":{"video":{"duration":30}}}`),
				},
				reqExt: &openrtb_ext.ExtRequestAdPod{
					VideoLengthMatching: "",
				},
				config:          nil,
				defaultDuration: 100,
			},
			want: want{
				duration: 30,
				status:   constant.StatusOK,
			},
		},
		{
			name: "duration_matching_exact",
			args: args{
				bid: &openrtb2.Bid{
					Ext: json.RawMessage(`{"prebid":{"video":{"duration":30}}}`),
				},
				reqExt: &openrtb_ext.ExtRequestAdPod{
					VideoLengthMatching: openrtb_ext.OWExactVideoLengthsMatching,
				},
				config: []*types.ImpAdPodConfig{
					{MaxDuration: 10},
					{MaxDuration: 20},
					{MaxDuration: 30},
					{MaxDuration: 40},
				},
				defaultDuration: 100,
			},
			want: want{
				duration: 30,
				status:   constant.StatusOK,
			},
		},
		{
			name: "duration_matching_exact_not_present",
			args: args{
				bid: &openrtb2.Bid{
					Ext: json.RawMessage(`{"prebid":{"video":{"duration":35}}}`),
				},
				reqExt: &openrtb_ext.ExtRequestAdPod{
					VideoLengthMatching: openrtb_ext.OWExactVideoLengthsMatching,
				},
				config: []*types.ImpAdPodConfig{
					{MaxDuration: 10},
					{MaxDuration: 20},
					{MaxDuration: 30},
					{MaxDuration: 40},
				},
				defaultDuration: 100,
			},
			want: want{
				duration: 35,
				status:   constant.StatusDurationMismatch,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration, status := getBidDuration(tt.args.bid, tt.args.reqExt, tt.args.config, tt.args.defaultDuration)
			assert.Equal(t, tt.want.duration, duration)
			assert.Equal(t, tt.want.status, status)
		})
	}
}

func Test_getDurationBasedOnDurationMatchingPolicy(t *testing.T) {
	type args struct {
		duration int64
		policy   openrtb_ext.OWVideoLengthMatchingPolicy
		config   []*types.ImpAdPodConfig
	}
	type want struct {
		duration int64
		status   constant.BidStatus
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "empty_duration_policy",
			args: args{
				duration: 10,
				policy:   "",
				config: []*types.ImpAdPodConfig{
					{MaxDuration: 10},
					{MaxDuration: 20},
					{MaxDuration: 30},
					{MaxDuration: 40},
				},
			},
			want: want{
				duration: 10,
				status:   constant.StatusOK,
			},
		},
		{
			name: "policy_exact",
			args: args{
				duration: 10,
				policy:   openrtb_ext.OWExactVideoLengthsMatching,
				config: []*types.ImpAdPodConfig{
					{MaxDuration: 10},
					{MaxDuration: 20},
					{MaxDuration: 30},
					{MaxDuration: 40},
				},
			},
			want: want{
				duration: 10,
				status:   constant.StatusOK,
			},
		},
		{
			name: "policy_exact_didnot_match",
			args: args{
				duration: 15,
				policy:   openrtb_ext.OWExactVideoLengthsMatching,
				config: []*types.ImpAdPodConfig{
					{MaxDuration: 10},
					{MaxDuration: 20},
					{MaxDuration: 30},
					{MaxDuration: 40},
				},
			},
			want: want{
				duration: 15,
				status:   constant.StatusDurationMismatch,
			},
		},
		{
			name: "policy_roundup_exact",
			args: args{
				duration: 20,
				policy:   openrtb_ext.OWRoundupVideoLengthMatching,
				config: []*types.ImpAdPodConfig{
					{MaxDuration: 10},
					{MaxDuration: 20},
					{MaxDuration: 30},
					{MaxDuration: 40},
				},
			},
			want: want{
				duration: 20,
				status:   constant.StatusOK,
			},
		},
		{
			name: "policy_roundup",
			args: args{
				duration: 25,
				policy:   openrtb_ext.OWRoundupVideoLengthMatching,
				config: []*types.ImpAdPodConfig{
					{MaxDuration: 10},
					{MaxDuration: 20},
					{MaxDuration: 30},
					{MaxDuration: 40},
				},
			},
			want: want{
				duration: 30,
				status:   constant.StatusOK,
			},
		},
		{
			name: "policy_roundup_didnot_match",
			args: args{
				duration: 45,
				policy:   openrtb_ext.OWRoundupVideoLengthMatching,
				config: []*types.ImpAdPodConfig{
					{MaxDuration: 10},
					{MaxDuration: 20},
					{MaxDuration: 30},
					{MaxDuration: 40},
				},
			},
			want: want{
				duration: 45,
				status:   constant.StatusDurationMismatch,
			},
		},

		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration, status := getDurationBasedOnDurationMatchingPolicy(tt.args.duration, tt.args.policy, tt.args.config)
			assert.Equal(t, tt.want.duration, duration)
			assert.Equal(t, tt.want.status, status)
		})
	}
}
