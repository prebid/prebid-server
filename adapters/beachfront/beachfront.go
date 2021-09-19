package beachfront

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const Seat = "beachfront"
const BidCapacity = 5

const defaultVideoEndpoint = "https://reachms.bfmio.com/bid.json?exchange_id"

const nurlVideoEndpointSuffix = "&prebidserver"

const beachfrontAdapterName = "BF_PREBID_S2S"
const beachfrontAdapterVersion = "0.9.2"

const minBidFloor = 0.01

const defaultVideoWidth = 300
const defaultVideoHeight = 250
const fakeIP = "255.255.255.255"

type BeachfrontAdapter struct {
	bannerEndpoint string
	extraInfo      ExtraInfo
}

type ExtraInfo struct {
	VideoEndpoint string `json:"video_endpoint,omitempty"`
}

type beachfrontRequests struct {
	Banner    beachfrontBannerRequest
	NurlVideo []beachfrontVideoRequest
	ADMVideo  []beachfrontVideoRequest
}

// ---------------------------------------------------
//              Video
// ---------------------------------------------------

type beachfrontVideoRequest struct {
	AppId             string              `json:"appId"`
	VideoResponseType string              `json:"videoResponseType"`
	Request           openrtb2.BidRequest `json:"request"`
}

// ---------------------------------------------------
//              Banner
// ---------------------------------------------------
type beachfrontBannerRequest struct {
	Slots          []beachfrontSlot                         `json:"slots"`
	Domain         string                                   `json:"domain"`
	Page           string                                   `json:"page"`
	Referrer       string                                   `json:"referrer"`
	Search         string                                   `json:"search"`
	Secure         int8                                     `json:"secure"`
	DeviceOs       string                                   `json:"deviceOs"`
	DeviceModel    string                                   `json:"deviceModel"`
	IsMobile       int8                                     `json:"isMobile"`
	UA             string                                   `json:"ua"`
	Dnt            int8                                     `json:"dnt"`
	User           openrtb2.User                            `json:"user"`
	AdapterName    string                                   `json:"adapterName"`
	AdapterVersion string                                   `json:"adapterVersion"`
	IP             string                                   `json:"ip"`
	RequestID      string                                   `json:"requestId"`
	Real204        bool                                     `json:"real204"`
	SChain         openrtb_ext.ExtRequestPrebidSChainSChain `json:"schain,omitempty"`
}

type beachfrontSlot struct {
	Slot     string           `json:"slot"`
	Id       string           `json:"id"`
	Bidfloor float64          `json:"bidfloor"`
	Sizes    []beachfrontSize `json:"sizes"`
}

type beachfrontSize struct {
	W uint64 `json:"w"`
	H uint64 `json:"h"`
}

// ---------------------------------------------------
// 				Banner response
// ---------------------------------------------------

type beachfrontResponseSlot struct {
	CrID  string  `json:"crid"`
	Price float64 `json:"price"`
	W     uint64  `json:"w"`
	H     uint64  `json:"h"`
	Slot  string  `json:"slot"`
	Adm   string  `json:"adm"`
}

type beachfrontVideoBidExtension struct {
	Duration int `json:"duration"`
}

func (a *BeachfrontAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	beachfrontRequests, errs := preprocess(request)

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	if request.Device != nil {
		if request.Device.UA != "" {
			headers.Add("User-Agent", request.Device.UA)
		}

		if request.Device.Language != "" {
			headers.Add("Accept-Language", request.Device.Language)
		}

		if request.Device.DNT != nil {
			headers.Add("DNT", strconv.Itoa(int(*request.Device.DNT)))
		}
	}

	var reqCount = len(beachfrontRequests.ADMVideo) + len(beachfrontRequests.NurlVideo)
	if len(beachfrontRequests.Banner.Slots) > 0 {
		reqCount++
	}

	var reqs = make([]*adapters.RequestData, reqCount)

	var nurlBump = 0
	var admBump = 0

	// At most, I only ever have one banner request, and it does not need the cookie, so doing it first.
	if len(beachfrontRequests.Banner.Slots) > 0 {
		bytes, err := json.Marshal(beachfrontRequests.Banner)

		if err == nil {
			reqs[0] = &adapters.RequestData{
				Method:  "POST",
				Uri:     a.bannerEndpoint,
				Body:    bytes,
				Headers: headers,
			}

			nurlBump++
			admBump++
		} else {
			errs = append(errs, err)
		}
	}

	if request.User != nil && request.User.BuyerUID != "" && reqCount > 0 {
		headers.Add("Cookie", "__io_cid="+request.User.BuyerUID)
	}

	for j := 0; j < len(beachfrontRequests.ADMVideo); j++ {
		bytes, err := json.Marshal(beachfrontRequests.ADMVideo[j].Request)
		if err == nil {
			reqs[j+nurlBump] = &adapters.RequestData{
				Method:  "POST",
				Uri:     a.extraInfo.VideoEndpoint + "=" + beachfrontRequests.ADMVideo[j].AppId,
				Body:    bytes,
				Headers: headers,
			}

			admBump++

		} else {
			errs = append(errs, err)
		}
	}

	for j := 0; j < len(beachfrontRequests.NurlVideo); j++ {
		bytes, err := json.Marshal(beachfrontRequests.NurlVideo[j].Request)

		if err == nil {
			bytes = append([]byte(`{"isPrebid":true,`), bytes[1:]...)
			reqs[j+admBump] = &adapters.RequestData{
				Method:  "POST",
				Uri:     a.extraInfo.VideoEndpoint + "=" + beachfrontRequests.NurlVideo[j].AppId + nurlVideoEndpointSuffix,
				Body:    bytes,
				Headers: headers,
			}
		} else {
			errs = append(errs, err)
		}
	}

	return reqs, errs
}

func preprocess(request *openrtb2.BidRequest) (beachfrontReqs beachfrontRequests, errs []error) {
	var videoImps = make([]openrtb2.Imp, 0)
	var bannerImps = make([]openrtb2.Imp, 0)

	for i := 0; i < len(request.Imp); i++ {
		if request.Imp[i].Banner != nil && request.Imp[i].Banner.Format != nil &&
			request.Imp[i].Banner.Format[0].H != 0 && request.Imp[i].Banner.Format[0].W != 0 {
			bannerImps = append(bannerImps, request.Imp[i])
		}

		if request.Imp[i].Video != nil {
			videoImps = append(videoImps, request.Imp[i])
		}
	}

	if len(bannerImps)+len(videoImps) == 0 {
		errs = append(errs, errors.New("no valid impressions were found in the request"))
		return
	}

	if len(bannerImps) > 0 {
		request.Imp = bannerImps
		beachfrontReqs.Banner, errs = getBannerRequest(request)
	}

	if len(videoImps) > 0 {
		var videoErrs []error
		var videoList []beachfrontVideoRequest

		request.Imp = videoImps

		videoList, videoErrs = getVideoRequests(request)
		errs = append(errs, videoErrs...)

		for i := 0; i < len(videoList); i++ {
			if videoList[i].VideoResponseType == "nurl" {
				beachfrontReqs.NurlVideo = append(beachfrontReqs.NurlVideo, videoList[i])
			}

			if videoList[i].VideoResponseType == "adm" {
				beachfrontReqs.ADMVideo = append(beachfrontReqs.ADMVideo, videoList[i])
			}
		}
	}

	return
}

func getAppId(ext openrtb_ext.ExtImpBeachfront, media openrtb_ext.BidType) (string, error) {
	var appid string
	var error error

	if ext.AppId != "" {
		appid = ext.AppId
	} else if media == openrtb_ext.BidTypeVideo && ext.AppIds.Video != "" {
		appid = ext.AppIds.Video
	} else if media == openrtb_ext.BidTypeBanner && ext.AppIds.Banner != "" {
		appid = ext.AppIds.Banner
	} else {
		error = errors.New("unable to determine the appId(s) from the supplied extension")
	}

	return appid, error
}

func getSchain(request *openrtb2.BidRequest) (openrtb_ext.ExtRequestPrebidSChain, error) {
	var schain openrtb_ext.ExtRequestPrebidSChain
	return schain, json.Unmarshal(request.Source.Ext, &schain)
}

/*
getBannerRequest, singular. A "Slot" is an "imp," and each Slot can have an AppId, so just one
request to the beachfront banner endpoint gets all banner Imps.
*/
func getBannerRequest(request *openrtb2.BidRequest) (beachfrontBannerRequest, []error) {
	var bfr beachfrontBannerRequest
	var errs = make([]error, 0, len(request.Imp))

	for i := 0; i < len(request.Imp); i++ {

		beachfrontExt, err := getBeachfrontExtension(request.Imp[i])

		if err != nil {
			errs = append(errs, err)
			continue
		}

		appid, err := getAppId(beachfrontExt, openrtb_ext.BidTypeBanner)

		if err != nil {
			// Failed to get an appid, so this request is junk.
			errs = append(errs, err)
			continue
		}

		setBidFloor(&beachfrontExt, &request.Imp[i])

		slot := beachfrontSlot{
			Id:       appid,
			Slot:     request.Imp[i].ID,
			Bidfloor: request.Imp[i].BidFloor,
		}

		for j := 0; j < len(request.Imp[i].Banner.Format); j++ {

			slot.Sizes = append(slot.Sizes, beachfrontSize{
				H: uint64(request.Imp[i].Banner.Format[j].H),
				W: uint64(request.Imp[i].Banner.Format[j].W),
			})
		}

		bfr.Slots = append(bfr.Slots, slot)
	}

	if len(bfr.Slots) == 0 {
		return bfr, errs
	}

	if request.Device != nil {
		bfr.IP = request.Device.IP
		bfr.DeviceModel = request.Device.Model
		bfr.DeviceOs = request.Device.OS
		if request.Device.DNT != nil {
			bfr.Dnt = *request.Device.DNT
		}
		if request.Device.UA != "" {
			bfr.UA = request.Device.UA
		}
	}

	var t = fallBackDeviceType(request)

	if t == openrtb2.DeviceTypeMobileTablet {
		bfr.Page = request.App.Bundle
		if request.App.Domain == "" {
			bfr.Domain = getDomain(request.App.Domain)
		} else {
			bfr.Domain = request.App.Domain
		}

		bfr.IsMobile = 1
	} else if t == openrtb2.DeviceTypePersonalComputer {
		bfr.Page = request.Site.Page
		if request.Site.Domain == "" {
			bfr.Domain = getDomain(request.Site.Page)
		} else {
			bfr.Domain = request.Site.Domain
		}

		bfr.IsMobile = 0
	}

	bfr.Secure = isSecure(bfr.Page)

	if request.User != nil && request.User.ID != "" {
		if bfr.User.ID == "" {
			bfr.User.ID = request.User.ID
		}
	}

	if request.User != nil && request.User.BuyerUID != "" {
		if bfr.User.BuyerUID == "" {
			bfr.User.BuyerUID = request.User.BuyerUID
		}
	}

	bfr.RequestID = request.ID
	bfr.AdapterName = beachfrontAdapterName
	bfr.AdapterVersion = beachfrontAdapterVersion

	if request.Imp[0].Secure != nil {
		bfr.Secure = *request.Imp[0].Secure
	}
	bfr.Real204 = true

	if request.Source != nil && request.Source.Ext != nil {
		schain, err := getSchain(request)
		if err == nil {
			bfr.SChain = schain.SChain
		}
	}

	return bfr, errs
}

func fallBackDeviceType(request *openrtb2.BidRequest) openrtb2.DeviceType {
	if request.Site != nil {
		return openrtb2.DeviceTypePersonalComputer
	}

	return openrtb2.DeviceTypeMobileTablet
}

func getVideoRequests(request *openrtb2.BidRequest) ([]beachfrontVideoRequest, []error) {
	var bfReqs = make([]beachfrontVideoRequest, len(request.Imp))
	var errs = make([]error, 0, len(request.Imp))
	var failedRequestIndicies = make([]int, 0)

	for i := 0; i < len(request.Imp); i++ {
		beachfrontExt, err := getBeachfrontExtension(request.Imp[i])

		if err != nil {
			// Failed to extract the beachfrontExt, so this request is junk.
			failedRequestIndicies = append(failedRequestIndicies, i)
			errs = append(errs, err)
			continue
		}

		appid, err := getAppId(beachfrontExt, openrtb_ext.BidTypeVideo)
		bfReqs[i].AppId = appid

		if err != nil {
			// Failed to get an appid, so this request is junk.
			failedRequestIndicies = append(failedRequestIndicies, i)
			errs = append(errs, err)
			continue
		}

		bfReqs[i].Request = *request
		var secure int8

		var deviceCopy openrtb2.Device
		if bfReqs[i].Request.Device == nil {
			deviceCopy = openrtb2.Device{}
		} else {
			deviceCopy = *bfReqs[i].Request.Device
		}

		if beachfrontExt.VideoResponseType == "nurl" {
			bfReqs[i].VideoResponseType = "nurl"
		} else {
			bfReqs[i].VideoResponseType = "adm"

			if deviceCopy.IP == "" {
				deviceCopy.IP = fakeIP
			}
		}

		if bfReqs[i].Request.Site != nil && bfReqs[i].Request.Site.Domain == "" && bfReqs[i].Request.Site.Page != "" {
			siteCopy := *bfReqs[i].Request.Site
			siteCopy.Domain = getDomain(bfReqs[i].Request.Site.Page)
			bfReqs[i].Request.Site = &siteCopy
			secure = isSecure(bfReqs[i].Request.Site.Page)
		}

		if bfReqs[i].Request.App != nil && bfReqs[i].Request.App.Domain == "" && bfReqs[i].Request.App.Bundle != "" {
			if bfReqs[i].Request.App.Bundle != "" {
				var chunks = strings.Split(strings.Trim(bfReqs[i].Request.App.Bundle, "_"), ".")

				if len(chunks) > 1 {
					appCopy := *bfReqs[i].Request.App
					appCopy.Domain = fmt.Sprintf("%s.%s", chunks[len(chunks)-(len(chunks)-1)], chunks[0])
					bfReqs[i].Request.App = &appCopy
				}
			}
		}

		if deviceCopy.DeviceType == 0 {
			// More fine graned deviceType methods will be added in the future
			deviceCopy.DeviceType = fallBackDeviceType(request)
		}
		bfReqs[i].Request.Device = &deviceCopy

		imp := request.Imp[i]

		imp.Banner = nil
		imp.Ext = nil
		imp.Secure = &secure
		setBidFloor(&beachfrontExt, &imp)

		if imp.Video.H == 0 && imp.Video.W == 0 {
			imp.Video.W = defaultVideoWidth
			imp.Video.H = defaultVideoHeight
		}

		if len(bfReqs[i].Request.Cur) == 0 {
			bfReqs[i].Request.Cur = make([]string, 1)
			bfReqs[i].Request.Cur[0] = "USD"
		}

		bfReqs[i].Request.Imp = nil
		bfReqs[i].Request.Imp = make([]openrtb2.Imp, 1)
		bfReqs[i].Request.Imp[0] = imp

	}

	// Strip out any failed requests
	if len(failedRequestIndicies) > 0 {
		for i := 0; i < len(failedRequestIndicies); i++ {
			bfReqs = removeVideoElement(bfReqs, failedRequestIndicies[i])
		}

	}
	return bfReqs, errs
}

func (a *BeachfrontAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode >= http.StatusInternalServerError {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("server error status code %d from %s. Run with request.debug = 1 for more info", response.StatusCode, externalRequest.Uri),
		}}
	}

	if response.StatusCode >= http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("request error status code %d from %s. Run with request.debug = 1 for more info", response.StatusCode, externalRequest.Uri),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{fmt.Errorf("unexpected status code %d from %s. Run with request.debug = 1 for more info", response.StatusCode, externalRequest.Uri)}
	}

	var bids []openrtb2.Bid
	var errs = make([]error, 0)
	var xtrnal openrtb2.BidRequest

	// For video, which uses RTB for the external request, this will unmarshal as expected. For banner, it will
	// only get the User struct and everything else will be nil
	if err := json.Unmarshal(externalRequest.Body, &xtrnal); err != nil {
		errs = append(errs, err)
	} else {
		bids, errs = postprocess(response, xtrnal, externalRequest.Uri, internalRequest.ID)
	}

	if len(errs) != 0 {
		return nil, errs
	}

	var dur beachfrontVideoBidExtension
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(BidCapacity)

	for i := 0; i < len(bids); i++ {

		// If we unmarshal without an error, this is an AdM video
		if err := json.Unmarshal(bids[i].Ext, &dur); err == nil {
			var impVideo openrtb_ext.ExtBidPrebidVideo
			impVideo.Duration = int(dur.Duration)

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:      &bids[i],
				BidType:  a.getBidType(externalRequest),
				BidVideo: &impVideo,
			})
		} else {
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bids[i],
				BidType: a.getBidType(externalRequest),
			})
		}
	}

	return bidResponse, errs
}

func setBidFloor(ext *openrtb_ext.ExtImpBeachfront, imp *openrtb2.Imp) {
	var floor float64

	if imp.BidFloor > 0 {
		floor = imp.BidFloor
	} else if ext.BidFloor > 0 {
		floor = ext.BidFloor
	} else {
		floor = minBidFloor
	}

	if floor <= minBidFloor {
		floor = 0
	}

	imp.BidFloor = floor
}

func (a *BeachfrontAdapter) getBidType(externalRequest *adapters.RequestData) openrtb_ext.BidType {
	t := strings.Split(externalRequest.Uri, "=")[0]
	if t == a.extraInfo.VideoEndpoint {
		return openrtb_ext.BidTypeVideo
	}

	return openrtb_ext.BidTypeBanner
}

func postprocess(response *adapters.ResponseData, xtrnal openrtb2.BidRequest, uri string, id string) ([]openrtb2.Bid, []error) {
	var beachfrontResp []beachfrontResponseSlot

	var openrtbResp openrtb2.BidResponse

	if err := json.Unmarshal(response.Body, &openrtbResp); err != nil || len(openrtbResp.SeatBid) == 0 {

		if err := json.Unmarshal(response.Body, &beachfrontResp); err != nil {
			return nil, []error{&errortypes.BadServerResponse{
				Message: "server response failed to unmarshal as valid rtb. Run with request.debug = 1 for more info",
			}}
		} else {
			return postprocessBanner(beachfrontResp, id)
		}
	}

	return postprocessVideo(openrtbResp.SeatBid[0].Bid, xtrnal, uri, id)
}

func postprocessBanner(beachfrontResp []beachfrontResponseSlot, id string) ([]openrtb2.Bid, []error) {

	var bids = make([]openrtb2.Bid, len(beachfrontResp))
	var errs = make([]error, 0)

	for i := 0; i < len(beachfrontResp); i++ {
		bids[i] = openrtb2.Bid{
			CrID:  beachfrontResp[i].CrID,
			ImpID: beachfrontResp[i].Slot,
			Price: beachfrontResp[i].Price,
			ID:    fmt.Sprintf("%sBanner", beachfrontResp[i].Slot),
			AdM:   beachfrontResp[i].Adm,
			H:     int64(beachfrontResp[i].H),
			W:     int64(beachfrontResp[i].W),
		}
	}

	return bids, errs
}

func postprocessVideo(bids []openrtb2.Bid, xtrnal openrtb2.BidRequest, uri string, id string) ([]openrtb2.Bid, []error) {

	var errs = make([]error, 0)

	if uri[len(uri)-len(nurlVideoEndpointSuffix):] == nurlVideoEndpointSuffix {

		for i := 0; i < len(bids); i++ {
			crid := extractNurlVideoCrid(bids[i].NURL)

			bids[i].CrID = crid
			bids[i].ImpID = xtrnal.Imp[i].ID
			bids[i].H = xtrnal.Imp[i].Video.H
			bids[i].W = xtrnal.Imp[i].Video.W
			bids[i].ID = fmt.Sprintf("%sNurlVideo", xtrnal.Imp[i].ID)
		}

	} else {
		for i := 0; i < len(bids); i++ {
			bids[i].ID = fmt.Sprintf("%sAdmVideo", bids[i].ImpID)
		}

	}
	return bids, errs
}

func extractNurlVideoCrid(nurl string) string {
	chunky := strings.SplitAfter(nurl, ":")
	if len(chunky) > 1 {
		return strings.TrimSuffix(chunky[2], ":")
	}

	return ""
}

func getBeachfrontExtension(imp openrtb2.Imp) (openrtb_ext.ExtImpBeachfront, error) {
	var err error
	var bidderExt adapters.ExtImpBidder
	var beachfrontExt openrtb_ext.ExtImpBeachfront

	if err = json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return beachfrontExt, &errortypes.BadInput{
			Message: fmt.Sprintf("ignoring imp id=%s, error while decoding extImpBidder, err: %s", imp.ID, err),
		}
	}

	if err = json.Unmarshal(bidderExt.Bidder, &beachfrontExt); err != nil {
		return beachfrontExt, &errortypes.BadInput{
			Message: fmt.Sprintf("ignoring imp id=%s, error while decoding extImpBeachfront, err: %s", imp.ID, err),
		}
	}

	return beachfrontExt, err
}

func getDomain(page string) string {
	protoURL := strings.Split(page, "//")
	var domainPage string

	if len(protoURL) > 1 {
		domainPage = protoURL[1]
	} else {
		domainPage = protoURL[0]
	}

	return strings.Split(domainPage, "/")[0]

}

func isSecure(page string) int8 {
	protoURL := strings.Split(page, "://")

	if len(protoURL) > 1 && protoURL[0] == "https" {
		return 1
	}

	return 0

}

func removeVideoElement(slice []beachfrontVideoRequest, s int) []beachfrontVideoRequest {
	if len(slice) >= s+1 {
		return append(slice[:s], slice[s+1:]...)
	}

	return []beachfrontVideoRequest{}
}

// Builder builds a new instance of the Beachfront adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	extraInfo, err := getExtraInfo(config.ExtraAdapterInfo)
	if err != nil {
		return nil, err
	}

	bidder := &BeachfrontAdapter{
		bannerEndpoint: config.Endpoint,
		extraInfo:      extraInfo,
	}
	return bidder, nil
}

func getExtraInfo(v string) (ExtraInfo, error) {
	if len(v) == 0 {
		return getDefaultExtraInfo(), nil
	}

	var extraInfo ExtraInfo
	if err := json.Unmarshal([]byte(v), &extraInfo); err != nil {
		return extraInfo, fmt.Errorf("invalid extra info: %v", err)
	}

	if extraInfo.VideoEndpoint == "" {
		extraInfo.VideoEndpoint = defaultVideoEndpoint
	}

	return extraInfo, nil
}

func getDefaultExtraInfo() ExtraInfo {
	return ExtraInfo{
		VideoEndpoint: defaultVideoEndpoint,
	}
}
