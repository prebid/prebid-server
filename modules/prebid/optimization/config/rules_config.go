package structs

import (
	"encoding/json"
)

type PbRulesEngine struct {
	Enabled                       bool      `json:"enabled,omitempty"`
	GenerateRulesFromBidderConfig bool      `json:"generateRulesFromBidderConfig,omitempty"`
	Timestamp                     string    `json:"timestamp,omitempty"`
	RuleSets                      []RuleSet `json:"ruleSets,omitempty"`
}

type RuleSet struct {
	Stage       string       `json:"stage,omitempty"`
	Name        string       `json:"name,omitempty"`
	Version     string       `json:"version,omitempty"`
	ModelGroups []ModelGroup `json:"modelGroups,omitempty"`
}

type ModelGroup struct {
	Weight       int
	AnalyticsKey string
	Version      string

	Schema []Schema `json:"schema,omitempty"`
	Rules  []Rule   `json:"rules,omitempty"`
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
