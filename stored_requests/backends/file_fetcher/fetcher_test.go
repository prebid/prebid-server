package file_fetcher

import (
	"context"
	"encoding/json"
	"testing"
)

func TestFileFetcher(t *testing.T) {
	fetcher, err := NewFileFetcher("./test")
	if err != nil {
		t.Errorf("Failed to create a Fetcher: %v", err)
	}

	storedReqs, errs := fetcher.FetchRequests(context.Background(), []string{"1", "2"})
	if len(errs) != 0 {
		t.Errorf("There shouldn't be any errors when requesting known stored requests. Got %v", errs)
	}
	value, hasId := storedReqs["1"]
	if !hasId {
		t.Fatalf("Expected stored request data to have id: %d", 1)
	}

	var req1Val map[string]string
	if err := json.Unmarshal(value, &req1Val); err != nil {
		t.Errorf("Failed to unmarshal 1: %v", err)
	}
	if len(req1Val) != 1 {
		t.Errorf("Unexpected req1Val length. Expected %d, Got %d", 1, len(req1Val))
	}
	data, hadKey := req1Val["test"]
	if !hadKey {
		t.Errorf("req1Val should have had a \"test\" key, but it didn't.")
	}
	if data != "foo" {
		t.Errorf(`Bad data in "test" of stored request "1". Expected %s, Got %s`, "foo", data)
	}

	value, hasId = storedReqs["2"]
	if !hasId {
		t.Fatalf("Expected stored request map to have id: %d", 2)
	}

	var req2Val string
	if err := json.Unmarshal(value, &req2Val); err != nil {
		t.Errorf("Failed to unmarshal %d: %v", 2, err)
	}
	if req2Val != `esca"ped` {
		t.Errorf(`Bad data in stored request "2". Expected %v, Got %s`, `esca"ped`, req2Val)
	}
}

func TestInvalidDirectory(t *testing.T) {
	_, err := NewFileFetcher("./nonexistant-directory")
	if err == nil {
		t.Errorf("There should be an error if we use a directory which doesn't exist.")
	}
}
