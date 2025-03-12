package madsense

import (
	"net/http"
	"net/url"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the MadSense adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	reqs := make([]*adapters.RequestData, 0, len(request.Imp))
	var errs []error

	appendReq := func(imps []openrtb2.Imp) {
		req, err := a.makeRequest(request, imps)
		if err != nil {
			errs = append(errs, err)
			return
		}
		if req != nil {
			reqs = append(reqs, req)
		}
	}

	var videoImps []openrtb2.Imp
	for i := range request.Imp {
		imp := &request.Imp[i]
		if imp.Banner != nil {
			appendReq(request.Imp[i : i+1])
		} else if imp.Video != nil {
			videoImps = append(videoImps, request.Imp[i])
		}
	}

	// we support video podding, so we want to send all video impressions in a single request
	appendReq(videoImps)

	return reqs, errs
}

func (a *adapter) makeRequest(request *openrtb2.BidRequest, imps []openrtb2.Imp) (*adapters.RequestData, error) {
	if len(imps) == 0 {
		return nil, nil
	}
	ext, err := parseImpExt(&imps[0])
	if err != nil {
		return nil, err
	}

	request.Imp = imps
	body, err := jsonutil.Marshal(request)
	if err != nil {
		return nil, err
	}

	companyId := ext.CompanyId
	if request.Test == 1 {
		companyId = "test"
	}

	return &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     a.getEndpointURL(companyId),
		Body:    body,
		Headers: getHeaders(request),
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}, nil
}

func (a *adapter) getEndpointURL(companyId string) string {
	params := url.Values{}
	params.Add("company_id", companyId)
	return a.endpoint + "?" + params.Encode()
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(response) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
	}

	var resp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &resp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Bad Server Response",
		}}
	}

	var bidErrors []error
	bidderResponse := adapters.NewBidderResponseWithBidsCapacity(1)
	for i := range resp.SeatBid {
		seatBid := &resp.SeatBid[i]
		for j := range seatBid.Bid {
			bid := &seatBid.Bid[j]
			typedBid, err := getTypedBidFromBid(bid)
			if err != nil {
				bidErrors = append(bidErrors, err)
				continue
			}
			bidderResponse.Bids = append(bidderResponse.Bids, typedBid)
		}
	}

	return bidderResponse, bidErrors
}
