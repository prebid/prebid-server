package floors

import (
	"reflect"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestValidateFloorParams(t *testing.T) {

	tt := []struct {
		name     string
		floorExt *openrtb_ext.PriceFloorRules
		Err      string
	}{
		{
			name: "Valid Skip Rate",
			floorExt: &openrtb_ext.PriceFloorRules{Data: &openrtb_ext.PriceFloorData{
				ModelGroups: []openrtb_ext.PriceFloorModelGroup{{
					ModelVersion: "Version 1",
					Schema:       openrtb_ext.PriceFloorSchema{Fields: []string{"mediaType", "size", "domain"}, Delimiter: "|"},
					Values: map[string]float64{
						"banner|300x250|www.website.com":              1.01,
						"banner|300x250|*":                            2.01,
						"banner|300x600|www.website.com|www.test.com": 3.01,
						"banner|300x600|*":                            4.01,
					}, Default: 0.01},
				}}},
			Err: "",
		},
		{
			name:     "Invalid Skip Rate at Root level",
			floorExt: &openrtb_ext.PriceFloorRules{SkipRate: -10},
			Err:      "Invalid SkipRate = '-10' at ext.floors.skiprate",
		},
		{
			name: "Invalid Skip Rate at Date level",
			floorExt: &openrtb_ext.PriceFloorRules{Data: &openrtb_ext.PriceFloorData{
				SkipRate: -10,
				ModelGroups: []openrtb_ext.PriceFloorModelGroup{{
					ModelVersion: "Version 1",
					Schema:       openrtb_ext.PriceFloorSchema{Fields: []string{"mediaType", "size", "domain"}},
					Values: map[string]float64{
						"*|*|www.website.com": 15.01,
						"*|*|*":               16.01,
					}, Default: 0.01},
				}}},
			Err: "Invalid SkipRate = '-10' at  at ext.floors.data.skiprate",
		},
		{
			name:     "Invalid FloorMin ",
			floorExt: &openrtb_ext.PriceFloorRules{FloorMin: -10},
			Err:      "Invalid FloorMin = '-10', value should be >= 0",
		},
		{
			name: "Invalid FloorSchemaVersion ",
			floorExt: &openrtb_ext.PriceFloorRules{Data: &openrtb_ext.PriceFloorData{
				FloorsSchemaVersion: "1",
				ModelGroups: []openrtb_ext.PriceFloorModelGroup{{
					ModelVersion: "Version 1",

					Schema: openrtb_ext.PriceFloorSchema{Fields: []string{"mediaType", "size", "domain"}, Delimiter: "|"},
					Values: map[string]float64{
						"banner|300x250|www.website.com": 1.01,
						"banner|300x600|*":               4.01,
					}, Default: 0.01},
				}}},
			Err: "Invalid FloorsSchemaVersion = '1', supported version 2",
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			if actErr := validateFloorParams(tc.floorExt); actErr != nil {
				if !reflect.DeepEqual(actErr.Error(), tc.Err) {
					t.Errorf("Incorrect Error: \nreturn:\t%v\nwant:\t%v", actErr.Error(), tc.Err)
				}
			}
		})
	}
}

func TestSelectValidFloorModelGroups(t *testing.T) {

	tt := []struct {
		name     string
		floorExt *openrtb_ext.PriceFloorRules
		Err      string
	}{
		{
			name: "Invalid Skip Rate in model Group 1",
			floorExt: &openrtb_ext.PriceFloorRules{Data: &openrtb_ext.PriceFloorData{
				ModelGroups: []openrtb_ext.PriceFloorModelGroup{{
					ModelWeight:  getIntPtr(50),
					SkipRate:     110,
					ModelVersion: "Version 1",
					Schema:       openrtb_ext.PriceFloorSchema{Fields: []string{"mediaType", "size", "domain"}},
					Values: map[string]float64{
						"banner|300x250|www.website.com": 1.01,
						"banner|300x250|*":               2.01,
						"banner|300x600|www.website.com": 3.01,
						"*|728x90|*":                     14.01,
						"*|*|www.website.com":            15.01,
						"*|*|*":                          16.01,
					}, Default: 0.01},
					{
						ModelWeight:  getIntPtr(50),
						SkipRate:     20,
						ModelVersion: "Version 2",
						Schema:       openrtb_ext.PriceFloorSchema{Fields: []string{"mediaType", "size", "domain"}},
						Values: map[string]float64{
							"banner|300x250|www.website.com": 1.01,
							"banner|300x250|*":               2.01,
							"*|*|www.website.com":            15.01,
							"*|*|*":                          16.01,
						}, Default: 0.01},
				}}},
			Err: "Invalid Floor Model = 'Version 1' due to SkipRate = '110' is out of range (1-100)",
		},
		{
			name: "Invalid model weight Model Group 1",
			floorExt: &openrtb_ext.PriceFloorRules{Data: &openrtb_ext.PriceFloorData{
				ModelGroups: []openrtb_ext.PriceFloorModelGroup{{
					ModelWeight:  getIntPtr(-1),
					SkipRate:     10,
					ModelVersion: "Version 1",
					Schema:       openrtb_ext.PriceFloorSchema{Fields: []string{"mediaType", "size", "domain"}},
					Values: map[string]float64{
						"banner|300x250|www.website.com": 1.01,
						"banner|300x250|*":               2.01,
						"*|728x90|*":                     14.01,
						"*|*|www.website.com":            15.01,
						"*|*|*":                          16.01,
					}, Default: 0.01},
					{
						ModelWeight:  getIntPtr(50),
						SkipRate:     20,
						ModelVersion: "Version 2",
						Schema:       openrtb_ext.PriceFloorSchema{Fields: []string{"mediaType", "size", "domain"}},
						Values: map[string]float64{
							"banner|300x250|www.website.com": 1.01,
							"banner|300x250|*":               2.01,
							"*|728x90|*":                     14.01,
							"*|*|www.website.com":            15.01,
							"*|*|*":                          16.01,
						}, Default: 0.01},
				}}},
			Err: "Invalid Floor Model = 'Version 1' due to ModelWeight = '-1' is out of range (1-100)",
		},
		{
			name: "Invalid Default Value",
			floorExt: &openrtb_ext.PriceFloorRules{Data: &openrtb_ext.PriceFloorData{
				ModelGroups: []openrtb_ext.PriceFloorModelGroup{{
					ModelWeight:  getIntPtr(50),
					ModelVersion: "Version 1",
					Schema:       openrtb_ext.PriceFloorSchema{Fields: []string{"mediaType", "size", "domain"}},
					Values: map[string]float64{
						"banner|300x250|www.website.com": 1.01,
						"*|728x90|*":                     14.01,
						"*|*|www.website.com":            15.01,
						"*|*|*":                          16.01,
					}, Default: -1.0000},
				}}},
			Err: "Invalid Floor Model = 'Version 1' due to Default = '-1' is less than 0",
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			_, ErrList := selectValidFloorModelGroups(tc.floorExt.Data.ModelGroups)

			if !reflect.DeepEqual(ErrList[0].Error(), tc.Err) {
				t.Errorf("Incorrect Error: \nreturn:\t%v\nwant:\t%v", ErrList[0].Error(), tc.Err)
			}

		})
	}
}

func TestValidateFloorRulesAndLowerValidRuleKey(t *testing.T) {

	tt := []struct {
		name         string
		floorExt     *openrtb_ext.PriceFloorRules
		Err          string
		expctedFloor map[string]float64
	}{
		{
			name: "Invalid floor rule banner|300x600|www.website.com|www.test.com",
			floorExt: &openrtb_ext.PriceFloorRules{Data: &openrtb_ext.PriceFloorData{
				ModelGroups: []openrtb_ext.PriceFloorModelGroup{{
					ModelVersion: "Version 1",
					Schema:       openrtb_ext.PriceFloorSchema{Fields: []string{"mediaType", "size", "domain"}, Delimiter: "|"},
					Values: map[string]float64{
						"BANNER|300x250|WWW.WEBSITE.COM":              1.01,
						"banner|300x250|*":                            2.01,
						"banner|300x600|www.website.com|www.test.com": 3.01,
						"banner|300x600|*":                            4.01,
						"banner|728x90|www.website.com":               5.01,
						"banner|728x90|*":                             6.01,
						"banner|*|www.website.com":                    7.01,
						"banner|*|*":                                  8.01,
						"*|300x250|www.website.com":                   9.01,
						"*|300x250|*":                                 10.01,
						"*|300x600|www.website.com":                   11.01,
						"*|300x600|*":                                 12.01,
						"*|728x90|www.website.com":                    13.01,
						"*|728x90|*":                                  14.01,
						"*|*|www.website.com":                         15.01,
						"*|*|*":                                       16.01,
					}, Default: 0.01},
				}}},
			Err: "Invalid Floor Rule = 'banner|300x600|www.website.com|www.test.com' for Schema Fields = '[mediaType size domain]'",
			expctedFloor: map[string]float64{
				"banner|300x250|www.website.com": 1.01,
				"banner|300x250|*":               2.01,
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
			},
		},
		{
			name: "Invalid floor rule banner|300x600|www.website.com|www.test.com",
			floorExt: &openrtb_ext.PriceFloorRules{Data: &openrtb_ext.PriceFloorData{
				ModelGroups: []openrtb_ext.PriceFloorModelGroup{{
					ModelVersion: "Version 1",
					Schema:       openrtb_ext.PriceFloorSchema{Fields: []string{"mediaType", "size", "domain"}, Delimiter: "|"},
					Values: map[string]float64{
						"banner|300x250|www.website.com": 1.01,
						"banner|300x250|*":               2.01,
						"banner|300x600|www.website.com": 3.01,
						"banner|300x600":                 4.01,
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
				}}},
			Err: "Invalid Floor Rule = 'banner|300x600' for Schema Fields = '[mediaType size domain]'",
			expctedFloor: map[string]float64{
				"banner|300x250|www.website.com": 1.01,
				"banner|300x250|*":               2.01,
				"banner|300x600|www.website.com": 3.01,
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
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ErrList := validateFloorRulesAndLowerValidRuleKey(tc.floorExt.Data.ModelGroups[0].Schema, tc.floorExt.Data.ModelGroups[0].Schema.Delimiter, tc.floorExt.Data.ModelGroups[0].Values)

			if !reflect.DeepEqual(ErrList[0].Error(), tc.Err) {
				t.Errorf("Incorrect Error: \nreturn:\t%v\nwant:\t%v", ErrList[0].Error(), tc.Err)
			}

			if !reflect.DeepEqual(tc.floorExt.Data.ModelGroups[0].Values, tc.expctedFloor) {
				t.Errorf("Mismatch in floor rules: \nreturn:\t%v\nwant:\t%v", tc.floorExt.Data.ModelGroups[0].Values, tc.expctedFloor)
			}

		})
	}
}
