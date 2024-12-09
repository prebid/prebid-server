package invibes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

const adapterVersion = "prebid_1.0.0"
const invibesBidVersion = "4"

type InvibesAdRequest struct {
	BidParamsJson string `json:"BidParamsJson"`
	Location      string `json:"Location"`
	Lid           string `json:"Lid"`
	IsTestBid     bool   `json:"IsTestBid"`
	Kw            string `json:"Kw"`
	IsAMP         bool   `json:"IsAmp"`
	Width         string `json:"Width"`
	Height        string `json:"Height"`
	GDPRConsent   string `json:"GdprConsent"`
	GDPR          bool   `json:"Gdpr"`
	Bvid          string `json:"Bvid"`
	InvibBVLog    bool   `json:"InvibBVLog"`
	VideoAdDebug  bool   `json:"VideoAdDebug"`
}
type InvibesBidParams struct {
	PlacementIDs []string                            `json:"PlacementIds"`
	BidVersion   string                              `json:"BidVersion"`
	Properties   map[string]InvibesPlacementProperty `json:"Properties"`
}
type InvibesPlacementProperty struct {
	Formats []openrtb2.Format `json:"Formats"`
	ImpID   string            `json:"ImpId"`
}
type InvibesInternalParams struct {
	BidParams   InvibesBidParams
	DomainID    int
	IsAMP       bool
	GDPR        bool
	GDPRConsent string

	TestBvid string
	TestLog  bool
}
type BidServerBidderResponse struct {
	Currency  string              `json:"currency"`
	TypedBids []BidServerTypedBid `json:"typedBids"`
	Error     string              `json:"error"`
}
type BidServerTypedBid struct {
	Bid          openrtb2.Bid `json:"bid"`
	DealPriority int          `json:"dealPriority"`
}

func (a *InvibesInternalParams) IsTestRequest() bool {
	return a.TestBvid != ""
}

type InvibesAdapter struct {
	EndpointTemplate *template.Template
}

// Builder builds a new instance of the Invibes adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	template, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := InvibesAdapter{
		EndpointTemplate: template,
	}
	return &bidder, nil
}

func (a *InvibesAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var httpRequests []*adapters.RequestData
	var tempErrors []error
	gdprApplies, consentString := readGDPR(request)

	var invibesInternalParams InvibesInternalParams = InvibesInternalParams{
		BidParams: InvibesBidParams{
			Properties: make(map[string]InvibesPlacementProperty),
			BidVersion: invibesBidVersion,
		},
	}

	for _, imp := range request.Imp {
		var bidderExt adapters.ExtImpBidder
		if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
			tempErrors = append(tempErrors, &errortypes.BadInput{
				Message: "Error parsing bidderExt object",
			})
			continue
		}
		var invibesExt openrtb_ext.ExtImpInvibes
		if err := jsonutil.Unmarshal(bidderExt.Bidder, &invibesExt); err != nil {
			tempErrors = append(tempErrors, &errortypes.BadInput{
				Message: "Error parsing invibesExt parameters",
			})
			continue
		}
		if imp.Banner == nil {
			tempErrors = append(tempErrors, &errortypes.BadInput{
				Message: "Banner not specified",
			})
			continue
		}

		adFormats := readAdFormats(*imp.Banner)

		invibesInternalParams.DomainID = invibesExt.DomainID
		invibesInternalParams.BidParams.PlacementIDs = append(invibesInternalParams.BidParams.PlacementIDs, strings.TrimSpace(invibesExt.PlacementID))
		invibesInternalParams.BidParams.Properties[invibesExt.PlacementID] = InvibesPlacementProperty{
			ImpID:   imp.ID,
			Formats: adFormats,
		}
		if invibesExt.Debug.TestBvid != "" {
			invibesInternalParams.TestBvid = invibesExt.Debug.TestBvid
		}
		invibesInternalParams.TestLog = invibesExt.Debug.TestLog
	}
	if reqInfo.PbsEntryPoint == metrics.ReqTypeAMP {
		invibesInternalParams.IsAMP = true
	}

	if len(invibesInternalParams.BidParams.PlacementIDs) == 0 {
		return nil, tempErrors
	}

	var finalErrors []error
	invibesInternalParams.GDPR = gdprApplies
	invibesInternalParams.GDPRConsent = consentString

	newHttpRequest, err := a.makeRequest(invibesInternalParams, reqInfo, httpRequests, request)
	if err != nil {
		finalErrors = append(finalErrors, err)
	} else if newHttpRequest != nil {
		httpRequests = append(httpRequests, newHttpRequest)
	}

	return httpRequests, finalErrors
}

func readGDPR(request *openrtb2.BidRequest) (bool, string) {
	consentString := ""
	if request.User != nil {
		var extUser openrtb_ext.ExtUser
		if err := jsonutil.Unmarshal(request.User.Ext, &extUser); err == nil {
			consentString = extUser.Consent
		}
	}
	gdprApplies := true
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

func readAdFormats(currentBanner openrtb2.Banner) []openrtb2.Format {
	var adFormats []openrtb2.Format
	if currentBanner.Format != nil {
		adFormats = currentBanner.Format
	} else if currentBanner.W != nil && currentBanner.H != nil {
		adFormats = []openrtb2.Format{
			{
				W: *currentBanner.W,
				H: *currentBanner.H,
			},
		}
	}
	return adFormats
}

func (a *InvibesAdapter) makeRequest(invibesParams InvibesInternalParams, reqInfo *adapters.ExtraRequestInfo, existingRequests []*adapters.RequestData, request *openrtb2.BidRequest) (*adapters.RequestData, error) {

	url, err := a.makeURL(request, invibesParams.DomainID)
	if err != nil {
		return nil, err
	}
	parameter, errp := a.makeParameter(invibesParams, request)
	if errp != nil {
		return nil, errp
	}
	body, errm := json.Marshal(parameter)
	if errm != nil {
		return nil, errm
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	if request.Device != nil {
		headers.Add("User-Agent", request.Device.UA)
	}

	if request.Device != nil {
		if request.Device.IP != "" {
			headers.Add("X-Forwarded-For", request.Device.IP)
		} else if request.Device.IPv6 != "" {
			headers.Add("X-Forwarded-For", request.Device.IPv6)
		}
	}
	if request.Site != nil {
		headers.Add("Referer", request.Site.Page)
	}
	headers.Add("Aver", adapterVersion)

	return &adapters.RequestData{
		Method:  "POST",
		Uri:     url,
		Headers: headers,
		Body:    body,
		ImpIDs:  getImpIDs(invibesParams.BidParams.Properties),
	}, nil
}

func (a *InvibesAdapter) makeParameter(invibesParams InvibesInternalParams, request *openrtb2.BidRequest) (*InvibesAdRequest, error) {
	var lid string = ""
	if request.User != nil && request.User.BuyerUID != "" {
		lid = request.User.BuyerUID
	}
	if request.Site == nil {
		return nil, &errortypes.BadInput{
			Message: "Site not specified",
		}
	}

	var width, height string
	if request.Device != nil {
		if request.Device.W > 0 {
			width = strconv.FormatInt(request.Device.W, 10)
		}

		if request.Device.H > 0 {
			height = strconv.FormatInt(request.Device.H, 10)
		}
	}

	var invRequest InvibesAdRequest
	bidParamsJson, err := json.Marshal(invibesParams.BidParams)
	if err == nil {
		invRequest = InvibesAdRequest{
			IsTestBid:     invibesParams.IsTestRequest(),
			BidParamsJson: string(bidParamsJson),
			Location:      request.Site.Page,
			Lid:           lid,
			Kw:            request.Site.Keywords,
			IsAMP:         invibesParams.IsAMP,
			Width:         width,
			Height:        height,
			GDPRConsent:   invibesParams.GDPRConsent,
			GDPR:          invibesParams.GDPR,
			Bvid:          invibesParams.TestBvid,
			InvibBVLog:    invibesParams.TestLog,
			VideoAdDebug:  invibesParams.TestLog,
		}
	}

	return &invRequest, err
}

func (a *InvibesAdapter) makeURL(request *openrtb2.BidRequest, domainID int) (string, error) {
	var subdomain string
	if domainID == 0 || domainID == 1 || domainID == 1001 {
		subdomain = "bid"
	} else if domainID < 1002 {
		subdomain = "bid" + strconv.Itoa(domainID)
	} else {
		subdomain = "bid" + strconv.Itoa(domainID-1000)
	}

	var endpointURL *url.URL
	endpointParams := macros.EndpointTemplateParams{ZoneID: subdomain}
	host, err := macros.ResolveMacros(a.EndpointTemplate, endpointParams)

	if err == nil {
		endpointURL, err = url.Parse(host)
	}
	if err != nil {
		return "", &errortypes.BadInput{
			Message: "Unable to parse url template: " + err.Error(),
		}
	}

	return endpointURL.String(), nil
}

func (a *InvibesAdapter) MakeBids(
	internalRequest *openrtb2.BidRequest,
	externalRequest *adapters.RequestData,
	response *adapters.ResponseData,
) (*adapters.BidderResponse, []error) {
	if response.StatusCode != http.StatusOK {
		return nil, []error{fmt.Errorf("Unexpected status code: %d.", response.StatusCode)}
	}

	bidResponse := BidServerBidderResponse{}
	if err := jsonutil.Unmarshal(response.Body, &bidResponse); err != nil {
		return nil, []error{err}
	}

	var parsedResponses = adapters.NewBidderResponseWithBidsCapacity(len(bidResponse.TypedBids))
	var errors []error
	parsedResponses.Currency = bidResponse.Currency

	if bidResponse.Error != "" {
		return nil, []error{fmt.Errorf("Server error: %s.", bidResponse.Error)}
	}
	for _, typedbid := range bidResponse.TypedBids {
		bid := typedbid.Bid
		parsedResponses.Bids = append(parsedResponses.Bids, &adapters.TypedBid{
			Bid:          &bid,
			BidType:      openrtb_ext.BidTypeBanner,
			DealPriority: typedbid.DealPriority,
		})
	}

	return parsedResponses, errors
}

func getImpIDs(bidParamsProperties map[string]InvibesPlacementProperty) []string {
	impIDs := make([]string, 0, len(bidParamsProperties))
	for i := range bidParamsProperties {
		impIDs = append(impIDs, bidParamsProperties[i].ImpID)
	}
	return impIDs
}
