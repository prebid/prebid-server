package file_fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/prebid/prebid-server/v3/stored_requests"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/stretchr/testify/assert"
)

func TestFileFetcher(t *testing.T) {
	// Load the test input files for testing
	fetcher, err := NewFileFetcher("./test")
	if err != nil {
		t.Errorf("Failed to create a Fetcher: %v", err)
	}

	// Test stored request and stored imps
	storedReqs, storedImps, errs := fetcher.FetchRequests(context.Background(), []string{"1", "2"}, []string{"some-imp"})
	assertErrorCount(t, 0, errs)

	validateStoredReqOne(t, storedReqs)
	validateStoredReqTwo(t, storedReqs)
	validateImp(t, storedImps)
}

func TestStoredResponseFileFetcher(t *testing.T) {
	// grab the fetcher that do not have /test/stored_responses/stored_responses FS directory
	directoryNotExistfetcher, err := NewFileFetcher("./test/stored_responses")
	if err != nil {
		t.Errorf("Failed to create a Fetcher: %v", err)
	}

	// we should receive 1 error since we do not have "stored_responses" directory in ./test/stored_responses
	_, errs := directoryNotExistfetcher.FetchResponses(context.Background(), []string{})
	assertErrorCount(t, 1, errs)

	// grab the fetcher that has /test/stored_responses FS directory
	fetcher, err := NewFileFetcher("./test")
	if err != nil {
		t.Errorf("Failed to create a Fetcher: %v", err)
	}

	// Test stored responses, we have 3 stored responses in ./test/stored_responses
	storedResps, errs := fetcher.FetchResponses(context.Background(), []string{"bar", "escaped", "does_not_exist"})
	// expect 1 error since we do not have "does_not_exist" stored response file from ./test
	assertErrorCount(t, 1, errs)

	validateStoredResponse[map[string]string](t, storedResps, "bar", func(val map[string]string) error {
		if len(val) != 1 {
			return fmt.Errorf("Unexpected value length. Expected %d, Got %d", 1, len(val))
		}

		data, hadKey := val["test"]
		if !hadKey {
			return fmt.Errorf(`missing key "test" in the value`)
		}

		expectedVal := "bar"
		if data != expectedVal {
			return fmt.Errorf(`Bad value for key "test". Expected "%s", Got "%s"`, expectedVal, data)
		}
		return nil
	})

	validateStoredResponse[string](t, storedResps, "escaped", func(val string) error {
		expectedVal := `esca"ped`
		if val != expectedVal {
			return fmt.Errorf(`Bad data. Expected "%v", Got "%s"`, expectedVal, val)
		}
		return nil
	})
}

func TestAccountFetcher(t *testing.T) {
	fetcher, err := NewFileFetcher("./test")
	assert.NoError(t, err, "Failed to create test fetcher")

	account, errs := fetcher.FetchAccount(context.Background(), json.RawMessage(`{"events_enabled":true}`), "valid")
	assertErrorCount(t, 0, errs)
	assert.JSONEq(t, `{"disabled":false, "events_enabled":true, "id":"valid" }`, string(account))

	account, errs = fetcher.FetchAccount(context.Background(), nil, "valid")
	assertErrorCount(t, 0, errs)
	assert.JSONEq(t, `{"disabled":false, "id":"valid" }`, string(account))

	_, errs = fetcher.FetchAccount(context.Background(), json.RawMessage(`{"events_enabled":true}`), "nonexistent")
	assertErrorCount(t, 1, errs)
	assert.Error(t, errs[0])
	assert.Equal(t, stored_requests.NotFoundError{ID: "nonexistent", DataType: "Account"}, errs[0])

	_, errs = fetcher.FetchAccount(context.Background(), json.RawMessage(`{"events_enabled"}`), "valid")
	assertErrorCount(t, 1, errs)
	assert.Error(t, errs[0])
	assert.Equal(t, fmt.Errorf("Invalid JSON Document"), errs[0])

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
	if err := jsonutil.UnmarshalValid(value, &req1Val); err != nil {
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
	if err := jsonutil.UnmarshalValid(value, &req2Val); err != nil {
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
	if err := jsonutil.UnmarshalValid(value, &impVal); err != nil {
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

func newCategoryFetcher(directory string) (stored_requests.CategoryFetcher, error) {
	fetcher, err := NewFileFetcher(directory)
	if err != nil {
		return nil, err
	}
	catfetcher, ok := fetcher.(stored_requests.CategoryFetcher)
	if !ok {
		return nil, fmt.Errorf("Failed to type cast fetcher to CategoryFetcher")
	}
	return catfetcher, nil
}

func TestCategoriesFetcherWithPublisher(t *testing.T) {
	fetcher, err := newCategoryFetcher("./test/category-mapping")
	if err != nil {
		t.Errorf("Failed to create a category Fetcher: %v", err)
	}
	category, err := fetcher.FetchCategories(context.TODO(), "test", "categories", "IAB1-1")
	assert.Equal(t, nil, err, "Categories were loaded incorrectly")
	assert.Equal(t, "Beverages", category, "Categories were loaded incorrectly")
}

func TestCategoriesFetcherWithoutPublisher(t *testing.T) {
	fetcher, err := newCategoryFetcher("./test/category-mapping")
	if err != nil {
		t.Errorf("Failed to create a category Fetcher: %v", err)
	}
	category, err := fetcher.FetchCategories(context.TODO(), "test", "", "IAB1-1")
	assert.Equal(t, nil, err, "Categories were loaded incorrectly")
	assert.Equal(t, "VideoGames", category, "Categories were loaded incorrectly")
}

func TestCategoriesFetcherNoCategory(t *testing.T) {
	fetcher, err := newCategoryFetcher("./test/category-mapping")
	if err != nil {
		t.Errorf("Failed to create a category Fetcher: %v", err)
	}
	_, fetchingErr := fetcher.FetchCategories(context.TODO(), "test", "", "IAB1-100")
	assert.Equal(t, fmt.Errorf("Unable to find category for adserver 'test', publisherId: '', iab category: 'IAB1-100'"),
		fetchingErr, "Categories were loaded incorrectly")
}

func TestCategoriesFetcherBrokenJson(t *testing.T) {
	fetcher, err := newCategoryFetcher("./test/category-mapping")
	if err != nil {
		t.Errorf("Failed to create a category Fetcher: %v", err)
	}
	_, fetchingErr := fetcher.FetchCategories(context.TODO(), "test", "broken", "IAB1-100")
	assert.Equal(t, fmt.Errorf("Unable to unmarshal categories for adserver: 'test', publisherId: 'broken'"),
		fetchingErr, "Categories were loaded incorrectly")
}

func TestCategoriesFetcherNoCategoriesFile(t *testing.T) {
	fetcher, err := newCategoryFetcher("./test/category-mapping")
	if err != nil {
		t.Errorf("Failed to create a category Fetcher: %v", err)
	}
	_, fetchingErr := fetcher.FetchCategories(context.TODO(), "test", "not_exists", "IAB1-100")
	assert.Equal(t, fmt.Errorf("Unable to find mapping file for adserver: 'test', publisherId: 'not_exists'"),
		fetchingErr, "Categories were loaded incorrectly")
}

// validateStoredResponse - reusable function in the stored response test to verify the actual data read from the fetcher
func validateStoredResponse[T any](t *testing.T, storedInfo map[string]json.RawMessage, id string, verifyFunc func(outputVal T) error) {
	storedValue, hasID := storedInfo[id]
	if !hasID {
		t.Fatalf(`Expected stored response data to have id: "%s"`, id)
	}

	var unmarshalledValue T
	if err := jsonutil.UnmarshalValid(storedValue, &unmarshalledValue); err != nil {
		t.Errorf(`Failed to unmarshal stored response data of id "%s": %v`, id, err)
	}

	if err := verifyFunc(unmarshalledValue); err != nil {
		t.Errorf(`Bad data in stored response of id: "%s": %v`, id, err)
	}
}
