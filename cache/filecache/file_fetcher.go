package filecache

import (
	"github.com/prebid/prebid-server/cache"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
)

// NewEagerConfigFetcher gets a ConfigFetcher which reads configs from local files.
// As the name suggests, it loads all the files in the directory _immediately_, and keeps
// them stored in memory for low-latency reads.
//
// This expects each file in the directory to be named "{config_id}.json".
// For example, a file at "directory/23.json" will store the data for the config with ID == "23".
func NewEagerConfigFetcher(directory string) (cache.ConfigFetcher, error) {
	fileInfos, err := ioutil.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	configs := make(map[string]json.RawMessage, len(fileInfos))
	for _, fileInfo := range fileInfos {
		fileData, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", directory, fileInfo.Name()))
		if err != nil {
			return nil, err
		}
		if strings.HasSuffix(fileInfo.Name(), ".json") { // Skip the .gitignore
			configs[strings.TrimSuffix(fileInfo.Name(), ".json")] = json.RawMessage(fileData)
		}
	}
	return &eagerFetcher{configs}, nil
}

type eagerFetcher struct {
	configs map[string]json.RawMessage
}

func (fetcher *eagerFetcher) GetConfigs(ids []string) (map[string]json.RawMessage, []error) {
	var errors []error = nil
	for _, id := range ids {
		if _, ok := fetcher.configs[id]; !ok {
			errors = append(errors, fmt.Errorf("No config found for id: %s", id))
		}
	}

	// Even though there may be many other IDs here, this still technically obeys
	// the interface contract. No need to do a partial copy
	return fetcher.configs, errors
}
