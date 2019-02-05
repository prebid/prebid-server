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
	categoriesData, catErr := setUpCategoryData()
	if catErr != nil {
		assert.Fail(t, "Categories creation error")
	}
	assert.Equal(t, "value1", categoriesData.Categories["appnexus"]["appnexus_disney"]["cat1"], "Categories don't match for appnexus")
	assert.Equal(t, "value4", categoriesData.Categories["freewheel"]["freewheel_espn"]["cat4"], "Categories don't match for freewheel")
}

func TestCategoriesWithInvalidJson(t *testing.T) {
	categoriesData, catErr := setUpCategoryData()
	if catErr != nil {
		assert.Fail(t, "Categories creation error")
	}

	assert.Equal(t, "value1", categoriesData.Categories["appnexus"]["appnexus_disney"]["cat1"], "Categories don't match for appnexus")
	assert.Equal(t, 0, len(categoriesData.Categories["broken"]), "Categories don't match. Invalid json should be skipped")
}

func TestGetCategoryWithPublisherId(t *testing.T) {
	categoriesData, catErr := setUpCategoryData()
	if catErr != nil {
		assert.Fail(t, "Categories creation error")
	}

	cat, _ := categoriesData.GetCategory("appnexus", "disney", "cat1")
	_, err := categoriesData.GetCategory("freewheel", "espn1", "cat1")

	assert.Equal(t, "value1", cat, "Category with publisherId doesn't match")
	assert.NotEmpty(t, err, "Error shoild not be empty")
}

func TestGetCategoryWithoutPublisherId(t *testing.T) {
	categoriesData, catErr := setUpCategoryData()
	if catErr != nil {
		assert.Fail(t, "Categories creation error")
	}
	cat, _ := categoriesData.GetCategory("nopublisher", "", "cat5")

	assert.Equal(t, "value5", cat, "Category with publisherId doesn't match")
}

func TestGetCategoryWithoutPrimaryAdServer(t *testing.T) {

	categoriesData, catErr := setUpCategoryData()
	if catErr != nil {
		assert.Fail(t, "Categories creation error")
	}

	_, err := categoriesData.GetCategory("", "disney", "cat1")

	assert.Equal(t, false, err == nil, "Category cannot be returned without primary ad server")
}

func setUpCategoryData() (cat categories.Categories, err error) {

	catData := make(map[string]map[string]json.RawMessage)
	testAPNCatData := make(map[string]json.RawMessage)
	testFWCatData := make(map[string]json.RawMessage)
	testNoPublissherCatData := make(map[string]json.RawMessage)
	testBrokenCatData := make(map[string]json.RawMessage)

	testAPNCatData["appnexus_disney"] = []byte(`{"cat1":"value1", "cat2":"value2"}`)
	testFWCatData["freewheel_espn"] = []byte(`{"cat3":"value3", "cat4":"value4"}`)
	testNoPublissherCatData["nopublisher"] = []byte(`{"cat5":"value5", "cat6":"value6"}`)
	testBrokenCatData["broken"] = []byte(`{invalid_json_text]}`)

	catData["appnexus"] = testAPNCatData
	catData["freewheel"] = testFWCatData
	catData["nopublisher"] = testNoPublissherCatData
	catData["broken"] = testBrokenCatData

	fileFetcher := MockCategoriesFetcher{catData}
	return categories.NewCategories(&fileFetcher)
}
