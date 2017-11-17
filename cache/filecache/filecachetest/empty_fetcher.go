package filecachetest

import (
	"encoding/json"
	"fmt"
	"github.com/prebid/prebid-server/cache"
)

// EmptyFetcher is a nil-object which doesn't support any server-side bid requests.
func EmptyFetcher() cache.ConfigFetcher {
	return instance
}

type emptyFetcher struct {}

func (fetcher *emptyFetcher) GetConfigs(ids []string) (map[string]json.RawMessage, []error) {
	errs := make([]error, 0, len(ids))
	for _, id := range ids {
		errs = append(errs, fmt.Errorf("Attempted request with server-side data id=%s, but none is configured.", id))
	}
	return nil, errs
}

var instance = emptyFetcher{}
