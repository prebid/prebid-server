package somoaudience

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/pbs"
	"golang.org/x/net/context/ctxhttp"
)

type SomoaudienceAdapter struct {
	http *adapters.HTTPAdapter
	URI  string
}

/* Name - export adapter name */
func (a *SomoaudienceAdapter) Name() string {
	return "Somoaudience"
}

// used for cookies and such
func (a *SomoaudienceAdapter) FamilyName() string {
	return "somoaudience"
}

func (a *SomoaudienceAdapter) SkipNoCookies() bool {
	return false
}

// parameters for Somoaudience adapter.
type somoaudienceParams struct {
	PlacementHash string `json:"placement_hash"`
}

func (a *SomoaudienceAdapter) callOne(ctx context.Context, req *pbs.PBSRequest, reqJSON bytes.Buffer, placementhash string) (result adapters.CallOneResult, err error) {
	httpReq, err := http.NewRequest("POST", a.URI+placementhash, &reqJSON)
	httpReq.Header.Add("Content-Type", "application/json;charset=utf-8")
	httpReq.Header.Add("Accept", "application/json")

	lsmResp, e := ctxhttp.Do(ctx, a.http.Client, httpReq)
	if e != nil {
		err = e
		return
	}

	defer lsmResp.Body.Close()
	body, _ := ioutil.ReadAll(lsmResp.Body)
	result.ResponseBody = string(body)

	result.StatusCode = lsmResp.StatusCode

	if lsmResp.StatusCode == 204 {
		return
	}

	if lsmResp.StatusCode != 200 {
		err = fmt.Errorf("HTTP status %d; body: %s", lsmResp.StatusCode, result.ResponseBody)
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

func (a *SomoaudienceAdapter) MakeOpenRtbBidRequest(req *pbs.PBSRequest, bidder *pbs.PBSBidder, mtype pbs.MediaType, unitInd int) (openrtb.BidRequest, error) {
	lsReq, err := adapters.MakeOpenRTBGeneric(req, bidder, a.FamilyName(), []pbs.MediaType{mtype}, true)

	if err != nil {
		return openrtb.BidRequest{}, err
	}

	if lsReq.Imp != nil && len(lsReq.Imp) > 0 {
		lsReq.Imp = lsReq.Imp[unitInd : unitInd+1]

		if lsReq.Imp[0].Banner != nil {
			lsReq.Imp[0].Banner.Format = nil
		}

		return lsReq, nil
	} else {
		return lsReq, errors.New("No supported impressions")
	}
}

func (a *SomoaudienceAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	requests := make([]bytes.Buffer, len(bidder.AdUnits)*2)
	reqIndex := 0
	for i, unit := range bidder.AdUnits {
		var params somoaudienceParams
		err := json.Unmarshal(unit.Params, &params)
		if err != nil {
			return nil, err
		}

		// BANNER
		lsReqB, err := a.MakeOpenRtbBidRequest(req, bidder, pbs.MEDIA_TYPE_BANNER, i)
		if err == nil {
			err = json.NewEncoder(&requests[reqIndex]).Encode(lsReqB)
			reqIndex = reqIndex + 1
			if err != nil {
				return nil, err
			}
		}

		// VIDEO
		lsReqV, err := a.MakeOpenRtbBidRequest(req, bidder, pbs.MEDIA_TYPE_VIDEO, i)
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
		var params somoaudienceParams

		if params.PlacementHash == "" {
			return nil, errors.New("Missing placement hash param")
		}

		if len(params.PlacementHash) != 32 {
			return nil, fmt.Errorf("Invalid placement hash param '%s'", params.PlacementHash)
		}

		go func(bidder *pbs.PBSBidder, reqJSON bytes.Buffer) {
			result, err := a.callOne(ctx, req, reqJSON, params.PlacementHash)
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

func NewSomoaudienceAdapter(config *adapters.HTTPAdapterConfig) *SomoaudienceAdapter {
	a := adapters.NewHTTPAdapter(config)
	return &SomoaudienceAdapter{
		http: a,
		URI:  "https://publisher-east.somoaudience.com/rtb/bid?s=",
	}
}
