package file_fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/stored_requests"
)

// NewFileFetcher lazy-loads various kinds of objects from the given directory
// Expected directory structure:
// - stored_requests/{id}.json for stored requests
// - stored_imps/{id}.json for stored imps
// - {adserver}.json non-publisher specific primary adserver categories
// - {adserver}/{adserver}_{account_id}.json publisher specific categories for primary adserver
func NewFileFetcher(directory string) (stored_requests.AllFetcher, error) {
	_, err := ioutil.ReadDir(directory)
	if err != nil {
		return &fileFetcher{}, err
	}
	// read - but don't store - all the files to warm os cache
	go filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err == nil && filepath.Ext(path) == ".json" {
			if _, err := ioutil.ReadFile(path); err != nil {
				glog.Warningf("Error reading %s: %v", path, err)
			}
		}
		return nil
	})
	return &fileFetcher{
		StoredRequestsDir: path.Join(directory, "stored_requests"),
		StoredImpsDir:     path.Join(directory, "stored_imps"),
		AccountsDir:       path.Join(directory, "accounts"),
		CategoriesDir:     path.Clean(directory),
		Categories:        make(map[string]map[string]stored_requests.Category),
	}, nil
}

type fileFetcher struct {
	StoredRequestsDir string
	StoredImpsDir     string
	AccountsDir       string
	CategoriesDir     string
	Categories        map[string]map[string]stored_requests.Category
}

func readJSONFile(dir string, key string) (json.RawMessage, error) {
	f := path.Join(dir, key) + ".json"
	if fileData, err := ioutil.ReadFile(f); err != nil {
		return nil, err
	} else {
		return json.RawMessage(fileData), nil
	}
}

func fetchObjects(dir string, dataType string, ids []string) (jsons map[string]json.RawMessage, errs []error) {
	jsons = make(map[string]json.RawMessage)
	for _, id := range ids {
		if data, err := readJSONFile(dir, id); err == nil {
			jsons[id] = data
		} else {
			errs = append(errs, stored_requests.NotFoundError{
				ID:       id,
				DataType: dataType,
			})
		}
	}
	return jsons, errs
}

// FetchRequests fetches the stored requests for the given IDs.
func (fetcher *fileFetcher) FetchRequests(ctx context.Context, requestIDs []string, impIDs []string) (requests map[string]json.RawMessage, imps map[string]json.RawMessage, errs []error) {
	requests, reqErrs := fetchObjects(fetcher.StoredRequestsDir, "Request", requestIDs)
	imps, impErrs := fetchObjects(fetcher.StoredImpsDir, "Imp", impIDs)
	return requests, imps, append(reqErrs, impErrs...)
}

// FetchCategories fetches the ad-server/publisher specific category for the given IAB category
func (fetcher *fileFetcher) FetchCategories(ctx context.Context, primaryAdServer, publisherId, iabCategory string) (string, error) {
	var fileName string
	if len(publisherId) != 0 {
		fileName = path.Join(primaryAdServer, fmt.Sprintf("%s_%s", primaryAdServer, publisherId))
	} else {
		fileName = path.Join(primaryAdServer, primaryAdServer)
	}

	data := fetcher.Categories[fileName]
	if data == nil {
		if file, err := readJSONFile(fetcher.CategoriesDir, fileName); err == nil {
			data = make(map[string]stored_requests.Category)

			if err := json.Unmarshal(file, &data); err != nil {
				return "", fmt.Errorf("Unable to unmarshal categories for adserver: '%s', publisherId: '%s'", primaryAdServer, publisherId)
			}
			fetcher.Categories[fileName] = data
		} else {
			return "", fmt.Errorf("Unable to find mapping file for adserver: '%s', publisherId: '%s'", primaryAdServer, publisherId)
		}
	}

	if resultCategory, ok := data[iabCategory]; ok {
		return resultCategory.Id, nil
	}
	return "", fmt.Errorf("Unable to find category for adserver '%s', publisherId: '%s', iab category: '%s'", primaryAdServer, publisherId, iabCategory)
}
