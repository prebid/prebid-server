package invibes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"text/template"

	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbsmetrics"
)

const adapterVersion = "prebid_1.0.0"

type InvibesAdRequest struct {
	BidParamsJson string
	Location      string
	Lid           string
	IsTestBid     bool
	Kw            string
	IsAmp         bool
	Width         string
	Height        string
	GdprConsent   string
	Gdpr          bool
	Bvid          string
	InvibBVLog    bool
	VideoAdDebug  bool
}
type InvibesBidParams struct {
	PlacementIds []string
	BidVersion   string
	Properties   map[string]InvibesPlacementProperty
}
type InvibesPlacementProperty struct {
	Formats []openrtb.Format
	ImpId   string
}
type InvibesInternalParams struct {
	BidParams   InvibesBidParams
	Host        string
	IsAmp       bool
	Gdpr        string
	GdprConsent string

	TestIp   string
	TestBvid string
	TestLog  bool
}
type BidServerBidderResponse struct {
	Currency  string              `json:"currency"`
	TypedBids []BidServerTypedBid `json:"typedBids"`
	Error     string              `json:"error"`
}
type BidServerTypedBid struct {
	Bid          openrtb.Bid `json:"bid"`
	DealPriority int         `json:"dealPriority"`
}

func (a *InvibesInternalParams) IsTestRequest() bool {
	return a.TestIp != "" || a.TestBvid != ""
}

type InvibesAdapter struct {
	EndpointTemplate template.Template
}

func NewInvibesBidder(endpointTemplate string) *InvibesAdapter {
	urlTemplate, err := template.New("endpointTemplate").Parse(endpointTemplate)
	if err != nil {
		glog.Fatal("Unable to parse endpoint url template")
		return nil
	}
	return &InvibesAdapter{EndpointTemplate: *urlTemplate}
}

func (a *InvibesAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{
			Message: "No imps in the bid request",
		}}
	}

	consentString := ""
	if request.User != nil {
		var extUser openrtb_ext.ExtUser
		if err := json.Unmarshal(request.User.Ext, &extUser); err == nil {
			consentString = extUser.Consent
		}
	}
	gdprApplies := "1"
	var extRegs openrtb_ext.ExtRegs
	if request.Regs != nil {
		if err := json.Unmarshal(request.Regs.Ext, &extRegs); err == nil {
			if extRegs.GDPR != nil && (*extRegs.GDPR == 0 || *extRegs.GDPR == 1) {
				gdprApplies = strconv.Itoa(int(*extRegs.GDPR))
			}
		}
	}

	var httpRequests []*adapters.RequestData
	var errors []error

	var invibesInternalParams InvibesInternalParams = InvibesInternalParams{
		BidParams: InvibesBidParams{
			Properties: make(map[string]InvibesPlacementProperty),
			BidVersion: "1",
		},
	}

	for _, imp := range request.Imp {
		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: "Error parsing bidderExt object",
			})
		}
		if len(errors) == 0 {
			var invibesExt openrtb_ext.ExtImpInvibes
			if err := json.Unmarshal(bidderExt.Bidder, &invibesExt); err != nil {
				errors = append(errors, &errortypes.BadInput{
					Message: "Error parsing invibesExt parameters",
				})
			}
			if imp.Banner == nil {
				errors = append(errors, &errortypes.BadInput{
					Message: "Banner not specified",
				})
			}
			if len(errors) == 0 {
				currentBanner := *imp.Banner
				var adFormats []openrtb.Format
				if currentBanner.Format != nil {
					adFormats = currentBanner.Format
				} else if currentBanner.W != nil && currentBanner.H != nil {
					adFormats = []openrtb.Format{
						{
							W: *currentBanner.W,
							H: *currentBanner.H,
						},
					}
				}

				invibesInternalParams.Host = invibesExt.Host
				invibesInternalParams.BidParams.PlacementIds = append(invibesInternalParams.BidParams.PlacementIds, strings.TrimSpace(invibesExt.PlacementId))
				invibesInternalParams.BidParams.Properties[invibesExt.PlacementId] = InvibesPlacementProperty{
					ImpId:   imp.ID,
					Formats: adFormats,
				}
				if invibesExt.Debug.TestIp != "" {
					invibesInternalParams.TestIp = invibesExt.Debug.TestIp
				}
				if invibesExt.Debug.TestBvid != "" {
					invibesInternalParams.TestBvid = invibesExt.Debug.TestBvid
				}
				invibesInternalParams.TestLog = invibesExt.Debug.TestLog
			}
		}
	}
	if reqInfo.PbsEntryPoint == pbsmetrics.ReqTypeAMP {
		invibesInternalParams.IsAmp = true
	}

	if len(invibesInternalParams.BidParams.PlacementIds) == 0 {
		return nil, errors
	}

	invibesInternalParams.Gdpr = gdprApplies
	invibesInternalParams.GdprConsent = consentString

	newHttpRequest, err := a.makeRequest(invibesInternalParams, reqInfo, httpRequests, request)
	if err != nil {
		errors = append(errors, err)
	} else if newHttpRequest != nil {
		httpRequests = append(httpRequests, newHttpRequest)
	}

	return httpRequests, errors
}

func (a *InvibesAdapter) makeRequest(invibesParams InvibesInternalParams, reqInfo *adapters.ExtraRequestInfo, existingRequests []*adapters.RequestData, request *openrtb.BidRequest) (*adapters.RequestData, error) {

	url, err := a.makeURL(request, invibesParams.Host)
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

	if invibesParams.TestIp != "" {
		headers.Add("X-Forwarded-For", invibesParams.TestIp)
	} else if request.Device != nil {
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
	}, nil
}

func (a *InvibesAdapter) makeParameter(invibesParams InvibesInternalParams, request *openrtb.BidRequest) (*InvibesAdRequest, error) {
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
			width = strconv.FormatUint(request.Device.W, 10)
		}

		if request.Device.H > 0 {
			height = strconv.FormatUint(request.Device.H, 10)
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
			IsAmp:         invibesParams.IsAmp,
			Width:         width,
			Height:        height,
			GdprConsent:   invibesParams.GdprConsent,
			Gdpr:          invibesParams.Gdpr != "0",
			Bvid:          invibesParams.TestBvid,
			InvibBVLog:    invibesParams.TestLog,
			VideoAdDebug:  invibesParams.TestLog,
		}
	}

	return &invRequest, err
}

func (a *InvibesAdapter) makeURL(request *openrtb.BidRequest, host string) (string, error) {
	var endpointURL *url.URL
	endpointParams := macros.EndpointTemplateParams{Host: host}
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
	internalRequest *openrtb.BidRequest,
	externalRequest *adapters.RequestData,
	response *adapters.ResponseData,
) (*adapters.BidderResponse, []error) {
	if response.StatusCode != http.StatusOK {
		return nil, []error{fmt.Errorf("Unexpected status code: %d.", response.StatusCode)}
	}

	bidResponse := BidServerBidderResponse{}
	if err := json.Unmarshal(response.Body, &bidResponse); err != nil {
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
