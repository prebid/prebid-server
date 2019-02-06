package categories

import (
	"fmt"
)

type Categories struct {
	Categories map[string]map[string]map[string]string
}

func (c *Categories) GetCategory(primaryAdServer, publisherId, iabCategory string) (string, error) {

	if primaryAdServerMapping, primaryMappingPresent := c.Categories[primaryAdServer]; primaryMappingPresent == true {
		if len(publisherId) > 0 {
			if publisherMapping, publisherMappingPresent := primaryAdServerMapping[primaryAdServer+"_"+publisherId]; publisherMappingPresent == true {
				return publisherMapping[iabCategory], nil
			}
		} else {
			return primaryAdServerMapping[primaryAdServer][iabCategory], nil
		}
	}
	return "", fmt.Errorf("Category '%s' not found for server: '%s', publisherId: '%s'",
		iabCategory, primaryAdServer, publisherId)
}
