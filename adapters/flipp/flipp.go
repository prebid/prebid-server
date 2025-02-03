package flipp

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/prebid/go-gdpr/consentconstants"
	"github.com/prebid/go-gdpr/vendorconsent"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/util/uuidutil"
)

const (
	bannerType                  = "banner"
	inlineDivName               = "inline"
	flippBidder                 = "flipp"
	defaultCurrency             = "USD"
	defaultStandardHeight int64 = 2400
	defaultCompactHeight  int64 = 600
)

var (
	count          int64 = 1
	adTypes              = []int64{4309, 641}
	dtxTypes             = []int64{5061}
	flippExtParams openrtb_ext.ImpExtFlipp
	customDataKey  string
)

type adapter struct {
	endpoint      string
	uuidGenerator uuidutil.UUIDGenerator
}

var (
	errRequestEmpty = errors.New("adapterRequest is empty")
)

// Builder builds a new instance of the Flipp adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint:      config.Endpoint,
		uuidGenerator: uuidutil.UUIDRandomGenerator{},
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	adapterRequests := make([]*adapters.RequestData, 0, len(request.Imp))
	var errors []error

	for _, imp := range request.Imp {
		adapterReq, err := a.processImp(request, imp)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		adapterRequests = append(adapterRequests, adapterReq)
	}
	if len(adapterRequests) == 0 {
		return nil, append(errors, errRequestEmpty)
	}
	return adapterRequests, errors
}

func (a *adapter) makeRequest(request *openrtb2.BidRequest, campaignRequestBody CampaignRequestBody, impID string) (*adapters.RequestData, error) {
	campaignRequestBodyJSON, err := json.Marshal(campaignRequestBody)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json")
	if request.Device != nil && request.Device.UA != "" {
		headers.Add("User-Agent", request.Device.UA)
	}
	return &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     a.endpoint,
		Body:    campaignRequestBodyJSON,
		Headers: headers,
		ImpIDs:  []string{impID},
	}, err
}

func (a *adapter) processImp(request *openrtb2.BidRequest, imp openrtb2.Imp) (*adapters.RequestData, error) {
	var flippExtParams openrtb_ext.ImpExtFlipp
	params, _, _, err := jsonparser.Get(imp.Ext, "bidder")
	if err != nil {
		return nil, fmt.Errorf("flipp params not found. %v", err)
	}
	err = jsonutil.Unmarshal(params, &flippExtParams)
	if err != nil {
		return nil, fmt.Errorf("unable to extract flipp params. %v", err)
	}

	publisherUrl, err := url.Parse(request.Site.Page)
	if err != nil {
		return nil, fmt.Errorf("unable to parse site url. %v", err)
	}

	var contentCode string
	if flippExtParams.Options.ContentCode != "" {
		contentCode = flippExtParams.Options.ContentCode
	} else if publisherUrl != nil {
		contentCode = publisherUrl.Query().Get("flipp-content-code")
	}

	placement := Placement{
		DivName: inlineDivName,
		SiteID:  &flippExtParams.SiteID,
		AdTypes: getAdTypes(flippExtParams.CreativeType),
		ZoneIds: flippExtParams.ZoneIds,
		Count:   &count,
		Prebid:  buildPrebidRequest(flippExtParams, request, imp),
		Properties: &Properties{
			ContentCode: &contentCode,
		},
		Options: flippExtParams.Options,
	}

	var userIP string
	if request.Device != nil && request.Device.IP != "" {
		userIP = request.Device.IP
	} else {
		return nil, fmt.Errorf("no IP set in flipp bidder params or request device")
	}

	var userKey string
	if request.User != nil && request.User.ID != "" {
		userKey = request.User.ID
	} else if flippExtParams.UserKey != "" && paramsUserKeyPermitted(request) {
		userKey = flippExtParams.UserKey
	} else {
		uid, err := a.uuidGenerator.Generate()
		if err != nil {
			return nil, fmt.Errorf("unable to generate user uuid. %v", err)
		}
		userKey = uid
	}

	keywordsArray := strings.Split(request.Site.Keywords, ",")

	campaignRequestBody := CampaignRequestBody{
		Placements: []*Placement{&placement},
		URL:        request.Site.Page,
		Keywords:   keywordsArray,
		IP:         userIP,
		User: &CampaignRequestBodyUser{
			Key: &userKey,
		},
	}

	adapterReq, err := a.makeRequest(request, campaignRequestBody, imp.ID)
	if err != nil {
		return nil, fmt.Errorf("make request failed with err %v", err)
	}

	return adapterReq, nil
}

func buildPrebidRequest(flippExtParams openrtb_ext.ImpExtFlipp, request *openrtb2.BidRequest, imp openrtb2.Imp) *PrebidRequest {
	var height int64
	var width int64
	if imp.Banner != nil && len(imp.Banner.Format) > 0 {
		height = imp.Banner.Format[0].H
		width = imp.Banner.Format[0].W
	}
	prebidRequest := PrebidRequest{
		CreativeType:            &flippExtParams.CreativeType,
		PublisherNameIdentifier: &flippExtParams.PublisherNameIdentifier,
		RequestID:               &imp.ID,
		Height:                  &height,
		Width:                   &width,
	}
	return &prebidRequest
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}

	var campaignResponseBody CampaignResponseBody
	if err := jsonutil.Unmarshal(responseData.Body, &campaignResponseBody); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = defaultCurrency
	for _, imp := range request.Imp {
		params, _, _, err := jsonparser.Get(imp.Ext, "bidder")
		if err != nil {
			return nil, []error{fmt.Errorf("flipp params not found. %v", err)}
		}
		err = jsonutil.Unmarshal(params, &flippExtParams)
		if err != nil {
			return nil, []error{fmt.Errorf("unable to extract flipp params. %v", err)}
		}
		for _, decision := range campaignResponseBody.Decisions.Inline {
			if *decision.Prebid.RequestID == imp.ID {
				b := &adapters.TypedBid{
					Bid:     buildBid(decision, imp.ID, flippExtParams),
					BidType: openrtb_ext.BidType(bannerType),
				}
				bidResponse.Bids = append(bidResponse.Bids, b)
			}
		}
	}
	return bidResponse, nil
}

func getAdTypes(creativeType string) []int64 {
	if creativeType == "DTX" {
		return dtxTypes
	}
	return adTypes
}

func buildBid(decision *InlineModel, impId string, flippExtParams openrtb_ext.ImpExtFlipp) *openrtb2.Bid {
	bid := &openrtb2.Bid{
		CrID:  fmt.Sprint(decision.CreativeID),
		Price: *decision.Prebid.Cpm,
		AdM:   *decision.Prebid.Creative,
		ID:    fmt.Sprint(decision.AdID),
		ImpID: impId,
	}
	if len(decision.Contents) > 0 || decision.Contents[0] != nil || decision.Contents[0].Data != nil {
		if decision.Contents[0].Data.Width != 0 {
			bid.W = decision.Contents[0].Data.Width
		}

		if flippExtParams.Options.StartCompact {
			bid.H = defaultCompactHeight
		} else {
			bid.H = defaultStandardHeight
		}

		if customDataInterface := decision.Contents[0].Data.CustomData; customDataInterface != nil {
			if customDataMap, ok := customDataInterface.(map[string]interface{}); ok {
				customDataKey := "standardHeight"
				if flippExtParams.Options.StartCompact {
					customDataKey = "compactHeight"
				}

				if value, exists := customDataMap[customDataKey]; exists {
					if floatVal, ok := value.(float64); ok {
						bid.H = int64(floatVal)
					}
				}
			}
		}
	}
	return bid
}

func paramsUserKeyPermitted(request *openrtb2.BidRequest) bool {
	if request.Regs != nil {
		if request.Regs.COPPA == 1 {
			return false
		}
		if request.Regs.GDPR != nil && *request.Regs.GDPR == 1 {
			return false
		}
	}
	if request.Ext != nil {
		var extData struct {
			TransmitEids *bool `json:"transmitEids,omitempty"`
		}
		if err := jsonutil.Unmarshal(request.Ext, &extData); err == nil {
			if extData.TransmitEids != nil && !*extData.TransmitEids {
				return false
			}
		}
	}
	if request.User != nil && request.User.Consent != "" {
		data, err := base64.RawURLEncoding.DecodeString(request.User.Consent)
		if err != nil {
			return true
		}
		consent, err := vendorconsent.Parse(data)
		if err != nil {
			return true
		}
		if !consent.PurposeAllowed(consentconstants.ContentSelectionDeliveryReporting) {
			return false
		}
	}
	return true
}
