package floors

import (
	"errors"
	"fmt"
	"testing"

	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestValidateFloorParams(t *testing.T) {

	tt := []struct {
		name     string
		floorExt *openrtb_ext.PriceFloorRules
		Err      error
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
		},
		{
			name:     "Invalid Skip Rate at Root level",
			floorExt: &openrtb_ext.PriceFloorRules{SkipRate: -10},
			Err:      errors.New("Invalid SkipRate = '-10' at ext.prebid.floors.skiprate"),
		},
		{
			name: "Invalid Skip Rate at Data level",
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
			Err: errors.New("Invalid SkipRate = '-10' at ext.prebid.floors.data.skiprate"),
		},
		{
			name:     "Invalid FloorMin ",
			floorExt: &openrtb_ext.PriceFloorRules{FloorMin: -10},
			Err:      errors.New("Invalid FloorMin = '-10', value should be >= 0"),
		},
		{
			name: "Invalid FloorSchemaVersion 2",
			floorExt: &openrtb_ext.PriceFloorRules{Data: &openrtb_ext.PriceFloorData{
				FloorsSchemaVersion: 1,
				ModelGroups: []openrtb_ext.PriceFloorModelGroup{{
					ModelVersion: "Version 1",
					Schema:       openrtb_ext.PriceFloorSchema{Fields: []string{"mediaType", "size", "domain"}, Delimiter: "|"},
					Values: map[string]float64{
						"banner|300x250|www.website.com": 1.01,
						"banner|300x600|*":               4.01,
					}, Default: 0.01},
				}}},
			Err: errors.New("Invalid FloorsSchemaVersion = '1', supported version 2"),
		},
		{
			name: "Invalid FloorSchemaVersion -2",
			floorExt: &openrtb_ext.PriceFloorRules{Data: &openrtb_ext.PriceFloorData{
				FloorsSchemaVersion: -2,
				ModelGroups: []openrtb_ext.PriceFloorModelGroup{{
					ModelVersion: "Version 1",
					Schema:       openrtb_ext.PriceFloorSchema{Fields: []string{"mediaType", "size", "domain"}, Delimiter: "|"},
					Values: map[string]float64{
						"banner|300x250|www.website.com": 1.01,
						"banner|300x600|*":               4.01,
					}, Default: 0.01},
				}}},
			Err: errors.New("Invalid FloorsSchemaVersion = '-2', supported version 2"),
		},
		{
			name: "Valid FloorSchemaVersion 0",
			floorExt: &openrtb_ext.PriceFloorRules{Data: &openrtb_ext.PriceFloorData{
				FloorsSchemaVersion: 0,
				ModelGroups: []openrtb_ext.PriceFloorModelGroup{{
					ModelVersion: "Version 1",
					Schema:       openrtb_ext.PriceFloorSchema{Fields: []string{"mediaType", "size", "domain"}, Delimiter: "|"},
					Values: map[string]float64{
						"banner|300x250|www.website.com": 1.01,
						"banner|300x600|*":               4.01,
					}, Default: 0.01},
				}}},
		},
		{
			name: "Valid FloorSchemaVersion 2",
			floorExt: &openrtb_ext.PriceFloorRules{Data: &openrtb_ext.PriceFloorData{
				FloorsSchemaVersion: 2,
				ModelGroups: []openrtb_ext.PriceFloorModelGroup{{
					ModelVersion: "Version 1",
					Schema:       openrtb_ext.PriceFloorSchema{Fields: []string{"mediaType", "size", "domain"}, Delimiter: "|"},
					Values: map[string]float64{
						"banner|300x250|www.website.com": 1.01,
						"banner|300x600|*":               4.01,
					}, Default: 0.01},
				}}},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			actErr := validateFloorParams(tc.floorExt)
			assert.Equal(t, actErr, tc.Err, tc.name)
		})
	}
}

func TestSelectValidFloorModelGroups(t *testing.T) {

	tt := []struct {
		name     string
		floorExt *openrtb_ext.PriceFloorRules
		account  config.Account
		Err      []error
	}{
		{
			name: "Invalid Skip Rate in model Group 1",
			account: config.Account{
				PriceFloors: config.AccountPriceFloors{
					MaxRule:       100,
					MaxSchemaDims: 5,
				},
			},
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
			Err: []error{errors.New("Invalid Floor Model = 'Version 1' due to SkipRate = '110' is out of range (1-100)")},
		},
		{
			name: "Invalid model weight Model Group 1",
			account: config.Account{
				PriceFloors: config.AccountPriceFloors{
					MaxRule:       100,
					MaxSchemaDims: 5,
				},
			},
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
			Err: []error{errors.New("Invalid Floor Model = 'Version 1' due to ModelWeight = '-1' is out of range (1-100)")},
		},
		{
			name: "Invalid Default Value",
			account: config.Account{
				PriceFloors: config.AccountPriceFloors{
					MaxRule:       100,
					MaxSchemaDims: 5,
				},
			},
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
			Err: []error{errors.New("Invalid Floor Model = 'Version 1' due to Default = '-1' is less than 0")},
		},
		{
			name: "Invalid Number of Schema dimensions",
			account: config.Account{
				PriceFloors: config.AccountPriceFloors{
					MaxRule:       100,
					MaxSchemaDims: 2,
				},
			},
			floorExt: &openrtb_ext.PriceFloorRules{Data: &openrtb_ext.PriceFloorData{
				ModelGroups: []openrtb_ext.PriceFloorModelGroup{{
					ModelVersion: "Version 1",
					Schema:       openrtb_ext.PriceFloorSchema{Fields: []string{"mediaType", "size", "domain"}},
					Values: map[string]float64{
						"banner|300x250|www.website.com": 1.01,
						"*|728x90|*":                     14.01,
						"*|*|www.website.com":            15.01,
						"*|*|*":                          16.01,
					}},
				}}},
			Err: []error{errors.New("Invalid Floor Model = 'Version 1' due to number of schema fields = '3' are greater than limit 2")},
		},
		{
			name: "Invalid Schema field creativeType",
			account: config.Account{
				PriceFloors: config.AccountPriceFloors{
					MaxRule:       100,
					MaxSchemaDims: 3,
				},
			},
			floorExt: &openrtb_ext.PriceFloorRules{Data: &openrtb_ext.PriceFloorData{
				ModelGroups: []openrtb_ext.PriceFloorModelGroup{{
					ModelVersion: "Version 1",
					Schema:       openrtb_ext.PriceFloorSchema{Fields: []string{"creativeType", "size", "domain"}},
					Values: map[string]float64{
						"banner|300x250|www.website.com": 1.01,
						"*|728x90|*":                     14.01,
						"*|*|www.website.com":            15.01,
						"*|*|*":                          16.01,
					}},
				}}},
			Err: []error{errors.New("Invalid schema dimension provided = 'creativeType' in Schema Fields = '[creativeType size domain]'")},
		},
		{
			name: "Invalid Number of rules",
			account: config.Account{
				PriceFloors: config.AccountPriceFloors{
					MaxRule:       3,
					MaxSchemaDims: 5,
				},
			},
			floorExt: &openrtb_ext.PriceFloorRules{Data: &openrtb_ext.PriceFloorData{
				ModelGroups: []openrtb_ext.PriceFloorModelGroup{{
					ModelVersion: "Version 1",
					Schema:       openrtb_ext.PriceFloorSchema{Fields: []string{"mediaType", "size", "domain"}},
					Values: map[string]float64{
						"banner|300x250|www.website.com": 1.01,
						"*|728x90|*":                     14.01,
						"*|*|www.website.com":            15.01,
						"*|*|*":                          16.01,
					}},
				}}},
			Err: []error{errors.New("Invalid Floor Model = 'Version 1' due to number of rules = '4' are greater than limit 3")},
		},
		{
			name: "No Modelgroup present",
			floorExt: &openrtb_ext.PriceFloorRules{Data: &openrtb_ext.PriceFloorData{
				ModelGroups: []openrtb_ext.PriceFloorModelGroup{}}},
			Err: []error{errors.New("No model group present in floors.data")},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			_, ErrList := selectValidFloorModelGroups(tc.floorExt.Data.ModelGroups, tc.account)
			assert.Equal(t, ErrList, tc.Err, tc.name)
		})
	}
}

func TestValidateFloorRulesAndLowerValidRuleKey(t *testing.T) {

	tt := []struct {
		name         string
		floorExt     *openrtb_ext.PriceFloorRules
		Err          []error
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
			Err: []error{errors.New("Invalid Floor Rule = 'banner|300x600|www.website.com|www.test.com' for Schema Fields = '[mediaType size domain]'")},
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
			Err: []error{errors.New("Invalid Floor Rule = 'banner|300x600' for Schema Fields = '[mediaType size domain]'")},
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
			assert.Equal(t, ErrList, tc.Err, tc.name)
			assert.Equal(t, tc.floorExt.Data.ModelGroups[0].Values, tc.expctedFloor, tc.name)
		})
	}
}

func TestValidateSchemaDimensions(t *testing.T) {
	tests := []struct {
		name   string
		fields []string
		err    error
	}{
		{
			name:   "valid_fields",
			fields: []string{"deviceType", "size"},
		},
		{
			name:   "invalid_fields",
			fields: []string{"deviceType", "dealType"},
			err:    fmt.Errorf("Invalid schema dimension provided = 'dealType' in Schema Fields = '[deviceType dealType]'"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSchemaDimensions(tt.fields)
			assert.Equal(t, tt.err, err)
		})
	}
}
