package beachfront

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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
const beachfrontAdapterVersion = "1.0.0"

const minBidFloor = 0.01

type BeachfrontAdapter struct {
	bannerEndpoint string
	extraInfo      ExtraInfo
}

type ExtraInfo struct {
	VideoEndpoint string `json:"video_endpoint,omitempty"`
}

type requests struct {
	Banner    bannerRequest
	NurlVideo []openrtb.BidRequest
	ADMVideo  map[string]*openrtb.BidRequest
}

// ---------------------------------------------------
//              Banner
// ---------------------------------------------------
type bannerRequest struct {
	Slots          []slot       `json:"slots"`
	Domain         string       `json:"domain"`
	Page           string       `json:"page"`
	Referrer       string       `json:"referrer"`
	Search         string       `json:"search"`
	Secure         int8         `json:"secure"`
	DeviceOs       string       `json:"deviceOs"`
	DeviceModel    string       `json:"deviceModel"`
	IsMobile       int8         `json:"isMobile"`
	UA             string       `json:"ua"`
	Dnt            int8         `json:"dnt"`
	User           openrtb.User `json:"user"`
	AdapterName    string       `json:"adapterName"`
	AdapterVersion string       `json:"adapterVersion"`
	IP             string       `json:"ip"`
	RequestID      string       `json:"requestId"`
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
	var beachfrontReqs requests

	wantsBanner, videoImps, errs := sortImps(request)
	if wantsBanner {
		beachfrontReqs.Banner, errs = getBannerRequest(request, errs)
	}

	if len(videoImps) > 0 {
		beachfrontReqs.NurlVideo, errs = getNurlRequests(
			request,
			videoImps,
			errs)
	}

	if len(videoImps) > 0 {
		beachfrontReqs.ADMVideo, errs = getAdmRequests(
			request,
			videoImps,
			errs)
	}

	if len(errs) > 0 && errortypes.ContainsFatalError(errs) {
		return nil, errs
	}

	var reqs = make(
		[]*adapters.RequestData,
		0,
		len(beachfrontReqs.ADMVideo)+len(beachfrontReqs.NurlVideo)+1)

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

	// one banner request potentially with multiple imps, and it does
	// not need the cookie, so doing it first.
	if len(beachfrontReqs.Banner.Slots) > 0 {
		bytes, err := json.Marshal(beachfrontReqs.Banner)

		if err == nil {
			reqs = append(reqs, &adapters.RequestData{
				Method:  "POST",
				Uri:     a.bannerEndpoint,
				Body:    bytes,
				Headers: headers,
			})
		} else {
			errs = append(errs, err)
		}
	}

	if request.User != nil && request.User.BuyerUID != "" {
		headers.Add("Cookie", "__io_cid="+request.User.BuyerUID)
	}

	// n nurl requests, each of which is one imp
	for i := 0; i < len(beachfrontReqs.NurlVideo); i++ {
		bytes, err := json.Marshal(beachfrontReqs.NurlVideo[i])
		ext, err := getBeachfrontExtension(beachfrontReqs.NurlVideo[i].Imp[0], openrtb_ext.BidTypeVideo)

		if err == nil {
			reqs = append(reqs, &adapters.RequestData{
				Method:  "POST",
				Uri:     a.extraInfo.VideoEndpoint + "=" + ext.AppId + nurlVideoEndpointSuffix,
				Body:    append([]byte(`{"isPrebid":true,`), bytes[1:]...),
				Headers: headers,
			})
		} else {
			errs = append(errs, err)
		}
	}

	// n adm requests, some of which may have multiple imps
	for appId, adm := range beachfrontReqs.ADMVideo {

		bytes, err := json.Marshal(adm)

		if err == nil {
			reqs = append(reqs, &adapters.RequestData{
				Method:  "POST",
				Uri:     a.extraInfo.VideoEndpoint + "=" + appId,
				Body:    bytes,
				Headers: headers,
			})
		} else {
			errs = append(errs, err)
		}
	}

	return reqs, errs
}

func sortImps(request *openrtb.BidRequest) (bool, []openrtb.Imp, []error) {
	var videoImps = make([]openrtb.Imp, 0, len(request.Imp))
	var errs = make([]error, 0, len(request.Imp))
	var wantsBanner bool

	for i := 0; i < len(request.Imp); i++ {
		if !wantsBanner {
			wantsBanner = request.Imp[i].Banner != nil
			// In a previous version, I was testing for a valid size for the Banner element, but
			// that is already done. If the size is not valid, I will never get here as the entire
			// request will get tossed in endpoints/openrtb2/auction.go (around L441 as of this writing).
		}

		if request.Imp[i].Video != nil {
			if request.Site != nil {
				if request.Site.Page != "" {
					// This will overwrite the secure value if it is included, but if
					// a web request is not https, it's not secure, and vice versa so this seems authoritative.
					secure := isPageSecure(request.Site.Page)
					request.Imp[i].Secure = &secure
				} else {
					errs = append(errs,
						errors.New(
							fmt.Sprintf("video impresion %s did not include a page value "+
								"which is required for a web request",
								request.Imp[i].ID)))
					continue
				}

				// Since request.Site is nil, request.App must be not nil or this whole request would be invalid
				// and would have already been discarded, but we could have gotten this far with Bundle == ""
				// as it is not required by RTB, but it is required by beachfront. Ditto for Device.IFA.
			} else {

				var skip bool
				if request.App.Bundle == "" {
					errs = append(errs,
						errors.New(
							fmt.Sprintf("video impression %s did not include an "+
								"App.Bundle value which is required for an app request",
								request.Imp[i].ID)))
					skip = true
				}

				if request.Device == nil || request.Device.IFA == "" {
					errs = append(errs,
						errors.New(
							fmt.Sprintf("video impression %s did not include a "+
								"Device.IFA value which is required for an app request",
								request.Imp[i].ID)))
					skip = true
				}

				if skip {
					continue
				}

				// This is a valid App request. Make sure Imp[i].Secure is set.
				if request.Imp[i].Secure == nil {
					var secure int8 = 0
					request.Imp[i].Secure = &secure
				}
			}

			videoImp := request.Imp[i]
			videoImp.Banner = nil
			videoImps = append(videoImps, videoImp)
		}
	}

	if len(videoImps) == 0 && !wantsBanner {
		errs = append(errs, &errortypes.BadInput{
			Message: "no valid impressions were found in the request",
		})
		return false, videoImps, errs
	}

	return wantsBanner, videoImps, errs
}

func deepCopyRequest(request *openrtb.BidRequest) openrtb.BidRequest {
	var content openrtb.Content
	var publisher openrtb.Publisher

	requestCopy := *request

	if requestCopy.Site != nil {
		site := *requestCopy.Site
		if site.Content != nil {
			content = *site.Content
			site.Content = &content
		}
		if site.Publisher != nil {
			publisher = *site.Publisher
			site.Publisher = &publisher
		}

		requestCopy.Site = &site
	}

	if requestCopy.App != nil {
		app := *requestCopy.App

		if app.Content != nil {
			content = *app.Content
			app.Content = &content
		}
		if app.Publisher != nil {
			publisher = *app.Publisher
			app.Publisher = &publisher
		}
		requestCopy.App = &app
	}

	if requestCopy.Device != nil {
		device := *requestCopy.Device

		if device.ConnectionType != nil {
			ctype := *device.ConnectionType
			device.ConnectionType = &ctype
		}

		if device.Geo != nil {
			deviceGeo := *device.Geo
			device.Geo = &deviceGeo
		}

		if device.DNT != nil {
			dnt := *device.DNT
			device.DNT = &dnt
		}

		if device.Lmt != nil {
			lmt := *device.Lmt
			device.Lmt = &lmt
		}

		requestCopy.Device = &device
	}

	if requestCopy.User != nil {
		user := *requestCopy.User

		if user.Geo != nil {
			userGeo := *user.Geo
			user.Geo = &userGeo
		}

		requestCopy.User = &user
	}

	if requestCopy.Source != nil {
		source := *requestCopy.Source
		requestCopy.Source = &source
	}

	if requestCopy.Regs != nil {
		regs := *requestCopy.Regs
		requestCopy.Regs = &regs
	}

	return requestCopy
}

func deepCopyImp(imp *openrtb.Imp) openrtb.Imp {
	impCopy := *imp

	if impCopy.Banner != nil {
		banner := *impCopy.Banner
		if banner.W != nil {
			w := *banner.W
			banner.W = &w
		}

		if banner.H != nil {
			h := *banner.H
			banner.W = &h
		}

		if banner.Pos != nil {
			pos := *banner.Pos
			banner.Pos = &pos
		}

		impCopy.Banner = &banner
	}

	if impCopy.Video != nil {
		video := *impCopy.Video

		if video.StartDelay != nil {
			sd := *video.StartDelay
			video.StartDelay = &sd
		}

		if video.Skip != nil {
			skip := *video.Skip
			video.Skip = &skip
		}

		if video.Pos != nil {
			pos := *video.Pos
			video.Pos = &pos
		}
		impCopy.Video = &video
	}

	if impCopy.PMP != nil {
		pmp := *impCopy.PMP
		impCopy.PMP = &pmp
	}

	if impCopy.Secure != nil {
		secure := *impCopy.Secure
		impCopy.Secure = &secure
	}

	return impCopy
}

func getBannerRequest(request *openrtb.BidRequest, errs []error) (bannerRequest, []error) {
	var secure int8
	var bannerReq bannerRequest
	bannerReq, secure, errs = impsToSlots(request.Imp, errs)

	if len(bannerReq.Slots) == 0 {
		errs = append(errs, errors.New("unable to construct a valid banner request"))
		return bannerReq, errs
	}

	if request.Site != nil && request.Site.Page != "" {
		bannerReq.Secure = isPageSecure(request.Site.Page)
	} else {
		bannerReq.Secure = secure
	}

	if request.Device != nil {
		bannerReq.IP = getIP(request.Device.IP)
		bannerReq.DeviceModel = request.Device.Model
		bannerReq.DeviceOs = request.Device.OS
		if request.Device.DNT != nil {
			bannerReq.Dnt = *request.Device.DNT
		}
		if request.Device.UA != "" {
			bannerReq.UA = request.Device.UA
		}
	}

	var t = fallBackDeviceType(request)

	if t == openrtb.DeviceTypeMobileTablet {
		bannerReq.Page = request.App.Bundle
		if request.App.Domain != "" {
			bannerReq.Domain = request.App.Domain
		}

		bannerReq.IsMobile = 1
	} else if t == openrtb.DeviceTypePersonalComputer {
		bannerReq.Page = request.Site.Page
		if request.Site.Domain == "" {
			bannerReq.Domain = getDomain(request.Site.Page)
		} else {
			bannerReq.Domain = request.Site.Domain
		}

		bannerReq.IsMobile = 0
	}

	if request.User != nil && request.User.ID != "" {
		bannerReq.User.ID = request.User.ID
	}

	if request.User != nil && request.User.BuyerUID != "" {
		if bannerReq.User.BuyerUID == "" {
			bannerReq.User.BuyerUID = request.User.BuyerUID
		}
	}

	bannerReq.RequestID = request.ID
	bannerReq.AdapterName = beachfrontAdapterName
	bannerReq.AdapterVersion = beachfrontAdapterVersion

	return bannerReq, errs
}

func impsToSlots(imps []openrtb.Imp, errs []error) (bannerRequest, int8, []error) {
	var bfr bannerRequest
	var secure int8 = 0

	for i := 0; i < len(imps); i++ {
		var imp = imps[i]
		if imp.Banner == nil {
			continue
		}

		if imp.Secure != nil {
			secure = *imp.Secure
		} else {
			secure = 0
		}
		beachfrontExt, err := getBeachfrontExtension(imp, openrtb_ext.BidTypeBanner)

		if err != nil {
			errs = append(errs, errors.New(fmt.Sprintf("%s (on banner imp id: %s)", err, imp.ID)))
			continue
		}

		appid, err := getAppId(beachfrontExt, openrtb_ext.BidTypeBanner)

		if err != nil {
			errs = append(errs, errors.New(fmt.Sprintf("%s (on banner imp id: %s)", err, imp.ID)))
			continue
		}

		slot := slot{
			Id:       appid,
			Slot:     imp.ID,
			Bidfloor: beachfrontExt.BidFloor,
		}

		if beachfrontExt.BidFloor <= minBidFloor {
			slot.Bidfloor = 0
		}

		for j := 0; j < len(imp.Banner.Format); j++ {
			slot.Sizes = append(slot.Sizes, size{
				H: imp.Banner.Format[j].H,
				W: imp.Banner.Format[j].W,
			})
		}

		if len(slot.Sizes) == 0 {
			if imp.Banner.W == nil || imp.Banner.H == nil {
				errs = append(errs,
					fmt.Errorf("request.imp[%d].banner has no sizes. Define \"w\" and \"h\", or include \"format\" elements", i))
				continue
			} else {
				slot.Sizes = append(slot.Sizes, size{
					W: *imp.Banner.W,
					H: *imp.Banner.H,
				})
			}

		}

		bfr.Slots = append(bfr.Slots, slot)
	}

	return bfr, secure, errs
}

func getNurlRequests(request *openrtb.BidRequest, imps []openrtb.Imp, errs []error) ([]openrtb.BidRequest, []error) {
	var nurlReqs []openrtb.BidRequest

	for i := 0; i < len(imps); i++ {
		ext, err := getBeachfrontExtension(imps[i], openrtb_ext.BidTypeVideo)

		if err != nil {
			errs = append(errs, errors.New(fmt.Sprintf("%s (on video imp id: %s)", err, imps[i].ID)))
			continue
		}

		if ext.VideoResponseType == "nurl" || ext.VideoResponseType == "both" {

			r := deepCopyRequest(request)
			r.Imp = []openrtb.Imp{imps[i]}

			nurlReqs = append(nurlReqs, prepVideoRequest(r))
		}

	}

	return nurlReqs, errs
}

func getAdmRequests(request *openrtb.BidRequest, imps []openrtb.Imp, errs []error) (map[string]*openrtb.BidRequest, []error) {
	var admMap = map[string]*openrtb.BidRequest{}
	for i := 0; i < len(imps); i++ {
		ext, err := getBeachfrontExtension(imps[i], openrtb_ext.BidTypeVideo)

		if err != nil {
			errs = append(errs, errors.New(fmt.Sprintf("%s (on video imp id: %s)", err, imps[i].ID)))
			continue
		}

		if ext.VideoResponseType == "adm" || ext.VideoResponseType == "both" {
			_, exists := admMap[ext.AppId]
			if exists {
				admMap[ext.AppId].Imp = append(admMap[ext.AppId].Imp, deepCopyImp(&imps[i]))
			} else {
				r := deepCopyRequest(request)
				r.Imp = []openrtb.Imp{deepCopyImp(&imps[i])}

				r = prepVideoRequest(r)
				admMap[ext.AppId] = &r
			}
		}
	}

	return admMap, errs
}

func fallBackDeviceType(request *openrtb.BidRequest) openrtb.DeviceType {
	if request.Site != nil {
		return openrtb.DeviceTypePersonalComputer
	}

	return openrtb.DeviceTypeMobileTablet
}

func prepVideoRequest(bfReq openrtb.BidRequest) openrtb.BidRequest {
	if bfReq.Site != nil && bfReq.Site.Domain == "" && bfReq.Site.Page != "" {
		bfReq.Site.Domain = getDomain(bfReq.Site.Page)
	}

	if bfReq.App != nil && bfReq.App.Domain == "" && bfReq.App.Bundle != "" {
		var chunks = strings.Split(strings.Trim(bfReq.App.Bundle, "_"), ".")

		if len(chunks) > 1 {
			bfReq.App.Domain =
				fmt.Sprintf("%s.%s", chunks[len(chunks)-(len(chunks)-1)], chunks[0])
		}

	}

	if bfReq.Device != nil {
		bfReq.Device.IP = getIP(bfReq.Device.IP)

		if bfReq.Device.DeviceType == 0 {
			bfReq.Device.DeviceType = fallBackDeviceType(&bfReq)
		}
	}

	if len(bfReq.Cur) == 0 {
		bfReq.Cur = []string{"USD"}
	}

	return bfReq
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

	bids, errs = postprocess(response, xtrnal, externalRequest.Uri)

	if len(errs) != 0 {
		return nil, errs
	}

	var dur videoBidExtension
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(BidCapacity)
	for i := 0; i < len(bids); i++ {

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

func postprocess(response *adapters.ResponseData, xtrnal openrtb.BidRequest, uri string) ([]openrtb.Bid, []error) {
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
			return postprocessBanner(beachfrontResp)
		}
	}

	if len(openrtbResp.SeatBid) != 0 {
		return postprocessVideo(openrtbResp.SeatBid[0].Bid, xtrnal, uri)
	}

	return nil, errs
}

func postprocessBanner(beachfrontResp []responseSlot) ([]openrtb.Bid, []error) {

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

func postprocessVideo(bids []openrtb.Bid, xtrnal openrtb.BidRequest, uri string) ([]openrtb.Bid, []error) {

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

func getBeachfrontExtension(imp openrtb.Imp, bidType openrtb_ext.BidType) (openrtb_ext.ExtImpBeachfront, error) {
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

	if err != nil {
		return beachfrontExt, err
	}

	beachfrontExt.AppId, err = getAppId(beachfrontExt, bidType)

	if err != nil {
		return beachfrontExt, err
	}

	if beachfrontExt.VideoResponseType != "nurl" && beachfrontExt.VideoResponseType != "both" {
		beachfrontExt.VideoResponseType = "adm"
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

func isPageSecure(page string) int8 {
	protoURL := strings.Split(page, "://")
	if len(protoURL) > 1 && protoURL[0] == "https" {
		return 1
	}

	return 0
}

/**
Returns an authoritative appId (exchange_id) for a specific impression
*/
func getAppId(ext openrtb_ext.ExtImpBeachfront, media openrtb_ext.BidType) (string, error) {
	var appid string
	var err error

	if ext.AppId != "" {
		appid = ext.AppId
	} else {
		if media == openrtb_ext.BidTypeVideo && ext.AppIds.Video != "" {
			appid = ext.AppIds.Video
		} else if media == openrtb_ext.BidTypeBanner && ext.AppIds.Banner != "" {
			appid = ext.AppIds.Banner
		}
	}

	if appid == "" {
		err = errors.New("unable to determine the appId(s) from the supplied extension")
	}

	return appid, err
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
