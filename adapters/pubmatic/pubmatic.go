package pubmatic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
	"golang.org/x/net/context/ctxhttp"
)

const MAX_IMPRESSIONS_PUBMATIC = 30

type PubmaticAdapter struct {
	http *adapters.HTTPAdapter
	URI  string
}

// used for cookies and such
func (a *PubmaticAdapter) Name() string {
	return "pubmatic"
}

func (a *PubmaticAdapter) SkipNoCookies() bool {
	return false
}

type pubmaticParams struct {
	PublisherId string `json:"publisherId"`
	AdSlot      string `json:"adSlot"`
}

func PrepareLogMessage(tID, pubId, adUnitId, bidID, details string, args ...interface{}) string {
	return fmt.Sprintf("[PUBMATIC] ReqID [%s] PubID [%s] AdUnit [%s] BidID [%s] %s \n",
		tID, pubId, adUnitId, bidID, details)
}

func (a *PubmaticAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	mediaTypes := []pbs.MediaType{pbs.MEDIA_TYPE_BANNER, pbs.MEDIA_TYPE_VIDEO}
	pbReq, err := adapters.MakeOpenRTBGeneric(req, bidder, a.Name(), mediaTypes)

	if err != nil {
		logf("[PUBMATIC] Failed to make ortb request for request id [%s] \n", pbReq.ID)
		return nil, err
	}

	adSlotFlag := false
	pubId := ""
	if len(bidder.AdUnits) > MAX_IMPRESSIONS_PUBMATIC {
		logf("[PUBMATIC] First %d impressions will be considered from request tid %s\n",
			MAX_IMPRESSIONS_PUBMATIC, pbReq.ID)
	}

	for i, unit := range bidder.AdUnits {
		var params pubmaticParams
		err := json.Unmarshal(unit.Params, &params)
		if err != nil {
			logf(PrepareLogMessage(pbReq.ID, params.PublisherId, unit.Code, unit.BidID,
				fmt.Sprintf("Ignored bid: invalid JSON  [%s] err [%s]", unit.Params, err.Error())))
			continue
		}

		if params.PublisherId == "" {
			logf(PrepareLogMessage(pbReq.ID, params.PublisherId, unit.Code, unit.BidID,
				fmt.Sprintf("Ignored bid: Publisher Id missing")))
			continue
		}
		pubId = params.PublisherId
		if params.AdSlot == "" {
			logf(PrepareLogMessage(pbReq.ID, params.PublisherId, unit.Code, unit.BidID,
				fmt.Sprintf("Ignored bid: adSlot missing")))
			continue
		}

		adSlotStr := strings.TrimSpace(params.AdSlot)
		adSlot := strings.Split(adSlotStr, "@")
		if len(adSlot) == 2 && adSlot[0] != "" && adSlot[1] != "" {
			// Fixes some segfaults. Since this is legacy code, I'm not looking into it too deeply
			if len(pbReq.Imp) <= i {
				break
			}
			if pbReq.Imp[i].Banner != nil {
				pbReq.Imp[i].Banner.Format = nil // pubmatic doesn't support
				adSize := strings.Split(strings.ToLower(strings.TrimSpace(adSlot[1])), "x")
				if len(adSize) == 2 {
					width, err := strconv.Atoi(strings.TrimSpace(adSize[0]))
					if err != nil {
						logf(PrepareLogMessage(pbReq.ID, params.PublisherId, unit.Code, unit.BidID,
							fmt.Sprintf("Ignored bid: invalid adSlot width [%s]", adSize[0])))
						continue
					}

					heightStr := strings.Split(strings.TrimSpace(adSize[1]), ":")
					height, err := strconv.Atoi(strings.TrimSpace(heightStr[0]))
					if err != nil {
						logf(PrepareLogMessage(pbReq.ID, params.PublisherId, unit.Code, unit.BidID,
							fmt.Sprintf("Ignored bid: invalid adSlot height [%s]", heightStr[0])))
						continue
					}

					pbReq.Imp[i].TagID = strings.TrimSpace(adSlot[0])
					pbReq.Imp[i].Banner.H = openrtb.Uint64Ptr(uint64(height))
					pbReq.Imp[i].Banner.W = openrtb.Uint64Ptr(uint64(width))
					adSlotFlag = true
				} else {
					logf(PrepareLogMessage(pbReq.ID, params.PublisherId, unit.Code, unit.BidID,
						fmt.Sprintf("Ignored bid: invalid adSize [%s]", adSize)))
					continue
				}
			}
		} else {
			logf(PrepareLogMessage(pbReq.ID, params.PublisherId, unit.Code, unit.BidID,
				fmt.Sprintf("Ignored bid: invalid adSlot [%s]", params.AdSlot)))
			continue
		}

		if pbReq.Site != nil {
			siteCopy := *pbReq.Site
			siteCopy.Publisher = &openrtb.Publisher{ID: params.PublisherId, Domain: req.Domain}
			pbReq.Site = &siteCopy
		}
		if pbReq.App != nil {
			appCopy := *pbReq.App
			appCopy.Publisher = &openrtb.Publisher{ID: params.PublisherId, Domain: req.Domain}
			pbReq.App = &appCopy
		}
	}

	if !(adSlotFlag) {
		return nil, &adapters.BadInputError{
			Message: "Incorrect adSlot / Publisher param",
		}
	}

	reqJSON, err := json.Marshal(pbReq)

	debug := &pbs.BidderDebug{
		RequestURI: a.URI,
	}

	if req.IsDebug {
		debug.RequestBody = string(reqJSON)
		bidder.Debug = append(bidder.Debug, debug)
	}

	userId, _, _ := req.Cookie.GetUID(a.Name())
	httpReq, err := http.NewRequest("POST", a.URI, bytes.NewBuffer(reqJSON))
	httpReq.Header.Add("Content-Type", "application/json;charset=utf-8")
	httpReq.Header.Add("Accept", "application/json")
	httpReq.AddCookie(&http.Cookie{
		Name:  "KADUSERCOOKIE",
		Value: userId,
	})

	pbResp, err := ctxhttp.Do(ctx, a.http.Client, httpReq)
	if err != nil {
		return nil, err
	}

	debug.StatusCode = pbResp.StatusCode

	if pbResp.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if pbResp.StatusCode == http.StatusBadRequest {
		return nil, &adapters.BadInputError{
			Message: fmt.Sprintf("HTTP status: %d", pbResp.StatusCode),
		}
	}

	if pbResp.StatusCode != http.StatusOK {
		return nil, &adapters.BadServerResponseError{
			Message: fmt.Sprintf("HTTP status: %d", pbResp.StatusCode),
		}
	}

	defer pbResp.Body.Close()
	body, err := ioutil.ReadAll(pbResp.Body)
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
			Message: fmt.Sprintf("HTTP status: %d", pbResp.StatusCode),
		}
	}

	bids := make(pbs.PBSBidSlice, 0)

	numBids := 0
	for _, sb := range bidResp.SeatBid {
		for _, bid := range sb.Bid {
			numBids++

			bidID := bidder.LookupBidID(bid.ImpID)
			if bidID == "" {
				return nil, &adapters.BadServerResponseError{
					Message: fmt.Sprintf("Unknown ad unit code '%s'", bid.ImpID),
				}
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
				DealId:      bid.DealID,
			}

			mediaType := getMediaTypeForImp(bid.ImpID, pbReq.Imp)
			pbid.CreativeMediaType = string(mediaType)

			bids = append(bids, &pbid)
			logf("[PUBMATIC] Returned Bid for PubID [%s] AdUnit [%s] BidID [%s] Size [%dx%d] Price [%f] \n",
				pubId, pbid.AdUnitCode, pbid.BidID, pbid.Width, pbid.Height, pbid.Price)
		}
	}

	return bids, nil
}

// getMediaTypeForImp figures out which media type this bid is for.
func getMediaTypeForImp(impId string, imps []openrtb.Imp) openrtb_ext.BidType {
	mediaType := openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			} else if imp.Audio != nil {
				mediaType = openrtb_ext.BidTypeAudio
			} else if imp.Native != nil {
				mediaType = openrtb_ext.BidTypeNative
			}
			return mediaType
		}
	}
	return mediaType
}

func logf(msg string, args ...interface{}) {
	if glog.V(2) {
		glog.Infof(msg, args)
	}
}

func NewPubmaticAdapter(config *adapters.HTTPAdapterConfig, uri string) *PubmaticAdapter {
	a := adapters.NewHTTPAdapter(config)

	return &PubmaticAdapter{
		http: a,
		URI:  uri,
	}
}
