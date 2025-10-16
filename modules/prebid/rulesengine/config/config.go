package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/prebid/prebid-server/v3/hooks"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/xeipuuv/gojsonschema"
)

const RulesEngineSchemaFile = "rules-engine-schema.json"
const RulesEngineSchemaFilePath = "modules/prebid/rulesengine/config/" + RulesEngineSchemaFile

type PbRulesEngine struct {
	Enabled   bool      `json:"enabled,omitempty"`
	Timestamp string    `json:"timestamp,omitempty"`
	RuleSets  []RuleSet `json:"rulesets,omitempty"`
}

type RuleSet struct {
	Stage       hooks.Stage  `json:"stage,omitempty"`
	Name        string       `json:"name,omitempty"`
	Version     string       `json:"version,omitempty"`
	ModelGroups []ModelGroup `json:"modelgroups,omitempty"`
}

type ModelGroup struct {
	Weight       int
	AnalyticsKey string
	Version      string

	Schema  []Schema `json:"schema,omitempty"`
	Rules   []Rule   `json:"rules,omitempty"`
	Default []Result `json:"default,omitempty"`
}

type Schema struct {
	Func string          `json:"function,omitempty"`
	Args json.RawMessage `json:"args,omitempty"`
}

type Rule struct {
	Conditions []string `json:"conditions,omitempty"`
	Results    []Result `json:"results,omitempty"`
}

type Result struct {
	Func string          `json:"function,omitempty"`
	Args json.RawMessage `json:"args,omitempty"`
}

// ResultFuncParams is a struct that holds parameters for result functions and is used in ExcludeBidders and IncludeBidders.
type ResultFuncParams struct {
	Bidders        []string `json:"bidders,omitempty"`
	SeatNonBid     int      `json:"seatnonbid,omitempty"`
	AnalyticsValue string   `json:"analyticsvalue,omitempty"`
	IfSyncedId     bool     `json:"ifsyncedid,omitempty"`
}

func CreateSchemaValidator(jsonSchemaFile string) (*gojsonschema.Schema, error) {
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
	for i := 0; i < len(r.ModelGroups); i++ {
		// Modelgroup weight defaults to 100
		if r.ModelGroups[i].Weight == 0 {
			r.ModelGroups[i].Weight = 100
		}

		if len(r.ModelGroups[i].Schema) > 0 && len(r.ModelGroups[i].Rules) == 0 {
			return fmt.Errorf("ModelGroup %d has schema functions but no rules", i)
		}

		if len(r.ModelGroups[i].Schema) == 0 && len(r.ModelGroups[i].Rules) > 0 {
			return fmt.Errorf("ModelGroup %d has no schema functions to test its rules against", i)
		}

		for j := 0; j < len(r.ModelGroups[i].Rules); j++ {
			if len(r.ModelGroups[i].Schema) != len(r.ModelGroups[i].Rules[j].Conditions) {
				return fmt.Errorf("ModelGroup %d number of schema functions differ from number of conditions of rule %d", i, j)
			}
		}
	}

	return nil
}

func NewConfig(jsonCfg json.RawMessage, validator *gojsonschema.Schema) (*PbRulesEngine, error) {
	cfg := &PbRulesEngine{}

	if err := validateConfig(jsonCfg, validator); err != nil {
		return nil, fmt.Errorf("JSON schema validation: %s", err.Error())
	}

	if err := jsonutil.UnmarshalValid(jsonCfg, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	for i := 0; i < len(cfg.RuleSets); i++ {
		if err := validateRuleSet(&cfg.RuleSets[i]); err != nil {
			return nil, fmt.Errorf("Ruleset no %d is invalid: %s", i, err.Error())
		}
	}

	return cfg, nil
}
