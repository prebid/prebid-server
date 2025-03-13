package optimizationmodule

import "encoding/json"

type Conf struct {
	Schema []Schema `json:"schema,omitempty"`
	Rule   []Rule   `json:"rules,omitempty"`
}

type Schema struct {
	Func string   `json:"function,omitempty"`
	Args []string `json:"args,omitempty"`
}

type Rule struct {
	Conditions []string `json:"conditions,omitempty"`
	Results    []Result `json:"results,omitempty"`
}

type Result struct {
	Func string          `json:"function,omitempty"`
	Args json.RawMessage `json:"args,omitempty"`
}
