package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/prebid/prebid-server/pbs"

	"golang.org/x/net/context/ctxhttp"

	"github.com/prebid/openrtb"
)

type RubiconAdapter struct {
	http         *HTTPAdapter
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

type rubiconParams struct {
	AccountId int `json:"accountId"`
	SiteId    int `json:"siteId"`
	ZoneId    int `json:"zoneId"`
}

type rubiconImpExtRP struct {
	ZoneID int `json:"zone_id"`
}

type rubiconImpExt struct {
	RP rubiconImpExtRP `json:"rp"`
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

type rubiSize struct {
	w uint16
	h uint16
}

var rubiSizeMap = map[rubiSize]int{
	rubiSize{w: 468, h: 60}:    1,
	rubiSize{w: 728, h: 90}:    2,
	rubiSize{w: 120, h: 600}:   8,
	rubiSize{w: 160, h: 600}:   9,
	rubiSize{w: 300, h: 600}:   10,
	rubiSize{w: 300, h: 250}:   15,
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

func (a *RubiconAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	rpReq := makeOpenRTBGeneric(req, bidder, a.FamilyName())

	for i, unit := range bidder.AdUnits {
		var params rubiconParams
		err := json.Unmarshal(unit.Params, &params)
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
		impExt := rubiconImpExt{RP: rubiconImpExtRP{ZoneID: params.ZoneId}}
		rpReq.Imp[i].Ext, err = json.Marshal(&impExt)

		primarySizeID, altSizeIDs, err := parseRubiconSizes(unit.Sizes)
		if err != nil {
			return nil, err
		}

		bannerExt := rubiconBannerExt{RP: rubiconBannerExtRP{SizeID: primarySizeID, AltSizeIDs: altSizeIDs, MIME: "text/html"}}
		rpReq.Imp[i].Banner.Ext, err = json.Marshal(&bannerExt)
		// params are per-unit, so site may overwrite itself
		siteExt := rubiconSiteExt{RP: rubiconSiteExtRP{SiteID: params.SiteId}}
		pubExt := rubiconPubExt{RP: rubiconPubExtRP{AccountID: params.AccountId}}
		if rpReq.Site != nil {
			rpReq.Site.Ext, err = json.Marshal(&siteExt)
			rpReq.Site.Publisher = &openrtb.Publisher{}
			rpReq.Site.Publisher.Ext, err = json.Marshal(&pubExt)
		}
		if rpReq.App != nil {
			rpReq.App.Ext, err = json.Marshal(&siteExt)
			rpReq.App.Publisher = &openrtb.Publisher{}
			rpReq.App.Publisher.Ext, err = json.Marshal(&pubExt)
		}
	}

	reqJSON, err := json.Marshal(rpReq)
	if err != nil {
		return nil, err
	}

	debug := &pbs.BidderDebug{
		RequestURI: a.URI,
	}

	if req.IsDebug {
		debug.RequestBody = string(reqJSON)
		bidder.Debug = append(bidder.Debug, debug)
	}

	httpReq, err := http.NewRequest("POST", a.URI, bytes.NewBuffer(reqJSON))
	httpReq.Header.Add("Content-Type", "application/json;charset=utf-8")
	httpReq.Header.Add("Accept", "application/json")
	httpReq.Header.Add("User-Agent", "prebid-server/1.0")
	httpReq.SetBasicAuth(a.XAPIUsername, a.XAPIPassword)
	// todo: add basic auth

	anResp, err := ctxhttp.Do(ctx, a.http.Client, httpReq)
	if err != nil {
		return nil, err
	}

	defer anResp.Body.Close()
	body, _ := ioutil.ReadAll(anResp.Body)
	responseBody := string(body)

	debug.StatusCode = anResp.StatusCode

	if anResp.StatusCode == 204 {
		return nil, nil
	}

	if anResp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP status %d; body: %s", anResp.StatusCode, responseBody)
	}

	if req.IsDebug {
		debug.ResponseBody = responseBody
	}

	var bidResp openrtb.BidResponse
	err = json.Unmarshal(body, &bidResp)
	if err != nil {
		return nil, err
	}

	bids := make(pbs.PBSBidSlice, 0)

	numBids := 0
	for _, sb := range bidResp.SeatBid {
		for _, bid := range sb.Bid {
			numBids++

			bidID := bidder.LookupBidID(bid.ImpID)
			if bidID == "" {
				return nil, fmt.Errorf("Unknown ad unit code '%s'", bid.ImpID)
			}

			pbid := pbs.PBSBid{
				BidID:       bidID,
				AdUnitCode:  bid.ImpID,
				BidderCode:  bidder.BidderCode,
				Price:       bid.Price,
				Adm:         bid.AdM,
				Creative_id: bid.CrID,
				Width:       bid.W,
				Height:      bid.H,
				DealId:      bid.DealID,
			}
			bids = append(bids, &pbid)
		}
	}

	return bids, nil
}

func NewRubiconAdapter(config *HTTPAdapterConfig, uri string, xuser string, xpass string, usersyncURL string) *RubiconAdapter {
	a := NewHTTPAdapter(config)

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
