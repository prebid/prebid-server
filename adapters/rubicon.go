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

// Conversion map from dimensions to internal identifiers
var rubiconSizeMap = map[string]int{
	"468x60":    1,
	"728x90":    2,
	"120x600":   8,
	"160x600":   9,
	"300x600":   10,
	"300x250":   15,
	"336x280":   16,
	"300x100":   19,
	"980x120":   31,
	"250x360":   32,
	"180x500":   33,
	"980x150":   35,
	"468x400":   37,
	"930x180":   38,
	"320x50":    43,
	"300x50":    44,
	"300x300":   48,
	"300x1050":  54,
	"970x90":    55,
	"970x250":   57,
	"1000x90":   58,
	"320x80":    59,
	"1000x1000": 61,
	"640x480":   65,
	"320x480":   67,
	"1800x1000": 68,
	"320x320":   72,
	"320x160":   73,
	"980x240":   78,
	"980x300":   79,
	"980x400":   80,
	"480x300":   83,
	"970x310":   94,
	"970x210":   96,
	"480x320":   101,
	"768x1024":  102,
	"480x280":   103,
	"1000x300":  113,
	"320x100":   117,
	"800x250":   125,
	"200x600":   126,
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

		primarySizeID := int(0)
		altSizeIDs := []int{}

		// Convert inbound dimensions to internal identifiers
		switch len(unit.Sizes) {
		case 0:
		default:
			primarySizeID = rubiconSizeMap[fmt.Sprintf("%dx%d", unit.Sizes[0].W, unit.Sizes[0].H)]
			fallthrough
		case 1:
			var extraSizes = unit.Sizes[1:]
			for _, size := range extraSizes {
				altSizeIDs = append(altSizeIDs, rubiconSizeMap[fmt.Sprintf("%dx%d", size.W, size.H)])
			}
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

	debug.StatusCode = anResp.StatusCode

	if anResp.StatusCode == 204 {
		return nil, nil
	}

	if anResp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP status code %d", anResp.StatusCode)
	}

	defer anResp.Body.Close()
	body, err := ioutil.ReadAll(anResp.Body)
	if err != nil {
		return nil, err
	}

	if req.IsDebug {
		debug.ResponseBody = string(body)
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
