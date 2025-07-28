package contxtful

import (
	"fmt"
	"net/http"
	"strings"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

// BidExtensionsParams holds the parameters needed to create bid extensions
type BidExtensionsParams struct {
	Price    float64
	Currency string
	BidType  string
	Width    int
	Height   int
}

// ContxtfulRequestPayload represents the complete request payload structure
type ContxtfulRequestPayload struct {
	ORTB2         *openrtb2.BidRequest   `json:"ortb2"`
	BidRequests   []ContxtfulBidRequest  `json:"bidRequests"`
	BidderRequest ContxtfulBidderRequest `json:"bidderRequest"`
	Config        ContxtfulConfig        `json:"config"`
}

// ContxtfulBidRequest represents an individual bid request
type ContxtfulBidRequest struct {
	Bidder string                    `json:"bidder"`
	Params ContxtfulBidRequestParams `json:"params"`
	BidID  string                    `json:"bidId"`
}

// ContxtfulBidRequestParams represents the parameters for a bid request
type ContxtfulBidRequestParams struct {
	PlacementID string `json:"placementId"`
}

// ContxtfulBidderRequest represents the bidder request information
type ContxtfulBidderRequest struct {
	BidderCode string `json:"bidderCode"`
}

// ContxtfulConfig represents the configuration section
type ContxtfulConfig struct {
	Contxtful ContxtfulConfigDetails `json:"contxtful"`
}

// ContxtfulConfigDetails represents the detailed configuration
type ContxtfulConfigDetails struct {
	Version  string `json:"version"`
	Customer string `json:"customer"`
}

// BidExtensions represents the complete bid extensions structure
type BidExtensions struct {
	OrigBidCPM float64         `json:"origbidcpm"`
	OrigBidCur string          `json:"origbidcur"`
	Prebid     PrebidExtension `json:"prebid"`
}

// PrebidExtension represents the prebid-specific extension data
type PrebidExtension struct {
	Type      string          `json:"type"`
	Meta      PrebidMeta      `json:"meta"`
	Targeting PrebidTargeting `json:"targeting"`
}

// PrebidMeta represents the meta information in prebid extensions
type PrebidMeta struct {
	AdapterCode string `json:"adaptercode"`
}

// PrebidTargeting represents the targeting information in prebid extensions
type PrebidTargeting struct {
	HBBidder string `json:"hb_bidder"`
	HBPB     string `json:"hb_pb"`
	HBSize   string `json:"hb_size"`
}

type ContxtfulExt struct {
	Reseller string `json:"reseller,omitempty"`
}

type ContxtfulExchangeBid struct {
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
	LURL        string       `json:"lurl,omitempty"`
	Ext         ContxtfulExt `json:"ext,omitempty"`
}

type BidProcessingContext struct {
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

type adapter struct {
	endpointTemplate *template.Template
}

// RequestConfig holds extracted configuration from various sources
type RequestConfig struct {
	Version    string
	CustomerID string
	Source     string // "uri", "bidder_config", or "impression"
}

// Essential constants
const (
	BidderName     = "contxtful"
	DefaultVersion = "v1"
	FieldBidder    = "bidder"
	PbsPath        = "/pbs/"
)

// Helper to safely extract bidder parameters from impression extension
func extractBidderParams(impExt jsonutil.RawMessage) (*openrtb_ext.ExtImpContxtful, error) {
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

// Builder builds a new instance of the Contxtful adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	return &adapter{
		endpointTemplate: template.Must(template.New("endpointTemplate").Parse(config.Endpoint)),
	}, nil
}

// Extract impression parameters (validation handled by JSON schema)
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

		if customerId == "" {
			customerId = contxtfulParams.CustomerId
		}
		validPlacements = append(validPlacements, contxtfulParams.PlacementId)
	}

	return validPlacements, customerId, errors
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
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

	// Build dynamic endpoint URL and validate impressions
	endpoint, validPlacements, customerId, endpointErrors := a.buildEndpointURL(request)
	errors = append(errors, endpointErrors...)

	if len(validPlacements) == 0 {
		return nil, errors
	}

	// Extract bidder config for payload creation
	bidderCustomerId, bidderVersion := extractBidderConfig(request)

	// Create payload
	payload := createRequestPayload(request, validPlacements, bidderCustomerId, bidderVersion, customerId)

	requestJSON, err := jsonutil.Marshal(payload)
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	requestData := &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     endpoint,
		Body:    requestJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}

	return []*adapters.RequestData{requestData}, errors
}

// buildEndpointURL validates impressions and creates the endpoint URL
func (a *adapter) buildEndpointURL(request *openrtb2.BidRequest) (string, []string, string, []error) {
	// Validate impressions and extract parameters
	validPlacements, customerId, validationErrors := validateImpressions(request)

	if len(validationErrors) > 0 {
		return "", validPlacements, customerId, validationErrors
	}

	// Extract bidder config
	bidderCustomerId, _ := extractBidderConfig(request)

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
		return "", validPlacements, customerId, []error{err}
	}

	return endpoint, validPlacements, customerId, nil
}

// Streamlined payload creation
func createRequestPayload(request *openrtb2.BidRequest, validPlacements []string, bidderCustomerId, bidderVersion, fallbackCustomerId string) ContxtfulRequestPayload {
	// Create clean request copy
	requestCopy := *request
	for i := range requestCopy.Imp {
		requestCopy.Imp[i].Ext = jsonutil.RawMessage(`{}`)
	}

	// Resolve configuration with fallbacks inline
	adapterVersion := bidderVersion
	if adapterVersion == "" {
		adapterVersion = DefaultVersion
	}

	customer := bidderCustomerId
	if customer == "" {
		customer = fallbackCustomerId
	}

	uid := extractUserIDForCookie(request)

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

	// Build bid requests
	bidRequests := make([]ContxtfulBidRequest, len(request.Imp))
	for i, imp := range request.Imp {
		bidRequests[i] = ContxtfulBidRequest{
			Bidder: BidderName,
			Params: ContxtfulBidRequestParams{
				PlacementID: validPlacements[i],
			},
			BidID: imp.ID,
		}
	}

	return ContxtfulRequestPayload{
		ORTB2:       &requestCopy,
		BidRequests: bidRequests,
		BidderRequest: ContxtfulBidderRequest{
			BidderCode: BidderName,
		},
		Config: ContxtfulConfig{
			Contxtful: ContxtfulConfigDetails{
				Version:  adapterVersion,
				Customer: customer,
			},
		},
	}
}

// Response format handlers
// Direct response processing without over-engineered handler pattern
func (a *adapter) processResponse(responseBody []byte, ctx *BidProcessingContext) bool {
	// Extract configuration from various sources with priority
	config, err := extractRequestConfig(ctx.requestData, ctx.request, ctx.bidderCustomerId)
	if err != nil {
		ctx.errors = append(ctx.errors, &errortypes.BadInput{Message: err.Error()})
		return true
	}

	// Store configuration in context for use in bid creation
	ctx.version = config.Version

	// Try PrebidJS format first
	var prebidBids []ContxtfulExchangeBid
	if err := jsonutil.Unmarshal(responseBody, &prebidBids); err == nil && len(prebidBids) > 0 &&
		prebidBids[0].CPM > 0 && prebidBids[0].RequestID != "" && prebidBids[0].Ad != "" {
		return a.processPrebidJSBids(prebidBids, ctx, config.CustomerID)
	}

	// Handle trace format (acknowledges response, no bids)
	var traceData []map[string]interface{}
	if err := jsonutil.Unmarshal(responseBody, &traceData); err == nil && len(traceData) > 0 {
		if _, exists := traceData[0]["traceId"]; exists {
			return true // Trace acknowledged
		}
	}

	return false
}

func (a *adapter) processPrebidJSBids(prebidBids []ContxtfulExchangeBid, ctx *BidProcessingContext, customerId string) bool {
	for _, prebidBid := range prebidBids {
		if prebidBid.CPM == 0 || prebidBid.RequestID == "" {
			continue
		}

		currency := prebidBid.Currency
		if currency == "" {
			currency = "USD"
		}

		a.createBid(prebidBid, currency, ctx, customerId)

		if prebidBid.Currency != "" {
			ctx.bidderResponse.Currency = prebidBid.Currency
		}
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

	ctx := &BidProcessingContext{
		request:          request,
		requestData:      requestData,
		bidderCustomerId: bidderCustomerId,
		bidderVersion:    bidderVersion,
		bidderResponse:   adapters.NewBidderResponse(),
		domain:           domain,
		adRequestID:      request.ID,
	}

	if a.processResponse(response.Body, ctx) {
		return ctx.bidderResponse, ctx.errors
	}

	return nil, []error{fmt.Errorf("failed to parse response as Contxtful relay or Prebid.js format")}
}

// Simplified bid creation using prebid bid struct
func (a *adapter) createBid(
	prebidBid ContxtfulExchangeBid,
	currency string,
	ctx *BidProcessingContext,
	customerId string,
) {
	// Determine media type from impression
	var bidType openrtb_ext.BidType = openrtb_ext.BidTypeBanner
	for _, imp := range ctx.request.Imp {
		if imp.ID == prebidBid.RequestID {
			if imp.Video != nil {
				bidType = openrtb_ext.BidTypeVideo
			} else if imp.Native != nil {
				bidType = openrtb_ext.BidTypeNative
			}
			break
		}
	}

	bid := &openrtb2.Bid{
		ID:    fmt.Sprintf("%s-%s", BidderName, prebidBid.RequestID),
		ImpID: prebidBid.RequestID,
		Price: prebidBid.CPM,
		AdM:   prebidBid.Ad,
		W:     int64(prebidBid.Width),
		H:     int64(prebidBid.Height),
		CrID:  prebidBid.CreativeID,
		NURL:  prebidBid.NURL,
		BURL:  prebidBid.BURL,
		LURL:  prebidBid.LURL,
	}

	if bidExtJSON, err := createBidExtensions(BidExtensionsParams{
		Price:    prebidBid.CPM,
		Currency: currency,
		BidType:  string(bidType),
		Width:    prebidBid.Width,
		Height:   prebidBid.Height,
	}); err == nil {
		bid.Ext = bidExtJSON
	}

	typedBid := &adapters.TypedBid{
		Bid:     bid,
		BidType: bidType,
	}

	ctx.bidderResponse.Bids = append(ctx.bidderResponse.Bids, typedBid)
}

func createBidExtensions(params BidExtensionsParams) (jsonutil.RawMessage, error) {
	bidExt := BidExtensions{
		OrigBidCPM: params.Price,
		OrigBidCur: params.Currency,
		Prebid: PrebidExtension{
			Type: params.BidType,
			Meta: PrebidMeta{
				AdapterCode: BidderName,
			},
			Targeting: PrebidTargeting{
				HBBidder: BidderName,
				HBPB:     fmt.Sprintf("%.2f", params.Price),
				HBSize:   fmt.Sprintf("%dx%d", params.Width, params.Height),
			},
		},
	}
	return jsonutil.Marshal(bidExt)
}

// Simple unified bidder config extraction - get ORTB2 data and extract params
func extractBidderConfig(request *openrtb2.BidRequest) (string, string) {
	if request == nil || request.Ext == nil {
		return "", ""
	}

	var requestExt openrtb_ext.ExtRequest

	if err := jsonutil.Unmarshal(request.Ext, &requestExt); err != nil {
		return "", ""
	}

	// Find contxtful bidder config and extract params from ORTB2 data
	for _, config := range requestExt.Prebid.BidderConfigs {
		for _, bidder := range config.Bidders {
			if bidder == BidderName && config.Config.ORTB2 != nil {
				return extractContxtfulParams(config.Config.ORTB2)
			}
		}
	}
	return "", ""
}

// Extract contxtful params from any ORTB2 data (unified for both bidder config and other sources)
func extractContxtfulParams(ortb2Data *openrtb_ext.ORTB2) (string, string) {
	var ortb2UserData []struct {
		Name string `json:"name"`
		Ext  struct {
			Params struct {
				CI string `json:"ci"` // Customer ID
				EV string `json:"ev"` // Version
			} `json:"params"`
		} `json:"ext"`
	}

	if err := jsonutil.Unmarshal(ortb2Data.User, &ortb2UserData); err != nil {
		return "", ""
	}

	// Find contxtful params
	for _, data := range ortb2UserData {
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

// extractFromURL parses Contxtful URL pattern: /{version}/pbs/{customerId}/bid
func extractFromURL(uri string) (version, customerID string) {
	idx := strings.Index(uri, PbsPath)
	if idx == -1 {
		return "", ""
	}

	// Extract version from before /prebid/
	beforePrebid := uri[:idx]
	if lastSlash := strings.LastIndex(beforePrebid, "/"); lastSlash != -1 {
		version = beforePrebid[lastSlash+1:]
	}

	// Extract customer ID from after /prebid/
	remaining := uri[idx+len(PbsPath):]
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
		var userExt openrtb_ext.ExtUser
		if err := jsonutil.Unmarshal(request.User.Ext, &userExt); err == nil {
			if userExt.Prebid != nil && userExt.Prebid.BuyerUIDs != nil {
				if uid := userExt.Prebid.BuyerUIDs[BidderName]; uid != "" {
					return uid
				}
			}
		}
	}

	return ""
}
