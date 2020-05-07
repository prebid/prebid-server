package adhese

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type AdheseAdapter struct {
	http             *adapters.HTTPAdapter
	endpointTemplate template.Template
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
	var targetParsed map[string]interface{}
	json.Unmarshal(parameters.Keywords, &targetParsed)
	for targetKey, targetRawValue := range targetParsed {
		var targetingValues = targetRawValue.([]interface{})
		for _, targetRawValKey := range targetingValues {
			var targetValueParsed = targetRawValKey.(string)
			cur, _ := m[targetKey]
			new := cur[:]
			m[targetKey] = append(new, targetValueParsed)
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

func extractGdprParameter(request *openrtb.BidRequest) string {
	if request.User != nil {
		var extUser openrtb_ext.ExtUser
		if err := json.Unmarshal(request.User.Ext, &extUser); err == nil {
			return "/xt" + extUser.Consent
		}
	}
	return ""
}

func extractRefererParameter(request *openrtb.BidRequest) string {
	if request.Site != nil && request.Site.Page != "" {
		return "/xf" + url.QueryEscape(request.Site.Page)
	}
	return ""
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
	endpointParams := macros.EndpointTemplateParams{Host: "ads-" + params.Account + ".adhese.com"}

	host, err := macros.ResolveMacros(*&a.endpointTemplate, endpointParams)
	if err != nil {
		errs = append(errs, WrapError("Could not compose url from template and request account val: "+err.Error()))
		return nil, errs
	}
	complete_url := fmt.Sprintf("%s%s%s%s%s",
		host,
		extractSlotParameter(params),
		extractTargetParameters(params),
		extractGdprParameter(request),
		extractRefererParameter(request))

	// If all the requests are invalid, Call to adaptor is skipped
	if len(request.Imp) == 0 {
		return nil, errs
	}

	return []*adapters.RequestData{{
		Method: "GET",
		Uri:    complete_url,
	}}, errs
}

func (a *AdheseAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode != http.StatusOK {
		return nil, []error{WrapError(fmt.Sprintf("Unexpected status code: %d.", response.StatusCode))}
	}

	var bidResponse openrtb.BidResponse

	var adheseBidResponseArray []openrtb_ext.AdheseBid
	if err := json.Unmarshal(response.Body, &adheseBidResponseArray); err != nil {
		return nil, []error{err, WrapError(fmt.Sprintf("Response %v could not be parsed as generic Adhese bid.", string(response.Body)))}
	}

	var adheseBid = adheseBidResponseArray[0]

	if adheseBid.Origin == "JERLICIA" {
		var extArray []openrtb_ext.AdheseExt
		var originDataArray []openrtb_ext.AdheseOriginData
		if err := json.Unmarshal(response.Body, &extArray); err != nil {
			return nil, []error{err, WrapError(fmt.Sprintf("Response %v could not be parsed to JERLICIA ext.", string(response.Body)))}
		}

		if err := json.Unmarshal(response.Body, &originDataArray); err != nil {
			return nil, []error{err, WrapError(fmt.Sprintf("Response %v could not be parsed to JERLICIA origin data.", string(response.Body)))}
		}
		bidResponse = convertAdheseBid(adheseBid, extArray[0], originDataArray[0])
	} else {
		bidResponse = convertAdheseOpenRtbBid(adheseBid)
	}

	price, _ := strconv.ParseFloat(adheseBid.Extension.Prebid.Cpm.Amount, 64)
	width, _ := strconv.ParseUint(adheseBid.Width, 10, 64)
	height, _ := strconv.ParseUint(adheseBid.Height, 10, 64)
	bidResponse.Cur = adheseBid.Extension.Prebid.Cpm.Currency
	if len(bidResponse.SeatBid) > 0 && len(bidResponse.SeatBid[0].Bid) > 0 {
		bidResponse.SeatBid[0].Bid[0].Price = price
		bidResponse.SeatBid[0].Bid[0].W = width
		bidResponse.SeatBid[0].Bid[0].H = height
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

func convertAdheseBid(adheseBid openrtb_ext.AdheseBid, adheseExt openrtb_ext.AdheseExt, adheseOriginData openrtb_ext.AdheseOriginData) openrtb.BidResponse {
	adheseExtJson, _ := json.Marshal(adheseOriginData)

	return openrtb.BidResponse{
		ID: adheseExt.Id,
		SeatBid: []openrtb.SeatBid{{
			Bid: []openrtb.Bid{{
				DealID: adheseExt.OrderId,
				CrID:   adheseExt.Id,
				AdM:    getAdMarkup(adheseBid, adheseExt),
				Ext:    adheseExtJson,
			}},
			Seat: "",
		}},
	}
}

func convertAdheseOpenRtbBid(adheseBid openrtb_ext.AdheseBid) openrtb.BidResponse {
	var response openrtb.BidResponse = adheseBid.OriginData
	if len(response.SeatBid) > 0 && len(response.SeatBid[0].Bid) > 0 {
		response.SeatBid[0].Bid[0].AdM = adheseBid.Body
	}
	return response
}

func getAdMarkup(adheseBid openrtb_ext.AdheseBid, adheseExt openrtb_ext.AdheseExt) string {
	if adheseExt.Ext == "js" && ContainsAny(adheseBid.Body, []string{"<script", "<div", "<html"}) {
		return adheseBid.Body + "<img src='" + adheseExt.ImpressionCounter + "' style='height:1px; width:1px; margin: -1px -1px; display:none;'/>"
	} else if adheseExt.Ext == "js" && ContainsAny(adheseBid.Body, []string{"<?xml", "<vast"}) {
		return adheseBid.Body
	} else {
		return adheseExt.Tag
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
	return NewAdheseBidder(adapters.NewHTTPAdapter(config).Client, uri)
}

func NewAdheseBidder(client *http.Client, uri string) *AdheseAdapter {
	template, _ := template.New("endpointTemplate").Parse(uri)
	return &AdheseAdapter{http: &adapters.HTTPAdapter{Client: client}, endpointTemplate: *template}
}

func printAsJson(obj interface{}) {
	outy, _ := json.Marshal(obj)
	fmt.Println(string(outy))
}
