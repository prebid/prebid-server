package roulax

import (
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	endpoint *template.Template
}

// Builder builds a new instance of the Roulax adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	template, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &adapter{
		endpoint: template,
	}
	return bidder, nil
}

// getImpAdotExt parses and return first imp ext or nil
func getImpRoulaxExt(imp *openrtb2.Imp) (openrtb_ext.ExtImpRoulax, error) {
	var extBidder adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &extBidder); err != nil {
		return openrtb_ext.ExtImpRoulax{}, err
	}
	var extImpRoulax openrtb_ext.ExtImpRoulax
	if err := jsonutil.Unmarshal(extBidder.Bidder, &extImpRoulax); err != nil {
		return openrtb_ext.ExtImpRoulax{}, err
	}
	return extImpRoulax, nil
}

// MakeRequests makes the HTTP requests which should be made to fetch bids.
func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) (res []*adapters.RequestData, errs []error) {
	reqJson, err := json.Marshal(request)
	if err != nil {
		return nil, append(errs, err)
	}
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	roulaxExt, err := getImpRoulaxExt(&request.Imp[0])
	if err != nil {
		return nil, append(errs, err)
	}

	endpointParams := macros.EndpointTemplateParams{AccountID: roulaxExt.Pid, PublisherID: roulaxExt.PublisherPath}
	url, err := macros.ResolveMacros(a.endpoint, endpointParams)
	if err != nil {
		return nil, append(errs, err)
	}

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     url,
		Body:    reqJson,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}}, errs
}

// MakeBids unpacks the server's response into Bids.
func (a *adapter) MakeBids(request *openrtb2.BidRequest, _ *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}
	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}
	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur
	var errs []error
	for _, seatBid := range response.SeatBid {
		for i, bid := range seatBid.Bid {
			bidType, err := getMediaTypeForBid(bid)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			})
		}
	}
	return bidResponse, errs
}

func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	default:
		return "", fmt.Errorf("Unable to fetch mediaType in impID: %s, mType: %d", bid.ImpID, bid.MType)
	}
}
