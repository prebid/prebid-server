package stored_requests

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/buger/jsonparser"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/prebid/prebid-server/metrics"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// ABFetcher is an AllFetcher implementing the A/B experiment selection rules for stored requests only
// The results from this fetcher MUST NOT be cached for the distribution rules to work.
// The composite fetcher layout is ABFetcher -> FetcherWithCache -> Backend Fetcher (Files, HTTP, Db)
// This fetcher SHOULD be in front of a caching fetcher to take advantage of caching on refetch.
type ABFetcher struct {
	fetcher       AllFetcher
	metricsEngine metrics.MetricsEngine
}

// WithABFetcher returns an AllFetcher that may replace requested requests
func WithABFetcher(fetcher AllFetcher, metricsEngine metrics.MetricsEngine) AllFetcher {
	return &ABFetcher{
		fetcher:       fetcher,
		metricsEngine: metricsEngine,
	}
}

// FetchRequests applies A/B experiment rules and refetches alternate stored requests if needed
// triggered by ext.prebid.storedrequest.ab_config existence as a list of ABConfig objects
func (f *ABFetcher) FetchRequests(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage, errs []error) {
	if requestData, impData, errs = f.fetcher.FetchRequests(ctx, requestIDs, impIDs); len(errs) > 0 || len(requestIDs) != 1 || len(requestData) == 0 {
		return
	}
	requestID := requestIDs[0]
	storedRequest := requestData[requestID]

	// Extract ab_config json if present
	abConfigs, dataType, _, err := jsonparser.Get(storedRequest, "ext", openrtb_ext.PrebidExtKey, "storedrequest", "ab_config")
	if dataType == jsonparser.NotExist {
		return
	}
	if err != nil {
		errs = append(errs, fmt.Errorf(`failed to read ext.prebid.storedrequest.ab_config from storedrequest "%s": %v`, requestID, err))
		return
	}
	if dataType != jsonparser.Array {
		errs = append(errs, fmt.Errorf(`bad format for ext.prebid.storedrequest.ab_config in storedrequest "%s": expecting a list`, requestID))
		return
	}

	// Run random selection and pick one ABConfig
	abConfig, err := runABSelection(abConfigs)
	if err != nil {
		errs = append(errs, fmt.Errorf(`error parsing ab_config rules from storedrequest "%s": %v`, requestID, err))
		return
	}
	if abConfig == nil {
		return
	}

	// Build lists of new IDs to fetch for replacements
	newImpIDs := []string{}
	for _, impID := range impIDs {
		if newImpID, replace := abConfig.ImpIDs[impID]; replace {
			newImpIDs = append(newImpIDs, newImpID)
		}
	}
	newReqIDs := []string{}
	if abConfig.RequestID != requestID && len(abConfig.RequestID) > 0 {
		newReqIDs = []string{abConfig.RequestID}
	}

	// Fetch replacement stored requests and imps
	newRequestData, newImpData, newErrs := f.fetcher.FetchRequests(ctx, newReqIDs, newImpIDs)
	if len(newErrs) > 0 {
		errs = append(errs, fmt.Errorf(`errors fetching replacement stored data for ab_config`))
		errs = append(errs, newErrs...)
		return // or not, if we want to power through and have a partial replace
	}

	// Replace the stored data for original IDs with the new replacement values.
	// This can be a little confusing (because the original ids are still present)
	// but avoids rewriting the main request json to replace the ids
	var analyticsInfo json.RawMessage
	if value, found := newRequestData[abConfig.RequestID]; found {
		storedRequest = value
		// patch loaded stored request with original ab_config info to preserve it,
		// and add ab_code for analytics
		analyticsInfo = json.RawMessage(
			fmt.Sprintf(`{"ext":{"%s":{"storedrequest":{"ab_config":%s,"ab_code":"%s"}}}}`,
				openrtb_ext.PrebidExtKey, abConfigs, abConfig.Code))
	} else {
		// patch original stored request with just ab_code, the ab_config is already there
		analyticsInfo = json.RawMessage(
			fmt.Sprintf(`{"ext":{"%s":{"storedrequest":{"ab_code":"%s"}}}}`,
				openrtb_ext.PrebidExtKey, abConfig.Code))
	}
	if enrichedRequest, err := jsonpatch.MergePatch(storedRequest, analyticsInfo); err == nil {
		requestData[requestID] = enrichedRequest
	} else {
		// applying the replacements will not be useful without analytics info
		errs = append(errs, fmt.Errorf(`cannot patch ext.prebid.storedrequest.ab_config in storedrequest id %s: %s`, abConfig.RequestID, err))
		return
	}
	// remap imps for which a replacement was requested and found
	for impID, newImpID := range abConfig.ImpIDs {
		impData[impID] = newImpData[newImpID]
	}
	return
}

// FetchAccount is just a pass-through to the underlying AllFetcher
func (f *ABFetcher) FetchAccount(ctx context.Context, accountID string) (account json.RawMessage, errs []error) {
	return f.fetcher.FetchAccount(ctx, accountID)
}

// FetchCategories is just a pass-through to the underlying AllFetcher
func (f *ABFetcher) FetchCategories(ctx context.Context, primaryAdServer, publisherId, iabCategory string) (string, error) {
	return f.fetcher.FetchCategories(ctx, primaryAdServer, publisherId, iabCategory)
}

// runABSelection receives distribution rules in the form of a json list of ABConfig objects
// and returns the randomly selected ABConfig (undecoded)
func runABSelection(jsonBytes []byte, keys ...string) (abConfig *ABConfig, err error) {
	var abConfigJSON []byte
	throw := 100 * rand.Float64()
	t := 0.0
	_, e := jsonparser.ArrayEach(jsonBytes, func(value []byte, dataType jsonparser.ValueType, offset int, e error) {
		if err != nil || e != nil || len(abConfigJSON) > 0 {
			return
		}
		if percent, err := jsonparser.GetFloat(value, "ratio"); err == nil {
			t += percent
			if throw < t {
				abConfigJSON = value
			}
		}
	}, keys...)
	if e != nil {
		return nil, e
	}
	if err != nil {
		return nil, err
	}
	if t > 100 {
		return nil, fmt.Errorf(`%v sum of probabilities %.1f greater than 100%%`, keys, t)
	}
	if len(abConfigJSON) == 0 {
		return nil, nil // control set
	}
	return unmarshalABConfig(abConfigJSON)
}

// unmarshalABConfig parses and validates ABConfig. Could also use the slower json.UnMarshal instead.
func unmarshalABConfig(abConfigJSON []byte) (*ABConfig, error) {
	// Parse and validate ABConfig. Could also use the slower json.UnMarshal instead.
	abConfig := ABConfig{}
	var err error
	abConfig.Code, err = jsonparser.GetString(abConfigJSON, "code")
	if err != nil {
		return nil, fmt.Errorf(`parsing "code" in ab_config: %v`, err)
	}
	if len(abConfig.Code) == 0 {
		return nil, fmt.Errorf(`"code" tag is required in all ab_config sections`)
	}
	abConfig.RequestID, _ = jsonparser.GetString(abConfigJSON, "request_id") // request_id can be missing, ignore error
	abConfig.ImpIDs = make(map[string]string)                                // imp_ids can be missing, so ignore error
	_ = jsonparser.ObjectEach(abConfigJSON,
		func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
			if len(value) > 0 && len(key) > 0 {
				abConfig.ImpIDs[string(key)] = string(value)
			}
			return nil
		},
		"imp_ids")
	if len(abConfig.RequestID) == 0 && len(abConfig.ImpIDs) == 0 {
		return nil, fmt.Errorf(`no replacement ids specified in ab_config for code="%s" `, abConfig.Code)
	}
	return &abConfig, nil
}

/* ABConfig defines an element such as in the list below:
[
	{
		"code": "1321312313312312",
		"ratio": 15.0,
		"request_id": "ABCD-0123-4567-111111", ⬅ this is the replacement stored request id
		"imp_ids": {
			"DCBA-3333-4444-012345": "DCBA-3333-4444-111111", ⬅ replacement stored imp id
			"DCBA-4444-5555-666666": "DCBA-4444-9999-222222", ⬅ replacement stored imp id
			...
		}
	},
	...
]
*/
type ABConfig struct {
	// Code is a string identifying this experiment.
	// The control set must not be specified (its ratio inferred and it makes no replacements)
	Code string `json:"code"`
	// Ratio is the percentage of requests that should have this experiment applied (0-100)
	// The sum of Ratio values for all the experiments in the group must be <100
	Ratio float64 `json:"ratio"`
	// RequestID is the replacement stored request id used in this experiment
	RequestID string `json:"request_id"`
	// ImpIDs maps request stored imp ids to replacement stored imp ids
	ImpIDs map[string]string `json:"imp_ids"`
}
