package file_fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/prebid/prebid-server/stored_requests"
	"io/ioutil"
	"strings"
)

// NewFileFetcher _immediately_ loads stored request data from local files.
// These are stored in memory for low-latency reads.
//
// This expects each file in the directory to be named "{config_id}.json".
// For example, when asked to fetch the request with ID == "23", it will return the data from "directory/23.json".
func NewFileFetcher(directory string) (stored_requests.Fetcher, error) {
	fileInfos, err := ioutil.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	storedReqs := make(map[string]json.RawMessage, len(fileInfos))
	for _, fileInfo := range fileInfos {
		fileData, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", directory, fileInfo.Name()))
		if err != nil {
			return nil, err
		}
		if strings.HasSuffix(fileInfo.Name(), ".json") { // Skip the .gitignore
			storedReqs[strings.TrimSuffix(fileInfo.Name(), ".json")] = json.RawMessage(fileData)
		}
	}
	return &eagerFetcher{storedReqs}, nil
}

type eagerFetcher struct {
	storedReqs map[string]json.RawMessage
}

func (fetcher *eagerFetcher) FetchRequests(ctx context.Context, ids []string) (map[string]json.RawMessage, []error) {
	var errors []error = nil
	for _, id := range ids {
		if _, ok := fetcher.storedReqs[id]; !ok {
			errors = append(errors, fmt.Errorf("No config found for id: %s", id))
		}
	}

	// Even though there may be many other IDs here, the interface contract doesn't prohibit this.
	// Returning the whole slice is much cheaper than making partial copies on each call.
	return fetcher.storedReqs, errors
}
