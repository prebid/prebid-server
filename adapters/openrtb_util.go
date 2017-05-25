package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"golang.org/x/net/context/ctxhttp"

	"github.com/prebid/openrtb"
	"github.com/prebid/prebid-server/pbs"
)

func makeOpenRTBGeneric(req *pbs.PBSRequest, bidder *pbs.PBSBidder, bidderFamily string) openrtb.BidRequest {

	imps := make([]openrtb.Imp, len(bidder.AdUnits))
	for i, unit := range bidder.AdUnits {
		if len(unit.Sizes) <= 0 {
			continue
		}

		imps[i] = openrtb.Imp{
			ID: unit.Code,
			Banner: &openrtb.Banner{
				W:        unit.Sizes[0].W,
				H:        unit.Sizes[0].H,
				Format:   unit.Sizes,
				TopFrame: unit.TopFrame,
			},
			Secure: req.Secure,
			// pmp
			// ext
		}
	}

	if req.App != nil {
		return openrtb.BidRequest{
			ID:     req.Tid,
			Imp:    imps,
			App:    req.App,
			Device: req.Device,
			Source: &openrtb.Source{
				TID: req.Tid,
			},
			AT:   1,
			TMax: req.TimeoutMillis,
		}
	}

	return openrtb.BidRequest{
		ID:  req.Tid,
		Imp: imps,
		Site: &openrtb.Site{
			Domain: req.Domain,
			Page:   req.Url,
		},
		Device: req.Device,
		User: &openrtb.User{
			BuyerUID: req.GetUserID(bidderFamily),
			ID:       req.GetUserID("adnxs"),
		},
		Source: &openrtb.Source{
			FD:  1, // upstream, aka header
			TID: req.Tid,
		},
		AT:   1,
		TMax: req.TimeoutMillis,
	}
}

// DefaultOpenRTBResponse will perform the http request, parse the bytes and unmarshal into a openrtb.BidResponse struct and then return a PBSBidSlice
var DefaultOpenRTBResponse = func(ctx context.Context, httpAdapter *HTTPAdapter, httpReq *http.Request, bidder *pbs.PBSBidder, debug *pbs.BidderDebug) (pbs.PBSBidSlice, error) {

	httpResp, err := ctxhttp.Do(ctx, httpAdapter.Client, httpReq)
	if err != nil {
		return nil, err
	}

	if debug != nil {
		debug.StatusCode = httpResp.StatusCode
	}

	if httpResp.StatusCode == 204 {
		return nil, nil
	}

	if httpResp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP status code %d", httpResp.StatusCode)
	}

	defer httpResp.Body.Close()
	body, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}

	if debug != nil {
		debug.ResponseBody = string(body)
	}

	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(body, &bidResp); err != nil {
		return nil, err
	}

	bids := make(pbs.PBSBidSlice, 0)

	for _, sb := range bidResp.SeatBid {
		for _, bid := range sb.Bid {
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
			}
			bids = append(bids, &pbid)
		}
	}

	return bids, nil
}
