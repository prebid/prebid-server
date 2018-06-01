package beachfront

import (
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
	"regexp"
	"strings"
)

const Seat = "beachfront"
const BidCapacity = 5

const BannerEndpoint = "https://display.bfmio.com/prebid_display"
const VideoEndpoint = "https://reachms.bfmio.com/bid.json?exchange_id="
const VideoEndpointSuffix = "&prebidserver"

const beachfrontAdapterName = "BF_PREBID_S2S"
const beachfrontAdapterVersion = "0.1.1"

type BeachfrontAdapter struct {
	http *adapters.HTTPAdapter
	URI  string
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

func (a *BeachfrontAdapter) Name() string {
	return "beachfront"
}

func (a *BeachfrontAdapter) SkipNoCookies() bool {
	return true
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

func (a *BeachfrontAdapter) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
	errs := make([]error, 0, len(request.Imp))

	var beachfrontRequests BeachfrontRequests
	var reqJSON []byte
	var uri string

	uri = func() string {
		for i := range request.Imp {
			if request.Imp[i].Video != nil {
				// If there are any video imps, we will be running a video auction
				// and dropping all of the banner actions.
				return VideoEndpoint
			}
		}
		return BannerEndpoint
	}()

	beachfrontRequests, err := preprocess(request, uri)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

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

	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

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
func preprocess(req *openrtb.BidRequest, uri string) (BeachfrontRequests, error) {
	var beachfrontReqs BeachfrontRequests
	var err error

	if uri == BannerEndpoint {
		beachfrontReqs.Banner, err = getBannerRequest(req)
		if err != nil {
			return beachfrontReqs, err
		}
	} else {
		// If there were any Video imps in the request, we have skipped to here.
		beachfrontReqs.Video, err = getVideoRequest(req)
		if err != nil {
			return beachfrontReqs, err
		}
	}

	return beachfrontReqs, err
}

func getBannerRequest(req *openrtb.BidRequest) (BeachfrontBannerRequest, error) {
	var bannerImps int = 0
	var beachfrontReq BeachfrontBannerRequest = NewBeachfrontBannerRequest()
	/*
	 step through the prebid request "imp" and inject into the beachfront request. If we got to here,
	 then we have already stepped through the requested imps and verified that none are Video, so no
	 reason to check that here, but there could be Audio or Native (or maybe they are filtered out before
	 I get here based on the capabilities in bidder-info/beachfront.yaml?) .
	  */

	for _, imp := range req.Imp {
		if imp.Audio != nil {
			// Place holder
		} else if imp.Native != nil {
			// Place holder
		} else if imp.Banner != nil {
			beachfrontReq.Slots = append(beachfrontReq.Slots, BeachfrontSlot{})
			beachfrontReq.Slots[bannerImps].Sizes = append(beachfrontReq.Slots[bannerImps].Sizes, BeachfrontSize{})
			for j := range imp.Banner.Format {
				if j > 0 {
					// multi-banner.json test
					beachfrontReq.Slots[bannerImps].Sizes = append(beachfrontReq.Slots[bannerImps].Sizes, BeachfrontSize{})
				}
				beachfrontReq.Slots[bannerImps].Sizes[j].H = imp.Banner.Format[j].H
				beachfrontReq.Slots[bannerImps].Sizes[j].W = imp.Banner.Format[j].W
			}

			beachfrontReq.Slots[bannerImps].Bidfloor = imp.BidFloor

			var bidderExt adapters.ExtImpBidder
			if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
				// possible banner error 2
				return beachfrontReq, err
			}

			var beachfrontExt openrtb_ext.ExtImpBeachfront
			if err := json.Unmarshal(bidderExt.Bidder, &beachfrontExt); err != nil {
				// possible banner error 3 - supplemental/unmarshal-error-banner.json
				return beachfrontReq, err
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

			beachfrontReq.Slots[bannerImps].Slot = req.Imp[bannerImps].ID
			beachfrontReq.Slots[bannerImps].Id = beachfrontExt.AppId
			bannerImps++

		}
	}

	if req.User != nil {
		beachfrontReq.User = req.User.BuyerUID
	}

	if req.App != nil {
		beachfrontReq.Domain = req.App.Domain
		beachfrontReq.Page = req.App.ID
	} else {
		beachfrontReq.Domain = strings.Split(strings.Split(req.Site.Page, "//")[1], "/")[0]
		beachfrontReq.Page = req.Site.Page
	}

	return beachfrontReq, nil
}

/*
Prepare the request that has been received from Prebid.js, translating it to the beachfront format
*/
func getVideoRequest(req *openrtb.BidRequest) (BeachfrontVideoRequest, error) {
	var videoImps int = 0
	var beachfrontReq BeachfrontVideoRequest = NewBeachfrontVideoRequest()

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

			beachfrontReq.Imp[videoImps].Video.H = imp.Video.H
			beachfrontReq.Imp[videoImps].Video.W = imp.Video.W

			var bidderExt adapters.ExtImpBidder
			if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
				// possible video error - supplemental/unmarshal-error-banner.json
				return beachfrontReq, err
			}

			var beachfrontVideoExt openrtb_ext.ExtImpBeachfront
			if err := json.Unmarshal(bidderExt.Bidder, &beachfrontVideoExt); err != nil {
				// possible video error - supplemental/unmarshal-error-banner.json
				return beachfrontReq, err
			}

			beachfrontReq.Imp[videoImps].Bidfloor = beachfrontVideoExt.BidFloor
			//   A unique identifier for this impression within the context of
			//   the bid request (typically, starts with 1 and increments).
			beachfrontReq.Imp[videoImps].Id = videoImps
			beachfrontReq.Imp[videoImps].ImpId = imp.ID

			if req.Device != nil {
				beachfrontReq.Device.Geo.Ip = req.Device.IP
				beachfrontReq.Device.Ua = req.Device.UA
			}

			beachfrontReq.AppId = beachfrontVideoExt.AppId

			videoImps++
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
		beachfrontReq.Domain = req.App.Domain
		beachfrontReq.Site.Page = req.App.ID
	} else {
		beachfrontReq.Domain = strings.Split(strings.Split(req.Site.Page, "//")[1], "/")[0]
		beachfrontReq.Site.Page = req.Site.Page
	}

	return beachfrontReq, nil
}

func (a *BeachfrontAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var bidResp openrtb.BidResponse
	var err error
	var bidtype openrtb_ext.BidType = openrtb_ext.BidTypeBanner
	var isVideo bool = false

	for i := range internalRequest.Imp {
		if internalRequest.Imp[i].Video != nil {
			isVideo = true
			bidtype = openrtb_ext.BidTypeVideo
			break
		}
	}

	bidResp, err = postprocess(response, externalRequest, internalRequest.ID, isVideo)
	if err != nil {
		return nil, []error{fmt.Errorf("Failed to process the beachfront response\n%s", err)}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(BidCapacity)

	var errs []error
	for _, sb := range bidResp.SeatBid {
		for i := 0; i < len(sb.Bid); i++ {

			bid := sb.Bid[i]
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidtype,
			})
		}
	}
	return bidResponse, errs
}

func postprocess(response *adapters.ResponseData, externalRequest *adapters.RequestData, id string, isVideo bool) (openrtb.BidResponse, error) {
	var beachfrontResp []BeachfrontResponseSlot
	var openrtbResp openrtb.BidResponse
	var err error

	if isVideo {
		// Regular video ad. Beachfront returns video ads in openRTB format (or close enough).
		if err = json.Unmarshal(response.Body, &openrtbResp); err != nil {
			return openrtbResp, err
		}
		return postprocessVideo(openrtbResp, externalRequest, id)
	} else {
		/* Beachfront currently returns banner ads in a sparse format which is just the openRTB seatbid
		object. It needs to be wrapped up in openrtb format.
		*/
		if err = json.Unmarshal(response.Body, &beachfrontResp); err != nil {
			return openrtbResp, err
		}

		openrtbResp.ID = id
		for range beachfrontResp {
			openrtbResp.SeatBid = append(openrtbResp.SeatBid, openrtb.SeatBid{})
		}

		return postprocessBanner(openrtbResp, beachfrontResp, id)
	}
}

func postprocessBanner(openrtbResp openrtb.BidResponse, beachfrontResp []BeachfrontResponseSlot, id string) (openrtb.BidResponse, error) {
	r, _ := regexp.Compile("\\\"([0-9]+)")
	for k, _ := range openrtbResp.SeatBid {
		openrtbResp.SeatBid[k].Bid = append(openrtbResp.SeatBid[k].Bid, openrtb.Bid{
			CrID:  fmt.Sprintf("%s", r.FindStringSubmatch(beachfrontResp[k].Adm)[1]),
			ImpID: beachfrontResp[k].Slot,
			Price: beachfrontResp[k].Price,
			ID:    id,
			AdM:   beachfrontResp[k].Adm,
			H:     beachfrontResp[k].H,
			W:     beachfrontResp[k].W,
		})

		openrtbResp.SeatBid[k].Seat = Seat
	}

	return openrtbResp, nil
}

func postprocessVideo(openrtbResp openrtb.BidResponse, externalRequest *adapters.RequestData, id string) (openrtb.BidResponse, error) {
	var xtrnal BeachfrontVideoRequest
	var err error

	if err = json.Unmarshal(externalRequest.Body, &xtrnal); err != nil {
		return openrtbResp, err
	}

	/* there will only be one seatBid because beachfront only returns a single video ad
	but if that were to change this should work on all of them:
	*/
	for i := range openrtbResp.SeatBid {
		for j := range openrtbResp.SeatBid[i].Bid {
			openrtbResp.SeatBid[i].Bid[j].ImpID = xtrnal.Imp[i].ImpId
			openrtbResp.SeatBid[i].Bid[j].CrID = xtrnal.Imp[i].ImpId
			openrtbResp.SeatBid[i].Bid[j].H = xtrnal.Imp[i].Video.H
			openrtbResp.SeatBid[i].Bid[j].W = xtrnal.Imp[i].Video.W
			openrtbResp.SeatBid[i].Bid[j].ID = id
		}
		openrtbResp.SeatBid[i].Seat = Seat
	}

	return openrtbResp, nil
}

func NewBeachfrontBidder(client *http.Client) *BeachfrontAdapter {
	a := &adapters.HTTPAdapter{Client: client}
	return &BeachfrontAdapter{
		http: a,
		URI:  BannerEndpoint,
	}
}
