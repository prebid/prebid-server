package conversant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/pbs"
	"golang.org/x/net/context/ctxhttp"
)

type ConversantLegacyAdapter struct {
	http *adapters.HTTPAdapter
	URI  string
}

// Corresponds to the bidder name in cookies and requests
func (a *ConversantLegacyAdapter) Name() string {
	return "conversant"
}

// Return true so no request will be sent unless user has been sync'ed.
func (a *ConversantLegacyAdapter) SkipNoCookies() bool {
	return true
}

type conversantParams struct {
	SiteID      string   `json:"site_id"`
	Secure      *int8    `json:"secure"`
	TagID       string   `json:"tag_id"`
	Position    *int8    `json:"position"`
	BidFloor    float64  `json:"bidfloor"`
	Mobile      *int8    `json:"mobile"`
	MIMEs       []string `json:"mimes"`
	API         []int8   `json:"api"`
	Protocols   []int8   `json:"protocols"`
	MaxDuration *int64   `json:"maxduration"`
}

func (a *ConversantLegacyAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	mediaTypes := []pbs.MediaType{pbs.MEDIA_TYPE_BANNER, pbs.MEDIA_TYPE_VIDEO}
	cnvrReq, err := adapters.MakeOpenRTBGeneric(req, bidder, a.Name(), mediaTypes)

	if err != nil {
		return nil, err
	}

	// Create a map of impression objects for both request creation
	// and response parsing.

	impMap := make(map[string]*openrtb2.Imp, len(cnvrReq.Imp))
	for idx := range cnvrReq.Imp {
		impMap[cnvrReq.Imp[idx].ID] = &cnvrReq.Imp[idx]
	}

	// Fill in additional info from custom params

	for _, unit := range bidder.AdUnits {
		var params conversantParams

		imp := impMap[unit.Code]
		if imp == nil {
			// Skip ad units that do not have corresponding impressions.
			continue
		}

		err := json.Unmarshal(unit.Params, &params)
		if err != nil {
			return nil, &errortypes.BadInput{
				Message: err.Error(),
			}
		}

		// Fill in additional Site info
		if params.SiteID != "" {
			if cnvrReq.Site != nil {
				cnvrReq.Site.ID = params.SiteID
			}
			if cnvrReq.App != nil {
				cnvrReq.App.ID = params.SiteID
			}
		}

		if params.Mobile != nil && !(cnvrReq.Site == nil) {
			cnvrReq.Site.Mobile = *params.Mobile
		}

		// Fill in additional impression info

		imp.DisplayManager = "prebid-s2s"
		imp.DisplayManagerVer = "1.0.1"
		imp.BidFloor = params.BidFloor
		imp.TagID = params.TagID

		var position *openrtb2.AdPosition
		if params.Position != nil {
			position = openrtb2.AdPosition(*params.Position).Ptr()
		}

		if imp.Banner != nil {
			imp.Banner.Pos = position
		} else if imp.Video != nil {
			imp.Video.Pos = position

			if len(params.API) > 0 {
				imp.Video.API = make([]openrtb2.APIFramework, 0, len(params.API))
				for _, api := range params.API {
					imp.Video.API = append(imp.Video.API, openrtb2.APIFramework(api))
				}
			}

			// Include protocols, mimes, and max duration if specified
			// These properties can also be specified in ad unit's video object,
			// but are overridden if the custom params object also contains them.

			if len(params.Protocols) > 0 {
				imp.Video.Protocols = make([]openrtb2.Protocol, 0, len(params.Protocols))
				for _, protocol := range params.Protocols {
					imp.Video.Protocols = append(imp.Video.Protocols, openrtb2.Protocol(protocol))
				}
			}

			if len(params.MIMEs) > 0 {
				imp.Video.MIMEs = make([]string, len(params.MIMEs))
				copy(imp.Video.MIMEs, params.MIMEs)
			}

			if params.MaxDuration != nil {
				imp.Video.MaxDuration = *params.MaxDuration
			}
		}

		// Take care not to override the global secure flag

		if (imp.Secure == nil || *imp.Secure == 0) && params.Secure != nil {
			imp.Secure = params.Secure
		}
	}

	// Do a quick check on required parameters

	if cnvrReq.Site != nil && cnvrReq.Site.ID == "" {
		return nil, &errortypes.BadInput{
			Message: "Missing site id",
		}
	}

	if cnvrReq.App != nil && cnvrReq.App.ID == "" {
		return nil, &errortypes.BadInput{
			Message: "Missing app id",
		}
	}

	// Start capturing debug info

	debug := &pbs.BidderDebug{
		RequestURI: a.URI,
	}

	if cnvrReq.Device == nil {
		cnvrReq.Device = &openrtb2.Device{}
	}

	// Convert request to json to be sent over http

	j, _ := json.Marshal(cnvrReq)

	if req.IsDebug {
		debug.RequestBody = string(j)
		bidder.Debug = append(bidder.Debug, debug)
	}

	httpReq, err := http.NewRequest("POST", a.URI, bytes.NewBuffer(j))
	httpReq.Header.Add("Content-Type", "application/json")
	httpReq.Header.Add("Accept", "application/json")

	resp, err := ctxhttp.Do(ctx, a.http.Client, httpReq)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if req.IsDebug {
		debug.StatusCode = resp.StatusCode
	}

	if resp.StatusCode == 204 {
		return nil, nil
	}

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusBadRequest {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("HTTP status: %d, body: %s", resp.StatusCode, string(body)),
		}
	}

	if resp.StatusCode != 200 {
		return nil, &errortypes.BadServerResponse{
			Message: fmt.Sprintf("HTTP status: %d, body: %s", resp.StatusCode, string(body)),
		}
	}

	if req.IsDebug {
		debug.ResponseBody = string(body)
	}

	var bidResp openrtb2.BidResponse

	err = json.Unmarshal(body, &bidResp)
	if err != nil {
		return nil, &errortypes.BadServerResponse{
			Message: err.Error(),
		}
	}

	bids := make(pbs.PBSBidSlice, 0)

	for _, seatbid := range bidResp.SeatBid {
		for _, bid := range seatbid.Bid {
			if bid.Price <= 0 {
				continue
			}

			imp := impMap[bid.ImpID]
			if imp == nil {
				// All returned bids should have a matching impression
				return nil, &errortypes.BadServerResponse{
					Message: fmt.Sprintf("Unknown impression id '%s'", bid.ImpID),
				}
			}

			bidID := bidder.LookupBidID(bid.ImpID)
			if bidID == "" {
				return nil, &errortypes.BadServerResponse{
					Message: fmt.Sprintf("Unknown ad unit code '%s'", bid.ImpID),
				}
			}

			pbsBid := pbs.PBSBid{
				BidID:       bidID,
				AdUnitCode:  bid.ImpID,
				Price:       bid.Price,
				Creative_id: bid.CrID,
				BidderCode:  bidder.BidderCode,
			}

			if imp.Video != nil {
				pbsBid.CreativeMediaType = "video"
				pbsBid.NURL = bid.AdM // Assign to NURL so it'll be interpreted as a vastUrl
				pbsBid.Width = imp.Video.W
				pbsBid.Height = imp.Video.H
			} else {
				pbsBid.CreativeMediaType = "banner"
				pbsBid.NURL = bid.NURL
				pbsBid.Adm = bid.AdM
				pbsBid.Width = bid.W
				pbsBid.Height = bid.H
			}

			bids = append(bids, &pbsBid)
		}
	}

	if len(bids) == 0 {
		return nil, nil
	}

	return bids, nil
}

func NewConversantLegacyAdapter(config *adapters.HTTPAdapterConfig, uri string) *ConversantLegacyAdapter {
	a := adapters.NewHTTPAdapter(config)

	return &ConversantLegacyAdapter{
		http: a,
		URI:  uri,
	}
}
