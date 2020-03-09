package adhese

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type AdheseAdapter struct {
	http             *adapters.HTTPAdapter
	dummyCacheBuster int
}

type AdheseParams struct {
	Account  string                  `json:"account"`
	Location string                  `json:"location"`
	Format   string                  `json:"format"`
	Keywords []*AdheseKeywordsParams `json:"targets,omitempty"`
}

type AdheseKeywordsParams struct {
	Key    string   `json:"key,omitempty"`
	Values []string `json:"value,omitempty"`
}

func (a *AdheseAdapter) Name() string {
	return "adhese"
}

func (a *AdheseAdapter) SkipNoCookies() bool {
	return false
}

func extractSlotParameter(parameters AdheseParams) string {
	return fmt.Sprintf("/sl%s-%s", parameters.Location, parameters.Format)
}

func extractTargetParameters(parameters AdheseParams) string {
	if parameters.Keywords == nil || len(parameters.Keywords) == 0 {
		return ""
	}
	m := make(map[string][]string)
	for _, kv := range parameters.Keywords {
		for _, tv := range kv.Values {
			cur, _ := m[kv.Key]
			new := cur[:]
			m[kv.Key] = append(new, tv)
		}
	}

	var parametersAsString = ""

	for k, v := range m {
		var parameterAsString = "/" + k
		for i := 0; i < len(v); i++ {
			parameterAsString += v[i]
			if (i + 1) < len(v) {
				parameterAsString += ";"
			}
		}
		parametersAsString += parameterAsString
	}
	return parametersAsString

}

func extractGdprParameter() string {
	//const gdprParams = (gdprConsent && gdprConsent.consentString) ? [ 'xt' + gdprConsent.consentString, 'tlall' ] : [];
	return ""
}

func extractRefererParameter() string {
	//const refererParams = (refererInfo && refererInfo.referer) ? [ 'xf' + base64urlEncode(refererInfo.referer) ] : [];
	return ""
}

func (a *AdheseAdapter) generateCacheBuster() string {
	if a.dummyCacheBuster > 0 {
		return fmt.Sprintf("?t=%d", a.dummyCacheBuster)
	}
	return fmt.Sprintf("?t=%d", time.Now().UnixNano())
}

func (a *AdheseAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	errs := make([]error, 0, len(request.Imp))

	var err error
	var imp = &request.Imp[0]
	var bidderExt adapters.ExtImpBidder

	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	var params AdheseParams
	if err := json.Unmarshal(bidderExt.Bidder, &params); err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	// Validate request
	if params.Account == "" || params.Location == "" || params.Format == "" {
		errs = append(errs, WrapError("Request is missing a required parameter (Account, Location and/or Format"))
		return nil, errs
	}

	// Compose url
	complete_url := fmt.Sprintf("https://ads-%s.adhese.com/json%s%s%s%s%s",
		params.Account,
		extractSlotParameter(params),
		extractTargetParameters(params),
		extractGdprParameter(),
		extractRefererParameter(),
		a.generateCacheBuster())

	// If all the requests are invalid, Call to adaptor is skipped
	if len(request.Imp) == 0 {
		return nil, errs
	}

	reqJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	//headers := http.Header{}
	//headers.Add("Content-Type", "application/json;charset=utf-8")
	//headers.Add("Accept", "application/json")
	return []*adapters.RequestData{{
		Method: "GET",
		Uri:    complete_url,
		Body:   reqJSON,
		//Headers: headers,
	}}, errs
}

// TODO: rewrite ---------------------------------------------
func (a *AdheseAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{WrapError(fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode))}
	}

	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	var errs []error
	for _, sb := range bidResp.SeatBid {
		for i := 0; i < len(sb.Bid); i++ {
			bid := sb.Bid[i]
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: getBidType(bid.Ext),
			})

		}
	}
	return bidResponse, errs
}

const bidTypeExtKey = "BidType"

func getBidType(bidExt json.RawMessage) openrtb_ext.BidType {
	// setting "banner" as the default bid type
	bidType := openrtb_ext.BidTypeBanner
	if bidExt != nil {
		bidExtMap := make(map[string]interface{})
		extbyte, err := json.Marshal(bidExt)
		if err == nil {
			err = json.Unmarshal(extbyte, &bidExtMap)
			if err == nil && bidExtMap[bidTypeExtKey] != nil {
				bidTypeVal := int(bidExtMap[bidTypeExtKey].(float64))
				switch bidTypeVal {
				case 0:
					bidType = openrtb_ext.BidTypeBanner
				case 1:
					bidType = openrtb_ext.BidTypeVideo
				case 2:
					bidType = openrtb_ext.BidTypeNative
				default:
					// default value is banner
					bidType = openrtb_ext.BidTypeBanner
				}
			}
		}
	}
	return bidType
}

// ----------------------------------
func WrapError(errorStr string) *errortypes.BadInput {
	return &errortypes.BadInput{Message: errorStr}
}

func NewAdheseAdapter(config *adapters.HTTPAdapterConfig) *AdheseAdapter {
	return NewAdheseBidder(adapters.NewHTTPAdapter(config).Client, 0)
}

// Set dummyCacheBuster to 0 in order to generate a cache buster
func NewAdheseBidder(client *http.Client, dummyCacheBuster int) *AdheseAdapter {
	return &AdheseAdapter{http: &adapters.HTTPAdapter{Client: client}, dummyCacheBuster: dummyCacheBuster}
}
