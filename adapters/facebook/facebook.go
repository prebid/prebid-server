package facebook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/pbs"
	"golang.org/x/net/context/ctxhttp"
	"github.com/prebid/prebid-server/adapters"
)

type FacebookAdapter struct {
	http         *adapters.HTTPAdapter
	URI          string
	nonSecureUri string
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

func (a *FacebookAdapter) SkipNoCookies() bool {
	return false
}

type facebookParams struct {
	PlacementId string `json:"placementId"`
}

func coinFlip() bool {
	return rand.Intn(2) != 0
}

func (a *FacebookAdapter) callOne(ctx context.Context, req *pbs.PBSRequest, reqJSON bytes.Buffer) (result adapters.CallOneResult, err error) {
	url := a.URI
	if coinFlip() {
		//50% of traffic to non-secure endpoint
		url = a.nonSecureUri
	}
	httpReq, _ := http.NewRequest("POST", url, &reqJSON)
	httpReq.Header.Add("Content-Type", "application/json")
	httpReq.Header.Add("Accept", "application/json")

	anResp, e := ctxhttp.Do(ctx, a.http.Client, httpReq)
	if e != nil {
		err = e
		return
	}

	result.StatusCode = anResp.StatusCode

	defer anResp.Body.Close()
	body, _ := ioutil.ReadAll(anResp.Body)
	result.ResponseBody = string(body)

	if anResp.StatusCode != 200 {
		err = fmt.Errorf("HTTP status %d; body: %s", anResp.StatusCode, result.ResponseBody)
		return
	}

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

	result.Bid = &pbs.PBSBid{
		AdUnitCode: bid.ImpID,
		Price:      bid.Price,
		Adm:        bid.AdM,
		Width:      300, // hard code as it's all FB supports
		Height:     250, // hard code as it's all FB supports
	}
	return
}

func (a *FacebookAdapter) MakeOpenRtbBidRequest(req *pbs.PBSRequest, bidder *pbs.PBSBidder, placementId string, mtype pbs.MediaType, pubId string, unitInd int) (openrtb.BidRequest, error) {
	fbReq, err := adapters.MakeOpenRTBGeneric(req, bidder, a.FamilyName(), []pbs.MediaType{mtype}, true)

	if err != nil {
		return openrtb.BidRequest{}, err
	}

	fbReq.Ext = a.platformJSON

	if fbReq.Imp != nil && len(fbReq.Imp) > 0 {
		fbReq.Imp = fbReq.Imp[unitInd : unitInd+1]

		if fbReq.Site != nil {
			siteCopy := *fbReq.Site
			siteCopy.Publisher = &openrtb.Publisher{ID: pubId}
			fbReq.Site = &siteCopy
		}
		if fbReq.App != nil {
			appCopy := *fbReq.App
			appCopy.Publisher = &openrtb.Publisher{ID: pubId}
			fbReq.App = &appCopy
		}
		fbReq.Imp[0].TagID = placementId

		return fbReq, nil
	} else {
		return fbReq, errors.New("No supported impressions")
	}
}

func (a *FacebookAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	requests := make([]bytes.Buffer, len(bidder.AdUnits)*2) // potentially we can for eachadUnit have 2 imps - BANNER and VIDEO
	reqIndex := 0
	for i, unit := range bidder.AdUnits {
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
		pubId := s[0]

		// BANNER
		fbReqB, err := a.MakeOpenRtbBidRequest(req, bidder, params.PlacementId, pbs.MEDIA_TYPE_BANNER, pubId, i)
		if err == nil {
			err = json.NewEncoder(&requests[reqIndex]).Encode(fbReqB)
			reqIndex = reqIndex + 1
			if err != nil {
				return nil, err
			}
		}

		// VIDEO
		fbReqV, err := a.MakeOpenRtbBidRequest(req, bidder, params.PlacementId, pbs.MEDIA_TYPE_VIDEO, pubId, i)
		if err == nil {
			err = json.NewEncoder(&requests[reqIndex]).Encode(fbReqV)
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
					result.Error = fmt.Errorf("Unknown ad unit code '%s'", result.Bid.AdUnitCode)
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

func NewFacebookAdapter(config *adapters.HTTPAdapterConfig, partnerID string, usersyncURL string) *FacebookAdapter {
	a := adapters.NewHTTPAdapter(config)

	info := &pbs.UsersyncInfo{
		URL:         usersyncURL,
		Type:        "redirect",
		SupportCORS: false,
	}

	return &FacebookAdapter{
		http: a,
		URI:  "https://an.facebook.com/placementbid.ortb",
		//for AB test
		nonSecureUri: "http://an.facebook.com/placementbid.ortb",
		usersyncInfo: info,
		platformJSON: openrtb.RawJSON(fmt.Sprintf("{\"platformid\": %s}", partnerID)),
	}
}
