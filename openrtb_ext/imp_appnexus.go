package openrtb_ext

import (
	"encoding/json"

	"github.com/prebid/prebid-server/util/jsonutil"
)

// ExtImpAppnexus defines the contract for bidrequest.imp[i].ext.prebid.bidder.appnexus
type ExtImpAppnexus struct {
	DeprecatedPlacementId    jsonutil.StringInt      `json:"placementId"`
	LegacyInvCode            string                  `json:"invCode"`
	LegacyTrafficSourceCode  string                  `json:"trafficSourceCode"`
	PlacementId              jsonutil.StringInt      `json:"placement_id"`
	InvCode                  string                  `json:"inv_code"`
	Member                   string                  `json:"member"`
	Keywords                 []*ExtImpAppnexusKeyVal `json:"keywords"`
	TrafficSourceCode        string                  `json:"traffic_source_code"`
	Reserve                  float64                 `json:"reserve"`
	Position                 string                  `json:"position"`
	UsePaymentRule           *bool                   `json:"use_pmt_rule"`
	DeprecatedUsePaymentRule *bool                   `json:"use_payment_rule"`
	// At this time we do no processing on the private sizes, so just leaving it as a JSON blob.
	PrivateSizes json.RawMessage `json:"private_sizes"`
	AdPodId      bool            `json:"generate_ad_pod_id"`
}

// ExtImpAppnexusKeyVal defines the contract for bidrequest.imp[i].ext.prebid.bidder.appnexus.keywords[i]
type ExtImpAppnexusKeyVal struct {
	Key    string   `json:"key,omitempty"`
	Values []string `json:"value,omitempty"`
}
