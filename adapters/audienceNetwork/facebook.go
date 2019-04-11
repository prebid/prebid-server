package audienceNetwork

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

	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/errortypes"
	"github.com/PubMatic-OpenWrap/prebid-server/pbs"
	"golang.org/x/net/context/ctxhttp"
)

type FacebookAdapter struct {
	http         *adapters.HTTPAdapter
	URI          string
	nonSecureUri string
	platformJSON json.RawMessage
}

var supportedHeight = map[uint64]bool{
	50:  true,
	90:  true,
	250: true,
}

// used for cookies and such
func (a *FacebookAdapter) Name() string {
	return "audienceNetwork"
}

// Facebook likes to parallelize to minimize latency
func (a *FacebookAdapter) SplitAdUnits() bool {
	return true
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

func (a *FacebookAdapter) callOne(ctx context.Context, reqJSON bytes.Buffer) (result adapters.CallOneResult, err error) {
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

	if anResp.StatusCode == http.StatusBadRequest {
		err = &errortypes.BadInput{
			Message: fmt.Sprintf("HTTP status %d; body: %s", anResp.StatusCode, result.ResponseBody),
		}
		return
	}

	if anResp.StatusCode == http.StatusNoContent {
		return
	}

	if anResp.StatusCode != http.StatusOK {
		err = &errortypes.BadServerResponse{
			Message: fmt.Sprintf("HTTP status %d; body: %s", anResp.StatusCode, result.ResponseBody),
		}
		return
	}

	var bidResp openrtb.BidResponse
	err = json.Unmarshal(body, &bidResp)
	if err != nil {
		err = &errortypes.BadServerResponse{
			Message: err.Error(),
		}
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
		AdUnitCode:        bid.ImpID,
		Price:             bid.Price,
		Adm:               bid.AdM,
		CreativeMediaType: "banner", //  hard code this, because that's all facebook supports now, can potentially update it dynamically from "template" field in the "adm"
	}
	return
}

func (a *FacebookAdapter) MakeOpenRtbBidRequest(req *pbs.PBSRequest, bidder *pbs.PBSBidder, placementId string, mtype pbs.MediaType, pubId string, unitInd int) (openrtb.BidRequest, error) {
	// this method creates imps for all ad units for the bidder with a single media type
	fbReq, err := adapters.MakeOpenRTBGeneric(req, bidder, a.Name(), []pbs.MediaType{mtype})

	if err != nil {
		return openrtb.BidRequest{}, err
	}

	fbReq.Ext = a.platformJSON

	if fbReq.Imp != nil && len(fbReq.Imp) > 0 {
		// only returns 1 imp for requested ad unit
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

		// if instl = 1 sent in, pass size (0,0) to facebook
		if fbReq.Imp[0].Instl == 1 && fbReq.Imp[0].Banner != nil {
			fbReq.Imp[0].Banner.W = openrtb.Uint64Ptr(0)
			fbReq.Imp[0].Banner.H = openrtb.Uint64Ptr(0)
		}
		// if instl = 0 and type is banner, do not send non supported size
		if fbReq.Imp[0].Instl == 0 && fbReq.Imp[0].Banner != nil {
			if !supportedHeight[*fbReq.Imp[0].Banner.H] {
				return fbReq, &errortypes.BadInput{
					Message: "Facebook do not support banner height other than 50, 90 and 250",
				}
			}
			// do not send legacy 320x50 size to facebook, instead use 0x50
			if *fbReq.Imp[0].Banner.W == 320 && *fbReq.Imp[0].Banner.H == 50 {
				fbReq.Imp[0].Banner.W = openrtb.Uint64Ptr(0)
			}
		}
		return fbReq, nil
	} else {
		return fbReq, &errortypes.BadInput{
			Message: "No supported impressions",
		}
	}
}

func (a *FacebookAdapter) GenerateRequestsForFacebook(req *pbs.PBSRequest, bidder *pbs.PBSBidder) ([]*openrtb.BidRequest, error) {
	requests := make([]*openrtb.BidRequest, len(bidder.AdUnits)*2) // potentially we can for eachadUnit have 2 imps - BANNER and VIDEO
	reqIndex := 0
	for i, unit := range bidder.AdUnits {
		var params facebookParams
		err := json.Unmarshal(unit.Params, &params)
		if err != nil {
			return nil, err
		}
		if params.PlacementId == "" {
			return nil, &errortypes.BadInput{
				Message: "Missing placementId param",
			}
		}
		s := strings.Split(params.PlacementId, "_")
		if len(s) != 2 {
			return nil, &errortypes.BadInput{
				Message: fmt.Sprintf("Invalid placementId param '%s'", params.PlacementId),
			}
		}
		pubId := s[0]

		// BANNER
		fbReqB, err := a.MakeOpenRtbBidRequest(req, bidder, params.PlacementId, pbs.MEDIA_TYPE_BANNER, pubId, i)
		if err == nil {
			requests[reqIndex] = &fbReqB
			reqIndex = reqIndex + 1
		}

		// VIDEO
		fbReqV, err := a.MakeOpenRtbBidRequest(req, bidder, params.PlacementId, pbs.MEDIA_TYPE_VIDEO, pubId, i)
		if err == nil {
			requests[reqIndex] = &fbReqV
			reqIndex = reqIndex + 1

		}

	}
	return requests[:reqIndex], nil
}

func (a *FacebookAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	ortbRequests, e := a.GenerateRequestsForFacebook(req, bidder)

	if e != nil {
		return nil, e
	}

	requests := make([]bytes.Buffer, len(ortbRequests))
	for i, ortbRequest := range ortbRequests {
		e = json.NewEncoder(&requests[i]).Encode(ortbRequest)
		if e != nil {
			return nil, e
		}
	}

	ch := make(chan adapters.CallOneResult)

	for i := range requests {
		go func(bidder *pbs.PBSBidder, reqJSON bytes.Buffer) {
			result, err := a.callOne(ctx, reqJSON)
			result.Error = err
			if result.Bid != nil {
				result.Bid.BidderCode = bidder.BidderCode
				unit := bidder.LookupAdUnit(result.Bid.AdUnitCode)
				if unit != nil {
					result.Bid.Width = unit.Sizes[0].W
					result.Bid.Height = unit.Sizes[0].H
				}
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
	for i := 0; i < len(requests); i++ {
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

func NewAdapterFromFacebook(config *adapters.HTTPAdapterConfig, partnerID string) adapters.Adapter {
	if partnerID == "" {
		glog.Errorf("No facebook partnerID specified. Calls to the Audience Network will fail. Did you set adapters.facebook.platform_id in the app config?")
		return &adapters.MisconfiguredAdapter{
			TheName: "audienceNetwork",
			Err:     errors.New("Audience Network is not configured properly on this Prebid Server deploy. If you believe this should work, contact the company hosting the service and tell them to check their configuration."),
		}
	}
	return NewFacebookAdapter(config, partnerID)
}

func NewFacebookAdapter(config *adapters.HTTPAdapterConfig, partnerID string) *FacebookAdapter {
	a := adapters.NewHTTPAdapter(config)

	return &FacebookAdapter{
		http: a,
		URI:  "https://an.facebook.com/placementbid.ortb",
		//for AB test
		nonSecureUri: "http://an.facebook.com/placementbid.ortb",
		platformJSON: json.RawMessage(fmt.Sprintf("{\"platformid\": %s}", partnerID)),
	}
}
