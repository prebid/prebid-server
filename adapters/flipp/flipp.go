package flipp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/gofrs/uuid"
	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const (
	bannerType      = "banner"
	inlineDivName   = "inline"
	flippBidder     = "flipp"
	defaultCurrency = "USD"
)

var (
	count    int64 = 1
	adTypes        = []int64{4309, 641}
	dtxTypes       = []int64{5061}
)

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the Flipp adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
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
		return nil, append(errors, fmt.Errorf("adapterRequest is empty"))
	}
	return adapterRequests, errors
}

func (a *adapter) makeRequest(request *openrtb2.BidRequest, campaignRequestBody CampaignRequestBody) (*adapters.RequestData, error) {
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
	}, err
}

func (a *adapter) processImp(request *openrtb2.BidRequest, imp openrtb2.Imp) (*adapters.RequestData, error) {
	var flippExtParams openrtb_ext.ImpExtFlipp
	params, _, _, err := jsonparser.Get(imp.Ext, "bidder")
	if err != nil {
		return nil, fmt.Errorf("flipp params not found. %v", err)
	}
	err = json.Unmarshal(params, &flippExtParams)
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
	if flippExtParams.IP != "" {
		userIP = flippExtParams.IP
	} else if request.Device != nil && request.Device.IP != "" {
		userIP = request.Device.IP
	} else {
		return nil, fmt.Errorf("no IP set in flipp bidder params or request device")
	}

	var userKey string
	if request.User != nil && request.User.ID != "" {
		userKey = request.User.ID
	} else if flippExtParams.UserKey != "" {
		userKey = flippExtParams.UserKey
	} else {
		uid, err := uuid.NewV4()
		if err != nil {
			return nil, fmt.Errorf("unable to generate user uuid. %v", err)
		}
		userKey = uid.String()
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

	adapterReq, err := a.makeRequest(request, campaignRequestBody)
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
	if err := json.Unmarshal(responseData.Body, &campaignResponseBody); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = defaultCurrency
	for _, imp := range request.Imp {
		for _, decision := range campaignResponseBody.Decisions.Inline {
			if *decision.Prebid.RequestID == imp.ID {
				b := &adapters.TypedBid{
					Bid:     buildBid(decision, imp.ID),
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

func buildBid(decision *InlineModel, impId string) *openrtb2.Bid {
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
		bid.H = 0
	}
	return bid
}
