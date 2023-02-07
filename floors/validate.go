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

func validateFloorParams(extFloorRules *openrtb_ext.PriceFloorRules) error {

	if extFloorRules.Data != nil && len(extFloorRules.Data.FloorsSchemaVersion) > 0 && extFloorRules.Data.FloorsSchemaVersion != "2" {
		return fmt.Errorf("Invalid FloorsSchemaVersion = '%v', supported version 2", extFloorRules.Data.FloorsSchemaVersion)
	}

	if extFloorRules.Data != nil && (extFloorRules.Data.SkipRate < skipRateMin || extFloorRules.Data.SkipRate > skipRateMax) {
		return fmt.Errorf("Invalid SkipRate = '%v' at  at ext.floors.data.skiprate", extFloorRules.Data.SkipRate)
	}

	if extFloorRules.SkipRate < skipRateMin || extFloorRules.SkipRate > skipRateMax {
		return fmt.Errorf("Invalid SkipRate = '%v' at ext.floors.skiprate", extFloorRules.SkipRate)
	}

	if extFloorRules.FloorMin < float64(0) {
		return fmt.Errorf("Invalid FloorMin = '%v', value should be >= 0", extFloorRules.FloorMin)
	}

	return nil
}

func selectValidFloorModelGroups(modelGroups []openrtb_ext.PriceFloorModelGroup) ([]openrtb_ext.PriceFloorModelGroup, []error) {
	var errs []error
	var validModelGroups []openrtb_ext.PriceFloorModelGroup
	for _, modelGroup := range modelGroups {
		if modelGroup.SkipRate < skipRateMin || modelGroup.SkipRate > skipRateMax {
			errs = append(errs, fmt.Errorf("Invalid Floor Model = '%v' due to SkipRate = '%v' is out of range (1-100)", modelGroup.ModelVersion, modelGroup.SkipRate))
			continue
		}

		if modelGroup.ModelWeight != nil && (*modelGroup.ModelWeight < modelWeightMin || *modelGroup.ModelWeight > modelWeightMax) {
			errs = append(errs, fmt.Errorf("Invalid Floor Model = '%v' due to ModelWeight = '%v' is out of range (1-100)", modelGroup.ModelVersion, *modelGroup.ModelWeight))
			continue
		}

		if modelGroup.Default < float64(0) {
			errs = append(errs, fmt.Errorf("Invalid Floor Model = '%v' due to Default = '%v' is less than 0", modelGroup.ModelVersion, modelGroup.Default))
			continue
		}

		validModelGroups = append(validModelGroups, modelGroup)
	}
	return validModelGroups, errs
}
