package rubicon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/prebid/prebid-server/pbs"

	"golang.org/x/net/context/ctxhttp"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
)

type RubiconAdapter struct {
	http         *adapters.HTTPAdapter
	URI          string
	usersyncInfo *pbs.UsersyncInfo
	XAPIUsername string
	XAPIPassword string
}

/* Name - export adapter name */
func (a *RubiconAdapter) Name() string {
	return "Rubicon"
}

// used for cookies and such
func (a *RubiconAdapter) FamilyName() string {
	return "rubicon"
}

func (a *RubiconAdapter) GetUsersyncInfo() *pbs.UsersyncInfo {
	return a.usersyncInfo
}

func (a *RubiconAdapter) SkipNoCookies() bool {
	return false
}

type rubiconParams struct {
	AccountId int                `json:"accountId"`
	SiteId    int                `json:"siteId"`
	ZoneId    int                `json:"zoneId"`
	Inventory json.RawMessage    `json:"inventory"`
	Visitor   json.RawMessage    `json:"visitor"`
	Video     rubiconVideoParams `json:"video"`
}

type rubiconImpExtRPTrack struct {
	Mint        string `json:"mint"`
	MintVersion string `json:"mint_version"`
}

type rubiconImpExtRP struct {
	ZoneID int                  `json:"zone_id"`
	Target json.RawMessage      `json:"target"`
	Track  rubiconImpExtRPTrack `json:"track"`
}

type rubiconImpExt struct {
	RP rubiconImpExtRP `json:"rp"`
}

type rubiconUserExtRP struct {
	Target json.RawMessage `json:"target"`
}

type rubiconUserExt struct {
	RP rubiconUserExtRP `json:"rp"`
}

type rubiconSiteExtRP struct {
	SiteID int `json:"site_id"`
}

type rubiconSiteExt struct {
	RP rubiconSiteExtRP `json:"rp"`
}

type rubiconPubExtRP struct {
	AccountID int `json:"account_id"`
}

type rubiconPubExt struct {
	RP rubiconPubExtRP `json:"rp"`
}

type rubiconBannerExtRP struct {
	SizeID     int    `json:"size_id,omitempty"`
	AltSizeIDs []int  `json:"alt_size_ids,omitempty"`
	MIME       string `json:"mime"`
}

type rubiconBannerExt struct {
	RP rubiconBannerExtRP `json:"rp"`
}

// ***** Video Extension *****
type rubiconVideoParams struct {
	Language     string `json:"language,omitempty"`
	PlayerHeight int    `json:"playerHeight,omitempty"`
	PlayerWidth  int    `json:"playerWidth,omitempty"`
	VideoSizeID  int    `json:"size_id,omitempty"`
	Skip         int    `json:"skip,omitempty"`
	SkipDelay    int    `json:"skipdelay,omitempty"`
}

type rubiconVideoExt struct {
	Skip      int               `json:"skip,omitempty"`
	SkipDelay int               `json:"skipdelay,omitempty"`
	RP        rubiconVideoExtRP `json:"rp"`
}

type rubiconVideoExtRP struct {
	SizeID int `json:"size_id,omitempty"`
}

type rubiconTargetingExt struct {
	RP rubiconTargetingExtRP `json:"rp"`
}

type rubiconTargetingExtRP struct {
	Targeting []rubiconTargetingObj `json:"targeting"`
}

type rubiconTargetingObj struct {
	Key    string   `json:"key"`
	Values []string `json:"values"`
}

type rubiconDeviceExtRP struct {
	PixelRatio float64 `json:"pixelratio"`
}

type rubiconDeviceExt struct {
	RP rubiconDeviceExtRP `json:"rp"`
}

type rubiconUser struct {
	Language string `json:"language"`
}

type rubiSize struct {
	w uint16
	h uint16
}

var rubiSizeMap = map[rubiSize]int{
	rubiSize{w: 468, h: 60}:    1,
	rubiSize{w: 728, h: 90}:    2,
	rubiSize{w: 728, h: 91}:    2,
	rubiSize{w: 120, h: 600}:   8,
	rubiSize{w: 160, h: 600}:   9,
	rubiSize{w: 300, h: 600}:   10,
	rubiSize{w: 300, h: 250}:   15,
	rubiSize{w: 300, h: 251}:   15,
	rubiSize{w: 336, h: 280}:   16,
	rubiSize{w: 300, h: 100}:   19,
	rubiSize{w: 980, h: 120}:   31,
	rubiSize{w: 250, h: 360}:   32,
	rubiSize{w: 180, h: 500}:   33,
	rubiSize{w: 980, h: 150}:   35,
	rubiSize{w: 468, h: 400}:   37,
	rubiSize{w: 930, h: 180}:   38,
	rubiSize{w: 320, h: 50}:    43,
	rubiSize{w: 300, h: 50}:    44,
	rubiSize{w: 300, h: 300}:   48,
	rubiSize{w: 300, h: 1050}:  54,
	rubiSize{w: 970, h: 90}:    55,
	rubiSize{w: 970, h: 250}:   57,
	rubiSize{w: 1000, h: 90}:   58,
	rubiSize{w: 320, h: 80}:    59,
	rubiSize{w: 1000, h: 1000}: 61,
	rubiSize{w: 640, h: 480}:   65,
	rubiSize{w: 320, h: 480}:   67,
	rubiSize{w: 1800, h: 1000}: 68,
	rubiSize{w: 320, h: 320}:   72,
	rubiSize{w: 320, h: 160}:   73,
	rubiSize{w: 980, h: 240}:   78,
	rubiSize{w: 980, h: 300}:   79,
	rubiSize{w: 980, h: 400}:   80,
	rubiSize{w: 480, h: 300}:   83,
	rubiSize{w: 970, h: 310}:   94,
	rubiSize{w: 970, h: 210}:   96,
	rubiSize{w: 480, h: 320}:   101,
	rubiSize{w: 768, h: 1024}:  102,
	rubiSize{w: 480, h: 280}:   103,
	rubiSize{w: 1000, h: 300}:  113,
	rubiSize{w: 320, h: 100}:   117,
	rubiSize{w: 800, h: 250}:   125,
	rubiSize{w: 200, h: 600}:   126,
}

func lookupSize(s openrtb.Format) (int, error) {
	if sz, ok := rubiSizeMap[rubiSize{w: uint16(s.W), h: uint16(s.H)}]; ok {
		return sz, nil
	}
	return 0, fmt.Errorf("Size %dx%d not found", s.W, s.H)
}

func parseRubiconSizes(sizes []openrtb.Format) (primary int, alt []int, err error) {
	alt = make([]int, 0, len(sizes)-1)
	for _, size := range sizes {
		rs, lerr := lookupSize(size)
		if lerr != nil {
			continue
		}
		if primary == 0 {
			primary = rs
		} else {
			alt = append(alt, rs)
		}
	}
	if primary == 0 {
		err = fmt.Errorf("No valid sizes")
	}
	return
}

func (a *RubiconAdapter) callOne(ctx context.Context, req *pbs.PBSRequest, reqJSON bytes.Buffer) (result adapters.CallOneResult, err error) {
	httpReq, err := http.NewRequest("POST", a.URI, &reqJSON)
	httpReq.Header.Add("Content-Type", "application/json;charset=utf-8")
	httpReq.Header.Add("Accept", "application/json")
	httpReq.Header.Add("User-Agent", "prebid-server/1.0")
	httpReq.SetBasicAuth(a.XAPIUsername, a.XAPIPassword)

	rubiResp, e := ctxhttp.Do(ctx, a.http.Client, httpReq)
	if e != nil {
		err = e
		return
	}

	defer rubiResp.Body.Close()
	body, _ := ioutil.ReadAll(rubiResp.Body)
	result.ResponseBody = string(body)

	result.StatusCode = rubiResp.StatusCode

	if rubiResp.StatusCode == 204 {
		return
	}

	if rubiResp.StatusCode != 200 {
		err = fmt.Errorf("HTTP status %d; body: %s", rubiResp.StatusCode, result.ResponseBody)
		return
	}

	var bidResp openrtb.BidResponse
	err = json.Unmarshal(body, &bidResp)
	if err != nil {
		return
	}
	if len(bidResp.SeatBid) == 0 {
		return
	}
	if len(bidResp.SeatBid[0].Bid) == 0 {
		return
	}
	bid := bidResp.SeatBid[0].Bid[0]

	result.Bid = &pbs.PBSBid{
		AdUnitCode:  bid.ImpID,
		Price:       bid.Price,
		Adm:         bid.AdM,
		Creative_id: bid.CrID,
		Width:       bid.W,
		Height:      bid.H,
		DealId:      bid.DealID,
	}

	// Pull out any server-side determined targeting
	var rpExtTrg rubiconTargetingExt

	if err := json.Unmarshal([]byte(bid.Ext), &rpExtTrg); err == nil {
		// Converting string => array(string) to string => string
		targeting := make(map[string]string)

		// Only pick off the first for now
		for _, target := range rpExtTrg.RP.Targeting {
			targeting[target.Key] = target.Values[0]
		}

		result.Bid.AdServerTargeting = targeting
	}

	return
}

func (a *RubiconAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	requests := make([]bytes.Buffer, len(bidder.AdUnits))
	supportedMediaTypes := []pbs.MediaType{pbs.MEDIA_TYPE_BANNER, pbs.MEDIA_TYPE_VIDEO}

	for i, unit := range bidder.AdUnits {
		rubiReq, err := adapters.MakeOpenRTBGeneric(req, bidder, a.FamilyName(), supportedMediaTypes, true)
		if err != nil {
			continue
		}

		// Only grab this ad unit
		// Not supporting multi-media-type add-unit yet
		rubiReq.Imp = rubiReq.Imp[i : i+1]

		// Amend it with RP-specific information
		var params rubiconParams
		err = json.Unmarshal(unit.Params, &params)

		if err != nil {
			return nil, err
		}
		if params.AccountId == 0 {
			return nil, errors.New("Missing accountId param")
		}
		if params.SiteId == 0 {
			return nil, errors.New("Missing siteId param")
		}
		if params.ZoneId == 0 {
			return nil, errors.New("Missing zoneId param")
		}

		var mint, mintVersion string
		mint = "prebid"
		mintVersion = req.SDK.Source + "_" + req.SDK.Platform + "_" + req.SDK.Version
		track := rubiconImpExtRPTrack{Mint: mint, MintVersion: mintVersion}

		impExt := rubiconImpExt{RP: rubiconImpExtRP{
			ZoneID: params.ZoneId,
			Target: params.Inventory,
			Track:  track,
		}}
		rubiReq.Imp[0].Ext, err = json.Marshal(&impExt)

		// Copy the $.user object and amend with $.user.ext.rp.target
		// Copy avoids race condition since it points to ref & shared with other adapters
		userCopy := *rubiReq.User
		userExt := rubiconUserExt{RP: rubiconUserExtRP{Target: params.Visitor}}
		userCopy.Ext, err = json.Marshal(&userExt)
		// Assign back our copy
		rubiReq.User = &userCopy

		deviceCopy := *rubiReq.Device
		deviceExt := rubiconDeviceExt{RP: rubiconDeviceExtRP{PixelRatio: rubiReq.Device.PxRatio}}
		deviceCopy.Ext, err = json.Marshal(&deviceExt)
		rubiReq.Device = &deviceCopy

		if rubiReq.Imp[0].Video != nil {
			videoExt := rubiconVideoExt{Skip: params.Video.Skip, SkipDelay: params.Video.SkipDelay, RP: rubiconVideoExtRP{SizeID: params.Video.VideoSizeID}}
			rubiReq.Imp[0].Video.Ext, err = json.Marshal(&videoExt)
		} else {
			primarySizeID, altSizeIDs, err := parseRubiconSizes(unit.Sizes)
			if err != nil {
				return nil, err
			}
			bannerExt := rubiconBannerExt{RP: rubiconBannerExtRP{SizeID: primarySizeID, AltSizeIDs: altSizeIDs, MIME: "text/html"}}
			rubiReq.Imp[0].Banner.Ext, err = json.Marshal(&bannerExt)
		}

		siteExt := rubiconSiteExt{RP: rubiconSiteExtRP{SiteID: params.SiteId}}
		pubExt := rubiconPubExt{RP: rubiconPubExtRP{AccountID: params.AccountId}}

		if rubiReq.Site != nil {
			siteCopy := *rubiReq.Site
			siteCopy.Ext, err = json.Marshal(&siteExt)
			siteCopy.Publisher = &openrtb.Publisher{}
			siteCopy.Publisher.Ext, err = json.Marshal(&pubExt)
			rubiReq.Site = &siteCopy
		} else {
			var rubiconUser rubiconUser
			err = json.Unmarshal(req.PBSUser, &rubiconUser)
			site := &openrtb.Site{}
			site.Content = &openrtb.Content{}
			site.Content.Language = rubiconUser.Language
			rubiReq.Site = site
		}

		if rubiReq.App != nil {
			appCopy := *rubiReq.App
			appCopy.Ext, err = json.Marshal(&siteExt)
			appCopy.Publisher = &openrtb.Publisher{}
			appCopy.Publisher.Ext, err = json.Marshal(&pubExt)
			rubiReq.App = &appCopy
		}

		err = json.NewEncoder(&requests[i]).Encode(rubiReq)
		if err != nil {
			return nil, err
		}
	}

	ch := make(chan adapters.CallOneResult)
	for i := range bidder.AdUnits {
		go func(bidder *pbs.PBSBidder, reqJSON bytes.Buffer) {
			result, err := a.callOne(ctx, req, reqJSON)
			result.Error = err
			if result.Bid != nil {
				result.Bid.BidderCode = bidder.BidderCode
				result.Bid.BidID = bidder.LookupBidID(result.Bid.AdUnitCode)
				if result.Bid.BidID == "" {
					result.Error = fmt.Errorf("Unknown ad unit code '%s'", result.Bid.AdUnitCode)
					result.Bid = nil
				}
			}
			ch <- result
		}(bidder, requests[i])
	}

	var err error

	bids := make(pbs.PBSBidSlice, 0)
	for i := 0; i < len(bidder.AdUnits); i++ {
		result := <-ch
		if result.Bid != nil {
			bids = append(bids, result.Bid)
		}
		if req.IsDebug {
			debug := &pbs.BidderDebug{
				RequestURI:   a.URI,
				RequestBody:  requests[i].String(),
				StatusCode:   result.StatusCode,
				ResponseBody: result.ResponseBody,
			}
			bidder.Debug = append(bidder.Debug, debug)
		}
		if result.Error != nil {
			fmt.Printf("Error in rubicon adapter: %v", result.Error)
			err = result.Error
		}
	}

	if len(bids) == 0 {
		return nil, err
	}
	return bids, nil
}

func appendTrackerToUrl(uri string, tracker string) (res string) {
	// Append integration method. Adapter init happens once
	urlObject, err := url.Parse(uri)
	// No other exception throwing mechanism in this stack, so ignoring parse errors.
	if err == nil {
		values := urlObject.Query()
		values.Add("tk_xint", tracker)
		urlObject.RawQuery = values.Encode()
		res = urlObject.String()
	} else {
		res = uri
	}
	return
}

func NewRubiconAdapter(config *adapters.HTTPAdapterConfig, uri string, xuser string, xpass string, tracker string, usersyncURL string) *RubiconAdapter {
	a := adapters.NewHTTPAdapter(config)

	uri = appendTrackerToUrl(uri, tracker)

	info := &pbs.UsersyncInfo{
		URL:         usersyncURL,
		Type:        "redirect",
		SupportCORS: false,
	}

	return &RubiconAdapter{
		http:         a,
		URI:          uri,
		usersyncInfo: info,
		XAPIUsername: xuser,
		XAPIPassword: xpass,
	}
}
