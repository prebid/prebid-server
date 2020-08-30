package beachfront

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const Seat = "beachfront"
const BidCapacity = 5

const defaultVideoEndpoint = "https://reachms.bfmio.com/bid.json?exchange_id"

const nurlVideoEndpointSuffix = "&prebidserver"

const beachfrontAdapterName = "BF_PREBID_S2S"
const beachfrontAdapterVersion = "0.9.0"

const minBidFloor = 0.01

const DefaultVideoWidth = 300
const DefaultVideoHeight = 250

type BeachfrontAdapter struct {
	bannerEndpoint string
	extraInfo      ExtraInfo
}

type ExtraInfo struct {
	VideoEndpoint string `json:"video_endpoint,omitempty"`
}

type requests struct {
	Banner    bannerRequest
	NurlVideo []videoRequest
	ADMVideo  videoRequest
}

// ---------------------------------------------------
//              Video
// ---------------------------------------------------

type videoRequest struct {
	AppId             string             `json:"appId"`
	VideoResponseType string             `json:"videoResponseType"`
	Request           openrtb.BidRequest `json:"request"`
}

// ---------------------------------------------------
//              Banner
// ---------------------------------------------------
type bannerRequest struct {
	Slots          []slot `json:"slots"`
	Domain         string `json:"domain"`
	Page           string `json:"page"`
	Referrer       string `json:"referrer"`
	Search         string `json:"search"`
	Secure         int8   `json:"secure"`
	DeviceOs       string `json:"deviceOs"`
	DeviceModel    string `json:"deviceModel"`
	IsMobile       int8   `json:"isMobile"`
	UA             string `json:"ua"`
	Dnt            int8   `json:"dnt"`
	User           openrtb.User     `json:"user"`
	AdapterName    string           `json:"adapterName"`
	AdapterVersion string           `json:"adapterVersion"`
	IP             string           `json:"ip"`
	RequestID      string           `json:"requestId"`
}

type slot struct {
	Slot     string  `json:"slot"`
	Id       string  `json:"id"`
	Bidfloor float64 `json:"bidfloor"`
	Sizes    []size  `json:"sizes"`
}

type size struct {
	W uint64 `json:"w"`
	H uint64 `json:"h"`
}

// ---------------------------------------------------
// 				Banner response
// ---------------------------------------------------

type responseSlot struct {
	CrID  string  `json:"crid"`
	Price float64 `json:"price"`
	W     uint64  `json:"w"`
	H     uint64  `json:"h"`
	Slot  string  `json:"slot"`
	Adm   string  `json:"adm"`
}

type videoBidExtension struct {
	Duration int `json:"duration"`
}

func (a *BeachfrontAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
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

	var reqCount = len(beachfrontRequests.NurlVideo)
	if len(beachfrontRequests.Banner.Slots) > 0 {
		reqCount++
	}

	reqCount = reqCount + len(beachfrontRequests.ADMVideo.Request.Imp)

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

	bumpAdm := false
	// @TODO - get the false cases on these errors
	for j := 0; j < len(beachfrontRequests.ADMVideo.Request.Imp); j++ {
		bytes, err := json.Marshal(beachfrontRequests.ADMVideo.Request.Imp[j])
		if err == nil {
			ext, err := getBeachfrontExtension(beachfrontRequests.ADMVideo.Request.Imp[j])
			if err == nil {
				appId, _ := getAppId(ext, openrtb_ext.BidTypeVideo)
				if err == nil {
					reqs[j+nurlBump] = &adapters.RequestData{
						Method:  "POST",
						Uri:     a.extraInfo.VideoEndpoint + "=" + appId,
						Body:    bytes,
						Headers: headers,
					}
				}
			}
			bumpAdm = true

		} else {
			errs = append(errs, err)
		}
	}

	if bumpAdm {
		admBump++
	}

	for j := 0; j < len(beachfrontRequests.NurlVideo); j++ {
		bytes, err := json.Marshal(beachfrontRequests.NurlVideo[j].Request)

		if err == nil {
			reqs[j+admBump] = &adapters.RequestData{
				Method:  "POST",
				Uri:     a.extraInfo.VideoEndpoint + "=" + beachfrontRequests.NurlVideo[j].AppId + nurlVideoEndpointSuffix,
				Body:    append([]byte(`{"isPrebid":true,`), bytes[1:]...),
				Headers: headers,
			}
		} else {
			errs = append(errs, err)
		}
	}

	return reqs, errs
}

func preprocess(request *openrtb.BidRequest) (beachfrontReqs requests, errs []error) {
	var videoImps = make([]openrtb.Imp, 0)
	var bannerImps = make([]openrtb.Imp, 0)

	for i := 0; i < len(request.Imp); i++ {
		// @TODO - I should separate out these dimensional checks, add any errors to the stack, and
		// set to some default if they are missing.
		if request.Imp[i].Banner != nil && (
			(request.Imp[i].Banner.Format[0].H != 0 && request.Imp[i].Banner.Format[0].W != 0) ||
			(request.Imp[i].Banner.H != nil && request.Imp[i].Banner.W != nil)) {
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

	// @TODO - follow this in and see ...
	if len(bannerImps) > 0 {
		// request.Imp = bannerImps		// ... why was I doing this????
		beachfrontReqs.Banner, errs = getBannerRequest(request)
	}

	/* We have video imps, so organize them into nurl imps and adm imps. The nurl imps will be
		sent sequentially and the adm imps in parallel.
	*/
	if len(videoImps) > 0 {
		admRequest := videoRequest{
			AppId : "",
			VideoResponseType : "",
			Request : *request,
		}
		admRequest.Request.Imp = nil

		for i := 0; i < len(videoImps); i++ {
			var ext openrtb_ext.ExtImpBeachfront
			ext, errs = prepVideoRequestExt(videoImps[i], errs)

			// @TODO - define these strings in a struct
			if ext.VideoResponseType == "nurl" || ext.VideoResponseType == "both" {
				beachfrontReqs.NurlVideo = append(beachfrontReqs.NurlVideo, videoRequest{
						AppId: "",
						VideoResponseType: ext.VideoResponseType,
						Request: *request,
					})
				j := len(beachfrontReqs.NurlVideo) - 1
				beachfrontReqs.NurlVideo[j].Request.Imp = nil
				beachfrontReqs.NurlVideo[j].Request.Imp = append(
					beachfrontReqs.NurlVideo[j].Request.Imp, videoImps[i])
				beachfrontReqs.NurlVideo[j].Request.Imp[0].Ext, _ = json.Marshal(ext)
			}

			if ext.VideoResponseType == "adm" || ext.VideoResponseType == "both" {
				admRequest.Request.Imp = append(admRequest.Request.Imp, videoImps[i])
			}

		}

		beachfrontReqs.ADMVideo = admRequest
		// Now I gotta stack of each. The nurls get done in sequence, so requests will include one for all
		// adm and one for each nurl
		/*
		} else {
			videoList, videoErrs = getParallelVideoRequests(request)
			errs = append(errs, videoErrs...)

			if videoList[0].VideoResponseType == "nurl" || videoList[0].VideoResponseType == "both" {
				beachfrontReqs.NurlVideo = append(beachfrontReqs.NurlVideo, videoList[0])
			}

			if videoList[0].VideoResponseType == "adm" || videoList[0].VideoResponseType == "both" {
				beachfrontReqs.ADMVideo = append(beachfrontReqs.ADMVideo, videoList[0])
			}
		}
		*/
	}

	return
}

func getAppId(ext openrtb_ext.ExtImpBeachfront, media openrtb_ext.BidType) (string, error) {
	var appid string
	var error error

	if fmt.Sprintf("%s", reflect.TypeOf(ext.AppId)) == "string" &&
		ext.AppId != "" {

		appid = ext.AppId
	} else if fmt.Sprintf("%s", reflect.TypeOf(ext.AppIds)) == "openrtb_ext.ExtImpBeachfrontAppIds" {
		if media == openrtb_ext.BidTypeVideo && ext.AppIds.Video != "" {
			appid = ext.AppIds.Video
		} else if media == openrtb_ext.BidTypeBanner && ext.AppIds.Banner != "" {
			appid = ext.AppIds.Banner
		}
	} else {
		error = errors.New("unable to determine the appId(s) from the supplied extension")
	}

	return appid, error
}

/*
getBannerRequest, singular. A "Slot" is an "imp," and each Slot can have an AppId, so just one
request to the beachfront banner endpoint gets all banner Imps.
*/
func getBannerRequest(request *openrtb.BidRequest) (bannerRequest, []error) {
	var bfr bannerRequest
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

		slot := slot{
			Id:       appid,
			Slot:     request.Imp[i].ID,
			Bidfloor: beachfrontExt.BidFloor,
		}

		if beachfrontExt.BidFloor <= minBidFloor {
			slot.Bidfloor = 0
		}

		for j := 0; j < len(request.Imp[i].Banner.Format); j++ {

			slot.Sizes = append(slot.Sizes, size{
				H: request.Imp[i].Banner.Format[j].H,
				W: request.Imp[i].Banner.Format[j].W,
			})
		}

		bfr.Slots = append(bfr.Slots, slot)
	}

	if len(bfr.Slots) == 0 {
		return bfr, errs
	}

	if request.Device != nil {
		bfr.IP = getIP(request.Device.IP)
		bfr.DeviceModel = request.Device.Model
		bfr.DeviceOs = request.Device.OS
		if request.Device.DNT != nil {
			bfr.Dnt = *request.Device.DNT
		}
		if request.Device.UA != "" {
			bfr.UA = request.Device.UA
		}
	}

	var t = fallBackDeviceType(*request)

	if t == openrtb.DeviceTypeMobileTablet {
		bfr.Page = request.App.Bundle
		if request.App.Domain == "" {
			bfr.Domain = getDomain(request.App.Domain)
		} else {
			bfr.Domain = request.App.Domain
		}

		bfr.IsMobile = 1
	} else if t == openrtb.DeviceTypePersonalComputer {
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

	return bfr, errs
}

func fallBackDeviceType(request openrtb.BidRequest) openrtb.DeviceType {
	if request.Site != nil {
		return openrtb.DeviceTypePersonalComputer
	}

	return openrtb.DeviceTypeMobileTablet
}

/**
A few modifications to the beachfront extension for a solitary Imp object
 */
func prepVideoRequestExt(requestImp  openrtb.Imp, errs []error) (openrtb_ext.ExtImpBeachfront, []error) {
	if requestImp.Video == nil {
		errs = append(errs, errors.New(
			fmt.Sprintf("no valid video elements were found in impression id = %s", requestImp.ID),
			),
		)
	}

	beachfrontExt, err := getBeachfrontExtension(requestImp)

	if err != nil {
		// Failed to extract the beachfrontExt, so this request is junk.
		errs = append(errs, err)
	}

	beachfrontExt.AppId, err = getAppId(beachfrontExt, openrtb_ext.BidTypeVideo)

	if err != nil {
		errs = append(errs, err)
	}

	if beachfrontExt.VideoResponseType != "adm" {
		beachfrontExt.VideoResponseType = "nurl"
	}

	return beachfrontExt, errs
}

func prepVideoRequest(bfReq videoRequest) (videoRequest, int8) {
	var secure int8 = 0

	if bfReq.Request.Site != nil && bfReq.Request.Site.Domain == "" && bfReq.Request.Site.Page != "" {
		bfReq.Request.Site.Domain = getDomain(bfReq.Request.Site.Page)

		secure = isSecure(bfReq.Request.Site.Page)
	}

	if bfReq.Request.App != nil && bfReq.Request.App.Domain == "" && bfReq.Request.App.Bundle != "" {
		if bfReq.Request.App.Bundle != "" {
			var chunks = strings.Split(strings.Trim(bfReq.Request.App.Bundle, "_"), ".")

			if len(chunks) > 1 {
				bfReq.Request.App.Domain =
					fmt.Sprintf("%s.%s", chunks[len(chunks)-(len(chunks)-1)], chunks[0])
			}
		}

	}

	if bfReq.Request.Device.DeviceType == 0 {
		// More fine grained deviceType methods will be added in the future
		bfReq.Request.Device.DeviceType = fallBackDeviceType(bfReq.Request)
	}

	if bfReq.Request.Device != nil {
		bfReq.Request.Device.IP = getIP(bfReq.Request.Device.IP)
	}

	if len(bfReq.Request.Cur) == 0 {
		bfReq.Request.Cur = make([]string, 1)
		bfReq.Request.Cur[0] = "USD"
	}

	return bfReq, secure
}

func prepImp(imp openrtb.Imp, beachfrontExt openrtb_ext.ExtImpBeachfront, secure int8) (openrtb.Imp, int8) {
	imp.Banner = nil
	imp.Ext = nil
	imp.Secure = &secure

	if beachfrontExt.BidFloor <= minBidFloor {
		imp.BidFloor = 0
	} else {
		imp.BidFloor = beachfrontExt.BidFloor
	}

	if imp.Video.H == 0 && imp.Video.W == 0 {
		imp.Video.W = DefaultVideoWidth
		imp.Video.H = DefaultVideoHeight
	}

	return imp, secure
}

/*
func getParallelVideoRequests(request  *openrtb.BidRequest) ([]videoRequest, []error) {
	var bfReq videoRequest
	var errs = make([]error, 0, len(request.Imp))
	var secure int8
	var beachfrontExt openrtb_ext.ExtImpBeachfront


	beachfrontExt, bfReq.AppId, bfReq.VideoResponseType, errs = prepVideoRequestExt(request, errs)

	bfReq.Request = *request
	bfReq, secure = prepVideoRequest(bfReq)

	for i := 0; i < len(request.Imp); i++ {
		if request.Imp[i].Video == nil {
			continue
		}
		bfReq.Request.Imp[i], secure = prepImp(bfReq.Request.Imp[i], beachfrontExt, secure)
		bfReq.Request.Imp[i].Video.Sequence = request.Imp[i].Video.Sequence
	}

	var bfReqs = make([]videoRequest, 1)
	bfReqs[0] = bfReq
	return bfReqs, errs
}
*/

func getSequentialVideoRequests(request *openrtb.BidRequest) ([]videoRequest, []error) {
	var bfReqs = make([]videoRequest, len(request.Imp))
	var errs = make([]error, 0, len(request.Imp))
	var secure int8
	var failedRequestIndices = make([]int, 0)
	var beachfrontExt openrtb_ext.ExtImpBeachfront
	var lastErrorCount = 0

	for i := 0; i < len(request.Imp); i++ {
		beachfrontExt, errs = prepVideoRequestExt(request.Imp[i], errs)
		if len(errs) > lastErrorCount {
			failedRequestIndices = append(failedRequestIndices, i)
			lastErrorCount = len(errs)
			continue
		}

		bfReqs[i].Request = *request
		bfReqs[i].Request.Imp[0], secure = prepImp(bfReqs[i].Request.Imp[0], beachfrontExt, secure)
	}

	// Strip out any failed requests
	if len(failedRequestIndices) > 0 {
		for i := 0; i < len(failedRequestIndices); i++ {
			bfReqs = removeVideoElement(bfReqs, failedRequestIndices[i])
		}
	}

	return bfReqs, errs
}

func (a *BeachfrontAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var bids []openrtb.Bid
	var errs []error

	if response.StatusCode == http.StatusNoContent || (response.StatusCode == http.StatusOK && len(response.Body) <= 2) {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("bad request status code %d from %s. Run with request.debug = 1 for more info", response.StatusCode, externalRequest.Uri),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{fmt.Errorf("unexpected status code %d from %s. Run with request.debug = 1 for more info", response.StatusCode, externalRequest.Uri)}
	}

	var xtrnal openrtb.BidRequest

	// For video, which uses RTB for the external request, this will unmarshal as expected. For banner, it will
	// only get the User struct and everything else will be nil
	if err := json.Unmarshal(externalRequest.Body, &xtrnal); err != nil {
		errs = append(errs, err)
	}

	bids, errs = postprocess(response, xtrnal, externalRequest.Uri, internalRequest.ID)

	if len(errs) != 0 {
		return nil, errs
	}

	var dur videoBidExtension
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(BidCapacity)
	for i := 0; i < len(bids); i++ {

		// If we unmarshal without an error, this is an AdM video
		if err := json.Unmarshal(bids[i].Ext, &dur); err == nil {
			var impVideo openrtb_ext.ExtBidPrebidVideo
			impVideo.Duration = dur.Duration

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

func (a *BeachfrontAdapter) getBidType(externalRequest *adapters.RequestData) openrtb_ext.BidType {
	t := strings.Split(externalRequest.Uri, "=")[0]
	if t == a.extraInfo.VideoEndpoint {
		return openrtb_ext.BidTypeVideo
	}

	return openrtb_ext.BidTypeBanner
}

func postprocess(response *adapters.ResponseData, xtrnal openrtb.BidRequest, uri string, id string) ([]openrtb.Bid, []error) {
	var beachfrontResp []responseSlot
	var errs = make([]error, 0)

	var openrtbResp openrtb.BidResponse

	// try it as a video
	if err := json.Unmarshal(response.Body, &openrtbResp); err != nil {
		errs = append(errs, err)

		// try it as a banner
		if err := json.Unmarshal(response.Body, &beachfrontResp); err != nil {
			errs = append(errs, err)
			return nil, errs
		} else {
			return postprocessBanner(beachfrontResp, id)
		}
	}

	return postprocessVideo(openrtbResp.SeatBid[0].Bid, xtrnal, uri, id)
}

func postprocessBanner(beachfrontResp []responseSlot, id string) ([]openrtb.Bid, []error) {

	var bids = make([]openrtb.Bid, len(beachfrontResp))
	var errs = make([]error, 0)

	for i := 0; i < len(beachfrontResp); i++ {
		bids[i] = openrtb.Bid{
			CrID:  beachfrontResp[i].CrID,
			ImpID: beachfrontResp[i].Slot,
			Price: beachfrontResp[i].Price,
			ID:    fmt.Sprintf("%sBanner", beachfrontResp[i].Slot),
			AdM:   beachfrontResp[i].Adm,
			H:     beachfrontResp[i].H,
			W:     beachfrontResp[i].W,
		}
	}

	return bids, errs
}

func postprocessVideo(bids []openrtb.Bid, xtrnal openrtb.BidRequest, uri string, id string) ([]openrtb.Bid, []error) {

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

func getBeachfrontExtension(imp openrtb.Imp) (openrtb_ext.ExtImpBeachfront, error) {
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

func getIP(ip string) string {
	// This will only effect testing. The backend will return "" for localhost IPs,
	// and seems not to know what IPv6 is, so just setting it to one that is not likely to
	// be used.
	if ip == "" || ip == "::1" || ip == "127.0.0.1" {
		return "192.168.255.255"
	}
	return ip
}

func removeVideoElement(slice []videoRequest, s int) []videoRequest {
	return append(slice[:s], slice[s+1:]...)
}

func NewBeachfrontBidder(bannerEndpoint string, extraAdapterInfo string) adapters.Bidder {
	var extraInfo ExtraInfo

	if len(extraAdapterInfo) == 0 {
		extraAdapterInfo = "{\"video_endpoint\":\"" + defaultVideoEndpoint + "\"}"
	}

	if err := json.Unmarshal([]byte(extraAdapterInfo), &extraInfo); err != nil {
		glog.Fatal("Invalid Beachfront extra adapter info: " + err.Error())
		return nil
	}

	if extraInfo.VideoEndpoint == "" {
		extraInfo.VideoEndpoint = defaultVideoEndpoint
	}

	return &BeachfrontAdapter{bannerEndpoint: bannerEndpoint, extraInfo: extraInfo}
}
