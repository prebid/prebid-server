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
func NewFileFetcher(directory string) (stored_requests.AllFetcher, error) {
	storedData, err := collectStoredData(directory, FileSystem{make(map[string]FileSystem), make(map[string]json.RawMessage)}, nil)
	return &eagerFetcher{storedData, nil}, err
}

type eagerFetcher struct {
	FileSystem FileSystem
	Categories map[string]map[string]stored_requests.Category
}

func (fetcher *eagerFetcher) FetchRequests(ctx context.Context, requestIDs []string, impIDs []string) (map[string]json.RawMessage, map[string]json.RawMessage, []error) {
	storedRequests := fetcher.FileSystem.Directories["stored_requests"].Files
	storedImpressions := fetcher.FileSystem.Directories["stored_imps"].Files
	errs := appendErrors("Request", requestIDs, storedRequests, nil)
	errs = appendErrors("Imp", impIDs, storedImpressions, errs)
	return storedRequests, storedImpressions, errs
}

// FetchAccount fetches the host account configuration for a publisher
func (fetcher *eagerFetcher) FetchAccount(ctx context.Context, accountID string) (json.RawMessage, []error) {
	if len(accountID) == 0 {
		return nil, []error{fmt.Errorf("Cannot look up an empty accountID")}
	}
	accountJSON, ok := fetcher.FileSystem.Directories["accounts"].Files[accountID]
	if !ok {
		return nil, []error{stored_requests.NotFoundError{
			ID:       accountID,
			DataType: "Account",
		}}
	}
	return accountJSON, nil
}

func (fetcher *eagerFetcher) FetchCategories(ctx context.Context, primaryAdServer, publisherId, iabCategory string) (string, error) {
	fileName := primaryAdServer

	if len(publisherId) != 0 {
		fileName = primaryAdServer + "_" + publisherId
	}

	if fetcher.Categories == nil {
		fetcher.Categories = make(map[string]map[string]stored_requests.Category)
	}
	if data, ok := fetcher.Categories[fileName]; ok {
		return data[iabCategory].Id, nil
	}

	if primaryAdServerDir, found := fetcher.FileSystem.Directories[primaryAdServer]; found {

		if file, ok := primaryAdServerDir.Files[fileName]; ok {

			tmp := make(map[string]stored_requests.Category)

			if err := json.Unmarshal(file, &tmp); err != nil {
				return "", fmt.Errorf("Unable to unmarshal categories for adserver: '%s', publisherId: '%s'", primaryAdServer, publisherId)
			}
			fetcher.Categories[fileName] = tmp
			resultCategory := tmp[iabCategory].Id
			primaryAdServerDir.Files[fileName] = nil

			if len(resultCategory) == 0 {
				return "", fmt.Errorf("Unable to find category for adserver '%s', publisherId: '%s', iab category: '%s'", primaryAdServer, publisherId, iabCategory)
			}
			return resultCategory, nil
		} else {
			return "", fmt.Errorf("Unable to find mapping file for adserver: '%s', publisherId: '%s'", primaryAdServer, publisherId)

		}

	}

	return "", fmt.Errorf("Category '%s' not found for server: '%s', publisherId: '%s'",
		iabCategory, primaryAdServer, publisherId)

}

type FileSystem struct {
	Directories map[string]FileSystem
	Files       map[string]json.RawMessage
}

func collectStoredData(directory string, fileSystem FileSystem, err error) (FileSystem, error) {
	if err != nil {
		return FileSystem{nil, nil}, err
	}
	fileInfos, err := ioutil.ReadDir(directory)
	if err != nil {
		return FileSystem{nil, nil}, err
	}
	data := make(map[string]json.RawMessage)

	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() {

			fs := FileSystem{make(map[string]FileSystem), make(map[string]json.RawMessage)}
			fileSys, innerErr := collectStoredData(directory+"/"+fileInfo.Name(), fs, err)
			if innerErr != nil {
				return FileSystem{nil, nil}, innerErr
			}
			fileSystem.Directories[fileInfo.Name()] = fileSys

		} else {
			if strings.HasSuffix(fileInfo.Name(), ".json") { // Skip the .gitignore
				fileData, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", directory, fileInfo.Name()))
				if err != nil {
					return FileSystem{nil, nil}, err
				}
				data[strings.TrimSuffix(fileInfo.Name(), ".json")] = json.RawMessage(fileData)

			}
		}

	}
	fileSystem.Files = data
	return fileSystem, err
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
