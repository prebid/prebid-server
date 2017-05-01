package adapters

import (
	//	"bytes"
	"context"
	//	"encoding/json"
	"errors"
	"fmt"
	//	"io/ioutil"
	//	"net/http"
	"github.com/prebid/prebid-server/pbs"
	"net/url"
	//	"golang.org/x/net/context/ctxhttp"
	//	"github.com/prebid/openrtb"
)

type FacebookAdapter struct {
	http         *HTTPAdapter
	URI          string
	usersyncInfo *pbs.UsersyncInfo
}

/* Name - export adapter name */
func (a *FacebookAdapter) Name() string {
	return "audienceNetwork"
}

// used for cookies and such
func (a *FacebookAdapter) FamilyName() string {
	return "audienceNetwork"
}

func (a *FacebookAdapter) GetUsersyncInfo() *pbs.UsersyncInfo {
	return a.usersyncInfo
}

type facebookParams struct {
}

func (a *FacebookAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	/*
		anReq := makeOpenRTBGeneric(req, bidder, a.FamilyName())
		for i, unit := range bidder.AdUnits {
			var params facebookParams
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

		if req.IsDebug {
			bidder.Debug.RequestURI = a.URI
			bidder.Debug.RequestBody = string(reqJSON)
		}

		httpReq, err := http.NewRequest("POST", a.URI, bytes.NewBuffer(reqJSON))
		httpReq.Header.Add("Content-Type", "application/json;charset=utf-8")
		httpReq.Header.Add("Accept", "application/json")

		anResp, err := ctxhttp.Do(ctx, a.http.Client, httpReq)
		if err != nil {
			return nil, err
		}

		if req.IsDebug {
			bidder.Debug.StatusCode = anResp.StatusCode
		}

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
			bidder.Debug.ResponseBody = string(body)
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
	*/
	return nil, errors.New("Not implemented")
}

func NewFacebookAdapter(config *HTTPAdapterConfig, partnerID string, externalURL string) *FacebookAdapter {
	a := NewHTTPAdapter(config)

	redirect_uri := fmt.Sprintf("%s/setuid?bidder=audienceNetwork&uid=$UID", externalURL)
	usersyncURL := fmt.Sprintf("https://www.facebook.com/audiencenetwork/idsync/?partner=%s&callback=", partnerID)
	info := &pbs.UsersyncInfo{
		URL:         fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirect_uri)),
		Type:        "redirect",
		SupportCORS: false,
	}

	return &FacebookAdapter{
		http:         a,
		URI:          "http://ib.adnxs.com/openrtb2?member_id=958",
		usersyncInfo: info,
	}
}
