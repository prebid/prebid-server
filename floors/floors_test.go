package floors

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/currency"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestIsRequestEnabledWithFloor(t *testing.T) {
	FalseFlag := false
	TrueFlag := true

	tt := []struct {
		name string
		in   *openrtb_ext.ExtRequest
		out  bool
	}{
		{
			name: "Request With Nil Floors",
			in:   &openrtb_ext.ExtRequest{},
			out:  true,
		},
		{
			name: "Request With Floors Disabled",
			in:   &openrtb_ext.ExtRequest{Prebid: openrtb_ext.ExtRequestPrebid{Floors: &openrtb_ext.PriceFloorRules{Enabled: &FalseFlag}}},
			out:  false,
		},
		{
			name: "Request With Floors Enabled",
			in:   &openrtb_ext.ExtRequest{Prebid: openrtb_ext.ExtRequestPrebid{Floors: &openrtb_ext.PriceFloorRules{Enabled: &TrueFlag}}},
			out:  true,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			out := IsRequestEnabledWithFloor(tc.in.Prebid.Floors)
			if !reflect.DeepEqual(out, tc.out) {
				t.Errorf("error: \nreturn:\t%v\nwant:\t%v", out, tc.out)
			}
		})
	}
}

func TestUpdateImpsWithFloorsVariousRuleKeys(t *testing.T) {

	floorExt := &openrtb_ext.PriceFloorRules{Data: &openrtb_ext.PriceFloorData{ModelGroups: []openrtb_ext.PriceFloorModelGroup{{Schema: openrtb_ext.PriceFloorSchema{Fields: []string{"mediaType", "country", "deviceType"}},
		Values: map[string]float64{
			"audio|USA|phone": 1.01,
		}, Default: 0.01}}}}

	floorExt2 := &openrtb_ext.PriceFloorRules{Data: &openrtb_ext.PriceFloorData{ModelGroups: []openrtb_ext.PriceFloorModelGroup{{Schema: openrtb_ext.PriceFloorSchema{Fields: []string{"channel", "country", "deviceType"}},
		Values: map[string]float64{
			"chName|USA|tablet": 1.01,
			"*|USA|tablet":      2.01,
		}, Default: 0.01}}}}

	floorExt3 := &openrtb_ext.PriceFloorRules{FloorMin: 1.00, Data: &openrtb_ext.PriceFloorData{ModelGroups: []openrtb_ext.PriceFloorModelGroup{{Schema: openrtb_ext.PriceFloorSchema{Fields: []string{"mediaType", "gptSlot", "bundle"}},
		Values: map[string]float64{
			"native|adslot123|bundle1":   0.01,
			"native|pbadslot123|bundle1": 0.01,
		}, Default: 0.01}}}}

	floorExt4 := &openrtb_ext.PriceFloorRules{FloorMin: 1.00, Data: &openrtb_ext.PriceFloorData{ModelGroups: []openrtb_ext.PriceFloorModelGroup{{Schema: openrtb_ext.PriceFloorSchema{Fields: []string{"mediaType", "pbAdSlot", "bundle"}},
		Values: map[string]float64{
			"native|pbadslot123|bundle1": 0.01,
		}, Default: 0.01}}}}
	tt := []struct {
		name     string
		floorExt *openrtb_ext.PriceFloorRules
		request  *openrtb2.BidRequest
		floorVal float64
		floorCur string
	}{
		{
			name: "audio|USA|phone",
			request: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
				},
				Device: &openrtb2.Device{Geo: &openrtb2.Geo{Country: "USA"}, UA: "Phone"},
				Imp:    []openrtb2.Imp{{ID: "1234", Audio: &openrtb2.Audio{MaxDuration: 10}}},
				Ext:    json.RawMessage(`{"prebid": {"floors": {"data": {"currency": "USD","skipRate": 0, "schema": {"fields": ["channel","size","domain"]},"values": {"chName|USA|tablet": 1.01, "*|*|*": 16.01},"default": 1},"channel": {"name": "chName","version": "ver1"}}}}`),
			},
			floorExt: floorExt,
			floorVal: 1.01,
			floorCur: "USD",
		},
		{
			name: "chName|USA|tablet",
			request: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
				},
				Device: &openrtb2.Device{Geo: &openrtb2.Geo{Country: "USA"}, UA: "tablet"},
				Imp:    []openrtb2.Imp{{ID: "1234", Audio: &openrtb2.Audio{MaxDuration: 10}}},
				Ext:    json.RawMessage(`{"prebid": {"channel": {"name": "chName","version": "ver1"}}}`)},
			floorExt: floorExt2,
			floorVal: 1.01,
			floorCur: "USD",
		},
		{
			name: "*|USA|tablet",
			request: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
				},
				Device: &openrtb2.Device{Geo: &openrtb2.Geo{Country: "USA"}, UA: "tablet"},
				Imp:    []openrtb2.Imp{{ID: "1234", Audio: &openrtb2.Audio{MaxDuration: 10}}},
				Ext:    json.RawMessage(`{"prebid": }`)},
			floorExt: floorExt2,
			floorVal: 2.01,
			floorCur: "USD",
		},
		{
			name: "native|gptSlot|bundle1",
			request: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Bundle:    "bundle1",
					Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
				},
				Device: &openrtb2.Device{Geo: &openrtb2.Geo{Country: "USA"}, UA: "tablet"},
				Imp:    []openrtb2.Imp{{ID: "1234", Native: &openrtb2.Native{}, Ext: json.RawMessage(`{"data": {"adserver": {"name": "gam","adslot": "adslot123"}, "pbadslot": "pbadslot123"}}`)}},
				Ext:    json.RawMessage(`{"prebid": { "floors": {"data": {"currency": "USD","skipRate": 0,"schema": {"fields": [ "mediaType", "size", "domain" ] },"values": {  "banner|300x250|www.website.com": 1.01, "banner|300x250|*": 2.01, "banner|300x600|www.website.com": 3.01,  "banner|300x600|*": 4.01, "banner|728x90|www.website.com": 5.01, "banner|728x90|*": 6.01, "banner|*|www.website.com": 7.01, "banner|*|*": 8.01, "*|300x250|www.website.com": 9.01, "*|300x250|*": 10.01, "*|300x600|www.website.com": 11.01,  "*|300x600|*": 12.01,  "*|728x90|www.website.com": 13.01, "*|728x90|*": 14.01,  "*|*|www.website.com": 15.01, "*|*|*": 16.01  }, "default": 1}}}}`),
			},
			floorExt: floorExt3,
			floorVal: 1.00,
			floorCur: "USD",
		},
		{
			name: "native|pbAdSlot|bundle1",
			request: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Bundle:    "bundle1",
					Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
				},
				Device: &openrtb2.Device{Geo: &openrtb2.Geo{Country: "USA"}, UA: "tablet"},
				Imp:    []openrtb2.Imp{{ID: "1234", Native: &openrtb2.Native{}, Ext: json.RawMessage(`{"data": {"adserver": {"name": "gam","adslot": "adslot123"}, "pbadslot": "pbadslot123"}}`)}},
				Ext:    json.RawMessage(`{"prebid": { "floors": {"data": {"currency": "USD","skipRate": 0,"schema": {"fields": [ "mediaType", "size", "domain" ] },"values": {  "banner|300x250|www.website.com": 1.01, "banner|300x250|*": 2.01, "banner|300x600|www.website.com": 3.01,  "banner|300x600|*": 4.01, "banner|728x90|www.website.com": 5.01, "banner|728x90|*": 6.01, "banner|*|www.website.com": 7.01, "banner|*|*": 8.01, "*|300x250|www.website.com": 9.01, "*|300x250|*": 10.01, "*|300x600|www.website.com": 11.01,  "*|300x600|*": 12.01,  "*|728x90|www.website.com": 13.01, "*|728x90|*": 14.01,  "*|*|www.website.com": 15.01, "*|*|*": 16.01  }, "default": 1}}}}`),
			},
			floorExt: floorExt4,
			floorVal: 1.00,
			floorCur: "USD",
		},
		{
			name: "native|gptSlot|bundle1",
			request: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Bundle:    "bundle1",
					Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
				},
				Device: &openrtb2.Device{Geo: &openrtb2.Geo{Country: "USA"}, UA: "tablet"},
				Imp:    []openrtb2.Imp{{ID: "1234", Native: &openrtb2.Native{}, Ext: json.RawMessage(`{"data": {"adserver": {"name": "ow","adslot": "adslot123"}, "pbadslot": "pbadslot123"}}`)}},
				Ext:    json.RawMessage(`{"prebid": { "floors": {"data": {"currency": "USD","skipRate": 0,"schema": {"fields": [ "mediaType", "size", "domain" ] },"values": {  "banner|300x250|www.website.com": 1.01, "banner|300x250|*": 2.01, "banner|300x600|www.website.com": 3.01,  "banner|300x600|*": 4.01, "banner|728x90|www.website.com": 5.01, "banner|728x90|*": 6.01, "banner|*|www.website.com": 7.01, "banner|*|*": 8.01, "*|300x250|www.website.com": 9.01, "*|300x250|*": 10.01, "*|300x600|www.website.com": 11.01,  "*|300x600|*": 12.01,  "*|728x90|www.website.com": 13.01, "*|728x90|*": 14.01,  "*|*|www.website.com": 15.01, "*|*|*": 16.01  }, "default": 1}}}}`),
			},
			floorExt: floorExt3,
			floorVal: 1.00,
			floorCur: "USD",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			_ = UpdateImpsWithFloors(tc.floorExt, tc.request, nil)
			if !reflect.DeepEqual(tc.request.Imp[0].BidFloor, tc.floorVal) {
				t.Errorf("Floor Value error: \nreturn:\t%v\nwant:\t%v", tc.request.Imp[0].BidFloor, tc.floorVal)
			}
			if !reflect.DeepEqual(tc.request.Imp[0].BidFloorCur, tc.floorCur) {
				t.Errorf("Floor Currency error: \nreturn:\t%v\nwant:\t%v", tc.request.Imp[0].BidFloor, tc.floorCur)
			}
		})
	}
}

func getCurrencyRates(rates map[string]map[string]float64) currency.Conversions {
	return currency.NewRates(rates)
}

func TestUpdateImpsWithFloors(t *testing.T) {

	rates := map[string]map[string]float64{
		"USD": {
			"INR": 70,
			"EUR": 0.9,
			"JPY": 5.09,
		},
	}

	floorExt := &openrtb_ext.PriceFloorRules{Data: &openrtb_ext.PriceFloorData{ModelGroups: []openrtb_ext.PriceFloorModelGroup{{Schema: openrtb_ext.PriceFloorSchema{Fields: []string{"mediaType", "size", "domain"}},
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
		}, Default: 0.01}}}}

	floorExt2 := &openrtb_ext.PriceFloorRules{Data: &openrtb_ext.PriceFloorData{ModelGroups: []openrtb_ext.PriceFloorModelGroup{{Schema: openrtb_ext.PriceFloorSchema{Fields: []string{"mediaType", "size", "siteDomain"}, Delimiter: "|"},
		Values: map[string]float64{
			"banner|300x250|www.publisher.com":   1.01,
			"banner|300x250|*":                   2.01,
			"banner|300x600|www.publisher.com":   3.01,
			"banner|300x600|*":                   4.01,
			"banner|728x90|www.website.com":      5.01,
			"banner|728x90|www.website.com|test": 5.01,
			"banner|728x90|*":                    6.01,
			"banner|*|www.website.com":           7.01,
			"banner|*|*":                         8.01,
			"video|*|*":                          9.01,
			"*|300x250|www.website.com":          10.01,
			"*|300x250|*":                        10.11,
			"*|300x600|www.website.com":          11.01,
			"*|300x600|*":                        12.01,
			"*|728x90|www.website.com":           13.01,
			"*|728x90|*":                         14.01,
			"*|*|www.website.com":                15.01,
			"*|*|*":                              16.01,
		}, Default: 0.01}}}}

	floorExt3 := &openrtb_ext.PriceFloorRules{Data: &openrtb_ext.PriceFloorData{ModelGroups: []openrtb_ext.PriceFloorModelGroup{
		{Schema: openrtb_ext.PriceFloorSchema{Fields: []string{"mediaType", "size", "pubDomain"}, Delimiter: "|"},
			Values: map[string]float64{
				"banner|300x250|www.publisher.com": 1.01,
				"banner|300x250|*":                 2.01,
				"banner|300x600|www.publisher.com": 3.01,
				"banner|300x600|*":                 4.01,
				"banner|728x90|www.website.com":    5.01,
				"banner|728x90|*":                  6.01,
				"banner|*|www.website.com":         7.01,
				"banner|*|*":                       8.01,
			}, Currency: "USD", Default: 0.01}}}, FloorMin: 1.0, FloorMinCur: "EUR"}

	floorExt4 := &openrtb_ext.PriceFloorRules{Data: &openrtb_ext.PriceFloorData{ModelGroups: []openrtb_ext.PriceFloorModelGroup{
		{Schema: openrtb_ext.PriceFloorSchema{Fields: []string{"mediaType", "size", "pubDomain"}, Delimiter: "|"},
			Values: map[string]float64{
				"banner|300x250|www.publisher.com": 1.01,
			}, SkipRate: 100, Default: 0.01}}}}
	width := int64(300)
	height := int64(600)
	tt := []struct {
		name     string
		floorExt *openrtb_ext.PriceFloorRules
		request  *openrtb2.BidRequest
		floorVal float64
		floorCur string
		Skipped  bool
	}{
		{
			name: "banner|300x250|www.website.com",
			request: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
				},
				Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}}},
				Ext: json.RawMessage(`{"prebid": { "floors": {"data": {"currency": "USD","skipRate": 0,"schema": {"fields": [ "mediaType", "size", "domain" ] },"values": {  "banner|300x250|www.website.com": 1.01, "banner|300x250|*": 2.01, "banner|300x600|www.website.com": 3.01,  "banner|300x600|*": 4.01, "banner|728x90|www.website.com": 5.01, "banner|728x90|*": 6.01, "banner|*|www.website.com": 7.01, "banner|*|*": 8.01, "*|300x250|www.website.com": 9.01, "*|300x250|*": 10.01, "*|300x600|www.website.com": 11.01,  "*|300x600|*": 12.01,  "*|728x90|www.website.com": 13.01, "*|728x90|*": 14.01,  "*|*|www.website.com": 15.01, "*|*|*": 16.01  }, "default": 1}}}}`),
			},
			floorExt: floorExt,
			floorVal: 1.01,
			floorCur: "USD",
		},
		{
			name: "banner|300x600|www.website.com",
			request: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Domain: "www.website.com",
				},
				Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{W: &width, H: &height}}},
				Ext: json.RawMessage(`{"prebid": { "floors": {"data": {"currency": "USD","skipRate": 0,"schema": {"fields": [ "mediaType", "size", "domain" ] },"values": {  "banner|300x250|www.website.com": 1.01, "banner|300x250|*": 2.01, "banner|300x600|www.website.com": 3.01,  "banner|300x600|*": 4.01, "banner|728x90|www.website.com": 5.01, "banner|728x90|*": 6.01, "banner|*|www.website.com": 7.01, "banner|*|*": 8.01, "*|300x250|www.website.com": 9.01, "*|300x250|*": 10.01, "*|300x600|www.website.com": 11.01,  "*|300x600|*": 12.01,  "*|728x90|www.website.com": 13.01, "*|728x90|*": 14.01,  "*|*|www.website.com": 15.01, "*|*|*": 16.01  }, "default": 1}}}}`),
			},
			floorExt: floorExt,
			floorVal: 3.01,
			floorCur: "USD",
		},
		{
			name: "*|*|www.website.com",
			request: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Domain: "www.website.com",
				},
				Imp: []openrtb2.Imp{{ID: "1234", Video: &openrtb2.Video{W: 640, H: 480}}},
				Ext: json.RawMessage(`{"prebid": { "floors": {"data": {"currency": "USD","skipRate": 0,"schema": {"fields": [ "mediaType", "size", "domain" ] },"values": {  "banner|300x250|www.website.com": 1.01, "banner|300x250|*": 2.01, "banner|300x600|www.website.com": 3.01,  "banner|300x600|*": 4.01, "banner|728x90|www.website.com": 5.01, "banner|728x90|*": 6.01, "banner|*|www.website.com": 7.01, "banner|*|*": 8.01, "*|300x250|www.website.com": 9.01, "*|300x250|*": 10.01, "*|300x600|www.website.com": 11.01,  "*|300x600|*": 12.01,  "*|728x90|www.website.com": 13.01, "*|728x90|*": 14.01,  "*|*|www.website.com": 15.01, "*|*|*": 16.01  }, "default": 1}}}}`),
			},
			floorExt: floorExt,
			floorVal: 15.01,
			floorCur: "USD",
		},
		{
			name: "*|300x250|www.website.com",
			request: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
				},
				Imp: []openrtb2.Imp{{ID: "1234", Video: &openrtb2.Video{W: 300, H: 250}}},
				Ext: json.RawMessage(`{"prebid": { "floors": {"data": {"currency": "USD","skipRate": 0,"schema": {"fields": [ "mediaType", "size", "domain" ] },"values": {  "banner|300x250|www.website.com": 1.01, "banner|300x250|*": 2.01, "banner|300x600|www.website.com": 3.01,  "banner|300x600|*": 4.01, "banner|728x90|www.website.com": 5.01, "banner|728x90|*": 6.01, "banner|*|www.website.com": 7.01, "banner|*|*": 8.01, "*|300x250|www.website.com": 9.01, "*|300x250|*": 10.01, "*|300x600|www.website.com": 11.01,  "*|300x600|*": 12.01,  "*|728x90|www.website.com": 13.01, "*|728x90|*": 14.01,  "*|*|www.website.com": 15.01, "*|*|*": 16.01  }, "default": 1}}}}`),
			},
			floorExt: floorExt,
			floorVal: 9.01,
			floorCur: "USD",
		},
		{
			name: "siteDomain, banner|300x600|*",
			request: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Domain: "www.website.com",
				},
				Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 600}}}}},
				Ext: json.RawMessage(`{"prebid": { "floors": {"data": {"currency": "USD","skipRate": 0,"schema": {"fields": [ "mediaType", "size", "siteDomain" ] },"values": {  "banner|300x250|www.website.com": 1.01, "banner|300x250|*": 2.01, "banner|300x600|www.website.com": 3.01,  "banner|300x600|*": 4.01, "banner|728x90|www.website.com": 5.01, "banner|728x90|*": 6.01, "banner|*|www.website.com": 7.01, "banner|*|*": 8.01, "*|300x250|www.website.com": 9.01, "*|300x250|*": 10.01, "*|300x600|www.website.com": 11.01,  "*|300x600|*": 12.01,  "*|728x90|www.website.com": 13.01, "*|728x90|*": 14.01,  "*|*|www.website.com": 15.01, "*|*|*": 16.01  }, "default": 1}}}}`),
			},
			floorExt: floorExt2,
			floorVal: 4.01,
			floorCur: "USD",
		},
		{
			name: "siteDomain, video|*|*",
			request: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Domain: "www.website.com",
				},
				Imp: []openrtb2.Imp{{ID: "1234", Video: &openrtb2.Video{W: 640, H: 480}}},
				Ext: json.RawMessage(`{"prebid": { "floors": {"data": {"currency": "USD","skipRate": 0,"schema": {"fields": [ "mediaType", "size", "siteDomain" ] },"values": {  "banner|300x250|www.website.com": 1.01, "banner|300x250|*": 2.01, "banner|300x600|www.website.com": 3.01,  "banner|300x600|*": 4.01, "banner|728x90|www.website.com": 5.01, "banner|728x90|*": 6.01, "banner|*|www.website.com": 7.01, "banner|*|*": 8.01, "*|300x250|www.website.com": 9.01, "*|300x250|*": 10.01, "*|300x600|www.website.com": 11.01,  "*|300x600|*": 12.01,  "*|728x90|www.website.com": 13.01, "*|728x90|*": 14.01,  "*|*|www.website.com": 15.01, "*|*|*": 16.01  }, "default": 1}}}}`),
			},
			floorExt: floorExt2,
			floorVal: 9.01,
			floorCur: "USD",
		},
		{
			name: "pubDomain, *|300x250|www.website.com",
			request: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
				},
				Imp: []openrtb2.Imp{{ID: "1234", Video: &openrtb2.Video{W: 300, H: 250}}},
				Ext: json.RawMessage(`{"prebid": { "floors": {"data": {"currency": "USD","skipRate": 0,"schema": {"fields": [ "mediaType", "size", "pubDomain" ] },"values": {  "banner|300x250|www.website.com": 1.01, "banner|300x250|*": 2.01, "banner|300x600|www.website.com": 3.01,  "banner|300x600|*": 4.01, "banner|728x90|www.website.com": 5.01, "banner|728x90|*": 6.01, "banner|*|www.website.com": 7.01, "banner|*|*": 8.01, "*|300x250|www.website.com": 9.01, "*|300x250|*": 10.01, "*|300x600|www.website.com": 11.01,  "*|300x600|*": 12.01,  "*|728x90|www.website.com": 13.01, "*|728x90|*": 14.01,  "*|*|www.website.com": 15.01, "*|*|*": 16.01  }, "default": 1}}}}`),
			},
			floorExt: floorExt2,
			floorVal: 9.01,
			floorCur: "USD",
		},
		{
			name: "pubDomain, Default Floor Value",
			request: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
				},
				Imp: []openrtb2.Imp{{ID: "1234", Video: &openrtb2.Video{W: 300, H: 250}}},
				Ext: json.RawMessage(`{"prebid": { "floors": {"data": {"currency": "USD","skipRate": 0,"schema": {"fields": [ "mediaType", "size", "pubDomain" ] },"values": {  "banner|300x250|www.website.com": 1.01, "banner|300x250|*": 2.01, "banner|300x600|www.website.com": 3.01,  "banner|300x600|*": 4.01, "banner|728x90|www.website.com": 5.01, "banner|728x90|*": 6.01, "banner|*|www.website.com": 7.01, "banner|*|*": 8.01, "*|300x250|www.website.com": 9.01, "*|300x250|*": 10.01, "*|300x600|www.website.com": 11.01,  "*|300x600|*": 12.01,  "*|728x90|www.website.com": 13.01, "*|728x90|*": 14.01,  "*|*|www.website.com": 15.01, "*|*|*": 16.01  }, "default": 1}}}}`),
			},
			floorExt: floorExt3,
			floorVal: 1.1111,
			floorCur: "USD",
		},
		{
			name: "pubDomain, Default Floor Value",
			request: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
				},
				Imp: []openrtb2.Imp{{ID: "1234", Video: &openrtb2.Video{W: 300, H: 250}}},
				Ext: json.RawMessage(`{"prebid": { "floors": {"data": {"currency": "USD","skipRate": 0,"schema": {"fields": [ "mediaType", "size", "pubDomain" ] },"values": {  "banner|300x250|www.website.com": 1.01, "banner|300x250|*": 2.01, "banner|300x600|www.website.com": 3.01,  "banner|300x600|*": 4.01, "banner|728x90|www.website.com": 5.01, "banner|728x90|*": 6.01, "banner|*|www.website.com": 7.01, "banner|*|*": 8.01, "*|300x250|www.website.com": 9.01, "*|300x250|*": 10.01, "*|300x600|www.website.com": 11.01,  "*|300x600|*": 12.01,  "*|728x90|www.website.com": 13.01, "*|728x90|*": 14.01,  "*|*|www.website.com": 15.01, "*|*|*": 16.01  }, "default": 1}}}}`),
			},
			floorExt: floorExt3,
			floorVal: 1.1111,
			floorCur: "USD",
		},
		{
			name: "Skiprate = 100, Check Skipped Flag",
			request: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
				},
				Imp: []openrtb2.Imp{{ID: "1234", Video: &openrtb2.Video{W: 300, H: 250}}},
				Ext: json.RawMessage(`{"prebid": { "floors": {"data": {"currency": "USD","skipRate": 0,"schema": {"fields": [ "mediaType", "size", "pubDomain" ] },"values": {  "banner|300x250|www.website.com": 1.01, "banner|300x250|*": 2.01, "banner|300x600|www.website.com": 3.01,  "banner|300x600|*": 4.01, "banner|728x90|www.website.com": 5.01, "banner|728x90|*": 6.01, "banner|*|www.website.com": 7.01, "banner|*|*": 8.01, "*|300x250|www.website.com": 9.01, "*|300x250|*": 10.01, "*|300x600|www.website.com": 11.01,  "*|300x600|*": 12.01,  "*|728x90|www.website.com": 13.01, "*|728x90|*": 14.01,  "*|*|www.website.com": 15.01, "*|*|*": 16.01  }, "default": 1}}}}`),
			},
			floorExt: floorExt4,
			floorVal: 0.0,
			floorCur: "",
			Skipped:  true,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			_ = UpdateImpsWithFloors(tc.floorExt, tc.request, getCurrencyRates(rates))
			if !reflect.DeepEqual(tc.request.Imp[0].BidFloor, tc.floorVal) {
				t.Errorf("Floor Value error: \nreturn:\t%v\nwant:\t%v", tc.request.Imp[0].BidFloor, tc.floorVal)
			}
			if !reflect.DeepEqual(tc.request.Imp[0].BidFloorCur, tc.floorCur) {
				t.Errorf("Floor Currency error: \nreturn:\t%v\nwant:\t%v", tc.request.Imp[0].BidFloor, tc.floorCur)
			}

			if !reflect.DeepEqual(*tc.floorExt.Skipped, tc.Skipped) {
				t.Errorf("Floor Skipped error: \nreturn:\t%v\nwant:\t%v", tc.floorExt.Skipped, tc.Skipped)
			}
		})
	}
}

func TestUpdateImpsWithModelGroups(t *testing.T) {
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
				ModelWeight:  50,
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

	rates := map[string]map[string]float64{
		"USD": {
			"INR": 70,
			"EUR": 0.9,
			"JPY": 5.09,
		},
	}
	tt := []struct {
		name         string
		floorExt     *openrtb_ext.PriceFloorRules
		request      *openrtb2.BidRequest
		floorVal     float64
		floorCur     string
		ModelVersion string
	}{
		{
			name: "banner|300x250|www.website.com",
			request: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Domain: "www.website.com",
				},
				Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}}},
				Ext: json.RawMessage(`{"prebid": { "floors": {"data": {"currency": "USD","skipRate": 0,"schema": {"fields": [ "mediaType", "size", "domain" ] },"values": {  "banner|300x250|www.website.com": 1.01, "banner|300x250|*": 2.01, "banner|300x600|www.website.com": 3.01,  "banner|300x600|*": 4.01, "banner|728x90|www.website.com": 5.01, "banner|728x90|*": 6.01, "banner|*|www.website.com": 7.01, "banner|*|*": 8.01, "*|300x250|www.website.com": 9.01, "*|300x250|*": 10.01, "*|300x600|www.website.com": 11.01,  "*|300x600|*": 12.01,  "*|728x90|www.website.com": 13.01, "*|728x90|*": 14.01,  "*|*|www.website.com": 15.01, "*|*|*": 16.01  }, "default": 1}}}}`),
			},
			floorExt:     floorExt,
			floorVal:     1.01,
			floorCur:     "USD",
			ModelVersion: "Version 2",
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			_ = UpdateImpsWithFloors(tc.floorExt, tc.request, getCurrencyRates(rates))
			if tc.floorExt.Skipped != nil && *tc.floorExt.Skipped != true {
				if !reflect.DeepEqual(tc.request.Imp[0].BidFloor, tc.floorVal) {
					t.Errorf("Floor Value error: \nreturn:\t%v\nwant:\t%v", tc.request.Imp[0].BidFloor, tc.floorVal)
				}
				if !reflect.DeepEqual(tc.request.Imp[0].BidFloorCur, tc.floorCur) {
					t.Errorf("Floor Currency error: \nreturn:\t%v\nwant:\t%v", tc.request.Imp[0].BidFloor, tc.floorCur)
				}

				if !reflect.DeepEqual(tc.floorExt.Data.ModelGroups[0].ModelVersion, tc.ModelVersion) {
					t.Errorf("Floor Model Version mismatch error: \nreturn:\t%v\nwant:\t%v", tc.floorExt.Data.ModelGroups[0].ModelVersion, tc.ModelVersion)
				}
			}
		})
	}
}

func TestUpdateImpsWithInvalidModelGroups(t *testing.T) {
	floorExt := &openrtb_ext.PriceFloorRules{Data: &openrtb_ext.PriceFloorData{
		ModelGroups: []openrtb_ext.PriceFloorModelGroup{{
			ModelWeight:  50,
			SkipRate:     110,
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
		}}}
	rates := map[string]map[string]float64{
		"USD": {
			"INR": 70,
			"EUR": 0.9,
			"JPY": 5.09,
		},
	}

	tt := []struct {
		name         string
		floorExt     *openrtb_ext.PriceFloorRules
		request      *openrtb2.BidRequest
		floorVal     float64
		floorCur     string
		ModelVersion string
		Err          string
	}{
		{
			name: "Invalid Skip Rate in model Group 1, with banner|300x250|www.website.com",
			request: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Domain: "www.website.com",
				},
				Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}}},
				Ext: json.RawMessage(`{"prebid": { "floors": {"data": {"currency": "USD","skipRate": 0,"schema": {"fields": [ "mediaType", "size", "domain" ] },"values": {  "banner|300x250|www.website.com": 1.01, "banner|300x250|*": 2.01, "banner|300x600|www.website.com": 3.01,  "banner|300x600|*": 4.01, "banner|728x90|www.website.com": 5.01, "banner|728x90|*": 6.01, "banner|*|www.website.com": 7.01, "banner|*|*": 8.01, "*|300x250|www.website.com": 9.01, "*|300x250|*": 10.01, "*|300x600|www.website.com": 11.01,  "*|300x600|*": 12.01,  "*|728x90|www.website.com": 13.01, "*|728x90|*": 14.01,  "*|*|www.website.com": 15.01, "*|*|*": 16.01  }, "default": 1}}}}`),
			},
			floorExt: floorExt,
			floorVal: 0.0,
			floorCur: "",
			Err:      "Invalid Floor Model = 'Version 1' due to SkipRate = '110'",
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ErrList := UpdateImpsWithFloors(tc.floorExt, tc.request, getCurrencyRates(rates))

			if !reflect.DeepEqual(tc.request.Imp[0].BidFloor, tc.floorVal) {
				t.Errorf("Floor Value error: \nreturn:\t%v\nwant:\t%v", tc.request.Imp[0].BidFloor, tc.floorVal)
			}
			if !reflect.DeepEqual(tc.request.Imp[0].BidFloorCur, tc.floorCur) {
				t.Errorf("Floor Currency error: \nreturn:\t%v\nwant:\t%v", tc.request.Imp[0].BidFloor, tc.floorCur)
			}

			if !reflect.DeepEqual(ErrList[0].Error(), tc.Err) {
				t.Errorf("Incorrect Error: \nreturn:\t%v\nwant:\t%v", ErrList[0].Error(), tc.Err)
			}

		})
	}
}

func TestUpdateImpsWithFloorsCurrecnyConversion(t *testing.T) {
	rates := map[string]map[string]float64{
		"USD": {
			"INR": 70,
			"EUR": 0.9,
			"JPY": 5.09,
		},
	}

	floorExt := &openrtb_ext.PriceFloorRules{FloorMin: 80, FloorMinCur: "INR", Data: &openrtb_ext.PriceFloorData{ModelGroups: []openrtb_ext.PriceFloorModelGroup{{Schema: openrtb_ext.PriceFloorSchema{Fields: []string{"mediaType", "size", "domain"}},
		Values: map[string]float64{
			"banner|300x250|www.website.com": 1.00,
			"banner|300x250|*":               2.01,
			"*|*|*":                          16.01,
		}, Default: 0.01}}}}
	floorExt2 := &openrtb_ext.PriceFloorRules{FloorMin: 1, FloorMinCur: "USD", Data: &openrtb_ext.PriceFloorData{Currency: "INR", ModelGroups: []openrtb_ext.PriceFloorModelGroup{{Schema: openrtb_ext.PriceFloorSchema{Fields: []string{"mediaType", "size", "domain"}},
		Values: map[string]float64{
			"banner|300x250|www.website.com": 65.00,
			"banner|300x250|*":               110.00,
		}, Default: 50.00}}}}
	floorExt3 := &openrtb_ext.PriceFloorRules{FloorMin: 1, FloorMinCur: "USD", Data: &openrtb_ext.PriceFloorData{ModelGroups: []openrtb_ext.PriceFloorModelGroup{{Schema: openrtb_ext.PriceFloorSchema{Fields: []string{"mediaType", "size", "domain"}},
		Values: map[string]float64{
			"banner|300x250|www.website.com": 2.00,
			"banner|300x250|*":               2.01,
			"*|*|*":                          16.01,
		}, Default: 0.01}}}}
	floorExt4 := &openrtb_ext.PriceFloorRules{FloorMin: 3, FloorMinCur: "USD", Data: &openrtb_ext.PriceFloorData{ModelGroups: []openrtb_ext.PriceFloorModelGroup{{Schema: openrtb_ext.PriceFloorSchema{Fields: []string{"mediaType", "size", "domain"}},
		Values: map[string]float64{
			"banner|300x250|www.website.com": 1.00,
			"banner|300x250|*":               2.01,
			"*|*|*":                          16.01,
		}, Default: 0.01}}}}
	tt := []struct {
		name     string
		floorExt *openrtb_ext.PriceFloorRules
		request  *openrtb2.BidRequest
		floorVal float64
		floorCur string
		Skipped  bool
	}{
		{
			name: "BidFloor(USD) Less than MinBidFloor(INR) with different currency",
			request: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
				},
				Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}}},
				Ext: json.RawMessage(`{"prebid": { "floors": {"data": {"currency": "USD","skipRate": 0,"schema": {"fields": [ "mediaType", "size", "domain" ] },"values": {  "banner|300x250|www.website.com": 1.01, "banner|300x250|*": 2.01, "banner|300x600|www.website.com": 3.01,  "banner|300x600|*": 4.01, "banner|728x90|www.website.com": 5.01, "banner|728x90|*": 6.01, "banner|*|www.website.com": 7.01, "banner|*|*": 8.01, "*|300x250|www.website.com": 9.01, "*|300x250|*": 10.01, "*|300x600|www.website.com": 11.01,  "*|300x600|*": 12.01,  "*|728x90|www.website.com": 13.01, "*|728x90|*": 14.01,  "*|*|www.website.com": 15.01, "*|*|*": 16.01  }, "default": 1}}}}`),
			},
			floorExt: floorExt,
			floorVal: 1.1429,
			floorCur: "USD",
		},
		{
			name: "BidFloor(INR) Less than MinBidFloor(USD) with different currency",
			request: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
				},
				Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}}},
				Ext: json.RawMessage(`{"prebid": { "floors": {"data": {"currency": "USD","skipRate": 0,"schema": {"fields": [ "mediaType", "size", "domain" ] },"values": {  "banner|300x250|www.website.com": 1.01, "banner|300x250|*": 2.01, "banner|300x600|www.website.com": 3.01,  "banner|300x600|*": 4.01, "banner|728x90|www.website.com": 5.01, "banner|728x90|*": 6.01, "banner|*|www.website.com": 7.01, "banner|*|*": 8.01, "*|300x250|www.website.com": 9.01, "*|300x250|*": 10.01, "*|300x600|www.website.com": 11.01,  "*|300x600|*": 12.01,  "*|728x90|www.website.com": 13.01, "*|728x90|*": 14.01,  "*|*|www.website.com": 15.01, "*|*|*": 16.01  }, "default": 1}}}}`),
			},
			floorExt: floorExt2,
			floorVal: 70,
			floorCur: "INR",
		},
		{
			name: "MinBidFloor Less than BidFloor with same currency",
			request: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
				},
				Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}}},
				Ext: json.RawMessage(`{"prebid": { "floors": {"data": {"currency": "USD","skipRate": 0,"schema": {"fields": [ "mediaType", "size", "domain" ] },"values": {  "banner|300x250|www.website.com": 1.01, "banner|300x250|*": 2.01, "banner|300x600|www.website.com": 3.01,  "banner|300x600|*": 4.01, "banner|728x90|www.website.com": 5.01, "banner|728x90|*": 6.01, "banner|*|www.website.com": 7.01, "banner|*|*": 8.01, "*|300x250|www.website.com": 9.01, "*|300x250|*": 10.01, "*|300x600|www.website.com": 11.01,  "*|300x600|*": 12.01,  "*|728x90|www.website.com": 13.01, "*|728x90|*": 14.01,  "*|*|www.website.com": 15.01, "*|*|*": 16.01  }, "default": 1}}}}`),
			},
			floorExt: floorExt3,
			floorVal: 2,
			floorCur: "USD",
		},
		{
			name: "BidFloor Less than MinBidFloor with same currency",
			request: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
				},
				Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}}},
				Ext: json.RawMessage(`{"prebid": { "floors": {"data": {"currency": "USD","skipRate": 0,"schema": {"fields": [ "mediaType", "size", "domain" ] },"values": {  "banner|300x250|www.website.com": 1.01, "banner|300x250|*": 2.01, "banner|300x600|www.website.com": 3.01,  "banner|300x600|*": 4.01, "banner|728x90|www.website.com": 5.01, "banner|728x90|*": 6.01, "banner|*|www.website.com": 7.01, "banner|*|*": 8.01, "*|300x250|www.website.com": 9.01, "*|300x250|*": 10.01, "*|300x600|www.website.com": 11.01,  "*|300x600|*": 12.01,  "*|728x90|www.website.com": 13.01, "*|728x90|*": 14.01,  "*|*|www.website.com": 15.01, "*|*|*": 16.01  }, "default": 1}}}}`),
			},
			floorExt: floorExt4,
			floorVal: 3,
			floorCur: "USD",
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			_ = UpdateImpsWithFloors(tc.floorExt, tc.request, getCurrencyRates(rates))
			if !reflect.DeepEqual(tc.request.Imp[0].BidFloor, tc.floorVal) {
				t.Errorf("Floor Value error: \nreturn:\t%v\nwant:\t%v", tc.request.Imp[0].BidFloor, tc.floorVal)
			}
			if !reflect.DeepEqual(tc.request.Imp[0].BidFloorCur, tc.floorCur) {
				t.Errorf("Floor Currency error: \nreturn:\t%v\nwant:\t%v", tc.request.Imp[0].BidFloor, tc.floorCur)
			}

		})
	}
}
