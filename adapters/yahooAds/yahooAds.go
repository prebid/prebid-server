package yahooAds

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
	"github.com/prebid/prebid-server/v4/util/ptrutil"
)

type adapter struct {
	URI string
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errors []error

	reqs := make([]*adapters.RequestData, 0, len(request.Imp))
	headers := http.Header{}

	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("x-openrtb-version", "2.6")

	if request.Device != nil && request.Device.UA != "" {
		headers.Set("User-Agent", request.Device.UA)
	}

	// Regs depends only on request-level data, so promote once and reuse the
	// result for every per-impression copy.
	promotedRegs, regsWarnings := promoteRegsExtTo26(request.Regs)
	errors = append(errors, regsWarnings...)

	for idx, imp := range request.Imp {
		var bidderExt adapters.ExtImpBidder
		err := jsonutil.Unmarshal(imp.Ext, &bidderExt)
		if err != nil {
			err = &errortypes.BadInput{
				Message: fmt.Sprintf("imp #%d: ext.bidder not provided", idx),
			}
			errors = append(errors, err)
			continue
		}

		var yahooAdsExt openrtb_ext.ExtImpYahooAds
		err = jsonutil.Unmarshal(bidderExt.Bidder, &yahooAdsExt)
		if err != nil {
			err = &errortypes.BadInput{
				Message: fmt.Sprintf("imp #%d: %s", idx, err.Error()),
			}
			errors = append(errors, err)
			continue
		}

		// Split up multi-impression requests into multiple requests so that
		// each split request is only associated to a single impression
		reqCopy := *request
		reqCopy.Imp = []openrtb2.Imp{imp}
		reqCopy.Regs = promotedRegs

		if request.Site != nil {
			siteCopy := *request.Site
			reqCopy.Site = &siteCopy
		} else if request.App != nil {
			appCopy := *request.App
			reqCopy.App = &appCopy
		}

		if err := changeRequestForBidService(&reqCopy, &yahooAdsExt); err != nil {
			errors = append(errors, err)
			continue
		}

		reqJSON, err := json.Marshal(&reqCopy)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		reqs = append(reqs, &adapters.RequestData{
			Method:  "POST",
			Uri:     a.URI,
			Body:    reqJSON,
			Headers: headers,
			ImpIDs:  openrtb_ext.GetImpIDs(reqCopy.Imp),
		})
	}

	return reqs, errors
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {

	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d.", response.StatusCode),
		}}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Bad server response: %d.", err),
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(internalRequest.Imp))

	for _, sb := range bidResp.SeatBid {
		for _, bid := range sb.Bid {
			bid := bid
			exists, mediaTypeId := getImpInfo(bid.ImpID, internalRequest.Imp)
			if !exists {
				return nil, []error{&errortypes.BadServerResponse{
					Message: fmt.Sprintf("Unknown ad unit code '%s'", bid.ImpID),
				}}
			}

			if openrtb_ext.BidTypeBanner != mediaTypeId &&
				openrtb_ext.BidTypeVideo != mediaTypeId {
				//only banner and video are mediaTypeId, anything else is ignored
				continue
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: mediaTypeId,
			})
		}
	}

	return bidResponse, nil
}

func getImpInfo(impId string, imps []openrtb2.Imp) (bool, openrtb_ext.BidType) {
	var mediaType openrtb_ext.BidType
	var exists bool
	for _, imp := range imps {
		if imp.ID == impId {
			exists = true
			if imp.Banner != nil {
				mediaType = openrtb_ext.BidTypeBanner
			} else if imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			}
			break
		}
	}
	return exists, mediaType
}

func changeRequestForBidService(request *openrtb2.BidRequest, extension *openrtb_ext.ExtImpYahooAds) error {
	/* Always override the tag ID and (site ID or app ID) of the request */
	request.Imp[0].TagID = extension.Pos
	if request.Site != nil {
		request.Site.ID = extension.Dcn
	} else if request.App != nil {
		request.App.ID = extension.Dcn
	}

	if request.Imp[0].Banner != nil {
		banner := *request.Imp[0].Banner
		request.Imp[0].Banner = &banner

		err := validateBanner(&banner)
		if err != nil {
			return err
		}
	}

	return nil
}

// promoteRegsExtTo26 moves regs.ext.{gpp, gpp_sid} to their top-level fields
// when the publisher sent them in ext and the top-level field is empty.
// When the top-level field is already set, the ext duplicate is removed so
// only the authoritative top-level value ships. A regs.ext that is not a JSON
// object is passed through untouched with a non-fatal warning.
func promoteRegsExtTo26(regs *openrtb2.Regs) (*openrtb2.Regs, []error) {
	if regs == nil || len(regs.Ext) == 0 {
		return regs, nil
	}

	var regsExt map[string]json.RawMessage
	if err := jsonutil.Unmarshal(regs.Ext, &regsExt); err != nil {
		return regs, []error{&errortypes.Warning{
			Message: "regs.ext is not a JSON object; skipped OpenRTB 2.6 regs promotion",
		}}
	}

	regsCopy := *regs
	modified := false

	if raw, ok := regsExt["gpp"]; ok {
		if regsCopy.GPP == "" {
			var v string
			if err := jsonutil.Unmarshal(raw, &v); err == nil {
				regsCopy.GPP = v
				delete(regsExt, "gpp")
				modified = true
			}
		} else {
			delete(regsExt, "gpp")
			modified = true
		}
	}

	if raw, ok := regsExt["gpp_sid"]; ok {
		if len(regsCopy.GPPSID) == 0 {
			var v []int8
			if err := jsonutil.Unmarshal(raw, &v); err == nil {
				regsCopy.GPPSID = v
				delete(regsExt, "gpp_sid")
				modified = true
			}
		} else {
			delete(regsExt, "gpp_sid")
			modified = true
		}
	}

	if !modified {
		return regs, nil
	}

	if len(regsExt) == 0 {
		regsCopy.Ext = nil
	} else {
		// stdlib json.Marshal is used because it writes a nil RawMessage as null.
		newExt, err := json.Marshal(regsExt)
		if err != nil {
			return regs, nil
		}
		regsCopy.Ext = newExt
	}

	return &regsCopy, nil
}

func validateBanner(banner *openrtb2.Banner) error {
	if banner.W != nil && banner.H != nil {
		if *banner.W == 0 || *banner.H == 0 {
			return fmt.Errorf("Invalid sizes provided for Banner %dx%d", *banner.W, *banner.H)
		}
		return nil
	}

	if len(banner.Format) == 0 {
		return fmt.Errorf("No sizes provided for Banner %v", banner.Format)
	}

	banner.W = ptrutil.ToPtr(banner.Format[0].W)
	banner.H = ptrutil.ToPtr(banner.Format[0].H)

	return nil
}

// Builder builds a new instance of the YahooAds adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		URI: config.Endpoint,
	}
	return bidder, nil
}
