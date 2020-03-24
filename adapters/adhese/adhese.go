package adhese

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type AdheseAdapter struct {
	http             *adapters.HTTPAdapter
	URI              string
	dummyCacheBuster int
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

func extractSlotParameter(parameters openrtb_ext.ExtImpAdhese) string {
	return fmt.Sprintf("/sl%s-%s", parameters.Location, parameters.Format)
}

func extractTargetParameters(parameters openrtb_ext.ExtImpAdhese) string {
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
		parametersAsString += "/" + k + strings.Join(v, ";")
	}
	params := strings.Split(parametersAsString, "/")
	sort.Strings(params)
	return strings.Join(params, "/")

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
		errs = append(errs, WrapError("Request could not be parsed as ExtImpBidder due to: "+err.Error()))
		return nil, errs
	}

	var params openrtb_ext.ExtImpAdhese
	if err := json.Unmarshal(bidderExt.Bidder, &params); err != nil {
		errs = append(errs, WrapError("Request could not be parsed as ExtImpAdhese due to: "+err.Error()))
		return nil, errs
	}

	// Validate request
	if params.Account == "" || params.Location == "" || params.Format == "" {
		errs = append(errs, WrapError("Request is missing a required parameter (Account, Location and/or Format)"))
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

	return []*adapters.RequestData{{
		Method: "GET",
		Uri:    complete_url,
		Body:   reqJSON,
	}}, errs
}

func (a *AdheseAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{WrapError(fmt.Sprintf("Unexpected status code: %d.", response.StatusCode))}
	}

	var originArray []openrtb_ext.AdheseOrigin
	var bidResponse openrtb.BidResponse

	if err := json.Unmarshal(response.Body, &originArray); err != nil {
		return nil, []error{err, WrapError(fmt.Sprintf("Response %v does not have an Origin.", string(response.Body)))}
	}

	if originArray[0].Origin == "JERLICIA" || originArray[0].Origin == "DALE" {
		var adheseBidResponseArray []openrtb_ext.AdheseBid
		if err := json.Unmarshal(response.Body, &adheseBidResponseArray); err != nil {
			return nil, []error{err, WrapError(fmt.Sprintf("Response %v could not be parsed as Adhese bid.", string(response.Body)))}
		}
		bidResponse = convertAdheseBid(adheseBidResponseArray[0])
	} else {
		var openRtbBidResponseArray []openrtb_ext.AdheseOpenRtbBid
		if err := json.Unmarshal(response.Body, &openRtbBidResponseArray); err != nil {
			return nil, []error{err, WrapError(fmt.Sprintf("Response %v could not be parsed as Adhese OpenRtb Bid.", string(response.Body)))}
		}
		bidResponse = convertAdheseOpenRtbBid(openRtbBidResponseArray[0])
	}

	bidderResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	if len(bidResponse.SeatBid) == 0 {
		return nil, []error{WrapError("Response resulted in an empty seatBid array.")}
	}

	var errs []error
	for _, sb := range bidResponse.SeatBid {
		for i := 0; i < len(sb.Bid); i++ {
			bid := sb.Bid[i]
			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: getBidType(bid.AdM),
			})

		}
	}
	return bidderResponse, errs
}

func convertAdheseBid(adheseBid openrtb_ext.AdheseBid) openrtb.BidResponse {
	price, _ := strconv.ParseFloat(adheseBid.Extension.Prebid.Cpm.Amount, 64)
	width, _ := strconv.ParseUint(adheseBid.Width, 10, 64)
	height, _ := strconv.ParseUint(adheseBid.Height, 10, 64)
	adheseObj, _ := json.Marshal(openrtb_ext.ExtAdhese{
		CreativeId:                adheseBid.Id,
		AdFormat:                  adheseBid.AdFormat,
		AdType:                    adheseBid.AdType,
		AdspaceId:                 adheseBid.AdspaceId,
		DealId:                    adheseBid.OrderId,
		LibId:                     adheseBid.LibId,
		OrderProperty:             adheseBid.OrderProperty,
		Priority:                  adheseBid.Priority,
		ViewableImpressionCounter: adheseBid.ViewableImpressionCounter,
	})
	return openrtb.BidResponse{
		ID: adheseBid.Id,
		SeatBid: []openrtb.SeatBid{{
			Bid: []openrtb.Bid{{
				DealID: adheseBid.OrderId,
				Price:  price,
				W:      width,
				H:      height,
				CID:    adheseBid.OrderId,
				CrID:   adheseBid.Id,
				NURL:   adheseBid.ImpressionCounter,
				BURL:   adheseBid.Tracker,
				AdM:    getAdMarkup(adheseBid),
				Ext:    adheseObj,
			}},
			Seat: "",
		}},
		BidID: adheseBid.OrderId,
		Cur:   adheseBid.Extension.Prebid.Cpm.Currency,
	}
}

func convertAdheseOpenRtbBid(adheseBid openrtb_ext.AdheseOpenRtbBid) openrtb.BidResponse {
	price, _ := strconv.ParseFloat(adheseBid.Extension.Prebid.Cpm.Amount, 64)
	width, _ := strconv.ParseUint(adheseBid.Width, 10, 64)
	height, _ := strconv.ParseUint(adheseBid.Height, 10, 64)
	var response openrtb.BidResponse = adheseBid.OriginData
	response.ID = adheseBid.Origin
	if adheseBid.OriginInstance != "" {
		response.ID = response.ID + "-" + adheseBid.OriginInstance
	}
	if len(response.SeatBid) > 0 && len(response.SeatBid[0].Bid) > 0 {
		response.SeatBid[0].Bid[0].Price = price
		response.SeatBid[0].Bid[0].W = width
		response.SeatBid[0].Bid[0].H = height
		response.SeatBid[0].Bid[0].AdM = adheseBid.Body
		if ContainsAny(adheseBid.Body, []string{"<script", "<div", "<html"}) {
			response.SeatBid[0].Bid[0].AdM += "<img src='" + adheseBid.ImpressionCounter + "' style='height:1px; width:1px; margin: -1px -1px; display:none;'/>"
		}
	}

	response.Cur = adheseBid.Extension.Prebid.Cpm.Currency
	return response
}

func getAdMarkup(adheseBid openrtb_ext.AdheseBid) string {
	if adheseBid.Ext == "js" && ContainsAny(adheseBid.Body, []string{"<script", "<div", "<html"}) {
		return adheseBid.Body + "<img src='" + adheseBid.ImpressionCounter + "' style='height:1px; width:1px; margin: -1px -1px; display:none;'/>"
	} else if adheseBid.Ext == "js" && ContainsAny(adheseBid.Body, []string{"<?xml", "<vast"}) {
		return adheseBid.Body
	} else {
		return adheseBid.Tag
	}
}

func getBidType(bidAdm string) openrtb_ext.BidType {
	if bidAdm != "" && ContainsAny(bidAdm, []string{"<?xml", "<vast"}) {
		return openrtb_ext.BidTypeVideo
	}
	return openrtb_ext.BidTypeBanner
}

func WrapError(errorStr string) *errortypes.BadInput {
	return &errortypes.BadInput{Message: errorStr}
}

func ContainsAny(raw string, keys []string) bool {
	for i := 0; i < len(keys); i++ {
		if strings.Contains(strings.ToLower(raw), keys[i]) {
			return true
		}
	}
	return false

}

func NewAdheseAdapter(config *adapters.HTTPAdapterConfig, uri string) *AdheseAdapter {
	return NewAdheseBidder(adapters.NewHTTPAdapter(config).Client, uri, 0)
}

// Set dummyCacheBuster to 0 in order to generate a cache buster
func NewAdheseBidder(client *http.Client, uri string, dummyCacheBuster int) *AdheseAdapter {
	return &AdheseAdapter{http: &adapters.HTTPAdapter{Client: client}, URI: uri, dummyCacheBuster: dummyCacheBuster}
}

func printAsJson(obj interface{}) {
	outy, _ := json.Marshal(obj)
	fmt.Println(string(outy))
}
