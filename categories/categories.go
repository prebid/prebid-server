package categories

import (
	"encoding/json"
	"fmt"
	"github.com/prebid/prebid-server/stored_requests"
)

type Categories struct {
	categories map[string]map[string]map[string]string
}

func NewCategories(categoriesFetcher stored_requests.Fetcher) (categories Categories, err error) {
	categoriesdData := make(map[string]map[string]map[string]string)
	for k, v := range categoriesFetcher.FetchCategories() {

		cat := make(map[string]map[string]string)
		for key, value := range v {
			tmp := make(map[string]string)

			if err := json.Unmarshal(value, &tmp); err != nil {
				return Categories{
					categories: nil,
				}, fmt.Errorf("Unable to unmarshal category for: '%s'", key)
			}
			cat[key] = tmp
		}
		categoriesdData[k] = cat
	}
	return Categories{
		categories: categoriesdData,
	}, nil
}

func (c *Categories) GetCategory(primaryAdServer string, publisherId string, iabCategory string) (string, error) {

	if primaryAdServerMapping, present := c.categories[primaryAdServer]; present == true {
		if len(publisherId) > 0 {
			if publisherMapping, present1 := primaryAdServerMapping[primaryAdServer+"_"+publisherId]; present1 == true {
				return publisherMapping[iabCategory], nil
			}

		} else {
			return primaryAdServerMapping[primaryAdServer][iabCategory], nil
		}
	}

	return "", fmt.Errorf("Category '%s' not found for server: '%s', publisherId: '%s'",
		iabCategory, primaryAdServer, publisherId)
}
