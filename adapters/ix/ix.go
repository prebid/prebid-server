package ix

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/prebid/prebid-server/pbs"

	"golang.org/x/net/context/ctxhttp"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
)

// maximum number of bid requests
const requestLimit = 20

type IxAdapter struct {
	http *adapters.HTTPAdapter
	URI  string
}

// Name is used for cookies and such
func (a *IxAdapter) Name() string {
	return string(openrtb_ext.BidderIx)
}

func (a *IxAdapter) SkipNoCookies() bool {
	return false
}

type indexParams struct {
	SiteID string `json:"siteId"`
}

type ixBidResult struct {
	Request      *callOneObject
	StatusCode   int
	ResponseBody string
	Bid          *pbs.PBSBid
	Error        error
}

type callOneObject struct {
	requestJSON bytes.Buffer
	width       uint64
	height      uint64
}

func isValidIXSize(f openrtb.Format, s [2]uint64) bool {
	if f.W != s[0] || f.H != s[1] {
		return false
	}
	return true
}

func (a *IxAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	var prioritizedRequests, requests []callOneObject

	if req.App != nil {
		return nil, &errortypes.BadInput{
			Message: "Index doesn't support apps",
		}
	}
	mediaTypes := []pbs.MediaType{pbs.MEDIA_TYPE_BANNER}
	indexReq, err := adapters.MakeOpenRTBGeneric(req, bidder, a.Name(), mediaTypes)

	if err != nil {
		return nil, err
	}

	indexReqImp := indexReq.Imp
	for i, unit := range bidder.AdUnits {

		// Fixes some segfaults. Since this is legacy code, I'm not looking into it too deeply
		if len(indexReqImp) <= i {
			break
		}

		var params indexParams
		err := json.Unmarshal(unit.Params, &params)
		if err != nil {
			return nil, &errortypes.BadInput{
				Message: fmt.Sprintf("unmarshal params '%s' failed: %v", unit.Params, err),
			}
		}

		if params.SiteID == "" {
			return nil, &errortypes.BadInput{
				Message: "Missing siteId param",
			}
		}

		for sizeIndex, format := range unit.Sizes {
			// Only grab this ad unit
			// Not supporting multi-media-type adunit yet
			thisImp := indexReqImp[i]

			thisImp.TagID = unit.Code
			thisImp.Banner.Format = []openrtb.Format{format}
			thisImp.Banner.W = &format.W
			thisImp.Banner.H = &format.H
			indexReq.Imp = []openrtb.Imp{thisImp}
			// Index spec says "adunit path representing ad server inventory" but we don't have this
			// ext is DFP div ID and KV pairs if avail
			//indexReq.Imp[i].Ext = json.RawMessage("{}")

			// Any objects pointed to by indexReq *must not be mutated*, or we will get race conditions.
			siteCopy := *indexReq.Site
			siteCopy.Publisher = &openrtb.Publisher{ID: params.SiteID}
			indexReq.Site = &siteCopy

			// spec also asks for publisher id if set
			// ext object on request for prefetch
			j, _ := json.Marshal(indexReq)

			request := callOneObject{requestJSON: *bytes.NewBuffer(j), width: format.W, height: format.H}

			// prioritize slots over sizes
			if sizeIndex == 0 {
				prioritizedRequests = append(prioritizedRequests, request)
			} else {
				requests = append(requests, request)
			}
		}
	}

	// cap the number of requests to requestLimit
	requests = append(prioritizedRequests, requests...)
	if len(requests) > requestLimit {
		requests = requests[:requestLimit]
	}

	if len(requests) == 0 {
		return nil, &errortypes.BadInput{
			Message: "Invalid ad unit/imp/size",
		}
	}

	ch := make(chan ixBidResult)
	for _, request := range requests {
		go func(bidder *pbs.PBSBidder, request callOneObject) {
			result, err := a.callOne(ctx, request.requestJSON)
			result.Request = &request
			result.Error = err
			if result.Bid != nil {
				result.Bid.BidderCode = bidder.BidderCode
				result.Bid.BidID = bidder.LookupBidID(result.Bid.AdUnitCode)
				result.Bid.Width = request.width
				result.Bid.Height = request.height

				if result.Bid.BidID == "" {
					result.Error = &errortypes.BadServerResponse{
						Message: fmt.Sprintf("Unknown ad unit code '%s'", result.Bid.AdUnitCode),
					}
					result.Bid = nil
				}
			}
			ch <- result
		}(bidder, request)
	}

	bids := make(pbs.PBSBidSlice, 0)
	for i := 0; i < len(requests); i++ {
		result := <-ch
		if result.Bid != nil && result.Bid.Price != 0 {
			bids = append(bids, result.Bid)
		}

		if req.IsDebug {
			debug := &pbs.BidderDebug{
				RequestURI:   a.URI,
				RequestBody:  result.Request.requestJSON.String(),
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

func (a *IxAdapter) callOne(ctx context.Context, reqJSON bytes.Buffer) (ixBidResult, error) {
	var result ixBidResult

	httpReq, _ := http.NewRequest("POST", a.URI, &reqJSON)
	httpReq.Header.Add("Content-Type", "application/json;charset=utf-8")
	httpReq.Header.Add("Accept", "application/json")

	ixResp, err := ctxhttp.Do(ctx, a.http.Client, httpReq)
	if err != nil {
		return result, err
	}

	result.StatusCode = ixResp.StatusCode

	if ixResp.StatusCode == http.StatusNoContent {
		return result, nil
	}

	if ixResp.StatusCode == http.StatusBadRequest {
		return result, &errortypes.BadInput{
			Message: fmt.Sprintf("HTTP status: %d", ixResp.StatusCode),
		}
	}

	if ixResp.StatusCode != http.StatusOK {
		return result, &errortypes.BadServerResponse{
			Message: fmt.Sprintf("HTTP status: %d", ixResp.StatusCode),
		}
	}

	defer ixResp.Body.Close()
	body, err := ioutil.ReadAll(ixResp.Body)
	if err != nil {
		return result, err
	}
	result.ResponseBody = string(body)

	var bidResp openrtb.BidResponse
	err = json.Unmarshal(body, &bidResp)
	if err != nil {
		return result, &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Error parsing response: %v", err),
		}
	}

	if len(bidResp.SeatBid) == 0 {
		return result, nil
	}
	if len(bidResp.SeatBid[0].Bid) == 0 {
		return result, nil
	}
	bid := bidResp.SeatBid[0].Bid[0]

	result.Bid = &pbs.PBSBid{
		AdUnitCode:        bid.ImpID,
		Price:             bid.Price,
		Adm:               bid.AdM,
		Creative_id:       bid.CrID,
		Width:             bid.W,
		Height:            bid.H,
		DealId:            bid.DealID,
		CreativeMediaType: string(openrtb_ext.BidTypeBanner),
	}
	return result, nil
}

func NewIxAdapter(config *adapters.HTTPAdapterConfig, uri string) *IxAdapter {
	a := adapters.NewHTTPAdapter(config)
	return &IxAdapter{
		http: a,
		URI:  uri,
	}
}
