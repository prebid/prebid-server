package synacormedia

import (
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"

	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type SynacorMediaAdapter struct {
	EndpointTemplate template.Template
}

type SyncEndpointTemplateParams struct {
	SeatId string
}

type ReqExt struct {
	SeatId string `json:"seatId"`
}

func NewSynacorMediaBidder(endpointTemplate string) adapters.Bidder {
	syncTemplate, err := template.New("endpointTemplate").Parse(endpointTemplate)
	if err != nil {
		glog.Fatal("Unable to parse endpoint url template")
		return nil
	}
	return &SynacorMediaAdapter{EndpointTemplate: *syncTemplate}
}

func (a *SynacorMediaAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	var bidRequests []*adapters.RequestData

	adapterReq, errors := a.makeRequest(request)
	if adapterReq != nil {
		bidRequests = append(bidRequests, adapterReq)
	}
	errs = append(errs, errors...)

	return bidRequests, errs
}

func (a *SynacorMediaAdapter) makeRequest(request *openrtb.BidRequest) (*adapters.RequestData, []error) {
	var errs []error
	var validImps []openrtb.Imp
	var re *ReqExt
	var firstExtImp *openrtb_ext.ExtImpSynacormedia = nil

	for _, imp := range request.Imp {
		validImp, err := getExtImpObj(&imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		validImps = append(validImps, imp)
		if firstExtImp == nil {
			firstExtImp = validImp
		}
	}

	if len(validImps) == 0 {
		return nil, errs
	}

	var err error

	if firstExtImp == nil || firstExtImp.SeatId == "" {
		return nil, append(errs, &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Impression missing seat id"),
		})
	}
	re = &ReqExt{SeatId: firstExtImp.SeatId}

	// create JSON Request Body
	request.Imp = validImps
	request.Ext, err = json.Marshal(re)
	if err != nil {
		return nil, append(errs, err)
	}

	reqJSON, err := json.Marshal(request)
	if err != nil {
		return nil, append(errs, err)
	}

	// set Request Headers
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	// create Request Uri
	reqUri, err := a.buildEndpointURL(firstExtImp)
	if err != nil {
		return nil, append(errs, err)
	}

	return &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     reqUri,
		Body:    reqJSON,
		Headers: headers,
	}, errs
}

// Builds enpoint url based on adapter-specific pub settings from imp.ext
func (adapter *SynacorMediaAdapter) buildEndpointURL(params *openrtb_ext.ExtImpSynacormedia) (string, error) {
	return macros.ResolveMacros(adapter.EndpointTemplate, macros.EndpointTemplateParams{Host: params.SeatId})
}

func getExtImpObj(imp *openrtb.Imp) (*openrtb_ext.ExtImpSynacormedia, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	var synacormediaExt openrtb_ext.ExtImpSynacormedia
	if err := json.Unmarshal(bidderExt.Bidder, &synacormediaExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	return &synacormediaExt, nil
}

// MakeBids make the bids for the bid response.
func (a *SynacorMediaAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	const errorMessage string = "Unexpected status code: %d. Run with request.debug = 1 for more info"
	switch {
	case response.StatusCode == http.StatusNoContent:
		return nil, nil
	case response.StatusCode == http.StatusBadRequest:
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf(errorMessage, response.StatusCode),
		}}
	case response.StatusCode != http.StatusOK:
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf(errorMessage, response.StatusCode),
		}}
	}

	var bidResp openrtb.BidResponse

	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			var mediaType = getMediaTypeForImp(sb.Bid[i].ImpID, internalRequest.Imp)
			if mediaType != openrtb_ext.BidTypeBanner && mediaType != openrtb_ext.BidTypeVideo {
				continue
			}
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: mediaType,
			})
		}
	}
	return bidResponse, nil
}

func getMediaTypeForImp(impId string, imps []openrtb.Imp) openrtb_ext.BidType {
	mediaType := openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Banner != nil {
				break
			}
			if imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
				break
			}
			if imp.Native != nil {
				mediaType = openrtb_ext.BidTypeNative
				break
			}
			if imp.Audio != nil {
				mediaType = openrtb_ext.BidTypeAudio
				break
			}
		}
	}
	return mediaType
}
