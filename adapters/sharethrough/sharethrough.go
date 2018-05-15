package sharethrough

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const hbEndpoint = "https://dumb-waiter.sharethrough.com/header-bid/v1"

// SharethroughAdapter converts the Sharethrough Adserver response into a
// prebid server compatible format
type SharethroughAdapter struct {
	http *adapters.HTTPAdapter
	URI  string
}

// Name returns the adapter name as a string
func (s SharethroughAdapter) Name() string {
	return "sharethrough"
}

type sharethroughParams struct {
	BidID        string `json:"bidId"`
	PlacementKey string `json:"placement_key"`
	HBVersion    string `json:"hbVersion"`
	StrVersion   string `json:"strVersion"`
	HBSource     string `json:"hbSource"`
}

func (s *SharethroughAdapter) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
	pKeys := make([]string, 0, len(request.Imp))
	potentialRequests := make([]*adapters.RequestData)
	errs := make([]error, 0, len(request.Imp))

	for i := 0; i < len(request.Imp); i++ {
		pKey, err := preprocess(&request.Imp[i])
		if pKey != "" {
			pKeys = append(pKeys, pkey)
		}

		// If the preprocessing failed, the server won't be able to bid on this Imp. Delete it, and note the error.
		if err != nil {
			errs = append(errs, err)
			request.Imp = append(request.Imp[:i], request.Imp[i+1:]...)
			i--
		}
	}

	hbURI := generateHBUri(pKey, "testBidID")

	// If all the requests were malformed, don't bother making a server call with no impressions.
	if len(request.Imp) == 0 {
		return nil, errs
	}

	reqJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     thisURI,
		Body:    reqJSON,
		Headers: headers,
	}}, errs
}

func preprocess(imp *openrtb.Imp) (pKey, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return "", err
	}

	var sharethroughExt openrtb_ext.ExtImpSharethrough
	if err := json.Unmarshal(bidderExt, &sharethroughExt); err != nil {
		return "", err
	}

	return sharethroughExt.PlacementKey, nil
}

func appendPkey(uri string, pKey string) string {
	if strings.Contains(uri, "?") {
		return uri + "&placement_key=" + pKey
	}

	return uri + "?placement_key=" + pKey
}

func keys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for key, _ := range m {
		keys = append(keys, key)
	}
	return keys
}

func generateHBUri(pKey string, bidID string) string {
	v := url.Values{}
	v.Set("placement_key", pKey)
	v.Set("bidId", bidID)
	v.Set("hbVersion", "test-version")
	v.Set("hbSource", "prebid-server")

	return hbEndpoint + "?" + v.Encode()
}
