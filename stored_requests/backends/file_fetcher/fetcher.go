package file_fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"strings"

	"github.com/prebid/prebid-server/stored_requests"
)

// NewFileFetcher lazy-loads various kinds of objects from the given directory
// Expected directory structure depends on data type - stored requests or categories,
// and may include:
// - stored_requests/{id}.json for stored requests
// - stored_imps/{id}.json for stored imps
// - {adserver}.json non-publisher specific primary adserver categories
// - {adserver}/{adserver}_{account_id}.json publisher specific categories for primary adserver
func NewFileFetcher(directory string) (stored_requests.AllFetcher, error) {
	if _, err := ioutil.ReadDir(directory); err != nil {
		return &fileFetcher{}, err
	}
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

func fetchAllObjects(dir string, glob string, dataType string) (jsons map[string]json.RawMessage, errs []error) {
	var ids, files []string
	var err error
	path := filepath.Join(dir, glob)
	if files, err = filepath.Glob(path); err != nil {
		return nil, append(errs, fmt.Errorf("Error scanning for %s in %s: %v", dataType, path, err))
	}
	for _, f := range files {
		fn := f[len(dir)+1:] // remove "<dir>/"
		id := strings.TrimSuffix(fn, filepath.Ext(fn))
		ids = append(ids, id)
	}
	return fetchObjects(dir, dataType, ids)
}

// FetchRequests fetches the stored requests for the given IDs.
func (fetcher *fileFetcher) FetchRequests(ctx context.Context, requestIDs []string, impIDs []string) (requests map[string]json.RawMessage, imps map[string]json.RawMessage, errs []error) {
	requests, reqErrs := fetchObjects(fetcher.StoredRequestsDir, "Request", requestIDs)
	imps, impErrs := fetchObjects(fetcher.StoredImpsDir, "Imp", impIDs)
	return requests, imps, append(reqErrs, impErrs...)
}

// FetchAllRequests returns comprehensive maps containing all the requests and imps in filesystem
func (fetcher *fileFetcher) FetchAllRequests(ctx context.Context) (requests map[string]json.RawMessage, imps map[string]json.RawMessage, errs []error) {
	requests, errs = fetchAllObjects(fetcher.StoredRequestsDir, "*.json", "Request")
	imps, impErrs := fetchAllObjects(fetcher.StoredImpsDir, "*.json", "Imp")
	return requests, imps, append(errs, impErrs...)
}

// FetchAllCategories loads and stores all the category mappings defined in the filesystem
func (fetcher *fileFetcher) FetchAllCategories(ctx context.Context) (categories map[string]json.RawMessage, errs []error) {
	categories, errs = fetchAllObjects(fetcher.CategoriesDir, "*/*.json", "Category")
	for name, mapping := range categories {
		data := make(map[string]stored_requests.Category)
		if err := json.Unmarshal(mapping, &data); err != nil {
			errs = append(errs, fmt.Errorf(`Unable to unmarshal categories from "%s/%s.json"`, fetcher.CategoriesDir, name))
			delete(categories, name)
		} else {
			fetcher.Categories[name] = data
		}
	}
	return categories, errs
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
