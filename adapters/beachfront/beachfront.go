package beachfront

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"github.com/golang/glog"
)

const Seat = "beachfront"
const TestID = "test_id"
const BidCapacity = 5

const BannerEndpoint = "https://display.bfmio.com/prebid_display"
const VideoEndpoint = "https://reachms.bfmio.com/bid.json?exchange_id="
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
	    	"user": {
		        "buyeruid" : "",
		        "id" : ""
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
	"adapterName": "` + beachfrontAdapterName + `",
	"adapterVersion":"` + beachfrontAdapterVersion + `",
	"ip":""
	}`

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

func (a *BeachfrontAdapter) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
	errs := make([]error, 0, len(request.Imp))

	if len(request.Imp) == 0 {
		return nil, errs
	}

	var beachfrontRequests BeachfrontRequests
	var reqJSON []byte

	a.URI = func() string {
		for i := range request.Imp {
			if request.Imp[i].Video != nil {
				return VideoEndpoint
			}
		}
		return BannerEndpoint
	}()

	beachfrontRequests, err := preprocess(request, a.URI)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	if a.URI == VideoEndpoint {
		reqJSON, err = json.Marshal(beachfrontRequests.Video)
		a.URI = a.URI + beachfrontRequests.Video.AppId + VideoEndpointSuffix
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

	if a.URI == BannerEndpoint {
		if request.User != nil {
			headers.Add("Cookie", "UserID="+request.User.ID+"; __io_cid="+request.User.BuyerUID+"; PublisherID="+request.Site.Publisher.ID)
		}
	}

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     a.URI,
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
	} else if uri == VideoEndpoint {
		beachfrontReqs.Video, err = getVideoRequest(req)
		if err != nil {
			return beachfrontReqs, err
		}
	} else {
		err = errors.New("Invalid beachfront endpoint")
	}

	return beachfrontReqs, err
}

func getBannerRequest(req *openrtb.BidRequest) (BeachfrontBannerRequest, error) {
	var beachfrontReq BeachfrontBannerRequest

	if req.Imp[0].Video != nil {
		return beachfrontReq, nil
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
			return beachfrontReq, nil
		} else if imp.Banner != nil {
			// Set the beachfront "size" values to match the prebid "format" values
			for j := range imp.Banner.Format {
				// The template has 1 Size struct, so use that one first, then add them as needed.
				if j > 0 {
					beachfrontReq.Slots[k].Sizes = append(beachfrontReq.Slots[k].Sizes, BeachfrontSize{})
				}

				glog.Info(beachfrontReq.Slots)
				glog.Info(j)
				glog.Info(k)
				glog.Info(len(beachfrontReq.Slots))
				glog.Info(len(beachfrontReq.Slots[k].Sizes))

				// 0011 // 1012 // 0120
				// 0011 // 1012 // 011?
				// 0011 // 1011


				beachfrontReq.Slots[k].Sizes[j].H = imp.Banner.Format[j].H
				beachfrontReq.Slots[k].Sizes[j].W = imp.Banner.Format[j].W
			}

			beachfrontReq.Slots[k].Bidfloor = imp.BidFloor

			var bidderExt adapters.ExtImpBidder
			if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
				return beachfrontReq, err
			}

			var beachfrontExt openrtb_ext.ExtImpBeachfront
			if err := json.Unmarshal(bidderExt.Bidder, &beachfrontExt); err != nil {
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

			beachfrontReq.Slots[k].Slot = req.Imp[k].ID

			beachfrontReq.Slots[k].Id = beachfrontExt.AppId
		}
	}

	if req.User != nil {
		//   Buyer-specific ID for the user as mapped by the exchange for
		//   the buyer. At least one of buyeruid or id is recommended.
		beachfrontReq.User = req.User.BuyerUID
	} else {
	}

	beachfrontReq.Domain = strings.Split(strings.Split(req.Site.Page, "//")[1], "/")[0]
	beachfrontReq.Page = req.Site.Page

	return beachfrontReq, nil
}

/*
Prepare the request that has been received from Prebid.js, translating it to the beachfront format
*/
func getVideoRequest(req *openrtb.BidRequest) (BeachfrontVideoRequest, error) {
	var beachfrontReq BeachfrontVideoRequest
	var i int = 1

	dec := json.NewDecoder(strings.NewReader(beachfrontVideoRequestTemplate))

	for {
		if err := dec.Decode(&beachfrontReq); err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}
	}

	// step through the prebid request "imp" and inject into the beachfrontVideo request
	for k, imp := range req.Imp {
		if imp.Video != nil {
			//   Unique ID of the bid request, provided by the exchange.
			beachfrontReq.Id = req.ID

			// The template has 1 Imp struct, so use that one first, then add them as needed.
			if k > 0 {
				beachfrontReq.Imp = append(beachfrontReq.Imp, BeachfrontVideoImp{})
			}

			beachfrontReq.Imp[k].Video.H = req.Imp[k].Video.H
			beachfrontReq.Imp[k].Video.W = req.Imp[k].Video.W

			var bidderExt adapters.ExtImpBidder
			if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
				return beachfrontReq, err
			}

			var beachfrontVideoExt openrtb_ext.ExtImpBeachfront
			if err := json.Unmarshal(bidderExt.Bidder, &beachfrontVideoExt); err != nil {
				return beachfrontReq, err
			}

			beachfrontReq.Imp[k].Bidfloor = beachfrontVideoExt.BidFloor
			//   A unique identifier for this impression within the context of
			//   the bid request (typically, starts with 1 and increments).
			beachfrontReq.Imp[k].Id = i
			beachfrontReq.Imp[k].ImpId = req.Imp[k].ID

			if req.Device != nil {
				beachfrontReq.Device.Geo.Ip = req.Device.IP
				beachfrontReq.Device.Ua = req.Device.UA
			}

			beachfrontReq.AppId = beachfrontVideoExt.AppId
		}
		i++
	}

	if req.User.ID != "" {
		//   Exchange-specific ID for the user. At least one of id or
		//   buyeruid is recommended.
		beachfrontReq.User.ID = req.User.ID
	} else {
	}

	if req.User.BuyerUID != "" {
		//   Buyer-specific ID for the user as mapped by the exchange for
		//   the buyer. At least one of buyeruid or id is recommended.
		beachfrontReq.User.BuyerUID = req.User.BuyerUID
	} else {
	}

	beachfrontReq.Domain = strings.Split(strings.Split(req.Site.Page, "//")[1], "/")[0]
	beachfrontReq.Site.Page = req.Site.Page

	return beachfrontReq, nil
}

func (a *BeachfrontAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var bidResp openrtb.BidResponse
	var err error
	var bidtype openrtb_ext.BidType = openrtb_ext.BidTypeBanner
	var isVideo bool = false

	if internalRequest.Imp[0].Video != nil {
		isVideo = true
		bidtype = openrtb_ext.BidTypeVideo
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
		// Regular video ad. Beachfront returns video ads in openRTB format.
		if err = json.Unmarshal(response.Body, &openrtbResp); err != nil {
			return openrtbResp, err
		}
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

	for i := range openrtbResp.SeatBid {
		for j := range openrtbResp.SeatBid[i].Bid {
			openrtbResp.SeatBid[i].Bid[j].ImpID = xtrnal.Imp[i].ImpId
			openrtbResp.SeatBid[i].Bid[j].CrID = xtrnal.Imp[i].ImpId
			openrtbResp.SeatBid[i].Bid[j].H = xtrnal.Imp[i].Video.H
			openrtbResp.SeatBid[i].Bid[j].W = xtrnal.Imp[i].Video.W
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
