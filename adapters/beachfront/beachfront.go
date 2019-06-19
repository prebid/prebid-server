package beachfront

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/prebid/prebid-server/errortypes"
	"net/http"
	"strconv"
	"strings"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/mxmCherry/openrtb"
)

const Seat = "beachfront"
const BidCapacity = 5

const BannerEndpoint = "https://display.bfmio.com/prebid_display"
const VideoEndpoint = "https://reachms.bfmio.com/bid.json?exchange_id="

const VideoEndpointSuffix = "&prebidserver"

const beachfrontAdapterName = "BF_PREBID_S2S"
const beachfrontAdapterVersion = "0.3.0"

type BeachfrontAdapter struct {
}

type BeachfrontRequests struct {
	Banner BeachfrontBannerRequest
	Video  BeachfrontVideoRequest
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

func getEndpoint(request *openrtb.BidRequest) (uri string) {
	for i := range request.Imp {
		if request.Imp[i].Video != nil {
			// If there are any video imps, we will be running a video auction
			// and dropping all of the banner actions.
			return VideoEndpoint
		}
	}
	return BannerEndpoint
}

func (a *BeachfrontAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var beachfrontRequests BeachfrontRequests
	var reqJSON []byte
	var uri string
	var errs = make([]error, 0, len(request.Imp))
	var err error
	var imps int

	uri = getEndpoint(request)

	beachfrontRequests, errs, imps = preprocess(request, uri)

	if uri == VideoEndpoint {
		reqJSON, err = json.Marshal(beachfrontRequests.Video)
		uri = uri + beachfrontRequests.Video.AppId + VideoEndpointSuffix
	} else {
		/*
			We will get here if request contains no Video imps, though it might have
			Audio or Native imps as well as banner.
		*/
		reqJSON, err = json.Marshal(beachfrontRequests.Banner)
	}

	if imps == 0 {
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
		addHeaderIfNonEmpty(headers, "X-Forwarded-For", request.Device.IP)
		addHeaderIfNonEmpty(headers, "Accept-Language", request.Device.Language)
		if request.Device.DNT != nil {
			addHeaderIfNonEmpty(headers, "DNT", strconv.Itoa(int(*request.Device.DNT)))
		}
	}

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     uri,
		Body:    reqJSON,
		Headers: headers,
	}}, errs
}

/*
We have received a prebid request. It needs to be converted to a beachfront request. This is complicated
by the fact that we have different servers for video/display and they have different contracts.
*/
func preprocess(req *openrtb.BidRequest, uri string) (BeachfrontRequests, []error, int) {
	var beachfrontReqs BeachfrontRequests
	var errs = make([]error, 0, len(req.Imp))
	var imps int

	if uri == BannerEndpoint {
		beachfrontReqs.Banner, errs, imps = getBannerRequest(req)
	} else {
		// If there were any Video imps in the request, we have skipped to here.
		beachfrontReqs.Video, errs, imps = getVideoRequest(req)
	}

	return beachfrontReqs, errs, imps
}

func getBannerRequest(req *openrtb.BidRequest) (BeachfrontBannerRequest, []error, int) {
	var bannerImpsIndex = 0
	var beachfrontReq = NewBeachfrontBannerRequest()
	var errs = make([]error, 0, len(req.Imp))
	var imps = 0

	// step through the prebid request "imp" and inject into the beachfront request.
	for _, imp := range req.Imp {
		if imp.Banner != nil {
			beachfrontReq.Slots = append(beachfrontReq.Slots, BeachfrontSlot{})
			bannerImpsIndex = len(beachfrontReq.Slots) - 1

			beachfrontReq.Slots[bannerImpsIndex].Sizes = append(beachfrontReq.Slots[bannerImpsIndex].Sizes, BeachfrontSize{})
			for j := range imp.Banner.Format {
				if j > 0 {
					beachfrontReq.Slots[bannerImpsIndex].Sizes = append(beachfrontReq.Slots[bannerImpsIndex].Sizes, BeachfrontSize{})
				}
				beachfrontReq.Slots[bannerImpsIndex].Sizes[j].H = imp.Banner.Format[j].H
				beachfrontReq.Slots[bannerImpsIndex].Sizes[j].W = imp.Banner.Format[j].W
			}

			var bidderExt adapters.ExtImpBidder
			if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
				// possible banner error 2
				errs = append(errs, err)
				continue
			}

			var beachfrontExt openrtb_ext.ExtImpBeachfront
			if err := json.Unmarshal(bidderExt.Bidder, &beachfrontExt); err != nil {
				// possible banner error 3 - supplemental/unmarshal-error-banner.json
				errs = append(errs, err)
				continue
			}

			if req.Device != nil {
				beachfrontReq.IP = req.Device.IP
				beachfrontReq.DeviceModel = req.Device.Model
				beachfrontReq.DeviceOs = req.Device.OS
				if req.Device.DNT != nil {
					beachfrontReq.Dnt = *req.Device.DNT
				}
				if req.Device.UA != "" {
					beachfrontReq.UA = req.Device.UA
				}
			}

			beachfrontReq.Slots[bannerImpsIndex].Bidfloor = beachfrontExt.BidFloor
			beachfrontReq.Slots[bannerImpsIndex].Slot = req.Imp[bannerImpsIndex].ID
			beachfrontReq.Slots[bannerImpsIndex].Id = beachfrontExt.AppId
		}

		imps++
	}

	// Just take the last one... I guess?
	if req.Imp[bannerImpsIndex].Secure != nil {
		beachfrontReq.Secure = *req.Imp[bannerImpsIndex].Secure
	}

	if req.User != nil {
		beachfrontReq.User.ID = req.User.ID
		beachfrontReq.User.BuyerUID = req.User.BuyerUID
	}

	if req.App != nil {
		beachfrontReq.Domain = req.App.Domain
		beachfrontReq.Page = req.App.ID
	} else {
		protoUrl := strings.Split(req.Site.Page, "//")
		var domainPage string
		// Resolves a panic for any Site.Page that does not include the protocol
		if len(protoUrl) > 1 {
			domainPage = protoUrl[1]
		} else {
			domainPage = protoUrl[0]
		}
		beachfrontReq.Domain = strings.Split(domainPage, "/")[0]
		beachfrontReq.Page = req.Site.Page
	}

	beachfrontReq.RequestID = req.ID

	return beachfrontReq, errs, imps
}

func getVideoRequest(req *openrtb.BidRequest) (BeachfrontVideoRequest, []error, int) {
	var videoImpsIndex = 0
	var beachfrontReq = NewBeachfrontVideoRequest()
	var errs = make([]error, 0, len(req.Imp))
	var imps = 0

	if req.App != nil {
		if req.App.Domain != "" {
			beachfrontReq.Site.Domain = req.App.Domain
			beachfrontReq.Site.Page = req.App.ID
		}
	} else {
		if req.Site.Page != "" {
			if req.Site.Domain == "" {
				if strings.Contains(req.Site.Page, "//") {
					// Remove protocol if exists
					beachfrontReq.Site.Domain = strings.Split(req.Site.Page, "//")[1]
				}
				if strings.Contains(beachfrontReq.Site.Domain, "/") {
					// Drop everything after the first "/"
					beachfrontReq.Site.Domain = strings.Split(beachfrontReq.Site.Domain, "/")[0]
				}
			} else {
				beachfrontReq.Site.Domain = req.Site.Domain
			}
			beachfrontReq.Site.Page = req.Site.Page
		}
	}

	/*
		The req could contain banner,audio,native and video imps when It arrives here. I am only
		interested in video

		The beachfront video endpoint is only capable of returning a single nurl and price, wrapped in
		an openrtb format, so even though I'm building a request here that will include multiple video
		impressions, only a single URL will be returned. Hopefully the beachfront endpoint can be modified
		in the future to return multiple video ads

	*/
	for _, imp := range req.Imp {
		if imp.Video != nil {
			beachfrontReq.ID = req.ID

			beachfrontReq.Imp = append(beachfrontReq.Imp, BeachfrontVideoImp{})
			videoImpsIndex = len(beachfrontReq.Imp) - 1

			beachfrontReq.Imp[videoImpsIndex].Video.H = imp.Video.H
			beachfrontReq.Imp[videoImpsIndex].Video.W = imp.Video.W

			var bidderExt adapters.ExtImpBidder
			if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
				errs = append(errs, err)
				continue
			}

			var beachfrontVideoExt openrtb_ext.ExtImpBeachfront
			if err := json.Unmarshal(bidderExt.Bidder, &beachfrontVideoExt); err != nil {
				errs = append(errs, err)
				continue
			}

			beachfrontReq.Imp[videoImpsIndex].Bidfloor = beachfrontVideoExt.BidFloor
			if imp.Secure != nil {
				beachfrontReq.Imp[videoImpsIndex].Secure = *imp.Secure
			} else {
				beachfrontReq.Imp[videoImpsIndex].Secure = 0
			}

			beachfrontReq.Imp[videoImpsIndex].Id = videoImpsIndex
			beachfrontReq.Imp[videoImpsIndex].ImpId = imp.ID

			if req.Device != nil {
				beachfrontReq.Device.IP = req.Device.IP
				beachfrontReq.Device.UA = req.Device.UA
				beachfrontReq.Device.JS = "1"
			}

			beachfrontReq.AppId = beachfrontVideoExt.AppId
			imps++
		}
	}

	if req.User != nil {
		if req.User.ID != "" {
			//   Exchange-specific ID for the user. At least one of id or
			//   buyeruid is recommended.
			beachfrontReq.User.ID = req.User.ID
		}

		if req.User.BuyerUID != "" {
			//   Buyer-specific ID for the user as mapped by the exchange for
			//   the buyer. At least one of buyeruid or id is recommended.
			beachfrontReq.User.BuyerUID = req.User.BuyerUID
		}

	}

	return beachfrontReq, errs, imps
}

func (a *BeachfrontAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var bids []openrtb.Bid
	var bidtype = getBidType(internalRequest)

	/*
		Beachfront is now sending an empty array and 200 as their "no results" response. This should catch that.
	*/

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

	bids, errs := postprocess(response, externalRequest, internalRequest.ID, bidtype)

	if len(errs) != 0 {
		return nil, errs
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(BidCapacity)

	for i := 0; i < len(bids); i++ {
		bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
			Bid:     &bids[i],
			BidType: bidtype,
		})
	}

	return bidResponse, errs
}

func postprocess(response *adapters.ResponseData, externalRequest *adapters.RequestData, id string, bidtype openrtb_ext.BidType) ([]openrtb.Bid, []error) {
	var beachfrontResp []BeachfrontResponseSlot
	var errs = make([]error, 0)

	if bidtype == openrtb_ext.BidTypeVideo {
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

		return postprocessBanner(beachfrontResp, id)
	}
}

func postprocessBanner(beachfrontResp []BeachfrontResponseSlot, id string) ([]openrtb.Bid, []error) {
	var bids = make([]openrtb.Bid, len(beachfrontResp))
	var errs = make([]error, 0)

	for i := range beachfrontResp {
		crid := extractBannerCrid(beachfrontResp[i].Adm)

		bids[i] = openrtb.Bid{
			CrID:  crid,
			ImpID: beachfrontResp[i].Slot,
			Price: beachfrontResp[i].Price,
			ID:    id,
			AdM:   beachfrontResp[i].Adm,
			H:     beachfrontResp[i].H,
			W:     beachfrontResp[i].W,
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
	for i := range internal.Imp {
		if internal.Imp[i].Video != nil {
			return openrtb_ext.BidTypeVideo
		}
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
