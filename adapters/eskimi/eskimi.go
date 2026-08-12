package eskimi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/adcom1"
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

// Builder builds a new instance of the Eskimi adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	return &adapter{endpoint: config.Endpoint}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{Message: "no impressions in the bid request"}}
	}

	outgoing := *request
	outgoing.Imp = append([]openrtb2.Imp(nil), request.Imp...)

	first, err := parseImpExt(outgoing.Imp[0])
	if err != nil {
		return nil, []error{err}
	}

	if err := setPlacementID(&outgoing, first.PlacementID); err != nil {
		return nil, []error{err}
	}

	applyRequestParams(&outgoing, first)

	// Imps whose ext fails to parse are dropped so malformed data is never
	// forwarded upstream; the parse error is still reported to the caller.
	imps := make([]openrtb2.Imp, 0, len(outgoing.Imp))
	impExts := make([]*openrtb_ext.ExtImpEskimi, 0, len(outgoing.Imp))
	imps = append(imps, outgoing.Imp[0])
	impExts = append(impExts, &first)
	var errs []error
	for i := 1; i < len(outgoing.Imp); i++ {
		ext, perr := parseImpExt(outgoing.Imp[i])
		if perr != nil {
			errs = append(errs, perr)
			continue
		}
		imps = append(imps, outgoing.Imp[i])
		impExts = append(impExts, &ext)
	}
	outgoing.Imp = imps

	applyImpParams(&outgoing, impExts)

	body, err := jsonutil.Marshal(&outgoing)
	if err != nil {
		return nil, append(errs, err)
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    body,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(outgoing.Imp),
	}}, errs
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}
	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	var errs []error
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	if response.Cur != "" {
		bidResponse.Currency = response.Cur
	}
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bidType, typeErr := getMediaTypeForBid(request.Imp, seatBid.Bid[i])
			if typeErr != nil {
				errs = append(errs, typeErr)
				continue
			}
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			})
		}
	}
	return bidResponse, errs
}

func parseImpExt(imp openrtb2.Imp) (openrtb_ext.ExtImpEskimi, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return openrtb_ext.ExtImpEskimi{}, &errortypes.BadInput{Message: fmt.Sprintf("invalid imp.ext for imp %s: %s", imp.ID, err)}
	}
	var eskimiExt openrtb_ext.ExtImpEskimi
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &eskimiExt); err != nil {
		return openrtb_ext.ExtImpEskimi{}, &errortypes.BadInput{Message: fmt.Sprintf("invalid imp.ext.bidder for imp %s: %s", imp.ID, err)}
	}
	return eskimiExt, nil
}

// setPlacementID writes the placement ID to site.ext.placementId or app.ext.placementId,
// preserving any other existing keys in the ext object.
func setPlacementID(request *openrtb2.BidRequest, placementID int64) error {
	patch := func(ext json.RawMessage) (json.RawMessage, error) {
		fields := map[string]json.RawMessage{}
		if len(ext) > 0 {
			if err := jsonutil.Unmarshal(ext, &fields); err != nil {
				return nil, err
			}
		}
		id, err := jsonutil.Marshal(placementID)
		if err != nil {
			return nil, err
		}
		fields["placementId"] = id
		return jsonutil.Marshal(fields)
	}

	switch {
	case request.Site != nil:
		site := *request.Site
		ext, err := patch(site.Ext)
		if err != nil {
			return &errortypes.BadInput{Message: fmt.Sprintf("failed to set placementId on site.ext: %s", err)}
		}
		site.Ext = ext
		request.Site = &site
	case request.App != nil:
		app := *request.App
		ext, err := patch(app.Ext)
		if err != nil {
			return &errortypes.BadInput{Message: fmt.Sprintf("failed to set placementId on app.ext: %s", err)}
		}
		app.Ext = ext
		request.App = &app
	default:
		return &errortypes.BadInput{Message: "request must contain either site or app"}
	}
	return nil
}

// applyRequestParams promotes request-level blocklists from the first imp's bidder ext
// when the request doesn't already set them. Existing publisher/wrapper values win.
func applyRequestParams(request *openrtb2.BidRequest, ext openrtb_ext.ExtImpEskimi) {
	if len(request.BCat) == 0 && len(ext.Bcat) > 0 {
		request.BCat = ext.Bcat
	}
	if len(request.BAdv) == 0 && len(ext.Badv) > 0 {
		request.BAdv = ext.Badv
	}
	if len(request.BApp) == 0 && len(ext.Bapp) > 0 {
		request.BApp = ext.Bapp
	}
}

// applyImpParams applies per-imp params from the bidder ext: defaults imp.secure to 1,
// fills bid floor when unset, and fans out battr to both banner.battr and video.battr.
func applyImpParams(request *openrtb2.BidRequest, exts []*openrtb_ext.ExtImpEskimi) {
	for i := range request.Imp {
		if request.Imp[i].Secure == nil {
			s := int8(1)
			request.Imp[i].Secure = &s
		}
		ext := exts[i]
		if ext == nil {
			continue
		}
		if request.Imp[i].BidFloor == 0 && ext.BidFloor > 0 {
			request.Imp[i].BidFloor = ext.BidFloor
			if ext.BidFloorCur != "" {
				request.Imp[i].BidFloorCur = ext.BidFloorCur
			}
		}
		if len(ext.Battr) > 0 {
			attrs := make([]adcom1.CreativeAttribute, len(ext.Battr))
			for j, v := range ext.Battr {
				attrs[j] = adcom1.CreativeAttribute(v)
			}
			if request.Imp[i].Banner != nil && len(request.Imp[i].Banner.BAttr) == 0 {
				banner := *request.Imp[i].Banner
				banner.BAttr = attrs
				request.Imp[i].Banner = &banner
			}
			if request.Imp[i].Video != nil && len(request.Imp[i].Video.BAttr) == 0 {
				video := *request.Imp[i].Video
				video.BAttr = attrs
				request.Imp[i].Video = &video
			}
		}
	}
}

// getMediaTypeForBid resolves the bid's media type. When bid.mtype is set it is treated
// as authoritative; an unsupported value returns an error rather than falling back to
// imp inference. When bid.mtype is unset, the imp's media type is used; multi-format
// imps without mtype cannot be disambiguated and return an error.
func getMediaTypeForBid(imps []openrtb2.Imp, bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	if bid.MType != 0 {
		switch bid.MType {
		case openrtb2.MarkupBanner:
			return openrtb_ext.BidTypeBanner, nil
		case openrtb2.MarkupVideo:
			return openrtb_ext.BidTypeVideo, nil
		default:
			return "", &errortypes.BadServerResponse{
				Message: fmt.Sprintf("unsupported bid.mtype %d for impression %s (banner and video only)", bid.MType, bid.ImpID),
			}
		}
	}
	for _, imp := range imps {
		if imp.ID != bid.ImpID {
			continue
		}
		switch {
		case imp.Banner != nil && imp.Video == nil:
			return openrtb_ext.BidTypeBanner, nil
		case imp.Video != nil && imp.Banner == nil:
			return openrtb_ext.BidTypeVideo, nil
		case imp.Banner != nil && imp.Video != nil:
			return "", &errortypes.BadServerResponse{
				Message: fmt.Sprintf("bid for multi-format imp %s requires bid.mtype to disambiguate", bid.ImpID),
			}
		}
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("unsupported media type for impression %s (banner and video only)", bid.ImpID),
		}
	}
	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("unable to resolve media type for impression %s", bid.ImpID),
	}
}
