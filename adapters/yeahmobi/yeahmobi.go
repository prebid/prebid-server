package yeahmobi

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
	"net/url"
	"text/template"
)

type YeahmobiAdapter struct {
	EndpointTemplate template.Template
}

func NewYeahmobiBidder(endpointTemplate string) adapters.Bidder {
	tpl, err := template.New("endpointTemplate").Parse(endpointTemplate)
	if err != nil {
		glog.Fatal("Unable parse url template err:" + err.Error())
		return nil
	}
	return &YeahmobiAdapter{EndpointTemplate: *tpl}
}

func (adapter *YeahmobiAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var adapterRequests []*adapters.RequestData

	adapterRequest, errs := adapter.makeRequest(request)
	if errs == nil {
		adapterRequests = append(adapterRequests, adapterRequest)
	}

	return adapterRequests, errs
}

func (adapter *YeahmobiAdapter) makeRequest(request *openrtb.BidRequest) (*adapters.RequestData, []error) {
	var errs []error

	yeahmobiExt, errs := getYeahmobiExt(request)

	if yeahmobiExt == nil {
		glog.Fatal("Invalid ExtImpYeahmobi value")
		return nil, errs
	}
	endPoint, err := adapter.getEndpoint(yeahmobiExt)
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
	}, errs
}

func transform(request *openrtb.BidRequest) {
	for i, imp := range request.Imp {
		if imp.Native != nil {
			var nativeRequest map[string]interface{}
			err := json.Unmarshal([]byte(request.Imp[i].Native.Request), &nativeRequest)
			if err == nil {
				_, exists := nativeRequest["native"]
				if exists {
					continue
				}
				nativeCopy := *request.Imp[i].Native
				nativeCopy.Request = "{\"native\":" + request.Imp[i].Native.Request + "}"
				request.Imp[i].Native = &nativeCopy
			}
		}
	}
}

func getYeahmobiExt(request *openrtb.BidRequest) (*openrtb_ext.ExtImpYeahmobi, []error) {
	var extImpYeahmobi openrtb_ext.ExtImpYeahmobi
	var errs []error

	for _, imp := range request.Imp {
		var extBidder adapters.ExtImpBidder
		err := json.Unmarshal(imp.Ext, &extBidder)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		err = json.Unmarshal(extBidder.Bidder, &extImpYeahmobi)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		break
	}

	return &extImpYeahmobi, errs

}

func (adapter *YeahmobiAdapter) getEndpoint(ext *openrtb_ext.ExtImpYeahmobi) (string, error) {
	return macros.ResolveMacros(adapter.EndpointTemplate, macros.EndpointTemplateParams{Host: "gw-" + url.QueryEscape(ext.ZoneId) + "-bid.yeahtargeter.com"})
}

// MakeBids make the bids for the bid response.
func (a *YeahmobiAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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

	var bidResp openrtb.BidResponse

	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			var mediaType = getBidType(sb.Bid[i].ImpID, internalRequest.Imp)
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: mediaType,
			})
		}
	}
	return bidResponse, nil

}

func getBidType(impId string, imps []openrtb.Imp) openrtb_ext.BidType {
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
