package nativo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the Nativo adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	requestCopy := *request
	requestCopy.Imp = populateNativoImpExt(request.Imp)

	requestJSON, err := jsonutil.Marshal(&requestCopy)
	if err != nil {
		return nil, []error{err}
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	requestData := &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    requestJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}

	return []*adapters.RequestData{requestData}, nil
}

// populateNativoImpExt hoists a validated placementId bidder param into imp.ext.nativo.placementid,
// the location Nativo's endpoint reads it from. Bidder params only ever land under
// imp.ext.prebid.bidder.nativo (validated against static/bidder-params/nativo.json) - nothing else
// copies them to the SSP-facing key, so without this the param is silently dropped from the outgoing
// request. See https://github.com/prebid/prebid-server/issues/4393.
func populateNativoImpExt(imps []openrtb2.Imp) []openrtb2.Imp {
	updatedImps := make([]openrtb2.Imp, len(imps))
	copy(updatedImps, imps)

	for i, imp := range updatedImps {
		var bidderExt adapters.ExtImpBidder
		if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
			continue
		}

		var nativoExt openrtb_ext.ImpExtNativo
		if err := jsonutil.Unmarshal(bidderExt.Bidder, &nativoExt); err != nil || nativoExt.PlacementID == 0 {
			continue
		}

		var extMap map[string]json.RawMessage
		if err := jsonutil.Unmarshal(imp.Ext, &extMap); err != nil {
			continue
		}

		nativoParamsJSON, err := jsonutil.Marshal(struct {
			PlacementID jsonutil.StringInt `json:"placementid"`
		}{PlacementID: nativoExt.PlacementID})
		if err != nil {
			continue
		}
		extMap["nativo"] = nativoParamsJSON

		updatedExt, err := jsonutil.Marshal(extMap)
		if err != nil {
			continue
		}
		updatedImps[i].Ext = updatedExt
	}

	return updatedImps
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(response) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
	}

	var bidResp openrtb2.BidResponse

	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Bad Server Response",
		}}
	}

	bidResponse := adapters.NewBidderResponse()

	// Get the ext from the request
	var extReq openrtb_ext.ExtRequest
	if len(request.Ext) > 0 {
		_ = jsonutil.Unmarshal(request.Ext, &extReq)
	}

	var errs []error
	for seatBidIndex := range bidResp.SeatBid {
		seatBid := bidResp.SeatBid[seatBidIndex]
		for bidIndex := range seatBid.Bid {
			bid := seatBid.Bid[bidIndex]
			bidType, err := getMediaTypeForImp(bid.ImpID, request.Imp)
			if err != nil {
				// Fallback needed in some cases with Prebid SDK
				bidType, err = getMediaTypeForBid(bid)
			}
			if err != nil {
				errs = append(errs, err)
				continue
			}

			b := &adapters.TypedBid{
				Bid:     &seatBid.Bid[bidIndex],
				BidType: bidType,
				BidMeta: getRendererMeta(&extReq),
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}
	return bidResponse, errs
}

// Attempt to set NativoRenderer as the meta renderer if it exists in the request ext
func getRendererMeta(ext *openrtb_ext.ExtRequest) *openrtb_ext.ExtBidPrebidMeta {
	if ext == nil || ext.Prebid.Sdk == nil {
		return nil
	}
	const rendererName = "NativoRenderer"
	for i := range ext.Prebid.Sdk.Renderers {
		renderer := &ext.Prebid.Sdk.Renderers[i]
		if strings.EqualFold(renderer.Name, rendererName) {
			return &openrtb_ext.ExtBidPrebidMeta{
				RendererName:    renderer.Name,
				RendererVersion: renderer.Version,
			}
		}
	}
	return nil
}

// Get media type from bid response
func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	if bid.Ext != nil {
		var bidExt openrtb_ext.ExtBid
		err := jsonutil.Unmarshal(bid.Ext, &bidExt)
		if err == nil && bidExt.Prebid != nil {
			return openrtb_ext.ParseBidType(string(bidExt.Prebid.Type))
		}
	}

	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("Failed to parse impression \"%s\" mediatype", bid.ImpID),
	}
}

// Get media type from bid request
func getMediaTypeForImp(impID string, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Native != nil {
				return openrtb_ext.BidTypeNative, nil
			} else if imp.Video != nil {
				return openrtb_ext.BidTypeVideo, nil
			} else if imp.Banner != nil {
				return openrtb_ext.BidTypeBanner, nil
			}
		}
	}
	return "", fmt.Errorf("Unrecognized impression type in response from nativo: %s", impID)
}
