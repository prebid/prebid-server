package contxtful

import (
	"fmt"
	"net/http"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

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
	OrigBidCPM float64                  `json:"origbidcpm"`
	OrigBidCur string                   `json:"origbidcur"`
	Prebid     openrtb_ext.ExtBidPrebid `json:"prebid"`
}

type ContxtfulExchangeBid struct {
	RequestID   string              `json:"requestId"`
	CPM         float64             `json:"cpm"`
	Currency    string              `json:"currency"`
	Width       int                 `json:"width"`
	Height      int                 `json:"height"`
	CreativeID  string              `json:"creativeId"`
	AdM         string              `json:"adm"`
	TTL         int                 `json:"ttl"`
	NetRevenue  bool                `json:"netRevenue"`
	MediaType   string              `json:"mediaType"`
	BidderCode  string              `json:"bidderCode"`
	PlacementID string              `json:"placementId"`
	TraceId     string              `json:"traceId,omitempty"`
	Random      float64             `json:"random,omitempty"`
	NURL        string              `json:"nurl"`
	BURL        string              `json:"burl"`
	LURL        string              `json:"lurl,omitempty"`
	Ext         jsonutil.RawMessage `json:"ext,omitempty"`
}

type BidProcessingContext struct {
	request        *openrtb2.BidRequest
	requestData    *adapters.RequestData
	customerId     string
	bidderResponse *adapters.BidderResponse
	errors         []error
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

	// Create payload
	payload := createRequestPayload(request, validPlacements, customerId)

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

	// Use bidder config customer as primary source for endpoint URL
	endpointCustomerId := customerId

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
func createRequestPayload(request *openrtb2.BidRequest, validPlacements []string, customerId string) ContxtfulRequestPayload {
	// Create clean request copy
	requestCopy := *request

	adapterVersion := DefaultVersion
	customer := customerId

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
	// Try PrebidJS format first
	var prebidBids []ContxtfulExchangeBid
	if err := jsonutil.Unmarshal(responseBody, &prebidBids); err == nil && len(prebidBids) > 0 {
		return a.processPrebidJSBids(prebidBids, ctx)
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

func (a *adapter) processPrebidJSBids(prebidBids []ContxtfulExchangeBid, ctx *BidProcessingContext) bool {
	for _, prebidBid := range prebidBids {
		if prebidBid.CPM == 0 || prebidBid.RequestID == "" {
			continue
		}
		if prebidBid.MediaType == "" {
			ctx.errors = append(ctx.errors, &errortypes.BadServerResponse{Message: "bid has no ad media type"})
			continue
		}
		if prebidBid.AdM == "" {
			ctx.errors = append(ctx.errors, &errortypes.BadServerResponse{Message: "bid has no ad markup"})
			continue
		}
		currency := prebidBid.Currency
		if currency == "" {
			currency = "USD"
		}

		a.createBid(prebidBid, currency, ctx)

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

	// Extract domain for event URLs
	domain := ""
	if request.Site != nil && request.Site.Domain != "" {
		domain = request.Site.Domain
	}

	ctx := &BidProcessingContext{
		request:        request,
		requestData:    requestData,
		bidderResponse: adapters.NewBidderResponse(),
		domain:         domain,
		adRequestID:    request.ID,
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
		AdM:   prebidBid.AdM,
		W:     int64(prebidBid.Width),
		H:     int64(prebidBid.Height),
		CrID:  prebidBid.CreativeID,
		NURL:  prebidBid.NURL,
		BURL:  prebidBid.BURL,
		LURL:  prebidBid.LURL,
		Ext:   prebidBid.Ext,
	}

	typedBid := &adapters.TypedBid{
		Bid:     bid,
		BidType: bidType,
	}

	ctx.bidderResponse.Bids = append(ctx.bidderResponse.Bids, typedBid)
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
