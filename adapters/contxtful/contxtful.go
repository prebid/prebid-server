package contxtful

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"text/template"

	"github.com/golang/glog"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

// Essential constants
const (
	BidderName     = "contxtful"
	DefaultVersion = "v1"
	FieldBidder    = "bidder"
	PrebidPath     = "/prebid/"
)

// Helper to safely extract bidder parameters from impression extension
func extractBidderParams(impExt json.RawMessage) (*openrtb_ext.ExtImpContxtful, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(impExt, &bidderExt); err != nil {
		return nil, err
	}

	var contxtfulParams openrtb_ext.ExtImpContxtful
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &contxtfulParams); err != nil {
		return nil, err
	}

	return &contxtfulParams, nil
}

type adapter struct {
	endpointTemplate   *template.Template
	monitoringEndpoint string
}

// Builder builds a new instance of the Contxtful adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	glog.Infof("[CONTXTFUL] Initializing Contxtful adapter")

	endpoint := config.Endpoint
	if endpoint == "" {
		return nil, fmt.Errorf("missing endpoint configuration")
	}

	endpointTemplate, err := template.New("endpointTemplate").Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	var extraInfo struct {
		MonitoringEndpoint string `json:"monitoringEndpoint"`
	}
	if err := json.Unmarshal([]byte(config.ExtraAdapterInfo), &extraInfo); err != nil {
		return nil, fmt.Errorf("unable to parse extra adapter info: %v", err)
	}

	if extraInfo.MonitoringEndpoint == "" {
		return nil, fmt.Errorf("missing monitoring endpoint configuration in extra_info")
	}

	return &adapter{
		endpointTemplate:   endpointTemplate,
		monitoringEndpoint: extraInfo.MonitoringEndpoint,
	}, nil
}

// Streamlined impression validation
func validateImpressions(request *openrtb2.BidRequest) ([]string, string, []error) {
	var errors []error
	var customerId string
	validPlacements := make([]string, 0, len(request.Imp))

	for _, imp := range request.Imp {
		contxtfulParams, err := extractBidderParams(imp.Ext)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		if contxtfulParams.PlacementId == "" {
			errors = append(errors, fmt.Errorf("placementId is required for contxtful bidder"))
			continue
		}
		if contxtfulParams.CustomerId == "" {
			errors = append(errors, fmt.Errorf("customerId is required for contxtful bidder"))
			continue
		}

		if customerId == "" {
			customerId = contxtfulParams.CustomerId
		}
		validPlacements = append(validPlacements, contxtfulParams.PlacementId)
	}

	return validPlacements, customerId, errors
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	glog.Infof("[CONTXTFUL] MakeRequests called for request ID: %s", request.ID)

	var errors []error

	// Build headers efficiently
	headers := http.Header{
		"Content-Type": []string{"application/json;charset=utf-8"},
		"Accept":       []string{"application/json"},
	}

	// Add privacy-related headers if available
	if reqInfo != nil && reqInfo.GlobalPrivacyControlHeader != "" {
		headers.Set("Sec-GPC", reqInfo.GlobalPrivacyControlHeader)
	}

	// Validate impressions and extract parameters
	validPlacements, customerId, validationErrors := validateImpressions(request)
	errors = append(errors, validationErrors...)

	if len(validPlacements) == 0 {
		return nil, errors
	}

	// Extract bidder config
	bidderCustomerId, bidderVersion := extractBidderConfig(request)

	// Use bidder config customer as primary source for endpoint URL
	endpointCustomerId := customerId
	if bidderCustomerId != "" {
		endpointCustomerId = bidderCustomerId
	}

	// Build dynamic endpoint URL
	endpointParams := macros.EndpointTemplateParams{
		AccountID: endpointCustomerId,
	}
	endpoint, err := macros.ResolveMacros(a.endpointTemplate, endpointParams)
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	// Create payload
	payload := createRequestPayload(request, validPlacements, bidderCustomerId, bidderVersion, customerId)

	requestJSON, err := json.Marshal(payload)
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	requestData := &adapters.RequestData{
		Method:  "POST",
		Uri:     endpoint,
		Body:    requestJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}

	return []*adapters.RequestData{requestData}, errors
}

// Streamlined payload creation
func createRequestPayload(request *openrtb2.BidRequest, validPlacements []string, bidderCustomerId, bidderVersion, fallbackCustomerId string) map[string]interface{} {
	// Create clean request copy
	requestCopy := *request
	for i := range requestCopy.Imp {
		requestCopy.Imp[i].Ext = json.RawMessage(`{}`)
	}

	// Resolve configuration with fallbacks inline
	version := bidderVersion
	if version == "" {
		version = DefaultVersion
	}

	customer := bidderCustomerId
	if customer == "" {
		customer = fallbackCustomerId
	}

	uid := extractUserIDForCookie(request)
	if uid != "" {
		glog.Infof("[CONTXTFUL] Raw UID being passed to relay: %s", uid)
	}

	// Ensure UID is written to ortb2.user.buyeruid with deep copy
	if uid != "" {
		if requestCopy.User == nil {
			requestCopy.User = &openrtb2.User{}
		} else {
			// Create a deep copy of the User object to avoid modifying the original
			userCopy := *requestCopy.User
			requestCopy.User = &userCopy
		}
		requestCopy.User.BuyerUID = uid
	}

	return map[string]interface{}{
		"ortb2": requestCopy,
		"bidRequests": func() []map[string]interface{} {
			bidRequests := make([]map[string]interface{}, len(request.Imp))
			for i, imp := range request.Imp {
				bidRequests[i] = map[string]interface{}{
					FieldBidder: BidderName,
					"params": map[string]interface{}{
						"placementId": validPlacements[i],
					},
					"bidId": imp.ID,
				}
			}
			return bidRequests
		}(),
		"bidderRequest": map[string]interface{}{
			"bidderCode": BidderName,
		},
		"config": map[string]interface{}{
			BidderName: map[string]interface{}{
				"version":  version,
				"customer": customer,
			},
		},
	}
}

type ContxtfulRelayResponse struct {
	Syncs   []string       `json:"syncs,omitempty"`
	TraceId string         `json:"traceId,omitempty"`
	Random  float64        `json:"random,omitempty"`
	Bids    []ContxtfulBid `json:"bids,omitempty"`
}

type ContxtfulBid struct {
	ImpID      string  `json:"impid"`
	Price      float64 `json:"price"`
	AdMarkup   string  `json:"adm"`
	Width      int     `json:"w"`
	Height     int     `json:"h"`
	CreativeID string  `json:"crid,omitempty"`
	NURL       string  `json:"nurl,omitempty"`
	BURL       string  `json:"burl,omitempty"`
	BidderCode string  `json:"bidderCode,omitempty"`
}

type ContxtfulExt struct {
	Reseller string `json:"reseller,omitempty"`
}

type PrebidJSBid struct {
	RequestID   string       `json:"requestId"`
	CPM         float64      `json:"cpm"`
	Currency    string       `json:"currency"`
	Width       int          `json:"width"`
	Height      int          `json:"height"`
	CreativeID  string       `json:"creativeId"`
	Ad          string       `json:"ad"`
	TTL         int          `json:"ttl"`
	NetRevenue  bool         `json:"netRevenue"`
	MediaType   string       `json:"mediaType"`
	BidderCode  string       `json:"bidderCode"`
	PlacementID string       `json:"placementId"`
	TraceId     string       `json:"traceId,omitempty"`
	Random      float64      `json:"random,omitempty"`
	NURL        string       `json:"nurl,omitempty"`
	BURL        string       `json:"burl,omitempty"`
	Ext         ContxtfulExt `json:"ext,omitempty"`
}

type bidProcessingContext struct {
	request          *openrtb2.BidRequest
	requestData      *adapters.RequestData
	bidderCustomerId string
	bidderVersion    string
	bidderResponse   *adapters.BidderResponse
	errors           []error
	// Event URL generation fields
	version     string
	domain      string
	adRequestID string
}

// Response format handlers
// Direct response processing without over-engineered handler pattern
func (a *adapter) processResponse(responseBody []byte, ctx *bidProcessingContext) bool {
	// Extract configuration from various sources with priority
	config, err := extractRequestConfig(ctx.requestData, ctx.request, ctx.bidderCustomerId)
	if err != nil {
		ctx.errors = append(ctx.errors, &errortypes.BadInput{Message: err.Error()})
		return true
	}

	// Store configuration in context for use in bid creation
	ctx.version = config.Version
	customerId := config.CustomerID

	// Try PrebidJS format first
	var prebidBids []PrebidJSBid
	if err := json.Unmarshal(responseBody, &prebidBids); err == nil && len(prebidBids) > 0 &&
		prebidBids[0].CPM > 0 && prebidBids[0].RequestID != "" && prebidBids[0].Ad != "" {
		return a.processPrebidJSBids(prebidBids, ctx, customerId)
	}

	// Try Relay format
	var relayResponses []ContxtfulRelayResponse
	if err := json.Unmarshal(responseBody, &relayResponses); err == nil && len(relayResponses) > 0 &&
		len(relayResponses[0].Bids) > 0 {
		return a.processRelayBids(relayResponses, ctx, customerId)
	}

	// Handle trace format (acknowledges response, no bids)
	var traceData []map[string]interface{}
	if err := json.Unmarshal(responseBody, &traceData); err == nil && len(traceData) > 0 {
		if _, exists := traceData[0]["traceId"]; exists {
			return true // Trace acknowledged
		}
	}

	return false
}

func (a *adapter) processPrebidJSBids(prebidBids []PrebidJSBid, ctx *bidProcessingContext, customerId string) bool {
	for _, prebidBid := range prebidBids {
		if prebidBid.CPM == 0 || prebidBid.RequestID == "" {
			continue
		}

		currency := prebidBid.Currency
		if currency == "" {
			currency = "USD"
		}

		a.createBid(
			prebidBid.RequestID, prebidBid.CreativeID, prebidBid.Ad,
			prebidBid.CPM, prebidBid.Width, prebidBid.Height,
			prebidBid.TraceId, fmt.Sprintf("%.6f", prebidBid.Random),
			currency, ctx, customerId, prebidBid.Ext.Reseller, prebidBid.PlacementID, prebidBid.NURL, prebidBid.BURL,
		)

		if prebidBid.Currency != "" {
			ctx.bidderResponse.Currency = prebidBid.Currency
		}
	}
	return true
}

func (a *adapter) processRelayBids(relayResponses []ContxtfulRelayResponse, ctx *bidProcessingContext, customerId string) bool {
	for _, relayResp := range relayResponses {
		for _, contxtfulBid := range relayResp.Bids {
			a.createBid(
				contxtfulBid.ImpID, contxtfulBid.CreativeID, contxtfulBid.AdMarkup,
				contxtfulBid.Price, contxtfulBid.Width, contxtfulBid.Height,
				relayResp.TraceId, fmt.Sprintf("%.6f", relayResp.Random),
				"USD", ctx, customerId, contxtfulBid.BidderCode, "", contxtfulBid.NURL, contxtfulBid.BURL,
			)
		}
		a.handleUserSyncs(relayResp.Syncs, ctx)
	}
	return true
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
	}

	if adapters.IsResponseStatusCodeNoContent(response) {
		return nil, nil
	}

	bidderCustomerId, bidderVersion := extractBidderConfig(request)

	// Extract domain for event URLs
	domain := ""
	if request.Site != nil && request.Site.Domain != "" {
		domain = request.Site.Domain
	}

	ctx := &bidProcessingContext{
		request:          request,
		requestData:      requestData,
		bidderCustomerId: bidderCustomerId,
		bidderVersion:    bidderVersion,
		bidderResponse:   adapters.NewBidderResponse(),
		domain:           domain,
		adRequestID:      request.ID,
	}
	ctx.bidderResponse.Currency = "USD"

	if a.processResponse(response.Body, ctx) {
		return ctx.bidderResponse, ctx.errors
	}

	return nil, []error{fmt.Errorf("failed to parse response as Contxtful relay or Prebid.js format")}
}

// Simplified bid creation with fewer parameters
func (a *adapter) createBid(
	impID, creativeID, adMarkup string,
	price float64,
	width, height int,
	traceID, random, currency string,
	ctx *bidProcessingContext,
	customerId string,
	bidderCode string,
	placementId string,
	responseNURL, responseBURL string,
) {
	// Determine media type from impression
	var bidType openrtb_ext.BidType = openrtb_ext.BidTypeBanner
	for _, imp := range ctx.request.Imp {
		if imp.ID == impID {
			if imp.Video != nil {
				bidType = openrtb_ext.BidTypeVideo
			} else if imp.Native != nil {
				bidType = openrtb_ext.BidTypeNative
			}
			break
		}
	}

	bid := &openrtb2.Bid{
		ID:    fmt.Sprintf("%s-%s", BidderName, impID),
		ImpID: impID,
		Price: price,
		AdM:   adMarkup,
		W:     int64(width),
		H:     int64(height),
		CrID:  creativeID,
	}

	bid.NURL, bid.BURL = a.generateEventURLs(ctx, ctx.version, customerId, bid.ID, bid.ImpID, bid.Price, traceID, random, width, height, responseNURL, responseBURL, bidderCode, currency, placementId, creativeID)

	if bidExtJSON, err := createBidExtensions(price, currency, string(bidType), width, height); err == nil {
		bid.Ext = bidExtJSON
	}

	typedBid := &adapters.TypedBid{
		Bid:     bid,
		BidType: bidType,
	}

	ctx.bidderResponse.Bids = append(ctx.bidderResponse.Bids, typedBid)
}

func (a *adapter) handleUserSyncs(syncs []string, ctx *bidProcessingContext) {
	if len(syncs) == 0 || len(ctx.bidderResponse.Bids) == 0 {
		return
	}

	firstBid := ctx.bidderResponse.Bids[0].Bid
	var bidExt map[string]interface{}
	if firstBid.Ext != nil {
		json.Unmarshal(firstBid.Ext, &bidExt)
	} else {
		bidExt = make(map[string]interface{})
	}

	bidExt["syncs"] = syncs

	if bidExtJSON, err := json.Marshal(bidExt); err == nil {
		firstBid.Ext = bidExtJSON
	}
}

func createBidExtensions(price float64, currency string, bidType string, width int, height int) (json.RawMessage, error) {
	completeExt := map[string]interface{}{
		"origbidcpm": price,
		"origbidcur": currency,
		"prebid": map[string]interface{}{
			"type": bidType,
			"meta": map[string]interface{}{
				"adaptercode": BidderName,
			},
			"targeting": map[string]string{
				"hb_bidder": BidderName,
				"hb_pb":     fmt.Sprintf("%.2f", price),
				"hb_size":   fmt.Sprintf("%dx%d", width, height),
			},
		},
	}
	return json.Marshal(completeExt)
}

// Simple unified bidder config extraction - get ORTB2 data and extract params
func extractBidderConfig(request *openrtb2.BidRequest) (string, string) {
	if request == nil || request.Ext == nil {
		return "", ""
	}

	var requestExt struct {
		Prebid struct {
			BidderConfig []struct {
				Bidders []string `json:"bidders"`
				Config  struct {
					ORTB2 json.RawMessage `json:"ortb2"`
				} `json:"config"`
			} `json:"bidderconfig"`
		} `json:"prebid"`
	}

	if err := json.Unmarshal(request.Ext, &requestExt); err != nil {
		return "", ""
	}

	// Find contxtful bidder config and extract params from ORTB2 data
	for _, config := range requestExt.Prebid.BidderConfig {
		for _, bidder := range config.Bidders {
			if bidder == BidderName && config.Config.ORTB2 != nil {
				return extractContxtfulParams(config.Config.ORTB2)
			}
		}
	}
	return "", ""
}

// Extract contxtful params from any ORTB2 data (unified for both bidder config and other sources)
func extractContxtfulParams(ortb2Data json.RawMessage) (string, string) {
	var ortb2 struct {
		User struct {
			Data []struct {
				Name string `json:"name"`
				Ext  struct {
					Params struct {
						CI string `json:"ci"` // Customer ID
						EV string `json:"ev"` // Version
					} `json:"params"`
				} `json:"ext"`
			} `json:"data"`
		} `json:"user"`
	}

	if err := json.Unmarshal(ortb2Data, &ortb2); err != nil {
		return "", ""
	}

	// Find contxtful params
	for _, data := range ortb2.User.Data {
		if data.Name == BidderName && data.Ext.Params.CI != "" {
			version := data.Ext.Params.EV
			if version == "" {
				version = DefaultVersion
			}
			return data.Ext.Params.CI, version
		}
	}
	return "", ""
}

// RequestConfig holds extracted configuration from various sources
type RequestConfig struct {
	Version    string
	CustomerID string
	Source     string // "uri", "bidder_config", or "impression"
}

// extractFromURL parses Contxtful URL pattern: /{version}/prebid/{customerId}/bid
func extractFromURL(uri string) (version, customerID string) {
	idx := strings.Index(uri, PrebidPath)
	if idx == -1 {
		return "", ""
	}

	// Extract version from before /prebid/
	beforePrebid := uri[:idx]
	if lastSlash := strings.LastIndex(beforePrebid, "/"); lastSlash != -1 {
		version = beforePrebid[lastSlash+1:]
	}

	// Extract customer ID from after /prebid/
	remaining := uri[idx+len(PrebidPath):]
	if extractedCustomerID, _, found := strings.Cut(remaining, "/"); found && extractedCustomerID != "" {
		customerID = extractedCustomerID
	}

	return version, customerID
}

// extractRequestConfig extracts configuration from multiple sources with priority
func extractRequestConfig(requestData *adapters.RequestData, request *openrtb2.BidRequest, bidderCustomerId string) (*RequestConfig, error) {
	// Priority 1: Extract from request URI
	if requestData != nil {
		version, customerID := extractFromURL(requestData.Uri)
		if customerID != "" {
			if version == "" {
				version = DefaultVersion
			}
			return &RequestConfig{
				Version:    version,
				CustomerID: customerID,
				Source:     "uri",
			}, nil
		}
	}

	// Priority 2: Use bidder config (already extracted)
	if bidderCustomerId != "" {
		return &RequestConfig{
			Version:    DefaultVersion,
			CustomerID: bidderCustomerId,
			Source:     "bidder_config",
		}, nil
	}

	// Priority 3: Extract from impression parameters
	if request != nil {
		for _, imp := range request.Imp {
			if contxtfulParams, err := extractBidderParams(imp.Ext); err == nil && contxtfulParams.CustomerId != "" {
				return &RequestConfig{
					Version:    DefaultVersion,
					CustomerID: contxtfulParams.CustomerId,
					Source:     "impression",
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("No customer ID found in request URI, bidder config, or impression parameters")
}

func extractUserIDForCookie(request *openrtb2.BidRequest) string {
	if request.User == nil {
		return ""
	}

	// Priority 1: Standard ORTB2 BuyerUID field
	if request.User.BuyerUID != "" {
		return request.User.BuyerUID
	}

	// Priority 2: PBS buyeruids map (simplified navigation)
	if request.User.Ext != nil {
		var userExt struct {
			Prebid struct {
				BuyerUIDs map[string]string `json:"buyeruids"`
			} `json:"prebid"`
		}
		if err := json.Unmarshal(request.User.Ext, &userExt); err == nil {
			if uid := userExt.Prebid.BuyerUIDs[BidderName]; uid != "" {
				return uid
			}
		}
	}

	return ""
}

// Direct event URL generation without config struct
func (a *adapter) generateEventURLs(ctx *bidProcessingContext, version string, customerID string, bidID string, impID string, price float64, traceID string, random string, width, height int, responseNURL string, responseBURL string, bidderCode string, currency string, placementId string, creativeId string) (nurl string, burl string) {
	baseURL := strings.TrimSuffix(a.monitoringEndpoint, "/") + fmt.Sprintf("/%s/prebid/%s/", version, customerID)

	// Common query parameters
	commonParams := fmt.Sprintf("?b=%s&a=%s&%s=%s&impId=%s&price=%.2f&traceId=%s&random=%s&domain=%s&adRequestId=%s&w=%d&h=%d&f=b&cur=%s&pId=%s&crId=%s",
		bidID, customerID, FieldBidder, bidderCode, impID, price, traceID, random, ctx.domain, ctx.adRequestID, width, height, currency, url.QueryEscape(placementId), url.QueryEscape(creativeId))

	nurl = baseURL + "pbs-impression" + commonParams
	burl = baseURL + "pbs-billing" + commonParams

	// Add response NURL/BURL as 'r' parameter if present
	if responseNURL != "" {
		nurl += "&r=" + url.QueryEscape(responseNURL)
	}
	if responseBURL != "" {
		burl += "&r=" + url.QueryEscape(responseBURL)
	}

	return nurl, burl
}
