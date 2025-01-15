package floors

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/currency"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

func TestIsRequestEnabledWithFloor(t *testing.T) {
	FalseFlag := false
	TrueFlag := true

	testCases := []struct {
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
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			out := tc.in.Prebid.Floors.GetEnabled()
			assert.Equal(t, out, tc.out, tc.name)
		})
	}
}

func getCurrencyRates(rates map[string]map[string]float64) currency.Conversions {
	return currency.NewRates(rates)
}

type mockPriceFloorFetcher struct{}

func (mpf *mockPriceFloorFetcher) Fetch(configs config.AccountPriceFloors) (*openrtb_ext.PriceFloorRules, string) {
	return nil, openrtb_ext.FetchNone
}

func (mpf *mockPriceFloorFetcher) Stop() {}

func TestEnrichWithPriceFloors(t *testing.T) {
	rates := map[string]map[string]float64{
		"USD": {
			"INR": 70,
			"EUR": 0.9,
			"JPY": 5.09,
		},
	}

	testAccountConfig := config.Account{
		PriceFloors: config.AccountPriceFloors{
			Enabled:        true,
			UseDynamicData: false,
			MaxRule:        100,
			MaxSchemaDims:  5,
		},
	}

	width := int64(300)
	height := int64(600)

	testCases := []struct {
		name              string
		bidRequestWrapper *openrtb_ext.RequestWrapper
		account           config.Account
		conversions       currency.Conversions
		Skipped           bool
		err               string
		expFloorVal       float64
		expFloorCur       string
		expPriceFlrLoc    string
		expSchemaVersion  int
	}{
		{
			name: "Floors disabled in account config",
			bidRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
					},
					Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}}},
					Ext: json.RawMessage(`{"prebid":{"floors":{"data":{"currency":"USD","modelgroups":[{"modelversion":"model 1 from req","currency":"USD","values":{"banner|300x600|www.website5.com":5,"*|*|*":7},"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"}}]},"enabled":true,"enforcement":{"enforcepbs":true,"floordeals":true,"enforcerate":100}}}}`),
				},
			},
			account: config.Account{
				PriceFloors: config.AccountPriceFloors{
					Enabled: false,
				},
			},
			err: "Floors feature is disabled at account or in the request",
		},
		{
			name: "Floors disabled in req.ext.prebid.floors.Enabled=false config",
			bidRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
					},
					Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}}},
					Ext: json.RawMessage(`{"prebid":{"floors":{"data":{"currency":"USD","modelgroups":[{"modelversion":"model 1 from req","currency":"USD","values":{"banner|300x600|www.website.com":5,"*|*|*":7},"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"}}]},"enabled":false,"enforcement":{"enforcepbs":true,"floordeals":true,"enforcerate":100}}}}`),
				},
			},
			account: testAccountConfig,
			err:     "Floors feature is disabled at account or in the request",
		},
		{
			name: "Floors enabled in req.ext.prebid.floors.Enabled and enabled account config",
			bidRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
					},
					Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}}},
					Ext: json.RawMessage(`{"prebid":{"floors":{"data":{"currency":"USD","modelgroups":[{"modelversion":"model 2 from req","currency":"USD","values":{"banner|300x250|www.website.com":5,"*|*|*":7},"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"}},{"modelversion":"model 1 from req","currency":"USD","values":{"banner|300x250|www.website.com":5,"*|*|*":7},"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"}}]},"enabled":true,"enforcement":{"enforcepbs":true,"floordeals":true,"enforcerate":100}}}}`),
				},
			},
			account:        testAccountConfig,
			expFloorVal:    5,
			expFloorCur:    "USD",
			expPriceFlrLoc: openrtb_ext.RequestLocation,
		},
		{
			name: "Skiprate = 100, Floors enabled in  req.ext.prebid.floors.Enabled and account config: Floors singalling skipped ",
			bidRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
					},
					Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}}},
					Ext: json.RawMessage(`{"prebid":{"floors":{"data":{"currency":"USD","modelgroups":[{"modelversion":"model 1 from req","currency":"USD","values":{"banner|300x250|www.website.com":5,"*|*|*":7},"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"}}]},"enabled":true,"skiprate": 100,"enforcement":{"enforcepbs":true,"floordeals":true,"enforcerate":100}}}}`),
				},
			},
			account: testAccountConfig,
			Skipped: true,
		},
		{
			name: "Single ModelGroup, Invalid Skiprate = 110: Floors singalling skipped",
			bidRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
					},
					Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}}},
					Ext: json.RawMessage(`{"prebid":{"floors":{"data":{"currency":"USD","modelgroups":[{"modelversion":"model 1 from req","currency":"USD","values":{"banner|300x250|www.website.com":5,"*|*|*":7},"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"}}]},"enabled":true,"skiprate": 110,"enforcement":{"enforcepbs":true,"floordeals":true,"enforcerate":100}}}}`),
				},
			},
			account:        testAccountConfig,
			err:            "Invalid SkipRate = '110' at ext.prebid.floors.skiprate",
			expPriceFlrLoc: openrtb_ext.RequestLocation,
		},
		{
			name: "Multiple ModelGroups, Invalid Skiprate = 110: in one group, Floors singalling done using second ModelGroup",
			bidRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
					},
					Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}}},
					Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":11,"floormincur":"USD","data":{"currency":"USD","floorsschemaversion":2,"modelgroups":[{"modelweight":50,"modelversion":"version2","schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":11.01,"*|*|www.website1.com":17.01},"default":21},{"modelweight":50,"modelversion":"version11","skiprate":110,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|300x250|*":11.01,"*|300x250|www.website1.com":100.01},"default":21}]},"enforcement":{"enforcepbs":true,"floordeals":true},"enabled":true}}}`),
				},
			},
			account:          testAccountConfig,
			err:              "Invalid Floor Model = 'version11' due to SkipRate = '110' is out of range (1-100)",
			expFloorVal:      11.01,
			expFloorCur:      "USD",
			expPriceFlrLoc:   openrtb_ext.RequestLocation,
			expSchemaVersion: 2,
		},
		{
			name: "Rule selection with Site object, banner|300x600|www.website.com",
			bidRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
					},
					Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{W: &width, H: &height}}},
					Ext: json.RawMessage(`{"prebid":{"floors":{"data":{"currency":"USD","modelgroups":[{"modelversion":"model 1 from req","currency":"USD","values":{"BANNER|300x600|WWW.WEBSITE.COM":5,"*|*|*":7},"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"}}]},"enabled":true,"enforcement":{"enforcepbs":true,"floordeals":true,"enforcerate":100}}}}`),
				},
			},
			account:        testAccountConfig,
			expFloorVal:    5,
			expFloorCur:    "USD",
			expPriceFlrLoc: openrtb_ext.RequestLocation,
		},
		{
			name: "Rule selection with App object, *|*|www.test.com",
			bidRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					App: &openrtb2.App{
						Domain: "www.test.com",
					},
					Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{W: &width, H: &height}}},
					Ext: json.RawMessage(`{"prebid":{"floors":{"data":{"currency":"USD","modelgroups":[{"modelversion":"model 1 from req","currency":"USD","values":{"banner|300x600|www.website.com":5,"*|*|www.test.com":15,"*|*|*":7},"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"}}]},"enabled":true,"enforcement":{"enforcepbs":true,"floordeals":true,"enforcerate":100}}}}`),
				},
			},
			account:        testAccountConfig,
			expFloorVal:    15,
			expFloorCur:    "USD",
			expPriceFlrLoc: openrtb_ext.RequestLocation,
		},
		{
			name: "Floors Signalling not done as req.ext.prebid.floors not provided",
			bidRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					App: &openrtb2.App{
						Domain: "www.test.com",
					},
					Imp: []openrtb2.Imp{{ID: "1234", BidFloor: 10, BidFloorCur: "EUR", Banner: &openrtb2.Banner{W: &width, H: &height}}},
					Ext: json.RawMessage(`{"prebid":{}}`),
				},
			},
			account:        testAccountConfig,
			expFloorVal:    10,
			expFloorCur:    "EUR",
			expPriceFlrLoc: openrtb_ext.NoDataLocation,
			err:            "Empty Floors data",
		},
		{
			name: "BidFloor(USD) Less than MinBidFloor(INR) with different currency",
			bidRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
					},
					Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}}},
					Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":80,"floormincur":"INR","data":{"currency":"USD","modelgroups":[{"modelversion":"model 1 from req","currency":"USD","values":{"banner|300x250|www.website.com":1,"*|*|www.test.com":15,"*|*|*":7},"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"}}]},"enabled":true,"enforcement":{"enforcepbs":true,"floordeals":true,"enforcerate":100}}}}`),
				},
			},
			account:        testAccountConfig,
			expFloorVal:    1.1429,
			expFloorCur:    "USD",
			expPriceFlrLoc: openrtb_ext.RequestLocation,
		},
		{
			name: "BidFloor(INR) Less than MinBidFloor(USD) with different currency",
			bidRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
					},
					Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}}},
					Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"floormincur":"USD","data":{"currency":"INR","modelgroups":[{"modelversion":"model 1 from req","values":{"banner|300x250|www.website.com":60,"*|*|www.test.com":65,"*|*|*":67},"Default":50,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"}}]},"enabled":true,"enforcement":{"enforcepbs":true,"floordeals":true,"enforcerate":100}}}}`),
				},
			},
			account:        testAccountConfig,
			expFloorVal:    70,
			expFloorCur:    "INR",
			expPriceFlrLoc: openrtb_ext.RequestLocation,
		},
		{
			name: "BidFloor is greater than MinBidFloor with same currency",
			bidRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
					},
					Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}}},
					Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"floormincur":"USD","data":{"currency":"USD","modelgroups":[{"modelversion":"model 1 from req","values":{"banner|300x250|www.website.com":2,"*|*|www.test.com":1.5,"*|*|*":1.7},"Default":5,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"}}]},"enabled":true,"enforcement":{"enforcepbs":true,"floordeals":true,"enforcerate":100}}}}`),
				},
			},
			account:        testAccountConfig,
			expFloorVal:    2,
			expFloorCur:    "USD",
			expPriceFlrLoc: openrtb_ext.RequestLocation,
		},
		{
			name: "BidFloor Less than MinBidFloor with same currency",
			bidRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
					},
					Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}}},
					Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":3,"floormincur":"USD","data":{"currency":"USD","modelgroups":[{"modelversion":"model 1 from req","values":{"banner|300x250|www.website.com":2,"*|*|www.test.com":1.5,"*|*|*":1.7},"Default":5,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"}}]},"enabled":true,"enforcement":{"enforcepbs":true,"floordeals":true,"enforcerate":100}}}}`),
				},
			},
			account:        testAccountConfig,
			expFloorVal:    3,
			expFloorCur:    "USD",
			expPriceFlrLoc: openrtb_ext.RequestLocation,
		},
		{
			name: "No Rule matching, default value provided",
			bidRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
					},
					Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}}},
					Ext: json.RawMessage(`{"prebid":{"floors":{"data":{"currency":"USD","modelgroups":[{"modelversion":"model 1 from req","values":{"banner|300x250|www.website.com1":2,"*|*|www.test2.com":1.5},"schema":{"fields":["mediaType","size","domain"]}, "default": 20}]},"enabled":true,"enforcement":{"enforcepbs":true,"floordeals":true,"enforcerate":100}}}}`),
				},
			},
			account:        testAccountConfig,
			expFloorVal:    20,
			expFloorCur:    "USD",
			expPriceFlrLoc: openrtb_ext.RequestLocation,
		},
		{
			name: "No Rule matching, default value less than floorMin",
			bidRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
					},
					Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}}},
					Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":15,"floormincur":"USD","data":{"currency":"USD","modelgroups":[{"modelversion":"model 1 from req","values":{"banner|300x250|www.website.com1":2,"*|*|www.test2.com":1.5},"schema":{"fields":["mediaType","size","domain"]}, "default": 5}]},"enabled":true,"enforcement":{"enforcepbs":true,"floordeals":true,"enforcerate":100}}}}`),
				},
			},
			account:        testAccountConfig,
			expFloorVal:    15,
			expFloorCur:    "USD",
			expPriceFlrLoc: openrtb_ext.RequestLocation,
		},
		{
			name: "imp.bidfloor provided, No Rule matching, MinBidFloor provided",
			bidRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
					},
					Imp: []openrtb2.Imp{{ID: "1234", BidFloor: 100, BidFloorCur: "INR", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}}},
					Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":2,"floormincur":"USD","data":{"currency":"USD","modelgroups":[{"modelversion":"model 1 from req","values":{"banner|300x250|www.website.com1":2,"*|*|www.test2.com":1.5},"schema":{"fields":["mediaType","size","domain"]}}]},"enabled":true,"enforcement":{"enforcepbs":true,"floordeals":true,"enforcerate":100}}}}`),
				},
			},
			account:        testAccountConfig,
			expFloorVal:    100,
			expFloorCur:    "INR",
			expPriceFlrLoc: openrtb_ext.RequestLocation,
		},
		{
			name:              "Empty RequestWrapper ",
			bidRequestWrapper: nil,
			account:           testAccountConfig,
			err:               "Empty bidrequest",
		},
		{
			name: "Invalid Floor Min Currency",
			bidRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
					},
					Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}}},
					Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":80,"floormincur":"ABCD","data":{"currency":"USD","modelgroups":[{"modelversion":"model 1 from req","currency":"USD","values":{"banner|300x250|www.website.com":1,"*|*|www.test.com":15,"*|*|*":7},"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"}}]},"enabled":true,"enforcement":{"enforcepbs":true,"floordeals":true,"enforcerate":100}}}}`),
				},
			},
			account:        testAccountConfig,
			err:            "Error in getting FloorMin value : 'currency: tag is not well-formed'",
			expPriceFlrLoc: openrtb_ext.RequestLocation,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			ErrList := EnrichWithPriceFloors(tc.bidRequestWrapper, tc.account, getCurrencyRates(rates), &mockPriceFloorFetcher{})
			if tc.bidRequestWrapper != nil {
				assert.Equal(t, tc.bidRequestWrapper.Imp[0].BidFloor, tc.expFloorVal, tc.name)
				assert.Equal(t, tc.bidRequestWrapper.Imp[0].BidFloorCur, tc.expFloorCur, tc.name)
				requestExt, err := tc.bidRequestWrapper.GetRequestExt()
				if err == nil {
					if tc.Skipped {
						assert.Equal(t, *requestExt.GetPrebid().Floors.Skipped, tc.Skipped, tc.name)
					} else {
						assert.Equal(t, requestExt.GetPrebid().Floors.PriceFloorLocation, tc.expPriceFlrLoc, tc.name)
						if tc.expSchemaVersion != 0 {
							assert.Equal(t, requestExt.GetPrebid().Floors.Data.FloorsSchemaVersion, tc.expSchemaVersion, tc.name)
						}
					}
				}
			}
			if len(ErrList) > 0 {
				assert.Equal(t, ErrList[0].Error(), tc.err, tc.name)
			}
		})
	}
}

func getTrue() *bool {
	trueFlag := true
	return &trueFlag
}

func getFalse() *bool {
	falseFlag := false
	return &falseFlag
}

type MockFetch struct {
	FakeFetch func(configs config.AccountPriceFloors) (*openrtb_ext.PriceFloorRules, string)
}

func (m *MockFetch) Stop() {}

func (m *MockFetch) Fetch(configs config.AccountPriceFloors) (*openrtb_ext.PriceFloorRules, string) {

	if !configs.UseDynamicData {
		return nil, openrtb_ext.FetchNone
	}
	priceFloors := openrtb_ext.PriceFloorRules{
		Enabled:            getTrue(),
		PriceFloorLocation: openrtb_ext.RequestLocation,
		Enforcement: &openrtb_ext.PriceFloorEnforcement{
			EnforcePBS:  getTrue(),
			EnforceRate: 100,
			FloorDeals:  getTrue(),
		},
		Data: &openrtb_ext.PriceFloorData{
			Currency: "USD",
			ModelGroups: []openrtb_ext.PriceFloorModelGroup{
				{
					ModelVersion: "Version 101",
					Currency:     "USD",
					Values: map[string]float64{
						"banner|300x600|www.website5.com": 15,
						"*|*|*":                           25,
					},
					Schema: openrtb_ext.PriceFloorSchema{
						Fields:    []string{"mediaType", "size", "domain"},
						Delimiter: "|",
					},
				},
			},
		},
	}
	return &priceFloors, openrtb_ext.FetchSuccess
}

func TestResolveFloors(t *testing.T) {
	rates := map[string]map[string]float64{
		"USD": {
			"INR": 70,
			"EUR": 0.9,
			"JPY": 5.09,
		},
	}

	testCases := []struct {
		name              string
		bidRequestWrapper *openrtb_ext.RequestWrapper
		account           config.Account
		conversions       currency.Conversions
		fetcher           FloorFetcher
		expErr            []error
		expFloors         *openrtb_ext.PriceFloorRules
	}{
		{
			name: "Dynamic fetch disabled, floors from request selected",
			bidRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
					},
					Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}}},
					Ext: json.RawMessage(`{"prebid":{"floors":{"data":{"currency":"USD","modelgroups":[{"modelversion":"model 1 from req","currency":"USD","values":{"banner|300x600|www.website5.com":5,"*|*|*":7},"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"}}]},"enabled":true,"enforcement":{"enforcepbs":true,"floordeals":true,"enforcerate":100}}}}`),
				},
			},
			account: config.Account{
				PriceFloors: config.AccountPriceFloors{
					Enabled:        true,
					UseDynamicData: false,
				},
			},
			fetcher: &MockFetch{},
			expFloors: &openrtb_ext.PriceFloorRules{
				Enabled:            getTrue(),
				FetchStatus:        openrtb_ext.FetchNone,
				PriceFloorLocation: openrtb_ext.RequestLocation,
				Enforcement: &openrtb_ext.PriceFloorEnforcement{
					EnforcePBS:  getTrue(),
					EnforceRate: 100,
					FloorDeals:  getTrue(),
				},
				Data: &openrtb_ext.PriceFloorData{
					Currency: "USD",
					ModelGroups: []openrtb_ext.PriceFloorModelGroup{
						{
							ModelVersion: "model 1 from req",
							Currency:     "USD",
							Values: map[string]float64{
								"banner|300x600|www.website5.com": 5,
								"*|*|*":                           7,
							},
							Schema: openrtb_ext.PriceFloorSchema{
								Fields:    []string{"mediaType", "size", "domain"},
								Delimiter: "|",
							},
						},
					},
				},
			},
		},
		{
			name: "Dynamic fetch enabled, floors from fetched selected",
			bidRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
					},
					Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}}},
				},
			},
			account: config.Account{
				PriceFloors: config.AccountPriceFloors{
					Enabled:        true,
					UseDynamicData: true,
				},
			},
			fetcher: &MockFetch{},
			expFloors: &openrtb_ext.PriceFloorRules{
				Enabled:            getTrue(),
				FetchStatus:        openrtb_ext.FetchSuccess,
				PriceFloorLocation: openrtb_ext.FetchLocation,
				Enforcement: &openrtb_ext.PriceFloorEnforcement{
					EnforcePBS:  getTrue(),
					FloorDeals:  getTrue(),
					EnforceRate: 100,
				},
				Data: &openrtb_ext.PriceFloorData{
					Currency: "USD",
					ModelGroups: []openrtb_ext.PriceFloorModelGroup{
						{
							ModelVersion: "Version 101",
							Currency:     "USD",
							Values: map[string]float64{
								"banner|300x600|www.website5.com": 15,
								"*|*|*":                           25,
							},
							Schema: openrtb_ext.PriceFloorSchema{
								Fields:    []string{"mediaType", "size", "domain"},
								Delimiter: "|",
							},
						},
					},
				},
			},
		},
		{
			name: "Dynamic fetch enabled, floors formed after merging",
			bidRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
					},
					Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}}},
					Ext: json.RawMessage(`{"prebid":{"floors":{"floormincur":"EUR","enabled":true,"data":{"currency":"USD","modelgroups":[{"modelversion":"model 1 from req","currency":"USD","values":{"banner|300x600|www.website5.com":5,"*|*|*":7},"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"}}]},"floormin":10.11,"enforcement":{"enforcepbs":true,"floordeals":true,"enforcerate":100}}}}`),
				},
			},
			account: config.Account{
				PriceFloors: config.AccountPriceFloors{
					Enabled:        true,
					UseDynamicData: true,
				},
			},
			fetcher: &MockFetch{},
			expFloors: &openrtb_ext.PriceFloorRules{
				Enabled:            getTrue(),
				FloorMin:           10.11,
				FloorMinCur:        "EUR",
				FetchStatus:        openrtb_ext.FetchSuccess,
				PriceFloorLocation: openrtb_ext.FetchLocation,
				Enforcement: &openrtb_ext.PriceFloorEnforcement{
					EnforcePBS:  getTrue(),
					EnforceRate: 100,
					FloorDeals:  getTrue(),
				},
				Data: &openrtb_ext.PriceFloorData{
					Currency: "USD",
					ModelGroups: []openrtb_ext.PriceFloorModelGroup{
						{
							ModelVersion: "Version 101",
							Currency:     "USD",
							Values: map[string]float64{
								"banner|300x600|www.website5.com": 15,
								"*|*|*":                           25,
							},
							Schema: openrtb_ext.PriceFloorSchema{
								Fields:    []string{"mediaType", "size", "domain"},
								Delimiter: "|",
							},
						},
					},
				},
			},
		},
		{
			name: "Dynamic fetch disabled, only enforcement object present in req.ext",
			bidRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
					},
					Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}}},
					Ext: json.RawMessage(`{"prebid":{"floors":{"enforcement":{"enforcepbs":true,"floordeals":true,"enforcerate":100}}}}`),
				},
			},
			account: config.Account{
				PriceFloors: config.AccountPriceFloors{
					Enabled:        true,
					UseDynamicData: false,
				},
			},
			fetcher: &MockFetch{},
			expFloors: &openrtb_ext.PriceFloorRules{
				Enforcement: &openrtb_ext.PriceFloorEnforcement{
					EnforcePBS:  getTrue(),
					EnforceRate: 100,
					FloorDeals:  getTrue(),
				},
				FetchStatus:        openrtb_ext.FetchNone,
				PriceFloorLocation: openrtb_ext.RequestLocation,
			},
		},
		{
			name: "Dynamic fetch enabled, floors from fetched selected and new URL is updated",
			bidRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
					},
					Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}}},
					Ext: json.RawMessage(`{"prebid":{"floors":{"floorendpoint":{"url":"http://test.com/floor"},"enabled":true}}}`),
				},
			},
			account: config.Account{
				PriceFloors: config.AccountPriceFloors{
					Enabled:        true,
					UseDynamicData: true,
				},
			},
			fetcher: &MockFetch{},
			expFloors: &openrtb_ext.PriceFloorRules{
				Enabled:            getTrue(),
				FetchStatus:        openrtb_ext.FetchSuccess,
				PriceFloorLocation: openrtb_ext.FetchLocation,
				Enforcement: &openrtb_ext.PriceFloorEnforcement{
					EnforcePBS:  getTrue(),
					FloorDeals:  getTrue(),
					EnforceRate: 100,
				},
				Location: &openrtb_ext.PriceFloorEndpoint{
					URL: "http://test.com/floor",
				},
				Data: &openrtb_ext.PriceFloorData{
					Currency: "USD",
					ModelGroups: []openrtb_ext.PriceFloorModelGroup{
						{
							ModelVersion: "Version 101",
							Currency:     "USD",
							Values: map[string]float64{
								"banner|300x600|www.website5.com": 15,
								"*|*|*":                           25,
							},
							Schema: openrtb_ext.PriceFloorSchema{
								Fields:    []string{"mediaType", "size", "domain"},
								Delimiter: "|",
							},
						},
					},
				},
			},
		},
		{
			name: "Dynamic Fetch Enabled but price floor fetcher is nil, floors from request is selected",
			bidRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
					},
					Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}}},
					Ext: json.RawMessage(`{"prebid":{"floors":{"data":{"currency":"USD","modelgroups":[{"modelversion":"model 1 from req","currency":"USD","values":{"banner|300x600|www.website5.com":5,"*|*|*":7},"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"}}]},"enabled":true,"enforcement":{"enforcepbs":true,"floordeals":true,"enforcerate":100}}}}`),
				},
			},
			account: config.Account{
				PriceFloors: config.AccountPriceFloors{
					Enabled:        true,
					UseDynamicData: true,
				},
			},
			fetcher: nil,
			expFloors: &openrtb_ext.PriceFloorRules{
				Enabled:            getTrue(),
				FetchStatus:        openrtb_ext.FetchNone,
				PriceFloorLocation: openrtb_ext.RequestLocation,
				Enforcement: &openrtb_ext.PriceFloorEnforcement{
					EnforcePBS:  getTrue(),
					EnforceRate: 100,
					FloorDeals:  getTrue(),
				},
				Data: &openrtb_ext.PriceFloorData{
					Currency: "USD",
					ModelGroups: []openrtb_ext.PriceFloorModelGroup{
						{
							ModelVersion: "model 1 from req",
							Currency:     "USD",
							Values: map[string]float64{
								"banner|300x600|www.website5.com": 5,
								"*|*|*":                           7,
							},
							Schema: openrtb_ext.PriceFloorSchema{
								Fields:    []string{"mediaType", "size", "domain"},
								Delimiter: "|",
							},
						},
					},
				},
			},
		},
		{
			name: "Dynamic Fetch Enabled but price floor fetcher is nil and request has no floors",
			bidRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
					},
					Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}}},
					Ext: json.RawMessage(`{"prebid":{}}`),
				},
			},
			account: config.Account{
				PriceFloors: config.AccountPriceFloors{
					Enabled:        true,
					UseDynamicData: true,
				},
			},
			fetcher: nil,
			expFloors: &openrtb_ext.PriceFloorRules{
				FetchStatus:        openrtb_ext.FetchNone,
				PriceFloorLocation: openrtb_ext.NoDataLocation,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resolvedFloors, _ := resolveFloors(tc.account, tc.bidRequestWrapper, getCurrencyRates(rates), tc.fetcher)
			assert.Equal(t, resolvedFloors, tc.expFloors, tc.name)
		})
	}
}

type MockFetchDataRate0 struct{}

func (m *MockFetchDataRate0) Fetch(configs config.AccountPriceFloors) (*openrtb_ext.PriceFloorRules, string) {

	if !configs.UseDynamicData {
		return nil, openrtb_ext.FetchNone
	}
	priceFloors := openrtb_ext.PriceFloorRules{
		Enabled:            getTrue(),
		PriceFloorLocation: openrtb_ext.RequestLocation,
		Enforcement: &openrtb_ext.PriceFloorEnforcement{
			EnforcePBS:  getTrue(),
			EnforceRate: 100,
			FloorDeals:  getTrue(),
		},
		Data: &openrtb_ext.PriceFloorData{
			Currency: "USD",
			ModelGroups: []openrtb_ext.PriceFloorModelGroup{
				{
					ModelVersion: "model from fetched",
					Currency:     "USD",
					Values: map[string]float64{
						"banner|300x600|www.website5.com": 15,
						"*|*|*":                           25,
					},
					Schema: openrtb_ext.PriceFloorSchema{
						Fields: []string{"mediaType", "size", "domain"},
					},
				},
			},
			UseFetchDataRate: ptrutil.ToPtr(0),
		},
	}
	return &priceFloors, openrtb_ext.FetchSuccess
}

func (m *MockFetchDataRate0) Stop() {

}

type MockFetchDataRate100 struct{}

func (m *MockFetchDataRate100) Fetch(configs config.AccountPriceFloors) (*openrtb_ext.PriceFloorRules, string) {

	if !configs.UseDynamicData {
		return nil, openrtb_ext.FetchNone
	}
	priceFloors := openrtb_ext.PriceFloorRules{
		Enabled:            getTrue(),
		PriceFloorLocation: openrtb_ext.RequestLocation,
		Enforcement: &openrtb_ext.PriceFloorEnforcement{
			EnforcePBS:  getTrue(),
			EnforceRate: 100,
			FloorDeals:  getTrue(),
		},
		Data: &openrtb_ext.PriceFloorData{
			Currency: "USD",
			ModelGroups: []openrtb_ext.PriceFloorModelGroup{
				{
					ModelVersion: "model from fetched",
					Currency:     "USD",
					Values: map[string]float64{
						"banner|300x600|www.website5.com": 15,
						"*|*|*":                           25,
					},
					Schema: openrtb_ext.PriceFloorSchema{
						Fields: []string{"mediaType", "size", "domain"},
					},
				},
			},
			UseFetchDataRate: ptrutil.ToPtr(100),
		},
	}
	return &priceFloors, openrtb_ext.FetchSuccess
}

func (m *MockFetchDataRate100) Stop() {

}

type MockFetchDataRateNotProvided struct{}

func (m *MockFetchDataRateNotProvided) Fetch(configs config.AccountPriceFloors) (*openrtb_ext.PriceFloorRules, string) {

	if !configs.UseDynamicData {
		return nil, openrtb_ext.FetchNone
	}
	priceFloors := openrtb_ext.PriceFloorRules{
		Enabled:            getTrue(),
		PriceFloorLocation: openrtb_ext.RequestLocation,
		Enforcement: &openrtb_ext.PriceFloorEnforcement{
			EnforcePBS:  getTrue(),
			EnforceRate: 100,
			FloorDeals:  getTrue(),
		},
		Data: &openrtb_ext.PriceFloorData{
			Currency: "USD",
			ModelGroups: []openrtb_ext.PriceFloorModelGroup{
				{
					ModelVersion: "model from fetched",
					Currency:     "USD",
					Values: map[string]float64{
						"banner|300x600|www.website5.com": 5,
						"*|*|*":                           15,
					},
					Schema: openrtb_ext.PriceFloorSchema{
						Fields: []string{"mediaType", "size", "domain"},
					},
				},
			},
		},
	}
	return &priceFloors, openrtb_ext.FetchSuccess
}

func (m *MockFetchDataRateNotProvided) Stop() {

}

func TestResolveFloorsWithUseDataRate(t *testing.T) {
	rates := map[string]map[string]float64{}

	testCases := []struct {
		name              string
		bidRequestWrapper *openrtb_ext.RequestWrapper
		account           config.Account
		conversions       currency.Conversions
		expErr            []error
		expFloors         *openrtb_ext.PriceFloorRules
		fetcher           FloorFetcher
	}{
		{
			name:    "Dynamic fetch enabled, floors from request selected as data rate 0",
			fetcher: &MockFetchDataRate0{},
			bidRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
					},
					Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}}},
					Ext: json.RawMessage(`{"prebid":{"floors":{"data":{"currency":"USD","modelgroups":[{"modelversion":"model 1 from req","currency":"USD","values":{"banner|300x600|www.website5.com":5,"*|*|*":7},"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"}}]},"enabled":true,"enforcement":{"enforcepbs":true,"floordeals":true,"enforcerate":100}}}}`),
				},
			},
			account: config.Account{
				PriceFloors: config.AccountPriceFloors{
					Enabled:        true,
					UseDynamicData: true,
				},
			},
			expFloors: &openrtb_ext.PriceFloorRules{
				Enabled:            getTrue(),
				FetchStatus:        openrtb_ext.FetchNone,
				PriceFloorLocation: openrtb_ext.RequestLocation,
				Enforcement: &openrtb_ext.PriceFloorEnforcement{
					EnforcePBS:  getTrue(),
					FloorDeals:  getTrue(),
					EnforceRate: 100,
				},
				Data: &openrtb_ext.PriceFloorData{
					Currency: "USD",
					ModelGroups: []openrtb_ext.PriceFloorModelGroup{
						{
							ModelVersion: "model 1 from req",
							Currency:     "USD",
							Values: map[string]float64{
								"banner|300x600|www.website5.com": 5,
								"*|*|*":                           7,
							},
							Schema: openrtb_ext.PriceFloorSchema{
								Fields:    []string{"mediaType", "size", "domain"},
								Delimiter: "|",
							},
						},
					},
				},
			},
		},
		{
			name:    "Dynamic fetch enabled, floors from fetched selected as data rate is 100",
			fetcher: &MockFetchDataRate100{},
			bidRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
					},
					Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}}},
				},
			},
			account: config.Account{
				PriceFloors: config.AccountPriceFloors{
					Enabled:        true,
					UseDynamicData: true,
				},
			},
			expFloors: &openrtb_ext.PriceFloorRules{
				Enabled:            getTrue(),
				FetchStatus:        openrtb_ext.FetchSuccess,
				PriceFloorLocation: openrtb_ext.FetchLocation,
				Enforcement: &openrtb_ext.PriceFloorEnforcement{
					EnforcePBS:  getTrue(),
					FloorDeals:  getTrue(),
					EnforceRate: 100,
				},
				Data: &openrtb_ext.PriceFloorData{
					Currency: "USD",
					ModelGroups: []openrtb_ext.PriceFloorModelGroup{
						{
							ModelVersion: "model from fetched",
							Currency:     "USD",
							Values: map[string]float64{
								"banner|300x600|www.website5.com": 15,
								"*|*|*":                           25,
							},
							Schema: openrtb_ext.PriceFloorSchema{
								Fields: []string{"mediaType", "size", "domain"},
							},
						},
					},
					UseFetchDataRate: ptrutil.ToPtr(100),
				},
			},
		},
		{
			name:    "Dynamic fetch enabled, floors from fetched selected as data rate not provided as default value = 100",
			fetcher: &MockFetchDataRateNotProvided{},
			bidRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{
						Publisher: &openrtb2.Publisher{Domain: "www.website.com"},
					},
					Imp: []openrtb2.Imp{{ID: "1234", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}}},
					Ext: json.RawMessage(`{"prebid":{"floors":{"data":{"currency":"USD","modelgroups":[{"modelversion":"model 1 from req","currency":"USD","values":{"banner|300x600|www.website5.com":5,"*|*|*":7},"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"}}]},"enabled":true,"enforcement":{"enforcepbs":true,"floordeals":true,"enforcerate":100}}}}`),
				},
			},
			account: config.Account{
				PriceFloors: config.AccountPriceFloors{
					Enabled:        true,
					UseDynamicData: true,
				},
			},
			expFloors: &openrtb_ext.PriceFloorRules{
				Enabled:            getTrue(),
				FetchStatus:        openrtb_ext.FetchSuccess,
				PriceFloorLocation: openrtb_ext.FetchLocation,
				Enforcement: &openrtb_ext.PriceFloorEnforcement{
					EnforcePBS:  getTrue(),
					FloorDeals:  getTrue(),
					EnforceRate: 100,
				},
				Data: &openrtb_ext.PriceFloorData{
					Currency: "USD",
					ModelGroups: []openrtb_ext.PriceFloorModelGroup{
						{
							ModelVersion: "model from fetched",
							Currency:     "USD",
							Values: map[string]float64{
								"banner|300x600|www.website5.com": 5,
								"*|*|*":                           15,
							},
							Schema: openrtb_ext.PriceFloorSchema{
								Fields: []string{"mediaType", "size", "domain"},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resolvedFloors, _ := resolveFloors(tc.account, tc.bidRequestWrapper, getCurrencyRates(rates), tc.fetcher)
			assert.Equal(t, resolvedFloors, tc.expFloors, tc.name)
		})
	}
}

func printFloors(floors *openrtb_ext.PriceFloorRules) string {
	fbytes, _ := jsonutil.Marshal(floors)
	return string(fbytes)
}

func TestCreateFloorsFrom(t *testing.T) {

	testAccountConfig := config.Account{
		PriceFloors: config.AccountPriceFloors{
			Enabled:        true,
			UseDynamicData: false,
			MaxRule:        100,
			MaxSchemaDims:  5,
		},
	}

	type args struct {
		floors        *openrtb_ext.PriceFloorRules
		account       config.Account
		fetchStatus   string
		floorLocation string
	}
	testCases := []struct {
		name  string
		args  args
		want  *openrtb_ext.PriceFloorRules
		want1 []error
	}{
		{
			name: "floor provider should be selected from floor json",
			args: args{
				account: testAccountConfig,
				floors: &openrtb_ext.PriceFloorRules{
					Enabled:     getTrue(),
					FloorMin:    10.11,
					FloorMinCur: "EUR",
					Enforcement: &openrtb_ext.PriceFloorEnforcement{
						EnforcePBS:  getTrue(),
						EnforceRate: 100,
						FloorDeals:  getTrue(),
					},
					Data: &openrtb_ext.PriceFloorData{
						Currency: "USD",
						ModelGroups: []openrtb_ext.PriceFloorModelGroup{
							{
								ModelVersion: "model from fetched",
								Currency:     "USD",
								Values: map[string]float64{
									"banner|300x600|www.website5.com": 15,
									"*|*|*":                           25,
								},
								Schema: openrtb_ext.PriceFloorSchema{
									Fields: []string{"mediaType", "size", "domain"},
								},
							},
						},
						FloorProvider: "PM",
					},
				},
				fetchStatus:   openrtb_ext.FetchSuccess,
				floorLocation: openrtb_ext.FetchLocation,
			},
			want: &openrtb_ext.PriceFloorRules{
				Enabled:            getTrue(),
				FloorMin:           10.11,
				FloorMinCur:        "EUR",
				FetchStatus:        openrtb_ext.FetchSuccess,
				PriceFloorLocation: openrtb_ext.FetchLocation,
				Enforcement: &openrtb_ext.PriceFloorEnforcement{
					EnforcePBS:  getTrue(),
					EnforceRate: 100,
					FloorDeals:  getTrue(),
				},
				Data: &openrtb_ext.PriceFloorData{
					Currency: "USD",
					ModelGroups: []openrtb_ext.PriceFloorModelGroup{
						{
							ModelVersion: "model from fetched",
							Currency:     "USD",
							Values: map[string]float64{
								"banner|300x600|www.website5.com": 15,
								"*|*|*":                           25,
							},
							Schema: openrtb_ext.PriceFloorSchema{
								Fields: []string{"mediaType", "size", "domain"},
							},
						},
					},
					FloorProvider: "PM",
				},
			},
		},
		{
			name: "floor provider will be empty if no value provided in floor json",
			args: args{
				account: testAccountConfig,
				floors: &openrtb_ext.PriceFloorRules{
					Enabled:     getTrue(),
					FloorMin:    10.11,
					FloorMinCur: "EUR",
					Enforcement: &openrtb_ext.PriceFloorEnforcement{
						EnforcePBS:  getTrue(),
						EnforceRate: 100,
						FloorDeals:  getTrue(),
					},
					Data: &openrtb_ext.PriceFloorData{
						Currency: "USD",
						ModelGroups: []openrtb_ext.PriceFloorModelGroup{
							{
								ModelVersion: "model from request",
								Currency:     "USD",
								Values: map[string]float64{
									"banner|300x600|www.website5.com": 15,
									"*|*|*":                           25,
								},
								Schema: openrtb_ext.PriceFloorSchema{
									Fields: []string{"mediaType", "size", "domain"},
								},
							},
						},
						FloorProvider: "",
					},
				},
				fetchStatus:   openrtb_ext.FetchSuccess,
				floorLocation: openrtb_ext.FetchLocation,
			},
			want: &openrtb_ext.PriceFloorRules{
				Enabled:            getTrue(),
				FloorMin:           10.11,
				FloorMinCur:        "EUR",
				FetchStatus:        openrtb_ext.FetchSuccess,
				PriceFloorLocation: openrtb_ext.FetchLocation,
				Enforcement: &openrtb_ext.PriceFloorEnforcement{
					EnforcePBS:  getTrue(),
					EnforceRate: 100,
					FloorDeals:  getTrue(),
				},
				Data: &openrtb_ext.PriceFloorData{
					Currency: "USD",
					ModelGroups: []openrtb_ext.PriceFloorModelGroup{
						{
							ModelVersion: "model from request",
							Currency:     "USD",
							Values: map[string]float64{
								"banner|300x600|www.website5.com": 15,
								"*|*|*":                           25,
							},
							Schema: openrtb_ext.PriceFloorSchema{
								Fields: []string{"mediaType", "size", "domain"},
							},
						},
					},
					FloorProvider: "",
				},
			},
		},
		{
			name: "only floor enforcement object present",
			args: args{
				account: testAccountConfig,
				floors: &openrtb_ext.PriceFloorRules{
					Enabled: getTrue(),
					Enforcement: &openrtb_ext.PriceFloorEnforcement{
						EnforcePBS:  getTrue(),
						EnforceRate: 100,
						FloorDeals:  getTrue(),
					},
				},
				fetchStatus:   openrtb_ext.FetchNone,
				floorLocation: openrtb_ext.RequestLocation,
			},
			want: &openrtb_ext.PriceFloorRules{
				FetchStatus:        openrtb_ext.FetchNone,
				PriceFloorLocation: openrtb_ext.RequestLocation,
				Enforcement: &openrtb_ext.PriceFloorEnforcement{
					EnforcePBS:  getTrue(),
					EnforceRate: 100,
					FloorDeals:  getTrue(),
				},
			},
		},
		{
			name: "Invalid modelGroup with skipRate = 110",
			args: args{
				account: testAccountConfig,
				floors: &openrtb_ext.PriceFloorRules{
					Enabled: getTrue(),
					Data: &openrtb_ext.PriceFloorData{
						Currency: "USD",
						ModelGroups: []openrtb_ext.PriceFloorModelGroup{
							{
								ModelVersion: "model from fetched",
								Currency:     "USD",
								SkipRate:     110,
								Values: map[string]float64{
									"banner|300x600|www.website5.com": 15,
									"*|*|*":                           25,
								},
								Schema: openrtb_ext.PriceFloorSchema{
									Fields: []string{"mediaType", "size", "domain"},
								},
							},
						},
					},
				},
				fetchStatus:   openrtb_ext.FetchNone,
				floorLocation: openrtb_ext.RequestLocation,
			},
			want: &openrtb_ext.PriceFloorRules{
				FetchStatus:        openrtb_ext.FetchNone,
				PriceFloorLocation: openrtb_ext.RequestLocation,
			},
			want1: []error{
				errors.New("Invalid Floor Model = 'model from fetched' due to SkipRate = '110' is out of range (1-100)"),
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, got1 := createFloorsFrom(tc.args.floors, tc.args.account, tc.args.fetchStatus, tc.args.floorLocation)
			assert.Equal(t, got1, tc.want1, tc.name)
			assert.Equal(t, got, tc.want, tc.name)
		})
	}
}

func TestIsPriceFloorsEnabled(t *testing.T) {
	type args struct {
		account           config.Account
		bidRequestWrapper *openrtb_ext.RequestWrapper
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Disabled in account and req",
			args: args{
				account: config.Account{PriceFloors: config.AccountPriceFloors{Enabled: false}},
				bidRequestWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						Ext: json.RawMessage(`{"prebid":{"floors":{"enabled": false} }}`),
					},
				},
			},
			want: false,
		},
		{
			name: "Enabled  in account and req",
			args: args{
				account: config.Account{PriceFloors: config.AccountPriceFloors{Enabled: true}},
				bidRequestWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						Ext: json.RawMessage(`{"prebid":{"floors":{"enabled": true} }}`),
					},
				},
			},
			want: true,
		},
		{
			name: "disabled  in account and enabled req",
			args: args{
				account: config.Account{PriceFloors: config.AccountPriceFloors{Enabled: false}},
				bidRequestWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						Ext: json.RawMessage(`{"prebid":{"floors":{"enabled": true} }}`),
					},
				},
			},
			want: false,
		},
		{
			name: "Enabled  in account and disabled in req",
			args: args{
				account: config.Account{PriceFloors: config.AccountPriceFloors{Enabled: true}},
				bidRequestWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						Ext: json.RawMessage(`{"prebid":{"floors":{"enabled": false} }}`)},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPriceFloorsEnabled(tt.args.account, tt.args.bidRequestWrapper)
			assert.Equal(t, got, tt.want, tt.name)
		})
	}
}

func TestResolveFloorMin(t *testing.T) {
	rates := map[string]map[string]float64{
		"USD": {
			"INR": 70,
			"EUR": 0.9,
			"JPY": 5.09,
		},
	}

	tt := []struct {
		name        string
		reqFloors   openrtb_ext.PriceFloorRules
		fetchFloors openrtb_ext.PriceFloorRules
		conversions currency.Conversions
		expPrice    Price
	}{
		{
			name: "FloorsMin present in request Floors only",
			reqFloors: openrtb_ext.PriceFloorRules{
				FloorMin:    10,
				FloorMinCur: "JPY",
			},
			fetchFloors: openrtb_ext.PriceFloorRules{},
			expPrice:    Price{FloorMin: 10, FloorMinCur: "JPY"},
		},
		{
			name: "FloorsMin, FloorMinCur and data currency present in request Floors",
			reqFloors: openrtb_ext.PriceFloorRules{
				FloorMin:    10,
				FloorMinCur: "JPY",
				Data: &openrtb_ext.PriceFloorData{
					Currency: "USD",
				},
			},
			fetchFloors: openrtb_ext.PriceFloorRules{},
			expPrice:    Price{FloorMin: 10, FloorMinCur: "JPY"},
		},
		{
			name: "FloorsMin and data currency present in request Floors",
			reqFloors: openrtb_ext.PriceFloorRules{
				FloorMin: 10,
				Data: &openrtb_ext.PriceFloorData{
					Currency: "USD",
				},
			},
			fetchFloors: openrtb_ext.PriceFloorRules{},
			expPrice:    Price{FloorMin: 10, FloorMinCur: "USD"},
		},
		{
			name: "FloorsMin and FloorMinCur present in request Floors and fetched floors",
			reqFloors: openrtb_ext.PriceFloorRules{
				FloorMin:    10,
				FloorMinCur: "USD",
			},
			fetchFloors: openrtb_ext.PriceFloorRules{
				FloorMin:    15,
				FloorMinCur: "INR",
			},
			expPrice: Price{FloorMin: 10, FloorMinCur: "USD"},
		},
		{
			name:      "FloorsMin present fetched floors only",
			reqFloors: openrtb_ext.PriceFloorRules{},
			fetchFloors: openrtb_ext.PriceFloorRules{
				FloorMin:    15,
				FloorMinCur: "EUR",
			},
			expPrice: Price{FloorMin: 15, FloorMinCur: "EUR"},
		},
		{
			name: "FloorMinCur present in reqFloors And FloorsMin, FloorMinCur present in fetched floors (Same Currency)",
			reqFloors: openrtb_ext.PriceFloorRules{
				FloorMinCur: "EUR",
			},
			fetchFloors: openrtb_ext.PriceFloorRules{
				FloorMin:    15,
				FloorMinCur: "EUR",
			},
			expPrice: Price{FloorMin: 15, FloorMinCur: "EUR"},
		},
		{
			name: "FloorMinCur present in reqFloors And FloorsMin, FloorMinCur present in fetched floors (Different Currency)",
			reqFloors: openrtb_ext.PriceFloorRules{
				FloorMinCur: "USD",
			},
			fetchFloors: openrtb_ext.PriceFloorRules{
				FloorMin:    15,
				FloorMinCur: "EUR",
			},
			expPrice: Price{FloorMin: 16.6667, FloorMinCur: "USD"},
		},
		{
			name: "FloorMin present in reqFloors And FloorMinCur present in fetched floors",
			reqFloors: openrtb_ext.PriceFloorRules{
				FloorMin: 11,
			},
			fetchFloors: openrtb_ext.PriceFloorRules{
				FloorMinCur: "EUR",
			},
			expPrice: Price{FloorMin: 11, FloorMinCur: "EUR"},
		},
		{
			name: "FloorMinCur present in reqFloors And FloorMin present in fetched floors",
			reqFloors: openrtb_ext.PriceFloorRules{
				FloorMinCur: "INR",
			},
			fetchFloors: openrtb_ext.PriceFloorRules{
				FloorMin: 12,
			},
			expPrice: Price{FloorMin: 12, FloorMinCur: "INR"},
		},
		{
			name: "FloorMinCur present in reqFloors And FloorMin, data currency present in fetched floors",
			reqFloors: openrtb_ext.PriceFloorRules{
				FloorMinCur: "INR",
			},
			fetchFloors: openrtb_ext.PriceFloorRules{
				FloorMin: 1,
				Data:     &openrtb_ext.PriceFloorData{Currency: "USD"},
			},
			expPrice: Price{FloorMin: 70, FloorMinCur: "INR"},
		},
		{
			name: "FloorMinCur present in fetched Floors And data currency present in reqFloors",
			reqFloors: openrtb_ext.PriceFloorRules{
				FloorMin: 2,
			},
			fetchFloors: openrtb_ext.PriceFloorRules{
				Data: &openrtb_ext.PriceFloorData{Currency: "USD"},
			},
			expPrice: Price{FloorMin: 2, FloorMinCur: "USD"},
		},
		{
			name:      "Data currency and FloorMin present in fetched floors",
			reqFloors: openrtb_ext.PriceFloorRules{},
			fetchFloors: openrtb_ext.PriceFloorRules{
				FloorMin: 12,
				Data:     &openrtb_ext.PriceFloorData{Currency: "USD"},
			},
			expPrice: Price{FloorMin: 12, FloorMinCur: "USD"},
		},
		{
			name:        "Empty reqFloors And Empty fetched floors",
			reqFloors:   openrtb_ext.PriceFloorRules{},
			fetchFloors: openrtb_ext.PriceFloorRules{},
			expPrice:    Price{FloorMin: 0.0, FloorMinCur: ""},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			price := resolveFloorMin(&tc.reqFloors, &tc.fetchFloors, getCurrencyRates(rates))
			if !reflect.DeepEqual(price.FloorMin, tc.expPrice.FloorMin) {
				t.Errorf("Floor Value error: \nreturn:\t%v\nwant:\t%v", price.FloorMin, tc.expPrice.FloorMin)
			}
			if !reflect.DeepEqual(price.FloorMinCur, tc.expPrice.FloorMinCur) {
				t.Errorf("Floor Currency error: \nreturn:\t%v\nwant:\t%v", price.FloorMinCur, tc.expPrice.FloorMinCur)
			}

		})
	}
}

func TestMergeFloors(t *testing.T) {

	rates := map[string]map[string]float64{
		"USD": {
			"INR": 70,
			"EUR": 0.9,
			"JPY": 5.09,
		},
	}

	type args struct {
		reqFloors   *openrtb_ext.PriceFloorRules
		fetchFloors *openrtb_ext.PriceFloorRules
	}
	tests := []struct {
		name string
		args args
		want *openrtb_ext.PriceFloorRules
	}{
		{
			name: "Fetched Floors are present and request Floors are empty",
			args: args{
				reqFloors: nil,
				fetchFloors: &openrtb_ext.PriceFloorRules{
					Enabled: getTrue(),
					Data: &openrtb_ext.PriceFloorData{
						Currency: "INR",
						SkipRate: 0,
						ModelGroups: []openrtb_ext.PriceFloorModelGroup{
							{
								ModelVersion: "Version 1",
								Currency:     "INR",
								Values: map[string]float64{
									"banner|300x600|www.website5.com": 20,
									"*|*|*":                           50,
								},
								Schema: openrtb_ext.PriceFloorSchema{
									Fields:    []string{"mediaType", "size", "domain"},
									Delimiter: "|",
								},
							},
						},
					},
				},
			},
			want: &openrtb_ext.PriceFloorRules{
				Enabled: getTrue(),
				Data: &openrtb_ext.PriceFloorData{
					Currency: "INR",
					SkipRate: 0,
					ModelGroups: []openrtb_ext.PriceFloorModelGroup{
						{
							ModelVersion: "Version 1",
							Currency:     "INR",
							Values: map[string]float64{
								"banner|300x600|www.website5.com": 20,
								"*|*|*":                           50,
							},
							Schema: openrtb_ext.PriceFloorSchema{
								Fields:    []string{"mediaType", "size", "domain"},
								Delimiter: "|",
							},
						},
					},
				},
			},
		},
		{
			name: "Fetched Floors are present and request Floors has floors disabled",
			args: args{
				reqFloors: &openrtb_ext.PriceFloorRules{
					Enabled: getFalse(),
				},
				fetchFloors: &openrtb_ext.PriceFloorRules{
					Enabled: getTrue(),
					Data: &openrtb_ext.PriceFloorData{
						Currency: "INR",
						SkipRate: 0,
						ModelGroups: []openrtb_ext.PriceFloorModelGroup{
							{
								ModelVersion: "Version 1",
								Currency:     "INR",
								Values: map[string]float64{
									"banner|300x600|www.website5.com": 20,
									"*|*|*":                           50,
								},
								Schema: openrtb_ext.PriceFloorSchema{
									Fields:    []string{"mediaType", "size", "domain"},
									Delimiter: "|",
								},
							},
						},
					},
				},
			},
			want: &openrtb_ext.PriceFloorRules{
				Enabled: getFalse(),
				Data: &openrtb_ext.PriceFloorData{
					Currency: "INR",
					SkipRate: 0,
					ModelGroups: []openrtb_ext.PriceFloorModelGroup{
						{
							ModelVersion: "Version 1",
							Currency:     "INR",
							Values: map[string]float64{
								"banner|300x600|www.website5.com": 20,
								"*|*|*":                           50,
							},
							Schema: openrtb_ext.PriceFloorSchema{
								Fields:    []string{"mediaType", "size", "domain"},
								Delimiter: "|",
							},
						},
					},
				},
			},
		},
		{
			name: "Fetched Floors are present and request Floors has enforcement (enforcepbs = true)",
			args: args{
				reqFloors: &openrtb_ext.PriceFloorRules{
					Enabled: getTrue(),
					Enforcement: &openrtb_ext.PriceFloorEnforcement{
						EnforceRate: 50,
						EnforcePBS:  getTrue(),
					},
				},
				fetchFloors: &openrtb_ext.PriceFloorRules{
					Enabled: getTrue(),
					Data: &openrtb_ext.PriceFloorData{
						Currency: "INR",
						SkipRate: 0,
						ModelGroups: []openrtb_ext.PriceFloorModelGroup{
							{
								ModelVersion: "Version 1",
								Currency:     "INR",
								Values: map[string]float64{
									"banner|300x600|www.website5.com": 20,
									"*|*|*":                           50,
								},
								Schema: openrtb_ext.PriceFloorSchema{
									Fields:    []string{"mediaType", "size", "domain"},
									Delimiter: "|",
								},
							},
						},
					},
				},
			},
			want: &openrtb_ext.PriceFloorRules{
				Enabled: getTrue(),
				Data: &openrtb_ext.PriceFloorData{
					Currency: "INR",
					SkipRate: 0,
					ModelGroups: []openrtb_ext.PriceFloorModelGroup{
						{
							ModelVersion: "Version 1",
							Currency:     "INR",
							Values: map[string]float64{
								"banner|300x600|www.website5.com": 20,
								"*|*|*":                           50,
							},
							Schema: openrtb_ext.PriceFloorSchema{
								Fields:    []string{"mediaType", "size", "domain"},
								Delimiter: "|",
							},
						},
					},
				},
				Enforcement: &openrtb_ext.PriceFloorEnforcement{
					EnforceRate: 50,
					EnforcePBS:  getTrue(),
				},
			},
		},
		{
			name: "Fetched Floors are present and request Floors has enforcement (enforcepbs = false)",
			args: args{
				reqFloors: &openrtb_ext.PriceFloorRules{
					Enabled: getTrue(),
					Enforcement: &openrtb_ext.PriceFloorEnforcement{
						EnforceRate: 50,
						EnforcePBS:  getFalse(),
					},
				},
				fetchFloors: &openrtb_ext.PriceFloorRules{
					Enabled: getTrue(),
					Data: &openrtb_ext.PriceFloorData{
						Currency: "INR",
						SkipRate: 0,
						ModelGroups: []openrtb_ext.PriceFloorModelGroup{
							{
								ModelVersion: "Version 1",
								Currency:     "INR",
								Values: map[string]float64{
									"banner|300x600|www.website5.com": 20,
									"*|*|*":                           50,
								},
								Schema: openrtb_ext.PriceFloorSchema{
									Fields:    []string{"mediaType", "size", "domain"},
									Delimiter: "|",
								},
							},
						},
					},
				},
			},
			want: &openrtb_ext.PriceFloorRules{
				Enabled: getTrue(),
				Data: &openrtb_ext.PriceFloorData{
					Currency: "INR",
					SkipRate: 0,
					ModelGroups: []openrtb_ext.PriceFloorModelGroup{
						{
							ModelVersion: "Version 1",
							Currency:     "INR",
							Values: map[string]float64{
								"banner|300x600|www.website5.com": 20,
								"*|*|*":                           50,
							},
							Schema: openrtb_ext.PriceFloorSchema{
								Fields:    []string{"mediaType", "size", "domain"},
								Delimiter: "|",
							},
						},
					},
				},
				Enforcement: &openrtb_ext.PriceFloorEnforcement{
					EnforceRate: 50,
					EnforcePBS:  getFalse(),
				},
			},
		},
		{
			name: "Fetched Floors are present and request Floors has Floormin",
			args: args{
				reqFloors: &openrtb_ext.PriceFloorRules{
					Enabled: getTrue(),
					Enforcement: &openrtb_ext.PriceFloorEnforcement{
						EnforceRate: 50,
						EnforcePBS:  getFalse(),
					},
					FloorMin:    5,
					FloorMinCur: "INR",
				},
				fetchFloors: &openrtb_ext.PriceFloorRules{
					Enabled: getTrue(),
					Data: &openrtb_ext.PriceFloorData{
						Currency: "INR",
						SkipRate: 0,
						ModelGroups: []openrtb_ext.PriceFloorModelGroup{
							{
								ModelVersion: "Version 1",
								Currency:     "INR",
								Values: map[string]float64{
									"banner|300x600|www.website5.com": 20,
									"*|*|*":                           50,
								},
								Schema: openrtb_ext.PriceFloorSchema{
									Fields:    []string{"mediaType", "size", "domain"},
									Delimiter: "|",
								},
							},
						},
					},
				},
			},
			want: &openrtb_ext.PriceFloorRules{
				Enabled: getTrue(),
				Data: &openrtb_ext.PriceFloorData{
					Currency: "INR",
					SkipRate: 0,
					ModelGroups: []openrtb_ext.PriceFloorModelGroup{
						{
							ModelVersion: "Version 1",
							Currency:     "INR",
							Values: map[string]float64{
								"banner|300x600|www.website5.com": 20,
								"*|*|*":                           50,
							},
							Schema: openrtb_ext.PriceFloorSchema{
								Fields:    []string{"mediaType", "size", "domain"},
								Delimiter: "|",
							},
						},
					},
				},
				Enforcement: &openrtb_ext.PriceFloorEnforcement{
					EnforceRate: 50,
					EnforcePBS:  getFalse(),
				},
				FloorMin:    5,
				FloorMinCur: "INR",
			},
		},
		{
			name: "Fetched Floors are present and request Floors has URL",
			args: args{
				reqFloors: &openrtb_ext.PriceFloorRules{
					Enabled: getTrue(),
					Enforcement: &openrtb_ext.PriceFloorEnforcement{
						EnforceRate: 50,
						EnforcePBS:  getFalse(),
					},
					FloorMin:    5,
					FloorMinCur: "INR",
					Location: &openrtb_ext.PriceFloorEndpoint{
						URL: "https://test.com/floors",
					},
				},
				fetchFloors: &openrtb_ext.PriceFloorRules{
					Enabled: getTrue(),
					Data: &openrtb_ext.PriceFloorData{
						Currency: "INR",
						SkipRate: 0,
						ModelGroups: []openrtb_ext.PriceFloorModelGroup{
							{
								ModelVersion: "Version 1",
								Currency:     "INR",
								Values: map[string]float64{
									"banner|300x600|www.website5.com": 20,
									"*|*|*":                           50,
								},
								Schema: openrtb_ext.PriceFloorSchema{
									Fields:    []string{"mediaType", "size", "domain"},
									Delimiter: "|",
								},
							},
						},
					},
				},
			},
			want: &openrtb_ext.PriceFloorRules{
				Enabled: getTrue(),
				Data: &openrtb_ext.PriceFloorData{
					Currency: "INR",
					SkipRate: 0,
					ModelGroups: []openrtb_ext.PriceFloorModelGroup{
						{
							ModelVersion: "Version 1",
							Currency:     "INR",
							Values: map[string]float64{
								"banner|300x600|www.website5.com": 20,
								"*|*|*":                           50,
							},
							Schema: openrtb_ext.PriceFloorSchema{
								Fields:    []string{"mediaType", "size", "domain"},
								Delimiter: "|",
							},
						},
					},
				},
				Enforcement: &openrtb_ext.PriceFloorEnforcement{
					EnforceRate: 50,
					EnforcePBS:  getFalse(),
				},
				FloorMin:    5,
				FloorMinCur: "INR",
				Location: &openrtb_ext.PriceFloorEndpoint{
					URL: "https://test.com/floors",
				},
			},
		},
		{
			name: "Fetched Floors has no enable atrribute are present and request Floors has URL",
			args: args{
				reqFloors: &openrtb_ext.PriceFloorRules{
					Enabled: getTrue(),
					Enforcement: &openrtb_ext.PriceFloorEnforcement{
						EnforceRate: 50,
						EnforcePBS:  getFalse(),
					},
					FloorMin:    5,
					FloorMinCur: "INR",
					Location: &openrtb_ext.PriceFloorEndpoint{
						URL: "https://test.com/floors",
					},
				},
				fetchFloors: &openrtb_ext.PriceFloorRules{
					Data: &openrtb_ext.PriceFloorData{
						Currency: "INR",
						SkipRate: 0,
						ModelGroups: []openrtb_ext.PriceFloorModelGroup{
							{
								ModelVersion: "Version 1",
								Currency:     "INR",
								Values: map[string]float64{
									"banner|300x600|www.website5.com": 20,
									"*|*|*":                           50,
								},
								Schema: openrtb_ext.PriceFloorSchema{
									Fields:    []string{"mediaType", "size", "domain"},
									Delimiter: "|",
								},
							},
						},
					},
				},
			},
			want: &openrtb_ext.PriceFloorRules{
				Enabled: getTrue(),
				Data: &openrtb_ext.PriceFloorData{
					Currency: "INR",
					SkipRate: 0,
					ModelGroups: []openrtb_ext.PriceFloorModelGroup{
						{
							ModelVersion: "Version 1",
							Currency:     "INR",
							Values: map[string]float64{
								"banner|300x600|www.website5.com": 20,
								"*|*|*":                           50,
							},
							Schema: openrtb_ext.PriceFloorSchema{
								Fields:    []string{"mediaType", "size", "domain"},
								Delimiter: "|",
							},
						},
					},
				},
				Enforcement: &openrtb_ext.PriceFloorEnforcement{
					EnforceRate: 50,
					EnforcePBS:  getFalse(),
				},
				FloorMin:    5,
				FloorMinCur: "INR",
				Location: &openrtb_ext.PriceFloorEndpoint{
					URL: "https://test.com/floors",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mergeFloors(tt.args.reqFloors, tt.args.fetchFloors, getCurrencyRates(rates)); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mergeFloors() = %v, want %v", got, tt.want)
			}
		})
	}
}
