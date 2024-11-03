package imds

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

const adapterVersion string = "pbs-go/1.0.0"

type adapter struct {
	EndpointTemplate *template.Template
}

type SyncEndpointTemplateParams struct {
	SeatId string
	TagId  string
}

type ReqExt struct {
	SeatId string `json:"seatId"`
}

// Builder builds a new instance of the Imds adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	template, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &adapter{
		EndpointTemplate: template,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	var bidRequests []*adapters.RequestData

	adapterReq, errors := a.makeRequest(request)
	if adapterReq != nil {
		bidRequests = append(bidRequests, adapterReq)
	}
	errs = append(errs, errors...)

	return bidRequests, errs
}

func (a *adapter) makeRequest(request *openrtb2.BidRequest) (*adapters.RequestData, []error) {
	var errs []error
	var validImps []openrtb2.Imp
	var re *ReqExt
	var firstExtImp *openrtb_ext.ExtImpImds = nil

	for _, imp := range request.Imp {
		validExtImpObj, err := getExtImpObj(&imp) // getExtImpObj returns {seatId:"", tagId:""}
		if err != nil {
			errs = append(errs, err)
			continue
		}
		// if the bid request is missing seatId or TagId then ignore it
		if validExtImpObj.SeatId == "" || validExtImpObj.TagId == "" {
			errs = append(errs, &errortypes.BadServerResponse{
				Message: "Invalid Impression",
			})
			continue
		}
		// right here is where we need to take out the tagId and then add it to imp
		imp.TagID = validExtImpObj.TagId
		validImps = append(validImps, imp)
		if firstExtImp == nil {
			firstExtImp = validExtImpObj
		}
	}

	if len(validImps) == 0 {
		return nil, errs
	}

	var err error

	if firstExtImp == nil || firstExtImp.SeatId == "" || firstExtImp.TagId == "" {
		return nil, append(errs, &errortypes.BadServerResponse{
			Message: "Invalid Impression",
		})
	}
	// this is where the empty seatId is filled
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
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}, errs
}

// Builds enpoint url based on adapter-specific pub settings from imp.ext
func (adapter *adapter) buildEndpointURL(params *openrtb_ext.ExtImpImds) (string, error) {
	return macros.ResolveMacros(adapter.EndpointTemplate, macros.EndpointTemplateParams{AccountID: url.QueryEscape(params.SeatId), SourceId: url.QueryEscape(adapterVersion)})
}

func getExtImpObj(imp *openrtb2.Imp) (*openrtb_ext.ExtImpImds, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	var imdsExt openrtb_ext.ExtImpImds
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &imdsExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	return &imdsExt, nil
}

// MakeBids make the bids for the bid response.
func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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

	var bidResp openrtb2.BidResponse

	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
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

func getMediaTypeForImp(impId string, imps []openrtb2.Imp) openrtb_ext.BidType {
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
