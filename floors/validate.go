package floors

import (
	"fmt"
	"strings"

	"github.com/prebid/prebid-server/openrtb_ext"
)

func validateFloorRules(Schema openrtb_ext.PriceFloorSchema, delimiter string, RuleValues map[string]float64) []error {
	var errs []error
	for key, val := range RuleValues {
		parsedKey := strings.Split(key, delimiter)
		if len(parsedKey) != len(Schema.Fields) {
			// Number of fields in rule and number of schema fields are not matching
			errs = append(errs, fmt.Errorf("Invalid Floor Rule = '%s' for Schema Fields = '%v'", key, Schema.Fields))
			delete(RuleValues, key)
		}
		delete(RuleValues, key)
		newKey := strings.ToLower(key)
		RuleValues[newKey] = val
	}
	return errs
}

func validateFloorModelGroups(modelGroups []openrtb_ext.PriceFloorModelGroup) ([]openrtb_ext.PriceFloorModelGroup, []error) {
	var errs []error
	var validModelGroups []openrtb_ext.PriceFloorModelGroup
	for _, modelGroup := range modelGroups {
		if modelGroup.SkipRate < SKIP_RATE_MIN || modelGroup.SkipRate > SKIP_RATE_MAX {
			errs = append(errs, fmt.Errorf("invalid Floor Model = '%v' due to SkipRate = '%v'", modelGroup.ModelVersion, modelGroup.SkipRate))
			continue
		}

		if modelGroup.ModelWeight < MODEL_WEIGHT_MIN_VALUE || modelGroup.ModelWeight > MODEL_WEIGHT_MAX_VALUE {
			errs = append(errs, fmt.Errorf("invalid Floor Model = '%v' due to ModelWeight = '%v'", modelGroup.ModelVersion, modelGroup.ModelWeight))
			continue
		}

		validModelGroups = append(validModelGroups, modelGroup)
	}
	return validModelGroups, errs
}
