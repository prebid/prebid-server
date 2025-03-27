package smaato

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/util/timeutil"
)

const clientVersion = "prebid_server_1.2"

type adMarkupType string

const (
	smtAdTypeImg       adMarkupType = "Img"
	smtAdTypeRichmedia adMarkupType = "Richmedia"
	smtAdTypeVideo     adMarkupType = "Video"
	smtAdTypeNative    adMarkupType = "Native"
)

// adapter describes a Smaato prebid server adapter.
type adapter struct {
	clock    timeutil.Time
	endpoint string
}

// userExtData defines User.Ext.Data object for Smaato
type userExtData struct {
	Keywords string `json:"keywords"`
	Gender   string `json:"gender"`
	Yob      int64  `json:"yob"`
}

// siteExt defines Site.Ext object for Smaato
type siteExt struct {
	Data siteExtData `json:"data"`
}

type siteExtData struct {
	Keywords string `json:"keywords"`
}

// bidRequestExt defines BidRequest.Ext object for Smaato
type bidRequestExt struct {
	Client string `json:"client"`
}

// bidExt defines Bid.Ext object for Smaato
type bidExt struct {
	Duration int      `json:"duration"`
	Curls    []string `json:"curls"`
}

// videoExt defines Video.Ext object for Smaato
type videoExt struct {
	Context string `json:"context,omitempty"`
}

// Builder builds a new instance of the Smaato adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		clock:    &timeutil.RealTime{},
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

// MakeRequests makes the HTTP requests which should be made to fetch bids.
func (adapter *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{Message: "No impressions in bid request."}}
	}

	// set data in request that is common for all requests
	if err := prepareCommonRequest(request); err != nil {
		return nil, []error{err}
	}

	isVideoEntryPoint := reqInfo.PbsEntryPoint == metrics.ReqTypeVideo

	if isVideoEntryPoint {
		return adapter.makePodRequests(request)
	} else {
		return adapter.makeIndividualRequests(request)
	}
}

// MakeBids unpacks the server's response into Bids.
func (adapter *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info.", response.StatusCode),
		}}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	adMarkupType, err := getAdMarkupType(response)
	if err != nil {
		return nil, []error{err}
	}

	var errors []error
	for _, seatBid := range bidResp.SeatBid {
		for i := 0; i < len(seatBid.Bid); i++ {
			bid := seatBid.Bid[i]

			bidExt, err := extractBidExt(&bid)
			if err != nil {
				errors = append(errors, err)
				continue
			}

			bid.AdM, err = renderAdMarkup(adMarkupType, &bidExt, bid)
			if err != nil {
				errors = append(errors, err)
				continue
			}

			bidType, err := convertAdMarkupTypeToMediaType(adMarkupType)
			if err != nil {
				errors = append(errors, err)
				continue
			}

			bidVideo, err := buildBidVideo(&bid, &bidExt, bidType)
			if err != nil {
				errors = append(errors, err)
				continue
			}

			bid.Exp = adapter.getTTLFromHeaderOrDefault(response)

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:      &bid,
				BidType:  bidType,
				BidVideo: bidVideo,
			})
		}
	}
	return bidResponse, errors
}

func (adapter *adapter) makeIndividualRequests(request *openrtb2.BidRequest) ([]*adapters.RequestData, []error) {
	imps := request.Imp

	requests := make([]*adapters.RequestData, 0, len(imps))
	errors := make([]error, 0, len(imps))

	for _, imp := range imps {
		impsByMediaType, err := splitImpressionsByMediaType(&imp)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		for _, impByMediaType := range impsByMediaType {
			request.Imp = []openrtb2.Imp{impByMediaType}
			if err := prepareIndividualRequest(request); err != nil {
				errors = append(errors, err)
				continue
			}

			requestData, err := adapter.makeRequest(request)
			if err != nil {
				errors = append(errors, err)
				continue
			}

			requests = append(requests, requestData)
		}
	}

	return requests, errors
}

func splitImpressionsByMediaType(imp *openrtb2.Imp) ([]openrtb2.Imp, error) {
	if imp.Banner == nil && imp.Video == nil && imp.Native == nil {
		return nil, &errortypes.BadInput{Message: "Invalid MediaType. Smaato only supports Banner, Video and Native."}
	}

	imps := make([]openrtb2.Imp, 0, 3)

	if imp.Banner != nil {
		impCopy := *imp
		impCopy.Video = nil
		impCopy.Native = nil
		imps = append(imps, impCopy)
	}

	if imp.Video != nil {
		impCopy := *imp
		impCopy.Banner = nil
		impCopy.Native = nil
		imps = append(imps, impCopy)
	}

	if imp.Native != nil {
		imp.Banner = nil
		imp.Video = nil
		imps = append(imps, *imp)
	}

	return imps, nil
}

func (adapter *adapter) makePodRequests(request *openrtb2.BidRequest) ([]*adapters.RequestData, []error) {
	pods, orderedKeys, errors := groupImpressionsByPod(request.Imp)
	requests := make([]*adapters.RequestData, 0, len(pods))

	for _, key := range orderedKeys {
		request.Imp = pods[key]

		if err := preparePodRequest(request); err != nil {
			errors = append(errors, err)
			continue
		}

		requestData, err := adapter.makeRequest(request)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		requests = append(requests, requestData)
	}

	return requests, errors
}

func (adapter *adapter) makeRequest(request *openrtb2.BidRequest) (*adapters.RequestData, error) {
	reqJSON, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	return &adapters.RequestData{
		Method:  "POST",
		Uri:     adapter.endpoint,
		Body:    reqJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}, nil
}

func getAdMarkupType(response *adapters.ResponseData) (adMarkupType, error) {
	if admType := adMarkupType(response.Headers.Get("X-Smt-Adtype")); admType != "" {
		return admType, nil
	} else {
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("X-Smt-Adtype header is missing."),
		}
	}
}

func (adapter *adapter) getTTLFromHeaderOrDefault(response *adapters.ResponseData) int64 {
	ttl := int64(300)

	if expiresAtMillis, err := strconv.ParseInt(response.Headers.Get("X-Smt-Expires"), 10, 64); err == nil {
		nowMillis := adapter.clock.Now().UnixNano() / 1000000
		ttl = (expiresAtMillis - nowMillis) / 1000
		if ttl < 0 {
			ttl = 0
		}
	}

	return ttl
}

func renderAdMarkup(adMarkupType adMarkupType, bidExt *bidExt, bid openrtb2.Bid) (string, error) {
	switch adMarkupType {
	case smtAdTypeImg, smtAdTypeRichmedia:
		return extractAdmBanner(bid.AdM, bidExt.Curls), nil
	case smtAdTypeVideo:
		return bid.AdM, nil
	case smtAdTypeNative:
		return extractAdmNative(bid.AdM)
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unknown markup type %s.", adMarkupType),
		}
	}
}

func convertAdMarkupTypeToMediaType(adMarkupType adMarkupType) (openrtb_ext.BidType, error) {
	switch adMarkupType {
	case smtAdTypeImg, smtAdTypeRichmedia:
		return openrtb_ext.BidTypeBanner, nil
	case smtAdTypeVideo:
		return openrtb_ext.BidTypeVideo, nil
	case smtAdTypeNative:
		return openrtb_ext.BidTypeNative, nil
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unknown markup type %s.", adMarkupType),
		}
	}
}

func prepareCommonRequest(request *openrtb2.BidRequest) error {
	if err := setUser(request); err != nil {
		return err
	}

	if err := setSite(request); err != nil {
		return err
	}

	setApp(request)
	setDOOH(request)

	return setExt(request)
}

func prepareIndividualRequest(request *openrtb2.BidRequest) error {
	imp := &request.Imp[0]

	if err := setPublisherId(request, imp); err != nil {
		return err
	}

	return setImpForAdspace(imp)
}

func preparePodRequest(request *openrtb2.BidRequest) error {
	if len(request.Imp) < 1 {
		return &errortypes.BadInput{Message: "No impressions in bid request."}
	}

	if err := setPublisherId(request, &request.Imp[0]); err != nil {
		return err
	}

	return setImpForAdBreak(request.Imp)
}

func setUser(request *openrtb2.BidRequest) error {
	if request.User != nil && request.User.Ext != nil {
		var userExtRaw map[string]json.RawMessage

		if err := jsonutil.Unmarshal(request.User.Ext, &userExtRaw); err != nil {
			return &errortypes.BadInput{Message: "Invalid user.ext."}
		}

		if userExtDataRaw, present := userExtRaw["data"]; present {
			var err error
			var userExtData userExtData

			if err = jsonutil.Unmarshal(userExtDataRaw, &userExtData); err != nil {
				return &errortypes.BadInput{Message: "Invalid user.ext.data."}
			}

			userCopy := *request.User

			if userExtData.Gender != "" {
				userCopy.Gender = userExtData.Gender
			}

			if userExtData.Yob != 0 {
				userCopy.Yob = userExtData.Yob
			}

			if userExtData.Keywords != "" {
				userCopy.Keywords = userExtData.Keywords
			}

			delete(userExtRaw, "data")

			if userCopy.Ext, err = json.Marshal(userExtRaw); err != nil {
				return err
			}

			request.User = &userCopy
		}
	}

	return nil
}

func setExt(request *openrtb2.BidRequest) error {
	var err error

	request.Ext, err = json.Marshal(bidRequestExt{Client: clientVersion})

	return err
}

func setSite(request *openrtb2.BidRequest) error {
	if request.Site != nil {
		siteCopy := *request.Site

		if request.Site.Ext != nil {
			var siteExt siteExt

			if err := jsonutil.Unmarshal(request.Site.Ext, &siteExt); err != nil {
				return &errortypes.BadInput{Message: "Invalid site.ext."}
			}

			siteCopy.Keywords = siteExt.Data.Keywords
			siteCopy.Ext = nil
		}
		request.Site = &siteCopy
	}

	return nil
}

func setApp(request *openrtb2.BidRequest) {
	if request.App != nil {
		appCopy := *request.App
		request.App = &appCopy
	}
}

func setDOOH(request *openrtb2.BidRequest) {
	if request.DOOH != nil {
		doohCopy := *request.DOOH
		request.DOOH = &doohCopy
	}
}

func setPublisherId(request *openrtb2.BidRequest, imp *openrtb2.Imp) error {
	publisherID, err := jsonparser.GetString(imp.Ext, "bidder", "publisherId")
	if err != nil {
		return &errortypes.BadInput{Message: "Missing publisherId parameter."}
	}

	if request.Site != nil {
		// Site is already a copy
		request.Site.Publisher = &openrtb2.Publisher{ID: publisherID}
		return nil
	} else if request.App != nil {
		// App is already a copy
		request.App.Publisher = &openrtb2.Publisher{ID: publisherID}
		return nil
	} else if request.DOOH != nil {
		// DOOH is already a copy
		request.DOOH.Publisher = &openrtb2.Publisher{ID: publisherID}
		return nil
	} else {
		return &errortypes.BadInput{Message: "Missing Site/App/DOOH."}
	}
}

func setImpForAdspace(imp *openrtb2.Imp) error {
	adSpaceID, err := jsonparser.GetString(imp.Ext, "bidder", "adspaceId")
	if err != nil {
		return &errortypes.BadInput{Message: "Missing adspaceId parameter."}
	}

	err = removeBidderNodeFromImpExt(imp)
	if err != nil {
		return err
	}

	if imp.Banner != nil || imp.Video != nil || imp.Native != nil {
		imp.TagID = adSpaceID
		return nil
	}

	return nil
}

func setImpForAdBreak(imps []openrtb2.Imp) error {
	if len(imps) < 1 {
		return &errortypes.BadInput{Message: "No impressions in bid request."}
	}

	firstImp := imps[0]
	adBreakID, err := jsonparser.GetString(firstImp.Ext, "bidder", "adbreakId")
	if err != nil {
		return &errortypes.BadInput{Message: "Missing adbreakId parameter."}
	}

	err = removeBidderNodeFromImpExt(&firstImp)
	if err != nil {
		return err
	}

	for i := range imps {
		imps[i].TagID = adBreakID
		imps[i].Ext = nil

		videoCopy := *(imps[i].Video)

		videoCopy.Sequence = int8(i + 1)
		videoCopy.Ext, _ = json.Marshal(&videoExt{Context: "adpod"})

		imps[i].Video = &videoCopy
	}

	imps[0].Ext = firstImp.Ext

	return nil
}

func removeBidderNodeFromImpExt(imp *openrtb2.Imp) error {
	if imp.Ext == nil {
		return nil
	}
	updatedExt := jsonparser.Delete(imp.Ext, "bidder")
	isEmpty := true
	err := jsonparser.ObjectEach(updatedExt, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		isEmpty = false
		return nil
	})

	if err != nil {
		return err
	}

	if isEmpty {
		imp.Ext = nil
	} else {
		imp.Ext = updatedExt
	}
	return nil
}

func groupImpressionsByPod(imps []openrtb2.Imp) (map[string]([]openrtb2.Imp), []string, []error) {
	pods := make(map[string][]openrtb2.Imp)
	orderKeys := make([]string, 0)
	errors := make([]error, 0, len(imps))

	for _, imp := range imps {
		if imp.Video == nil {
			errors = append(errors, &errortypes.BadInput{Message: "Invalid MediaType. Smaato only supports Video for AdPod."})
			continue
		}

		pod := strings.Split(imp.ID, "_")[0]
		if _, present := pods[pod]; !present {
			orderKeys = append(orderKeys, pod)
		}
		pods[pod] = append(pods[pod], imp)
	}
	return pods, orderKeys, errors
}

func buildBidVideo(bid *openrtb2.Bid, bidExt *bidExt, bidType openrtb_ext.BidType) (*openrtb_ext.ExtBidPrebidVideo, error) {
	if bidType != openrtb_ext.BidTypeVideo {
		return nil, nil
	}

	if bidExt == nil {
		return nil, nil
	}

	var primaryCategory string
	if len(bid.Cat) > 0 {
		primaryCategory = bid.Cat[0]
	}

	return &openrtb_ext.ExtBidPrebidVideo{
		Duration:        bidExt.Duration,
		PrimaryCategory: primaryCategory,
	}, nil
}

func extractBidExt(bid *openrtb2.Bid) (bidExt, error) {
	var bidExt bidExt

	if bid.Ext == nil {
		return bidExt, nil
	}
	if err := jsonutil.Unmarshal(bid.Ext, &bidExt); err != nil {
		return bidExt, &errortypes.BadServerResponse{Message: "Invalid bid.ext."}
	}
	return bidExt, nil
}
