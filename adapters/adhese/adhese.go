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

	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type AdheseAdapter struct {
	endpointTemplate template.Template
}

func extractSlotParameter(parameters openrtb_ext.ExtImpAdhese) string {
	return fmt.Sprintf("/sl%s-%s", url.PathEscape(parameters.Location), url.PathEscape(parameters.Format))
}

func extractTargetParameters(parameters openrtb_ext.ExtImpAdhese) string {
	if len(parameters.Keywords) == 0 {
		return ""
	}
	var parametersAsString = ""
	var targetParsed map[string]interface{}
	err := json.Unmarshal(parameters.Keywords, &targetParsed)
	if err != nil {
		return ""
	}

	targetKeys := make([]string, 0, len(targetParsed))
	for key := range targetParsed {
		targetKeys = append(targetKeys, key)
	}
	sort.Strings(targetKeys)

	for _, targetKey := range targetKeys {
		var targetingValues = targetParsed[targetKey].([]interface{})
		parametersAsString += "/" + url.PathEscape(targetKey)
		for _, targetRawValKey := range targetingValues {
			var targetValueParsed = targetRawValKey.(string)
			parametersAsString += targetValueParsed + ";"
		}
		parametersAsString = strings.TrimRight(parametersAsString, ";")
	}

	return parametersAsString
}

func extractGdprParameter(request *openrtb2.BidRequest) string {
	if request.User != nil {
		var extUser openrtb_ext.ExtUser
		if err := json.Unmarshal(request.User.Ext, &extUser); err == nil {
			return "/xt" + extUser.Consent
		}
	}
	return ""
}

func extractRefererParameter(request *openrtb2.BidRequest) string {
	if request.Site != nil && request.Site.Page != "" {
		return "/xf" + url.QueryEscape(request.Site.Page)
	}
	return ""
}

func extractIfaParameter(request *openrtb2.BidRequest) string {
	if request.Device != nil && request.Device.IFA != "" {
		return "/xz" + url.QueryEscape(request.Device.IFA)
	}
	return ""
}

func (a *AdheseAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	errs := make([]error, 0, len(request.Imp))

	var err error

	// If all the requests are invalid, Call to adaptor is skipped
	if len(request.Imp) == 0 {
		errs = append(errs, WrapReqError("Imp is empty"))
		return nil, errs
	}

	var imp = &request.Imp[0]
	var bidderExt adapters.ExtImpBidder

	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		errs = append(errs, WrapReqError("Request could not be parsed as ExtImpBidder due to: "+err.Error()))
		return nil, errs
	}

	var params openrtb_ext.ExtImpAdhese
	if err := json.Unmarshal(bidderExt.Bidder, &params); err != nil {
		errs = append(errs, WrapReqError("Request could not be parsed as ExtImpAdhese due to: "+err.Error()))
		return nil, errs
	}

	// Compose url
	endpointParams := macros.EndpointTemplateParams{AccountID: params.Account}

	host, err := macros.ResolveMacros(*&a.endpointTemplate, endpointParams)
	if err != nil {
		errs = append(errs, WrapReqError("Could not compose url from template and request account val: "+err.Error()))
		return nil, errs
	}
	complete_url := fmt.Sprintf("%s%s%s%s%s%s",
		host,
		extractSlotParameter(params),
		extractTargetParameters(params),
		extractGdprParameter(request),
		extractRefererParameter(request),
		extractIfaParameter(request))

	return []*adapters.RequestData{{
		Method: "GET",
		Uri:    complete_url,
	}}, errs
}

func (a *AdheseAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	} else if response.StatusCode != http.StatusOK {
		return nil, []error{WrapServerError(fmt.Sprintf("Unexpected status code: %d.", response.StatusCode))}
	}

	var bidResponse openrtb2.BidResponse

	var adheseBidResponseArray []AdheseBid
	if err := json.Unmarshal(response.Body, &adheseBidResponseArray); err != nil {
		return nil, []error{err, WrapServerError(fmt.Sprintf("Response %v could not be parsed as generic Adhese bid.", string(response.Body)))}
	}

	var adheseBid = adheseBidResponseArray[0]

	if adheseBid.Origin == "JERLICIA" {
		var extArray []AdheseExt
		var originDataArray []AdheseOriginData
		if err := json.Unmarshal(response.Body, &extArray); err != nil {
			return nil, []error{err, WrapServerError(fmt.Sprintf("Response %v could not be parsed to JERLICIA ext.", string(response.Body)))}
		}

		if err := json.Unmarshal(response.Body, &originDataArray); err != nil {
			return nil, []error{err, WrapServerError(fmt.Sprintf("Response %v could not be parsed to JERLICIA origin data.", string(response.Body)))}
		}
		bidResponse = convertAdheseBid(adheseBid, extArray[0], originDataArray[0])
	} else {
		bidResponse = convertAdheseOpenRtbBid(adheseBid)
	}

	price, err := strconv.ParseFloat(adheseBid.Extension.Prebid.Cpm.Amount, 64)
	if err != nil {
		return nil, []error{err, WrapServerError(fmt.Sprintf("Could not parse Price %v as float ", string(adheseBid.Extension.Prebid.Cpm.Amount)))}
	}
	width, err := strconv.ParseInt(adheseBid.Width, 10, 64)
	if err != nil {
		return nil, []error{err, WrapServerError(fmt.Sprintf("Could not parse Width %v as int ", string(adheseBid.Width)))}
	}
	height, err := strconv.ParseInt(adheseBid.Height, 10, 64)
	if err != nil {
		return nil, []error{err, WrapServerError(fmt.Sprintf("Could not parse Height %v as int ", string(adheseBid.Height)))}
	}
	bidResponse.Cur = adheseBid.Extension.Prebid.Cpm.Currency
	if len(bidResponse.SeatBid) > 0 && len(bidResponse.SeatBid[0].Bid) > 0 {
		bidResponse.SeatBid[0].Bid[0].Price = price
		bidResponse.SeatBid[0].Bid[0].W = width
		bidResponse.SeatBid[0].Bid[0].H = height
	}

	bidderResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	if len(bidResponse.SeatBid) == 0 {
		return nil, []error{WrapServerError("Response resulted in an empty seatBid array.")}
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

func convertAdheseBid(adheseBid AdheseBid, adheseExt AdheseExt, adheseOriginData AdheseOriginData) openrtb2.BidResponse {
	adheseExtJson, err := json.Marshal(adheseOriginData)
	if err != nil {
		glog.Error(fmt.Sprintf("Unable to parse adhese Origin Data as JSON due to %v", err))
		adheseExtJson = make([]byte, 0)
	}
	return openrtb2.BidResponse{
		ID: adheseExt.Id,
		SeatBid: []openrtb2.SeatBid{{
			Bid: []openrtb2.Bid{{
				DealID: adheseExt.OrderId,
				CrID:   adheseExt.Id,
				AdM:    getAdMarkup(adheseBid, adheseExt),
				Ext:    adheseExtJson,
			}},
			Seat: "",
		}},
	}
}

func convertAdheseOpenRtbBid(adheseBid AdheseBid) openrtb2.BidResponse {
	var response openrtb2.BidResponse = adheseBid.OriginData
	if len(response.SeatBid) > 0 && len(response.SeatBid[0].Bid) > 0 {
		response.SeatBid[0].Bid[0].AdM = adheseBid.Body
	}
	return response
}

func getAdMarkup(adheseBid AdheseBid, adheseExt AdheseExt) string {
	if adheseExt.Ext == "js" {
		if ContainsAny(adheseBid.Body, []string{"<script", "<div", "<html"}) {
			counter := ""
			if len(adheseExt.ImpressionCounter) > 0 {
				counter = "<img src='" + adheseExt.ImpressionCounter + "' style='height:1px; width:1px; margin: -1px -1px; display:none;'/>"
			}
			return adheseBid.Body + counter
		}
		if ContainsAny(adheseBid.Body, []string{"<?xml", "<vast"}) {
			return adheseBid.Body
		}
	}
	return adheseExt.Tag
}

func getBidType(bidAdm string) openrtb_ext.BidType {
	if bidAdm != "" && ContainsAny(bidAdm, []string{"<?xml", "<vast"}) {
		return openrtb_ext.BidTypeVideo
	}
	return openrtb_ext.BidTypeBanner
}

func WrapReqError(errorStr string) *errortypes.BadInput {
	return &errortypes.BadInput{Message: errorStr}
}

func WrapServerError(errorStr string) *errortypes.BadServerResponse {
	return &errortypes.BadServerResponse{Message: errorStr}
}

func ContainsAny(raw string, keys []string) bool {
	lowerCased := strings.ToLower(raw)
	for i := 0; i < len(keys); i++ {
		if strings.Contains(lowerCased, keys[i]) {
			return true
		}
	}
	return false

}

// Builder builds a new instance of the Adhese adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	template, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &AdheseAdapter{
		endpointTemplate: *template,
	}
	return bidder, nil
}
