package config

import (
	"encoding/json"
	"fmt"
)

type Config struct {
	AllowReject bool `json:"allow_reject"`
	Attributes  struct {
		Name string `json:"name"`
	} `json:"attributes"`
}

func New(conf json.RawMessage) (Config, error) {
	// we can set default values
	c := &Config{AllowReject: true}
	if err := json.Unmarshal(conf, c); err != nil {
		return *c, fmt.Errorf("failed to parse config: %s", err)
	}

	return *c, nil
}
