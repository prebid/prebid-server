package operaads

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"text/template"

	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/util/ptrutil"

	"github.com/prebid/openrtb/v20/openrtb2"
)

type adapter struct {
	epTemplate *template.Template
}

var (
	errBannerFormatMiss = errors.New("Size information missing for banner")
	errDeviceOrOSMiss   = errors.New("Impression is missing device OS information")
)

// Builder builds a new instance of the operaads adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
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
		errs = append(errs, &errortypes.BadInput{
			Message: err.Error(),
		})
		return nil, errs
	}

	for _, imp := range request.Imp {
		var bidderExt adapters.ExtImpBidder
		if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}
		var operaadsExt openrtb_ext.ImpExtOperaads
		if err := jsonutil.Unmarshal(bidderExt.Bidder, &operaadsExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}
		macro := macros.EndpointTemplateParams{PublisherID: operaadsExt.PublisherID, AccountID: operaadsExt.EndpointID}
		endpoint, err := macros.ResolveMacros(a.epTemplate, &macro)
		if err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}
		imp.TagID = operaadsExt.PlacementID
		formats := make([]interface{}, 0, 1)
		if imp.Native != nil {
			formats = append(formats, imp.Native)
		}
		if imp.Video != nil {
			formats = append(formats, imp.Video)
		}
		if imp.Banner != nil {
			formats = append(formats, imp.Banner)
		}
		for _, format := range formats {
			req, err := flatImp(*request, imp, headers, endpoint, format)
			if err != nil {
				errs = append(errs, &errortypes.BadInput{
					Message: err.Error(),
				})
				continue
			}
			if req != nil {
				requestData = append(requestData, req)
			}
		}
	}
	return requestData, errs
}

func flatImp(requestCopy openrtb2.BidRequest, impCopy openrtb2.Imp, headers http.Header, endpoint string, format interface{}) (*adapters.RequestData, error) {
	switch format.(type) {
	case *openrtb2.Video:
		impCopy.Native = nil
		impCopy.Banner = nil
		impCopy.ID = buildOperaImpId(impCopy.ID, openrtb_ext.BidTypeVideo)
	case *openrtb2.Banner:
		impCopy.Video = nil
		impCopy.Native = nil
		impCopy.ID = buildOperaImpId(impCopy.ID, openrtb_ext.BidTypeBanner)
	case *openrtb2.Native:
		impCopy.Video = nil
		impCopy.Banner = nil
		impCopy.ID = buildOperaImpId(impCopy.ID, openrtb_ext.BidTypeNative)
	default: // do not need flat
		return nil, nil
	}
	err := convertImpression(&impCopy)
	if err != nil {
		return nil, err
	}
	requestCopy.Imp = []openrtb2.Imp{impCopy}
	reqJSON, err := json.Marshal(&requestCopy)
	if err != nil {
		return nil, err
	}
	return &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     endpoint,
		Body:    reqJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(requestCopy.Imp),
	}, nil
}

func checkRequest(request *openrtb2.BidRequest) error {
	if request.Device == nil || len(request.Device.OS) == 0 {
		return errDeviceOrOSMiss
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
		err := jsonutil.Unmarshal([]byte(imp.Native.Request), &v)
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
			bannerCopy.W = ptrutil.ToPtr(f.W)
			bannerCopy.H = ptrutil.ToPtr(f.H)
			return &bannerCopy, nil
		} else {
			return nil, errBannerFormatMiss
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
	if err := jsonutil.Unmarshal(response.Body, &parsedResponse); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: err.Error(),
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)
	for _, sb := range parsedResponse.SeatBid {
		for i := 0; i < len(sb.Bid); i++ {
			bid := sb.Bid[i]
			if bid.Price != 0 {
				var bidType openrtb_ext.BidType
				bid.ImpID, bidType = parseOriginImpId(bid.ImpID)
				bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
					Bid:     &bid,
					BidType: bidType,
				})
			}
		}
	}
	return bidResponse, nil
}

func buildOperaImpId(originId string, bidType openrtb_ext.BidType) string {
	return strings.Join([]string{originId, "opa", string(bidType)}, ":")
}

func parseOriginImpId(impId string) (originId string, bidType openrtb_ext.BidType) {
	items := strings.Split(impId, ":")
	if len(items) < 2 {
		return impId, ""
	}
	return strings.Join(items[:len(items)-2], ":"), openrtb_ext.BidType(items[len(items)-1])
}
