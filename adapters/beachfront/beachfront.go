package beachfront

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
)

const ForceQA = true
const Seat = "beachfront"
const VideoFlag = "video"
const TestID = "test_id"

const BannerEndpoint = "https://display.bfmio.com/prebid_display"

// const VideoEndpoint = "https://reachms.bfmio.com/bid.json?exchange_id="
// const VideoEndpoint = "http://10.0.0.181/dump.php?exchange_id="
const VideoEndpoint = "http://qa.bfmio.com/bid.json?exchange_id="
const VideoEndpointSuffix = "&prebidserver"

const beachfrontAdapterName = "BF_PREBID_S2S"
const beachfrontAdapterVersion = "0.1.1"

const beachfrontVideoRequestTemplate = `{
    		"isPrebid": true,
    		"appId": "",
    		"domain": "",
    		"id": "",
	    	"imp": [{
	    		"id": 0,
	    		"impid" : "",
	      		"video": {
				"w": 0,
				"h": 0
	      		},
	      		"bidfloor": 0.00
	    	}],
	    	"site": {
	      		"page": ""
	    	},
	    	"device": {
			"ua":"Go-http-client/1.1",
	      		"devicetype": 0,
	      		"geo": {}
	    	},
	    	"cur": ["USD"]
  	}`

const beachfrontBannerRequestTemplate = `{
	"slots":[
		{
			"slot":"",
			"id":"",
			"bidfloor": 0.00,
			"sizes":[
				{
					"w":0,
					"h":0
				}
			]
		}
	],
	"domain":"",
	"page":"",
	"referrer":"",
	"search":"",
	"secure":1,
	"deviceOs":"",
	"deviceModel":"",
	"isMobile":0,
	"ua":"Go-http-client/1.1",
	"dnt":0,
	"user": "",
	"adapterName": "` + beachfrontAdapterName + `",
	"adapterVersion":"` + beachfrontAdapterVersion + `",
	"ip":""
	}`

type BeachfrontAdapter struct {
	http *adapters.HTTPAdapter
}

type BeachfrontRequests struct {
	Banner    BeachfrontBannerRequest
	Video     BeachfrontVideoRequest
	VideoFlag bool
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

// -----------------------------------------------------------------------

type BeachfrontResponseSlot struct {
	CrID  string  `json:"crid"`
	Price float64 `json:"price"`
	W     uint64  `json:"w"`
	H     uint64  `json:"h"`
	Slot  string  `json:"slot"`
	Adm   string  `json:"adm"`
}

// -----------------------------------------------------------------------

// Name - export adapter name
func (a *BeachfrontAdapter) Name() string {
	return "beachfront"
}

// Corresponds to the bidder name in cookies and requests
/*
func (a *BeachfrontAdapter) FamilyName() string {
	return "beachfront"
}
*/

func (a *BeachfrontAdapter) SkipNoCookies() bool {
	return false
}

func (a *BeachfrontAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	bids := make(pbs.PBSBidSlice, 0)
	return bids, nil
}

func (a *BeachfrontAdapter) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
	out, _ := json.Marshal(request)
	fmt.Printf("request /n%s/n", request)
	fmt.Printf("out /n%s/n", out)
	// glog.Info(out)

	errs := make([]error, 0, len(request.Imp))

	if len(request.Imp) == 0 {
		return nil, errs
	}

	var uri string
	var beachfrontRequests BeachfrontRequests
	var reqJSON []byte

	beachfrontRequests, uri, err := preprocess(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	if beachfrontRequests.VideoFlag {
		reqJSON, err = json.Marshal(beachfrontRequests.Video)
	} else {
		reqJSON, err = json.Marshal(beachfrontRequests.Banner)
	}

	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	glog.Info("\nUser.ID : ", request.User.ID)
	glog.Info("\nUser.BuyerUID : ", request.User.BuyerUID)
	glog.Info("\nRequest URL : ", uri)

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
func preprocess(req *openrtb.BidRequest) (BeachfrontRequests, string, error) {
	var beachfrontReqs BeachfrontRequests
	var video BeachfrontVideoRequest

	beachfrontReqs.VideoFlag = false

	banner, uri, err := getBannerRequest(req)
	if err != nil {
		return beachfrontReqs, uri, err
	}

	// We did not get all the way through the request without hitting a video Imp,
	if uri == VideoFlag {
		beachfrontReqs.VideoFlag = true
		video, uri, err = getVideoRequest(req)
		if err != nil {
			return beachfrontReqs, uri, err
		}
	}

	beachfrontReqs.Banner = banner
	beachfrontReqs.Video = video

	return beachfrontReqs, uri, nil

}

func getBannerRequest(req *openrtb.BidRequest) (BeachfrontBannerRequest, string, error) {
	var beachfrontReq BeachfrontBannerRequest
	var uri string = BannerEndpoint

	if req.Imp[0].Video != nil {
		return beachfrontReq, VideoFlag, nil
	}

	dec := json.NewDecoder(strings.NewReader(beachfrontBannerRequestTemplate))
	for {
		if err := dec.Decode(&beachfrontReq); err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}
	}

	// step through the prebid request "imp" and inject into the beachfront request.
	for k, imp := range req.Imp {
		if imp.Video != nil {
			return beachfrontReq, VideoFlag, nil
		} else if imp.Banner != nil {
			// Set the beachfront "size" values to match the prebid "format" values
			for j := range imp.Banner.Format {
				// The template has 1 Size struct, so use that one first, then add them as needed.
				if j > 0 {
					beachfrontReq.Slots[k].Sizes = append(beachfrontReq.Slots[k].Sizes, BeachfrontSize{})
				}

				beachfrontReq.Slots[k].Sizes[j].H = imp.Banner.Format[j].H
				beachfrontReq.Slots[k].Sizes[j].W = imp.Banner.Format[j].W
			}

			beachfrontReq.Slots[k].Bidfloor = imp.BidFloor

			var bidderExt adapters.ExtImpBidder
			if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
				return beachfrontReq, uri, err
			}

			var beachfrontExt openrtb_ext.ExtImpBeachfront
			if err := json.Unmarshal(bidderExt.Bidder, &beachfrontExt); err != nil {
				return beachfrontReq, uri, err
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

			beachfrontReq.Slots[k].Slot = req.Imp[k].ID

			beachfrontReq.Slots[k].Id = beachfrontExt.AppId
		}
	}

	beachfrontReq.Domain = strings.Split(strings.Split(req.Site.Page, "//")[1], "/")[0]
	beachfrontReq.Page = req.Site.Page

	return beachfrontReq, uri, nil

}

/*
Prepare the request that has been received from Prebid.js, translating it to the beachfront format
*/
func getVideoRequest(req *openrtb.BidRequest) (BeachfrontVideoRequest, string, error) {
	var beachfrontVideoReq BeachfrontVideoRequest
	var uri string = VideoEndpoint
	var i int = 1

	dec := json.NewDecoder(strings.NewReader(beachfrontVideoRequestTemplate))

	for {
		if err := dec.Decode(&beachfrontVideoReq); err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}
	}

	// step through the prebid request "imp" and inject into the beachfrontVideo request
	for k, imp := range req.Imp {
		if imp.Video != nil {
			//   Unique ID of the bid request, provided by the exchange.
			beachfrontVideoReq.Id = req.ID

			// The template has 1 Imp struct, so use that one first, then add them as needed.
			if k > 0 {
				beachfrontVideoReq.Imp = append(beachfrontVideoReq.Imp, BeachfrontVideoImp{})
			}

			beachfrontVideoReq.Imp[k].Video.H = req.Imp[k].Video.H
			beachfrontVideoReq.Imp[k].Video.W = req.Imp[k].Video.W

			var bidderExt adapters.ExtImpBidder
			if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
				return beachfrontVideoReq, uri, err
			}

			var beachfrontVideoExt openrtb_ext.ExtImpBeachfront
			if err := json.Unmarshal(bidderExt.Bidder, &beachfrontVideoExt); err != nil {
				return beachfrontVideoReq, uri, err
			}

			beachfrontVideoReq.Imp[k].Bidfloor = beachfrontVideoExt.BidFloor
			//   A unique identifier for this impression within the context of
			//   the bid request (typically, starts with 1 and increments).
			beachfrontVideoReq.Imp[k].Id = i

			beachfrontVideoReq.Imp[k].ImpId = req.Imp[k].ID
			// beachfrontVideoReq.Imp[k].CrID = req.

			if req.Device != nil {
				beachfrontVideoReq.Device.Geo.Ip = req.Device.IP
				beachfrontVideoReq.Device.Ua = req.Device.UA
			}

			beachfrontVideoReq.AppId = beachfrontVideoExt.AppId
		}
		i++
	}

	beachfrontVideoReq.Domain = strings.Split(strings.Split(req.Site.Page, "//")[1], "/")[0]
	beachfrontVideoReq.Site.Page = req.Site.Page
	uri = uri + beachfrontVideoReq.AppId + VideoEndpointSuffix

	return beachfrontVideoReq, uri, nil
}

func (a *BeachfrontAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) ([]*adapters.TypedBid, []error) {
	var bidResp openrtb.BidResponse
	var err error
	var bidtype openrtb_ext.BidType = openrtb_ext.BidTypeBanner
	var isVideo bool = false

	if internalRequest.Imp[0].Video != nil {
		isVideo = true
		bidtype = openrtb_ext.BidTypeVideo
	}

	// I have the __io_cid cookie when I get here in video. Should I set the user id to this?
	glog.Info("\nreceived	:", response.Headers.Get("Set-Cookie"))
	glog.Info(internalRequest)
	glog.Info(externalRequest)

	// Cookie debugging

	/*
		c := new(http.Cookie)
		c.Value = response.Headers.Get("Set-Cookie")
		h := new(url.URL)
		h.Host = "localhost"

		glog.Info("\nc		:", c)
		glog.Info("\njar	:", a.http.Client)
	*/

	// end cookie debugging

	bidResp, err = postprocess(response, externalRequest, internalRequest.ID, isVideo)
	if err != nil {
		return nil, []error{fmt.Errorf("Failed to process the beachfront response\n%s", err)}
	}

	bids := make([]*adapters.TypedBid, 0, 5)
	for _, sb := range bidResp.SeatBid {
		for _, bid := range sb.Bid {
			bids = append(bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidtype,
			})
		}
	}

	return bids, nil
}

func postprocess(response *adapters.ResponseData, externalRequest *adapters.RequestData, id string, isVideo bool) (openrtb.BidResponse, error) {
	var beachfrontResp []BeachfrontResponseSlot
	var openrtbResp openrtb.BidResponse
	var err error

	if isVideo {
		// Regular video ad. Beachfront returns video ads in openRTB format.
		if err = json.Unmarshal(response.Body, &openrtbResp); err != nil {
			return openrtbResp, err
		}
		glog.Info(openrtbResp)
		return postprocessVideo(openrtbResp, externalRequest)
	} else {
		if id != TestID {
			/* Beachfront currently returns banner ads in a sparse format which is just the openRTB seatbid
			object. It needs to be wrapped up in openrtb format.
			*/
			if err = json.Unmarshal(response.Body, &beachfrontResp); err != nil {
				return openrtbResp, err
			}

			openrtbResp.ID = id
			openrtbResp.SeatBid = append(openrtbResp.SeatBid, openrtb.SeatBid{})
		} else {
			if err = json.Unmarshal(response.Body, &openrtbResp); err != nil {
				return openrtbResp, err
			}

		}

		return postprocessBanner(openrtbResp, beachfrontResp)
	}

}

func postprocessBanner(openrtbResp openrtb.BidResponse, beachfrontResp []BeachfrontResponseSlot) (openrtb.BidResponse, error) {
	if beachfrontResp == nil {
		return openrtbResp, nil
	}

	r, _ := regexp.Compile("\\\"([0-9]+)")
	for k, _ := range openrtbResp.SeatBid {
		openrtbResp.SeatBid[k].Bid = append(openrtbResp.SeatBid[k].Bid, openrtb.Bid{
			ID:    fmt.Sprintf("%s", r.FindStringSubmatch(beachfrontResp[k].Adm)[1]),
			ImpID: beachfrontResp[k].Slot,
			Price: beachfrontResp[k].Price,
			CrID:  beachfrontResp[k].CrID,
			AdM:   beachfrontResp[k].Adm,
			H:     beachfrontResp[k].H,
			W:     beachfrontResp[k].W,
		})

		openrtbResp.SeatBid[k].Seat = Seat
	}

	return openrtbResp, nil
}

func postprocessVideo(openrtbResp openrtb.BidResponse, externalRequest *adapters.RequestData) (openrtb.BidResponse, error) {
	var xtrnal BeachfrontVideoRequest
	var err error

	if err = json.Unmarshal(externalRequest.Body, &xtrnal); err != nil {
		return openrtbResp, err
	}

	for i, _ := range openrtbResp.SeatBid {
		for j, _ := range openrtbResp.SeatBid[i].Bid {
			openrtbResp.SeatBid[i].Bid[j].ImpID = xtrnal.Imp[i].ImpId
			openrtbResp.SeatBid[i].Bid[j].CrID = xtrnal.Imp[i].ImpId // find a better value or random
			openrtbResp.SeatBid[i].Bid[j].H = xtrnal.Imp[i].Video.H
			openrtbResp.SeatBid[i].Bid[j].W = xtrnal.Imp[i].Video.W

			if ForceQA {
				openrtbResp.SeatBid[i].Bid[j].NURL = strings.Replace(openrtbResp.SeatBid[i].Bid[j].NURL,
					"evt.bfmio.com",
					"qa.bfmio.com", 1)
			}
		}
		openrtbResp.SeatBid[i].Seat = Seat
	}

	return openrtbResp, nil
}

func NewBeachfrontAdapter(config *adapters.HTTPAdapterConfig) *BeachfrontAdapter {
	return NewBeachfrontBidder(adapters.NewHTTPAdapter(config).Client)
}

func NewBeachfrontBidder(client *http.Client) *BeachfrontAdapter {
	a := &adapters.HTTPAdapter{Client: client}
	return &BeachfrontAdapter{
		http: a,
	}
}
