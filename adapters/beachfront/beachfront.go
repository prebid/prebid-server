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

	beachfrontRequests, errs = preprocess(request)

	if len(errs) > 0 {
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

	if len(beachfrontRequests.Video) > 0 {
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
	}

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

	return reqs, errs
}

func preprocess(request *openrtb.BidRequest) (beachfrontReqs beachfrontRequests, errs []error) {

	var videoImps = make([]openrtb.Imp, 0)
	var bannerImps = make([]openrtb.Imp, 0)

	var weGotNothing bool = true

	for i := 0; i < len(request.Imp); i++ {
		if request.Imp[i].Banner != nil {
			weGotNothing = false
			bannerImps = append(bannerImps, request.Imp[i])
		}

		if request.Imp[i].Video != nil {
			weGotNothing = false
			videoImps = append(videoImps, request.Imp[i])
		}
	}

	if weGotNothing {
		errs = append(errs, errors.New("no valid impressions were found in the request"))
		return
	}

	request.Imp = make([]openrtb.Imp, 0)

	if len(bannerImps) > 0 {
		request.Imp = bannerImps
		beachfrontReqs.Banner, errs = getBannerRequest(request)
	}

	if len(videoImps) > 0 {
		request.Imp = videoImps
		beachfrontReqs.Video, errs = getVideoRequests(request)
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
	r.Cur = append(r.Cur, "USD")
	r.IsPrebid = true

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

/*
getBannerRequest, singular. A "Slot" is an "imp," and each Slot can have an AppId, so just one
request to the beachfront banner endpoint gets all banner Imps.
*/
func getBannerRequest(request *openrtb.BidRequest) (beachfrontBannerRequest, []error) {
	var beachfrontReq beachfrontBannerRequest
	var errs = make([]error, 0, len(request.Imp))

	var slotIndex = 0
	var impIndex = 0

	beachfrontReq = newBeachfrontBannerRequest()

	for i := 0; i < len(request.Imp); i++ {
		beachfrontReq.Slots = append(beachfrontReq.Slots, beachfrontSlot{})
		slotIndex = len(beachfrontReq.Slots) - 1

		for j := 0; j < len(request.Imp[i].Banner.Format); j++ {
			beachfrontReq.Slots[slotIndex].Sizes = append(beachfrontReq.Slots[slotIndex].Sizes, beachfrontSize{})
			beachfrontReq.Slots[slotIndex].Sizes[j].H = request.Imp[i].Banner.Format[j].H
			beachfrontReq.Slots[slotIndex].Sizes[j].W = request.Imp[i].Banner.Format[j].W
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

		beachfrontExt, err := getBeachfrontExtension(request.Imp[i])

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
		beachfrontReq.IsMobile = 1
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
		beachfrontReq.IsMobile = 0
	}

	return beachfrontReq, errs
}

/*
getVideoRequests, plural. One request to the endpoint can have one appId, and can return one nurl,
so each video imp is a call to the endpoint.
*/
func getVideoRequests(request *openrtb.BidRequest) ([]beachfrontVideoRequest, []error) {
	var videoImpsIndex = 0
	var beachfrontReqs = make([]beachfrontVideoRequest, 0)
	var errs = make([]error, 0, len(request.Imp))
	var appId string
	var impCount = 0

	for i := 0; i < len(request.Imp); i++ {
		r := newBeachfrontVideoRequest()

		if request.App != nil {
			if request.App.Domain != "" {
				r.Site.Domain = request.App.Domain
				r.Site.Page = request.App.ID
				r.Site.Mobile = 1
			}
		} else {
			if request.Site.Page != "" {
				if request.Site.Domain == "" {
					if strings.Contains(request.Site.Page, "//") {
						// Remove protocol if exists
						r.Site.Domain = strings.Split(request.Site.Page, "//")[1]
					}
					if strings.Contains(r.Site.Domain, "/") {
						// Drop everything after the first "/"
						r.Site.Domain = strings.Split(r.Site.Domain, "/")[0]
					}
				} else {
					r.Site.Domain = request.Site.Domain
				}
				r.Site.Page = request.Site.Page
			}
		}

		r.Imp = append(r.Imp, beachfrontVideoImp{})
		videoImpsIndex = len(r.Imp) - 1

		if request.Imp[i].Video.H != 0 && request.Imp[i].Video.W != 0 {
			r.Imp[videoImpsIndex].Video.W = request.Imp[i].Video.W
			r.Imp[videoImpsIndex].Video.H = request.Imp[i].Video.H
		} else {
			r.Imp[videoImpsIndex].Video.W = DefaultVideoWidth
			r.Imp[videoImpsIndex].Video.H = DefaultVideoHeight
		}

		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(request.Imp[i].Ext, &bidderExt); err != nil {
			errs = append(errs, err)
			continue
		}

		r.Imp[videoImpsIndex].Id = videoImpsIndex
		r.Imp[videoImpsIndex].ImpId = request.Imp[i].ID

		beachfrontExt, err := getBeachfrontExtension(request.Imp[i])

		if beachfrontExt.AppId != "" {
			appId = beachfrontExt.AppId
		} else {
			appId = beachfrontExt.AppIds.Video
		}

		if err == nil {
			r.AppId = appId
			r.Imp[videoImpsIndex].Bidfloor = beachfrontExt.BidFloor
		} else {
			errs = append(errs, err)
			continue
		}

		if request.Device != nil {
			r.Device.IP = request.Device.IP
			r.Device.UA = request.Device.UA
			r.Device.JS = "1"
		}

		if request.User != nil {
			if request.User.ID != "" {
				//   Exchange-specific ID for the user. At least one of id or
				//   buyeruid is recommended.
				r.User.ID = request.User.ID
			}

			if request.User.BuyerUID != "" {
				//   Buyer-specific ID for the user as mapped by the exchange for
				//   the buyer. At least one of buyeruid or id is recommended.
				r.User.BuyerUID = request.User.BuyerUID
			}

		}

		r.ID = request.ID

		impCount++
		beachfrontReqs = append(beachfrontReqs, r)
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
			BidType: getBidType(bids[i]),
		})
	}

	return bidResponse, errs
}

func postprocess(response *adapters.ResponseData, externalRequest *adapters.RequestData, id string) ([]openrtb.Bid, []error) {
	var beachfrontResp []beachfrontResponseSlot
	var errs = make([]error, 0)
	// var list = json.Unmarshal()

	var openrtbResp openrtb.BidResponse

	// try it as a video
	if err := json.Unmarshal(response.Body, &openrtbResp); err != nil {

		// try it as a banner
		if err := json.Unmarshal(response.Body, &beachfrontResp); err != nil {
			// it's neither. I'm not appending these errors
			// errs = append(errs, err)
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
		bids[i].ImpID = xtrnal.Imp[0].ImpId
		bids[i].H = xtrnal.Imp[0].Video.H
		bids[i].W = xtrnal.Imp[0].Video.W
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

func addHeaderIfNonEmpty(headers http.Header, headerName string, headerValue string) {
	if len(headerValue) > 0 {
		headers.Add(headerName, headerValue)
	}
}

func NewBeachfrontBidder() *BeachfrontAdapter {
	return &BeachfrontAdapter{}
}
