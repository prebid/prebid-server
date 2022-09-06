package floors

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestPrepareRuleCombinations(t *testing.T) {
	tt := []struct {
		name string
		in   []string
		n    int
		del  string
		out  []string
	}{
		{
			name: "Schema items, n = 1",
			in:   []string{"A"},
			n:    1,
			del:  "|",
			out: []string{
				"a",
				"*",
			},
		},
		{
			name: "Schema items, n = 2",
			in:   []string{"A", "B"},
			n:    2,
			del:  "|",
			out: []string{
				"a|b",
				"a|*",
				"*|b",
				"*|*",
			},
		},
		{
			name: "Schema items, n = 3",
			in:   []string{"A", "B", "C"},
			n:    3,
			del:  "|",
			out: []string{
				"a|b|c",
				"a|b|*",
				"a|*|c",
				"*|b|c",
				"a|*|*",
				"*|b|*",
				"*|*|c",
				"*|*|*",
			},
		},
		{
			name: "Schema items, n = 4",
			in:   []string{"A", "B", "C", "D"},
			n:    4,
			del:  "|",
			out: []string{
				"a|b|c|d",
				"a|b|c|*",
				"a|b|*|d",
				"a|*|c|d",
				"*|b|c|d",
				"a|b|*|*",
				"a|*|c|*",
				"a|*|*|d",
				"*|b|c|*",
				"*|b|*|d",
				"*|*|c|d",
				"a|*|*|*",
				"*|b|*|*",
				"*|*|c|*",
				"*|*|*|d",
				"*|*|*|*",
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			out := prepareRuleCombinations(tc.in, tc.n, tc.del)
			if !reflect.DeepEqual(out, tc.out) {
				t.Errorf("error: \nreturn:\t%v\nwant:\t%v", out, tc.out)
			}
		})
	}
}

func TestCreateRuleKeys(t *testing.T) {
	tt := []struct {
		name        string
		floorSchema openrtb_ext.PriceFloorSchema
		request     *openrtb2.BidRequest
		imp         openrtb2.Imp
		out         []string
	}{
		{
			name: "CreateRule with banner mediatype, size and domain",
			request: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Domain: "www.test.com",
				},
				Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}}},
				Ext: json.RawMessage(`{"prebid": { "floors": {"data": {"currency": "USD","skipRate": 0,"schema": {"fields": [ "mediaType", "size", "domain" ] },"values": {  "banner|300x250|www.website.com": 1.01, "banner|300x250|*": 2.01, "banner|300x600|www.website.com": 3.01,  "banner|300x600|*": 4.01, "banner|728x90|www.website.com": 5.01, "banner|728x90|*": 6.01, "banner|*|www.website.com": 7.01, "banner|*|*": 8.01, "*|300x250|www.website.com": 9.01, "*|300x250|*": 10.01, "*|300x600|www.website.com": 11.01,  "*|300x600|*": 12.01,  "*|728x90|www.website.com": 13.01, "*|728x90|*": 14.01,  "*|*|www.website.com": 15.01, "*|*|*": 16.01  }, "default": 1}}}}`),
			},
			floorSchema: openrtb_ext.PriceFloorSchema{Delimiter: "|", Fields: []string{"mediaType", "size", "domain"}},
			imp:         openrtb2.Imp{ID: "1234", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}},
			out:         []string{"banner", "300x250", "www.test.com"},
		},
		{
			name: "CreateRule with video mediatype, size and domain",
			request: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Domain: "www.test.com",
				},
				Imp: []openrtb2.Imp{{ID: "1234", Video: &openrtb2.Video{W: 640, H: 480}}},
				Ext: json.RawMessage(`{"prebid": { "floors": {"data": {"currency": "USD","skipRate": 0,"schema": {"fields": [ "mediaType", "size", "domain" ] },"values": {  "banner|300x250|www.website.com": 1.01, "banner|300x250|*": 2.01, "banner|300x600|www.website.com": 3.01,  "banner|300x600|*": 4.01, "banner|728x90|www.website.com": 5.01, "banner|728x90|*": 6.01, "banner|*|www.website.com": 7.01, "banner|*|*": 8.01, "*|300x250|www.website.com": 9.01, "*|300x250|*": 10.01, "*|300x600|www.website.com": 11.01,  "*|300x600|*": 12.01,  "*|728x90|www.website.com": 13.01, "*|728x90|*": 14.01,  "*|*|www.website.com": 15.01, "*|*|*": 16.01  }, "default": 1}}}}`),
			},
			floorSchema: openrtb_ext.PriceFloorSchema{Delimiter: "|", Fields: []string{"mediaType", "size", "domain"}},
			imp:         openrtb2.Imp{ID: "1234", Video: &openrtb2.Video{W: 640, H: 480}},
			out:         []string{"video", "640x480", "www.test.com"},
		},
		{
			name: "CreateRule with video mediatype, size and domain",
			request: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Domain: "www.test.com",
				},
				Imp: []openrtb2.Imp{{ID: "1234", Video: &openrtb2.Video{W: 300, H: 250}}},
				Ext: json.RawMessage(`{"prebid": { "floors": {"data": {"currency": "USD","skipRate": 0,"schema": {"fields": [ "mediaType", "size", "domain" ] },"values": {  "banner|300x250|www.website.com": 1.01, "banner|300x250|*": 2.01, "banner|300x600|www.website.com": 3.01,  "banner|300x600|*": 4.01, "banner|728x90|www.website.com": 5.01, "banner|728x90|*": 6.01, "banner|*|www.website.com": 7.01, "banner|*|*": 8.01, "*|300x250|www.website.com": 9.01, "*|300x250|*": 10.01, "*|300x600|www.website.com": 11.01,  "*|300x600|*": 12.01,  "*|728x90|www.website.com": 13.01, "*|728x90|*": 14.01,  "*|*|www.website.com": 15.01, "*|*|*": 16.01  }, "default": 1}}}}`),
			},
			floorSchema: openrtb_ext.PriceFloorSchema{Delimiter: "|", Fields: []string{"mediaType", "size", "domain"}},
			imp:         openrtb2.Imp{ID: "1234", Video: &openrtb2.Video{W: 300, H: 250}},
			out:         []string{"video", "300x250", "www.test.com"},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			out := createRuleKey(tc.floorSchema, tc.request, tc.imp)
			if !reflect.DeepEqual(out, tc.out) {
				t.Errorf("error: \nreturn:\t%v\nwant:\t%v", out, tc.out)
			}
		})
	}
}

func TestShouldSkipFloors(t *testing.T) {

	tt := []struct {
		name                string
		ModelGroupsSkipRate int
		DataSkipRate        int
		RootSkipRate        int
		out                 bool
		randomGen           func(int) int
	}{
		{
			name:                "ModelGroupsSkipRate=10 with skip = true",
			ModelGroupsSkipRate: 10,
			DataSkipRate:        0,
			RootSkipRate:        0,
			randomGen:           func(i int) int { return 5 },
			out:                 true,
		},
		{
			name:                "ModelGroupsSkipRate=100 with skip = true",
			ModelGroupsSkipRate: 100,
			DataSkipRate:        0,
			RootSkipRate:        0,
			randomGen:           func(i int) int { return 5 },
			out:                 true,
		},
		{
			name:                "ModelGroupsSkipRate=0 with skip = false",
			ModelGroupsSkipRate: 0,
			DataSkipRate:        0,
			RootSkipRate:        0,
			randomGen:           func(i int) int { return 5 },
			out:                 false,
		},
		{
			name:                "DataSkipRate=50  with with skip = true",
			ModelGroupsSkipRate: 0,
			DataSkipRate:        50,
			RootSkipRate:        0,
			randomGen:           func(i int) int { return 40 },
			out:                 true,
		},
		{
			name:                "RootSkipRate=50  with with skip = true",
			ModelGroupsSkipRate: 0,
			DataSkipRate:        0,
			RootSkipRate:        60,
			randomGen:           func(i int) int { return 40 },
			out:                 true,
		},
		{
			name:                "RootSkipRate=50  with with skip = false",
			ModelGroupsSkipRate: 0,
			DataSkipRate:        0,
			RootSkipRate:        60,
			randomGen:           func(i int) int { return 70 },
			out:                 false,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			out := shouldSkipFloors(tc.ModelGroupsSkipRate, tc.DataSkipRate, tc.RootSkipRate, tc.randomGen)
			if !reflect.DeepEqual(out, tc.out) {
				t.Errorf("error: \nreturn:\t%v\nwant:\t%v", out, tc.out)
			}
		})
	}

}

func TestSelectFloorModelGroup(t *testing.T) {
	floorExt := &openrtb_ext.PriceFloorRules{Data: &openrtb_ext.PriceFloorData{
		SkipRate: 30,
		ModelGroups: []openrtb_ext.PriceFloorModelGroup{{
			ModelWeight:  50,
			SkipRate:     10,
			ModelVersion: "Version 1",
			Schema:       openrtb_ext.PriceFloorSchema{Fields: []string{"mediaType", "size", "domain"}},
			Values: map[string]float64{
				"banner|300x250|www.website.com": 1.01,
				"banner|300x250|*":               2.01,
				"banner|300x600|www.website.com": 3.01,
				"banner|300x600|*":               4.01,
				"banner|728x90|www.website.com":  5.01,
				"banner|728x90|*":                6.01,
				"banner|*|www.website.com":       7.01,
				"banner|*|*":                     8.01,
				"*|300x250|www.website.com":      9.01,
				"*|300x250|*":                    10.01,
				"*|300x600|www.website.com":      11.01,
				"*|300x600|*":                    12.01,
				"*|728x90|www.website.com":       13.01,
				"*|728x90|*":                     14.01,
				"*|*|www.website.com":            15.01,
				"*|*|*":                          16.01,
			}, Default: 0.01},
			{
				ModelWeight:  25,
				SkipRate:     20,
				ModelVersion: "Version 2",
				Schema:       openrtb_ext.PriceFloorSchema{Fields: []string{"mediaType", "size", "domain"}},
				Values: map[string]float64{
					"banner|300x250|www.website.com": 1.01,
					"banner|300x250|*":               2.01,
					"banner|300x600|www.website.com": 3.01,
					"banner|300x600|*":               4.01,
					"banner|728x90|www.website.com":  5.01,
					"banner|728x90|*":                6.01,
					"banner|*|www.website.com":       7.01,
					"banner|*|*":                     8.01,
					"*|300x250|www.website.com":      9.01,
					"*|300x250|*":                    10.01,
					"*|300x600|www.website.com":      11.01,
					"*|300x600|*":                    12.01,
					"*|728x90|www.website.com":       13.01,
					"*|728x90|*":                     14.01,
					"*|*|www.website.com":            15.01,
					"*|*|*":                          16.01,
				}, Default: 0.01},
		}}}

	tt := []struct {
		name         string
		floorExt     *openrtb_ext.PriceFloorRules
		ModelVersion string
		fn           func(int) int
	}{
		{
			name:         "Version 2 Selection",
			floorExt:     floorExt,
			ModelVersion: "Version 2",
			fn:           func(i int) int { return 5 },
		},
		{
			name:         "Version 1 Selection",
			floorExt:     floorExt,
			ModelVersion: "Version 1",
			fn:           func(i int) int { return 55 },
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			selectFloorModelGroup(tc.floorExt.Data.ModelGroups, tc.fn)

			if !reflect.DeepEqual(tc.floorExt.Data.ModelGroups[0].ModelVersion, tc.ModelVersion) {
				t.Errorf("Floor Model Version mismatch error: \nreturn:\t%v\nwant:\t%v", tc.floorExt.Data.ModelGroups[0].ModelVersion, tc.ModelVersion)
			}

		})
	}
}
