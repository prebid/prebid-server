package floors

import (
	"reflect"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestValidateFloorSkipRates(t *testing.T) {
	floorExt1 := &openrtb_ext.PriceFloorRules{Data: &openrtb_ext.PriceFloorData{
		ModelGroups: []openrtb_ext.PriceFloorModelGroup{{
			ModelVersion: "Version 1",
			Schema:       openrtb_ext.PriceFloorSchema{Fields: []string{"mediaType", "size", "domain"}, Delimiter: "|"},
			Values: map[string]float64{
				"banner|300x250|www.website.com":              1.01,
				"banner|300x250|*":                            2.01,
				"banner|300x600|www.website.com|www.test.com": 3.01,
				"banner|300x600|*":                            4.01,
			}, Default: 0.01},
		}}}
	floorExt2 := &openrtb_ext.PriceFloorRules{SkipRate: -10}

	floorExt3 := &openrtb_ext.PriceFloorRules{Data: &openrtb_ext.PriceFloorData{
		SkipRate: -10,
		ModelGroups: []openrtb_ext.PriceFloorModelGroup{{
			ModelVersion: "Version 1",
			Schema:       openrtb_ext.PriceFloorSchema{Fields: []string{"mediaType", "size", "domain"}},
			Values: map[string]float64{
				"*|*|www.website.com": 15.01,
				"*|*|*":               16.01,
			}, Default: 0.01},
		}}}

	tt := []struct {
		name     string
		floorExt *openrtb_ext.PriceFloorRules
		Err      string
	}{
		{
			name:     "Valid Skip Rate",
			floorExt: floorExt1,
			Err:      "",
		},
		{
			name:     "Invalid Skip Rate at Root level",
			floorExt: floorExt2,
			Err:      "Invalid SkipRate at root level = '-10'",
		},
		{
			name:     "Invalid Skip Rate at Date level",
			floorExt: floorExt3,
			Err:      "Invalid SkipRate at data level = '-10'",
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ErrList := validateFloorSkipRates(tc.floorExt)

			if len(ErrList) > 0 && !reflect.DeepEqual(ErrList[0].Error(), tc.Err) {
				t.Errorf("Incorrect Error: \nreturn:\t%v\nwant:\t%v", ErrList[0].Error(), tc.Err)
			}

		})
	}
}

func TestValidateFloorModelGroups(t *testing.T) {
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

	floorExt2 := &openrtb_ext.PriceFloorRules{Data: &openrtb_ext.PriceFloorData{
		ModelGroups: []openrtb_ext.PriceFloorModelGroup{{
			ModelWeight:  -1,
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

	tt := []struct {
		name         string
		floorExt     *openrtb_ext.PriceFloorRules
		ModelVersion string
		Err          string
	}{
		{
			name:         "Invalid Skip Rate in model Group 1, with banner|300x250|www.website.com",
			floorExt:     floorExt,
			ModelVersion: "Version 1",
			Err:          "Invalid Floor Model = 'Version 1' due to SkipRate = '110'",
		},
		{
			name:         "Invalid model weight Model Group 1, with banner|300x250|www.website.com",
			floorExt:     floorExt2,
			ModelVersion: "Version 1",
			Err:          "Invalid Floor Model = 'Version 1' due to ModelWeight = '-1'",
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			_, ErrList := validateFloorModelGroups(tc.floorExt.Data.ModelGroups)

			if !reflect.DeepEqual(tc.floorExt.Data.ModelGroups[0].ModelVersion, tc.ModelVersion) {
				t.Errorf("Floor Model Version mismatch error: \nreturn:\t%v\nwant:\t%v", tc.floorExt.Data.ModelGroups[0].ModelVersion, tc.ModelVersion)
			}

			if !reflect.DeepEqual(ErrList[0].Error(), tc.Err) {
				t.Errorf("Incorrect Error: \nreturn:\t%v\nwant:\t%v", ErrList[0].Error(), tc.Err)
			}

		})
	}
}

func TestValidateFloorRules(t *testing.T) {
	floorExt := &openrtb_ext.PriceFloorRules{Data: &openrtb_ext.PriceFloorData{
		ModelGroups: []openrtb_ext.PriceFloorModelGroup{{
			ModelVersion: "Version 1",
			Schema:       openrtb_ext.PriceFloorSchema{Fields: []string{"mediaType", "size", "domain"}, Delimiter: "|"},
			Values: map[string]float64{
				"banner|300x250|www.website.com":              1.01,
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
		}}}

	floorExt2 := &openrtb_ext.PriceFloorRules{Data: &openrtb_ext.PriceFloorData{
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
		}}}

	tt := []struct {
		name     string
		floorExt *openrtb_ext.PriceFloorRules
		Err      string
	}{
		{
			name:     "Invalid floor rule banner|300x600|www.website.com|www.test.com",
			floorExt: floorExt,
			Err:      "Invalid Floor Rule = 'banner|300x600|www.website.com|www.test.com' for Schema Fields = '[mediaType size domain]'",
		},
		{
			name:     "Invalid floor rule banner|300x600|www.website.com|www.test.com",
			floorExt: floorExt2,
			Err:      "Invalid Floor Rule = 'banner|300x600' for Schema Fields = '[mediaType size domain]'",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ErrList := validateFloorRules(tc.floorExt.Data.ModelGroups[0].Schema, tc.floorExt.Data.ModelGroups[0].Schema.Delimiter, tc.floorExt.Data.ModelGroups[0].Values)

			if !reflect.DeepEqual(ErrList[0].Error(), tc.Err) {
				t.Errorf("Incorrect Error: \nreturn:\t%v\nwant:\t%v", ErrList[0].Error(), tc.Err)
			}

		})
	}
}
