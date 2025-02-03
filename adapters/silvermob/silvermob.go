package silvermob

import (
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type SilverMobAdapter struct {
	endpoint *template.Template
}

func isValidHost(host string) bool {
	return host == "eu" || host == "us" || host == "apac" || host == "global"
}

// Builder builds a new instance of the SilverMob adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	template, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &SilverMobAdapter{
		endpoint: template,
	}
	return bidder, nil
}

func GetHeaders(request *openrtb2.BidRequest) *http.Header {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("X-Openrtb-Version", "2.5")

	if request.Device != nil {
		if len(request.Device.UA) > 0 {
			headers.Add("User-Agent", request.Device.UA)
		}

		if len(request.Device.IPv6) > 0 {
			headers.Add("X-Forwarded-For", request.Device.IPv6)
		}

		if len(request.Device.IP) > 0 {
			headers.Add("X-Forwarded-For", request.Device.IP)
		}
	}

	return &headers
}

func (a *SilverMobAdapter) MakeRequests(
	openRTBRequest *openrtb2.BidRequest,
	reqInfo *adapters.ExtraRequestInfo,
) (
	[]*adapters.RequestData,
	[]error,
) {
	requestCopy := *openRTBRequest
	impCount := len(openRTBRequest.Imp)
	requestData := make([]*adapters.RequestData, 0, impCount)
	errs := []error{}

	var err error

	for _, imp := range openRTBRequest.Imp {
		var silvermobExt *openrtb_ext.ExtSilverMob

		silvermobExt, err = a.getImpressionExt(&imp)

		if err != nil {
			errs = append(errs, err)
			continue
		}

		url, err := a.buildEndpointURL(silvermobExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		requestCopy.Imp = []openrtb2.Imp{imp}
		reqJSON, err := json.Marshal(requestCopy)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		reqData := &adapters.RequestData{
			Method:  http.MethodPost,
			Body:    reqJSON,
			Uri:     url,
			Headers: *GetHeaders(&requestCopy),
			ImpIDs:  openrtb_ext.GetImpIDs(requestCopy.Imp),
		}

		requestData = append(requestData, reqData)
	}

	return requestData, errs
}

func (a *SilverMobAdapter) getImpressionExt(imp *openrtb2.Imp) (*openrtb_ext.ExtSilverMob, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("error unmarshaling imp.ext: %s", err.Error()),
		}
	}
	var silvermobExt openrtb_ext.ExtSilverMob
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &silvermobExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("error unmarshaling imp.ext.bidder: %s", err.Error()),
		}
	}
	return &silvermobExt, nil
}

func (a *SilverMobAdapter) buildEndpointURL(params *openrtb_ext.ExtSilverMob) (string, error) {
	if isValidHost(params.Host) {
		endpointParams := macros.EndpointTemplateParams{ZoneID: params.ZoneID, Host: params.Host}
		return macros.ResolveMacros(a.endpoint, endpointParams)
	} else {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("invalid host %s", params.Host),
		}
	}
}

func (a *SilverMobAdapter) MakeBids(
	openRTBRequest *openrtb2.BidRequest,
	requestToBidder *adapters.RequestData,
	bidderRawResponse *adapters.ResponseData,
) (
	bidderResponse *adapters.BidderResponse,
	errs []error,
) {

	if bidderRawResponse.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if bidderRawResponse.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Bad Request status code: %d. Run with request.debug = 1 for more info", bidderRawResponse.StatusCode),
		}}
	}

	if bidderRawResponse.StatusCode != http.StatusOK {
		return nil, []error{fmt.Errorf("Unexpected status code: %d. Run with request.debug = 1 for more info", bidderRawResponse.StatusCode)}
	}

	responseBody := bidderRawResponse.Body
	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseBody, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Error unmarshaling server Response: %s", err),
		}}
	}

	if len(bidResp.SeatBid) == 0 {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Empty SeatBid array",
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bid := sb.Bid[i]
			bidType, err := getBidMediaTypeFromMtype(&bid)

			if err != nil {
				errs = append(errs, err)
			} else {
				bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
					Bid:     &bid,
					BidType: bidType,
				})
			}

		}
	}

	return bidResponse, errs
}

func getBidMediaTypeFromMtype(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	default:
		return "", fmt.Errorf("Unable to fetch mediaType for imp: %s", bid.ImpID)
	}
}
