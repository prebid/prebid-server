package yandex

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"text/template"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/errortypes"
	"github.com/prebid/prebid-server/v2/macros"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

const (
	refererQueryKey  = "target-ref"
	currencyQueryKey = "ssp-cur"
	impIdQueryKey    = "imp-id"
)

type adapter struct {
	endpoint *template.Template
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	template, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &adapter{
		endpoint: template,
	}

	return bidder, nil
}

func (a *adapter) MakeRequests(requestData *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var (
		requests []*adapters.RequestData
		errors   []error
	)

	referer := getReferer(requestData)
	currency := getCurrency(requestData)

	for i := range requestData.Imp {
		imp := requestData.Imp[i]
		var yandexExt openrtb_ext.ExtImpYandex

		if err := getYandexImpExt(imp, &yandexExt); err != nil {
			errors = append(errors, err)
			continue
		}

		if err := modifyImp(&imp); err != nil {
			errors = append(errors, err)
			continue
		}

		var resolvedUrl, err = a.resolveUrl(yandexExt, referer, currency)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		splittedRequestData := splitRequestDataByImp(requestData, imp)

		requestBody, err := json.Marshal(splittedRequestData)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		requests = append(requests, &adapters.RequestData{
			Method:  "POST",
			Uri:     resolvedUrl,
			Body:    requestBody,
			Headers: getHeaders(&splittedRequestData),
		})
	}

	return requests, errors
}

func getHeaders(request *openrtb2.BidRequest) http.Header {
	headers := http.Header{}

	if request.Device != nil {
		addNonEmptyHeaders(&headers, map[string]string{
			"Accept-Language": request.Device.Language,
			"User-Agent":      request.Device.UA,
			"X-Forwarded-For": request.Device.IP,
			"X-Real-Ip":       request.Device.IP,
			"Content-Type":    "application/json;charset=utf-8",
			"Accept":          "application/json",
		})
	}

	return headers
}

func addNonEmptyHeaders(headers *http.Header, headerValues map[string]string) {
	for key, value := range headerValues {
		if len(value) > 0 {
			headers.Add(key, value)
		}
	}
}

// Request is in shared memory, so we have to make a shallow copy for further modification (imp is already a shallow copy)
func splitRequestDataByImp(request *openrtb2.BidRequest, imp openrtb2.Imp) openrtb2.BidRequest {
	requestCopy := *request
	requestCopy.Imp = []openrtb2.Imp{imp}

	return requestCopy
}

func getYandexImpExt(imp openrtb2.Imp, yandexExt *openrtb_ext.ExtImpYandex) error {
	var ext adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &ext); err != nil {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: unable to unmarshal ext", imp.ID),
		}
	}

	if err := json.Unmarshal(ext.Bidder, yandexExt); err != nil {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: unable to unmarshal ext.bidder", imp.ID),
		}
	}

	return nil
}

func modifyImp(imp *openrtb2.Imp) error {
	if imp.Banner != nil {
		banner, err := modifyBanner(*imp.Banner)
		if banner != nil {
			imp.Banner = banner
		}
		return err
	}

	if imp.Native != nil {
		return nil
	}

	return &errortypes.BadInput{
		Message: fmt.Sprintf("Unsupported format. Yandex only supports banner and native types. Ignoring imp id #%s", imp.ID),
	}
}

func modifyBanner(banner openrtb2.Banner) (*openrtb2.Banner, error) {
	format := banner.Format

	if banner.W == nil || banner.H == nil || *banner.W == 0 || *banner.H == 0 {
		if len(format) == 0 {
			return nil, &errortypes.BadInput{
				Message: "Invalid size provided for Banner",
			}
		}

		firstFormat := format[0]
		banner.H = &firstFormat.H
		banner.W = &firstFormat.W
	}

	return &banner, nil
}

// "Un-templates" the endpoint by replacing macroses and adding the required query parameters
func (a *adapter) resolveUrl(yandexExt openrtb_ext.ExtImpYandex, referer string, currency string) (string, error) {
	pageID := strconv.Itoa(int(yandexExt.PageID))
	impID := strconv.Itoa(int(yandexExt.ImpID))
	params := macros.EndpointTemplateParams{PageID: pageID}

	endpointStr, err := macros.ResolveMacros(a.endpoint, params)
	if err != nil {
		return "", err
	}

	parsedUrl, err := url.Parse(endpointStr)
	if err != nil {
		return "", err
	}

	addNonEmptyQueryParams(parsedUrl, map[string]string{
		refererQueryKey:  referer,
		currencyQueryKey: currency,
		impIdQueryKey:    impID,
	})

	return parsedUrl.String(), nil
}

func addNonEmptyQueryParams(url *url.URL, queryMap map[string]string) {
	query := url.Query()
	for key, value := range queryMap {
		if len(value) > 0 {
			query.Add(key, value)
		}
	}

	url.RawQuery = query.Encode()
}

func getReferer(request *openrtb2.BidRequest) string {
	if request.Site == nil {
		return ""
	}

	return request.Site.Domain
}

func getCurrency(request *openrtb2.BidRequest) string {
	if len(request.Cur) == 0 {
		return ""
	}

	return request.Cur[0]
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, _ *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {

	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}

	var bidResponse openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &bidResponse); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Bad server response: %d", err),
		}}
	}

	bidResponseWithCapacity := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))

	var errors []error

	for _, seatBid := range bidResponse.SeatBid {
		for i := range seatBid.Bid {
			bid := seatBid.Bid[i]

			imp, err := lookupImpById(bid.ImpID, request.Imp)
			if err != nil {
				errors = append(errors, err)
				continue
			}

			bidType, err := getBidType(*imp)
			if err != nil {
				errors = append(errors, err)
				continue
			}

			bidResponseWithCapacity.Bids = append(bidResponseWithCapacity.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidType,
			})
		}
	}

	return bidResponseWithCapacity, errors
}

func lookupImpById(impID string, imps []openrtb2.Imp) (*openrtb2.Imp, error) {
	for _, imp := range imps {
		if impID == imp.ID {
			return &imp, nil
		}
	}

	return nil, &errortypes.BadInput{
		Message: fmt.Sprintf("Invalid bid imp ID #%s does not match any imp IDs from the original bid request", impID),
	}
}

func getBidType(imp openrtb2.Imp) (openrtb_ext.BidType, error) {
	if imp.Native != nil {
		return openrtb_ext.BidTypeNative, nil
	}

	if imp.Banner != nil {
		return openrtb_ext.BidTypeBanner, nil
	}

	if imp.Audio != nil {
		return openrtb_ext.BidTypeAudio, nil
	}

	if imp.Video != nil {
		return openrtb_ext.BidTypeVideo, nil
	}

	return "", &errortypes.BadInput{
		Message: fmt.Sprintf("Processing an invalid impression; cannot resolve impression type for imp #%s", imp.ID),
	}
}
