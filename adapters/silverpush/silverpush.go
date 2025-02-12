package silverpush

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
)

const (
	bidderConfig  = "sp_pb_ortb"
	bidderVersion = "1.0.0"
)

type adapter struct {
	bidderName string
	endpoint   string
}

type SilverPushImpExt map[string]json.RawMessage

type SilverPushReqExt struct {
	PublisherId string  `json:"publisherId"`
	BidFloor    float64 `json:"bidfloor"`
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	imps := request.Imp
	var errors []error
	requests := make([]*adapters.RequestData, 0, len(imps))

	for _, imp := range imps {
		impsByMediaType := impressionByMediaType(&imp)

		request.Imp = []openrtb2.Imp{impsByMediaType}

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
	return requests, errors
}

func (a *adapter) makeRequest(req *openrtb2.BidRequest) (*adapters.RequestData, error) {
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
		ImpIDs:  openrtb_ext.GetImpIDs(req.Imp),
	}, nil
}

func validateRequest(req *openrtb2.BidRequest) error {
	imp := &req.Imp[0]
	var silverPushExt openrtb_ext.ImpExtSilverpush
	if err := setPublisherId(req, imp, &silverPushExt); err != nil {
		return err
	}
	if err := setUser(req); err != nil {
		return err
	}

	setDevice(req)

	if err := setExtToRequest(req, silverPushExt.PublisherId); err != nil {
		return err
	}
	return setImpForAdExchange(imp, &silverPushExt)
}

func setDevice(req *openrtb2.BidRequest) {
	if req.Device == nil {
		return
	}
	deviceCopy := *req.Device
	if len(deviceCopy.UA) == 0 {
		return
	}
	deviceCopy.OS = getOS(deviceCopy.UA)
	if isMobile(deviceCopy.UA) {
		deviceCopy.DeviceType = 1
	} else if isCTV(deviceCopy.UA) {
		deviceCopy.DeviceType = 3
	} else {
		deviceCopy.DeviceType = 2
	}

	req.Device = &deviceCopy
}

func setUser(req *openrtb2.BidRequest) error {
	var extUser openrtb_ext.ExtUser
	var userExtRaw map[string]json.RawMessage

	if req.User != nil && req.User.Ext != nil {
		if err := jsonutil.Unmarshal(req.User.Ext, &userExtRaw); err != nil {
			return &errortypes.BadInput{Message: "Invalid user.ext."}
		}
		if userExtDataRaw, ok := userExtRaw["data"]; ok {
			if err := jsonutil.Unmarshal(userExtDataRaw, &extUser); err != nil {
				return &errortypes.BadInput{Message: "Invalid user.ext.data."}
			}
			var userCopy = *req.User
			if isValidEids(extUser.Eids) {
				userExt, err := json.Marshal(
					&openrtb2.User{
						EIDs: extUser.Eids,
					})
				if err != nil {
					return &errortypes.BadInput{Message: "Error in marshaling user.eids."}
				}

				userCopy.Ext = userExt
				req.User = &userCopy
			}
		}
	}
	return nil
}

func setExtToRequest(req *openrtb2.BidRequest, publisherID string) error {
	record := map[string]string{
		"bc":          bidderConfig + "_" + bidderVersion,
		"publisherId": publisherID,
	}
	reqExt, err := json.Marshal(record)
	if err != nil {
		return err
	}
	req.Ext = reqExt
	return nil
}

func setImpForAdExchange(imp *openrtb2.Imp, impExt *openrtb_ext.ImpExtSilverpush) error {
	if impExt.BidFloor == 0 {
		if imp.Banner != nil {
			imp.BidFloor = 0.05
		} else if imp.Video != nil {
			imp.BidFloor = 0.1
		}
	} else {
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
	bannerCopy.W = ptrutil.ToPtr(banner.Format[0].W)
	bannerCopy.H = ptrutil.ToPtr(banner.Format[0].H)

	return &bannerCopy, nil
}

func setPublisherId(req *openrtb2.BidRequest, imp *openrtb2.Imp, impExt *openrtb_ext.ImpExtSilverpush) error {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	if err := jsonutil.Unmarshal(bidderExt.Bidder, impExt); err != nil {
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

func impressionByMediaType(imp *openrtb2.Imp) openrtb2.Imp {
	impCopy := *imp
	if imp.Banner != nil {
		impCopy.Video = nil
	}
	if imp.Video != nil {
		impCopy.Banner = nil

	}
	return impCopy
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(response) {
		return nil, nil
	}
	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var bidResp openrtb2.BidResponse

	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(internalRequest.Imp))

	// overrride default currency
	if bidResp.Cur != "" {
		bidResponse.Currency = bidResp.Cur
	}

	for _, sb := range bidResp.SeatBid {
		for i := 0; i < len(sb.Bid); i++ {
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: getMediaTypeForImp(sb.Bid[i]),
			})
		}
	}
	return bidResponse, nil
}

// getMediaTypeForImp figures out which media type this bid is for.
// SilverPush doesn't support multi-type impressions.
// If both banner and video exist, take banner as we do not want in-banner video.
func getMediaTypeForImp(bid openrtb2.Bid) openrtb_ext.BidType {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo
	default:
		return ""
	}
}

// Builder builds a new instance of the silverpush adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint:   config.Endpoint,
		bidderName: string(bidderName),
	}
	return bidder, nil
}
