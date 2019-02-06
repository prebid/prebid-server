package categories_test

import (
	"github.com/prebid/prebid-server/categories"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCategoriesCorrectJson(t *testing.T) {
	categoriesData := setUpCategoryData()

	assert.Equal(t, "value1", categoriesData.Categories["appnexus"]["appnexus_disney"]["cat1"], "Categories don't match for appnexus")
	assert.Equal(t, "value4", categoriesData.Categories["freewheel"]["freewheel_espn"]["cat4"], "Categories don't match for freewheel")
}

func TestGetCategoryWithPublisherId(t *testing.T) {
	categoriesData := setUpCategoryData()

	cat, _ := categoriesData.GetCategory("appnexus", "disney", "cat1")
	_, err := categoriesData.GetCategory("freewheel", "espn1", "cat1")

	assert.Equal(t, "value1", cat, "Category with publisherId doesn't match")
	assert.NotEmpty(t, err, "Error shoild not be empty")
}

func TestGetCategoryWithoutPublisherId(t *testing.T) {
	categoriesData := setUpCategoryData()

	cat, _ := categoriesData.GetCategory("nopublisher", "", "cat5")

	assert.Equal(t, "value5", cat, "Category with publisherId doesn't match")
}

func TestGetCategoryWithoutPrimaryAdServer(t *testing.T) {

	categoriesData := setUpCategoryData()

	_, err := categoriesData.GetCategory("", "disney", "cat1")

	assert.Equal(t, false, err == nil, "Category cannot be returned without primary ad server")
}

func setUpCategoryData() (cat categories.Categories) {

	catData := make(map[string]map[string]map[string]string)
	testAPNCatData := make(map[string]map[string]string)
	testFWCatData := make(map[string]map[string]string)
	testNoPublissherCatData := make(map[string]map[string]string)

	appnDisney := make(map[string]string)
	appnDisney["cat1"] = "value1"
	appnDisney["cat2"] = "value2"

	fwEspn := make(map[string]string)
	fwEspn["cat3"] = "value3"
	fwEspn["cat4"] = "value4"

	noPub := make(map[string]string)
	noPub["cat5"] = "value5"
	noPub["cat6"] = "value6"

	testAPNCatData["appnexus_disney"] = appnDisney
	testFWCatData["freewheel_espn"] = fwEspn
	testNoPublissherCatData["nopublisher"] = noPub

	catData["appnexus"] = testAPNCatData
	catData["freewheel"] = testFWCatData
	catData["nopublisher"] = testNoPublissherCatData

	return categories.Categories{Categories: catData}
}
