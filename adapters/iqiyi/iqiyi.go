package iqiyi

import (
	"encoding/json"
	"fmt"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"net/http"
	"text/template"
)

type adapter struct {
	endpoint *template.Template
}

func pickCurrency(req *openrtb2.BidRequest, resp *openrtb2.BidResponse) string {
	if resp.Cur != "" {
		return resp.Cur
	}

	if len(req.Cur) > 0 && req.Cur[0] != "" {
		return req.Cur[0]
	}

	return "USD"
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	template, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err == nil {
		_, err = macros.ResolveMacros(template, macros.EndpointTemplateParams{})
	}

	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %w", err)
	}

	bidder := &adapter{
		endpoint: template,
	}
	return bidder, nil
}

func (a *adapter) buildEndpointURL(params *openrtb_ext.ExtImpIqiyi) (string, error) {
	endpointParams := macros.EndpointTemplateParams{AccountID: params.AccountID}
	return macros.ResolveMacros(a.endpoint, endpointParams)
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	if len(request.Imp) == 0 {
		errs = append(errs, &errortypes.BadInput{Message: "No impression in the request"})
		return nil, errs
	}

	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(request.Imp[0].Ext, &bidderExt); err != nil {
		errs = append(errs, &errortypes.BadInput{Message: "bad Iqiyi bidder ext"})
		return nil, errs
	}

	var iqiyiExt openrtb_ext.ExtImpIqiyi
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &iqiyiExt); err != nil {
		errs = append(errs, &errortypes.BadInput{Message: "bad Iqiyi bidder ext"})
		return nil, errs
	}

	for i := range request.Imp {
		imp := &request.Imp[i]
		if imp.Banner != nil {
			b := *imp.Banner
			if (b.W == nil || b.H == nil || *b.W == 0 || *b.H == 0) && len(b.Format) > 0 {
				first := b.Format[0]
				b.W = &first.W
				b.H = &first.H
				imp.Banner = &b
			}
		}
		if imp.BidFloorCur == "" && imp.BidFloor > 0 {
			imp.BidFloorCur = "USD"
		}
	}

	url, err := a.buildEndpointURL(&iqiyiExt)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	reqJson, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     url,
		Body:    reqJson,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}}, nil
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected http status code: %d", response.StatusCode),
		}}
	}

	var serverBidResponse openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &serverBidResponse); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)
	bidResponse.Currency = pickCurrency(internalRequest, &serverBidResponse)

	for _, seatbid := range serverBidResponse.SeatBid {
		for i := range seatbid.Bid {
			mediaType, err := getMediaTypeForImp(seatbid.Bid[i])
			if err != nil {
				return nil, []error{err}
			}
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatbid.Bid[i],
				BidType: mediaType,
			})
		}
	}

	return bidResponse, nil
}

func getMediaTypeForImp(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unsupported mtype %d for bid %s", bid.MType, bid.ID),
		}
	}
}
