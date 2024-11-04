package algorix

import (
	"encoding/json"
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

type algorixVideoExt struct {
	Rewarded int `json:"rewarded"`
}

type algorixResponseBidExt struct {
	MediaType string `json:"mediaType"`
}

// Builder builds a new instance of the AlgoriX adapter for the given bidder with the given config.
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
	algorixExt, err := getImpAlgoriXExt(&request.Imp[0])

	if err != nil {
		return nil, &errortypes.BadInput{Message: "Invalid ExtImpAlgoriX value"}
	}

	endPoint, err := a.getEndPoint(algorixExt)
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

// get ImpAlgoriXExt From First Imp. Only check and get first Imp.Ext.Bidder to ExtImpAlgorix
func getImpAlgoriXExt(imp *openrtb2.Imp) (*openrtb_ext.ExtImpAlgorix, error) {
	var extImpAlgoriX openrtb_ext.ExtImpAlgorix
	var extBidder adapters.ExtImpBidder
	err := jsonutil.Unmarshal(imp.Ext, &extBidder)
	if err != nil {
		return nil, err
	}
	err = jsonutil.Unmarshal(extBidder.Bidder, &extImpAlgoriX)
	if err != nil {
		return nil, err
	}
	return &extImpAlgoriX, nil
}

func getRegionInfo(region string) string {
	switch region {
	case "APAC":
		return "apac.xyz"
	case "USE":
		return "use.xyz"
	case "EUC":
		return "euc.xyz"
	default:
		return "xyz"
	}
}

func (a *adapter) getEndPoint(ext *openrtb_ext.ExtImpAlgorix) (string, error) {
	endPointParams := macros.EndpointTemplateParams{
		SourceId:  url.PathEscape(ext.Sid),
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
		if request.Imp[i].Video != nil {
			var impExt adapters.ExtImpBidder
			err := jsonutil.Unmarshal(request.Imp[i].Ext, &impExt)
			if err != nil {
				continue
			}
			if impExt.Prebid != nil && impExt.Prebid.IsRewardedInventory != nil && *impExt.Prebid.IsRewardedInventory == 1 {
				videoCopy := *request.Imp[i].Video
				videoExt := algorixVideoExt{Rewarded: 1}
				videoCopy.Ext, err = json.Marshal(&videoExt)
				if err != nil {
					continue
				}
				request.Imp[i].Video = &videoCopy
			}
		}
	}
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d.", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d.", response.StatusCode),
		}}
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
	var bidExt algorixResponseBidExt
	err := jsonutil.Unmarshal(bid.Ext, &bidExt)
	if err == nil {
		switch bidExt.MediaType {
		case "banner":
			return openrtb_ext.BidTypeBanner, nil
		case "native":
			return openrtb_ext.BidTypeNative, nil
		case "video":
			return openrtb_ext.BidTypeVideo, nil
		}
	}
	var mediaType openrtb_ext.BidType
	var typeCnt = 0
	for _, imp := range imps {
		if imp.ID == bid.ImpID {
			if imp.Banner != nil {
				typeCnt += 1
				mediaType = openrtb_ext.BidTypeBanner
			}
			if imp.Native != nil {
				typeCnt += 1
				mediaType = openrtb_ext.BidTypeNative
			}
			if imp.Video != nil {
				typeCnt += 1
				mediaType = openrtb_ext.BidTypeVideo
			}
		}
	}
	if typeCnt == 1 {
		return mediaType, nil
	}
	return mediaType, fmt.Errorf("unable to fetch mediaType in multi-format: %s", bid.ImpID)
}
