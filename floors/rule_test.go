package floors

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/currency"
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

func TestUpdateImpExtWithFloorDetails(t *testing.T) {
	tt := []struct {
		name         string
		matchedRule  string
		floorRuleVal float64
		floorVal     float64
		imp          *openrtb_ext.ImpWrapper
		expected     json.RawMessage
	}{
		{
			name:         "Nil ImpExt",
			matchedRule:  "test|123|xyz",
			floorRuleVal: 5.5,
			floorVal:     5.5,
			imp:          &openrtb_ext.ImpWrapper{Imp: &openrtb2.Imp{ID: "1234", Video: &openrtb2.Video{W: 300, H: 250}}},
			expected:     []byte(`{"prebid":{"floors":{"floorRule":"test|123|xyz","floorRuleValue":5.5,"floorValue":5.5}}}`),
		},
		{
			name:         "Empty ImpExt",
			matchedRule:  "test|123|xyz",
			floorRuleVal: 5.5,
			floorVal:     5.5,
			imp:          &openrtb_ext.ImpWrapper{Imp: &openrtb2.Imp{ID: "1234", Video: &openrtb2.Video{W: 300, H: 250}, Ext: json.RawMessage{}}},
			expected:     []byte(`{"prebid":{"floors":{"floorRule":"test|123|xyz","floorRuleValue":5.5,"floorValue":5.5}}}`),
		},
		{
			name:         "With prebid Ext",
			matchedRule:  "banner|www.test.com|*",
			floorRuleVal: 5.5,
			floorVal:     15.5,
			imp:          &openrtb_ext.ImpWrapper{Imp: &openrtb2.Imp{ID: "1234", Video: &openrtb2.Video{W: 300, H: 250}, Ext: []byte(`{"prebid": {"test": true}}`)}},
			expected:     []byte(`{"prebid":{"floors":{"floorRule":"banner|www.test.com|*","floorRuleValue":5.5,"floorValue":15.5}}}`),
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			updateImpExtWithFloorDetails(tc.imp, tc.matchedRule, tc.floorRuleVal, tc.floorVal)
			if tc.imp.Ext != nil && !reflect.DeepEqual(tc.imp.Ext, tc.expected) {
				t.Errorf("error: \nreturn:\t%v\n want:\t%v", string(tc.imp.Ext), string(tc.expected))
			}
		})
	}
}

func TestCreateRuleKeys(t *testing.T) {
	tt := []struct {
		name        string
		floorSchema openrtb_ext.PriceFloorSchema
		request     *openrtb2.BidRequest
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
			out:         []string{"banner", "300x250", "www.test.com"},
		},
		{
			name: "CreateRule with video mediatype, size and domain",
			request: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Domain: "www.test.com",
				},
				Imp: []openrtb2.Imp{{ID: "1234", Video: &openrtb2.Video{W: 640, H: 480, Placement: 1}}},
			},
			floorSchema: openrtb_ext.PriceFloorSchema{Delimiter: "|", Fields: []string{"mediaType", "size", "domain"}},
			out:         []string{"video-instream", "640x480", "www.test.com"},
		},
		{
			name: "CreateRule with video mediatype, size and domain",
			request: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Domain: "www.test.com",
				},
				Imp: []openrtb2.Imp{{ID: "1234", Video: &openrtb2.Video{W: 300, H: 250, Placement: 2}}},
			},
			floorSchema: openrtb_ext.PriceFloorSchema{Delimiter: "|", Fields: []string{"mediaType", "size", "domain"}},
			out:         []string{"video-outstream", "300x250", "www.test.com"},
		},
		{
			name: "CreateRule with audio mediatype, adUnitCode and domain",
			request: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Domain: "www.test.com",
				},
				Imp: []openrtb2.Imp{{ID: "1234", TagID: "tag123", Audio: &openrtb2.Audio{MaxDuration: 300}}},
			},
			floorSchema: openrtb_ext.PriceFloorSchema{Delimiter: "|", Fields: []string{"mediaType", "adUnitCode", "siteDomain"}},
			out:         []string{"audio", "tag123", "www.test.com"},
		},
		{
			name: "CreateRule with audio mediatype, adUnitCode=* and domain",
			request: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Domain: "www.test.com",
				},
				Imp: []openrtb2.Imp{{ID: "1234", Audio: &openrtb2.Audio{MaxDuration: 300}}},
			},
			floorSchema: openrtb_ext.PriceFloorSchema{Delimiter: "|", Fields: []string{"mediaType", "adUnitCode", "siteDomain"}},
			out:         []string{"audio", "*", "www.test.com"},
		},
		{
			name: "CreateRule with native mediatype, bundle and domain",
			request: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Domain: "www.test.com",
					Bundle: "bundle123",
				},
				Imp: []openrtb2.Imp{{ID: "1234", Native: &openrtb2.Native{Request: "Test"}}},
			},
			floorSchema: openrtb_ext.PriceFloorSchema{Delimiter: "|", Fields: []string{"mediaType", "bundle", "siteDomain"}},
			out:         []string{"native", "bundle123", "www.test.com"},
		},
		{
			name: "CreateRule with native, banner mediatype, bundle and domain",
			request: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Domain: "www.test.com",
					Bundle: "bundle123",
				},
				Imp: []openrtb2.Imp{{ID: "1234", Audio: &openrtb2.Audio{MaxDuration: 300}, Native: &openrtb2.Native{Request: "Test"}}},
			},
			floorSchema: openrtb_ext.PriceFloorSchema{Delimiter: "|", Fields: []string{"mediaType", "bundle", "siteDomain"}},
			out:         []string{"*", "bundle123", "www.test.com"},
		},
		{
			name: "CreateRule with channel, country, deviceType",
			request: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Publisher: &openrtb2.Publisher{
						Domain: "www.test.com",
					},
					Bundle: "bundle123",
				},
				Device: &openrtb2.Device{Geo: &openrtb2.Geo{Country: "USA"}, UA: "tablet"},
				Imp:    []openrtb2.Imp{{ID: "1234", Native: &openrtb2.Native{Request: "Test"}}},
				Ext:    json.RawMessage(`{"prebid": {"channel": {"name": "chName","version": "ver1"}}}`),
			},
			floorSchema: openrtb_ext.PriceFloorSchema{Delimiter: "|", Fields: []string{"channel", "country", "deviceType"}},
			out:         []string{"chName", "USA", "tablet"},
		},
		{
			name: "CreateRule with channel, size, deviceType=desktop",
			request: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Publisher: &openrtb2.Publisher{
						Domain: "www.test.com",
					},
					Bundle: "bundle123",
				},
				Device: &openrtb2.Device{Geo: &openrtb2.Geo{Country: "USA"}, UA: "SomeDevice"},
				Imp:    []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 100, H: 200}, {W: 200, H: 300}}}}},
				Ext:    json.RawMessage(`{"prebid": {"test": "1}}`),
			},
			floorSchema: openrtb_ext.PriceFloorSchema{Delimiter: "|", Fields: []string{"channel", "size", "deviceType"}},
			out:         []string{"*", "*", "desktop"},
		},
		{
			name: "CreateRule with pubDomain, country, deviceType",
			request: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Publisher: &openrtb2.Publisher{
						Domain: "www.test.com",
					},
					Bundle: "bundle123",
				},
				Device: &openrtb2.Device{Geo: &openrtb2.Geo{Country: "USA"}, UA: "Phone"},
				Imp:    []openrtb2.Imp{{ID: "1234", Native: &openrtb2.Native{Request: "Test"}}},
				Ext:    json.RawMessage(`{"prebid": {"channel": {"name": "chName","version": "ver1"}}}`),
			},
			floorSchema: openrtb_ext.PriceFloorSchema{Delimiter: "|", Fields: []string{"pubDomain", "country", "deviceType"}},
			out:         []string{"www.test.com", "USA", "phone"},
		},
		{
			name: "CreateRule with pubDomain, gptSlot, deviceType",
			request: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Publisher: &openrtb2.Publisher{
						Domain: "www.test.com",
					},
				},
				Device: &openrtb2.Device{Geo: &openrtb2.Geo{Country: "USA"}},
				Imp: []openrtb2.Imp{{ID: "1234", Native: &openrtb2.Native{Request: "Test"},
					Ext: json.RawMessage(`{"data": {"adserver": {"name": "gam","adslot": "adslot123"}, "pbadslot": "pbadslot123"}}`),
				}},
				Ext: json.RawMessage(`{"prebid": {"channel": {"name": "chName","version": "ver1"}}}`),
			},
			floorSchema: openrtb_ext.PriceFloorSchema{Delimiter: "|", Fields: []string{"pubDomain", "gptSlot", "deviceType"}},
			out:         []string{"www.test.com", "adslot123", "*"},
		},
		{
			name: "CreateRule with pubDomain, gptSlot, deviceType",
			request: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Publisher: &openrtb2.Publisher{
						Domain: "www.test.com",
					},
				},
				Device: &openrtb2.Device{Geo: &openrtb2.Geo{Country: "USA"}},
				Imp: []openrtb2.Imp{{ID: "1234", Native: &openrtb2.Native{Request: "Test"},
					Ext: json.RawMessage(`{"data": {"adserver": {"name": "test","adslot": "adslot123"}, "pbadslot": "pbadslot123"}}`),
				}},
				Ext: json.RawMessage(`{"prebid": {"channel": {"name": "chName","version": "ver1"}}}`),
			},
			floorSchema: openrtb_ext.PriceFloorSchema{Delimiter: "|", Fields: []string{"pubDomain", "gptSlot", "deviceType"}},
			out:         []string{"www.test.com", "pbadslot123", "*"},
		},
		{
			name: "CreateRule with domain, adUnitCode, channel",
			request: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Publisher: &openrtb2.Publisher{
						Domain: "www.test.com",
					},
				},
				Device: &openrtb2.Device{Geo: &openrtb2.Geo{Country: "USA"}},
				Imp: []openrtb2.Imp{{ID: "1234", Native: &openrtb2.Native{Request: "Test"},
					Ext: json.RawMessage(`{"data": {"adserver": {"name": "test","adslot": "adslot123"}, "pbadslot": "pbadslot123"}}`),
				}},
			},
			floorSchema: openrtb_ext.PriceFloorSchema{Delimiter: "|", Fields: []string{"domain", "adUnitCode", "channel"}},
			out:         []string{"www.test.com", "pbadslot123", "*"},
		},
		{
			name: "CreateRule with domain, adUnitCode, channel",
			request: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Publisher: &openrtb2.Publisher{
						Domain: "www.test.com",
					},
				},
				Device: &openrtb2.Device{Geo: &openrtb2.Geo{Country: "USA"}},
				Imp: []openrtb2.Imp{{ID: "1234", Native: &openrtb2.Native{Request: "Test"},
					Ext: json.RawMessage(`{"gpid":  "gpid_134"}`),
				}},
			},
			floorSchema: openrtb_ext.PriceFloorSchema{Delimiter: "|", Fields: []string{"domain", "adUnitCode", "channel"}},
			out:         []string{"www.test.com", "gpid_134", "*"},
		},
		{
			name: "CreateRule with domain, adUnitCode, channel",
			request: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Publisher: &openrtb2.Publisher{
						Domain: "www.test.com",
					},
				},
				Device: &openrtb2.Device{Geo: &openrtb2.Geo{Country: "USA"}},
				Imp:    []openrtb2.Imp{{ID: "1234", Native: &openrtb2.Native{}, Ext: json.RawMessage(`{"prebid": {"storedrequest": {"id": "storedid_123"}}}`)}},
			},
			floorSchema: openrtb_ext.PriceFloorSchema{Delimiter: "|", Fields: []string{"domain", "adUnitCode", "channel"}},
			out:         []string{"www.test.com", "storedid_123", "*"},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			out := createRuleKey(tc.floorSchema, tc.request, tc.request.Imp[0])
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
		{
			name:                "RootSkipRate=100  with with skip = true",
			ModelGroupsSkipRate: 0,
			DataSkipRate:        0,
			RootSkipRate:        100,
			randomGen:           func(i int) int { return 100 },
			out:                 true,
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

func getIntPtr(v int) *int {
	return &v
}

func TestSelectFloorModelGroup(t *testing.T) {
	floorExt := &openrtb_ext.PriceFloorRules{Data: &openrtb_ext.PriceFloorData{
		SkipRate: 30,
		ModelGroups: []openrtb_ext.PriceFloorModelGroup{{
			ModelWeight:  getIntPtr(50),
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
				ModelWeight:  getIntPtr(25),
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

func Test_getMinFloorValue(t *testing.T) {
	rates := map[string]map[string]float64{
		"USD": {
			"INR": 81.17,
		},
	}

	type args struct {
		floorExt    *openrtb_ext.PriceFloorRules
		imp         openrtb2.Imp
		conversions currency.Conversions
	}
	tests := []struct {
		name    string
		args    args
		want    float64
		want1   string
		wantErr bool
	}{
		{
			name: "Floor min is available in imp and floor ext",
			args: args{
				floorExt: &openrtb_ext.PriceFloorRules{FloorMin: 2.0, FloorMinCur: "INR", Data: &openrtb_ext.PriceFloorData{Currency: "INR"}},
				imp:      openrtb2.Imp{Ext: json.RawMessage(`{"prebid":{"floors":{"floorMinCur": "INR","floorMin":1.0}}}`)},
			},
			want:    1,
			want1:   "INR",
			wantErr: false,
		},
		{
			name: "Floor min and floor min currency is available in imp ext only",
			args: args{
				floorExt: &openrtb_ext.PriceFloorRules{},
				imp:      openrtb2.Imp{Ext: json.RawMessage(`{"prebid":{"floors":{"floorMinCur": "INR", "floorMin": 1.0}}}`)},
			},
			want:    0.0123,
			want1:   "USD",
			wantErr: false,
		},
		{
			name: "Floor min is available in floor ext only",
			args: args{
				floorExt: &openrtb_ext.PriceFloorRules{FloorMin: 1.0, FloorMinCur: "EUR", Data: &openrtb_ext.PriceFloorData{Currency: "EUR"}},
				imp:      openrtb2.Imp{Ext: json.RawMessage(`{"prebid":{"floors":{}}}`)},
			},
			want:    1.0,
			want1:   "EUR",
			wantErr: false,
		},
		{
			name: "Floor min is available in floorExt and currency is available in imp",
			args: args{
				floorExt: &openrtb_ext.PriceFloorRules{FloorMin: 2.0, Data: &openrtb_ext.PriceFloorData{Currency: "INR"}},
				imp:      openrtb2.Imp{Ext: json.RawMessage(`{"prebid":{"floors":{"floorMinCur": "INR"}}}`)},
			},
			want:    2,
			want1:   "INR",
			wantErr: false,
		},
		{
			name: "Floor min is available in ImpExt and currency is available in floorExt",
			args: args{
				floorExt: &openrtb_ext.PriceFloorRules{FloorMinCur: "USD", Data: &openrtb_ext.PriceFloorData{Currency: "INR"}},
				imp:      openrtb2.Imp{Ext: json.RawMessage(`{"prebid":{"floors":{"FloorMin": 2.0}}}`)},
			},
			want:    162.34,
			want1:   "INR",
			wantErr: false,
		},
		{
			name: "Floor Min and floor Currency are in Imp and only floor currency is available in floor ext",
			args: args{
				floorExt: &openrtb_ext.PriceFloorRules{FloorMinCur: "USD"},
				imp:      openrtb2.Imp{Ext: json.RawMessage(`{"prebid":{"floors":{"floorMinCur": "USD","floorMin":1.0}}}`)},
			},
			want:    1,
			want1:   "USD",
			wantErr: false,
		},
		{
			name: "Currency are different in floor ext and imp",
			args: args{
				floorExt: &openrtb_ext.PriceFloorRules{FloorMin: 0.0, FloorMinCur: "EUR", Data: &openrtb_ext.PriceFloorData{Currency: "INR"}},
				imp:      openrtb2.Imp{Ext: json.RawMessage(`{"prebid":{"floors":{"floorMinCur": "USD","floorMin":1.0}}}`)},
			},
			want:    81.17,
			want1:   "INR",
			wantErr: false,
		},
		{
			name: "Floor min is 0 in imp ",
			args: args{
				floorExt: &openrtb_ext.PriceFloorRules{FloorMin: 2.0, FloorMinCur: "JPY", Data: &openrtb_ext.PriceFloorData{Currency: "INR"}},
				imp:      openrtb2.Imp{Ext: json.RawMessage(`{"prebid":{"floors":{"floorMinCur": "USD","floorMin":0.0}}}`)},
			},
			want:    162.34,
			want1:   "INR",
			wantErr: false,
		},
		{
			name: "Floor Currency is empty in imp",
			args: args{
				floorExt: &openrtb_ext.PriceFloorRules{FloorMin: 1.0, FloorMinCur: "EUR", Data: &openrtb_ext.PriceFloorData{Currency: "EUR"}},
				imp:      openrtb2.Imp{Ext: json.RawMessage(`{"prebid":{"floors":{"floorMinCur": "","floorMin":-1.0}}}`)},
			},
			want:    1.0,
			want1:   "EUR",
			wantErr: false,
		},
		{
			name: "Invalid input",
			args: args{
				floorExt: &openrtb_ext.PriceFloorRules{FloorMinCur: "EUR", Data: &openrtb_ext.PriceFloorData{}},
				imp:      openrtb2.Imp{Ext: json.RawMessage(`{`)},
			},
			want:    0.0,
			want1:   "USD",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := getMinFloorValue(tt.args.floorExt, tt.args.imp, getCurrencyRates(rates))
			if (err != nil) != tt.wantErr {
				t.Errorf("getMinFloorValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getMinFloorValue() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("getMinFloorValue() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
