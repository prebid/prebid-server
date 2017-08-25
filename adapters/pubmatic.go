package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/prebid/openrtb"
	"github.com/prebid/prebid-server/pbs"
	"golang.org/x/net/context/ctxhttp"
)

type PubmaticAdapter struct {
	http         *HTTPAdapter
	URI          string
	usersyncInfo *pbs.UsersyncInfo
}

/* Name - export adapter name */
func (a *PubmaticAdapter) Name() string {
	return "Pubmatic"
}

// used for cookies and such
func (a *PubmaticAdapter) FamilyName() string {
	return "pubmatic"
}

func (a *PubmaticAdapter) GetUsersyncInfo() *pbs.UsersyncInfo {
	return a.usersyncInfo
}

func (a *PubmaticAdapter) SkipNoCookies() bool {
	return false
}

type pubmaticParams struct {
	PublisherId string `json:"publisherId"`
	AdSlot      string `json:"adSlot"`
}

func (a *PubmaticAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	mediaTypes := []pbs.MediaType{pbs.MEDIA_TYPE_BANNER, pbs.MEDIA_TYPE_VIDEO}
	pbReq, err := makeOpenRTBGeneric(req, bidder, a.FamilyName(), mediaTypes, true)

	if err != nil {
		return nil, err
	}

	for i, unit := range bidder.AdUnits {
		var params pubmaticParams
		err := json.Unmarshal(unit.Params, &params)
		if err != nil {
			return nil, err
		}
		if params.PublisherId == "" {
			return nil, errors.New("Missing publisherId param")
		}
		if params.AdSlot == "" {
			return nil, errors.New("Missing adSlot param")
		}
		if pbReq.Imp[i].Banner != nil {
			pbReq.Imp[i].Banner.Format = nil
		} // pubmatic doesn't support
		pbReq.Imp[i].TagID = params.AdSlot
		if pbReq.Site != nil {
			pbReq.Site.Publisher = &openrtb.Publisher{ID: params.PublisherId}
		}
		if pbReq.App != nil {
			pbReq.App.Publisher = &openrtb.Publisher{ID: params.PublisherId}
		}
	}

	reqJSON, err := json.Marshal(pbReq)

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
	httpReq.AddCookie(&http.Cookie{
		Name:  "KADUSERCOOKIE",
		Value: req.GetUserID(a.FamilyName()),
	})

	pbResp, err := ctxhttp.Do(ctx, a.http.Client, httpReq)
	if err != nil {
		return nil, err
	}

	debug.StatusCode = pbResp.StatusCode

	if pbResp.StatusCode == 204 {
		return nil, nil
	}

	if pbResp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP status: %d", pbResp.StatusCode)
	}

	defer pbResp.Body.Close()
	body, err := ioutil.ReadAll(pbResp.Body)
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

func NewPubmaticAdapter(config *HTTPAdapterConfig, uri string, externalURL string) *PubmaticAdapter {
	a := NewHTTPAdapter(config)
	redirect_uri := fmt.Sprintf("%s/setuid?bidder=pubmatic&uid=", externalURL)
	usersyncURL := "//ads.pubmatic.com/AdServer/js/user_sync.html?predirect="

	info := &pbs.UsersyncInfo{
		URL:         fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirect_uri)),
		Type:        "iframe",
		SupportCORS: false,
	}

	return &PubmaticAdapter{
		http:         a,
		URI:          uri,
		usersyncInfo: info,
	}
}
