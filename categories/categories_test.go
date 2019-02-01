package categories_test

import (
	"context"
	"encoding/json"
	"github.com/prebid/prebid-server/categories"
	"github.com/stretchr/testify/assert"
	"testing"
)

type MockCategoriesFetcher struct {
	Categories map[string]map[string]json.RawMessage
}

func (f *MockCategoriesFetcher) FetchCategories() (categories map[string]map[string]json.RawMessage) {
	return f.Categories
}

func (f *MockCategoriesFetcher) FetchRequests(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage, errs []error) {
	return nil, nil, nil
}

func TestCategoriesWithCorrectJson(t *testing.T) {
	catData := make(map[string]map[string]json.RawMessage)
	testAPNCatData := make(map[string]json.RawMessage)
	testFWCatData := make(map[string]json.RawMessage)

	testAPNCatData["appnexus"] = []byte(`{"cat1":"value1", "cat2":"value2"}`)
	testFWCatData["freewheel"] = []byte(`{"cat3":"value3", "cat4":"value4"}`)

	catData["appnexus"] = testAPNCatData
	catData["freewheel"] = testFWCatData
	fileFetcher := MockCategoriesFetcher{catData}
	categories, _ := categories.NewCategories(&fileFetcher)

	assert.Equal(t, "value1", categories.Categories["appnexus"]["appnexus"]["cat1"], "Categories don't match for appnexus")
	assert.Equal(t, "value4", categories.Categories["freewheel"]["freewheel"]["cat4"], "Categories don't match for freewheel")
}

func TestCategoriesWithInvalidJson(t *testing.T) {
	catData := make(map[string]map[string]json.RawMessage)
	testAPNCatData := make(map[string]json.RawMessage)
	testFWCatData := make(map[string]json.RawMessage)

	testAPNCatData["appnexus"] = []byte(`{"cat1":"value1", "cat2":"value2"}`)
	testFWCatData["freewheel"] = []byte(`{invalid_json_text]}`)

	catData["appnexus"] = testAPNCatData
	catData["freewheel"] = testFWCatData
	fileFetcher := MockCategoriesFetcher{catData}
	categories, _ := categories.NewCategories(&fileFetcher)

	assert.Equal(t, "value1", categories.Categories["appnexus"]["appnexus"]["cat1"], "Categories don't match for appnexus")
	assert.Equal(t, 0, len(categories.Categories["freewheel"]), "Categories don't match. Should be empty")
}

func TestGetCategoryWithPuiblisherId(t *testing.T) {
	catData := make(map[string]map[string]json.RawMessage)
	testAPNCatData := make(map[string]json.RawMessage)
	testFWCatData := make(map[string]json.RawMessage)

	testAPNCatData["appnexus_disney"] = []byte(`{"cat1":"value1", "cat2":"value2"}`)
	testFWCatData["freewheel_espn"] = []byte(`{"cat3":"value3", "cat4":"value4"}`)

	catData["appnexus"] = testAPNCatData
	catData["freewheel"] = testFWCatData
	fileFetcher := MockCategoriesFetcher{catData}
	categories, _ := categories.NewCategories(&fileFetcher)

	cat, _ := categories.GetCategory("appnexus", "disney", "cat1")
	_, err := categories.GetCategory("freewheel", "espn1", "cat1")

	assert.Equal(t, "value1", cat, "Category with publisherId doesn't match")
	assert.NotEmpty(t, err, "Error shoild not be empty")
}

func TestGetCategoryWithoutPuiblisherId(t *testing.T) {
	catData := make(map[string]map[string]json.RawMessage)
	testAPNCatData := make(map[string]json.RawMessage)
	testFWCatData := make(map[string]json.RawMessage)

	testAPNCatData["appnexus"] = []byte(`{"cat1":"value1", "cat2":"value2"}`)
	testFWCatData["freewheel"] = []byte(`{"cat3":"value3", "cat4":"value4"}`)

	catData["appnexus"] = testAPNCatData
	catData["freewheel"] = testFWCatData
	fileFetcher := MockCategoriesFetcher{catData}
	categories, _ := categories.NewCategories(&fileFetcher)

	cat, _ := categories.GetCategory("appnexus", "", "cat1")

	assert.Equal(t, "value1", cat, "Category with publisherId doesn't match")
}
