package beachfront

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

const Seat = "beachfront"
const BidCapacity = 5

const BannerEndpoint = "https://display.bfmio.com/prebid_display"
const VideoEndpoint = "https://reachms.bfmio.com/bid.json?exchange_id"

const VideoEndpointSuffix = "&prebidserver"

const beachfrontAdapterName = "BF_PREBID_S2S"
const beachfrontAdapterVersion = "0.7.0"

const minBidFloor = 0.01

const DefaultVideoWidth = 300
const DefaultVideoHeight = 250

type BeachfrontAdapter struct {
}

type beachfrontRequests struct {
	Banner    beachfrontBannerRequest
	NurlVideo []beachfrontNurlVideoRequest
	ADMVideo  []beachfrontADMVideoRequest
}

// ---------------------------------------------------
//              NurlVideo
// ---------------------------------------------------

type beachfrontADMVideoRequest struct {
	AppId   string             `json:"appId"`
	Request openrtb.BidRequest `json:"request"`
}
type beachfrontNurlVideoRequest struct {
	IsPrebid bool                  `json:"isPrebid"`
	AppId    string                `json:"appId"`
	ID       string                `json:"id"`
	Imp      []beachfrontVideoImp  `json:"imp"`
	Site     openrtb.Site          `json:"site"`
	Device   beachfrontVideoDevice `json:"device"`
	User     openrtb.User          `json:"user"`
	Cur      []string              `json:"cur"`
}

type beachfrontVideoImp struct {
	Video    beachfrontSize `json:"video"`
	Bidfloor float64        `json:"bidfloor"`
	Id       int            `json:"id"`
	ImpId    string         `json:"impid"`
	Secure   int8           `json:"secure"`
}

type beachfrontVideoDevice struct {
	UA string `json:"ua"`
	IP string `json:"ip"`
	JS string `json:"js"`
}

// ---------------------------------------------------
//              Banner
// ---------------------------------------------------
type beachfrontBannerRequest struct {
	Slots          []beachfrontSlot `json:"slots"`
	Domain         string           `json:"domain"`
	Page           string           `json:"page"`
	Referrer       string           `json:"referrer"`
	Search         string           `json:"search"`
	Secure         int8             `json:"secure"`
	DeviceOs       string           `json:"deviceOs"`
	DeviceModel    string           `json:"deviceModel"`
	IsMobile       int8             `json:"isMobile"`
	UA             string           `json:"ua"`
	Dnt            int8             `json:"dnt"`
	User           openrtb.User     `json:"user"`
	AdapterName    string           `json:"adapterName"`
	AdapterVersion string           `json:"adapterVersion"`
	IP             string           `json:"ip"`
	RequestID      string           `json:"requestId"`
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

func (a *BeachfrontAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var beachfrontRequests beachfrontRequests
	var errs = make([]error, 0, len(request.Imp))

	beachfrontRequests, errs = preprocess(request)

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

	reqCount += len(beachfrontRequests.ADMVideo)

	var reqs = make([]*adapters.RequestData, reqCount)

	var bump = 0

	// At most, I only ever have one banner request, and it does not need the cookie, so doing it first.
	if len(beachfrontRequests.Banner.Slots) > 0 {
		bytes, err := json.Marshal(beachfrontRequests.Banner)

		if err == nil {
			reqs[0] = &adapters.RequestData{
				Method:  "POST",
				Uri:     BannerEndpoint,
				Body:    bytes,
				Headers: headers,
			}
		} else {
			errs = append(errs, err)
		}

		reqCount--

		bump++
	}

	if request.User != nil && request.User.BuyerUID != "" && reqCount > 0 {
		headers.Add("Cookie", "__io_cid="+request.User.BuyerUID)
	}

	for j := 0; j < len(beachfrontRequests.ADMVideo); j++ {
		bytes, err := json.Marshal(beachfrontRequests.ADMVideo[j].Request)

		if err == nil {
			reqs[j+bump] = &adapters.RequestData{
				Method:  "POST",
				Uri:     VideoEndpoint + "=" + beachfrontRequests.ADMVideo[j].AppId,
				Body:    bytes,
				Headers: headers,
			}

		} else {
			errs = append(errs, err)
		}
	}

	for j := 0; j < len(beachfrontRequests.NurlVideo); j++ {
		bytes, err := json.Marshal(beachfrontRequests.NurlVideo[j])

		if err == nil {
			reqs[j+bump] = &adapters.RequestData{
				Method:  "POST",
				Uri:     VideoEndpoint + "=" + beachfrontRequests.NurlVideo[j].AppId + VideoEndpointSuffix,
				Body:    bytes,
				Headers: headers,
			}
		} else {
			errs = append(errs, err)
		}

		bump++
	}

	return reqs, errs
}

func preprocess(request *openrtb.BidRequest) (beachfrontReqs beachfrontRequests, errs []error) {
	var videoImps = make([]openrtb.Imp, 0)
	var bannerImps = make([]openrtb.Imp, 0)

	for i := 0; i < len(request.Imp); i++ {
		if request.Imp[i].Banner != nil {
			bannerImps = append(bannerImps, request.Imp[i])
		}

		if request.Imp[i].Video != nil {
			videoImps = append(videoImps, request.Imp[i])
		}
	}

	if len(bannerImps) == 0 && len(videoImps) == 0 {
		errs = append(errs, errors.New("no valid impressions were found in the request"))
		return
	}

	if len(bannerImps) > 0 {
		request.Imp = bannerImps
		beachfrontReqs.Banner, errs = getBannerRequest(request)
	}

	if len(videoImps) > 0 {
		request.Imp = videoImps

		var videoErrs []error
		beachfrontReqs.ADMVideo, videoErrs = getADMVideoRequests(request)
		errs = append(errs, videoErrs...)
	}

	return
}

func newBeachfrontBannerRequest() beachfrontBannerRequest {
	r := beachfrontBannerRequest{}
	r.AdapterName = beachfrontAdapterName
	r.AdapterVersion = beachfrontAdapterVersion

	return r
}

func newBeachfrontADMVideoRequest() beachfrontADMVideoRequest {
	r := beachfrontADMVideoRequest{}

	return r
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
		error = errors.New("unable to determine the banner appId from the supplied extension")
	}

	return appid, error
}

/*
getBannerRequest, singular. A "Slot" is an "imp," and each Slot can have an AppId, so just one
request to the beachfront banner endpoint gets all banner Imps.
*/
func getBannerRequest(request *openrtb.BidRequest) (beachfrontBannerRequest, []error) {
	var bfr beachfrontBannerRequest
	var errs = make([]error, 0, len(request.Imp))

	bfr = newBeachfrontBannerRequest()

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

		slot := beachfrontSlot{}
		slot.Id = appid

		if beachfrontExt.BidFloor > minBidFloor  {
			slot.Bidfloor = beachfrontExt.BidFloor
		}

		slot.Slot = request.Imp[i].ID

		for j := 0; j < len(request.Imp[i].Banner.Format); j++ {

			slot.Sizes = append(slot.Sizes, beachfrontSize{
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

	site := getSite(request)
	bfr.IsMobile = site.Mobile
	bfr.Page = site.Page
	bfr.Domain = site.Domain

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

	if request.Imp[0].Secure != nil {
		bfr.Secure = *request.Imp[0].Secure
	}

	return bfr, errs
}


func getADMVideoRequests(request *openrtb.BidRequest) ([]beachfrontADMVideoRequest, []error) {
	var beachfrontReqs = make([]beachfrontADMVideoRequest, len(request.Imp))
	var errs = make([]error, 0, len(request.Imp))
	var bad = make([]int, 0)

	for i := 0; i < len(request.Imp); i++ {
		beachfrontExt, err := getBeachfrontExtension(request.Imp[i])

		if err != nil {
			// Failed to extract the beachfrontExt, so this request is junk.
			bad = append(bad, i)
			errs = append(errs, err)
			continue
		}

		appid, err := getAppId(beachfrontExt, openrtb_ext.BidTypeVideo)

		if err != nil {
			// Failed to get an appid, so this request is junk.
			bad = append(bad, i)
			errs = append(errs, err)
			continue
		}


		beachfrontReqs[i] = newBeachfrontADMVideoRequest()
		beachfrontReqs[i].AppId = appid

		imp := openrtb.Imp{}
		imp = request.Imp[i]

		beachfrontReqs[i].Request = *request
		beachfrontReqs[i].Request.Imp = make([]openrtb.Imp, 1, 1)
		beachfrontReqs[i].Request.Imp[0] = imp

		beachfrontReqs[i].Request.Device.IP	= getIP(beachfrontReqs[i].Request.Device.IP	)

		if beachfrontExt.BidFloor > minBidFloor {
			beachfrontReqs[i].Request.Imp[0].BidFloor = beachfrontExt.BidFloor
		}

		beachfrontReqs[i].Request.Ext = nil
	}

	// Strip out any failed requests
	if len(bad) > 0 {
		for i := 0; i < len(bad); i++ {
			beachfrontReqs = removeRTBVideoElement(beachfrontReqs, bad[i])
		}

	}
	return beachfrontReqs, errs
}



func (a *BeachfrontAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var bids []openrtb.Bid

	if response.StatusCode == http.StatusOK && len(response.Body) <= 2 {
		return nil, nil
	}

	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{fmt.Errorf("unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode)}
	}

	bids, errs := postprocess(response, externalRequest, internalRequest.ID)

	if len(errs) != 0 {
		return nil, errs
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(BidCapacity)

	for i := 0; i < len(bids); i++ {
		bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
			Bid:     &bids[i],
			BidType: getBidType(externalRequest),
		})
	}

	return bidResponse, errs
}

func postprocess(response *adapters.ResponseData, externalRequest *adapters.RequestData, id string) ([]openrtb.Bid, []error) {
	var beachfrontResp []beachfrontResponseSlot
	var errs = make([]error, 0)

	var openrtbResp openrtb.BidResponse

	// try it as a video
	if err := json.Unmarshal(response.Body, &openrtbResp); err != nil {

		// try it as a banner
		if err := json.Unmarshal(response.Body, &beachfrontResp); err != nil {
			errs = append(errs, err)
			return nil, errs
		} else {
			return postprocessBanner(beachfrontResp, externalRequest, id)
		}
	}

	return postprocessVideo(openrtbResp.SeatBid[0].Bid, externalRequest, id)
}

func postprocessBanner(beachfrontResp []beachfrontResponseSlot, externalRequest *adapters.RequestData, id string) ([]openrtb.Bid, []error) {

	var xtrnal beachfrontBannerRequest
	var bids = make([]openrtb.Bid, len(beachfrontResp))
	var errs = make([]error, 0)

	if err := json.Unmarshal(externalRequest.Body, &xtrnal); err != nil {
		errs = append(errs, err)
		return bids, errs
	}

	for i := 0; i < len(beachfrontResp); i++ {
		bids[i] = openrtb.Bid{
			CrID:  beachfrontResp[i].CrID,
			ImpID: beachfrontResp[i].Slot,
			Price: beachfrontResp[i].Price,
			ID:    id,
			AdM:   beachfrontResp[i].Adm,
			H:     beachfrontResp[i].H,
			W:     beachfrontResp[i].W,
		}
	}

	return bids, errs
}

func postprocessVideo(bids []openrtb.Bid, externalRequest *adapters.RequestData, id string) ([]openrtb.Bid, []error) {

	var xtrnal beachfrontNurlVideoRequest
	var errs = make([]error, 0)

	if xtrnal.IsPrebid {

		if err := json.Unmarshal(externalRequest.Body, &xtrnal); err != nil {
			errs = append(errs, err)
			return bids, errs
		}

		for i := 0; i < len(bids); i++ {
			crid := extractVideoCrid(bids[i].NURL)

			bids[i].CrID = crid
			bids[i].ImpID = xtrnal.Imp[i].ImpId
			bids[i].H = xtrnal.Imp[i].Video.H
			bids[i].W = xtrnal.Imp[i].Video.W
			bids[i].ID = id
		}
	} else {
		return bids, errs
	}

	return bids, errs
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
			Message: fmt.Sprintf("ignoring imp id=%s, error while decoding extImpBidder, err: %s", imp.ID, err),
		}
	}

	return beachfrontExt, err
}

func getDomain(page string) string {
	protoUrl := strings.Split(page, "//")
	var domainPage string

	if len(protoUrl) > 1 {
		domainPage = protoUrl[1]
	} else {
		domainPage = protoUrl[0]
	}

	return strings.Split(domainPage, "/")[0]

}

func getSite(request *openrtb.BidRequest) (site openrtb.Site) {

	if request.App != nil {

		if request.App.Domain == "" {
			site.Domain = getDomain(request.App.Domain)
		} else {
			site.Domain = request.App.Domain
		}

		site.Page = request.App.Bundle
		site.Mobile = 1
	} else {
		if request.Site.Page != "" {
			if request.Site.Domain == "" {
				site.Domain = getDomain(request.Site.Page)
			} else {
				site.Domain = request.Site.Domain
			}
			site.Page = request.Site.Page
		}

		site.Mobile = 0
	}
	return site
}

func getIP(ip string) string {
		// This will only effect testing. The backend will return "" for localhost IPs,
		// and seems not to know what IPv6 is, so just setting it to one that is not likely to
		// be used.
		if ip == "::1" || ip == "127.0.0.1" {
			return "192.168.255.255"
		}
	return ip
}

func getBidType(externalRequest *adapters.RequestData) openrtb_ext.BidType {
	t := strings.Split(externalRequest.Uri, "=")[0]
	if t == VideoEndpoint {
		return openrtb_ext.BidTypeVideo
	}

	return openrtb_ext.BidTypeBanner
}

func extractVideoCrid(nurl string) string {
	chunky := strings.SplitAfter(nurl, ":")
	return strings.TrimSuffix(chunky[2], ":")
}

func removeVideoElement(slice []beachfrontNurlVideoRequest, s int) []beachfrontNurlVideoRequest {
	return append(slice[:s], slice[s+1:]...)
}

func removeRTBVideoElement(slice []beachfrontADMVideoRequest, s int) []beachfrontADMVideoRequest {
	return append(slice[:s], slice[s+1:]...)
}

func NewBeachfrontBidder() *BeachfrontAdapter {
	return &BeachfrontAdapter{}
}
