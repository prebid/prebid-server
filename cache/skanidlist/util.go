package skanidlist

import "github.com/prebid/prebid-server/cache/skanidlist/model"

func extract(skanIDList model.SKANIDList) map[string]bool {
	skanIDs := map[string]bool{}

	for _, skanID := range skanIDList.SKAdNetworkIDs {
		skanIDs[skanID.SKAdNetworkID] = true
	}

	return skanIDs
}
