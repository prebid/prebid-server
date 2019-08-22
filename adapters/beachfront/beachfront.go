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

type beachfrontRequests struct {
	Banner beachfrontBannerRequest
	Video  []beachfrontVideoRequest
	Audio  openrtb.Audio
	Native openrtb.Native
}

// ---------------------------------------------------
//              Video
// ---------------------------------------------------

type beachfrontVideoRequest struct {
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
	var reqs = make([]*adapters.RequestData, len(beachfrontRequests.Video))

	beachfrontRequests, errs = preprocess(request)

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

	// At most, I only ever have one banner request, and it does not need the cookie, so doing it first.
	if len(beachfrontRequests.Banner.Slots) > 0 {
		bytes, err := json.Marshal(beachfrontRequests.Banner)

		if err == nil {
			reqs = append(reqs, &adapters.RequestData{
				Method:  "POST",
				Uri:     BannerEndpoint,
				Body:    bytes,
				Headers: headers,
			})
		}
	}

	if request.User != nil && request.User.BuyerUID != "" {
		headers.Add("Cookie", "__io_cid="+request.User.BuyerUID)
	}

	for i := 0; i < len(beachfrontRequests.Video); i++ {

		bytes, err := json.Marshal(beachfrontRequests.Video[i])

		if err == nil {
			reqs = append(reqs, &adapters.RequestData{
				Method:  "POST",
				Uri:     VideoEndpoint + beachfrontRequests.Video[i].AppId + VideoEndpointSuffix,
				Body:    bytes,
				Headers: headers,
			})
		} else {
			continue
		}
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

		if len(errs) > 0 {
			// getBannerRequest can only generate one error
			bannerErr := errs[0]

			beachfrontReqs.Video, errs = getVideoRequests(request)
			errs = append(errs, bannerErr)
		} else {
			beachfrontReqs.Video, errs = getVideoRequests(request)
		}

	}

	return
}

func newBeachfrontBannerRequest() beachfrontBannerRequest {
	r := beachfrontBannerRequest{}
	r.AdapterName = beachfrontAdapterName
	r.AdapterVersion = beachfrontAdapterVersion

	return r
}

func newBeachfrontVideoRequest() beachfrontVideoRequest {
	r := beachfrontVideoRequest{}
	r.IsPrebid = true

	r.Cur = append(r.Cur, "USD")
	return r
}

func getBeachfrontExtension(imp openrtb.Imp) (openrtb_ext.ExtImpBeachfront, error) {
	var err error
	var bidderExt adapters.ExtImpBidder
	var beachfrontExt openrtb_ext.ExtImpBeachfront

	if err = json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return beachfrontExt , &errortypes.BadInput{
			Message: fmt.Sprintf("ignoring imp id=%s, error while decoding extImpBidder, err: %s", imp.ID, err),
		}
	}

	if err = json.Unmarshal(bidderExt.Bidder, &beachfrontExt); err != nil {
		return beachfrontExt , &errortypes.BadInput{
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
		if request.App.Domain != "" {
			site.Domain = request.App.Domain
		}
		if request.App.Domain != "" {
			site.Page = request.App.ID
		}

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

/*
getBannerRequest, singular. A "Slot" is an "imp," and each Slot can have an AppId, so just one
request to the beachfront banner endpoint gets all banner Imps.
*/
func getBannerRequest(request *openrtb.BidRequest) (beachfrontBannerRequest, []error) {
	var bfBannerRequest beachfrontBannerRequest
	var errs = make([]error, 0, len(request.Imp))

	bfBannerRequest = newBeachfrontBannerRequest()

	// The request that gets to here only has imps that contain a banner element. They may also contain
	// a video element, but those are being ignored in this function.
	for i := 0; i < len(request.Imp); i++ {

		// 1. Get the extension. Doing this first because if this fails, no need for the rest.
		beachfrontExt, err := getBeachfrontExtension(request.Imp[i])
		slot := beachfrontSlot{}

		if err == nil {

			slot.Bidfloor = beachfrontExt.BidFloor
			slot.Slot = request.Imp[i].ID

			if beachfrontExt.AppId != "" {
				slot.Id = beachfrontExt.AppId
			} else {
				slot.Id = beachfrontExt.AppIds.Banner
			}

		} else {
			errs = append(errs, err)
			continue
		}

		// 2. sizes
		for j := 0; j < len(request.Imp[i].Banner.Format); j++ {

			slot.Sizes = append(slot.Sizes, beachfrontSize{
				H: request.Imp[i].Banner.Format[j].H,
				W: request.Imp[i].Banner.Format[j].W,
			})
		}

		bfBannerRequest.Slots = append(bfBannerRequest.Slots, slot)
	}

	// 3. Device
	if request.Device != nil {
		bfBannerRequest.IP = request.Device.IP
		bfBannerRequest.DeviceModel = request.Device.Model
		bfBannerRequest.DeviceOs = request.Device.OS
		if request.Device.DNT != nil {
			bfBannerRequest.Dnt = *request.Device.DNT
		}
		if request.Device.UA != "" {
			bfBannerRequest.UA = request.Device.UA
		}
	}

	// 4. Domain / Page / Mobile
	site := getSite(request)
	bfBannerRequest.IsMobile = site.Mobile
	bfBannerRequest.Page = site.Page
	bfBannerRequest.Domain = site.Domain

	// 5. User
	if request.User != nil {
		if bfBannerRequest.User.ID == "" {
			bfBannerRequest.User.ID = request.User.ID
		}

		if bfBannerRequest.User.BuyerUID == "" {
			bfBannerRequest.User.BuyerUID = request.User.BuyerUID
		}
	}

	// 6. request ID
	bfBannerRequest.RequestID = request.ID

	// 7. unique to banner
	if request.Imp[0].Secure != nil {
		bfBannerRequest.Secure = *request.Imp[0].Secure
	}

	return bfBannerRequest, errs
}

/*
getVideoRequests, plural. One request to the endpoint can have one appId, and can return one nurl,
so each video imp is a call to the endpoint.
*/
func getVideoRequests(request *openrtb.BidRequest) ([]beachfrontVideoRequest, []error) {
	var beachfrontReqs = make([]beachfrontVideoRequest, 0)
	var errs = make([]error, 0, len(request.Imp))

	for i := 0; i < len(request.Imp); i++ {
		bfVideoRequest := newBeachfrontVideoRequest()
		bfVideoRequest.Imp = append(bfVideoRequest.Imp, beachfrontVideoImp{})

		// 1. Extension
		beachfrontExt, err := getBeachfrontExtension(request.Imp[i])

		if err == nil {
			bfVideoRequest.Imp[0].Bidfloor = beachfrontExt.BidFloor

			if beachfrontExt.AppId != "" {
				bfVideoRequest.AppId = beachfrontExt.AppId
			} else {
				bfVideoRequest.AppId = beachfrontExt.AppIds.Video
			}
		} else {
			// Failed to extract the beachfrontExt, so this request is junk.
			errs = append(errs, err)
			continue
		}

		// 2. sizes
		if request.Imp[i].Video.H != 0 && request.Imp[i].Video.W != 0 {
			bfVideoRequest.Imp[0].Video.W = request.Imp[i].Video.W
			bfVideoRequest.Imp[0].Video.H = request.Imp[i].Video.H
		} else {
			bfVideoRequest.Imp[0].Video.W = DefaultVideoWidth
			bfVideoRequest.Imp[0].Video.H = DefaultVideoHeight
		}

		// 3. Device
		if request.Device != nil {
			bfVideoRequest.Device.IP = request.Device.IP
			bfVideoRequest.Device.UA = request.Device.UA
			bfVideoRequest.Device.JS = "1"
		}

		// 4. Domain / Page / Mobile
		bfVideoRequest.Site = getSite(request)

		// 5. User
		if request.User != nil {
			if request.User.ID != "" {
				//   Exchange-specific ID for the user. At least one of id or
				//   buyeruid is recommended.
				bfVideoRequest.User.ID = request.User.ID
			}

			if request.User.BuyerUID != "" {
				//   Buyer-specific ID for the user as mapped by the exchange for
				//   the buyer. At least one of buyeruid or id is recommended.
				bfVideoRequest.User.BuyerUID = request.User.BuyerUID
			}

		}

		// 6. request ID
		bfVideoRequest.ID = request.ID

		// 7. Unique to video
		bfVideoRequest.Imp[0].Id = 0
		bfVideoRequest.Imp[0].ImpId = request.Imp[i].ID

		beachfrontReqs = append(beachfrontReqs, bfVideoRequest)
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
			BidType: getBidType(bids[i]),
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

	var xtrnal beachfrontVideoRequest
	var errs = make([]error, 0)

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

	return bids, errs
}

func getBidType(bid openrtb.Bid) openrtb_ext.BidType {
	if bid.AdM != "" {
		return openrtb_ext.BidTypeBanner
	} else if bid.NURL != "" {
		return openrtb_ext.BidTypeVideo
	}

	return ""
}

func extractVideoCrid(nurl string) string {
	chunky := strings.SplitAfter(nurl, ":")
	return strings.TrimSuffix(chunky[2], ":")
}

func removeSlot(s []beachfrontSlot, i int) []beachfrontSlot {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}

func addHeaderIfNonEmpty(headers http.Header, headerName string, headerValue string) {
	if len(headerValue) > 0 {
		headers.Add(headerName, headerValue)
	}
}

func NewBeachfrontBidder() *BeachfrontAdapter {
	return &BeachfrontAdapter{}
}
