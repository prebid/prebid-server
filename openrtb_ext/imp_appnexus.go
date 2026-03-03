package openrtb_ext

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

// ExtImpAppnexus defines the contract for bidrequest.imp[i].ext.prebid.bidder.appnexus
type ExtImpAppnexus struct {
	DeprecatedPlacementId    jsonutil.StringInt     `json:"placementId"`
	LegacyInvCode            string                 `json:"invCode"`
	LegacyTrafficSourceCode  string                 `json:"trafficSourceCode"`
	PlacementId              jsonutil.StringInt     `json:"placement_id"`
	InvCode                  string                 `json:"inv_code"`
	Member                   jsonutil.IntString     `json:"member"`
	Keywords                 ExtImpAppnexusKeywords `json:"keywords"`
	TrafficSourceCode        string                 `json:"traffic_source_code"`
	Reserve                  float64                `json:"reserve"`
	Position                 string                 `json:"position"`
	UsePaymentRule           *bool                  `json:"use_pmt_rule"`
	DeprecatedUsePaymentRule *bool                  `json:"use_payment_rule"`
	// At this time we do no processing on the private sizes, so just leaving it as a JSON blob.
	PrivateSizes  json.RawMessage `json:"private_sizes"`
	AdPodId       bool            `json:"generate_ad_pod_id"`
	ExtInvCode    string          `json:"ext_inv_code"`
	ExternalImpId string          `json:"external_imp_id"`
}

type ExtImpAppnexusKeywords string

// extImpAppnexusKeyVal defines the contract for bidrequest.imp[i].ext.prebid.bidder.appnexus.keywords[i]
type extImpAppnexusKeyVal struct {
	Key    string   `json:"key,omitempty"`
	Values []string `json:"value,omitempty"`
}

func (ks *ExtImpAppnexusKeywords) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return nil
	}

	switch b[0] {
	case '{':
		var results map[string][]string
		if err := jsonutil.UnmarshalValid(b, &results); err != nil {
			return err
		}

		var keywords strings.Builder
		for key, values := range results {
			if len(values) == 0 {
				keywords.WriteString(fmt.Sprintf("%s,", key))
			} else {
				for _, val := range values {
					keywords.WriteString(fmt.Sprintf("%s=%s,", key, val))
				}
			}
		}
		if len(keywords.String()) > 0 {
			*ks = ExtImpAppnexusKeywords(keywords.String()[:keywords.Len()-1])
		}
	case '[':
		var results []extImpAppnexusKeyVal
		if err := jsonutil.UnmarshalValid(b, &results); err != nil {
			return err
		}
		var kvs strings.Builder
		for _, kv := range results {
			if len(kv.Values) == 0 {
				kvs.WriteString(fmt.Sprintf("%s,", kv.Key))
			} else {
				for _, val := range kv.Values {
					kvs.WriteString(fmt.Sprintf("%s=%s,", kv.Key, val))
				}
			}
		}
		if len(kvs.String()) > 0 {
			*ks = ExtImpAppnexusKeywords(kvs.String()[:kvs.Len()-1])
		}
	case '"':
		var keywords string
		if err := jsonutil.UnmarshalValid(b, &keywords); err != nil {
			return err
		}
		*ks = ExtImpAppnexusKeywords(keywords)
	}
	return nil
}

func (ks *ExtImpAppnexusKeywords) String() string {
	return *(*string)(ks)
}
