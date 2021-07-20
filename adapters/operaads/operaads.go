package operaads

import (
	"encoding/json"
	"fmt"
	"github.com/prebid/prebid-server/macros"
	"net/http"
	"text/template"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type adapter struct {
	epTemplate *template.Template
}

// Builder builds a new instance of the operaads adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	epTemplate, err := template.New("endpoint").Parse(config.Endpoint)
	if err != nil {
		return nil, err
	}
	bidder := &adapter{
		epTemplate: epTemplate,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	impCount := len(request.Imp)
	requestData := make([]*adapters.RequestData, 0, impCount)
	errs := []error{}
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	err := checkRequest(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	for _, imp := range request.Imp {
		requestCopy := *request
		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		var operaadsExt openrtb_ext.ImpExtOperaads
		if err := json.Unmarshal(bidderExt.Bidder, &operaadsExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		err := convertImpression(&imp)
		if err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		imp.TagID = operaadsExt.PlacementID

		requestCopy.Imp = []openrtb2.Imp{imp}
		reqJSON, err := json.Marshal(&requestCopy)
		if err != nil {
			errs = append(errs, err)
			return nil, errs
		}

		macro := macros.EndpointTemplateParams{PublisherID: operaadsExt.PublisherID, AccountID: operaadsExt.EndpointID}
		endpoint, err := macros.ResolveMacros(*a.epTemplate, &macro)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		reqData := &adapters.RequestData{
			Method:  http.MethodPost,
			Uri:     endpoint,
			Body:    reqJSON,
			Headers: headers,
		}
		requestData = append(requestData, reqData)
	}
	return requestData, errs
}

func checkRequest(request *openrtb2.BidRequest) error {
	if request.Device == nil || len(request.Device.OS) == 0 {
		return &errortypes.BadInput{
			Message: "Impression is missing device OS information",
		}
	}

	return nil
}

func convertImpression(imp *openrtb2.Imp) error {
	if imp.Banner != nil {
		bannerCopy, err := convertBanner(imp.Banner)
		if err != nil {
			return err
		}
		imp.Banner = bannerCopy
	}
	if imp.Native != nil && imp.Native.Request != "" {
		v := make(map[string]interface{})
		err := json.Unmarshal([]byte(imp.Native.Request), &v)
		if err != nil {
			return err
		}
		_, ok := v["native"]
		if !ok {
			body, err := json.Marshal(struct {
				Native interface{} `json:"native"`
			}{
				Native: v,
			})
			if err != nil {
				return err
			}
			native := *imp.Native
			native.Request = string(body)
			imp.Native = &native
		}
	}
	return nil
}

// make sure that banner has openrtb 2.3-compatible size information
func convertBanner(banner *openrtb2.Banner) (*openrtb2.Banner, error) {
	if banner.W == nil || banner.H == nil || *banner.W == 0 || *banner.H == 0 {
		if len(banner.Format) > 0 {
			f := banner.Format[0]

			bannerCopy := *banner

			bannerCopy.W = openrtb2.Int64Ptr(f.W)
			bannerCopy.H = openrtb2.Int64Ptr(f.H)

			return &bannerCopy, nil
		} else {
			return nil, &errortypes.BadInput{
				Message: "Size information missing for banner",
			}
		}
	}
	return banner, nil
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var parsedResponse openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &parsedResponse); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: err.Error(),
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)

	for _, sb := range parsedResponse.SeatBid {
		for i := 0; i < len(sb.Bid); i++ {
			bid := sb.Bid[i]
			if bid.Price != 0 {
				bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
					Bid:     &bid,
					BidType: getMediaTypeForImp(bid.ImpID, internalRequest.Imp),
				})
			}
		}
	}
	return bidResponse, nil
}

func getMediaTypeForImp(impId string, imps []openrtb2.Imp) openrtb_ext.BidType {
	mediaType := openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			} else if imp.Native != nil {
				mediaType = openrtb_ext.BidTypeNative
			}
			return mediaType
		}
	}
	return mediaType
}
