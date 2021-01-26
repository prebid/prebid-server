package lifestreet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/pbs"
	"golang.org/x/net/context/ctxhttp"
)

type LifestreetAdapter struct {
	http *adapters.HTTPAdapter
	URI  string
}

// used for cookies and such
func (a *LifestreetAdapter) Name() string {
	return "lifestreet"
}

func (a *LifestreetAdapter) SkipNoCookies() bool {
	return false
}

// parameters for Lifestreet adapter.
type lifestreetParams struct {
	SlotTag string `json:"slot_tag"`
}

func (a *LifestreetAdapter) callOne(ctx context.Context, req *pbs.PBSRequest, reqJSON bytes.Buffer) (result adapters.CallOneResult, err error) {
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

	t := openrtb_ext.BidTypeBanner

	if bid.Ext != nil {
		var e openrtb_ext.ExtBid
		err = json.Unmarshal(bid.Ext, &e)
		if err != nil {
			return
		}
		t = e.Prebid.Type
	}

	result.Bid = &pbs.PBSBid{
		AdUnitCode:        bid.ImpID,
		Price:             bid.Price,
		Adm:               bid.AdM,
		Creative_id:       bid.CrID,
		Width:             bid.W,
		Height:            bid.H,
		DealId:            bid.DealID,
		NURL:              bid.NURL,
		CreativeMediaType: string(t),
	}
	return
}

func (a *LifestreetAdapter) MakeOpenRtbBidRequest(req *pbs.PBSRequest, bidder *pbs.PBSBidder, slotTag string, mtype pbs.MediaType, unitInd int) (openrtb.BidRequest, error) {
	lsReq, err := adapters.MakeOpenRTBGeneric(req, bidder, a.Name(), []pbs.MediaType{mtype})

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
		return lsReq, &errortypes.BadInput{
			Message: "No supported impressions",
		}
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
			return nil, &errortypes.BadInput{
				Message: "Missing slot_tag param",
			}
		}
		s := strings.Split(params.SlotTag, ".")
		if len(s) != 2 {
			return nil, &errortypes.BadInput{
				Message: fmt.Sprintf("Invalid slot_tag param '%s'", params.SlotTag),
			}
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

	ch := make(chan adapters.CallOneResult)
	for i := range bidder.AdUnits {
		go func(bidder *pbs.PBSBidder, reqJSON bytes.Buffer) {
			result, err := a.callOne(ctx, req, reqJSON)
			result.Error = err
			if result.Bid != nil {
				result.Bid.BidderCode = bidder.BidderCode
				result.Bid.BidID = bidder.LookupBidID(result.Bid.AdUnitCode)
				if result.Bid.BidID == "" {
					result.Error = &errortypes.BadServerResponse{
						Message: fmt.Sprintf("Unknown ad unit code '%s'", result.Bid.AdUnitCode),
					}
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

func NewLifestreetLegacyAdapter(config *adapters.HTTPAdapterConfig, endpoint string) *LifestreetAdapter {
	a := adapters.NewHTTPAdapter(config)
	return &LifestreetAdapter{
		http: a,
		URI:  endpoint,
	}
}
