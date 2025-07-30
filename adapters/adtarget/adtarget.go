package adtarget

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type AdtargetAdapter struct {
	endpoint string
}

type adtargetImpExt struct {
	Adtarget openrtb_ext.ExtImpAdtarget `json:"adtarget"`
}

func (a *AdtargetAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {

	totalImps := len(request.Imp)
	errors := make([]error, 0, totalImps)
	imp2source := make(map[int][]int)

	for i := 0; i < totalImps; i++ {

		sourceId, err := validateImpressionAndSetExt(&request.Imp[i])

		if err != nil {
			errors = append(errors, err)
			continue
		}

		if _, ok := imp2source[sourceId]; !ok {
			imp2source[sourceId] = make([]int, 0, totalImps-i)
		}

		imp2source[sourceId] = append(imp2source[sourceId], i)

	}

	totalReqs := len(imp2source)
	if 0 == totalReqs {
		return nil, errors
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	reqs := make([]*adapters.RequestData, 0, totalReqs)

	imps := request.Imp
	request.Imp = make([]openrtb2.Imp, 0, len(imps))
	for sourceId, impIndexes := range imp2source {
		request.Imp = request.Imp[:0]

		for i := 0; i < len(impIndexes); i++ {
			request.Imp = append(request.Imp, imps[impIndexes[i]])
		}

		body, err := json.Marshal(request)
		if err != nil {
			errors = append(errors, fmt.Errorf("error while encoding bidRequest, err: %s", err))
			return nil, errors
		}

		reqs = append(reqs, &adapters.RequestData{
			Method:  "POST",
			Uri:     a.endpoint + fmt.Sprintf("?aid=%d", sourceId),
			Body:    body,
			Headers: headers,
			ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
		})
	}

	return reqs, errors
}

func (a *AdtargetAdapter) MakeBids(bidReq *openrtb2.BidRequest, unused *adapters.RequestData, httpRes *adapters.ResponseData) (*adapters.BidderResponse, []error) {

	if httpRes.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if httpRes.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", httpRes.StatusCode),
		}}
	}
	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(httpRes.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("error while decoding response, err: %s", err),
		}}
	}

	bidResponse := adapters.NewBidderResponse()
	var errors []error

	var impOK bool
	for _, sb := range bidResp.SeatBid {
		for i := 0; i < len(sb.Bid); i++ {

			bid := sb.Bid[i]

			impOK = false
			mediaType := openrtb_ext.BidTypeBanner
			for _, imp := range bidReq.Imp {
				if imp.ID == bid.ImpID {

					impOK = true

					if imp.Video != nil {
						mediaType = openrtb_ext.BidTypeVideo
						break
					}
				}
			}

			if !impOK {
				errors = append(errors, &errortypes.BadServerResponse{
					Message: fmt.Sprintf("ignoring bid id=%s, request doesn't contain any impression with id=%s", bid.ID, bid.ImpID),
				})
				continue
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: mediaType,
			})
		}
	}

	return bidResponse, errors
}

func validateImpressionAndSetExt(imp *openrtb2.Imp) (int, error) {

	if imp.Banner == nil && imp.Video == nil {
		return 0, &errortypes.BadInput{
			Message: fmt.Sprintf("ignoring imp id=%s, Adtarget supports only Video and Banner", imp.ID),
		}
	}

	if 0 == len(imp.Ext) {
		return 0, &errortypes.BadInput{
			Message: fmt.Sprintf("ignoring imp id=%s, extImpBidder is empty", imp.ID),
		}
	}

	var bidderExt adapters.ExtImpBidder

	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return 0, &errortypes.BadInput{
			Message: fmt.Sprintf("ignoring imp id=%s, error while decoding extImpBidder, err: %s", imp.ID, err),
		}
	}

	impExt := openrtb_ext.ExtImpAdtarget{}
	err := jsonutil.Unmarshal(bidderExt.Bidder, &impExt)
	if err != nil {
		return 0, &errortypes.BadInput{
			Message: fmt.Sprintf("ignoring imp id=%s, error while decoding impExt, err: %s", imp.ID, err),
		}
	}

	// common extension for all impressions
	var impExtBuffer []byte

	impExtBuffer, err = json.Marshal(&adtargetImpExt{
		Adtarget: impExt,
	})
	if err != nil {
		return 0, &errortypes.BadInput{
			Message: fmt.Sprintf("ignoring imp id=%s, error while encoding impExt, err: %s", imp.ID, err),
		}
	}
	if impExt.BidFloor > 0 {
		imp.BidFloor = impExt.BidFloor
	}

	imp.Ext = impExtBuffer

	aid, err := impExt.SourceId.Int64()
	if err != nil {
		return 0, &errortypes.BadInput{
			Message: fmt.Sprintf("ignoring imp id=%s, aid parsing err: %s", imp.ID, err),
		}
	}
	return int(aid), nil
}

// Builder builds a new instance of the Adtarget adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &AdtargetAdapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}
