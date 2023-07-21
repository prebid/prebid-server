package silverpush

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const (
	bidderConfig  = "sp_pb_ortb"
	bidderVersion = "1.0.0"
)

type SilverPushAdapter struct {
	bidderName string
	endpoint   string
}

type SilverPushImpExt map[string]json.RawMessage

type SilverPushReqExt struct {
	PublisherId string  `json:"publisherId"`
	BidFloor    float64 `json:"bidfloor"`
}

func (a *SilverPushAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {

	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{Message: "No impressions in bid request."}}
	}

	return a.ValidateAndProcessRequest(request)
}

// Validate and Process the request return []*adapters.RequestData and []error
func (a *SilverPushAdapter) ValidateAndProcessRequest(request *openrtb2.BidRequest) ([]*adapters.RequestData, []error) {
	imps := request.Imp
	requests := make([]*adapters.RequestData, 0, len(imps))
	errors := make([]error, 0, len(imps))

	for _, imp := range imps {
		impsByMediaType, err := impressionByMediaType(&imp)
		if err != nil {
			errors = append(errors, err)
		}

		for _, impByMediaType := range impsByMediaType {
			request.Imp = []openrtb2.Imp{impByMediaType}

			if err := validateRequest(request); err != nil {
				errors = append(errors, err)
				continue
			}
			requestData, err := a.makeRequest(request)
			if err != nil {
				errors = append(errors, err)
				continue
			}

			requests = append(requests, requestData)

		}

	}

	return requests, errors

}

func (a *SilverPushAdapter) makeRequest(req *openrtb2.BidRequest) (*adapters.RequestData, error) {
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("X-Openrtb-Version", "2.5")
	return &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    reqJSON,
		Headers: headers,
	}, nil
}

func validateRequest(req *openrtb2.BidRequest) error {
	imp := &req.Imp[0]
	var silverPushExt openrtb_ext.ImpExtSilverpush
	err := setPublisherId(req, imp, &silverPushExt)
	if err != nil {
		return err
	}
	if err := setUser(req); err != nil {
		return err
	}
	setDevice(req)
	setExtToRequest(req, silverPushExt.PublisherId)
	return setImpForAdExchange(imp, &silverPushExt)
}

func setDevice(req *openrtb2.BidRequest) {
	if req.Device != nil {
		deviceCopy := *req.Device
		if len(deviceCopy.UA) > 0 {
			deviceCopy.OS = getOS(deviceCopy.UA)
			if isMobile(deviceCopy.UA) {
				deviceCopy.DeviceType = 1
			} else if isCTV(deviceCopy.UA) {
				deviceCopy.DeviceType = 3
			} else {
				deviceCopy.DeviceType = 2
			}
		}
		req.Device = &deviceCopy
	}
}

func setUser(req *openrtb2.BidRequest) error {
	var extUser openrtb_ext.ExtUser
	if req.User != nil && req.User.Ext != nil {
		var userCopy = *req.User
		if err := json.Unmarshal(req.User.Ext, &extUser); err != nil {
			return &errortypes.BadInput{Message: "Invalid user.ext."}
		}
		if IsValidEids(extUser.Eids) {
			req.User = &userCopy
		}
	}
	return nil
}

func setExtToRequest(req *openrtb2.BidRequest, publisherID string) {

	record := map[string]string{
		"bc":          bidderConfig + "_" + bidderVersion,
		"publisherId": publisherID,
	}
	reqExt, _ := json.Marshal(record)
	req.Ext = reqExt

}
func setImpForAdExchange(imp *openrtb2.Imp, impExt *openrtb_ext.ImpExtSilverpush) error {

	if imp.BidFloor == 0 && impExt.BidFloor > 0 {
		imp.BidFloor = impExt.BidFloor
	}
	if imp.Banner != nil {
		bannerCopy, err := setBannerDimension(imp.Banner)
		if err != nil {
			return err
		}
		imp.Banner = bannerCopy
	}

	if imp.Video != nil {
		videoCopy, err := checkVideoDimension(imp.Video)
		if err != nil {
			return err
		}
		imp.Video = videoCopy
	}

	return nil
}

func checkVideoDimension(video *openrtb2.Video) (*openrtb2.Video, error) {
	videoCopy := *video
	if videoCopy.MaxDuration == 0 {
		videoCopy.MaxDuration = 120
	}
	if videoCopy.MaxDuration < videoCopy.MinDuration {
		videoCopy.MaxDuration = videoCopy.MinDuration
		videoCopy.MinDuration = 0
	}
	if videoCopy.API == nil || videoCopy.MIMEs == nil || videoCopy.Protocols == nil || videoCopy.MinDuration < 0 {
		return nil, &errortypes.BadInput{Message: "Invalid or missing video field(s)"}
	}
	return &videoCopy, nil
}

func setBannerDimension(banner *openrtb2.Banner) (*openrtb2.Banner, error) {
	if banner.W != nil && banner.H != nil {
		return banner, nil
	}
	if len(banner.Format) == 0 {
		return banner, &errortypes.BadInput{Message: "No sizes provided for Banner."}
	}
	bannerCopy := *banner
	bannerCopy.W = openrtb2.Int64Ptr(banner.Format[0].W)
	bannerCopy.H = openrtb2.Int64Ptr(banner.Format[0].H)

	return &bannerCopy, nil
}
func setPublisherId(req *openrtb2.BidRequest, imp *openrtb2.Imp, impExt *openrtb_ext.ImpExtSilverpush) error {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	if err := json.Unmarshal(bidderExt.Bidder, impExt); err != nil {
		return &errortypes.BadInput{
			Message: err.Error(),
		}
	}
	if impExt.PublisherId == "" {
		return &errortypes.BadInput{Message: "Missing publisherId parameter."}
	}
	if req.Site != nil {
		siteCopy := *req.Site
		if siteCopy.Publisher == nil {
			siteCopy.Publisher = &openrtb2.Publisher{ID: impExt.PublisherId}
		} else {
			publisher := *siteCopy.Publisher
			publisher.ID = impExt.PublisherId
			siteCopy.Publisher = &publisher
		}
		req.Site = &siteCopy

	} else if req.App != nil {
		appCopy := *req.App
		if appCopy.Publisher == nil {
			appCopy.Publisher = &openrtb2.Publisher{ID: impExt.PublisherId}
		} else {
			publisher := *appCopy.Publisher
			publisher.ID = impExt.PublisherId
			appCopy.Publisher = &publisher
		}
		appCopy.Publisher = &openrtb2.Publisher{ID: impExt.PublisherId}
		req.App = &appCopy

	}

	return nil
}

func impressionByMediaType(imp *openrtb2.Imp) ([]openrtb2.Imp, error) {

	if imp.Banner == nil && imp.Video == nil {
		return nil, &errortypes.BadInput{Message: "Invalid MediaType. SilverPush only supports Banner, Video"}
	}
	imps := make([]openrtb2.Imp, 0, 2)

	if imp.Banner != nil {
		impCopy := *imp
		impCopy.Video = nil
		imps = append(imps, impCopy)
	}
	if imp.Video != nil {
		impCopy := *imp
		impCopy.Banner = nil
		imps = append(imps, impCopy)
	}
	return imps, nil
}

func (a *SilverPushAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		fmt.Println(response.StatusCode)
		return nil, nil
	}
	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode)}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode)}}
	}

	var bidResp openrtb2.BidResponse

	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(internalRequest.Imp))

	// overrride default currency
	if bidResp.Cur != "" {
		bidResponse.Currency = bidResp.Cur
	}

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: getMediaTypeForImp(sb.Bid[i].ImpID, internalRequest.Imp),
			})
		}
	}
	return bidResponse, nil
}

// getMediaTypeForImp figures out which media type this bid is for.
// SilverPush doesn't support multi-type impressions.
// If both banner and video exist, take banner as we do not want in-banner video.
func getMediaTypeForImp(impId string, imps []openrtb2.Imp) (mediaType openrtb_ext.BidType) {

	mediaType = openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Banner != nil {
				mediaType = openrtb_ext.BidTypeBanner
			} else if imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			}
		}
	}
	return
}

// Builder builds a new instance of the silverpush adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &SilverPushAdapter{
		endpoint:   config.Endpoint,
		bidderName: string(bidderName),
	}
	return bidder, nil
}
