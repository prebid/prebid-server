package openrtb_ext

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

type granularityTestData struct {
	json   []byte
	target PriceGranularity
}

func TestGranularityUnmarshal(t *testing.T) {
	testGroups := []struct {
		desc        string
		in          []granularityTestData
		expectError bool
	}{
		{
			desc:        "Unmarshal without error",
			in:          validGranularityTests,
			expectError: false,
		},
		{
			desc: "Malformed json. Expect unmarshall error",
			in: []granularityTestData{
				{json: []byte(`[]`)},
			},
			expectError: true,
		},
	}
	for _, tg := range testGroups {
		for i, tc := range tg.in {
			var resolved PriceGranularity
			err := jsonutil.UnmarshalValid(tc.json, &resolved)

			// Assert validation error
			if tg.expectError && !assert.Errorf(t, err, "%s test case %d", tg.desc, i) {
				continue
			}

			// Assert Targeting.Precision
			assert.Equal(t, tc.target.Precision, resolved.Precision, "%s test case %d", tg.desc, i)

			// Assert Targeting.Ranges
			if assert.Len(t, resolved.Ranges, len(tc.target.Ranges), "%s test case %d", tg.desc, i) {
				expected := make(map[string]struct{}, len(tc.target.Ranges))
				for _, r := range tc.target.Ranges {
					expected[fmt.Sprintf("%2.2f-%2.2f-%2.2f", r.Min, r.Max, r.Increment)] = struct{}{}
				}
				for _, actualRange := range resolved.Ranges {
					targetRange := fmt.Sprintf("%2.2f-%2.2f-%2.2f", actualRange.Min, actualRange.Max, actualRange.Increment)
					_, exists := expected[targetRange]
					assert.True(t, exists, "%s test case %d target.range %s not found", tg.desc, i, targetRange)
				}
			}
		}
	}
}

var validGranularityTests []granularityTestData = []granularityTestData{
	{
		json: []byte(`{"precision": 4, "ranges": [{"min": 0, "max": 5, "increment": 0.1}, {"min": 5, "max":10, "increment":0.5}, {"min":10, "max":20, "increment":1}]}`),
		target: PriceGranularity{
			Precision: ptrutil.ToPtr(4),
			Ranges: []GranularityRange{
				{Min: 0.0, Max: 5.0, Increment: 0.1},
				{Min: 5.0, Max: 10.0, Increment: 0.5},
				{Min: 10.0, Max: 20.0, Increment: 1.0},
			},
		},
	},
	{
		json: []byte(`{"ranges":[{ "max":5, "increment": 0.05}, {"max": 10, "increment": 0.25}, {"max": 20, "increment": 0.5}]}`),
		target: PriceGranularity{
			Ranges: []GranularityRange{
				{Min: 0.0, Max: 5.0, Increment: 0.05},
				{Min: 0.0, Max: 10.0, Increment: 0.25},
				{Min: 0.0, Max: 20.0, Increment: 0.5},
			},
		},
	},
	{
		json: []byte(`"medium"`),
		target: PriceGranularity{
			Precision: ptrutil.ToPtr(2),
			Ranges: []GranularityRange{{
				Min:       0,
				Max:       20,
				Increment: 0.1}},
		},
	},
	{
		json: []byte(`{ "precision": 3, "ranges": [{"max":20, "increment":0.005}]}`),
		target: PriceGranularity{
			Precision: ptrutil.ToPtr(3),
			Ranges:    []GranularityRange{{Min: 0.0, Max: 20.0, Increment: 0.005}},
		},
	},
	{
		json: []byte(`{"precision": 0, "ranges": [{"max":5, "increment": 1}, {"max": 10, "increment": 2}, {"max": 20, "increment": 5}]}`),
		target: PriceGranularity{
			Precision: ptrutil.ToPtr(0),
			Ranges: []GranularityRange{
				{Min: 0.0, Max: 5.0, Increment: 1.0},
				{Min: 0.0, Max: 10.0, Increment: 2.0},
				{Min: 0.0, Max: 20.0, Increment: 5.0},
			},
		},
	},
	{
		json: []byte(`{"precision": 2, "ranges": [{"min": 0.5, "max":5, "increment": 0.1}, {"min": 54, "max": 10, "increment": 1}, {"min": -42, "max": 20, "increment": 5}]}`),
		target: PriceGranularity{
			Precision: ptrutil.ToPtr(2),
			Ranges: []GranularityRange{
				{Min: 0.5, Max: 5.0, Increment: 0.1},
				{Min: 54.0, Max: 10.0, Increment: 1.0},
				{Min: -42.0, Max: 20.0, Increment: 5.0},
			},
		},
	},
	{
		json:   []byte(`{}`),
		target: PriceGranularity{},
	},
	{
		json: []byte(`{"precision": 2}`),
		target: PriceGranularity{
			Precision: ptrutil.ToPtr(2),
		},
	},
	{
		json: []byte(`{"precision": 2, "ranges":[]}`),
		target: PriceGranularity{
			Precision: ptrutil.ToPtr(2),
			Ranges:    []GranularityRange{},
		},
	},
}

func TestCloneExtRequestPrebid(t *testing.T) {
	testCases := []struct {
		name       string
		prebid     *ExtRequestPrebid
		prebidCopy *ExtRequestPrebid                            // manual copy of above prebid object to verify against
		mutator    func(t *testing.T, prebid *ExtRequestPrebid) // function to modify the prebid object
	}{
		{
			name:       "Nil", // Verify the nil case
			prebid:     nil,
			prebidCopy: nil,
			mutator:    func(t *testing.T, prebid *ExtRequestPrebid) {},
		},
		{
			name: "NoMutateTest",
			prebid: &ExtRequestPrebid{
				Aliases:              map[string]string{"alias1": "bidder1"},
				BidAdjustmentFactors: map[string]float64{"bidder5": 1.2},
				BidderParams:         json.RawMessage(`{}`),
				Channel: &ExtRequestPrebidChannel{
					Name:    "ABC",
					Version: "1.0",
				},
				Debug: true,
				Experiment: &Experiment{
					AdsCert: &AdsCert{
						Enabled: false,
					},
				},
				Server: &ExtRequestPrebidServer{
					ExternalUrl: "http://www.example.com",
					GvlID:       5,
					DataCenter:  "Universe Central",
				},
				SupportDeals: true,
			},
			prebidCopy: &ExtRequestPrebid{
				Aliases:              map[string]string{"alias1": "bidder1"},
				BidAdjustmentFactors: map[string]float64{"bidder5": 1.2},
				BidderParams:         json.RawMessage(`{}`),
				Channel: &ExtRequestPrebidChannel{
					Name:    "ABC",
					Version: "1.0",
				},
				Debug: true,
				Experiment: &Experiment{
					AdsCert: &AdsCert{
						Enabled: false,
					},
				},
				Server: &ExtRequestPrebidServer{
					ExternalUrl: "http://www.example.com",
					GvlID:       5,
					DataCenter:  "Universe Central",
				},
				SupportDeals: true,
			},
			mutator: func(t *testing.T, prebid *ExtRequestPrebid) {},
		},
		{
			name: "GeneralTest",
			prebid: &ExtRequestPrebid{
				Aliases:              map[string]string{"alias1": "bidder1"},
				BidAdjustmentFactors: map[string]float64{"bidder5": 1.2},
				BidderParams:         json.RawMessage(`{}`),
				Channel: &ExtRequestPrebidChannel{
					Name:    "ABC",
					Version: "1.0",
				},
				Debug: true,
				Experiment: &Experiment{
					AdsCert: &AdsCert{
						Enabled: false,
					},
				},
				Server: &ExtRequestPrebidServer{
					ExternalUrl: "http://www.example.com",
					GvlID:       5,
					DataCenter:  "Universe Central",
				},
				SupportDeals: true,
			},
			prebidCopy: &ExtRequestPrebid{
				Aliases:              map[string]string{"alias1": "bidder1"},
				BidAdjustmentFactors: map[string]float64{"bidder5": 1.2},
				BidderParams:         json.RawMessage(`{}`),
				Channel: &ExtRequestPrebidChannel{
					Name:    "ABC",
					Version: "1.0",
				},
				Debug: true,
				Experiment: &Experiment{
					AdsCert: &AdsCert{
						Enabled: false,
					},
				},
				Server: &ExtRequestPrebidServer{
					ExternalUrl: "http://www.example.com",
					GvlID:       5,
					DataCenter:  "Universe Central",
				},
				SupportDeals: true,
			},
			mutator: func(t *testing.T, prebid *ExtRequestPrebid) {
				prebid.Aliases["alias2"] = "bidder52"
				prebid.Aliases["alias1"] = "some other"
				prebid.AliasGVLIDs = map[string]uint16{"alias2": 42}
				prebid.BidAdjustmentFactors["alias2"] = 1.1
				delete(prebid.BidAdjustmentFactors, "bidder5")
				prebid.BidderParams = json.RawMessage(`{"someJSON": true}`)
				prebid.Channel = nil
				prebid.Data = &ExtRequestPrebidData{
					EidPermissions: []ExtRequestPrebidDataEidPermission{{Source: "mySource", Bidders: []string{"sauceBidder"}}},
				}
				prebid.Events = json.RawMessage(`{}`)
				prebid.Server.GvlID = 7
				prebid.SupportDeals = false
			},
		},
		{
			name: "BidderConfig",
			prebid: &ExtRequestPrebid{
				BidderConfigs: []BidderConfig{
					{
						Bidders: []string{"Bidder1", "bidder2"},
						Config:  &Config{&ORTB2{Site: json.RawMessage(`{"value":"config1"}`)}},
					},
					{
						Bidders: []string{"Bidder5", "bidder17"},
						Config:  &Config{&ORTB2{App: json.RawMessage(`{"value":"config2"}`)}},
					},
					{
						Bidders: []string{"foo"},
						Config:  &Config{&ORTB2{User: json.RawMessage(`{"value":"config3"}`)}},
					},
					{
						Bidders: []string{"Bidder9"},
						Config:  &Config{&ORTB2{Device: json.RawMessage(`{"value":"config4"}`)}},
					},
				},
			},
			prebidCopy: &ExtRequestPrebid{
				BidderConfigs: []BidderConfig{
					{
						Bidders: []string{"Bidder1", "bidder2"},
						Config:  &Config{&ORTB2{Site: json.RawMessage(`{"value":"config1"}`)}},
					},
					{
						Bidders: []string{"Bidder5", "bidder17"},
						Config:  &Config{&ORTB2{App: json.RawMessage(`{"value":"config2"}`)}},
					},
					{
						Bidders: []string{"foo"},
						Config:  &Config{&ORTB2{User: json.RawMessage(`{"value":"config3"}`)}},
					},
					{
						Bidders: []string{"Bidder9"},
						Config:  &Config{&ORTB2{Device: json.RawMessage(`{"value":"config4"}`)}},
					},
				},
			},
			mutator: func(t *testing.T, prebid *ExtRequestPrebid) {
				prebid.BidderConfigs[0].Bidders = append(prebid.BidderConfigs[0].Bidders, "bidder4")
				prebid.BidderConfigs[1] = BidderConfig{
					Bidders: []string{"george"},
					Config:  &Config{nil},
				}
				prebid.BidderConfigs[2].Config.ORTB2.User = json.RawMessage(`{"id": 345}`)
				prebid.BidderConfigs[3].Config.ORTB2.Device = json.RawMessage(`{"id": 999}`)
				prebid.BidderConfigs = append(prebid.BidderConfigs, BidderConfig{
					Bidders: []string{"bidder2"},
					Config:  &Config{&ORTB2{}},
				})
			},
		},
		{
			name: "Cache",
			prebid: &ExtRequestPrebid{
				Cache: &ExtRequestPrebidCache{
					Bids: &ExtRequestPrebidCacheBids{
						ReturnCreative: ptrutil.ToPtr(true),
					},
					VastXML: &ExtRequestPrebidCacheVAST{
						ReturnCreative: ptrutil.ToPtr(false),
					},
				},
			},
			prebidCopy: &ExtRequestPrebid{
				Cache: &ExtRequestPrebidCache{
					Bids: &ExtRequestPrebidCacheBids{
						ReturnCreative: ptrutil.ToPtr(true),
					},
					VastXML: &ExtRequestPrebidCacheVAST{
						ReturnCreative: ptrutil.ToPtr(false),
					},
				},
			},
			mutator: func(t *testing.T, prebid *ExtRequestPrebid) {
				prebid.Cache.Bids.ReturnCreative = ptrutil.ToPtr(false)
				prebid.Cache.Bids = nil
				prebid.Cache.VastXML = &ExtRequestPrebidCacheVAST{
					ReturnCreative: ptrutil.ToPtr(true),
				}
			},
		},
		{
			name: "Currency",
			prebid: &ExtRequestPrebid{
				CurrencyConversions: &ExtRequestCurrency{
					ConversionRates: map[string]map[string]float64{"A": {"X": 5.4}},
					UsePBSRates:     ptrutil.ToPtr(false),
				},
			},
			prebidCopy: &ExtRequestPrebid{
				CurrencyConversions: &ExtRequestCurrency{
					ConversionRates: map[string]map[string]float64{"A": {"X": 5.4}},
					UsePBSRates:     ptrutil.ToPtr(false),
				},
			},
			mutator: func(t *testing.T, prebid *ExtRequestPrebid) {
				prebid.CurrencyConversions.ConversionRates["A"]["X"] = 3.4
				prebid.CurrencyConversions.ConversionRates["B"] = make(map[string]float64)
				prebid.CurrencyConversions.ConversionRates["B"]["Y"] = 0.76
				prebid.CurrencyConversions.UsePBSRates = ptrutil.ToPtr(true)
			},
		},
		{
			name: "Data",
			prebid: &ExtRequestPrebid{
				Data: &ExtRequestPrebidData{
					EidPermissions: []ExtRequestPrebidDataEidPermission{
						{
							Source:  "Sauce",
							Bidders: []string{"G", "H"},
						},
						{
							Source:  "Black Hole",
							Bidders: []string{"Q", "P"},
						},
					},
					Bidders: []string{"A", "B", "C"},
				},
			},
			prebidCopy: &ExtRequestPrebid{
				Data: &ExtRequestPrebidData{
					EidPermissions: []ExtRequestPrebidDataEidPermission{
						{
							Source:  "Sauce",
							Bidders: []string{"G", "H"},
						},
						{
							Source:  "Black Hole",
							Bidders: []string{"Q", "P"},
						},
					},
					Bidders: []string{"A", "B", "C"},
				},
			},
			mutator: func(t *testing.T, prebid *ExtRequestPrebid) {
				prebid.Data.EidPermissions[0].Source = "Fuzzy Bunnies"
				prebid.Data.EidPermissions[1].Bidders[0] = "X"
				prebid.Data.EidPermissions[0].Bidders = append(prebid.Data.EidPermissions[0].Bidders, "R")
				prebid.Data.EidPermissions = append(prebid.Data.EidPermissions, ExtRequestPrebidDataEidPermission{Source: "Harry"})
				prebid.Data.Bidders[1] = "D"
				prebid.Data.Bidders = append(prebid.Data.Bidders, "E")
			},
		},
		{
			name: "Multibid",
			prebid: &ExtRequestPrebid{
				MultiBid: []*ExtMultiBid{
					{Bidder: "somebidder", MaxBids: ptrutil.ToPtr(3), TargetBidderCodePrefix: "SB"},
					{Bidders: []string{"A", "B", "C"}, MaxBids: ptrutil.ToPtr(4)},
				},
			},
			prebidCopy: &ExtRequestPrebid{
				MultiBid: []*ExtMultiBid{
					{Bidder: "somebidder", MaxBids: ptrutil.ToPtr(3), TargetBidderCodePrefix: "SB"},
					{Bidders: []string{"A", "B", "C"}, MaxBids: ptrutil.ToPtr(4)},
				},
			},
			mutator: func(t *testing.T, prebid *ExtRequestPrebid) {
				prebid.MultiBid[0].MaxBids = ptrutil.ToPtr(2)
				prebid.MultiBid[1].Bidders = []string{"C", "D", "E", "F"}
				prebid.MultiBid = []*ExtMultiBid{
					{Bidder: "otherbid"},
				}
			},
		},
		{
			name: "PassthroughSChains",
			prebid: &ExtRequestPrebid{
				SChains: []*ExtRequestPrebidSChain{
					{
						Bidders: []string{"A", "B", "C"},
						SChain: openrtb2.SupplyChain{
							Complete: 1,
							Ver:      "2.2",
							Ext:      json.RawMessage(`{"foo": "bar"}`),
							Nodes: []openrtb2.SupplyChainNode{
								{
									ASI:    "something",
									Domain: "example.com",
									HP:     openrtb2.Int8Ptr(1),
								},
							},
						},
					},
				},
			},
			prebidCopy: &ExtRequestPrebid{
				SChains: []*ExtRequestPrebidSChain{
					{
						Bidders: []string{"A", "B", "C"},
						SChain: openrtb2.SupplyChain{
							Complete: 1,
							Ver:      "2.2",
							Ext:      json.RawMessage(`{"foo": "bar"}`),
							Nodes: []openrtb2.SupplyChainNode{
								{
									ASI:    "something",
									Domain: "example.com",
									HP:     openrtb2.Int8Ptr(1),
								},
							},
						},
					},
				},
			},
			mutator: func(t *testing.T, prebid *ExtRequestPrebid) {
				prebid.Passthrough = json.RawMessage(`{"bar": "foo"}`)
				prebid.SChains[0].Bidders = append(prebid.SChains[0].Bidders, "D")
				prebid.SChains[0].SChain.Ver = "2.3"
				prebid.SChains[0].SChain.Complete = 0
				prebid.SChains[0].SChain.Nodes[0].Name = "Alice"
				prebid.SChains[0].SChain.Nodes[0].ASI = "New ASI"
				prebid.SChains[0].SChain.Nodes[0].HP = openrtb2.Int8Ptr(0)
				prebid.SChains[0].SChain.Nodes = append(prebid.SChains[0].SChain.Nodes, prebid.SChains[0].SChain.Nodes[0])
				prebid.SChains = append(prebid.SChains, prebid.SChains[0])
			},
		},
		{
			name: "StoredRequest",
			prebid: &ExtRequestPrebid{
				StoredRequest: &ExtStoredRequest{
					ID: "abc123",
				},
			},
			prebidCopy: &ExtRequestPrebid{
				StoredRequest: &ExtStoredRequest{
					ID: "abc123",
				},
			},
			mutator: func(t *testing.T, prebid *ExtRequestPrebid) {
				prebid.StoredRequest.ID = "nada"
				prebid.StoredRequest = &ExtStoredRequest{ID: "ID"}
			},
		},
		{
			name: "Targeting",
			prebid: &ExtRequestPrebid{
				Targeting: &ExtRequestTargeting{
					PriceGranularity: &PriceGranularity{
						Precision: ptrutil.ToPtr(2),
						Ranges: []GranularityRange{
							{Max: 2.0, Increment: 0.1},
							{Max: 10.0, Increment: 0.5},
							{Max: 20.0, Increment: 1.0},
						},
					},
					IncludeWinners: ptrutil.ToPtr(true),
					IncludeBrandCategory: &ExtIncludeBrandCategory{
						PrimaryAdServer:     1,
						Publisher:           "Bob",
						TranslateCategories: ptrutil.ToPtr(true),
					},
					DurationRangeSec: []int{1, 2, 3},
				},
			},
			prebidCopy: &ExtRequestPrebid{
				Targeting: &ExtRequestTargeting{
					PriceGranularity: &PriceGranularity{
						Precision: ptrutil.ToPtr(2),
						Ranges: []GranularityRange{
							{Max: 2.0, Increment: 0.1},
							{Max: 10.0, Increment: 0.5},
							{Max: 20.0, Increment: 1.0},
						},
					},
					IncludeWinners: ptrutil.ToPtr(true),
					IncludeBrandCategory: &ExtIncludeBrandCategory{
						PrimaryAdServer:     1,
						Publisher:           "Bob",
						TranslateCategories: ptrutil.ToPtr(true),
					},
					DurationRangeSec: []int{1, 2, 3},
				},
			},
			mutator: func(t *testing.T, prebid *ExtRequestPrebid) {
				prebid.Targeting.PriceGranularity.Ranges[1].Max = 12
				prebid.Targeting.PriceGranularity.Ranges[1].Min = 2.0
				prebid.Targeting.PriceGranularity.Ranges = append(prebid.Targeting.PriceGranularity.Ranges, GranularityRange{Max: 50, Increment: 2.0})
				prebid.Targeting.IncludeWinners = nil
				prebid.Targeting.IncludeBidderKeys = ptrutil.ToPtr(true)
				prebid.Targeting.IncludeBrandCategory.TranslateCategories = ptrutil.ToPtr(false)
				prebid.Targeting.IncludeBrandCategory = nil
				prebid.Targeting.DurationRangeSec[1] = 5
				prebid.Targeting.DurationRangeSec = append(prebid.Targeting.DurationRangeSec, 1)
				prebid.Targeting.AppendBidderNames = true
			},
		},
		{
			name: "NoSale",
			prebid: &ExtRequestPrebid{
				NoSale: []string{"A", "B", "C"},
			},
			prebidCopy: &ExtRequestPrebid{
				NoSale: []string{"A", "B", "C"},
			},
			mutator: func(t *testing.T, prebid *ExtRequestPrebid) {
				prebid.NoSale[1] = "G"
				prebid.NoSale = append(prebid.NoSale, "D")
			},
		},
		{
			name: "AlternateBidderCodes",
			prebid: &ExtRequestPrebid{
				AlternateBidderCodes: &ExtAlternateBidderCodes{
					Enabled: true,
					Bidders: map[string]ExtAdapterAlternateBidderCodes{
						"X": {Enabled: true, AllowedBidderCodes: []string{"A", "B", "C"}},
						"Y": {Enabled: false, AllowedBidderCodes: []string{"C", "B", "G"}},
					},
				},
			},
			prebidCopy: &ExtRequestPrebid{
				AlternateBidderCodes: &ExtAlternateBidderCodes{
					Enabled: true,
					Bidders: map[string]ExtAdapterAlternateBidderCodes{
						"X": {Enabled: true, AllowedBidderCodes: []string{"A", "B", "C"}},
						"Y": {Enabled: false, AllowedBidderCodes: []string{"C", "B", "G"}},
					},
				},
			},
			mutator: func(t *testing.T, prebid *ExtRequestPrebid) {
				newAABC := prebid.AlternateBidderCodes.Bidders["X"]
				newAABC.Enabled = false
				newAABC.AllowedBidderCodes[1] = "F"
				newAABC.AllowedBidderCodes = append(newAABC.AllowedBidderCodes, "Z")
				prebid.AlternateBidderCodes.Bidders["X"] = newAABC
				prebid.AlternateBidderCodes.Bidders["Z"] = ExtAdapterAlternateBidderCodes{Enabled: true, AllowedBidderCodes: []string{"G", "Z"}}
				prebid.AlternateBidderCodes.Enabled = false
				prebid.AlternateBidderCodes = nil
			},
		},
		{
			name: "Floors",
			prebid: &ExtRequestPrebid{
				Floors: &PriceFloorRules{
					FloorMin:    0.25,
					FloorMinCur: "EUR",
					Location: &PriceFloorEndpoint{
						URL: "http://www.example.com",
					},
					Data: &PriceFloorData{
						Currency: "USD",
						SkipRate: 3,
						ModelGroups: []PriceFloorModelGroup{
							{
								Currency:    "USD",
								ModelWeight: ptrutil.ToPtr(0),
								SkipRate:    2,
								Schema: PriceFloorSchema{
									Fields:    []string{"A", "B"},
									Delimiter: "^",
								},
								Values: map[string]float64{"A": 2, "B": 1},
							},
						},
					},
					Enforcement: &PriceFloorEnforcement{
						EnforceJS:   ptrutil.ToPtr(true),
						FloorDeals:  ptrutil.ToPtr(false),
						EnforceRate: 5,
					},
					Enabled:       ptrutil.ToPtr(true),
					FloorProvider: "Someone",
				},
			},
			prebidCopy: &ExtRequestPrebid{
				Floors: &PriceFloorRules{
					FloorMin:    0.25,
					FloorMinCur: "EUR",
					Location: &PriceFloorEndpoint{
						URL: "http://www.example.com",
					},
					Data: &PriceFloorData{
						Currency: "USD",
						SkipRate: 3,
						ModelGroups: []PriceFloorModelGroup{
							{
								Currency:    "USD",
								ModelWeight: ptrutil.ToPtr(0),
								SkipRate:    2,
								Schema: PriceFloorSchema{
									Fields:    []string{"A", "B"},
									Delimiter: "^",
								},
								Values: map[string]float64{"A": 2, "B": 1},
							},
						},
					},
					Enforcement: &PriceFloorEnforcement{
						EnforceJS:   ptrutil.ToPtr(true),
						FloorDeals:  ptrutil.ToPtr(false),
						EnforceRate: 5,
					},
					Enabled:       ptrutil.ToPtr(true),
					FloorProvider: "Someone",
				},
			},
			mutator: func(t *testing.T, prebid *ExtRequestPrebid) {
				prebid.Floors.Data.ModelGroups[0].Schema.Fields[1] = "C"
				prebid.Floors.Data.ModelGroups[0].Schema.Fields = append(prebid.Floors.Data.ModelGroups[0].Schema.Fields, "D")
				prebid.Floors.Data.ModelGroups[0].Schema.Delimiter = ","
				prebid.Floors.Data.ModelGroups[0].Currency = "CRO"
				prebid.Floors.Data.ModelGroups[0].ModelWeight = ptrutil.ToPtr(8)
				prebid.Floors.Data.ModelGroups[0].Values["A"] = 0
				prebid.Floors.Data.ModelGroups[0].Values["C"] = 7
				prebid.Floors.Data.ModelGroups = append(prebid.Floors.Data.ModelGroups, PriceFloorModelGroup{})
				prebid.Floors.FloorMin = 0.3
				prebid.Floors.FetchStatus = "arf"
				prebid.Floors.Location.URL = "www.nothere.com"
				prebid.Floors.Location = nil
				prebid.Floors.Enabled = nil
				prebid.Floors.Skipped = ptrutil.ToPtr(true)
				prebid.Floors.Enforcement.BidAdjustment = ptrutil.ToPtr(true)
				prebid.Floors.Enforcement.EnforceJS = ptrutil.ToPtr(false)
				prebid.Floors.Enforcement.FloorDeals = nil
				prebid.Floors.FloorProvider = ""
			},
		},
		{
			name: "MultiBidMap",
			prebid: &ExtRequestPrebid{
				MultiBidMap: map[string]ExtMultiBid{
					"A": {
						Bidder:                 "J",
						Bidders:                []string{"X", "Y", "Z"},
						MaxBids:                ptrutil.ToPtr(5),
						TargetBidderCodePrefix: ">>",
					},
					"B": {
						Bidder:  "J",
						Bidders: []string{"One", "Two", "Three"},
					},
				},
			},
			prebidCopy: &ExtRequestPrebid{
				MultiBidMap: map[string]ExtMultiBid{
					"A": {
						Bidder:                 "J",
						Bidders:                []string{"X", "Y", "Z"},
						MaxBids:                ptrutil.ToPtr(5),
						TargetBidderCodePrefix: ">>",
					},
					"B": {
						Bidder:  "J",
						Bidders: []string{"One", "Two", "Three"},
					},
				},
			},
			mutator: func(t *testing.T, prebid *ExtRequestPrebid) {
				mulbidB := prebid.MultiBidMap["B"]
				mulbidB.TargetBidderCodePrefix = "|"
				mulbidB.Bidders[1] = "Five"
				mulbidB.Bidders = append(mulbidB.Bidders, "Six")
				mulbidB.MaxBids = ptrutil.ToPtr(2)
				prebid.MultiBidMap["B"] = mulbidB
				prebid.MultiBidMap["C"] = ExtMultiBid{Bidder: "alpha", MaxBids: ptrutil.ToPtr(3)}
			},
		},
		{
			name: "MultiBidMap",
			prebid: &ExtRequestPrebid{
				AdServerTargeting: []AdServerTarget{
					{
						Key:    "A",
						Source: "Sauce",
						Value:  "Gold",
					},
					{
						Key:    "B",
						Source: "Omega",
						Value:  "Dirt",
					},
				},
			},
			prebidCopy: &ExtRequestPrebid{
				AdServerTargeting: []AdServerTarget{
					{
						Key:    "A",
						Source: "Sauce",
						Value:  "Gold",
					},
					{
						Key:    "B",
						Source: "Omega",
						Value:  "Dirt",
					},
				},
			},
			mutator: func(t *testing.T, prebid *ExtRequestPrebid) {
				prebid.AdServerTargeting[0].Key = "Five"
				prebid.AdServerTargeting[1].Value = "Dust"
				prebid.AdServerTargeting = append(prebid.AdServerTargeting, AdServerTarget{Key: "Val"})
				prebid.AdServerTargeting = nil
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			clone := test.prebid.Clone()
			if test.prebid != nil {
				assert.NotSame(t, test.prebid, clone)
			}
			test.mutator(t, test.prebid)
			assert.Equal(t, test.prebidCopy, clone)
		})
	}

}
