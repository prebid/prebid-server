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

func NewConfig(data json.RawMessage) (PbRulesEngine, error) {
	var cfg PbRulesEngine

	// Schema validation
	if err := ValidateConfig(data); err != nil {
		return cfg, err
	}

	// Unmarshal
	if err := jsonutil.UnmarshalValid(data, &cfg); err != nil {
		return cfg, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate further

	return cfg, nil
}

const jsonSchemaFile = "rules-engine-schema.json"

func ValidateConfig(rawCfg json.RawMessage) error {
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
