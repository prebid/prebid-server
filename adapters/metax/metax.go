package metax

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
)

type adapter struct {
	template *template.Template
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	errs := make([]error, 0, len(request.Imp))

	// split impressions
	reqDatas := make([]*adapters.RequestData, 0, len(request.Imp))
	for _, imp := range request.Imp {
		metaxExt, err := parseBidderExt(&imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if err := preprocessImp(&imp); err != nil {
			errs = append(errs, err)
			continue
		}

		endpoint, err := a.getEndpoint(metaxExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		requestCopy := *request
		requestCopy.Imp = []openrtb2.Imp{imp}
		reqJSON, err := json.Marshal(requestCopy)
		if err != nil {
			errs = append(errs, err)
			return nil, errs
		}

		headers := http.Header{}
		headers.Add("Content-Type", "application/json;charset=utf-8")
		headers.Add("Accept", "application/json")
		reqDatas = append(reqDatas, &adapters.RequestData{
			Method:  "POST",
			Uri:     endpoint,
			Body:    reqJSON,
			Headers: headers,
			ImpIDs:  openrtb_ext.GetImpIDs(requestCopy.Imp),
		})
	}

	return reqDatas, errs
}

func (a *adapter) MakeBids(bidReq *openrtb2.BidRequest, reqData *adapters.RequestData, respData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(respData) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(respData); err != nil {
		return nil, []error{err}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(respData.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	// additional no content check
	if len(bidResp.SeatBid) == 0 || len(bidResp.SeatBid[0].Bid) == 0 {
		return nil, nil
	}

	resp := adapters.NewBidderResponseWithBidsCapacity(len(bidResp.SeatBid[0].Bid))
	if len(bidResp.Cur) != 0 {
		resp.Currency = bidResp.Cur
	}
	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bid := &sb.Bid[i]
			bidType, err := getBidType(bid)
			if err != nil {
				return nil, []error{err}
			}
			resp.Bids = append(resp.Bids, &adapters.TypedBid{
				Bid:      bid,
				BidType:  bidType,
				BidVideo: getBidVideo(bid),
			})
		}
	}
	return resp, nil
}

func (a *adapter) getEndpoint(ext *openrtb_ext.ExtImpMetaX) (string, error) {
	params := macros.EndpointTemplateParams{
		PublisherID: strconv.Itoa(ext.PublisherID),
		AdUnit:      strconv.Itoa(ext.Adunit),
	}
	return macros.ResolveMacros(a.template, params)
}

func parseBidderExt(imp *openrtb2.Imp) (*openrtb_ext.ExtImpMetaX, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, err
	}

	var metaxExt openrtb_ext.ExtImpMetaX
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &metaxExt); err != nil {
		return nil, errors.New("Wrong MetaX bidder ext")
	}

	return &metaxExt, nil
}

func preprocessImp(imp *openrtb2.Imp) error {
	if imp == nil {
		return errors.New("imp is nil")
	}

	if imp.Banner != nil {
		imp.Banner = assignBannerSize(imp.Banner)
	}

	return nil
}

func assignBannerSize(banner *openrtb2.Banner) *openrtb2.Banner {
	if banner.W != nil && banner.H != nil {
		return banner
	}

	if len(banner.Format) == 0 {
		return banner
	}

	return assignBannerWidthAndHeight(banner, banner.Format[0].W, banner.Format[0].H)
}

func assignBannerWidthAndHeight(banner *openrtb2.Banner, w, h int64) *openrtb2.Banner {
	bannerCopy := *banner
	bannerCopy.W = ptrutil.ToPtr(w)
	bannerCopy.H = ptrutil.ToPtr(h)
	return &bannerCopy
}

func getBidType(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	case openrtb2.MarkupAudio:
		return openrtb_ext.BidTypeAudio, nil
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unsupported MType %d", bid.MType),
		}
	}
}

func getBidVideo(bid *openrtb2.Bid) *openrtb_ext.ExtBidPrebidVideo {
	bidVideo := openrtb_ext.ExtBidPrebidVideo{}
	if len(bid.Cat) > 0 {
		bidVideo.PrimaryCategory = bid.Cat[0]
	}
	if bid.Dur > 0 {
		bidVideo.Duration = int(bid.Dur)
	}
	return &bidVideo
}

// Builder builds a new instance of the MetaX adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	if config.Endpoint == "" {
		return nil, errors.New("endpoint is empty")
	}

	templ, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint: %v", err)
	}

	bidder := &adapter{
		template: templ,
	}
	return bidder, nil
}
