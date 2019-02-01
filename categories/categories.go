package categories

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/prebid/prebid-server/stored_requests"
)

type Categories struct {
	Categories map[string]map[string]map[string]string
}

func NewCategories(categoriesFetcher stored_requests.Fetcher) (categories Categories, err error) {
	categoriesdData := make(map[string]map[string]map[string]string)
	for k, v := range categoriesFetcher.FetchCategories() {

		cat := make(map[string]map[string]string)
		for key, value := range v {
			tmp := make(map[string]string)

			if err := json.Unmarshal(value, &tmp); err != nil {
				glog.Warning("Unable to unmarshal category for: ", key)
				continue
			}
			cat[key] = tmp
		}
		categoriesdData[k] = cat
	}
	return Categories{
		Categories: categoriesdData,
	}, nil
}

func (c *Categories) GetCategory(primaryAdServer, publisherId, iabCategory string) (string, error) {

	if primaryAdServerMapping, primaryMappingPresent := c.Categories[primaryAdServer]; primaryMappingPresent == true {
		if len(publisherId) > 0 {
			if publisherMapping, publisherMappingpresent := primaryAdServerMapping[primaryAdServer+"_"+publisherId]; publisherMappingpresent == true {
				return publisherMapping[iabCategory], nil
			}
		} else {
			return primaryAdServerMapping[primaryAdServer][iabCategory], nil
		}
	}
	return "", fmt.Errorf("Category '%s' not found for server: '%s', publisherId: '%s'",
		iabCategory, primaryAdServer, publisherId)
}
