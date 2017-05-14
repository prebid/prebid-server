package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/prebid/prebid-server/pbs"
	"io/ioutil"
	"net/http"

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
	AccountId int   `json:"accountId"`
	SiteId    int   `json:"siteId"`
	ZoneId    int   `json:"zoneId"`
	Sizes     []int `json:"sizes"`
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
	SizeID int    `json:"size_id"`
	MIME   string `json:"mime"`
}

type rubiconBannerExt struct {
	RP rubiconBannerExtRP `json:"rp"`
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
		if len(params.Sizes) > 0 {
			bannerExt := rubiconBannerExt{RP: rubiconBannerExtRP{SizeID: params.Sizes[0], MIME: "text/html"}}
			rpReq.Imp[i].Banner.Ext, err = json.Marshal(&bannerExt)
			rpReq.Imp[i].Banner.Format = nil
			rpReq.Imp[i].Banner.W = 0
			rpReq.Imp[i].Banner.H = 0
		}
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
		return nil, errors.New(fmt.Sprintf("HTTP status code %d", anResp.StatusCode))
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
				return nil, errors.New(fmt.Sprintf("Unknown ad unit code '%s'", bid.ImpID))
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
