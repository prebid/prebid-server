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
)

type adapter struct {
	endpoint string
}

type MissenaAdRequest struct {
	RequestId        string `json:"request_id"`
	Timeout          int    `json:"timeout"`
	Referer          string `json:"referer"`
	RefererCanonical string `json:"referer_canonical"`
	GDPRConsent      string `json:"consent_string"`
	GDPR             bool   `json:"consent_required"`
	Placement        string `json:"placement"`
	TestMode         string `json:"test"`
}

type MissenaBidServerResponse struct {
	Ad        string  `json:"ad"`
	Cpm       float64 `json:"cpm"`
	Currency  string  `json:"currency"`
	RequestId string  `json:"requestId"`
}

type MissenaInternalParams struct {
	ApiKey           string
	RequestId        string
	Timeout          int
	Referer          string
	RefererCanonical string
	GDPRConsent      string
	GDPR             bool
	Placement        string
	TestMode         string
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

func (a *adapter) makeRequest(missenaParams MissenaInternalParams, reqInfo *adapters.ExtraRequestInfo, impID string, request *openrtb2.BidRequest) (*adapters.RequestData, error) {
	url := a.endpoint + "?t=" + missenaParams.ApiKey

	missenaRequest := MissenaAdRequest{
		RequestId:        request.ID,
		Timeout:          2000,
		Referer:          request.Site.Page,
		RefererCanonical: request.Site.Domain,
		GDPRConsent:      missenaParams.GDPRConsent,
		GDPR:             missenaParams.GDPR,
		Placement:        missenaParams.Placement,
		TestMode:         missenaParams.TestMode,
	}

	body, errm := json.Marshal(missenaRequest)
	if errm != nil {
		return nil, errm
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
		ImpIDs:  []string{impID},
	}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {

	var httpRequests []*adapters.RequestData
	var errors []error
	gdprApplies, consentString := readGDPR(request)

	missenaInternalParams := MissenaInternalParams{
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

		missenaInternalParams.ApiKey = missenaExt.ApiKey
		missenaInternalParams.Placement = missenaExt.Placement
		missenaInternalParams.TestMode = missenaExt.TestMode

		newHttpRequest, err := a.makeRequest(missenaInternalParams, requestInfo, imp.ID, request)
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

	var missenaResponse MissenaBidServerResponse
	if err := jsonutil.Unmarshal(responseData.Body, &missenaResponse); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)
	bidResponse.Currency = missenaResponse.Currency

	responseBid := &openrtb2.Bid{
		ID:    request.ID,
		Price: float64(missenaResponse.Cpm),
		ImpID: request.Imp[0].ID,
		AdM:   missenaResponse.Ad,
		CrID:  missenaResponse.RequestId,
	}

	b := &adapters.TypedBid{
		Bid:     responseBid,
		BidType: openrtb_ext.BidTypeBanner,
	}

	bidResponse.Bids = append(bidResponse.Bids, b)

	return bidResponse, nil
}
