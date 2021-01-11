package ix

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"golang.org/x/net/context/ctxhttp"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
)

type IxAdapter struct {
	http        *adapters.HTTPAdapter
	URI         string
	maxRequests int
}

func (a *IxAdapter) Name() string {
	return string(openrtb_ext.BidderIx)
}

func (a *IxAdapter) SkipNoCookies() bool {
	return false
}

type indexParams struct {
	SiteID string `json:"siteId"`
}

type ixBidResult struct {
	Request      *callOneObject
	StatusCode   int
	ResponseBody string
	Bid          *pbs.PBSBid
	Error        error
}

type callOneObject struct {
	requestJSON bytes.Buffer
	width       uint64
	height      uint64
	bidType     string
}

func (a *IxAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	var prioritizedRequests, requests []callOneObject

	mediaTypes := []pbs.MediaType{pbs.MEDIA_TYPE_BANNER, pbs.MEDIA_TYPE_VIDEO}
	indexReq, err := adapters.MakeOpenRTBGeneric(req, bidder, a.Name(), mediaTypes)
	if err != nil {
		return nil, err
	}

	indexReqImp := indexReq.Imp
	for i, unit := range bidder.AdUnits {
		// Supposedly fixes some segfaults
		if len(indexReqImp) <= i {
			break
		}

		var params indexParams
		err := json.Unmarshal(unit.Params, &params)
		if err != nil {
			return nil, &errortypes.BadInput{
				Message: fmt.Sprintf("unmarshal params '%s' failed: %v", unit.Params, err),
			}
		}

		if params.SiteID == "" {
			return nil, &errortypes.BadInput{
				Message: "Missing siteId param",
			}
		}

		for sizeIndex, format := range unit.Sizes {
			// Only grab this ad unit. Not supporting multi-media-type adunit yet.
			thisImp := indexReqImp[i]

			thisImp.TagID = unit.Code
			if thisImp.Banner != nil {
				thisImp.Banner.Format = []openrtb.Format{format}
				thisImp.Banner.W = &format.W
				thisImp.Banner.H = &format.H
			}
			indexReq.Imp = []openrtb.Imp{thisImp}
			// Index spec says "adunit path representing ad server inventory" but we don't have this
			// ext is DFP div ID and KV pairs if avail
			//indexReq.Imp[i].Ext = json.RawMessage("{}")

			if indexReq.Site != nil {
				// Any objects pointed to by indexReq *must not be mutated*, or we will get race conditions.
				siteCopy := *indexReq.Site
				siteCopy.Publisher = &openrtb.Publisher{ID: params.SiteID}
				indexReq.Site = &siteCopy
			}

			bidType := ""
			if thisImp.Banner != nil {
				bidType = string(openrtb_ext.BidTypeBanner)
			} else if thisImp.Video != nil {
				bidType = string(openrtb_ext.BidTypeVideo)
			}
			j, _ := json.Marshal(indexReq)
			request := callOneObject{requestJSON: *bytes.NewBuffer(j), width: format.W, height: format.H, bidType: bidType}

			// prioritize slots over sizes
			if sizeIndex == 0 {
				prioritizedRequests = append(prioritizedRequests, request)
			} else {
				requests = append(requests, request)
			}
		}
	}

	// cap the number of requests to maxRequests
	requests = append(prioritizedRequests, requests...)
	if len(requests) > a.maxRequests {
		requests = requests[:a.maxRequests]
	}

	if len(requests) == 0 {
		return nil, &errortypes.BadInput{
			Message: "Invalid ad unit/imp/size",
		}
	}

	ch := make(chan ixBidResult)
	for _, request := range requests {
		go func(bidder *pbs.PBSBidder, request callOneObject) {
			result, err := a.callOne(ctx, request.requestJSON)
			result.Request = &request
			result.Error = err
			if result.Bid != nil {
				result.Bid.BidderCode = bidder.BidderCode
				result.Bid.BidID = bidder.LookupBidID(result.Bid.AdUnitCode)
				result.Bid.Width = request.width
				result.Bid.Height = request.height
				result.Bid.CreativeMediaType = request.bidType

				if result.Bid.BidID == "" {
					result.Error = &errortypes.BadServerResponse{
						Message: fmt.Sprintf("Unknown ad unit code '%s'", result.Bid.AdUnitCode),
					}
					result.Bid = nil
				}
			}
			ch <- result
		}(bidder, request)
	}

	bids := make(pbs.PBSBidSlice, 0)
	for i := 0; i < len(requests); i++ {
		result := <-ch
		if result.Bid != nil && result.Bid.Price != 0 {
			bids = append(bids, result.Bid)
		}

		if req.IsDebug {
			debug := &pbs.BidderDebug{
				RequestURI:   a.URI,
				RequestBody:  result.Request.requestJSON.String(),
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

func (a *IxAdapter) callOne(ctx context.Context, reqJSON bytes.Buffer) (ixBidResult, error) {
	var result ixBidResult

	httpReq, _ := http.NewRequest("POST", a.URI, &reqJSON)
	httpReq.Header.Add("Content-Type", "application/json;charset=utf-8")
	httpReq.Header.Add("Accept", "application/json")

	ixResp, err := ctxhttp.Do(ctx, a.http.Client, httpReq)
	if err != nil {
		return result, err
	}

	result.StatusCode = ixResp.StatusCode

	if ixResp.StatusCode == http.StatusNoContent {
		return result, nil
	}

	if ixResp.StatusCode == http.StatusBadRequest {
		return result, &errortypes.BadInput{
			Message: fmt.Sprintf("HTTP status: %d", ixResp.StatusCode),
		}
	}

	if ixResp.StatusCode != http.StatusOK {
		return result, &errortypes.BadServerResponse{
			Message: fmt.Sprintf("HTTP status: %d", ixResp.StatusCode),
		}
	}

	defer ixResp.Body.Close()
	body, err := ioutil.ReadAll(ixResp.Body)
	if err != nil {
		return result, err
	}
	result.ResponseBody = string(body)

	var bidResp openrtb.BidResponse
	err = json.Unmarshal(body, &bidResp)
	if err != nil {
		return result, &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Error parsing response: %v", err),
		}
	}

	if len(bidResp.SeatBid) == 0 {
		return result, nil
	}
	if len(bidResp.SeatBid[0].Bid) == 0 {
		return result, nil
	}
	bid := bidResp.SeatBid[0].Bid[0]

	pbid := pbs.PBSBid{
		AdUnitCode:  bid.ImpID,
		Price:       bid.Price,
		Adm:         bid.AdM,
		Creative_id: bid.CrID,
		Width:       bid.W,
		Height:      bid.H,
		DealId:      bid.DealID,
	}

	result.Bid = &pbid
	return result, nil
}

func (a *IxAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	nImp := len(request.Imp)
	if nImp > a.maxRequests {
		request.Imp = request.Imp[:a.maxRequests]
		nImp = a.maxRequests
	}

	// Multi-size banner imps are split into single-size requests.
	// The first size imp requests are added to the first slice.
	// Additional size requests are added to the second slice and are merged with the first at the end.
	// Preallocate the max possible size to avoid reallocating arrays.
	requests := make([]*adapters.RequestData, 0, a.maxRequests)
	multiSizeRequests := make([]*adapters.RequestData, 0, a.maxRequests-nImp)
	errs := make([]error, 0, 1)

	headers := http.Header{
		"Content-Type": {"application/json;charset=utf-8"},
		"Accept":       {"application/json"}}

	imps := request.Imp
	for iImp := range imps {
		request.Imp = imps[iImp : iImp+1]
		if request.Site != nil {
			if err := setSitePublisherId(request, iImp); err != nil {
				errs = append(errs, err)
				continue
			}
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
				} else {
					errs = append(errs, err)
				}
				if len(multiSizeRequests) == cap(multiSizeRequests) {
					break
				}
			}
		} else if requestData, err := createRequestData(a, request, &headers); err == nil {
			requests = append(requests, requestData)
		} else {
			errs = append(errs, err)
		}
	}
	request.Imp = imps

	return append(requests, multiSizeRequests...), errs
}

func setSitePublisherId(request *openrtb.BidRequest, iImp int) error {
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

	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(request.Imp[0].Ext, &bidderExt); err != nil {
		return err
	}

	var ixExt openrtb_ext.ExtImpIx
	if err := json.Unmarshal(bidderExt.Bidder, &ixExt); err != nil {
		return err
	}

	request.Site.Publisher.ID = ixExt.SiteId
	return nil
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
		if imp.Banner != nil {
			impMediaType[imp.ID] = openrtb_ext.BidTypeBanner
		} else if imp.Video != nil {
			impMediaType[imp.ID] = openrtb_ext.BidTypeVideo
		} else if imp.Native != nil {
			impMediaType[imp.ID] = openrtb_ext.BidTypeNative
		} else if imp.Audio != nil {
			impMediaType[imp.ID] = openrtb_ext.BidTypeAudio
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

func NewIxLegacyAdapter(config *adapters.HTTPAdapterConfig, endpoint string) *IxAdapter {
	return &IxAdapter{
		http:        adapters.NewHTTPAdapter(config),
		URI:         endpoint,
		maxRequests: 20,
	}
}

// Builder builds a new instance of the Ix adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &IxAdapter{
		URI:         config.Endpoint,
		maxRequests: 20,
	}
	return bidder, nil
}
