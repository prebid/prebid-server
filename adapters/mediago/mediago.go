package mediago

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
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
	EndpointTemplate *template.Template
}

type mediagoResponseBidExt struct {
	MediaType string `json:"mediaType"`
}

// Builder builds a new instance of the MediaGo adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	endpoint, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}
	bidder := &adapter{
		EndpointTemplate: endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
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

	mediagoExt, err := getMediaGoExt(request)
	if err != nil {
		return nil, err
	}
	endPoint, err := a.getEndPoint(mediagoExt)
	if err != nil {
		return nil, err
	}

	preProcess(request)
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("x-openrtb-version", "2.5")

	return &adapters.RequestData{
		Method:  "POST",
		Uri:     endPoint,
		Body:    reqBody,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}, nil
}

// getMediaGoExt get MediaGoExt From ext.bidderparams or ext of First Imp. Only check and get first Imp.Ext.Bidder to ExtImpMediago
func getMediaGoExt(request *openrtb2.BidRequest) (*openrtb_ext.ExtMediaGo, error) {
	var extMediaGo openrtb_ext.ExtMediaGo
	var extBidder adapters.ExtImpBidder

	// first get the mediago ext from ext.bidderparams
	reqExt := &openrtb_ext.ExtRequest{}
	err := jsonutil.Unmarshal(request.Ext, &reqExt)
	if err != nil {
		err = jsonutil.Unmarshal(reqExt.Prebid.BidderParams, &extMediaGo)
		if err != nil && extMediaGo.Token != "" {
			return &extMediaGo, nil
		}
	}

	// fallback to get token and region from first imp
	imp := request.Imp[0]
	err = jsonutil.Unmarshal(imp.Ext, &extBidder)
	if err != nil {
		return nil, err
	}

	var extImpMediaGo openrtb_ext.ExtImpMediaGo
	err = jsonutil.Unmarshal(extBidder.Bidder, &extImpMediaGo)
	if err != nil {
		return nil, err
	}
	if extImpMediaGo.Token != "" {
		extMediaGo.Token = extImpMediaGo.Token
		extMediaGo.Region = extImpMediaGo.Region

		return &extMediaGo, nil
	}
	return nil, errors.New("mediago token not found")

}

func getRegionInfo(region string) string {
	switch region {
	case "APAC":
		return "jp"
	case "EU":
		return "eu"
	case "US":
		return "us"
	default:
		return "us"
	}
}

func (a *adapter) getEndPoint(ext *openrtb_ext.ExtMediaGo) (string, error) {
	endPointParams := macros.EndpointTemplateParams{
		AccountID: url.PathEscape(ext.Token),
		Host:      url.PathEscape(getRegionInfo(ext.Region)),
	}
	return macros.ResolveMacros(a.EndpointTemplate, endPointParams)
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

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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
	var errs []error

	for _, seatBid := range bidResp.SeatBid {
		for idx := range seatBid.Bid {
			mediaType, err := getBidType(seatBid.Bid[idx], internalRequest.Imp)
			if err != nil {
				errs = append(errs, err)
			} else {
				bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
					Bid:     &seatBid.Bid[idx],
					BidType: mediaType,
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
				if imp.Banner != nil {
					return openrtb_ext.BidTypeBanner, nil
				}
				if imp.Native != nil {
					return openrtb_ext.BidTypeNative, nil
				}
			}
		}
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unsupported MType %d", bid.MType),
		}
	}

}
