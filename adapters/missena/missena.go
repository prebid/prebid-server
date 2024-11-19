package missena

import (
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/version"
)

type adapter struct {
	endpoint string
}

type MissenaAdRequest struct {
	Adunit           string                `json:"adunit,omitempty"`
	COPPA            int8                  `json:"coppa,omitempty"`
	Currency         string                `json:"currency,omitempty"`
	EIDs             []openrtb2.EID        `json:"userEids,omitempty"`
	Floor            float64               `json:"floor,omitempty"`
	FloorCurrency    string                `json:"floor_currency,omitempty"`
	GDPR             bool                  `json:"consent_required,omitempty"`
	GDPRConsent      string                `json:"consent_string,omitempty"`
	IdempotencyKey   string                `json:"ik,omitempty"`
	Referer          string                `json:"referer,omitempty"`
	RefererCanonical string                `json:"referer_canonical,omitempty"`
	RequestID        string                `json:"request_id,omitempty"`
	SChain           *openrtb2.SupplyChain `json:"schain,omitempty"`
	Timeout          int                   `json:"timeout,omitempty"`
	URL              string                `json:"url,omitempty"`
	UserParams       UserParams            `json:"params"`
	USPrivacy        string                `json:"us_privacy,omitempty"`
	Version          string                `json:"version,omitempty"`
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
	Sample    string         `json:"sample,omitempty"`
	Settings  map[string]any `json:"settings,omitempty"`
}

type InternalParams struct {
	APIKey           string
	Formats          []string
	GDPR             bool
	GDPRConsent      string
	Placement        string
	Referer          string
	RefererCanonical string
	RequestID        string
	Sample           string
	Settings         map[string]any
	Timeout          int
}

type MissenaAdapter struct {
	EndpointTemplate *template.Template
}

// Builder builds a new instance of the Foo adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func getCurrency(currencies []string) (string, error) {
	eurAvailable := false
	for _, cur := range currencies {
		if cur == "USD" {
			return "USD", nil
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

func (a *adapter) makeRequest(missenaParams InternalParams, _ *adapters.ExtraRequestInfo, imp openrtb2.Imp, request *openrtb2.BidRequest) (*adapters.RequestData, error) {
	url := a.endpoint + "?t=" + missenaParams.APIKey
	currency, err := getCurrency(request.Cur)
	if err != nil {
		// TODO: convert unsupported currency on response
		return nil, err
	}

	var schain *openrtb2.SupplyChain
	if request.Source != nil {
		schain = request.Source.SChain
	}

	missenaRequest := MissenaAdRequest{
		Adunit:           imp.ID,
		COPPA:            request.Regs.COPPA,
		Currency:         currency,
		EIDs:             request.User.EIDs,
		Floor:            imp.BidFloor,
		FloorCurrency:    imp.BidFloorCur,
		GDPR:             missenaParams.GDPR,
		GDPRConsent:      missenaParams.GDPRConsent,
		IdempotencyKey:   request.ID,
		Referer:          request.Site.Page,
		RefererCanonical: request.Site.Domain,
		RequestID:        request.ID,
		SChain:           schain,
		Timeout:          2000,
		UserParams: UserParams{
			Formats:   missenaParams.Formats,
			Placement: missenaParams.Placement,
			Settings:  missenaParams.Settings,
		},
		Version: version.Ver,
	}

	body, err := json.Marshal(missenaRequest)
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
	if request.Site != nil {
		headers.Add("Referer", request.Site.Page)
	}

	return &adapters.RequestData{
		Method:  "POST",
		Uri:     url,
		Headers: headers,
		Body:    body,
		ImpIDs:  []string{imp.ID},
	}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var httpRequests []*adapters.RequestData
	var errors []error
	gdprApplies, consentString := readGDPR(request)

	params := InternalParams{
		GDPR:        gdprApplies,
		GDPRConsent: consentString,
	}

	for _, imp := range request.Imp {
		var bidderExt adapters.ExtImpBidder
		if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: "Error parsing bidderExt object",
			})
			continue
		}

		var missenaExt openrtb_ext.ExtImpMissena
		if err := jsonutil.Unmarshal(bidderExt.Bidder, &missenaExt); err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: "Error parsing missenaExt parameters",
			})
			continue
		}

		params.APIKey = missenaExt.APIKey
		params.Placement = missenaExt.Placement
		params.Sample = missenaExt.Sample

		newHttpRequest, err := a.makeRequest(params, requestInfo, imp, request)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		httpRequests = append(httpRequests, newHttpRequest)
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
