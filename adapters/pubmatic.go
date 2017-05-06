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
	"net/url"

	"golang.org/x/net/context/ctxhttp"

	"github.com/prebid/openrtb"
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

type pubmaticParams struct {
	PublisherId string `json:"publisherId"`
	AdSlot      string `json:"adSlot"`
}

func (a *PubmaticAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	pbReq := makeOpenRTBGeneric(req, bidder, a.FamilyName())
	pbReq.AT = 0 // this seems to break their bidder otherwise
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
		pbReq.Imp[i].Banner.Format = nil // pubmatic doesn't support
		pbReq.Imp[i].TagID = params.AdSlot
		pbReq.Site.Publisher = &openrtb.Publisher{ID: params.PublisherId}
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
		return nil, errors.New(fmt.Sprintf("HTTP status: %d", pbResp.StatusCode))
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

	redirect_uri := fmt.Sprintf("%s/setuid?bidder=pubmatic&uid=$UID", externalURL)
	usersyncURL := "http://ads.pubmatic.com/AdServer/js/user_sync.html?p=31445&s=21446&predirect="

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
