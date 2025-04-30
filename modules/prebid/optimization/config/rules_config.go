package structs

import "encoding/json"

type ModelGroup struct {
	Weight       int
	AnalyticsKey string
	Version      string

	Schema []Schema `json:"schema,omitempty"`
	Rule   []Rule   `json:"rules,omitempty"`
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
