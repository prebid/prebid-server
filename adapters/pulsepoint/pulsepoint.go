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

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
	"golang.org/x/net/context/ctxhttp"
)

type PulsePointAdapter struct {
	http *adapters.HTTPAdapter
	URI  string
}

// Builds an instance of PulsePointAdapter
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &PulsePointAdapter{
		URI: config.Endpoint,
	}
	return bidder, nil
}

func (a *PulsePointAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	errs := make([]error, 0, len(request.Imp))

	var err error
	pubID := ""
	imps := make([]openrtb2.Imp, 0, len(request.Imp))
	for i := 0; i < len(request.Imp); i++ {
		imp := request.Imp[i]
		var bidderExt adapters.ExtImpBidder
		if err = json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}
		var pulsepointExt openrtb_ext.ExtImpPulsePoint
		if err = json.Unmarshal(bidderExt.Bidder, &pulsepointExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}
		// parse pubid and keep it for reference
		if pubID == "" && pulsepointExt.PubID > 0 {
			pubID = strconv.Itoa(pulsepointExt.PubID)
		}
		// tag id to be sent
		imp.TagID = strconv.Itoa(pulsepointExt.TagID)
		imps = append(imps, imp)
	}

	// verify there are valid impressions
	if len(imps) == 0 {
		return nil, errs
	}

	// add the publisher id from ext to the site.pub.id or app.pub.id
	if request.Site != nil {
		site := *request.Site
		if site.Publisher != nil {
			publisher := *site.Publisher
			publisher.ID = pubID
			site.Publisher = &publisher
		} else {
			site.Publisher = &openrtb2.Publisher{ID: pubID}
		}
		request.Site = &site
	} else if request.App != nil {
		app := *request.App
		if app.Publisher != nil {
			publisher := *app.Publisher
			publisher.ID = pubID
			app.Publisher = &publisher
		} else {
			app.Publisher = &openrtb2.Publisher{ID: pubID}
		}
		request.App = &app
	}

	request.Imp = imps
	reqJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     a.URI,
		Body:    reqJSON,
		Headers: headers,
	}}, errs
}

func (a *PulsePointAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	// passback
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	// bad requests
	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Bad user input: HTTP status %d", response.StatusCode),
		}}
	}
	// error
	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Bad server response: HTTP status %d", response.StatusCode),
		}}
	}
	// parse response
	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)
	// map imps by id
	impsByID := make(map[string]openrtb2.Imp)
	for i := 0; i < len(internalRequest.Imp); i++ {
		impsByID[internalRequest.Imp[i].ID] = internalRequest.Imp[i]
	}

	var errs []error
	for _, sb := range bidResp.SeatBid {
		for i := 0; i < len(sb.Bid); i++ {
			bid := sb.Bid[i]
			imp := impsByID[bid.ImpID]
			bidType := getBidType(imp)
			if &imp != nil && bidType != "" {
				bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
					Bid:     &bid,
					BidType: bidType,
				})
			}
		}
	}
	return bidResponse, errs
}

func getBidType(imp openrtb2.Imp) openrtb_ext.BidType {
	// derive the bidtype purely from the impression itself
	if imp.Banner != nil {
		return openrtb_ext.BidTypeBanner
	} else if imp.Video != nil {
		return openrtb_ext.BidTypeVideo
	} else if imp.Audio != nil {
		return openrtb_ext.BidTypeAudio
	} else if imp.Native != nil {
		return openrtb_ext.BidTypeNative
	}
	return ""
}

/////////////////////////////////
// Legacy implementation: Start
/////////////////////////////////

func NewPulsePointLegacyAdapter(config *adapters.HTTPAdapterConfig, uri string) *PulsePointAdapter {
	a := adapters.NewHTTPAdapter(config)

	return &PulsePointAdapter{
		http: a,
		URI:  uri,
	}
}

// used for cookies and such
func (a *PulsePointAdapter) Name() string {
	return "pulsepoint"
}

// parameters for pulsepoint adapter.
type PulsepointParams struct {
	PublisherId int    `json:"cp"`
	TagId       int    `json:"ct"`
	AdSize      string `json:"cf"`
}

func (a *PulsePointAdapter) SkipNoCookies() bool {
	return false
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
		publisher := &openrtb2.Publisher{ID: strconv.Itoa(params.PublisherId)}
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
					ppReq.Imp[i].Banner.W = openrtb2.Int64Ptr(int64(width))
				} else {
					return nil, &errortypes.BadInput{
						Message: fmt.Sprintf("Invalid Width param %s", size[0]),
					}
				}
				height, err := strconv.Atoi(size[1])
				if err == nil {
					ppReq.Imp[i].Banner.H = openrtb2.Int64Ptr(int64(height))
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

	var bidResp openrtb2.BidResponse
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

/////////////////////////////////
// Legacy implementation: End
/////////////////////////////////
