package floors

import (
	"fmt"
	"strings"

	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

var validSchemaDimensions = map[string]struct{}{
	SiteDomain: {},
	PubDomain:  {},
	Domain:     {},
	Bundle:     {},
	Channel:    {},
	MediaType:  {},
	Size:       {},
	GptSlot:    {},
	AdUnitCode: {},
	Country:    {},
	DeviceType: {},
}

// validateSchemaDimensions validates schema dimesions given in floors JSON
func validateSchemaDimensions(fields []string) error {
	for i := range fields {
		if _, isPresent := validSchemaDimensions[fields[i]]; !isPresent {
			return fmt.Errorf("Invalid schema dimension provided = '%s' in Schema Fields = '%v'", fields[i], fields)
		}
	}
	return nil
}

// validateFloorRulesAndLowerValidRuleKey validates rule keys for number of schema dimension fields and drops invalid rules.
// It also lower case of rule if any charactor in a rule is upper
func validateFloorRulesAndLowerValidRuleKey(schema openrtb_ext.PriceFloorSchema, delimiter string, ruleValues map[string]float64) []error {
	var errs []error
	for key, val := range ruleValues {
		parsedKey := strings.Split(key, delimiter)
		if len(parsedKey) != len(schema.Fields) {
			// Number of fields in rule and number of schema fields are not matching
			errs = append(errs, fmt.Errorf("Invalid Floor Rule = '%s' for Schema Fields = '%v'", key, schema.Fields))
			delete(ruleValues, key)
			continue
		}
		lowerKey := strings.ToLower(key)
		if strings.Compare(key, lowerKey) != 0 {
			delete(ruleValues, key)
			ruleValues[lowerKey] = val
		}
	}
	return errs
}

// validateFloorParams validates SchemaVersion, SkipRate and FloorMin
func validateFloorParams(extFloorRules *openrtb_ext.PriceFloorRules) error {
	if extFloorRules.Data != nil && extFloorRules.Data.FloorsSchemaVersion != 0 && extFloorRules.Data.FloorsSchemaVersion != 2 {
		return fmt.Errorf("Invalid FloorsSchemaVersion = '%v', supported version 2", extFloorRules.Data.FloorsSchemaVersion)
	}

	if extFloorRules.Data != nil && (extFloorRules.Data.SkipRate < skipRateMin || extFloorRules.Data.SkipRate > skipRateMax) {
		return fmt.Errorf("Invalid SkipRate = '%v' at ext.prebid.floors.data.skiprate", extFloorRules.Data.SkipRate)
	}

	if extFloorRules.SkipRate < skipRateMin || extFloorRules.SkipRate > skipRateMax {
		return fmt.Errorf("Invalid SkipRate = '%v' at ext.prebid.floors.skiprate", extFloorRules.SkipRate)
	}

	if extFloorRules.FloorMin < 0.0 {
		return fmt.Errorf("Invalid FloorMin = '%v', value should be >= 0", extFloorRules.FloorMin)
	}

	return nil
}

// selectValidFloorModelGroups validates each modelgroup for SkipRate and ModelGroup and drops invalid modelGroups
func selectValidFloorModelGroups(modelGroups []openrtb_ext.PriceFloorModelGroup, account config.Account) ([]openrtb_ext.PriceFloorModelGroup, []error) {
	var errs []error
	var validModelGroups []openrtb_ext.PriceFloorModelGroup
	if len(modelGroups) == 0 {
		return validModelGroups, []error{fmt.Errorf("No model group present in floors.data")}
	}

	for _, modelGroup := range modelGroups {
		if err := validateSchemaDimensions(modelGroup.Schema.Fields); err != nil {
			errs = append(errs, err)
			continue
		}

		if account.PriceFloors.MaxSchemaDims > 0 && len(modelGroup.Schema.Fields) > account.PriceFloors.MaxSchemaDims {
			errs = append(errs, fmt.Errorf("Invalid Floor Model = '%v' due to number of schema fields = '%v' are greater than limit %v", modelGroup.ModelVersion, len(modelGroup.Schema.Fields), account.PriceFloors.MaxSchemaDims))
			continue
		}

		if account.PriceFloors.MaxRule > 0 && len(modelGroup.Values) > account.PriceFloors.MaxRule {
			errs = append(errs, fmt.Errorf("Invalid Floor Model = '%v' due to number of rules = '%v' are greater than limit %v", modelGroup.ModelVersion, len(modelGroup.Values), account.PriceFloors.MaxRule))
			continue
		}

		if modelGroup.SkipRate < skipRateMin || modelGroup.SkipRate > skipRateMax {
			errs = append(errs, fmt.Errorf("Invalid Floor Model = '%v' due to SkipRate = '%v' is out of range (1-100)", modelGroup.ModelVersion, modelGroup.SkipRate))
			continue
		}

		if modelGroup.ModelWeight != nil && (*modelGroup.ModelWeight < modelWeightMin || *modelGroup.ModelWeight > modelWeightMax) {
			errs = append(errs, fmt.Errorf("Invalid Floor Model = '%v' due to ModelWeight = '%v' is out of range (1-100)", modelGroup.ModelVersion, *modelGroup.ModelWeight))
			continue
		}

		if modelGroup.Default < 0.0 {
			errs = append(errs, fmt.Errorf("Invalid Floor Model = '%v' due to Default = '%v' is less than 0", modelGroup.ModelVersion, modelGroup.Default))
			continue
		}

		validModelGroups = append(validModelGroups, modelGroup)
	}
	return validModelGroups, errs
}
