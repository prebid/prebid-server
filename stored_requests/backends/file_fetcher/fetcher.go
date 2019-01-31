package file_fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/prebid/prebid-server/stored_requests"
)

// NewFileFetcher _immediately_ loads stored request data from local files.
// These are stored in memory for low-latency reads.
//
// This expects each file in the directory to be named "{config_id}.json".
// For example, when asked to fetch the request with ID == "23", it will return the data from "directory/23.json".
func NewFileFetcher(directory string) (stored_requests.Fetcher, error) {
	storedData := make(map[string]map[string]json.RawMessage)
	fileInfos, err := ioutil.ReadDir(directory)
	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() {
			fmt.Print(fileInfo.Name())
			data, err := collectStoredData(directory + "/" + fileInfo.Name())
			storedData[fileInfo.Name()] = data
			if err != nil {
				return nil, err
			}
		}
	}
	return &eagerFetcher{storedData}, err
}

type eagerFetcher struct {
	storedData map[string]map[string]json.RawMessage
}

func (fetcher *eagerFetcher) FetchRequests(ctx context.Context, requestIDs []string, impIDs []string) (map[string]json.RawMessage, map[string]json.RawMessage, []error) {
	errs := appendErrors("Request", requestIDs, fetcher.storedData["stored_requests"], nil)
	errs = appendErrors("Imp", impIDs, fetcher.storedData["stored_imps"], errs)
	return fetcher.storedData["stored_requests"], fetcher.storedData["stored_imps"], errs
}

func (fetcher *eagerFetcher) FetchCategories() map[string]map[string]json.RawMessage {
	return fetcher.storedData
}

func collectStoredData(directory string) (map[string]json.RawMessage, error) {
	fileInfos, err := ioutil.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	data := make(map[string]json.RawMessage, len(fileInfos))
	for _, fileInfo := range fileInfos {
		fileData, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", directory, fileInfo.Name()))
		if err != nil {
			return nil, err
		}
		if strings.HasSuffix(fileInfo.Name(), ".json") { // Skip the .gitignore
			data[strings.TrimSuffix(fileInfo.Name(), ".json")] = json.RawMessage(fileData)
		}
	}
	return data, nil
}

func appendErrors(dataType string, ids []string, data map[string]json.RawMessage, errs []error) []error {
	for _, id := range ids {
		if _, ok := data[id]; !ok {
			errs = append(errs, stored_requests.NotFoundError{
				ID:       id,
				DataType: dataType,
			})
		}
	}
	return errs
}
