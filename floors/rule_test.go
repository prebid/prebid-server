package floors

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/currency"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

func TestPrepareRuleCombinations(t *testing.T) {
	testCases := []struct {
		name string
		in   []string
		del  string
		exp  []string
	}{
		{
			name: "Schema items, n = 1",
			in:   []string{"A"},
			del:  "|",
			exp: []string{
				"a",
				"*",
			},
		},
		{
			name: "Schema items, n = 2",
			in:   []string{"A", "B"},
			del:  "|",
			exp: []string{
				"a|b",
				"a|*",
				"*|b",
				"*|*",
			},
		},
		{
			name: "Schema items, n = 3",
			in:   []string{"A", "B", "C"},
			del:  "|",
			exp: []string{
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
			del:  "|",
			exp: []string{
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
		{
			name: "Schema items, n = 1 with wildcards",
			in:   []string{"*"},
			del:  "|",
			exp: []string{
				"*",
			},
		},
		{
			name: "Schema items, n = 2 with wildcard at index = 0",
			in:   []string{"*", "B"},
			del:  "|",
			exp: []string{
				"*|b",
				"*|*",
			},
		},
		{
			name: "Schema items, n = 2 with wildcards at index = 1",
			in:   []string{"A", "*"},
			del:  "|",
			exp: []string{
				"a|*",
				"*|*",
			},
		},

		{
			name: "Schema items, n = 2 wildcards at index = 0,1",
			in:   []string{"*", "*"},
			del:  "|",
			exp: []string{
				"*|*",
			},
		},

		{
			name: "Schema items, n = 3 wildcard at index = 0",
			in:   []string{"*", "B", "C"},
			del:  "|",
			exp: []string{
				"*|b|c",
				"*|b|*",
				"*|*|c",
				"*|*|*",
			},
		},
		{
			name: "Schema items, n = 3 wildcard at index = 1",
			in:   []string{"A", "*", "C"},
			del:  "|",
			exp: []string{
				"a|*|c",
				"a|*|*",
				"*|*|c",
				"*|*|*",
			},
		},
		{
			name: "Schema items, n = 3 with wildcard at index = 2",
			in:   []string{"A", "B", "*"},
			del:  "|",
			exp: []string{
				"a|b|*",
				"a|*|*",
				"*|b|*",
				"*|*|*",
			},
		},
		{
			name: "Schema items, n = 3 with wildcard at index = 0,2",
			in:   []string{"*", "B", "*"},
			del:  "|",
			exp: []string{
				"*|b|*",
				"*|*|*",
			},
		},
		{
			name: "Schema items, n = 3 with wildcard at index = 0,1",
			in:   []string{"*", "*", "C"},
			del:  "|",
			exp: []string{
				"*|*|c",
				"*|*|*",
			},
		},
		{
			name: "Schema items, n = 3 with wildcard at index = 1,2",
			in:   []string{"A", "*", "*"},
			del:  "|",
			exp: []string{
				"a|*|*",
				"*|*|*",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			act := prepareRuleCombinations(tc.in, tc.del)
			assert.Equal(t, tc.exp, act, tc.name)
		})
	}
}

func TestUpdateImpExtWithFloorDetails(t *testing.T) {
	testCases := []struct {
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
			imp:          &openrtb_ext.ImpWrapper{Imp: &openrtb2.Imp{ID: "1234", Video: &openrtb2.Video{W: ptrutil.ToPtr[int64](300), H: ptrutil.ToPtr[int64](250)}}},
			expected:     []byte(`{"prebid":{"floors":{"floorrule":"test|123|xyz","floorrulevalue":5.5,"floorvalue":5.5}}}`),
		},
		{
			name:         "Empty ImpExt",
			matchedRule:  "test|123|xyz",
			floorRuleVal: 5.5,
			floorVal:     5.5,
			imp:          &openrtb_ext.ImpWrapper{Imp: &openrtb2.Imp{ID: "1234", Video: &openrtb2.Video{W: ptrutil.ToPtr[int64](300), H: ptrutil.ToPtr[int64](250)}, Ext: json.RawMessage{}}},
			expected:     []byte(`{"prebid":{"floors":{"floorrule":"test|123|xyz","floorrulevalue":5.5,"floorvalue":5.5}}}`),
		},
		{
			name:         "With prebid Ext",
			matchedRule:  "banner|www.test.com|*",
			floorRuleVal: 5.5,
			floorVal:     15.5,
			imp:          &openrtb_ext.ImpWrapper{Imp: &openrtb2.Imp{ID: "1234", Video: &openrtb2.Video{W: ptrutil.ToPtr[int64](300), H: ptrutil.ToPtr[int64](250)}, Ext: []byte(`{"prebid": {"test": true}}`)}},
			expected:     []byte(`{"prebid":{"floors":{"floorrule":"banner|www.test.com|*","floorrulevalue":5.5,"floorvalue":15.5}}}`),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			updateImpExtWithFloorDetails(tc.imp, tc.matchedRule, tc.floorRuleVal, tc.floorVal)
			_ = tc.imp.RebuildImp()
			if tc.imp.Ext != nil {
				assert.Equal(t, tc.imp.Ext, tc.expected, tc.name)
			}
		})
	}
}

func TestCreateRuleKeys(t *testing.T) {
	testCases := []struct {
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
				Imp: []openrtb2.Imp{{ID: "1234", Video: &openrtb2.Video{W: ptrutil.ToPtr[int64](640), H: ptrutil.ToPtr[int64](480), Placement: 1}}},
			},
			floorSchema: openrtb_ext.PriceFloorSchema{Delimiter: "|", Fields: []string{"mediaType", "size", "domain"}},
			out:         []string{"video", "640x480", "www.test.com"},
		},
		{
			name: "CreateRule with video mediatype, size and domain",
			request: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Domain: "www.test.com",
				},
				Imp: []openrtb2.Imp{{ID: "1234", Video: &openrtb2.Video{W: ptrutil.ToPtr[int64](300), H: ptrutil.ToPtr[int64](250), Placement: 2}}},
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
			name: "CreateRule with channel, country, deviceType=tablet",
			request: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Publisher: &openrtb2.Publisher{
						Domain: "www.test.com",
					},
					Bundle: "bundle123",
				},
				Device: &openrtb2.Device{Geo: &openrtb2.Geo{Country: "USA"}, UA: "Windows NT touch"},
				Imp:    []openrtb2.Imp{{ID: "1234", Native: &openrtb2.Native{Request: "Test"}}},
				Ext:    json.RawMessage(`{"prebid": {"channel": {"name": "chName","version": "ver1"}}}`),
			},
			floorSchema: openrtb_ext.PriceFloorSchema{Delimiter: "|", Fields: []string{"channel", "country", "deviceType"}},
			out:         []string{"chName", "USA", "tablet"},
		},
		{
			name: "CreateRule with channel, country, deviceType=desktop",
			request: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Publisher: &openrtb2.Publisher{
						Domain: "www.test.com",
					},
					Bundle: "bundle123",
				},
				Device: &openrtb2.Device{Geo: &openrtb2.Geo{Country: "USA"}, UA: "Windows NT"},
				Imp:    []openrtb2.Imp{{ID: "1234", Native: &openrtb2.Native{Request: "Test"}}},
				Ext:    json.RawMessage(`{"prebid": {"channel": {"name": "chName","version": "ver1"}}}`),
			},
			floorSchema: openrtb_ext.PriceFloorSchema{Delimiter: "|", Fields: []string{"channel", "country", "deviceType"}},
			out:         []string{"chName", "USA", "desktop"},
		},
		{
			name: "CreateRule with channel, country, deviceType=desktop",
			request: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Publisher: &openrtb2.Publisher{
						Domain: "www.test.com",
					},
					Bundle: "bundle123",
				},
				Device: &openrtb2.Device{Geo: &openrtb2.Geo{Country: "USA"}, UA: "Windows NT"},
				Imp:    []openrtb2.Imp{{ID: "1234", Native: &openrtb2.Native{Request: "Test"}}},
				Ext:    json.RawMessage(`{"prebid": {"channel": {"name": "chName","version": "ver1"}}}`),
			},
			floorSchema: openrtb_ext.PriceFloorSchema{Delimiter: "|", Fields: []string{"channel", "country", "deviceType"}},
			out:         []string{"chName", "USA", "desktop"},
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
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			out := createRuleKey(tc.floorSchema, &openrtb_ext.RequestWrapper{BidRequest: tc.request}, &openrtb_ext.ImpWrapper{Imp: &tc.request.Imp[0]})
			assert.Equal(t, out, tc.out, tc.name)
		})
	}
}

func TestShouldSkipFloors(t *testing.T) {

	testCases := []struct {
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
			randomGen:           func(i int) int { return 0 },
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
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			out := shouldSkipFloors(tc.ModelGroupsSkipRate, tc.DataSkipRate, tc.RootSkipRate, tc.randomGen)
			assert.Equal(t, out, tc.out, tc.name)
		})
	}

}

func TestSelectFloorModelGroup(t *testing.T) {
	weightNilModelGroup := openrtb_ext.PriceFloorModelGroup{ModelWeight: nil}
	weight01ModelGroup := openrtb_ext.PriceFloorModelGroup{ModelWeight: getIntPtr(1)}
	weight25ModelGroup := openrtb_ext.PriceFloorModelGroup{ModelWeight: getIntPtr(25)}
	weight50ModelGroup := openrtb_ext.PriceFloorModelGroup{ModelWeight: getIntPtr(50)}

	testCases := []struct {
		name               string
		ModelGroup         []openrtb_ext.PriceFloorModelGroup
		fn                 func(int) int
		expectedModelGroup []openrtb_ext.PriceFloorModelGroup
	}{
		{
			name: "ModelGroup with default weight selection",
			ModelGroup: []openrtb_ext.PriceFloorModelGroup{
				weightNilModelGroup,
			},
			fn: func(i int) int { return 0 },
			expectedModelGroup: []openrtb_ext.PriceFloorModelGroup{
				weight01ModelGroup,
			},
		},
		{
			name: "ModelGroup with weight = 25 selection",
			ModelGroup: []openrtb_ext.PriceFloorModelGroup{
				weight25ModelGroup,
				weight50ModelGroup,
			},
			fn: func(i int) int { return 5 },
			expectedModelGroup: []openrtb_ext.PriceFloorModelGroup{
				weight25ModelGroup,
			},
		},
		{
			name: "ModelGroup with weight = 50 selection",
			ModelGroup: []openrtb_ext.PriceFloorModelGroup{
				weight50ModelGroup,
			},
			fn: func(i int) int { return 55 },
			expectedModelGroup: []openrtb_ext.PriceFloorModelGroup{
				weight50ModelGroup,
			},
		},
		{
			name: "ModelGroup with weight = 25 selection",
			ModelGroup: []openrtb_ext.PriceFloorModelGroup{
				weight25ModelGroup,
				weight50ModelGroup,
			},
			fn: func(i int) int { return 80 },
			expectedModelGroup: []openrtb_ext.PriceFloorModelGroup{
				weight25ModelGroup,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp := selectFloorModelGroup(tc.ModelGroup, tc.fn)
			assert.Equal(t, resp, tc.expectedModelGroup)
		})
	}
}

func TestGetMinFloorValue(t *testing.T) {
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
	testCases := []struct {
		name    string
		args    args
		want    float64
		want1   string
		wantErr error
	}{
		{
			name: "Floor min is available in imp and floor ext",
			args: args{
				floorExt: &openrtb_ext.PriceFloorRules{FloorMin: 2.0, FloorMinCur: "INR", Data: &openrtb_ext.PriceFloorData{Currency: "INR"}},
				imp:      openrtb2.Imp{Ext: json.RawMessage(`{"prebid":{"floors":{"floorMinCur": "INR","floorMin":1.0}}}`)},
			},
			want:  1,
			want1: "INR",
		},
		{
			name: "Floor min and floor min currency is available in imp ext only",
			args: args{
				floorExt: &openrtb_ext.PriceFloorRules{},
				imp:      openrtb2.Imp{Ext: json.RawMessage(`{"prebid":{"floors":{"floorMinCur": "INR", "floorMin": 1.0}}}`)},
			},
			want:  0.0123,
			want1: "USD",
		},
		{
			name: "Floor min is available in floor ext only",
			args: args{
				floorExt: &openrtb_ext.PriceFloorRules{FloorMin: 1.0, FloorMinCur: "EUR", Data: &openrtb_ext.PriceFloorData{Currency: "EUR"}},
				imp:      openrtb2.Imp{Ext: json.RawMessage(`{"prebid":{"floors":{}}}`)},
			},
			want:  1.0,
			want1: "EUR",
		},
		{
			name: "Floor min is available in floorExt and currency is available in imp",
			args: args{
				floorExt: &openrtb_ext.PriceFloorRules{FloorMin: 2.0, Data: &openrtb_ext.PriceFloorData{Currency: "INR"}},
				imp:      openrtb2.Imp{Ext: json.RawMessage(`{"prebid":{"floors":{"floorMinCur": "INR"}}}`)},
			},
			want:  2,
			want1: "INR",
		},
		{
			name: "Floor min is available in ImpExt and currency is available in floorExt",
			args: args{
				floorExt: &openrtb_ext.PriceFloorRules{FloorMinCur: "USD", Data: &openrtb_ext.PriceFloorData{Currency: "INR"}},
				imp:      openrtb2.Imp{Ext: json.RawMessage(`{"prebid":{"floors":{"FloorMin": 2.0}}}`)},
			},
			want:  162.34,
			want1: "INR",
		},
		{
			name: "Floor Min and floor Currency are in Imp and only floor currency is available in floor ext",
			args: args{
				floorExt: &openrtb_ext.PriceFloorRules{FloorMinCur: "USD"},
				imp:      openrtb2.Imp{Ext: json.RawMessage(`{"prebid":{"floors":{"floorMinCur": "USD","floorMin":1.0}}}`)},
			},
			want:  1,
			want1: "USD",
		},
		{
			name: "Currency are different in floor ext and imp",
			args: args{
				floorExt: &openrtb_ext.PriceFloorRules{FloorMin: 0.0, FloorMinCur: "EUR", Data: &openrtb_ext.PriceFloorData{Currency: "INR"}},
				imp:      openrtb2.Imp{Ext: json.RawMessage(`{"prebid":{"floors":{"floorMinCur": "USD","floorMin":1.0}}}`)},
			},
			want:  81.17,
			want1: "INR",
		},
		{
			name: "Floor min is 0 in imp ",
			args: args{
				floorExt: &openrtb_ext.PriceFloorRules{FloorMin: 2.0, FloorMinCur: "JPY", Data: &openrtb_ext.PriceFloorData{Currency: "INR"}},
				imp:      openrtb2.Imp{Ext: json.RawMessage(`{"prebid":{"floors":{"floorMinCur": "USD","floorMin":0.0}}}`)},
			},
			want:  162.34,
			want1: "INR",
		},
		{
			name: "Floor Currency is empty in imp",
			args: args{
				floorExt: &openrtb_ext.PriceFloorRules{FloorMin: 1.0, FloorMinCur: "EUR", Data: &openrtb_ext.PriceFloorData{Currency: "EUR"}},
				imp:      openrtb2.Imp{Ext: json.RawMessage(`{"prebid":{"floors":{"floorMinCur": "","floorMin":-1.0}}}`)},
			},
			want:  1.0,
			want1: "EUR",
		},
		{
			name: "Invalid input",
			args: args{
				floorExt: &openrtb_ext.PriceFloorRules{FloorMinCur: "EUR", Data: &openrtb_ext.PriceFloorData{}},
				imp:      openrtb2.Imp{Ext: json.RawMessage(`{`)},
			},
			want:    0.0,
			want1:   "",
			wantErr: errors.New("Error in getting FloorMin value : 'expects \" or n, but found \x00'"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, got1, err := getMinFloorValue(tc.args.floorExt, &openrtb_ext.ImpWrapper{Imp: &tc.args.imp}, getCurrencyRates(rates))
			assert.Equal(t, tc.wantErr, err, tc.name)
			assert.Equal(t, tc.want, got, tc.name)
			assert.Equal(t, tc.want1, got1, tc.name)
		})
	}
}

func TestSortCombinations(t *testing.T) {
	type args struct {
		comb            [][]int
		numSchemaFields int
	}
	tests := []struct {
		name    string
		args    args
		expComb [][]int
	}{
		{
			name: "With schema fields = 3",
			args: args{
				comb:            [][]int{{0}, {1}, {2}},
				numSchemaFields: 3,
			},
			expComb: [][]int{{2}, {1}, {0}},
		},
		{
			name: "With schema fields = 3",
			args: args{
				comb:            [][]int{{0, 1}, {1, 2}, {0, 2}},
				numSchemaFields: 3,
			},
			expComb: [][]int{{1, 2}, {0, 2}, {0, 1}},
		},
		{
			name: "With schema fields = 4",
			args: args{
				comb:            [][]int{{0, 1, 2}, {1, 2, 3}, {0, 2, 3}, {0, 1, 3}},
				numSchemaFields: 3,
			},
			expComb: [][]int{{1, 2, 3}, {0, 2, 3}, {0, 1, 3}, {0, 1, 2}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sortCombinations(tt.args.comb, tt.args.numSchemaFields)
			assert.Equal(t, tt.expComb, tt.args.comb)
		})
	}
}

func TestGenerateCombinations(t *testing.T) {

	tests := []struct {
		name            string
		numSchemaFields int
		numWildCard     int
		expComb         [][]int
	}{
		{
			name:            "With schema fields = 3, wildcard = 1",
			numSchemaFields: 3,
			numWildCard:     1,
			expComb:         [][]int{{0}, {1}, {2}},
		},
		{
			name:            "With schema fields = 3, wildcard = 2",
			numSchemaFields: 3,
			numWildCard:     2,
			expComb:         [][]int{{0, 1}, {0, 2}, {1, 2}},
		},
		{
			name:            "With schema fields = 3, wildcard = 3",
			numSchemaFields: 3,
			numWildCard:     3,
			expComb:         [][]int{{0, 1, 2}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotComb := generateCombinations(tt.numSchemaFields, tt.numWildCard)
			assert.Equal(t, tt.expComb, gotComb)
		})
	}
}

func TestGetDeviceType(t *testing.T) {
	tests := []struct {
		name    string
		request *openrtb2.BidRequest
		want    string
	}{
		{
			name:    "user agent contains Phone",
			request: &openrtb2.BidRequest{Device: &openrtb2.Device{UA: "Mozilla/5.0 (Phone Samsung Mobile; Win64; x64)"}},
			want:    "phone",
		},
		{
			name:    "user agent contains iPhone",
			request: &openrtb2.BidRequest{Device: &openrtb2.Device{UA: "Safari(iPhone Apple Mobile)"}},
			want:    "phone",
		},
		{
			name:    "user agent contains Mobile.*Android",
			request: &openrtb2.BidRequest{Device: &openrtb2.Device{UA: "Mozilla/5.0 (Mobile Android; Win64; x64)"}},
			want:    "phone",
		},
		{
			name:    "user agent contains Android.*Mobile",
			request: &openrtb2.BidRequest{Device: &openrtb2.Device{UA: "Mozilla/5.0 (Android Redmi Mobile; Win64; x64)"}},
			want:    "phone",
		},
		{
			name:    "user agent contains Mobile.*Android",
			request: &openrtb2.BidRequest{Device: &openrtb2.Device{UA: "Mozilla/5.0 (Mobile pixel Android; Win64; x64)"}},
			want:    "phone",
		},
		{
			name:    "user agent contains Windows NT touch",
			request: &openrtb2.BidRequest{Device: &openrtb2.Device{UA: "Mozilla/5.0 (Windows NT touch 10.0; Win64; x64)"}},
			want:    "tablet",
		},
		{
			name:    "user agent contains ipad",
			request: &openrtb2.BidRequest{Device: &openrtb2.Device{UA: "Mozilla/5.0 (ipad 13.10; Win64; x64)"}},
			want:    "tablet",
		},
		{
			name:    "user agent contains Window NT.*touch",
			request: &openrtb2.BidRequest{Device: &openrtb2.Device{UA: "Mozilla/5.0 (compatible; MSIE 10.0; Windows NT 6.2; Win64; x64; Trident/6.0; Touch)"}},
			want:    "tablet",
		},
		{
			name:    "user agent contains touch.* Window NT",
			request: &openrtb2.BidRequest{Device: &openrtb2.Device{UA: "Mozilla/5.0 (touch realme Windows NT Win64; x64)"}},
			want:    "tablet",
		},
		{
			name:    "user agent contains Android",
			request: &openrtb2.BidRequest{Device: &openrtb2.Device{UA: "Mozilla/5.0 (Android; Win64; x64)"}},
			want:    "tablet",
		},
		{
			name:    "user agent not matching phone or tablet",
			request: &openrtb2.BidRequest{Device: &openrtb2.Device{UA: "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"}},
			want:    "desktop",
		},
		{
			name:    "empty user agent",
			request: &openrtb2.BidRequest{Device: &openrtb2.Device{}},
			want:    "*",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getDeviceType(&openrtb_ext.RequestWrapper{BidRequest: tt.request})
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetAdUnitCode(t *testing.T) {
	tests := []struct {
		name string
		imp  *openrtb_ext.ImpWrapper
		want string
	}{
		{
			name: "imp.ext.gpid",
			imp:  &openrtb_ext.ImpWrapper{Imp: &openrtb2.Imp{Ext: json.RawMessage(`{"gpid":"test_gpid"}`)}},
			want: "test_gpid",
		},
		{
			name: "imp.TagID",
			imp:  &openrtb_ext.ImpWrapper{Imp: &openrtb2.Imp{TagID: "tag_1"}},
			want: "tag_1",
		},
		{
			name: "imp.ext.data.pbadslot",
			imp:  &openrtb_ext.ImpWrapper{Imp: &openrtb2.Imp{Ext: json.RawMessage(`{"data":{"pbadslot":"pbslot_1"}}`)}},
			want: "pbslot_1",
		},
		{
			name: "imp.ext.prebid.storedrequest.id",
			imp:  &openrtb_ext.ImpWrapper{Imp: &openrtb2.Imp{Ext: json.RawMessage(`{"prebid": {"storedrequest":{"id":"123"}}}`)}},
			want: "123",
		},
		{
			name: "empty adUnitCode",
			imp:  &openrtb_ext.ImpWrapper{Imp: &openrtb2.Imp{}},
			want: "*",
		},
		{
			name: "empty_imp.ext.prebid.storedrequest.id",
			imp:  &openrtb_ext.ImpWrapper{Imp: &openrtb2.Imp{Ext: json.RawMessage(`{"prebid": {"custom":{"id":"123"}}}`)}},
			want: "*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getAdUnitCode(tt.imp)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetGptSlot(t *testing.T) {
	tests := []struct {
		name string
		imp  *openrtb_ext.ImpWrapper
		want string
	}{
		{
			name: "imp.ext.data.adserver.adslot",
			imp:  &openrtb_ext.ImpWrapper{Imp: &openrtb2.Imp{Ext: json.RawMessage(`{"data":{"adserver": {"name": "gam", "adslot": "slot_1"}}}`)}},
			want: "slot_1",
		},
		{
			name: "gptSlot = imp.ext.data.pbadslot",
			imp:  &openrtb_ext.ImpWrapper{Imp: &openrtb2.Imp{Ext: json.RawMessage(`{"data":{"pbadslot":"pbslot_1"}}`)}},
			want: "pbslot_1",
		},
		{
			name: "empty gptSlot",
			imp:  &openrtb_ext.ImpWrapper{Imp: &openrtb2.Imp{}},
			want: "*",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getGptSlot(tt.imp)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetSizeValue(t *testing.T) {
	tests := []struct {
		name string
		imp  *openrtb2.Imp
		want string
	}{
		{
			name: "banner: only one size exists in imp.banner.format",
			imp:  &openrtb2.Imp{Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}},
			want: "300x250",
		},
		{
			name: "banner: no imp.banner.format",
			imp:  &openrtb2.Imp{Banner: &openrtb2.Banner{W: getInt64Ptr(320), H: getInt64Ptr(240)}},
			want: "320x240",
		},
		{
			name: "video:  imp.video.w and  imp.video.h present",
			imp:  &openrtb2.Imp{Video: &openrtb2.Video{W: ptrutil.ToPtr[int64](120), H: ptrutil.ToPtr[int64](240)}},
			want: "120x240",
		},
		{
			name: "banner: more than one size exists in imp.banner.format",
			imp:  &openrtb2.Imp{Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 200, H: 300}}}},
			want: "*",
		},
		{
			name: "Audo creative",
			imp:  &openrtb2.Imp{Audio: &openrtb2.Audio{}},
			want: "*",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getSizeValue(tt.imp)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetMediaType(t *testing.T) {
	tests := []struct {
		name string
		imp  *openrtb2.Imp
		want string
	}{
		{
			name: "more than one of these: imp.banner, imp.video, imp.native, imp.audio present",
			imp:  &openrtb2.Imp{Video: &openrtb2.Video{W: ptrutil.ToPtr[int64](120), H: ptrutil.ToPtr[int64](240)}, Banner: &openrtb2.Banner{W: getInt64Ptr(320), H: getInt64Ptr(240)}},
			want: "*",
		},
		{
			name: "only banner present",
			imp:  &openrtb2.Imp{Banner: &openrtb2.Banner{W: getInt64Ptr(320), H: getInt64Ptr(240)}},
			want: "banner",
		},
		{
			name: "video-outstream present",
			imp:  &openrtb2.Imp{Video: &openrtb2.Video{W: ptrutil.ToPtr[int64](120), H: ptrutil.ToPtr[int64](240), Placement: 2}},
			want: "video-outstream",
		},
		{
			name: "video-instream present",
			imp:  &openrtb2.Imp{Video: &openrtb2.Video{W: ptrutil.ToPtr[int64](120), H: ptrutil.ToPtr[int64](240), Placement: 1}},
			want: "video",
		},
		{
			name: "only audio",
			imp:  &openrtb2.Imp{Audio: &openrtb2.Audio{MinDuration: 10}},
			want: "audio",
		},
		{
			name: "only native",
			imp:  &openrtb2.Imp{Native: &openrtb2.Native{Request: "test_req"}},
			want: "native",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getMediaType(tt.imp)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetSiteDomain(t *testing.T) {
	type args struct {
		request *openrtb_ext.RequestWrapper
	}
	tests := []struct {
		name    string
		request *openrtb_ext.RequestWrapper
		want    string
	}{
		{
			name:    "Site Domain present",
			request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{Site: &openrtb2.Site{Domain: "abc.xyz.com"}}},
			want:    "abc.xyz.com",
		},
		{
			name:    "Site Domain not present",
			request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{Site: &openrtb2.Site{}}},
			want:    "*",
		},
		{
			name:    "App Domain present",
			request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{App: &openrtb2.App{Domain: "cde.rtu.com"}}},
			want:    "cde.rtu.com",
		},
		{
			name:    "App Domain not present",
			request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{App: &openrtb2.App{}}},
			want:    "*",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getSiteDomain(tt.request)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetPublisherDomain(t *testing.T) {
	type args struct {
	}
	tests := []struct {
		name    string
		request *openrtb_ext.RequestWrapper
		want    string
	}{
		{
			name:    "Site publisher domain present",
			request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{Site: &openrtb2.Site{Publisher: &openrtb2.Publisher{Domain: "qwe.xyz.com"}}}},
			want:    "qwe.xyz.com",
		},
		{
			name:    "Site publisher domain not present",
			request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{Site: &openrtb2.Site{Publisher: &openrtb2.Publisher{}}}},
			want:    "*",
		},
		{
			name:    "App publisher domain present",
			request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{App: &openrtb2.App{Publisher: &openrtb2.Publisher{Domain: "xyz.com"}}}},
			want:    "xyz.com",
		},
		{
			name:    "App publisher domain not present",
			request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{App: &openrtb2.App{}}},
			want:    "*",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getPublisherDomain(tt.request)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetDomain(t *testing.T) {
	type args struct {
	}
	tests := []struct {
		name    string
		request *openrtb_ext.RequestWrapper
		want    string
	}{
		{
			name:    "Site domain present",
			request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{Site: &openrtb2.Site{Domain: "qwe.xyz.com", Publisher: &openrtb2.Publisher{Domain: "abc.xyz.com"}}}},
			want:    "qwe.xyz.com",
		},
		{
			name:    "Site publisher domain present",
			request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{Site: &openrtb2.Site{Publisher: &openrtb2.Publisher{Domain: "abc.xyz.com"}}}},
			want:    "abc.xyz.com",
		},
		{
			name:    "Site domain not present",
			request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{Site: &openrtb2.Site{Publisher: &openrtb2.Publisher{}}}},
			want:    "*",
		},
		{
			name:    "App publisher domain present",
			request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{App: &openrtb2.App{Domain: "abc.com", Publisher: &openrtb2.Publisher{Domain: "xyz.com"}}}},
			want:    "abc.com",
		},
		{
			name:    "App domain present",
			request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{App: &openrtb2.App{Publisher: &openrtb2.Publisher{Domain: "xyz.com"}}}},
			want:    "xyz.com",
		},
		{
			name:    "App domain not present",
			request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{App: &openrtb2.App{}}},
			want:    "*",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getDomain(tt.request)
			assert.Equal(t, tt.want, got)
		})
	}
}

func getIntPtr(v int) *int {
	return &v
}

func getInt64Ptr(v int64) *int64 {
	return &v
}
