package admixer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/pbs"
	"golang.org/x/net/context/ctxhttp"
)

type AdmixerAdapter struct {
	http         *adapters.HTTPAdapter
	URI          string
	usersyncInfo *pbs.UsersyncInfo
}

/* Name - export adapter name */
func (a *AdmixerAdapter) Name() string {
	return "admixer"
}

// used for cookies and such
func (a *AdmixerAdapter) FamilyName() string {
	return "am-uid"
}

func (a *AdmixerAdapter) GetUsersyncInfo() *pbs.UsersyncInfo {
	return a.usersyncInfo
}

func (a *AdmixerAdapter) SkipNoCookies() bool {
	return false
}

// parameters for Lifestreet adapter.
type admixerParams struct {
	ZoneOId string `json:"zoneOId"`
}

func (a *AdmixerAdapter) callOne(ctx context.Context, req *pbs.PBSRequest, reqJSON bytes.Buffer) (result adapters.CallOneResult, err error) {
	httpReq, err := http.NewRequest("POST", a.URI, &reqJSON)
	httpReq.Header.Add("Content-Type", "application/json;charset=utf-8")
	httpReq.Header.Add("Accept", "application/json")

	admixerResp, e := ctxhttp.Do(ctx, a.http.Client, httpReq)
	if e != nil {
		err = e
		return
	}

	defer admixerResp.Body.Close()
	body, _ := ioutil.ReadAll(admixerResp.Body)
	result.ResponseBody = string(body)

	result.StatusCode = admixerResp.StatusCode

	if admixerResp.StatusCode == 204 {
		return
	}

	if admixerResp.StatusCode != 200 {
		err = fmt.Errorf("HTTP status %d; body: %s", admixerResp.StatusCode, result.ResponseBody)
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

	result.Bid = &pbs.PBSBid{
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

func (a *AdmixerAdapter) MakeOpenRtbBidRequest(req *pbs.PBSRequest, bidder *pbs.PBSBidder, zoneOId string, mtype pbs.MediaType, unitInd int) (openrtb.BidRequest, error) {
	admixerRq, err := adapters.MakeOpenRTBGeneric(req, bidder, a.FamilyName(), []pbs.MediaType{mtype}, true)

	if err != nil {
		return openrtb.BidRequest{}, err
	}

	if admixerRq.Imp != nil && len(admixerRq.Imp) > 0 {
		admixerRq.Imp = admixerRq.Imp[unitInd : unitInd+1]

		if admixerRq.Imp[0].Banner != nil {
			admixerRq.Imp[0].Banner.Format = nil
		}
		admixerRq.Imp[0].TagID = zoneOId

		return admixerRq, nil
	} else {
		return admixerRq, errors.New("No supported impressions")
	}
}

func (a *AdmixerAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	requests := make([]bytes.Buffer, len(bidder.AdUnits)*2)
	reqIndex := 0
	for i, unit := range bidder.AdUnits {
		var params admixerParams
		err := json.Unmarshal(unit.Params, &params)
		if err != nil {
			return nil, err
		}
		if params.ZoneOId == "" {
			return nil, errors.New("Missing slot_tag param")
		}

		// BANNER
		lsReqB, err := a.MakeOpenRtbBidRequest(req, bidder, params.ZoneOId, pbs.MEDIA_TYPE_BANNER, i)
		if err == nil {
			err = json.NewEncoder(&requests[reqIndex]).Encode(lsReqB)
			reqIndex = reqIndex + 1
			if err != nil {
				return nil, err
			}
		}

		// VIDEO
		lsReqV, err := a.MakeOpenRtbBidRequest(req, bidder, params.ZoneOId, pbs.MEDIA_TYPE_VIDEO, i)
		if err == nil {
			err = json.NewEncoder(&requests[reqIndex]).Encode(lsReqV)
			reqIndex = reqIndex + 1
			if err != nil {
				return nil, err
			}
		}
	}

	ch := make(chan adapters.CallOneResult)
	for i, _ := range bidder.AdUnits {
		go func(bidder *pbs.PBSBidder, reqJSON bytes.Buffer) {
			result, err := a.callOne(ctx, req, reqJSON)
			result.Error = err
			if result.Bid != nil {
				result.Bid.BidderCode = bidder.BidderCode
				result.Bid.BidID = bidder.LookupBidID(result.Bid.AdUnitCode)
				if result.Bid.BidID == "" {
					result.Error = fmt.Errorf("Unknown zone code '%s'", result.Bid.AdUnitCode)
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
			err = result.Error
		}
	}

	if len(bids) == 0 {
		return nil, err
	}
	return bids, nil
}

func NewAdmixerAdapter(config *adapters.HTTPAdapterConfig, externalURL string) *AdmixerAdapter {
	a := adapters.NewHTTPAdapter(config)

	redirect_uri := fmt.Sprintf("%s/setuid?bidder=am-uid&uid=$$visitor_cookie$$", externalURL)
	usersyncURL := "//inv-nets.admixer.net/cm.aspx?ssp=prebid&rurl="

	info := &pbs.UsersyncInfo{
		URL:         fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirect_uri)),
		Type:        "redirect",
		SupportCORS: false,
	}

	return &AdmixerAdapter{
		http:         a,
		URI:          "http://adx.admixer.net/prebidserv.aspx",
		usersyncInfo: info,
	}
}
