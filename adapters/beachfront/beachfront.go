package beachfront

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/mxmCherry/openrtb"
)

const Seat = "beachfront"
const BidCapacity = 5

const BannerEndpoint = "https://display.bfmio.com/prebid_display"
const VideoEndpoint = "https://reachms.bfmio.com/bid.json?exchange_id="
const VideoEndpointSuffix = "&prebidserver"

const beachfrontAdapterName = "BF_PREBID_S2S"
const beachfrontAdapterVersion = "0.1.1"

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
	Domain   string                `json:"domain"`
	Id       string                `json:"id"`
	Imp      []BeachfrontVideoImp  `json:"imp"`
	Site     BeachfrontSite        `json:"site"`
	Device   BeachfrontVideoDevice `json:"device"`
	User     openrtb.User          `json:"user"`
	Cur      []string              `json:"cur"`
}

type BeachfrontSite struct {
	Page string `json:"page"`
}

type BeachfrontPublisher struct {
	Id string `json:"id"`
}

type BeachfrontVideoDevice struct {
	Ua         string             `json:"ua"`
	Devicetype int                `json:"deviceType"`
	Geo        BeachfrontVideoGeo `json:"geo"`
}

type BeachfrontVideoGeo struct {
	Ip string `json:"ip"`
}

type BeachfrontVideoImp struct {
	Video    BeachfrontSize `json:"video"`
	Bidfloor float64        `json:"bidfloor"`
	Id       int            `json:"id"`
	ImpId    string         `json:"impid"`
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
	Ua             string           `json:"ua"`
	Dnt            int8             `json:"dnt"`
	User           string           `json:"user"`
	AdapterName    string           `json:"adapterName"`
	AdapterVersion string           `json:"adapterVersion"`
	Ip             string           `json:"ip"`
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

func (a *BeachfrontAdapter) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
	var beachfrontRequests BeachfrontRequests
	var reqJSON []byte
	var uri string
	var errs = make([]error, 0)
	var err error
	var imps int

	uri = getEndpoint(request)

	beachfrontRequests, errs, imps = preprocess(request, uri)

	// These are fatal errors -------------
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
		errs = append(errs, errors.New("No valid impressions were found"))
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

	if uri == BannerEndpoint {
		if request.User != nil {
			headers.Add("Cookie", "UserID="+request.User.ID+"; __io_cid="+request.User.BuyerUID)
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
	var bannerImpsIndex int = 0
	var beachfrontReq BeachfrontBannerRequest = NewBeachfrontBannerRequest()
	var errs = make([]error, 0, len(req.Imp))
	var imps int = 0

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

			beachfrontReq.Slots[bannerImpsIndex].Bidfloor = imp.BidFloor

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
				beachfrontReq.Ip = req.Device.IP
				beachfrontReq.DeviceModel = req.Device.Model
				beachfrontReq.DeviceOs = req.Device.OS
				beachfrontReq.Dnt = req.Device.DNT
				if req.Device.UA != "" {
					beachfrontReq.Ua = req.Device.UA
				}
			}

			beachfrontReq.Slots[bannerImpsIndex].Slot = req.Imp[bannerImpsIndex].ID
			beachfrontReq.Slots[bannerImpsIndex].Id = beachfrontExt.AppId
		}

		imps++
	}

	if req.User != nil {
		beachfrontReq.User = req.User.BuyerUID
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

	return beachfrontReq, errs, imps
}

/*
Prepare the request that has been received from Prebid.js, translating it to the beachfront format
*/
func getVideoRequest(req *openrtb.BidRequest) (BeachfrontVideoRequest, []error, int) {
	var videoImpsIndex int = 0
	var beachfrontReq BeachfrontVideoRequest = NewBeachfrontVideoRequest()
	var errs = make([]error, 0, len(req.Imp))
	var imps int = 0

	/*
		The req could contain banner,audio,native and video imps when It arrives here. I am only
		interested in video

		The beach front video endpoint is only capable of returning a single nurl and price, wrapped in
		an openrtb format, so even though I'm building a request here that will include multiple video
		impressions, only a single URL will be returned. Hopefully the beachfront endpoint can be modified
		in the future to return multiple video ads

	*/
	for _, imp := range req.Imp {
		if imp.Video != nil {
			beachfrontReq.Id = req.ID

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
			beachfrontReq.Imp[videoImpsIndex].Id = videoImpsIndex
			beachfrontReq.Imp[videoImpsIndex].ImpId = imp.ID

			if req.Device != nil {
				beachfrontReq.Device.Geo.Ip = req.Device.IP
				beachfrontReq.Device.Ua = req.Device.UA
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

	if req.App != nil {
		if req.App.Domain != "" {
			beachfrontReq.Domain = req.App.Domain
			beachfrontReq.Site.Page = req.App.ID
		}
	} else {
		if req.Site.Page != "" {
			if req.Site.Domain == "" {
				if strings.Contains(req.Site.Page, "//") {
					// Remove protocol if exists
					beachfrontReq.Domain = strings.Split(req.Site.Page, "//")[1]
				}
				if strings.Contains(beachfrontReq.Domain, "/") {
					// Drop everything after the first "/"
					beachfrontReq.Domain = strings.Split(beachfrontReq.Domain, "/")[0]
				}
			} else {
				beachfrontReq.Domain = req.Site.Domain
			}
			beachfrontReq.Site.Page = req.Site.Page
		}
	}

	return beachfrontReq, errs, imps
}

func (a *BeachfrontAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var bids []openrtb.Bid
	var bidtype openrtb_ext.BidType = getBidType(internalRequest)
	var errors = make([]error, 0)

	bids, errs := postprocess(response, externalRequest, internalRequest.ID, bidtype)

	if len(errs) != 0 {
		errors = append(errors, errs...)
		err := &errortypes.BadServerResponse{
			Message: "Failed to process the beachfront response",
		}

		errors = append(errors, err)
		return nil, errors
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(BidCapacity)

	for i := 0; i < len(bids); i++ {
		bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
			Bid:     &bids[i],
			BidType: bidtype,
		})
	}

	return bidResponse, errors
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
	var bids []openrtb.Bid = make([]openrtb.Bid, len(beachfrontResp))
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

func NewBeachfrontBidder() *BeachfrontAdapter {
	return &BeachfrontAdapter{}
}
