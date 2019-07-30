package beachfront

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
	"strconv"
	"strings"
)

const Seat = "beachfront"
const BidCapacity = 5

const BannerEndpoint = "https://display.bfmio.com/prebid_display"
const VideoEndpoint = "https://reachms.bfmio.com/bid.json?exchange_id="

const VideoEndpointSuffix = "&prebidserver"

const beachfrontAdapterName = "BF_PREBID_S2S"
const beachfrontAdapterVersion = "0.6.0"

const DefaultVideoWidth = 300
const DefaultVideoHeight = 250

type BeachfrontAdapter struct {
}

type BeachfrontRequests struct {
	Banner BeachfrontBannerRequest
	Video  BeachfrontVideoRequest
	Audio  openrtb.Audio
	Native openrtb.Native
}

// ---------------------------------------------------
//              Video
// ---------------------------------------------------

type BeachfrontVideoRequest struct {
	IsPrebid bool                  `json:"isPrebid"`
	AppId    string                `json:"appId"`
	ID       string                `json:"id"`
	Imp      []BeachfrontVideoImp  `json:"imp"`
	Site     openrtb.Site          `json:"site"`
	Device   BeachfrontVideoDevice `json:"device"`
	User     openrtb.User          `json:"user"`
	Cur      []string              `json:"cur"`
}

// Soooo close, but not quite openRTB
type BeachfrontVideoImp struct {
	Video    BeachfrontSize `json:"video"`
	Bidfloor float64        `json:"bidfloor"`
	Id       int            `json:"id"`
	ImpId    string         `json:"impid"`
	Secure   int8           `json:"secure"`
}

type BeachfrontVideoDevice struct {
	UA string `json:"ua"`
	IP string `json:"ip"`
	JS string `json:"js"`
}

// ---------------------------------------------------
//              Banner
// ---------------------------------------------------
type BeachfrontBannerRequest struct {
	Slots          []BeachfrontSlot `json:"slots"`
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

type BeachfrontSlot struct {
	Slot     string           `json:"slot"`
	Id       string           `json:"id"`
	Bidfloor float64          `json:"bidfloor"`
	Sizes    []BeachfrontSize `json:"sizes"`
}

type BeachfrontSize struct {
	W uint64 `json:"w"`
	H uint64 `json:"h"`
}

// ---------------------------------------------------
// 				Banner response
// ---------------------------------------------------

type BeachfrontResponseSlot struct {
	CrID  string  `json:"crid"`
	Price float64 `json:"price"`
	W     uint64  `json:"w"`
	H     uint64  `json:"h"`
	Slot  string  `json:"slot"`
	Adm   string  `json:"adm"`
}

func (a *BeachfrontAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var beachfrontRequests BeachfrontRequests
	var videoJSON []byte
	var bannerJSON []byte
	// var audioJSON []byte
	// var nativeJSON []byte
	var errs = make([]error, 0, len(request.Imp))
	var err error

	out, _ := json.Marshal(request)
	glog.Info( fmt.Sprintf("\n -- Original request:\n %s", out) )

	beachfrontRequests, errs , bannerImpCount , videoImpCount , audioImpCount , nativeImpCount := preprocess(request)

	// @todo add err to errs
	videoJSON, err = json.Marshal(beachfrontRequests.Video)
	// audioJSON, err = json.Marshal(beachfrontRequests.Banner)
	// nativeJSON, err = json.Marshal(beachfrontRequests.Banner)

	if videoImpCount + bannerImpCount + audioImpCount + nativeImpCount == 0 {
		errs = append(errs, errors.New("no valid impressions were found"))
		return nil, errs
	}

	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	// ------------------------------------

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	if request.Device != nil {
		addHeaderIfNonEmpty(headers, "User-Agent", request.Device.UA)
		addHeaderIfNonEmpty(headers, "Accept-Language", request.Device.Language)
		if request.Device.DNT != nil {
			addHeaderIfNonEmpty(headers, "DNT", strconv.Itoa(int(*request.Device.DNT)))
		}
	}

	if request.User != nil && request.User.BuyerUID != "" {
		addHeaderIfNonEmpty(headers, "Cookie", "__io_cid="+request.User.BuyerUID)
	}
	reqs := make([]*adapters.RequestData, 0)

	if(videoImpCount > 0) {
		reqs = append(reqs, &adapters.RequestData{
			Method:  "POST",
			Uri:     VideoEndpoint + beachfrontRequests.Video.AppId + VideoEndpointSuffix,
			Body:    videoJSON,
			Headers: headers,

		})

	}

	if(bannerImpCount > 0) {
		bannerJSON, err = json.Marshal(beachfrontRequests.Banner)

		reqs = append(reqs, &adapters.RequestData{
			Method:  "POST",
			Uri:     BannerEndpoint,
			Body:    bannerJSON,
			Headers: headers,
		})
	}

	return reqs, errs
}

func preprocess(request *openrtb.BidRequest)(
	beachfrontReqs BeachfrontRequests,
	errs []error,
	bannerImpCount ,
	videoImpCount ,
	audioImpCount ,
	nativeImpCount int  ) {

	var videoImps = make([]openrtb.Imp,0)
	var bannerImps = make([]openrtb.Imp,0)
	var audioImps = make([]openrtb.Imp,0)
	var nativeImps = make([]openrtb.Imp,0)

	for i := range request.Imp {
		if request.Imp[i].Banner != nil {
			bannerImps = append(bannerImps, request.Imp[i])
			// bannerImps[bannerImpCount].Video = nil
			bannerImpCount++
		}

		if request.Imp[i].Video != nil {
			videoImps = append(videoImps, request.Imp[i])
			// videoImps[videoImpCount].Banner = nil
			videoImpCount++
		}

		if request.Imp[i].Audio != nil {
			audioImps = append(audioImps, request.Imp[i])
			audioImpCount++

			// @TODO -- handle audio
			audioImpCount = 0
		}

		if request.Imp[i].Native != nil {
			nativeImps = append(nativeImps, request.Imp[i])
			nativeImpCount++

			// @TODO -- handle native
			nativeImpCount = 0
		}
	}

	request.Imp = make([] openrtb.Imp, 0)

	if(bannerImpCount > 0) {
		request.Imp = bannerImps
		beachfrontReqs.Banner, errs = getBannerRequests(request)
	}

	if(videoImpCount > 0) {
		beachfrontReqs.Video, errs = getVideoRequest(request, videoImps)
	}

	if(audioImpCount > 0) {
		beachfrontReqs.Audio, errs = getAudioRequest(request, audioImps)
	}

	if(nativeImpCount > 0) {
		beachfrontReqs.Native, errs = getNativeRequest(request, nativeImps)
	}

	return
}

func NewBeachfrontBannerRequest() BeachfrontBannerRequest {
	r := BeachfrontBannerRequest{}
	r.AdapterName = beachfrontAdapterName
	r.AdapterVersion = beachfrontAdapterVersion

	return r
}

func NewBeachfrontVideoRequest() BeachfrontVideoRequest {
	r := BeachfrontVideoRequest{}
	r.Cur = append(r.Cur, "USD")
	r.IsPrebid = true

	return r
}

func NewBeachfrontAudioRequest()  openrtb.Audio {
	r := openrtb.Audio{}
	return r
}

func NewBeachfrontNativeRequest()  openrtb.Native {
	r := openrtb.Native{}
	return r
}

func getBeachfrontExtension(imp openrtb.Imp) (openrtb_ext.ExtImpBeachfront, error) {
	var err error
	var bidderExt adapters.ExtImpBidder
	var beachfrontExt openrtb_ext.ExtImpBeachfront

	if err = json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return beachfrontExt, err
	}

	if err = json.Unmarshal(bidderExt.Bidder, &beachfrontExt); err != nil {
		return beachfrontExt, err
	}

	return beachfrontExt, err
}

func getBannerRequests(request *openrtb.BidRequest) (BeachfrontBannerRequest, []error) {
	var beachfrontReq BeachfrontBannerRequest
	var errs = make([]error, 0, len(request.Imp))

	var slotIndex = 0
	var impIndex = 0

	beachfrontReq = NewBeachfrontBannerRequest()

	for _, imp := range request.Imp {
		beachfrontReq.Slots = append(beachfrontReq.Slots, BeachfrontSlot{})
		slotIndex = len(beachfrontReq.Slots) - 1

		for j := range imp.Banner.Format {
			beachfrontReq.Slots[slotIndex].Sizes = append(beachfrontReq.Slots[slotIndex].Sizes, BeachfrontSize{})
			beachfrontReq.Slots[slotIndex].Sizes[j].H = imp.Banner.Format[j].H
			beachfrontReq.Slots[slotIndex].Sizes[j].W = imp.Banner.Format[j].W
		}

		if request.Device != nil {
			beachfrontReq.IP = request.Device.IP
			beachfrontReq.DeviceModel = request.Device.Model
			beachfrontReq.DeviceOs = request.Device.OS
			if request.Device.DNT != nil {
				beachfrontReq.Dnt = *request.Device.DNT
			}
			if request.Device.UA != "" {
				beachfrontReq.UA = request.Device.UA
			}
		}

		beachfrontExt, err := getBeachfrontExtension(imp)

		if err == nil {
			beachfrontReq.Slots[slotIndex].Bidfloor = beachfrontExt.BidFloor
			beachfrontReq.Slots[slotIndex].Slot = request.Imp[impIndex].ID
		} else {
			errs = append(errs, err)
			continue
		}


		if beachfrontExt.AppId != "" {
			beachfrontReq.Slots[slotIndex].Id = beachfrontExt.AppId
		} else {
			beachfrontReq.Slots[slotIndex].Id = beachfrontExt.AppIds.Banner
		}
		impIndex++
	}

	beachfrontReq.RequestID = request.ID

	if request.Imp[0].Secure != nil {
		beachfrontReq.Secure = *request.Imp[0].Secure
	}

	if request.User != nil {
		beachfrontReq.User.ID = request.User.ID
		beachfrontReq.User.BuyerUID = request.User.BuyerUID
	}

	if request.App != nil {
		beachfrontReq.Domain = request.App.Domain
		beachfrontReq.Page = request.App.ID
	} else {
		protoUrl := strings.Split(request.Site.Page, "//")
		var domainPage string
		// Resolves a panic for any Site.Page that does not include the protocol
		if len(protoUrl) > 1 {
			domainPage = protoUrl[1]
		} else {
			domainPage = protoUrl[0]
		}
		beachfrontReq.Domain = strings.Split(domainPage, "/")[0]
		beachfrontReq.Page = request.Site.Page
	}

	return beachfrontReq, errs
}

func getVideoRequest(request *openrtb.BidRequest, imps []openrtb.Imp) (BeachfrontVideoRequest, []error) {
	var videoImpsIndex = 0
	var beachfrontReq = NewBeachfrontVideoRequest()
	var errs = make([]error, 0, len(request.Imp))
	var appId string
	var impCount = 0

	request.Imp = imps

	if request.App != nil {
		if request.App.Domain != "" {
			beachfrontReq.Site.Domain = request.App.Domain
			beachfrontReq.Site.Page = request.App.ID
		}
	} else {
		if request.Site.Page != "" {
			if request.Site.Domain == "" {
				if strings.Contains(request.Site.Page, "//") {
					// Remove protocol if exists
					beachfrontReq.Site.Domain = strings.Split(request.Site.Page, "//")[1]
				}
				if strings.Contains(beachfrontReq.Site.Domain, "/") {
					// Drop everything after the first "/"
					beachfrontReq.Site.Domain = strings.Split(beachfrontReq.Site.Domain, "/")[0]
				}
			} else {
				beachfrontReq.Site.Domain = request.Site.Domain
			}
			beachfrontReq.Site.Page = request.Site.Page
		}
	}

	for _, imp := range request.Imp {
		beachfrontReq.ID = request.ID

		beachfrontReq.Imp = append(beachfrontReq.Imp, BeachfrontVideoImp{})
		videoImpsIndex = len(beachfrontReq.Imp) - 1

		if imp.Video.H != 0 && imp.Video.W != 0 {
			beachfrontReq.Imp[videoImpsIndex].Video.W = imp.Video.W
			beachfrontReq.Imp[videoImpsIndex].Video.H = imp.Video.H
		} else {
			beachfrontReq.Imp[videoImpsIndex].Video.W = DefaultVideoWidth
			beachfrontReq.Imp[videoImpsIndex].Video.H = DefaultVideoHeight
		}

		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errs = append(errs, err)
			continue
		}

		beachfrontReq.Imp[videoImpsIndex].Id = videoImpsIndex
		beachfrontReq.Imp[videoImpsIndex].ImpId = imp.ID

		beachfrontExt, err := getBeachfrontExtension(imp)


		if beachfrontExt.AppId != "" {
			appId = beachfrontExt.AppId
		} else {
			// errs = append(errs, errors.New("no valid "))
			continue
		}

		if err == nil {
			beachfrontReq.AppId = appId
			beachfrontReq.Imp[videoImpsIndex].Bidfloor = beachfrontExt.BidFloor
		} else {
			errs = append(errs, err)
			continue
		}

		impCount++
	}

	if request.Device != nil {
		beachfrontReq.Device.IP = request.Device.IP
		beachfrontReq.Device.UA = request.Device.UA
		beachfrontReq.Device.JS = "1"
	}

	if request.User != nil {
		if request.User.ID != "" {
			//   Exchange-specific ID for the user. At least one of id or
			//   buyeruid is recommended.
			beachfrontReq.User.ID = request.User.ID
		}

		if request.User.BuyerUID != "" {
			//   Buyer-specific ID for the user as mapped by the exchange for
			//   the buyer. At least one of buyeruid or id is recommended.
			beachfrontReq.User.BuyerUID = request.User.BuyerUID
		}

	}

	return beachfrontReq, errs
}

func getAudioRequest(request *openrtb.BidRequest, imps []openrtb.Imp) (openrtb.Audio, []error) {
	return NewBeachfrontAudioRequest(), nil
}

func getNativeRequest(request *openrtb.BidRequest, imps []openrtb.Imp) (openrtb.Native, []error) {
	return NewBeachfrontNativeRequest()	, nil
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
		return nil, []error{fmt.Errorf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode)}
	}

	bids, errs := postprocess(response, externalRequest, internalRequest.ID)

	if len(errs) != 0 {
		return nil, errs
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(BidCapacity)

	for i := 0; i < len(bids); i++ {
		bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
			Bid:     &bids[i],
		})
	}

	return bidResponse, errs
}

func postprocess(response *adapters.ResponseData, externalRequest *adapters.RequestData, id string) ([]openrtb.Bid, []error) {
	var beachfrontResp []BeachfrontResponseSlot
	var errs = make([]error, 0)
	// var list = json.Unmarshal()


	glog.Info( fmt.Sprintf("\n -- Response:\n %s", response.Body) )

	// for i := o; i < len(response.Body)

	// return nil, nil

	// if bidtype == openrtb_ext.BidTypeVideo {
	if false {
		var openrtbResp openrtb.BidResponse
		if err := json.Unmarshal(response.Body, &openrtbResp); err != nil {
			errs = append(errs, err)
			return nil, errs
		}
		return postprocessVideo(openrtbResp.SeatBid[0].Bid, externalRequest, id)
	} else {
		if err := json.Unmarshal(response.Body, &beachfrontResp); err != nil {
			errs = append(errs, err)
			return nil, errs
		}

		return postprocessBanner(beachfrontResp, externalRequest, id)
	}
}

func postprocessBanner(beachfrontResp []BeachfrontResponseSlot, externalRequest *adapters.RequestData, id string) ([]openrtb.Bid, []error) {
	var xtrnal BeachfrontBannerRequest
	var bids = make([]openrtb.Bid, len(beachfrontResp))
	var errs = make([]error, 0)

	if err := json.Unmarshal(externalRequest.Body, &xtrnal); err != nil {
		errs = append(errs, err)
		return bids, errs
	}


	for i := range beachfrontResp {
		bids[i] = openrtb.Bid{
			CrID:  beachfrontResp[i].CrID,
			ImpID: xtrnal.RequestID,
			Price: beachfrontResp[i].Price,
			ID:    id,
			AdM:   beachfrontResp[i].Adm,
			H:     beachfrontResp[i].H,
			W:     beachfrontResp[i].W
		}
	}

	// Am not adding any errors
	return bids, errs
}

func postprocessVideo(bids []openrtb.Bid, externalRequest *adapters.RequestData, id string) ([]openrtb.Bid, []error) {
	var xtrnal BeachfrontVideoRequest
	var errs = make([]error, 0)

	if err := json.Unmarshal(externalRequest.Body, &xtrnal); err != nil {
		errs = append(errs, err)
		return bids, errs
	}

	for i := range bids {
		crid := extractVideoCrid(bids[i].NURL)

		bids[i].CrID = crid
		bids[i].ImpID = xtrnal.Imp[0].ImpId
		bids[i].H = xtrnal.Imp[0].Video.H
		bids[i].W = xtrnal.Imp[0].Video.W
		bids[i].ID = id
	}

	return bids, errs
}

func extractBannerCrid(adm string) string {
	chunky := strings.SplitAfter(adm, "\"")
	return strings.TrimSuffix(chunky[1], "\"")
}

func getBidType(internal *openrtb.BidRequest) openrtb_ext.BidType {
	var uri = VideoEndpoint
	if uri == VideoEndpoint {
		return openrtb_ext.BidTypeVideo
	}

	return openrtb_ext.BidTypeBanner
}

func extractVideoCrid(nurl string) string {
	chunky := strings.SplitAfter(nurl, ":")
	return strings.TrimSuffix(chunky[2], ":")
}

// Thank you, brightroll.
func addHeaderIfNonEmpty(headers http.Header, headerName string, headerValue string) {
	if len(headerValue) > 0 {
		headers.Add(headerName, headerValue)
	}
}

func NewBeachfrontBidder() *BeachfrontAdapter {
	return &BeachfrontAdapter{}
}
