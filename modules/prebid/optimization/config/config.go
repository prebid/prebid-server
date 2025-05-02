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

const rulesEngineSchemaFile = "rules-engine-schema.json"

func createSchemaValidator(jsonSchemaFile string) (*gojsonschema.Schema, error) {
	jsonSchemaFilePath, err := filepath.Abs(jsonSchemaFile)
	if err != nil {
		return nil, err
	}

	schemaLoader := gojsonschema.NewReferenceLoader("file:///" + jsonSchemaFilePath)
	return gojsonschema.NewSchema(schemaLoader)
}

func validateConfig(rawCfg json.RawMessage, schemaValidator *gojsonschema.Schema) error {
	result, err := schemaValidator.Validate(gojsonschema.NewBytesLoader(rawCfg))
	if err != nil {
		return err
	}
	if !result.Valid() {
		errBuilder := bytes.NewBuffer(make([]byte, 0, 300))
		for _, err := range result.Errors() {
			errBuilder.WriteString("[" + err.String() + "] ")
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

func NewConfig(jsonCfg json.RawMessage) (*PbRulesEngine, error) {
	cfg := &PbRulesEngine{}

	validator, err := createSchemaValidator(rulesEngineSchemaFile)
	if err != nil {
		return nil, fmt.Errorf("Error creating validator: %s", err.Error())
	}

	if err = validateConfig(jsonCfg, validator); err != nil {
		return nil, fmt.Errorf("JSON schema validation: %s", err.Error())
	}

	if err = jsonutil.UnmarshalValid(jsonCfg, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	for i := 0; i < len(cfg.RuleSets); i++ {
		if err = validateRuleSet(&cfg.RuleSets[i]); err != nil {
			return nil, fmt.Errorf("Ruleset no %d is invalid: %s", i, err.Error())
		}
	}

	return cfg, nil
}
