package adocean

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"text/template"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/macros"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

const (
	adapterVersion = "2.0.0"
	maxUriLength   = 8000
	slaveIDLength  = 10
)

type responseAdUnit struct {
	ID       string   `json:"id"`
	CrID     string   `json:"crid"`
	Currency string   `json:"currency"`
	Price    string   `json:"price"`
	TTL      string   `json:"ttl"`
	Width    string   `json:"width"`
	Height   string   `json:"height"`
	IsVideo  bool     `json:"isVideo"`
	Code     string   `json:"code"`
	ADomain  []string `json:"adomain"`
	Error    string   `json:"error"`
}

type adapter struct {
	endpointTemplate *template.Template
}

// Builder builds a new instance of the AdOcean adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	endpointTemplate, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, errors.New("unable to parse endpoint template")
	}

	return &adapter{endpointTemplate: endpointTemplate}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{
			Message: "No impression in the bid request",
		}}
	}
	requests := make([]*adapters.RequestData, 0, len(request.Imp))
	var errs []error

	for index := range request.Imp {
		imp := &request.Imp[index]
		if err := validateImp(imp); err != nil {
			errs = append(errs, err)
			continue
		}
		params, err := parseImpExt(imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		requestData, err := a.makeRequest(request, imp, params)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		requests = append(requests, requestData)
	}

	return requests, errs
}

func validateImp(imp *openrtb2.Imp) error {
	if imp.Banner == nil && imp.Video == nil {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("ignoring imp id=%s: AdOcean supports only banner and instream video", imp.ID),
		}
	}
	if imp.Video != nil && (imp.Video.Plcmt == adcom1.VideoPlcmtAccompanyingContent ||
		imp.Video.Plcmt == adcom1.VideoPlcmtNoContent ||
		imp.Video.Placement == adcom1.VideoPlacementInBanner) {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("ignoring imp id=%s: AdOcean doesn't support outstream video", imp.ID),
		}
	}
	return nil
}

func parseImpExt(imp *openrtb2.Imp) (*openrtb_ext.ExtImpAdOcean, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("ignoring imp id=%s: failed to parse ext.bidder: %v", imp.ID, err),
		}
	}

	var params openrtb_ext.ExtImpAdOcean
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &params); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("ignoring imp id=%s: failed to parse AdOcean parameters: %v", imp.ID, err),
		}
	}

	return &params, nil
}

func (a *adapter) resolveEndpointTemplate(emitterPrefix string) (string, error) {
	endpoint, err := macros.ResolveMacros(a.endpointTemplate, macros.EndpointTemplateParams{Host: emitterPrefix})
	if err != nil {
		return "", &errortypes.BadInput{Message: "unable to resolve endpoint template: " + err.Error()}
	}
	return endpoint, nil
}

func (a *adapter) makeRequest(request *openrtb2.BidRequest, imp *openrtb2.Imp, params *openrtb_ext.ExtImpAdOcean) (*adapters.RequestData, error) {
	endpoint, err := a.resolveEndpointTemplate(params.EmitterPrefix)
	if err != nil {
		return nil, err
	}

	requestURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, &errortypes.BadInput{Message: "malformed endpoint URL: " + err.Error()}
	}

	randomizedPart := rand.Intn(90000000) + 10000000
	if request.Test == 1 {
		randomizedPart = 10000000
	}
	requestURL.Path = "/_" + strconv.Itoa(randomizedPart) + "/ad.json"
	// RFC 3986 requires that spaces in query parameters be encoded as %20,
	// but the Go standard library encodes them as +
	query, err := buildQuery(request, imp, params)
	if err != nil {
		return nil, err
	}
	requestURL.RawQuery = strings.ReplaceAll(query.Encode(), "+", "%20")
	if len(requestURL.String()) >= maxUriLength {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("AdOcean request URL exceeds maximum length of %d characters", maxUriLength),
		}
	}

	return &adapters.RequestData{
		Method:  http.MethodGet,
		Uri:     requestURL.String(),
		Headers: buildHeaders(request),
		ImpIDs:  []string{imp.ID},
	}, nil
}

func buildQuery(request *openrtb2.BidRequest, imp *openrtb2.Imp, params *openrtb_ext.ExtImpAdOcean) (url.Values, error) {
	query := url.Values{}
	query.Set("pbsrv_v", adapterVersion)
	if params.MasterID == "" || params.SlaveID == "" {
		return nil, &errortypes.BadInput{
			Message: "missing required AdOcean parameters: masterId and slaveId must be provided",
		}
	}
	query.Set("id", params.MasterID)
	query.Set("slaves", shortSlaveID(params.SlaveID))

	if request.Regs != nil && request.Regs.GDPR != nil {
		query.Set("gdpr", strconv.Itoa(int(*request.Regs.GDPR)))
	}
	if request.User != nil {
		if request.User.Consent != "" {
			query.Set("gdpr_consent", request.User.Consent)
		}
		if request.User.BuyerUID != "" {
			query.Set("aouserid", request.User.BuyerUID)
		}
	}

	for key, value := range params.EmitterRequestParams {
		query.Add(key, fmt.Sprint(value))
	}

	if imp.Video != nil {
		query.Set("spots", "1")
		if imp.Video.MaxDuration > 0 {
			maxDuration := strconv.FormatInt(imp.Video.MaxDuration, 10)
			query.Set("dur", maxDuration)
			query.Set("maxdur", maxDuration)
		}
		if imp.Video.MinDuration > 0 {
			query.Set("mindur", strconv.FormatInt(imp.Video.MinDuration, 10))
		}
	} else if imp.Banner != nil {
		if sizes := getBannerSizes(imp.Banner); len(sizes) > 0 {
			query.Set("aosize", strings.Join(sizes, ","))
		}
	}

	return query, nil
}

func shortSlaveID(slaveID string) string {
	if len(slaveID) <= slaveIDLength {
		return slaveID
	}
	return slaveID[len(slaveID)-slaveIDLength:]
}

func getBannerSizes(banner *openrtb2.Banner) []string {
	if len(banner.Format) > 0 {
		sizes := make([]string, 0, len(banner.Format))
		for _, format := range banner.Format {
			sizes = append(sizes, strconv.FormatInt(format.W, 10)+"x"+strconv.FormatInt(format.H, 10))
		}
		return sizes
	}

	if banner.W != nil && banner.H != nil {
		return []string{strconv.FormatInt(*banner.W, 10) + "x" + strconv.FormatInt(*banner.H, 10)}
	}
	return nil
}

func buildHeaders(request *openrtb2.BidRequest) http.Header {
	headers := http.Header{
		"Accept":       []string{"application/json"},
		"Content-Type": []string{"application/json;charset=utf-8"},
	}
	if request.Device != nil {
		if request.Device.UA != "" {
			headers.Set("User-Agent", request.Device.UA)
		}
		if request.Device.IP != "" {
			headers.Set("X-Forwarded-For", request.Device.IP)
		} else if request.Device.IPv6 != "" {
			headers.Set("X-Forwarded-For", request.Device.IPv6)
		}
	}
	if request.Site != nil && request.Site.Page != "" {
		headers.Set("Referer", request.Site.Page)
	}
	return headers
}

func (a *adapter) MakeBids(
	internalRequest *openrtb2.BidRequest,
	externalRequest *adapters.RequestData,
	response *adapters.ResponseData,
) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: "unexpected status code: 400",
		}}
	}
	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("unexpected status code: %d", response.StatusCode),
		}}
	}

	var adUnits []responseAdUnit
	if err := jsonutil.Unmarshal(response.Body, &adUnits); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "failed to decode AdOcean response: " + err.Error(),
		}}
	}

	bidderResponse := adapters.NewBidderResponseWithBidsCapacity(len(adUnits))
	var lastCurrency *string = nil
	var errs []error
	for _, adUnit := range adUnits {
		if adUnit.Error == "true" {
			continue
		}
		impID, found := findImpID(internalRequest, adUnit.ID)
		if !found {
			continue
		}

		typedBid, currency, err := makeBid(adUnit, impID)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		bidderResponse.Bids = append(bidderResponse.Bids, typedBid)
		if lastCurrency == nil {
			lastCurrency = &currency
		} else if *lastCurrency != currency {
			errs = append(errs, &errortypes.BadServerResponse{
				Message: fmt.Sprintf("inconsistent currencies in AdOcean response: %s and %s", *lastCurrency, currency),
			})
			continue
		}
		bidderResponse.Currency = currency
	}

	if len(bidderResponse.Bids) == 0 {
		return nil, errs
	}

	return bidderResponse, errs
}

func findImpID(internalRequest *openrtb2.BidRequest, placementID string) (string, bool) {
	for index := range internalRequest.Imp {
		imp := &internalRequest.Imp[index]

		params, err := parseImpExt(imp)
		if err == nil && params.SlaveID == placementID {
			return imp.ID, true
		}
	}
	return "", false
}

func makeBid(adUnit responseAdUnit, impID string) (*adapters.TypedBid, string, error) {
	if adUnit.Code == "" || adUnit.Height == "" || adUnit.Width == "" || adUnit.Price == "" {
		return nil, "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("incomplete bid for AdOcean placement %q", adUnit.ID),
		}
	}

	price, err := strconv.ParseFloat(adUnit.Price, 64)
	if err != nil {
		return nil, "", invalidBidField(adUnit.ID, "price", err)
	}
	width, err := strconv.ParseInt(adUnit.Width, 10, 64)
	if err != nil {
		return nil, "", invalidBidField(adUnit.ID, "width", err)
	}
	height, err := strconv.ParseInt(adUnit.Height, 10, 64)
	if err != nil {
		return nil, "", invalidBidField(adUnit.ID, "height", err)
	}
	ttl, err := strconv.ParseInt(adUnit.TTL, 10, 64)
	if err != nil && adUnit.TTL != "" {
		return nil, "", invalidBidField(adUnit.ID, "ttl", err)
	}
	adMarkup, err := url.PathUnescape(adUnit.Code)
	if err != nil {
		return nil, "", invalidBidField(adUnit.ID, "code", err)
	}

	bidType := openrtb_ext.BidTypeBanner
	if adUnit.IsVideo {
		bidType = openrtb_ext.BidTypeVideo
	}

	aDomain := adUnit.ADomain
	if aDomain == nil {
		aDomain = []string{}
	}

	return &adapters.TypedBid{
		Bid: &openrtb2.Bid{
			ID:      adUnit.ID,
			ImpID:   impID,
			Price:   price,
			AdM:     adMarkup,
			CrID:    adUnit.CrID,
			ADomain: aDomain,
			W:       width,
			H:       height,
			Exp:     ttl,
		},
		BidMeta: &openrtb_ext.ExtBidPrebidMeta{
			AdvertiserDomains: aDomain,
		},
		BidType: bidType,
	}, adUnit.Currency, nil
}

func invalidBidField(placementID string, field string, err error) error {
	return &errortypes.BadServerResponse{
		Message: fmt.Sprintf("invalid %s in bid for AdOcean placement %q: %v", field, placementID, err),
	}
}
