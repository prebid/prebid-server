package file_fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/prebid/prebid-server/stored_requests"
	"github.com/stretchr/testify/assert"
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

	storedReqs, storedImps, errs = fetcher.(stored_requests.Fetcher).FetchAllRequests(context.Background())
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

func newCategoryFetcher(t *testing.T, directory string) stored_requests.CategoryFetcher {
	fetcher, err := NewFileFetcher(directory)
	if err != nil {
		t.Errorf("Failed to create a category Fetcher: %v", err)
	}
	catfetcher, ok := fetcher.(stored_requests.CategoryFetcher)
	if !ok {
		t.Errorf("Failed to type cast fetcher to CategoryFetcher")
	}
	return catfetcher
}

func TestCategoriesFetcherAllCategories(t *testing.T) {
	fetcher := newCategoryFetcher(t, "./test/category-mapping")
	categories, errs := fetcher.FetchAllCategories(nil)
	assertErrorCount(t, 1, errs)
	assert.EqualError(t, errs[0], `Unable to unmarshal categories from "test/category-mapping/test/test_broken.json"`)
	assert.Equalf(t, len(categories), 2, "Expected 2 categories preloaded, got %d", len(categories))
	assert.Containsf(t, categories, "test/test", "test/test missing from preloaded data set")
	assert.Containsf(t, categories, "test/test_categories", "test/test_categories missing from preloaded data set")
	cat, err := fetcher.FetchCategories(nil, "test", "categories", "IAB1-1")
	assert.Nil(t, err, "Unexpected error translating category IAB1-1")
	assert.Equal(t, "Beverages", cat, "test/test_categories missing expected category IAB1-1")
	cat, err = fetcher.FetchCategories(nil, "test", "", "IAB1-1")
	assert.Nil(t, err, "Unexpected error translating category IAB1-1")
	assert.Equal(t, "VideoGames", cat, "test/test missing expected category IAB1-1")
}

func TestCategoriesFetcherWithPublisher(t *testing.T) {
	fetcher := newCategoryFetcher(t, "./test/category-mapping")
	category, err := fetcher.FetchCategories(nil, "test", "categories", "IAB1-1")
	assert.Equal(t, nil, err, "Categories were loaded incorrectly")
	assert.Equal(t, "Beverages", category, "Categories were loaded incorrectly")
}

func TestCategoriesFetcherWithoutPublisher(t *testing.T) {
	fetcher := newCategoryFetcher(t, "./test/category-mapping")
	category, err := fetcher.FetchCategories(nil, "test", "", "IAB1-1")
	assert.Equal(t, nil, err, "Categories were loaded incorrectly")
	assert.Equal(t, "VideoGames", category, "Categories were loaded incorrectly")
}

func TestCategoriesFetcherNoCategory(t *testing.T) {
	fetcher := newCategoryFetcher(t, "./test/category-mapping")
	_, fetchingErr := fetcher.FetchCategories(nil, "test", "", "IAB1-100")
	assert.Equal(t, fmt.Errorf("Unable to find category for adserver 'test', publisherId: '', iab category: 'IAB1-100'"),
		fetchingErr, "Categories were loaded incorrectly")
}

func TestCategoriesFetcherBrokenJson(t *testing.T) {
	fetcher := newCategoryFetcher(t, "./test/category-mapping")
	_, fetchingErr := fetcher.FetchCategories(nil, "test", "broken", "IAB1-100")
	assert.Equal(t, fmt.Errorf("Unable to unmarshal categories for adserver: 'test', publisherId: 'broken'"),
		fetchingErr, "Categories were loaded incorrectly")
}

func TestCategoriesFetcherNoCategoriesFile(t *testing.T) {
	fetcher := newCategoryFetcher(t, "./test/category-mapping")
	_, fetchingErr := fetcher.FetchCategories(nil, "test", "not_exists", "IAB1-100")
	assert.Equal(t, fmt.Errorf("Unable to find mapping file for adserver: 'test', publisherId: 'not_exists'"),
		fetchingErr, "Categories were loaded incorrectly")
}
