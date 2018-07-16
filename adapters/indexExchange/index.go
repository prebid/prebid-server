package indexExchange

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/prebid/prebid-server/pbs"

	"golang.org/x/net/context/ctxhttp"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
)

type IndexAdapter struct {
	http *adapters.HTTPAdapter
	URI  string
}

// used for cookies and such
func (a *IndexAdapter) Name() string {
	return "indexExchange"
}

func (a *IndexAdapter) SkipNoCookies() bool {
	return false
}

type indexParams struct {
	SiteID int `json:"siteID"`
}

func (a *IndexAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	if req.App != nil {
		return nil, &adapters.BadInputError{
			Message: "Index doesn't support apps",
		}
	}
	mediaTypes := []pbs.MediaType{pbs.MEDIA_TYPE_BANNER, pbs.MEDIA_TYPE_VIDEO}
	indexReq, err := adapters.MakeOpenRTBGeneric(req, bidder, a.Name(), mediaTypes)

	if err != nil {
		return nil, err
	}

	for i, unit := range bidder.AdUnits {
		var params indexParams
		err := json.Unmarshal(unit.Params, &params)
		if err != nil {
			return nil, &adapters.BadInputError{
				Message: fmt.Sprintf("unmarshal params '%s' failed: %v", unit.Params, err),
			}
		}
		if params.SiteID == 0 {
			return nil, &adapters.BadInputError{
				Message: "Missing siteID param",
			}
		}

		// Fixes some segfaults. Since this is legacy code, I'm not looking into it too deeply
		if len(indexReq.Imp) <= i {
			break
		}

		indexReq.Imp[i].TagID = unit.Code
		// Index spec says "adunit path representing ad server inventory" but we don't have this
		// ext is DFP div ID and KV pairs if avail
		//indexReq.Imp[i].Ext = openrtb.RawJSON("{}")
		siteCopy := *indexReq.Site
		siteCopy.Publisher = &openrtb.Publisher{ID: fmt.Sprintf("%d", params.SiteID)}
		indexReq.Site = &siteCopy
	}
	// spec also asks for publisher id if set
	// ext object on request for prefetch

	j, _ := json.Marshal(indexReq)

	debug := &pbs.BidderDebug{
		RequestURI: a.URI,
	}

	if req.IsDebug {
		debug.RequestBody = string(j)
		bidder.Debug = append(bidder.Debug, debug)
	}

	httpReq, err := http.NewRequest("POST", a.URI, bytes.NewBuffer(j))
	httpReq.Header.Add("Content-Type", "application/json;charset=utf-8")
	httpReq.Header.Add("Accept", "application/json")

	ixResp, err := ctxhttp.Do(ctx, a.http.Client, httpReq)
	if err != nil {
		return nil, err
	}

	debug.StatusCode = ixResp.StatusCode

	if ixResp.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if ixResp.StatusCode == http.StatusBadRequest {
		return nil, &adapters.BadInputError{
			Message: fmt.Sprintf("HTTP status: %d", ixResp.StatusCode),
		}
	}

	if ixResp.StatusCode != http.StatusOK {
		return nil, &adapters.BadServerResponseError{
			Message: fmt.Sprintf("HTTP status: %d", ixResp.StatusCode),
		}
	}

	defer ixResp.Body.Close()
	body, err := ioutil.ReadAll(ixResp.Body)
	if err != nil {
		return nil, err
	}

	if req.IsDebug {
		debug.ResponseBody = string(body)
	}

	var bidResp openrtb.BidResponse
	err = json.Unmarshal(body, &bidResp)
	if err != nil {
		return nil, &adapters.BadServerResponseError{
			Message: fmt.Sprintf("Error parsing response: %v", err),
		}
	}

	bids := make(pbs.PBSBidSlice, 0)

	numBids := 0
	for _, sb := range bidResp.SeatBid {
		for i, bid := range sb.Bid {
			numBids++

			bidID := bidder.LookupBidID(bid.ImpID)
			if bidID == "" {
				return nil, &adapters.BadServerResponseError{
					Message: fmt.Sprintf("Unknown ad unit code '%s'", bid.ImpID),
				}
			}

			pbid := pbs.PBSBid{
				BidID:       bidID,
				AdUnitCode:  bidder.AdUnits[i].Code, // todo: check this
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

func NewIndexAdapter(config *adapters.HTTPAdapterConfig, uri string) *IndexAdapter {
	a := adapters.NewHTTPAdapter(config)
	return &IndexAdapter{
		http: a,
		URI:  uri,
	}
}
