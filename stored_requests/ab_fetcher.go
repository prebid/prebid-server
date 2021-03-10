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

// ABFetcher is a request-aware AllFetcher implementing the A/B experiment selection rules
// The results from this fetcher MUST NOT be cached for the distribution rules to work.
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

// FetchRequests applies A/B experiment rules and refetches alternate stored requests if
// triggered by ext.prebid.storedrequest.ab_config existence as a map of testKey: percent
func (f *ABFetcher) FetchRequests(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage, errs []error) {
	if requestData, impData, errs = f.fetcher.FetchRequests(ctx, requestIDs, impIDs); len(errs) > 0 || len(requestIDs) != 1 || len(requestData) == 0 {
		return
	}
	requestID := requestIDs[0]
	storedRequest := requestData[requestID]
	value, dataType, _, err := jsonparser.Get(storedRequest, "ext", openrtb_ext.PrebidExtKey, "storedrequest", "ab_config")
	if dataType == jsonparser.NotExist {
		return
	}
	if err != nil {
		return
	}
	if dataType != jsonparser.Object {
		errs = append(errs, fmt.Errorf("ext.prebid.storedrequest.ab_config should be a map in storedrequest id=%s", requestIDs[0]))
		return
	}
	newRequestID, err := runABSelection(value)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to parse ext.prebid.storedrequest.ab_config in storedrequest id=%s: %v", requestIDs[0], err))
		return
	}
	if newRequestID == "" || newRequestID == requestIDs[0] {
		// main branch selected, nothing to do (or we could patch "ab_selected" ?)
		return
	}
	// load and replace selected stored request
	newRequestData, _, newErrs := f.fetcher.FetchRequests(ctx, []string{newRequestID}, []string{})
	if len(newErrs) > 0 || newRequestData[newRequestID] == nil {
		errs = append(errs, newErrs...)
		return
	}
	// patch loaded stored request with ab_config info, for analytics
	analyticsInfo := json.RawMessage(
		fmt.Sprintf(`{"ext":{"%s":{"storedrequest":{"ab_config":%s,"ab_selected":"%s"}}}}`,
			openrtb_ext.PrebidExtKey, value, newRequestID))
	if enrichedRequest, err := jsonpatch.MergePatch(newRequestData[newRequestID], analyticsInfo); err == nil {
		requestData[requestID] = enrichedRequest
	} else {
		errs = append(errs, fmt.Errorf(`Cannot patch ext.prebid.storedrequest.ab_config in storedrequest id %s: %s`, newRequestID, err))
		// we can replace the request, but can't add the analytics info, so it will not be useful
		return
	}
	// augment requestData["imp"] with selected key
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

// runABSelection receives distribution rules in the form of a `{key: probability}` json map
// and returns the randomly selected key applying the rules.
func runABSelection(abConfig []byte) (selected string, err error) {
	throw := 100 * rand.Float64()
	t := 0.0
	err = jsonparser.ObjectEach(abConfig, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		//if t > 100 {
		//	return fmt.Errorf(`ab_config sum of probabilities %.1f greater than 100%%`, t)
		//}
		if percent, err := jsonparser.GetFloat(value); err != nil {
			return err
		} else {
			t += percent
			if throw < t && len(selected) == 0 {
				selected = string(key)
			}
		}
		return nil
	})
	return
}
