package missena

import (
	"fmt"
	"net/http"
	"net/url"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/version"
)

type adapter struct {
	EndpointTemplate *template.Template
}

type MissenaAdRequest struct {
	Adunit         string               `json:"adunit,omitempty"`
	BuyerUID       string               `json:"buyeruid,omitempty"`
	Currency       string               `json:"currency,omitempty"`
	EIDs           []openrtb2.EID       `json:"userEids,omitempty"`
	Floor          float64              `json:"floor,omitempty"`
	FloorCurrency  string               `json:"floor_currency,omitempty"`
	IdempotencyKey string               `json:"ik,omitempty"`
	RequestID      string               `json:"request_id,omitempty"`
	Timeout        int64                `json:"timeout,omitempty"`
	UserParams     UserParams           `json:"params"`
	ORTB2          *openrtb2.BidRequest `json:"ortb2"`
	Version        string               `json:"version,omitempty"`
}

type BidServerResponse struct {
	Ad        string  `json:"ad"`
	Cpm       float64 `json:"cpm"`
	Currency  string  `json:"currency"`
	RequestID string  `json:"requestId"`
}

type UserParams struct {
	Formats   []string       `json:"formats,omitempty"`
	Placement string         `json:"placement,omitempty" default:"sticky"`
	TestMode  string         `json:"test,omitempty"`
	Settings  map[string]any `json:"settings,omitempty"`
}

type MissenaAdapter struct {
	EndpointTemplate *template.Template
}

var defaultCur = "USD"

// Builder builds a new instance of the Foo adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	endpoint, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}
	bidder := &adapter{
		EndpointTemplate: endpoint,
	}
	return bidder, nil
}

func getCurrency(currencies []string) (string, error) {
	eurAvailable := false
	for _, cur := range currencies {
		if cur == defaultCur {
			return defaultCur, nil
		}
		if cur == "EUR" {
			eurAvailable = true
		}
	}
	if eurAvailable {
		return "EUR", nil
	}
	return "", fmt.Errorf("no currency supported %v", currencies)
}

func (a *adapter) getEndPoint(ext *openrtb_ext.ExtImpMissena) (string, error) {
	endPointParams := macros.EndpointTemplateParams{
		PublisherID: url.PathEscape(ext.APIKey),
	}
	return macros.ResolveMacros(a.EndpointTemplate, endPointParams)
}

func (a *adapter) makeRequest(imp openrtb2.Imp, request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo, params *openrtb_ext.ExtImpMissena, gdprApplies bool, consentString string) (*adapters.RequestData, error) {
	endpointURL, err := a.getEndPoint(params)
	if err != nil {
		return nil, err
	}
	cur, err := getCurrency(request.Cur)
	if err != nil {
		cur = defaultCur
	}

	var floor float64
	var floorCur string
	if imp.BidFloor != 0 {
		floor = imp.BidFloor
		floorCur, err = getCurrency(request.Cur)
		if err != nil {
			floorCur = defaultCur
			floor, err = requestInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, floorCur)
			if err != nil {
				return nil, err
			}
		}
	}

	missenaRequest := MissenaAdRequest{
		Adunit:         imp.ID,
		Currency:       cur,
		Floor:          floor,
		FloorCurrency:  floorCur,
		IdempotencyKey: request.ID,
		ORTB2:          request,
		RequestID:      request.ID,
		Timeout:        request.TMax,
		UserParams: UserParams{
			Formats:   params.Formats,
			Placement: params.Placement,
			TestMode:  params.TestMode,
			Settings:  params.Settings,
		},
		Version: version.Ver,
	}

	body, err := jsonutil.Marshal(missenaRequest)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	if request.Device != nil {
		headers.Add("User-Agent", request.Device.UA)
		if request.Device.IP != "" {
			headers.Add("X-Forwarded-For", request.Device.IP)
		} else if request.Device.IPv6 != "" {
			headers.Add("X-Forwarded-For", request.Device.IPv6)
		}
	}
	if request.Site != nil && request.Site.Page != "" {
		headers.Add("Referer", request.Site.Page)
		pageURL, err := url.Parse(request.Site.Page)
		if err == nil {
			origin := fmt.Sprintf("%s://%s", pageURL.Scheme, pageURL.Host)
			headers.Add("Origin", origin)
		}
	}

	return &adapters.RequestData{
		Method:  "POST",
		Uri:     endpointURL,
		Headers: headers,
		Body:    body,
		ImpIDs:  []string{imp.ID},
	}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var httpRequests []*adapters.RequestData
	var errors []error
	gdprApplies, consentString := readGDPR(request)

	for _, imp := range request.Imp {
		var bidderExt adapters.ExtImpBidder
		if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: fmt.Sprintf("Error parsing bidderExt object: %v, input: %s", err, string(imp.Ext)),
			})
			continue
		}

		var missenaExt *openrtb_ext.ExtImpMissena
		if err := jsonutil.Unmarshal(bidderExt.Bidder, &missenaExt); err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: "Error parsing missenaExt parameters",
			})
			continue
		}

		newHttpRequest, err := a.makeRequest(imp, request, requestInfo, missenaExt, gdprApplies, consentString)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		httpRequests = append(httpRequests, newHttpRequest)
		// We only support one impression per request
		// So return on the first working one
		break
	}

	return httpRequests, errors
}

func readGDPR(request *openrtb2.BidRequest) (bool, string) {
	consentString := ""
	if request.User != nil {
		var extUser openrtb_ext.ExtUser
		if err := jsonutil.Unmarshal(request.User.Ext, &extUser); err == nil {
			consentString = extUser.Consent
		}
	}
	gdprApplies := false
	var extRegs openrtb_ext.ExtRegs
	if request.Regs != nil {
		if err := jsonutil.Unmarshal(request.Regs.Ext, &extRegs); err == nil {
			if extRegs.GDPR != nil {
				gdprApplies = (*extRegs.GDPR == 1)
			}
		}
	}
	return gdprApplies, consentString
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode == http.StatusBadRequest {
		err := &errortypes.BadInput{
			Message: "Unexpected status code: 400. Bad request from publisher. Run with request.debug = 1 for more info.",
		}
		return nil, []error{err}
	}

	if responseData.StatusCode != http.StatusOK {
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info.", responseData.StatusCode),
		}
		return nil, []error{err}
	}

	var missenaResponse BidServerResponse
	if err := jsonutil.Unmarshal(responseData.Body, &missenaResponse); err != nil {
		return nil, []error{err}
	}

	bidRes := adapters.NewBidderResponseWithBidsCapacity(1)
	bidRes.Currency = missenaResponse.Currency

	responseBid := &openrtb2.Bid{
		ID:    request.ID,
		Price: float64(missenaResponse.Cpm),
		ImpID: request.Imp[0].ID,
		AdM:   missenaResponse.Ad,
		CrID:  missenaResponse.RequestID,
	}

	b := &adapters.TypedBid{
		Bid:     responseBid,
		BidType: openrtb_ext.BidTypeBanner,
	}

	bidRes.Bids = append(bidRes.Bids, b)

	return bidRes, nil
}
