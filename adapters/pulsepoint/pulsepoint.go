package pulsepoint

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
	"golang.org/x/net/context/ctxhttp"
)

type PulsePointAdapter struct {
	http *adapters.HTTPAdapter
	URI  string
}

// used for cookies and such
func (a *PulsePointAdapter) Name() string {
	return "pulsepoint"
}

func (a *PulsePointAdapter) SkipNoCookies() bool {
	return false
}

// parameters for pulsepoint adapter.
type PulsepointParams struct {
	PublisherId int    `json:"cp"`
	TagId       int    `json:"ct"`
	AdSize      string `json:"cf"`
}

func (a *PulsePointAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	mediaTypes := []pbs.MediaType{pbs.MEDIA_TYPE_BANNER}
	ppReq, err := adapters.MakeOpenRTBGeneric(req, bidder, a.Name(), mediaTypes)

	if err != nil {
		return nil, err
	}

	for i, unit := range bidder.AdUnits {
		var params PulsepointParams
		err := json.Unmarshal(unit.Params, &params)
		if err != nil {
			return nil, &errortypes.BadInput{
				Message: err.Error(),
			}
		}
		if params.PublisherId == 0 {
			return nil, &errortypes.BadInput{
				Message: "Missing PublisherId param cp",
			}
		}
		if params.TagId == 0 {
			return nil, &errortypes.BadInput{
				Message: "Missing TagId param ct",
			}
		}
		if params.AdSize == "" {
			return nil, &errortypes.BadInput{
				Message: "Missing AdSize param cf",
			}
		}
		// Fixes some segfaults. Since this is legacy code, I'm not looking into it too deeply
		if len(ppReq.Imp) <= i {
			break
		}
		ppReq.Imp[i].TagID = strconv.Itoa(params.TagId)
		publisher := &openrtb.Publisher{ID: strconv.Itoa(params.PublisherId)}
		if ppReq.Site != nil {
			siteCopy := *ppReq.Site
			siteCopy.Publisher = publisher
			ppReq.Site = &siteCopy
		} else {
			appCopy := *ppReq.App
			appCopy.Publisher = publisher
			ppReq.App = &appCopy
		}
		if ppReq.Imp[i].Banner != nil {
			var size = strings.Split(strings.ToLower(params.AdSize), "x")
			if len(size) == 2 {
				width, err := strconv.Atoi(size[0])
				if err == nil {
					ppReq.Imp[i].Banner.W = openrtb.Uint64Ptr(uint64(width))
				} else {
					return nil, &errortypes.BadInput{
						Message: fmt.Sprintf("Invalid Width param %s", size[0]),
					}
				}
				height, err := strconv.Atoi(size[1])
				if err == nil {
					ppReq.Imp[i].Banner.H = openrtb.Uint64Ptr(uint64(height))
				} else {
					return nil, &errortypes.BadInput{
						Message: fmt.Sprintf("Invalid Height param %s", size[1]),
					}
				}
			} else {
				return nil, &errortypes.BadInput{
					Message: fmt.Sprintf("Invalid AdSize param %s", params.AdSize),
				}
			}
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

	httpReq, err := http.NewRequest("POST", a.URI, bytes.NewBuffer(reqJSON))
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

func NewPulsePointAdapter(config *adapters.HTTPAdapterConfig, uri string) *PulsePointAdapter {
	a := adapters.NewHTTPAdapter(config)

	return &PulsePointAdapter{
		http: a,
		URI:  uri,
	}
}
