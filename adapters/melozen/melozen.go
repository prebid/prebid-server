package melozen

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	endpointTemplate *template.Template
}

// Builder builds a new instance of the MeloZen adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	template, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &adapter{
		endpointTemplate: template,
	}

	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var requests []*adapters.RequestData
	var errors []error
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	requestCopy := *request
	for _, imp := range request.Imp {
		// Extract Melozen Params
		var strImpExt adapters.ExtImpBidder
		if err := jsonutil.Unmarshal(imp.Ext, &strImpExt); err != nil {
			errors = append(errors, err)
			continue
		}
		var strImpParams openrtb_ext.ImpExtMeloZen
		if err := jsonutil.Unmarshal(strImpExt.Bidder, &strImpParams); err != nil {
			errors = append(errors, err)
			continue
		}

		url, err := macros.ResolveMacros(a.endpointTemplate, macros.EndpointTemplateParams{PublisherID: strImpParams.PubId})
		if err != nil {
			errors = append(errors, err)
			continue
		}
		// Convert Floor into USD
		if imp.BidFloor > 0 && imp.BidFloorCur != "" && !strings.EqualFold(imp.BidFloorCur, "USD") {
			convertedValue, err := reqInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, "USD")
			if err != nil {
				errors = append(errors, err)
				continue
			}
			imp.BidFloorCur = "USD"
			imp.BidFloor = convertedValue
		}

		impressionsByMediaType, err := splitImpressionsByMediaType(&imp)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		for _, impression := range impressionsByMediaType {
			requestCopy.Imp = []openrtb2.Imp{impression}

			requestJSON, err := json.Marshal(requestCopy)
			if err != nil {
				errors = append(errors, err)
				continue
			}

			requestData := &adapters.RequestData{
				Method:  "POST",
				Uri:     url,
				Body:    requestJSON,
				Headers: headers,
				ImpIDs:  openrtb_ext.GetImpIDs(requestCopy.Imp),
			}
			requests = append(requests, requestData)
		}
	}

	return requests, errors
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(response) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
	}

	var bidReq openrtb2.BidRequest
	if err := jsonutil.Unmarshal(requestData.Body, &bidReq); err != nil {
		return nil, []error{err}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidderResponse := adapters.NewBidderResponse()
	var errors []error
	for _, seatBid := range bidResp.SeatBid {
		for i := range seatBid.Bid {
			bid := &seatBid.Bid[i]
			bidType, err := getMediaTypeForBid(*bid)
			if err != nil {
				errors = append(errors, err)
				continue
			}

			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				BidType: bidType,
				Bid:     bid,
			})
		}
	}
	return bidderResponse, errors
}

func splitImpressionsByMediaType(impression *openrtb2.Imp) ([]openrtb2.Imp, error) {
	if impression.Banner == nil && impression.Native == nil && impression.Video == nil {
		return nil, &errortypes.BadInput{Message: "Invalid MediaType. MeloZen only supports Banner, Video and Native."}
	}

	impressions := make([]openrtb2.Imp, 0, 2)

	if impression.Banner != nil {
		impCopy := *impression
		impCopy.Video = nil
		impCopy.Native = nil
		impressions = append(impressions, impCopy)
	}

	if impression.Video != nil {
		impCopy := *impression
		impCopy.Banner = nil
		impCopy.Native = nil
		impressions = append(impressions, impCopy)
	}

	if impression.Native != nil {
		impCopy := *impression
		impCopy.Banner = nil
		impCopy.Video = nil
		impressions = append(impressions, impCopy)
	}

	return impressions, nil
}

func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {

	if bid.Ext != nil {
		var bidExt openrtb_ext.ExtBid
		err := jsonutil.Unmarshal(bid.Ext, &bidExt)
		if err == nil && bidExt.Prebid != nil {
			return openrtb_ext.ParseBidType(string(bidExt.Prebid.Type))
		}
	}

	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("Failed to parse bid mediatype for impression \"%s\"", bid.ImpID),
	}
}
