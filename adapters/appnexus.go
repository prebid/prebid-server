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

type AppNexusAdapter struct {
	http         *HTTPAdapter
	URI          string
	usersyncInfo *pbs.UsersyncInfo
}

/* Name - export adapter name */
func (a *AppNexusAdapter) Name() string {
	return "AppNexus"
}

// used for cookies and such
func (a *AppNexusAdapter) FamilyName() string {
	return "adnxs"
}

func (a *AppNexusAdapter) GetUsersyncInfo() *pbs.UsersyncInfo {
	return a.usersyncInfo
}

type appnexusParams struct {
	PlacementId string `json:"placementId"`
	invCode     string `json:"invCode"`
	member      string `json:"member"`
}

func (a *AppNexusAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	anReq := makeOpenRTBGeneric(req, bidder, a.FamilyName())
	for i, unit := range bidder.AdUnits {
		var params appnexusParams
		err := json.Unmarshal(unit.Params, &params)
		if err != nil {
			return nil, err
		}
		if params.PlacementId == "" {
			return nil, errors.New("Missing placementId param")
		}
		anReq.Imp[i].TagID = params.PlacementId
		// TODO: support member + invCode
	}

	reqJSON, err := json.Marshal(anReq)
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

func NewAppNexusAdapter(config *HTTPAdapterConfig, externalURL string) *AppNexusAdapter {
	a := NewHTTPAdapter(config)

	redirect_uri := fmt.Sprintf("%s/setuid?bidder=adnxs&uid=$UID", externalURL)
	usersyncURL := "https://ib.adnxs.com/getuid?"

	info := &pbs.UsersyncInfo{
		URL:         fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirect_uri)),
		Type:        "redirect",
		SupportCORS: false,
	}

	return &AppNexusAdapter{
		http: a,
		// TODO: Get new endpoint from sweeney
		URI:          "http://ib.adnxs.com/openrtb2?member_id=958",
		usersyncInfo: info,
	}
}
