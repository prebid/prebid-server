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

	storedReqs, storedImps, errs := fetcher.FetchRequests(context.Background(), []string{"1", "2"}, []string{"some-imp"})
	assertErrorCount(t, 0, errs)

	validateStoredReqOne(t, storedReqs)
	validateStoredReqTwo(t, storedReqs)
	validateImp(t, storedImps)
}

func TestInvalidDirectory(t *testing.T) {
	_, err := NewFileFetcher("./nonexistant-directory")
	if err == nil {
		t.Errorf("There should be an error if we use a directory which doesn't exist.")
	}
}

func validateStoredReqOne(t *testing.T, storedRequests map[string]json.RawMessage) {
	value, hasID := storedRequests["1"]
	if !hasID {
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
}

func validateStoredReqTwo(t *testing.T, storedRequests map[string]json.RawMessage) {
	value, hasId := storedRequests["2"]
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

func validateImp(t *testing.T, storedImps map[string]json.RawMessage) {
	value, hasId := storedImps["some-imp"]
	if !hasId {
		t.Fatal("Expected Stored Imp map to have id: some-imp")
	}

	var impVal map[string]bool
	if err := json.Unmarshal(value, &impVal); err != nil {
		t.Errorf("Failed to unmarshal some-imp: %v", err)
	}
	if len(impVal) != 1 {
		t.Errorf("Unexpected impVal length. Expected %d, Got %d", 1, len(impVal))
	}
	data, hadKey := impVal["imp"]
	if !hadKey {
		t.Errorf("some-imp should have had a \"imp\" key, but it didn't.")
	}
	if !data {
		t.Errorf(`Bad data in "imp" of stored request "some-imp". Expected true, Got %t`, data)
	}
}

func assertErrorCount(t *testing.T, num int, errs []error) {
	t.Helper()
	if len(errs) != num {
		t.Errorf("Wrong number of errors. Expected %d. Got %d. Errors are %v", num, len(errs), errs)
	}
}
