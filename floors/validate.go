package floors

import (
	"fmt"
	"strings"

	"github.com/prebid/prebid-server/openrtb_ext"
)

func validateFloorRulesAndLowerValidRuleKey(schema openrtb_ext.PriceFloorSchema, delimiter string, ruleValues map[string]float64) []error {
	var errs []error
	for key, val := range ruleValues {
		parsedKey := strings.Split(key, delimiter)
		delete(ruleValues, key)
		if len(parsedKey) != len(schema.Fields) {
			// Number of fields in rule and number of schema fields are not matching
			errs = append(errs, fmt.Errorf("Invalid Floor Rule = '%s' for Schema Fields = '%v'", key, schema.Fields))
			continue
		}
		newKey := strings.ToLower(key)
		ruleValues[newKey] = val
	}
	return errs
}

func validateFloorSkipRates(floorExt *openrtb_ext.PriceFloorRules) error {

	if floorExt.Data != nil && (floorExt.Data.SkipRate < skipRateMin || floorExt.Data.SkipRate > skipRateMax) {
		return fmt.Errorf("Invalid SkipRate at data level = '%v'", floorExt.Data.SkipRate)
	}

	if floorExt.SkipRate < skipRateMin || floorExt.SkipRate > skipRateMax {
		return fmt.Errorf("Invalid SkipRate at root level = '%v'", floorExt.SkipRate)
	}
	return nil
}

func selectValidFloorModelGroups(modelGroups []openrtb_ext.PriceFloorModelGroup) ([]openrtb_ext.PriceFloorModelGroup, []error) {
	var errs []error
	var validModelGroups []openrtb_ext.PriceFloorModelGroup
	for _, modelGroup := range modelGroups {
		if modelGroup.SkipRate < skipRateMin || modelGroup.SkipRate > skipRateMax {
			errs = append(errs, fmt.Errorf("Invalid Floor Model = '%v' due to SkipRate = '%v'", modelGroup.ModelVersion, modelGroup.SkipRate))
			continue
		}

		if modelGroup.ModelWeight < modelWeightMin || modelGroup.ModelWeight > modelWeightMax {
			errs = append(errs, fmt.Errorf("Invalid Floor Model = '%v' due to ModelWeight = '%v'", modelGroup.ModelVersion, modelGroup.ModelWeight))
			continue
		}

		validModelGroups = append(validModelGroups, modelGroup)
	}
	return validModelGroups, errs
}
