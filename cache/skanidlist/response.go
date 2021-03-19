package skanidlist

import "github.com/prebid/prebid-server/cache/skanidlist/model"

type response struct {
	skanIDList model.SKANIDList
	updated    bool
	err        error
}
