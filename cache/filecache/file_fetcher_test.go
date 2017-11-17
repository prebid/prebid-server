package filecache

import (
	"testing"
	"encoding/json"
)

func TestFileFetcher(t *testing.T) {
	fetcher, err := NewEagerConfigFetcher("./filecachetest")
	if err != nil {
		t.Errorf("Failed to create a ConfigFetcher: %v", err)
	}

	configs, errs := fetcher.GetConfigs([]string{"config-1", "config-2"})
	if len(errs) != 0 {
		t.Errorf("There shouldn't be any errors when requesting known configs. Got %v", errs)
	}
	value, hasId := configs["config-1"]
	if !hasId {
		t.Fatalf("Expected config map to have id: config-1")
	}

	var config1Val map[string]string
	if err := json.Unmarshal(value, &config1Val); err != nil {
		t.Errorf("Failed to unmarshal config-1: %v", err)
	}
	if len(config1Val) != 1 {
		t.Errorf("Unexpected config1Val length. Expected %v, Got %s", 1, len(config1Val))
	}
	data, hadKey := config1Val["test"]
	if !hadKey {
		t.Errorf("config1Val should have had a \"test\" key, but it didn't.")
	}
	if data != "foo" {
		t.Errorf("Unexpected config-1 \"test\" data. Expected %s, Got %s", "foo", data)
	}

	value, hasId = configs["config-2"]
	if !hasId {
		t.Fatalf("Expected config map to have id: config-2")
	}

	var config2Val string
	if err := json.Unmarshal(value, &config2Val); err != nil {
		t.Errorf("Failed to unmarshal config-2: %v", err)
	}
	if config2Val != "esca\"ped" {
		t.Errorf("Bad config-2 data. Expected %v, Got %s", "esca\"ped", config2Val)
	}
}

func TestInvalidDirectory(t *testing.T) {
	_, err := NewEagerConfigFetcher("./nonexistant-directory")
	if err == nil {
		t.Errorf("There should be an error if we use a directory which doesn't exist.")
	}
}
