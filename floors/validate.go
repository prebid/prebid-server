package floors

import (
	"fmt"
	"strings"

	"github.com/prebid/prebid-server/openrtb_ext"
)

func validateFloorRules(schema openrtb_ext.PriceFloorSchema, delimiter string, ruleValues map[string]float64) []error {
	var errs []error
	for key, val := range ruleValues {
		parsedKey := strings.Split(key, delimiter)
		if len(parsedKey) != len(schema.Fields) {
			// Number of fields in rule and number of schema fields are not matching
			errs = append(errs, fmt.Errorf("Invalid Floor Rule = '%s' for Schema Fields = '%v'", key, schema.Fields))
			delete(ruleValues, key)
			continue
		}
		delete(ruleValues, key)
		newKey := strings.ToLower(key)
		ruleValues[newKey] = val
	}
	return errs
}

func validateFloorSkipRates(floorExt *openrtb_ext.PriceFloorRules) []error {
	var errs []error

	if floorExt.Data != nil && (floorExt.Data.SkipRate < SKIP_RATE_MIN || floorExt.Data.SkipRate > SKIP_RATE_MAX) {
		errs = append(errs, fmt.Errorf("Invalid SkipRate at data level = '%v'", floorExt.Data.SkipRate))
		return errs
	}

	if floorExt.SkipRate < SKIP_RATE_MIN || floorExt.SkipRate > SKIP_RATE_MAX {
		errs = append(errs, fmt.Errorf("Invalid SkipRate at root level = '%v'", floorExt.SkipRate))
	}
	return errs
}

func selectValidFloorModelGroups(modelGroups []openrtb_ext.PriceFloorModelGroup) ([]openrtb_ext.PriceFloorModelGroup, []error) {
	var errs []error
	var validModelGroups []openrtb_ext.PriceFloorModelGroup
	for _, modelGroup := range modelGroups {
		if modelGroup.SkipRate < SKIP_RATE_MIN || modelGroup.SkipRate > SKIP_RATE_MAX {
			errs = append(errs, fmt.Errorf("Invalid Floor Model = '%v' due to SkipRate = '%v'", modelGroup.ModelVersion, modelGroup.SkipRate))
			continue
		}

		if modelGroup.ModelWeight < MODEL_WEIGHT_MIN_VALUE || modelGroup.ModelWeight > MODEL_WEIGHT_MAX_VALUE {
			errs = append(errs, fmt.Errorf("Invalid Floor Model = '%v' due to ModelWeight = '%v'", modelGroup.ModelVersion, modelGroup.ModelWeight))
			continue
		}

		validModelGroups = append(validModelGroups, modelGroup)
	}
	return validModelGroups, errs
}
