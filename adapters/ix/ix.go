package ix

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
)

var maxRequests = 20 // it's not a const for the unit test convenience

type IxAdapter struct {
	http *adapters.HTTPAdapter
	URI  string
}

// Name is used for cookies and such
func (a *IxAdapter) Name() string {
	return string(openrtb_ext.BidderIx)
}

func (a *IxAdapter) SkipNoCookies() bool {
	return false
}

func (a *IxAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	return nil, nil
}

func (a *IxAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	logAsJSON("bid-request: %s", request)

	nImp := len(request.Imp)
	if nImp == 0 {
		return nil, nil
	}
	if nImp > maxRequests {
		request.Imp = request.Imp[:maxRequests]
		nImp = maxRequests
	}

	// Multi-size banner imps are split into single-size requests.
	// The first size imp requests are added to the first slice.
	// Additional size requests are added to the second slice and are merged with the first at the end.
	// Preallocate the max possible size to avoid reallocating arrays.
	requests := make([]*adapters.RequestData, 0, maxRequests)
	multiSizeRequests := make([]*adapters.RequestData, 0, maxRequests-nImp)
	errs := make([]error, 0, 1)

	headers := http.Header{
		"Content-Type": {"application/json;charset=utf-8"},
		"Accept":       {"application/json"}}

	imps := request.Imp
	for iImp := range imps {
		request.Imp = imps[iImp : iImp+1]
		if request.Site != nil {
			setSitePublisherId(request, iImp)
		}

		if request.Imp[0].Banner != nil {
			banner := *request.Imp[0].Banner
			request.Imp[0].Banner = &banner
			formats := getBannerFormats(&banner)
			for iFmt := range formats {
				banner.Format = formats[iFmt : iFmt+1]
				banner.W = openrtb.Uint64Ptr(banner.Format[0].W)
				banner.H = openrtb.Uint64Ptr(banner.Format[0].H)
				if requestData, err := createRequestData(a, request, &headers); err == nil {
					if iFmt == 0 {
						requests = append(requests, requestData)
					} else {
						multiSizeRequests = append(multiSizeRequests, requestData)
					}
				}
				if len(multiSizeRequests) == cap(multiSizeRequests) {
					break
				}
			}
		} else if requestData, err := createRequestData(a, request, &headers); err == nil {
			requests = append(requests, requestData)
		}
	}
	request.Imp = imps

	return append(requests, multiSizeRequests...), errs
}

func setSitePublisherId(request *openrtb.BidRequest, iImp int) {
	if iImp == 0 {
		// first impression - create a site and pub copy
		site := *request.Site
		if site.Publisher == nil {
			site.Publisher = &openrtb.Publisher{}
		} else {
			publisher := *site.Publisher
			site.Publisher = &publisher
		}
		request.Site = &site
	}

	// 'adapters/bidder.go': Bidder implementations may safely assume that this
	// JSON has been validated by their static/bidder-params/{bidder}.json file.
	var bidderExt adapters.ExtImpBidder
	var ixExt openrtb_ext.ExtImpIx
	json.Unmarshal(request.Imp[0].Ext, &bidderExt)
	json.Unmarshal(bidderExt.Bidder, &ixExt)

	request.Site.Publisher.ID = ixExt.SiteId
}

func getBannerFormats(banner *openrtb.Banner) []openrtb.Format {
	if len(banner.Format) == 0 && banner.W != nil && banner.H != nil {
		banner.Format = []openrtb.Format{{W: *banner.W, H: *banner.H}}
	}
	return banner.Format
}

func createRequestData(a *IxAdapter, request *openrtb.BidRequest, headers *http.Header) (*adapters.RequestData, error) {
	body, err := json.Marshal(request)
	return &adapters.RequestData{
		Method:  "POST",
		Uri:     a.URI,
		Body:    body,
		Headers: *headers,
	}, err
}

func (a *IxAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if glog.V(3) {
		glog.Infof("bid-response: status-code: %d, body: %s", response.StatusCode, response.Body)
	}
	switch {
	case response.StatusCode == http.StatusNoContent:
		return nil, nil
	case response.StatusCode == http.StatusBadRequest:
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	case response.StatusCode != http.StatusOK:
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var bidResponse openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResponse); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("JSON parsing error: %v", err),
		}}
	}

	// Until the time we support multi-format ad units, we'll use a bid request impression media type
	// as a bid response bid type. They are linked by the impression id.
	impMediaType := map[string]openrtb_ext.BidType{}
	for _, imp := range internalRequest.Imp {
		if imp.Video != nil {
			impMediaType[imp.ID] = openrtb_ext.BidTypeVideo
		} else {
			impMediaType[imp.ID] = openrtb_ext.BidTypeBanner
		}
	}

	bidderResponse := adapters.NewBidderResponseWithBidsCapacity(5)
	bidderResponse.Currency = bidResponse.Cur

	var errs []error

	for _, seatBid := range bidResponse.SeatBid {
		for _, bid := range seatBid.Bid {
			bidType, ok := impMediaType[bid.ImpID]
			if !ok {
				errs = append(errs, fmt.Errorf("Unmatched impression id: %s.", bid.ImpID))
			}
			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidType,
			})
		}
	}

	return bidderResponse, errs
}

func logAsJSON(format string, arg interface{}) {
	if glog.V(3) {
		if s, err := json.Marshal(arg); err == nil {
			glog.Infof(format, s)
		}
	}
}

// Builder builds a new instance of the Ix adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &IxAdapter{
		URI: config.Endpoint,
	}
	return bidder, nil
}

func NewIxLegacyAdapter(config *adapters.HTTPAdapterConfig, uri string) *IxAdapter {
	return nil
}
