package improvedigital

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/errortypes"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

const (
	isRewardedInventory              = "is_rewarded_inventory"
	stateRewardedInventoryEnable     = "1"
	consentProvidersSettingsInputKey = "ConsentedProvidersSettings"
	consentProvidersSettingsOutKey   = "consented_providers_settings"
	consentedProvidersKey            = "consented_providers"
	publisherEndpointParam           = "{PublisherId}"
)

type ImprovedigitalAdapter struct {
	endpoint string
}

// BidExt represents Improved Digital bid extension with line item ID and buying type values
type BidExt struct {
	Improvedigital struct {
		LineItemID int    `json:"line_item_id"`
		BuyingType string `json:"buying_type"`
	}
}

// ImpExtBidder represents Improved Digital bid extension with Publisher ID
type ImpExtBidder struct {
	Bidder struct {
		PublisherID int `json:"publisherId"`
	}
}

var dealDetectionRegEx = regexp.MustCompile("(classic|deal)")

// MakeRequests makes the HTTP requests which should be made to fetch bids.
func (a *ImprovedigitalAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	numRequests := len(request.Imp)
	errors := make([]error, 0)
	adapterRequests := make([]*adapters.RequestData, 0, numRequests)

	// Split multi-imp request into multiple ad server requests. SRA is currently not recommended.
	for i := 0; i < numRequests; i++ {
		if adapterReq, err := a.makeRequest(*request, request.Imp[i]); err == nil {
			adapterRequests = append(adapterRequests, adapterReq)
		} else {
			errors = append(errors, err)
		}
	}

	return adapterRequests, errors
}

func (a *ImprovedigitalAdapter) makeRequest(request openrtb2.BidRequest, imp openrtb2.Imp) (*adapters.RequestData, error) {
	// Handle Rewarded Inventory
	impExt, err := getImpExtWithRewardedInventory(imp)
	if err != nil {
		return nil, err
	}
	if impExt != nil {
		imp.Ext = impExt
	}

	request.Imp = []openrtb2.Imp{imp}

	userExtAddtlConsent, err := a.getAdditionalConsentProvidersUserExt(request)
	if err != nil {
		return nil, err
	}

	if len(userExtAddtlConsent) > 0 {
		userCopy := *request.User
		userCopy.Ext = userExtAddtlConsent
		request.User = &userCopy
	}

	reqJSON, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	return &adapters.RequestData{
		Method:  "POST",
		Uri:     a.buildEndpointURL(imp),
		Body:    reqJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}, nil
}

// MakeBids unpacks the server's response into Bids.
func (a *ImprovedigitalAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var bidResp openrtb2.BidResponse
	var impMap = make(map[string]openrtb2.Imp)
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	if len(bidResp.SeatBid) == 0 {
		return nil, nil
	}

	if len(bidResp.SeatBid) > 1 {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected SeatBid! Must be only one but have: %d", len(bidResp.SeatBid)),
		}}
	}

	seatBid := bidResp.SeatBid[0]
	if len(seatBid.Bid) == 0 {
		return nil, nil
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(seatBid.Bid))
	bidResponse.Currency = bidResp.Cur

	for i := range internalRequest.Imp {
		impMap[internalRequest.Imp[i].ID] = internalRequest.Imp[i]
	}

	for i := range seatBid.Bid {
		bid := seatBid.Bid[i]

		bidType, err := getBidType(bid, impMap)
		if err != nil {
			return nil, []error{err}
		}

		if bid.Ext != nil {
			var bidExt BidExt
			err = json.Unmarshal(bid.Ext, &bidExt)
			if err != nil {
				return nil, []error{err}
			}

			bidExtImprovedigital := bidExt.Improvedigital
			if bidExtImprovedigital.LineItemID != 0 && dealDetectionRegEx.MatchString(bidExtImprovedigital.BuyingType) {
				bid.DealID = strconv.Itoa(bidExtImprovedigital.LineItemID)
			}
		}

		bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
			Bid:     &bid,
			BidType: bidType,
		})
	}
	return bidResponse, nil
}

// Builder builds a new instance of the Improvedigital adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &ImprovedigitalAdapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func getBidType(bid openrtb2.Bid, impMap map[string]openrtb2.Imp) (openrtb_ext.BidType, error) {
	// there must be a matching imp against bid.ImpID
	imp, found := impMap[bid.ImpID]
	if !found {
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Failed to find impression for ID: \"%s\"", bid.ImpID),
		}
	}

	// if MType is not set in server response, try to determine it
	if bid.MType == 0 {
		if !isMultiFormatImp(imp) {
			// Not a bid for multi format impression. So, determine MType from impression
			if imp.Banner != nil {
				bid.MType = openrtb2.MarkupBanner
			} else if imp.Video != nil {
				bid.MType = openrtb2.MarkupVideo
			} else if imp.Audio != nil {
				bid.MType = openrtb2.MarkupAudio
			} else if imp.Native != nil {
				bid.MType = openrtb2.MarkupNative
			} else { // This should not happen.
				// Let's handle it just in case by returning an error.
				return "", &errortypes.BadServerResponse{
					Message: fmt.Sprintf("Could not determine MType from impression with ID: \"%s\"", bid.ImpID),
				}
			}
		} else {
			return "", &errortypes.BadServerResponse{
				Message: fmt.Sprintf("Bid must have non-zero MType for multi format impression with ID: \"%s\"", bid.ImpID),
			}
		}
	}

	// map MType to BidType
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupAudio:
		return openrtb_ext.BidTypeAudio, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	default:
		// This shouldn't happen. Let's handle it just in case by returning an error.
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unsupported MType %d for impression with ID: \"%s\"", bid.MType, bid.ImpID),
		}
	}
}

func isMultiFormatImp(imp openrtb2.Imp) bool {
	formatCount := 0
	if imp.Banner != nil {
		formatCount++
	}
	if imp.Video != nil {
		formatCount++
	}
	if imp.Audio != nil {
		formatCount++
	}
	if imp.Native != nil {
		formatCount++
	}
	return formatCount > 1
}

// This method responsible to clone request and convert additional consent providers string to array when additional consent provider found
func (a *ImprovedigitalAdapter) getAdditionalConsentProvidersUserExt(request openrtb2.BidRequest) ([]byte, error) {
	var cpStr string

	// If user/user.ext not defined, no need to parse additional consent
	if request.User == nil || request.User.Ext == nil {
		return nil, nil
	}

	// Start validating additional consent
	// Check key exist user.ext.ConsentedProvidersSettings
	var userExtMap = make(map[string]json.RawMessage)
	if err := json.Unmarshal(request.User.Ext, &userExtMap); err != nil {
		return nil, err
	}

	cpsMapValue, cpsJSONFound := userExtMap[consentProvidersSettingsInputKey]
	if !cpsJSONFound {
		return nil, nil
	}

	// Check key exist user.ext.ConsentedProvidersSettings.consented_providers
	var cpMap = make(map[string]json.RawMessage)
	if err := json.Unmarshal(cpsMapValue, &cpMap); err != nil {
		return nil, err
	}

	cpMapValue, cpJSONFound := cpMap[consentedProvidersKey]
	if !cpJSONFound {
		return nil, nil
	}
	// End validating additional consent

	// Trim enclosing quotes after casting json.RawMessage to string
	consentStr := strings.Trim((string)(cpMapValue), "\"")
	// Split by ~ and take only the second string (if exists) as the consented providers spec
	var consentStrParts = strings.Split(consentStr, "~")
	if len(consentStrParts) < 2 {
		return nil, nil
	}
	cpStr = strings.TrimSpace(consentStrParts[1])
	if len(cpStr) == 0 {
		return nil, nil
	}

	// Prepare consent providers string
	cpStr = fmt.Sprintf("[%s]", strings.Replace(cpStr, ".", ",", -1))
	cpMap[consentedProvidersKey] = json.RawMessage(cpStr)

	cpJSON, err := json.Marshal(cpMap)
	if err != nil {
		return nil, err
	}
	userExtMap[consentProvidersSettingsOutKey] = cpJSON

	extJson, err := json.Marshal(userExtMap)
	if err != nil {
		return nil, err
	}

	return extJson, nil
}

func getImpExtWithRewardedInventory(imp openrtb2.Imp) ([]byte, error) {
	var ext = make(map[string]json.RawMessage)
	if err := json.Unmarshal(imp.Ext, &ext); err != nil {
		return nil, err
	}

	prebidJSONValue, prebidJSONFound := ext["prebid"]
	if !prebidJSONFound {
		return nil, nil
	}

	var prebidMap = make(map[string]json.RawMessage)
	if err := json.Unmarshal(prebidJSONValue, &prebidMap); err != nil {
		return nil, err
	}

	if rewardedInventory, foundRewardedInventory := prebidMap[isRewardedInventory]; foundRewardedInventory && string(rewardedInventory) == stateRewardedInventoryEnable {
		ext[isRewardedInventory] = json.RawMessage(`true`)
		impExt, err := json.Marshal(ext)
		if err != nil {
			return nil, err
		}

		return impExt, nil
	}

	return nil, nil
}

func (a *ImprovedigitalAdapter) buildEndpointURL(imp openrtb2.Imp) string {
	publisherEndpoint := ""
	var impBidder ImpExtBidder

	err := json.Unmarshal(imp.Ext, &impBidder)
	if err == nil && impBidder.Bidder.PublisherID != 0 {
		publisherEndpoint = strconv.Itoa(impBidder.Bidder.PublisherID) + "/"
	}

	return strings.Replace(a.endpoint, publisherEndpointParam, publisherEndpoint, -1)
}
