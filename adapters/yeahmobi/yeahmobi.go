package yeahmobi

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
type yeahmobiBidExt struct {
	VideoCreativeInfo *yeahmobiBidExtVideo `json:"video,omitempty"`
}
type yeahmobiBidExtVideo struct {
	Duration *int `json:"duration,omitempty"`
}

// Builder builds a new instance of the Yeahmobi adapter for the given bidder with the given config.
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
	var adapterRequests []*adapters.RequestData

	adapterRequest, errs := a.makeRequest(request)
	if errs == nil {
		adapterRequests = append(adapterRequests, adapterRequest)
	}

	return adapterRequests, errs
}

func (a *adapter) makeRequest(request *openrtb2.BidRequest) (*adapters.RequestData, []error) {
	var errs []error

	yeahmobiExt, errs := getYeahmobiExt(request)

	if yeahmobiExt == nil {
		return nil, errs
	}
	endPoint, err := a.getEndpoint(yeahmobiExt)
	if err != nil {
		return nil, append(errs, err)
	}
	transform(request)
	reqBody, err := json.Marshal(request)

	if err != nil {
		return nil, append(errs, err)
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	return &adapters.RequestData{
		Method:  "POST",
		Uri:     endPoint,
		Body:    reqBody,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}, errs
}

func transform(request *openrtb2.BidRequest) {
	for i, imp := range request.Imp {
		if imp.Native != nil {
			var nativeRequest map[string]interface{}
			nativeCopyRequest := make(map[string]interface{})
			err := jsonutil.Unmarshal([]byte(request.Imp[i].Native.Request), &nativeRequest)
			//just ignore the bad native request
			if err == nil {
				_, exists := nativeRequest["native"]
				if exists {
					continue
				}

				nativeCopyRequest["native"] = nativeRequest
				nativeReqByte, err := json.Marshal(nativeCopyRequest)
				//just ignore the bad native request
				if err != nil {
					continue
				}

				nativeCopy := *request.Imp[i].Native
				nativeCopy.Request = string(nativeReqByte)
				request.Imp[i].Native = &nativeCopy
			}
		}
	}
}

func getYeahmobiExt(request *openrtb2.BidRequest) (*openrtb_ext.ExtImpYeahmobi, []error) {
	var extImpYeahmobi openrtb_ext.ExtImpYeahmobi
	var errs []error

	for _, imp := range request.Imp {
		var extBidder adapters.ExtImpBidder
		err := jsonutil.Unmarshal(imp.Ext, &extBidder)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		err = jsonutil.Unmarshal(extBidder.Bidder, &extImpYeahmobi)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		break
	}

	return &extImpYeahmobi, errs

}

func (a *adapter) getEndpoint(ext *openrtb_ext.ExtImpYeahmobi) (string, error) {
	return macros.ResolveMacros(a.EndpointTemplate, macros.EndpointTemplateParams{Host: "gw-" + url.QueryEscape(ext.ZoneId) + "-bid.yeahtargeter.com"})
}

// MakeBids make the bids for the bid response.
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
	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			var mediaType = getBidType(sb.Bid[i].ImpID, internalRequest.Imp)
			bid := sb.Bid[i]
			typedBid := &adapters.TypedBid{
				Bid:      &bid,
				BidType:  mediaType,
				BidVideo: &openrtb_ext.ExtBidPrebidVideo{},
			}
			if bid.Ext != nil {
				var bidExt *yeahmobiBidExt
				err := jsonutil.Unmarshal(bid.Ext, &bidExt)
				if err != nil {
					return nil, []error{fmt.Errorf("bid.ext json unmarshal error")}
				} else if bidExt != nil {
					if bidExt.VideoCreativeInfo != nil && bidExt.VideoCreativeInfo.Duration != nil {
						typedBid.BidVideo.Duration = *bidExt.VideoCreativeInfo.Duration
					}
				}
			}
			bidResponse.Bids = append(bidResponse.Bids, typedBid)
		}
	}
	return bidResponse, nil

}

func getBidType(impId string, imps []openrtb2.Imp) openrtb_ext.BidType {
	bidType := openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Banner != nil {
				break
			}
			if imp.Video != nil {
				bidType = openrtb_ext.BidTypeVideo
				break
			}
			if imp.Native != nil {
				bidType = openrtb_ext.BidTypeNative
				break
			}

		}
	}
	return bidType
}
