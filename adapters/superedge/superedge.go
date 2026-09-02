package superedge

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/macros"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

type adapter struct {
	EndpointTemplate *template.Template
}

// Builder builds a new instance of the SuperEdge adapter for the given bidder with the given config.
func Builder(_ openrtb_ext.BidderName, config config.Adapter, _ config.Server) (adapters.Bidder, error) {
	endpoint, err := template.New("").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}
	bidder := &adapter{EndpointTemplate: endpoint}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var adapterRequests []*adapters.RequestData
	var errs []error
	adapterRequest, err := a.makeRequest(request)
	if err == nil {
		adapterRequests = append(adapterRequests, adapterRequest)
	} else {
		errs = append(errs, err)
	}
	return adapterRequests, errs
}

func (a *adapter) makeRequest(request *openrtb2.BidRequest) (*adapters.RequestData, error) {
	superEdgeExt, err := getSuperEdgeExt(request)
	if err != nil {
		return nil, err
	}
	endPoint, err := a.getEndPoint(superEdgeExt)
	if err != nil {
		return nil, err
	}
	requestCopy := *request
	requestCopy.Imp = make([]openrtb2.Imp, len(request.Imp))
	copy(requestCopy.Imp, request.Imp)
	preProcess(&requestCopy)
	reqBody, err := jsonutil.Marshal(&requestCopy)
	if err != nil {
		return nil, err
	}
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("x-openrtb-version", "2.6")
	return &adapters.RequestData{
		Method:  "POST",
		Uri:     endPoint,
		Body:    reqBody,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}, nil
}

// getSuperEdgeExt extracts ExtSuperEdge from the first imp's ext.bidder.
func getSuperEdgeExt(request *openrtb2.BidRequest) (*openrtb_ext.ExtSuperEdge, error) {
	var extSuperEdge openrtb_ext.ExtSuperEdge

	if len(request.Imp) == 0 {
		return nil, errors.New("superEdge sk not found")
	}

	var extBidder adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(request.Imp[0].Ext, &extBidder); err != nil {
		return nil, err
	}

	if err := jsonutil.Unmarshal(extBidder.Bidder, &extSuperEdge); err != nil {
		return nil, err
	}

	if extSuperEdge.Sk == "" {
		return nil, errors.New("superEdge sk not found")
	}
	return &extSuperEdge, nil
}

func (a *adapter) getEndPoint(ext *openrtb_ext.ExtSuperEdge) (string, error) {
	endPointParams := macros.EndpointTemplateParams{
		Host:      url.PathEscape(getRegionHost(ext.Region)),
		AccountID: url.PathEscape(ext.Sk),
	}
	return macros.ResolveMacros(a.EndpointTemplate, endPointParams)
}

func getRegionHost(region string) string {
	switch region {
	case "EU":
		return "rtb-eu"
	case "APAC":
		return "rtb-sg"
	default:
		return "rtb-us"
	}
}

func preProcess(request *openrtb2.BidRequest) {
	for i := range request.Imp {
		if request.Imp[i].Banner != nil {
			banner := *request.Imp[i].Banner
			if (banner.W == nil || banner.H == nil || *banner.W == 0 || *banner.H == 0) && len(banner.Format) > 0 {
				firstFormat := banner.Format[0]
				banner.W = &firstFormat.W
				banner.H = &firstFormat.H
				request.Imp[i].Banner = &banner
			}
		}
	}
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, _ *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(response) {
		return nil, nil
	}
	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
	}
	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)
	if bidResp.Cur != "" {
		bidResponse.Currency = bidResp.Cur
	}
	var errs []error
	for _, seatBid := range bidResp.SeatBid {
		for idx := range seatBid.Bid {
			bidType, err := getBidType(seatBid.Bid[idx], internalRequest.Imp)
			if err != nil {
				errs = append(errs, err)
			} else {
				bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
					Bid:     &seatBid.Bid[idx],
					BidType: bidType,
				})
			}
		}
	}
	return bidResponse, errs
}

func getBidType(bid openrtb2.Bid, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	default:
		for _, imp := range imps {
			if imp.ID == bid.ImpID {
				if imp.Banner != nil && imp.Native == nil {
					return openrtb_ext.BidTypeBanner, nil
				}
				if imp.Native != nil && imp.Banner == nil {
					return openrtb_ext.BidTypeNative, nil
				}
			}
		}
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unsupported MType %d", bid.MType),
		}
	}
}
