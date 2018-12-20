package onemobile

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
	"golang.org/x/net/context/ctxhttp"
	"io/ioutil"
	"net/http"
)

type OneMobileAdapter struct {
	http *adapters.HTTPAdapter
	URI  string
}

// used for cookies and such
func (a *OneMobileAdapter) Name() string {
	return "onemobile"
}

func (a *OneMobileAdapter) SkipNoCookies() bool {
	return false
}

type OneMobileParams struct {
	dcn string `json:"dcn"`
	pos string `json:"pos"`
}

func (a *OneMobileAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	mediaTypes := []pbs.MediaType{pbs.MEDIA_TYPE_BANNER}
	ppReq, err := adapters.MakeOpenRTBGeneric(req, bidder, a.Name(), mediaTypes)

	if err != nil {
		return nil, err
	}

	var unit pbs.PBSAdUnit
	var params OneMobileParams

	unit = bidder.AdUnits[0]

	err = json.Unmarshal(unit.Params, &params)
	if err != nil {
		return nil, &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	if params.dcn == "" {
		return nil, &errortypes.BadInput{
			Message: "Missing param dcn",
		}
	}

	if params.pos == "" {
		return nil, &errortypes.BadInput{
			Message: "Missing param pos",
		}
	}

	reqJSON, err := json.Marshal(ppReq)
	debug := &pbs.BidderDebug{
		RequestURI: a.URI,
	}

	if req.IsDebug {
		debug.RequestBody = string(reqJSON)
		bidder.Debug = append(bidder.Debug, debug)
	}

	var endpointUrl = a.URI + "/cmd=bid&dcn" + params.dcn + "&pos=" + params.pos

	httpReq, err := http.NewRequest("GET", endpointUrl, nil)
	httpReq.Header.Add("Content-Type", "application/json;charset=utf-8")
	httpReq.Header.Add("Accept", "application/json")

	ppResp, err := ctxhttp.Do(ctx, a.http.Client, httpReq)
	if err != nil {
		return nil, err
	}

	debug.StatusCode = ppResp.StatusCode

	if ppResp.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if ppResp.StatusCode == http.StatusBadRequest {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("HTTP status: %d", ppResp.StatusCode),
		}
	}

	if ppResp.StatusCode != http.StatusOK {
		return nil, &errortypes.BadServerResponse{
			Message: fmt.Sprintf("HTTP status: %d", ppResp.StatusCode),
		}
	}

	defer ppResp.Body.Close()
	body, err := ioutil.ReadAll(ppResp.Body)
	if err != nil {
		return nil, err
	}

	if req.IsDebug {
		debug.ResponseBody = string(body)
	}

	var bidResp openrtb.BidResponse
	err = json.Unmarshal(body, &bidResp)
	if err != nil {
		return nil, &errortypes.BadServerResponse{
			Message: err.Error(),
		}
	}

	bids := make(pbs.PBSBidSlice, 0)

	for _, sb := range bidResp.SeatBid {
		for _, bid := range sb.Bid {
			bidID := bidder.LookupBidID(bid.ImpID)
			if bidID == "" {
				return nil, &errortypes.BadServerResponse{
					Message: fmt.Sprintf("Unknown ad unit code '%s'", bid.ImpID),
				}
			}

			pbid := pbs.PBSBid{
				BidID:             bidID,
				AdUnitCode:        bid.ImpID,
				BidderCode:        bidder.BidderCode,
				Price:             bid.Price,
				Adm:               bid.AdM,
				Creative_id:       bid.CrID,
				Width:             bid.W,
				Height:            bid.H,
				CreativeMediaType: string(openrtb_ext.BidTypeBanner),
			}
			bids = append(bids, &pbid)
		}
	}

	return bids, nil
}

func NewOneMobileAdapter(config *adapters.HTTPAdapterConfig, uri string) *OneMobileAdapter {
	a := adapters.NewHTTPAdapter(config)

	return &OneMobileAdapter{
		http: a,
		URI:  uri,
	}
}
