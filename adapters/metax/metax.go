package metax

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/errortypes"
	"github.com/prebid/prebid-server/v2/macros"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/prebid/prebid-server/v2/util/ptrutil"
)

const SupportedCurrency = "USD"

type adapter struct {
	template *template.Template
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	errs := make([]error, 0, len(request.Imp))

	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{
			Message: "No impressions in the request",
		}}
	}

	// split impressions
	reqDatas := make([]*adapters.RequestData, 0, len(request.Imp))
	for i := range request.Imp {
		imp := &request.Imp[i]
		impCopy := *imp
		requestCopy := *request

		metaxExt, err := parseBidderExt(imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if err := validateParams(metaxExt); err != nil {
			errs = append(errs, err)
			continue
		}

		if err := preprocessImp(&impCopy); err != nil {
			errs = append(errs, err)
			continue
		}

		endpoint, err := a.getEndpoint(metaxExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		impCopy.Ext = nil
		requestCopy.Imp = []openrtb2.Imp{impCopy}
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
	if respData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if respData.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Bad request", respData.StatusCode),
		}}
	}

	if respData.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d", respData.StatusCode),
		}}
	}

	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(respData.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	// additional no content check
	if len(bidResp.SeatBid) == 0 || len(bidResp.SeatBid[0].Bid) == 0 {
		return nil, nil
	}

	resp := adapters.NewBidderResponseWithBidsCapacity(len(bidResp.SeatBid[0].Bid))
	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bid := &sb.Bid[i]
			resp.Bids = append(resp.Bids, &adapters.TypedBid{
				Bid:     bid,
				BidType: getBidType(bidReq.Imp, bid.ImpID),
			})
		}
	}
	return resp, nil
}

func (a *adapter) getEndpoint(ext *openrtb_ext.ExtImpMetaX) (string, error) {
	params := macros.EndpointTemplateParams{
		PublisherID: url.PathEscape(ext.PublisherID),
		AdUnit:      url.PathEscape(ext.Adunit),
	}
	return macros.ResolveMacros(a.template, params)
}

func parseBidderExt(imp *openrtb2.Imp) (*openrtb_ext.ExtImpMetaX, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, err
	}

	var metaxExt openrtb_ext.ExtImpMetaX
	if err := json.Unmarshal(bidderExt.Bidder, &metaxExt); err != nil {
		return nil, errors.New("Wrong MetaX bidder ext")
	}

	return &metaxExt, nil
}

// publisher ID and adunit is numeric at this moment
func validateParams(ext *openrtb_ext.ExtImpMetaX) error {
	if _, err := strconv.Atoi(ext.PublisherID); err != nil {
		return errors.New("invalid publisher ID")
	}
	if _, err := strconv.Atoi(ext.Adunit); err != nil {
		return errors.New("invalid adunit")
	}
	return nil
}

func preprocessImp(imp *openrtb2.Imp) error {
	if imp == nil {
		return errors.New("imp is nil")
	}

	if imp.Banner != nil {
		bannerCopy, err := assignBannerSize(imp.Banner)
		if err != nil {
			return err
		}
		imp.Banner = bannerCopy
	}

	// clean inventory
	switch {
	case imp.Video != nil:
		imp.Banner = nil
	case imp.Banner != nil:
		imp.Video = nil
	default:
	}

	return nil
}

func assignBannerSize(banner *openrtb2.Banner) (*openrtb2.Banner, error) {
	if banner.W != nil && banner.H != nil {
		return banner, nil
	}

	return assignBannerWidthAndHeight(banner, banner.Format[0].W, banner.Format[0].H), nil
}

func assignBannerWidthAndHeight(banner *openrtb2.Banner, w, h int64) *openrtb2.Banner {
	bannerCopy := *banner
	bannerCopy.W = ptrutil.ToPtr(w)
	bannerCopy.H = ptrutil.ToPtr(h)
	return &bannerCopy
}

func getBidType(imps []openrtb2.Imp, impID string) openrtb_ext.BidType {
	for _, imp := range imps {
		if imp.ID != impID {
			continue
		}
		switch {
		case imp.Banner != nil:
			return openrtb_ext.BidTypeBanner
		case imp.Video != nil:
			return openrtb_ext.BidTypeVideo
		case imp.Native != nil:
			return openrtb_ext.BidTypeNative
		case imp.Audio != nil:
			return openrtb_ext.BidTypeAudio
		default:
		}
	}
	return openrtb_ext.BidTypeBanner
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
