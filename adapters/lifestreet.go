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
	"strings"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/pbs"
	"golang.org/x/net/context/ctxhttp"
)

type LifestreetAdapter struct {
	http         *HTTPAdapter
	URI          string
	usersyncInfo *pbs.UsersyncInfo
}

/* Name - export adapter name */
func (a *LifestreetAdapter) Name() string {
	return "Lifestreet"
}

// used for cookies and such
func (a *LifestreetAdapter) FamilyName() string {
	return "lifestreet"
}

func (a *LifestreetAdapter) GetUsersyncInfo() *pbs.UsersyncInfo {
	return a.usersyncInfo
}

func (a *LifestreetAdapter) SkipNoCookies() bool {
	return false
}

// parameters for Lifestreet adapter.
type lifestreetParams struct {
	SlotTag string `json:"slot_tag"`
}

func (a *LifestreetAdapter) callOne(ctx context.Context, req *pbs.PBSRequest, reqJSON bytes.Buffer) (result callOneResult, err error) {
	httpReq, err := http.NewRequest("POST", a.URI, &reqJSON)
	httpReq.Header.Add("Content-Type", "application/json;charset=utf-8")
	httpReq.Header.Add("Accept", "application/json")

	lsmResp, e := ctxhttp.Do(ctx, a.http.Client, httpReq)
	if e != nil {
		err = e
		return
	}

	defer lsmResp.Body.Close()
	body, _ := ioutil.ReadAll(lsmResp.Body)
	result.responseBody = string(body)

	result.statusCode = lsmResp.StatusCode

	if lsmResp.StatusCode == 204 {
		return
	}

	if lsmResp.StatusCode != 200 {
		err = fmt.Errorf("HTTP status %d; body: %s", lsmResp.StatusCode, result.responseBody)
		return
	}

	var bidResp openrtb.BidResponse
	err = json.Unmarshal(body, &bidResp)
	if err != nil {
		return
	}
	if len(bidResp.SeatBid) == 0 || len(bidResp.SeatBid[0].Bid) == 0 {
		return
	}
	bid := bidResp.SeatBid[0].Bid[0]

	result.bid = &pbs.PBSBid{
		AdUnitCode:  bid.ImpID,
		Price:       bid.Price,
		Adm:         bid.AdM,
		Creative_id: bid.CrID,
		Width:       bid.W,
		Height:      bid.H,
		DealId:      bid.DealID,
		NURL:        bid.NURL,
	}
	return
}

func (a *LifestreetAdapter) MakeOpenRtbBidRequest(req *pbs.PBSRequest, bidder *pbs.PBSBidder, slotTag string, mtype pbs.MediaType, unitInd int) (openrtb.BidRequest, error) {
	lsReq, err := makeOpenRTBGeneric(req, bidder, a.FamilyName(), []pbs.MediaType{mtype}, true)

	if err != nil {
		return openrtb.BidRequest{}, err
	}

	if lsReq.Imp != nil && len(lsReq.Imp) > 0 {
		lsReq.Imp = lsReq.Imp[unitInd : unitInd+1]

		if lsReq.Imp[0].Banner != nil {
			lsReq.Imp[0].Banner.Format = nil
		}
		lsReq.Imp[0].TagID = slotTag

		return lsReq, nil
	} else {
		return lsReq, errors.New("No supported impressions")
	}
}

func (a *LifestreetAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	requests := make([]bytes.Buffer, len(bidder.AdUnits)*2)
	reqIndex := 0
	for i, unit := range bidder.AdUnits {
		var params lifestreetParams
		err := json.Unmarshal(unit.Params, &params)
		if err != nil {
			return nil, err
		}
		if params.SlotTag == "" {
			return nil, errors.New("Missing slot_tag param")
		}
		s := strings.Split(params.SlotTag, ".")
		if len(s) != 2 {
			return nil, fmt.Errorf("Invalid slot_tag param '%s'", params.SlotTag)
		}

		// BANNER
		lsReqB, err := a.MakeOpenRtbBidRequest(req, bidder, params.SlotTag, pbs.MEDIA_TYPE_BANNER, i)
		if err == nil {
			err = json.NewEncoder(&requests[reqIndex]).Encode(lsReqB)
			reqIndex = reqIndex + 1
			if err != nil {
				return nil, err
			}
		}

		// VIDEO
		lsReqV, err := a.MakeOpenRtbBidRequest(req, bidder, params.SlotTag, pbs.MEDIA_TYPE_VIDEO, i)
		if err == nil {
			err = json.NewEncoder(&requests[reqIndex]).Encode(lsReqV)
			reqIndex = reqIndex + 1
			if err != nil {
				return nil, err
			}
		}
	}

	ch := make(chan callOneResult)
	for i, _ := range bidder.AdUnits {
		go func(bidder *pbs.PBSBidder, reqJSON bytes.Buffer) {
			result, err := a.callOne(ctx, req, reqJSON)
			result.Error = err
			if result.bid != nil {
				result.bid.BidderCode = bidder.BidderCode
				result.bid.BidID = bidder.LookupBidID(result.bid.AdUnitCode)
				if result.bid.BidID == "" {
					result.Error = fmt.Errorf("Unknown ad unit code '%s'", result.bid.AdUnitCode)
					result.bid = nil
				}
			}
			ch <- result
		}(bidder, requests[i])
	}

	var err error

	bids := make(pbs.PBSBidSlice, 0)
	for i := 0; i < len(bidder.AdUnits); i++ {
		result := <-ch
		if result.bid != nil {
			bids = append(bids, result.bid)
		}
		if req.IsDebug {
			debug := &pbs.BidderDebug{
				RequestURI:   a.URI,
				RequestBody:  requests[i].String(),
				StatusCode:   result.statusCode,
				ResponseBody: result.responseBody,
			}
			bidder.Debug = append(bidder.Debug, debug)
		}
		if result.Error != nil {
			err = result.Error
		}
	}

	if len(bids) == 0 {
		return nil, err
	}
	return bids, nil
}

func NewLifestreetAdapter(config *HTTPAdapterConfig, externalURL string) *LifestreetAdapter {
	a := NewHTTPAdapter(config)

	redirect_uri := fmt.Sprintf("%s/setuid?bidder=lifestreet&uid=$$visitor_cookie$$", externalURL)
	usersyncURL := "//ads.lfstmedia.com/idsync/137062?synced=1&ttl=1s&rurl="

	info := &pbs.UsersyncInfo{
		URL:         fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirect_uri)),
		Type:        "redirect",
		SupportCORS: false,
	}

	return &LifestreetAdapter{
		http:         a,
		URI:          "https://prebid.s2s.lfstmedia.com/adrequest",
		usersyncInfo: info,
	}
}
