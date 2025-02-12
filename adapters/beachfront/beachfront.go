package beachfront

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
)

const Seat = "beachfront"
const BidCapacity = 5

const defaultVideoEndpoint = "https://reachms.bfmio.com/bid.json?exchange_id"

const nurlVideoEndpointSuffix = "&prebidserver"

const beachfrontAdapterName = "BF_PREBID_S2S"
const beachfrontAdapterVersion = "1.0.0"

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
//
//	Banner
//
// ---------------------------------------------------
type beachfrontBannerRequest struct {
	Slots          []beachfrontSlot     `json:"slots"`
	Domain         string               `json:"domain"`
	Page           string               `json:"page"`
	Referrer       string               `json:"referrer"`
	Search         string               `json:"search"`
	Secure         int8                 `json:"secure"`
	DeviceOs       string               `json:"deviceOs"`
	DeviceModel    string               `json:"deviceModel"`
	IsMobile       int8                 `json:"isMobile"`
	UA             string               `json:"ua"`
	Dnt            int8                 `json:"dnt"`
	User           openrtb2.User        `json:"user"`
	AdapterName    string               `json:"adapterName"`
	AdapterVersion string               `json:"adapterVersion"`
	IP             string               `json:"ip"`
	RequestID      string               `json:"requestId"`
	Real204        bool                 `json:"real204"`
	SChain         openrtb2.SupplyChain `json:"schain,omitempty"`
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
	beachfrontRequests, errs := preprocess(request, reqInfo)

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

	if len(beachfrontRequests.Banner.Slots) > 0 {
		bytes, err := json.Marshal(beachfrontRequests.Banner)

		if err == nil {
			reqs[0] = &adapters.RequestData{
				Method:  "POST",
				Uri:     a.bannerEndpoint,
				Body:    bytes,
				Headers: headers,
				ImpIDs:  getBannerImpIDs(beachfrontRequests.Banner.Slots),
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
				ImpIDs:  openrtb_ext.GetImpIDs(beachfrontRequests.ADMVideo[j].Request.Imp),
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
				ImpIDs:  openrtb_ext.GetImpIDs(beachfrontRequests.NurlVideo[j].Request.Imp),
			}
		} else {
			errs = append(errs, err)
		}
	}

	return reqs, errs
}

func preprocess(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) (beachfrontReqs beachfrontRequests, errs []error) {
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
		beachfrontReqs.Banner, errs = getBannerRequest(request, reqInfo)
	}

	if len(videoImps) > 0 {
		var videoErrs []error
		var videoList []beachfrontVideoRequest

		request.Imp = videoImps
		request.Ext = nil

		videoList, videoErrs = getVideoRequests(request, reqInfo)
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
	return schain, jsonutil.Unmarshal(request.Source.Ext, &schain)
}

func getBannerRequest(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) (beachfrontBannerRequest, []error) {
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
			errs = append(errs, err)
			continue
		}

		if fatal, err := setBidFloor(&beachfrontExt, &request.Imp[i], reqInfo); err != nil {
			errs = append(errs, err)
			if fatal {
				continue
			}
		}

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

	if t == adcom1.DeviceMobile {
		bfr.Page = request.App.Bundle
		if request.App.Domain == "" {
			bfr.Domain = getDomain(request.App.Domain)
		} else {
			bfr.Domain = request.App.Domain
		}

		bfr.IsMobile = 1
	} else if t == adcom1.DevicePC {
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

func fallBackDeviceType(request *openrtb2.BidRequest) adcom1.DeviceType {
	if request.Site != nil {
		return adcom1.DevicePC
	}

	return adcom1.DeviceMobile
}

func getVideoRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]beachfrontVideoRequest, []error) {
	var bfReqs = make([]beachfrontVideoRequest, len(request.Imp))
	var errs = make([]error, 0, len(request.Imp))
	var failedRequestIndicies = make([]int, 0)

	for i := 0; i < len(request.Imp); i++ {
		beachfrontExt, err := getBeachfrontExtension(request.Imp[i])

		if err != nil {
			failedRequestIndicies = append(failedRequestIndicies, i)
			errs = append(errs, err)
			continue
		}

		appid, err := getAppId(beachfrontExt, openrtb_ext.BidTypeVideo)
		bfReqs[i].AppId = appid

		if err != nil {
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
			deviceCopy.DeviceType = fallBackDeviceType(request)
		}
		bfReqs[i].Request.Device = &deviceCopy

		imp := request.Imp[i]

		imp.Banner = nil
		imp.Ext = nil
		imp.Secure = &secure
		if fatal, err := setBidFloor(&beachfrontExt, &imp, reqInfo); err != nil {
			errs = append(errs, err)
			if fatal {
				failedRequestIndicies = append(failedRequestIndicies, i)
				continue
			}
		}

		wNilOrZero := imp.Video.W == nil || *imp.Video.W == 0
		hNilOrZero := imp.Video.H == nil || *imp.Video.H == 0
		if wNilOrZero || hNilOrZero {
			videoCopy := *imp.Video

			if wNilOrZero {
				videoCopy.W = ptrutil.ToPtr[int64](defaultVideoWidth)
			}

			if hNilOrZero {
				videoCopy.H = ptrutil.ToPtr[int64](defaultVideoHeight)
			}

			imp.Video = &videoCopy
		}

		if len(bfReqs[i].Request.Cur) == 0 {
			bfReqs[i].Request.Cur = make([]string, 1)
			bfReqs[i].Request.Cur[0] = "USD"
		}

		bfReqs[i].Request.Imp = nil
		bfReqs[i].Request.Imp = make([]openrtb2.Imp, 1)
		bfReqs[i].Request.Imp[0] = imp

	}

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

	if err := jsonutil.Unmarshal(externalRequest.Body, &xtrnal); err != nil {
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

		if err := jsonutil.Unmarshal(bids[i].Ext, &dur); err == nil && dur.Duration > 0 {

			impVideo := openrtb_ext.ExtBidPrebidVideo{
				Duration: int(dur.Duration),
			}

			if len(bids[i].Cat) > 0 {
				impVideo.PrimaryCategory = bids[i].Cat[0]
			}

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

func setBidFloor(ext *openrtb_ext.ExtImpBeachfront, imp *openrtb2.Imp, reqInfo *adapters.ExtraRequestInfo) (bool, error) {
	var initialImpBidfloor float64 = imp.BidFloor
	var err error

	if imp.BidFloorCur != "" && strings.ToUpper(imp.BidFloorCur) != "USD" && imp.BidFloor > 0 {
		imp.BidFloor, err = reqInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, "USD")

		var convertedFromCurrency = imp.BidFloorCur
		imp.BidFloorCur = "USD"

		if err != nil {
			if ext.BidFloor > minBidFloor {
				imp.BidFloor = ext.BidFloor
				return false, &errortypes.Warning{
					Message: fmt.Sprintf("The following error was recieved from the currency converter while attempting to convert the imp.bidfloor value of %.2f from %s to USD:\n%s\nThe provided value of imp.ext.beachfront.bidfloor, %.2f USD is being used as a fallback.",
						initialImpBidfloor,
						convertedFromCurrency,
						err,
						ext.BidFloor,
					),
				}
			} else {
				return true, &errortypes.BadInput{
					Message: fmt.Sprintf("The following error was recieved from the currency converter while attempting to convert the imp.bidfloor value of %.2f from %s to USD:\n%s\nA value of imp.ext.beachfront.bidfloor was not provided. The bid is being skipped.",
						initialImpBidfloor,
						convertedFromCurrency,
						err,
					),
				}
			}
		}
	}

	if imp.BidFloor < ext.BidFloor {
		imp.BidFloor = ext.BidFloor
	}

	if imp.BidFloor > minBidFloor {
		imp.BidFloorCur = "USD"
	} else {
		imp.BidFloor = 0
		imp.BidFloorCur = ""
	}

	return false, nil
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

	if err := jsonutil.Unmarshal(response.Body, &openrtbResp); err != nil || len(openrtbResp.SeatBid) == 0 {

		if err := jsonutil.Unmarshal(response.Body, &beachfrontResp); err != nil {
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
			bids[i].H = ptrutil.ValueOrDefault(xtrnal.Imp[i].Video.H)
			bids[i].W = ptrutil.ValueOrDefault(xtrnal.Imp[i].Video.W)
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

	if err = jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return beachfrontExt, &errortypes.BadInput{
			Message: fmt.Sprintf("ignoring imp id=%s, error while decoding extImpBidder, err: %s", imp.ID, err),
		}
	}

	if err = jsonutil.Unmarshal(bidderExt.Bidder, &beachfrontExt); err != nil {
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
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
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
	if err := jsonutil.Unmarshal([]byte(v), &extraInfo); err != nil {
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

func getBannerImpIDs(bfs []beachfrontSlot) []string {
	impIDs := make([]string, len(bfs))
	for i := range bfs {
		impIDs[i] = bfs[i].Slot
	}
	return impIDs
}
