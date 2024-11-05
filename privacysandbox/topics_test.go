package privacysandbox

import (
	"encoding/json"
	"sort"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/stretchr/testify/assert"
)

func TestParseTopicsFromHeader(t *testing.T) {
	type args struct {
		secBrowsingTopics string
	}
	tests := []struct {
		name      string
		args      args
		wantTopic []Topic
		wantError []error
	}{
		{
			name:      "empty header",
			args:      args{secBrowsingTopics: "	 "},
			wantTopic: []Topic{},
			wantError: nil,
		},
		{
			name:      "invalid header value",
			args:      args{secBrowsingTopics: "some-sec-cookie-value"},
			wantTopic: []Topic{},
			wantError: []error{
				&errortypes.DebugWarning{
					Message:     "Invalid field in Sec-Browsing-Topics header: some-sec-cookie-value",
					WarningCode: errortypes.SecBrowsingTopicsWarningCode,
				},
			},
		},
		{
			name:      "header with only finish padding",
			args:      args{secBrowsingTopics: "();p=P0000000000000000000000000000000"},
			wantTopic: []Topic{},
			wantError: nil,
		},
		{
			name: "header with one valid field",
			args: args{secBrowsingTopics: "(1);v=chrome.1:1:2, ();p=P00000000000"},
			wantTopic: []Topic{
				{
					SegTax:   600,
					SegClass: "2",
					SegIDs:   []int{1},
				},
			},
			wantError: nil,
		},
		{
			name: "header without finish padding",
			args: args{secBrowsingTopics: "(1);v=chrome.1:1:2"},
			wantTopic: []Topic{
				{
					SegTax:   600,
					SegClass: "2",
					SegIDs:   []int{1},
				},
			},
			wantError: nil,
		},
		{
			name: "header with more than 10 valid field, should return only 10",
			args: args{secBrowsingTopics: "(1);v=chrome.1:1:2, (2);v=chrome.1:1:2, (3);v=chrome.1:1:2,  (4);v=chrome.1:1:2,  (5);v=chrome.1:1:2,  (6);v=chrome.1:1:2,  (7);v=chrome.1:1:2,  (8);v=chrome.1:1:2,  (9);v=chrome.1:1:2,  (10);v=chrome.1:1:2,  (11);v=chrome.1:1:2,  (12);v=chrome.1:1:2, ();p=P00000000000"},
			wantTopic: []Topic{
				{
					SegTax:   600,
					SegClass: "2",
					SegIDs:   []int{1},
				},
				{
					SegTax:   600,
					SegClass: "2",
					SegIDs:   []int{2},
				},
				{
					SegTax:   600,
					SegClass: "2",
					SegIDs:   []int{3},
				},
				{
					SegTax:   600,
					SegClass: "2",
					SegIDs:   []int{4},
				},
				{
					SegTax:   600,
					SegClass: "2",
					SegIDs:   []int{5},
				},
				{
					SegTax:   600,
					SegClass: "2",
					SegIDs:   []int{6},
				},
				{
					SegTax:   600,
					SegClass: "2",
					SegIDs:   []int{7},
				},
				{
					SegTax:   600,
					SegClass: "2",
					SegIDs:   []int{8},
				},
				{
					SegTax:   600,
					SegClass: "2",
					SegIDs:   []int{9},
				},
				{
					SegTax:   600,
					SegClass: "2",
					SegIDs:   []int{10},
				},
			},
			wantError: []error{
				&errortypes.DebugWarning{
					Message:     "Invalid field in Sec-Browsing-Topics header: (11);v=chrome.1:1:2 discarded due to limit reached.",
					WarningCode: errortypes.SecBrowsingTopicsWarningCode,
				},
				&errortypes.DebugWarning{
					Message:     "Invalid field in Sec-Browsing-Topics header: (12);v=chrome.1:1:2 discarded due to limit reached.",
					WarningCode: errortypes.SecBrowsingTopicsWarningCode,
				},
			},
		},
		{
			name: "header with one valid field having multiple segIDs",
			args: args{secBrowsingTopics: "(1 2);v=chrome.1:1:2, ();p=P00000000000"},
			wantTopic: []Topic{
				{
					SegTax:   600,
					SegClass: "2",
					SegIDs:   []int{1, 2},
				},
			},
			wantError: nil,
		},
		{
			name: "header with two valid fields having different taxonomies",
			args: args{secBrowsingTopics: "(1);v=chrome.1:1:2, (1);v=chrome.1:2:2, ();p=P0000000000"},
			wantTopic: []Topic{
				{
					SegTax:   600,
					SegClass: "2",
					SegIDs:   []int{1},
				},
				{
					SegTax:   601,
					SegClass: "2",
					SegIDs:   []int{1},
				},
			},
			wantError: nil,
		},
		{
			name: "header with one valid field and another invalid field (w/o segIDs), should return only one valid field",
			args: args{secBrowsingTopics: "(1);v=chrome.1:2:3, ();v=chrome.1:2:3, ();p=P0000000000"},
			wantTopic: []Topic{
				{
					SegTax:   601,
					SegClass: "3",
					SegIDs:   []int{1},
				},
			},
			wantError: []error{
				&errortypes.DebugWarning{
					Message:     "Invalid field in Sec-Browsing-Topics header: ();v=chrome.1:2:3",
					WarningCode: errortypes.SecBrowsingTopicsWarningCode,
				},
			},
		},
		{
			name: "header with two valid fields having different model version",
			args: args{secBrowsingTopics: "(1);v=chrome.1:2:3, (2);v=chrome.1:2:3, ();p=P0000000000"},
			wantTopic: []Topic{
				{
					SegTax:   601,
					SegClass: "3",
					SegIDs:   []int{1},
				},
				{
					SegTax:   601,
					SegClass: "3",
					SegIDs:   []int{2},
				},
			},
			wantError: nil,
		},
		{
			name: "header with one valid fields and two invalid fields (one with taxanomy < 0 and another with taxanomy > 10), should return only one valid field",
			args: args{secBrowsingTopics: "(1);v=chrome.1:11:2, (1);v=chrome.1:5:6, (1);v=chrome.1:0:2, ();p=P0000000000"},
			wantTopic: []Topic{
				{
					SegTax:   604,
					SegClass: "6",
					SegIDs:   []int{1},
				},
			},
			wantError: []error{
				&errortypes.DebugWarning{
					Message:     "Invalid field in Sec-Browsing-Topics header: (1);v=chrome.1:11:2",
					WarningCode: errortypes.SecBrowsingTopicsWarningCode,
				},
				&errortypes.DebugWarning{
					Message:     "Invalid field in Sec-Browsing-Topics header: (1);v=chrome.1:0:2",
					WarningCode: errortypes.SecBrowsingTopicsWarningCode,
				},
			},
		},
		{
			name: "header with with valid fields having special characters (whitespaces, etc)",
			args: args{secBrowsingTopics: "(1 2 4		6 7			4567	  ) ; v=chrome.1: 1 : 2, (1);v=chrome.1, ();p=P0000000000"},
			wantTopic: []Topic{
				{
					SegTax:   600,
					SegClass: "2",
					SegIDs:   []int{1, 2, 4, 6, 7, 4567},
				},
			},
			wantError: []error{
				&errortypes.DebugWarning{
					Message:     "Invalid field in Sec-Browsing-Topics header: (1);v=chrome.1",
					WarningCode: errortypes.SecBrowsingTopicsWarningCode,
				},
			},
		},
		{
			name:      "header with one valid field having a negative segId, drop field",
			args:      args{secBrowsingTopics: "(1 -3);v=chrome.1:1:2, ();p=P00000000000"},
			wantTopic: []Topic{},
			wantError: []error{
				&errortypes.DebugWarning{
					Message:     "Invalid field in Sec-Browsing-Topics header: (1 -3);v=chrome.1:1:2",
					WarningCode: errortypes.SecBrowsingTopicsWarningCode,
				},
			},
		},
		{
			name:      "header with one valid field having a segId=0, drop field",
			args:      args{secBrowsingTopics: "(1 0);v=chrome.1:1:2, ();p=P00000000000"},
			wantTopic: []Topic{},
			wantError: []error{
				&errortypes.DebugWarning{
					Message:     "Invalid field in Sec-Browsing-Topics header: (1 0);v=chrome.1:1:2",
					WarningCode: errortypes.SecBrowsingTopicsWarningCode,
				},
			},
		},
		{
			name:      "header with one valid field having a segId value more than MaxInt, drop field",
			args:      args{secBrowsingTopics: "(1 9223372036854775808);v=chrome.1:1:2, ();p=P00000000000"},
			wantTopic: []Topic{},
			wantError: []error{
				&errortypes.DebugWarning{
					Message:     "Invalid field in Sec-Browsing-Topics header: (1 9223372036854775808);v=chrome.1:1:2",
					WarningCode: errortypes.SecBrowsingTopicsWarningCode,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTopic, gotError := ParseTopicsFromHeader(tt.args.secBrowsingTopics)
			assert.Equal(t, tt.wantTopic, gotTopic)
			assert.Equal(t, tt.wantError, gotError)
		})
	}
}

func TestUpdateUserDataWithTopics(t *testing.T) {
	type args struct {
		userData     []openrtb2.Data
		headerData   []Topic
		topicsDomain string
	}
	tests := []struct {
		name string
		args args
		want []openrtb2.Data
	}{
		{
			name: "empty topics, empty user data, no change in user data",
			args: args{
				userData:   nil,
				headerData: nil,
			},
			want: nil,
		},
		{
			name: "empty topics, non-empty user data, no change in user data",
			args: args{
				userData: []openrtb2.Data{
					{
						ID:   "1",
						Name: "data1",
						Segment: []openrtb2.Segment{
							{ID: "1"},
							{ID: "2"},
						},
					},
				},
				headerData: nil,
			},
			want: []openrtb2.Data{
				{
					ID:   "1",
					Name: "data1",
					Segment: []openrtb2.Segment{
						{ID: "1"},
						{ID: "2"},
					},
				},
			},
		},
		{
			name: "topicsDomain empty, no change in user data",
			args: args{
				userData: []openrtb2.Data{
					{
						ID:   "1",
						Name: "data1",
						Segment: []openrtb2.Segment{
							{ID: "1"},
							{ID: "2"},
						},
					},
				},
				headerData: []Topic{
					{
						SegTax:   600,
						SegClass: "2",
						SegIDs:   []int{1, 2},
					},
				},
				topicsDomain: "",
			},
			want: []openrtb2.Data{
				{
					ID:   "1",
					Name: "data1",
					Segment: []openrtb2.Segment{
						{ID: "1"},
						{ID: "2"},
					},
				},
			},
		},
		{
			name: "non-empty topics, empty user data, topics from header copied to user data",
			args: args{
				userData: nil,
				headerData: []Topic{
					{
						SegTax:   600,
						SegClass: "2",
						SegIDs:   []int{1, 2},
					},
				},
				topicsDomain: "ads.pubmatic.com",
			},
			want: []openrtb2.Data{
				{
					Name: "ads.pubmatic.com",
					Segment: []openrtb2.Segment{
						{ID: "1"},
						{ID: "2"},
					},
					Ext: json.RawMessage(`{"segtax":600,"segclass":"2"}`),
				},
			},
		},
		{
			name: "non-empty topics, non-empty user data, topics from header copied to user data",
			args: args{
				userData: []openrtb2.Data{
					{
						ID:   "1",
						Name: "data1",
						Segment: []openrtb2.Segment{
							{ID: "1"},
							{ID: "2"},
						},
					},
				},
				headerData: []Topic{
					{
						SegTax:   600,
						SegClass: "2",
						SegIDs:   []int{3, 4},
					},
				},
				topicsDomain: "ads.pubmatic.com",
			},
			want: []openrtb2.Data{
				{
					ID:   "1",
					Name: "data1",
					Segment: []openrtb2.Segment{
						{ID: "1"},
						{ID: "2"},
					},
				},
				{
					Name: "ads.pubmatic.com",
					Segment: []openrtb2.Segment{
						{ID: "3"},
						{ID: "4"},
					},
					Ext: json.RawMessage(`{"segtax":600,"segclass":"2"}`),
				},
			},
		},
		{
			name: "non-empty topics, user data with invalid data.ext field, topics from header copied to user data",
			args: args{
				userData: []openrtb2.Data{
					{
						ID:   "1",
						Name: "data1",
						Segment: []openrtb2.Segment{
							{ID: "1"},
							{ID: "2"},
						},
						Ext: json.RawMessage(`{`),
					},
				},
				headerData: []Topic{
					{
						SegTax:   600,
						SegClass: "2",
						SegIDs:   []int{3, 4},
					},
				},
				topicsDomain: "ads.pubmatic.com",
			},
			want: []openrtb2.Data{
				{
					ID:   "1",
					Name: "data1",
					Segment: []openrtb2.Segment{
						{ID: "1"},
						{ID: "2"},
					},
					Ext: json.RawMessage(`{`),
				},
				{
					Name: "ads.pubmatic.com",
					Segment: []openrtb2.Segment{
						{ID: "3"},
						{ID: "4"},
					},
					Ext: json.RawMessage(`{"segtax":600,"segclass":"2"}`),
				},
			},
		},
		{
			name: "non-empty topics, user data with invalid topic details (invalid segtax and segclass), topics from header copied to user data",
			args: args{
				userData: []openrtb2.Data{
					{
						ID:   "1",
						Name: "chrome.com",
						Segment: []openrtb2.Segment{
							{ID: "1"},
							{ID: "2"},
						},
						Ext: json.RawMessage(`{"segtax":0,"segclass":""}`),
					},
				},
				headerData: []Topic{
					{
						SegTax:   600,
						SegClass: "2",
						SegIDs:   []int{3, 4},
					},
				},
				topicsDomain: "ads.pubmatic.com",
			},
			want: []openrtb2.Data{
				{
					ID:   "1",
					Name: "chrome.com",
					Segment: []openrtb2.Segment{
						{ID: "1"},
						{ID: "2"},
					},
					Ext: json.RawMessage(`{"segtax":0,"segclass":""}`),
				},
				{
					Name: "ads.pubmatic.com",
					Segment: []openrtb2.Segment{
						{ID: "3"},
						{ID: "4"},
					},
					Ext: json.RawMessage(`{"segtax":600,"segclass":"2"}`),
				},
			},
		},
		{
			name: "non-empty topics, user data with non matching topic details (different topicdomains, segtax and segclass), topics from header copied to user data",
			args: args{
				userData: []openrtb2.Data{
					{
						ID:   "1",
						Name: "chrome.com",
						Segment: []openrtb2.Segment{
							{ID: "1"},
							{ID: "2"},
						},
						Ext: json.RawMessage(`{"segtax":600,"segclass":"2"}`),
					},
					{
						ID:   "2",
						Name: "ads.pubmatic.com",
						Segment: []openrtb2.Segment{
							{ID: "5"},
							{ID: "6"},
						},
						Ext: json.RawMessage(`{"segtax":601,"segclass":"3"}`),
					},
					{
						ID:   "3",
						Name: "ads.pubmatic.com",
						Segment: []openrtb2.Segment{
							{ID: "7"},
							{ID: "8"},
						},
						Ext: json.RawMessage(`{"segtax":602,"segclass":"4"}`),
					},
				},
				headerData: []Topic{
					{
						SegTax:   600,
						SegClass: "2",
						SegIDs:   []int{3, 4},
					},
					{
						SegTax:   602,
						SegClass: "2",
						SegIDs:   []int{3, 4},
					},
				},
				topicsDomain: "ads.pubmatic.com",
			},
			want: []openrtb2.Data{
				{
					ID:   "1",
					Name: "chrome.com",
					Segment: []openrtb2.Segment{
						{ID: "1"},
						{ID: "2"},
					},
					Ext: json.RawMessage(`{"segtax":600,"segclass":"2"}`),
				},
				{
					ID:   "2",
					Name: "ads.pubmatic.com",
					Segment: []openrtb2.Segment{
						{ID: "5"},
						{ID: "6"},
					},
					Ext: json.RawMessage(`{"segtax":601,"segclass":"3"}`),
				},
				{
					ID:   "3",
					Name: "ads.pubmatic.com",
					Segment: []openrtb2.Segment{
						{ID: "7"},
						{ID: "8"},
					},
					Ext: json.RawMessage(`{"segtax":602,"segclass":"4"}`),
				},
				{
					Name: "ads.pubmatic.com",
					Segment: []openrtb2.Segment{
						{ID: "3"},
						{ID: "4"},
					},
					Ext: json.RawMessage(`{"segtax":600,"segclass":"2"}`),
				},
				{
					Name: "ads.pubmatic.com",
					Segment: []openrtb2.Segment{
						{ID: "3"},
						{ID: "4"},
					},
					Ext: json.RawMessage(`{"segtax":602,"segclass":"2"}`),
				},
			},
		},
		{
			name: "non-empty topics, user data with same topic details (matching segtax and segclass), topics from header merged with user data (filter unique segIDs)",
			args: args{
				userData: []openrtb2.Data{
					{
						ID:   "1",
						Name: "ads.pubmatic.com",
						Segment: []openrtb2.Segment{
							{ID: "1"},
							{ID: "2"},
							{ID: "3"},
						},
						Ext: json.RawMessage(`{"segtax":600,"segclass":"2"}`),
					},
				},
				headerData: []Topic{
					{
						SegTax:   600,
						SegClass: "2",
						SegIDs:   []int{2, 3, 4},
					},
				},
				topicsDomain: "ads.pubmatic.com",
			},
			want: []openrtb2.Data{
				{
					ID:   "1",
					Name: "ads.pubmatic.com",
					Segment: []openrtb2.Segment{
						{ID: "1"},
						{ID: "2"},
						{ID: "3"},
						{ID: "4"},
					},
					Ext: json.RawMessage(`{"segtax":600,"segclass":"2"}`),
				},
			},
		},
		{
			name: "non-empty topics, user data with duplicate topic details (matching segtax and segclass and segIDs), topics from header merged with user data (filter unique segIDs), user.data will not be deduped",
			args: args{
				userData: []openrtb2.Data{
					{
						ID:   "1",
						Name: "ads.pubmatic.com",
						Segment: []openrtb2.Segment{
							{ID: "1"},
							{ID: "2"},
							{ID: "3"},
						},
						Ext: json.RawMessage(`{"segtax":600,"segclass":"2"}`),
					},
					{
						ID:   "1",
						Name: "ads.pubmatic.com",
						Segment: []openrtb2.Segment{
							{ID: "1"},
							{ID: "2"},
							{ID: "3"},
						},
						Ext: json.RawMessage(`{"segtax":600,"segclass":"2"}`),
					},
				},
				headerData: []Topic{
					{
						SegTax:   600,
						SegClass: "2",
						SegIDs:   []int{2, 3, 4},
					},
				},
				topicsDomain: "ads.pubmatic.com",
			},
			want: []openrtb2.Data{
				{
					ID:   "1",
					Name: "ads.pubmatic.com",
					Segment: []openrtb2.Segment{
						{ID: "1"},
						{ID: "2"},
						{ID: "3"},
						{ID: "4"},
					},
					Ext: json.RawMessage(`{"segtax":600,"segclass":"2"}`),
				},
				{
					ID:   "1",
					Name: "ads.pubmatic.com",
					Segment: []openrtb2.Segment{
						{ID: "1"},
						{ID: "2"},
						{ID: "3"},
					},
					Ext: json.RawMessage(`{"segtax":600,"segclass":"2"}`),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UpdateUserDataWithTopics(tt.args.userData, tt.args.headerData, tt.args.topicsDomain)
			sort.Slice(got, func(i, j int) bool {
				if got[i].Name == got[j].Name {
					return string(got[i].Ext) < string(got[j].Ext)
				}
				return got[i].Name < got[j].Name
			})
			sort.Slice(tt.want, func(i, j int) bool {
				if tt.want[i].Name == tt.want[j].Name {
					return string(tt.want[i].Ext) < string(tt.want[j].Ext)
				}
				return tt.want[i].Name < tt.want[j].Name
			})

			for g := range got {
				sort.Slice(got[g].Segment, func(i, j int) bool {
					return got[g].Segment[i].ID < got[g].Segment[j].ID
				})
			}
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}
