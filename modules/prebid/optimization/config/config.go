package structs

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/xeipuuv/gojsonschema"
)

const jsonSchemaFile = "rules-engine-schema.json"

func validateConfig(rawCfg json.RawMessage) error {
	jsonSchemaFilePath, err := filepath.Abs(jsonSchemaFile)
	if err != nil {
		return errors.New("filepath.Abs: " + err.Error())
	}

	schemaLoader := gojsonschema.NewReferenceLoader("file:///" + jsonSchemaFilePath)
	schemaValidator, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return errors.New("NewSchema: " + err.Error())
	}

	result, err := schemaValidator.Validate(gojsonschema.NewBytesLoader(rawCfg))
	if err != nil {
		return errors.New("Validate: " + err.Error())
	}
	if !result.Valid() {
		errBuilder := bytes.NewBuffer(make([]byte, 0, 300))
		for _, err := range result.Errors() {
			errBuilder.WriteString(err.String() + " | ")
		}
		return errors.New(errBuilder.String())
	}

	return nil
}

func validateRuleSet(r *RuleSet) error {
	modelGroupWeights := 0
	for i := 0; i < len(r.ModelGroups); i++ {
		if r.ModelGroups[i].Weight == 100 && len(r.ModelGroups) > 1 {
			return fmt.Errorf("Weight of model group %d is 100, leaving no margin for other model group weights", i)
		}

		for j := 0; j < len(r.ModelGroups[i].Rules); j++ {
			if len(r.ModelGroups[i].Schema) != len(r.ModelGroups[i].Rules[j].Conditions) {
				return fmt.Errorf("ModelGroup %d number of schema functions differ from number of conditions of rule %d", i, j)
			}
		}

		if r.ModelGroups[i].Weight == 0 {
			r.ModelGroups[i].Weight = 100
		}

		modelGroupWeights += r.ModelGroups[i].Weight
	}

	if modelGroupWeights != 100 && modelGroupWeights != len(r.ModelGroups)*100 {
		return fmt.Errorf("Model group weights do not add to 100. Sum %d", modelGroupWeights)
	}

	return nil
}

func NewConfig(data json.RawMessage) (PbRulesEngine, error) {
	var cfg PbRulesEngine

	if err := validateConfig(data); err != nil {
		return cfg, err
	}

	if err := jsonutil.UnmarshalValid(data, &cfg); err != nil {
		return cfg, fmt.Errorf("failed to parse config: %w", err)
	}

	for i := 0; i < len(cfg.RuleSets); i++ {
		if err := validateRuleSet(&cfg.RuleSets[i]); err != nil {
			return cfg, fmt.Errorf("Ruleset no %d is invalid: %s", i, err.Error())
		}
	}

	return cfg, nil
}
