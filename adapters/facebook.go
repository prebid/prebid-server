package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/prebid/openrtb"
	"github.com/prebid/prebid-server/pbs"
	"golang.org/x/net/context/ctxhttp"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type FacebookAdapter struct {
	http         *HTTPAdapter
	URI          string
	usersyncInfo *pbs.UsersyncInfo
	platformJSON openrtb.RawJSON
}

/* Name - export adapter name */
func (a *FacebookAdapter) Name() string {
	return "audienceNetwork"
}

// used for cookies and such
func (a *FacebookAdapter) FamilyName() string {
	return "audienceNetwork"
}

// Facebook likes to parallelize to minimize latency
func (a *FacebookAdapter) SplitAdUnits() bool {
	return true
}

func (a *FacebookAdapter) GetUsersyncInfo() *pbs.UsersyncInfo {
	return a.usersyncInfo
}

type facebookParams struct {
	PlacementId string `json:"placementId"`
}

type fbResult struct {
	statusCode   int
	responseBody string
	bid          *pbs.PBSBid
	Error        error
}

func (a *FacebookAdapter) CallOne(ctx context.Context, req *pbs.PBSRequest, reqJSON bytes.Buffer) (result fbResult, err error) {
	httpReq, _ := http.NewRequest("POST", a.URI, &reqJSON)
	httpReq.Header.Add("Content-Type", "application/json")
	httpReq.Header.Add("Accept", "application/json")

	anResp, e := ctxhttp.Do(ctx, a.http.Client, httpReq)
	if e != nil {
		err = e
		return
	}

	result.statusCode = anResp.StatusCode

	if anResp.StatusCode != 200 {
		defer anResp.Body.Close()
		body, _ := ioutil.ReadAll(anResp.Body)
		err = errors.New(fmt.Sprintf("HTTP status %d; body: %s", anResp.StatusCode, string(body)))
		return
	}

	defer anResp.Body.Close()
	body, _ := ioutil.ReadAll(anResp.Body)

	result.responseBody = string(body)

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

	result.bid = &pbs.PBSBid{
		AdUnitCode: bid.ImpID,
		Price:      bid.Price / 100, // convert from cents to dollars
		Adm:        bid.AdM,
		Width:      300, // hard code as it's all FB supports
		Height:     250, // hard code as it's all FB supports
	}
	return
}

func (a *FacebookAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	requests := make([]bytes.Buffer, len(bidder.AdUnits))
	for i, unit := range bidder.AdUnits {
		fbReq := makeOpenRTBGeneric(req, bidder, a.FamilyName())
		fbReq.Ext = a.platformJSON

		// only grab this ad unit
		fbReq.Imp = fbReq.Imp[i : i+1]

		var params facebookParams
		err := json.Unmarshal(unit.Params, &params)
		if err != nil {
			return nil, err
		}
		if params.PlacementId == "" {
			return nil, errors.New("Missing placementId param")
		}
		s := strings.Split(params.PlacementId, "_")
		if len(s) != 2 {
			return nil, fmt.Errorf("Invalid placementId param '%s'", params.PlacementId)
		}
		fbReq.Site.Publisher = &openrtb.Publisher{ID: s[0]}
		fbReq.Imp[0].TagID = params.PlacementId

		err = json.NewEncoder(&requests[i]).Encode(fbReq)
		if err != nil {
			return nil, err
		}
	}

	ch := make(chan fbResult)
	for i, _ := range bidder.AdUnits {
		go func(bidder *pbs.PBSBidder, reqJSON bytes.Buffer) {
			result, err := a.CallOne(ctx, req, reqJSON)
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

	if err == nil || len(bids) > 0 {
		return bids, nil
	} else {
		return nil, err
	}
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
		URI:          "https://an.facebook.com/placementbid.ortb",
		usersyncInfo: info,
		platformJSON: openrtb.RawJSON(fmt.Sprintf("{\"platformid\": %s}", partnerID)),
	}
}
