package richaudience

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"text/template"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type RichaudienceAdapter struct {
	EndpointTemplate template.Template
}

type richaudienceRequest struct {
	ID     string             `json:"id,omitempty"`
	Imp    []openrtb2.Imp     `json:"imp,omitempty"`
	User   richaudienceUser   `json:"user,omitempty"`
	Device richaudienceDevice `json:"device,omitempty"`
	Site   richaudienceSite   `json:"site,omitempty"`
	Test   int8               `json:"test,omitempty"`
}

type richaudienceUser struct {
	BuyerUID string              `json:"buyeruid,omitempty"`
	Ext      richaudienceUserExt `json:"ext,omitempty"`
}

type richaudienceUserExt struct {
	Eids []openrtb_ext.ExtUserEid `json:"eids,omitempty"`
	GDPR string                   `json:"gdpr,omitempty"`
}

type richaudienceDevice struct {
	//GEO only Java, IFA
	IP  string `json:"ip,omitempty"`
	Lmt int8   `json:"lmt,omitempty"`
	DNT int8   `json:"dnt,omitempty"`
	UA  string `json:"ua,omitempty"`
}
type richaudienceSite struct {
	//Site: Cat IAB
	Domain string `json:"domain,omitempty"`
	Page   string `json:"page,omitempty"`
}

// Builder builds a new instance of the Foo adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {

	template, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	if config.Endpoint != "" && strings.Contains(config.Endpoint, "richaudience") {
		bidder := &RichaudienceAdapter{
			EndpointTemplate: *template,
		}

		return bidder, nil
	}

	return nil, nil
}

func (a *RichaudienceAdapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	richaudienceRequest := richaudienceRequest{}

	raiHeaders := http.Header{}
	//Global Vars
	secure := int8(0)
	//Object IMP
	resImps := make([]openrtb2.Imp, 0, len(request.Imp))
	raiErrors := make([]error, 0, len(request.Imp))

	var regsExt *openrtb_ext.ExtRegs
	if request.Regs != nil && request.Regs.Ext != nil {
		if err := json.Unmarshal(request.Regs.Ext, &regsExt); err != nil {
			fmt.Println(err)
		}
	}

	if regsExt != nil {
		fmt.Println(regsExt)
	}

	setHeaders(&raiHeaders)

	richaudienceRequest.ID = request.ID

	raiImp, err := setImp(request, &secure, &richaudienceRequest)

	if err != nil {
		fmt.Printf("%s", err)
	}

	resImps = append(resImps, raiImp)
	richaudienceRequest.Imp = resImps

	setSite(request, &richaudienceRequest)

	err = setDevice(request, &richaudienceRequest)
	if err != nil {
		raiErrors = append(raiErrors, &errortypes.BadInput{
			Message: err.Error(),
		})
		return nil, raiErrors
	}

	err = setUser(request, &richaudienceRequest)
	if err != nil {
		raiErrors = append(raiErrors, &errortypes.BadInput{
			Message: err.Error(),
		})
		return nil, raiErrors
	}

	//User: consent
	req, err := json.Marshal(richaudienceRequest)
	if err != nil {
		fmt.Printf("%s", err)
	}

	urlParams := macros.EndpointTemplateParams{Host: "http://ortb.richaudience.com/ortb/?bidder=pbs"}
	url, err := macros.ResolveMacros(&a.EndpointTemplate, urlParams)

	requestData := &adapters.RequestData{
		Method:  "POST",
		Uri:     url,
		Body:    req,
		Headers: raiHeaders,
	}

	return []*adapters.RequestData{requestData}, raiErrors
}

func (a *RichaudienceAdapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {

	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode == http.StatusBadRequest {
		err := &errortypes.BadInput{
			Message: "Unexpected status code: 400. Bad request from publisher. Run with request.debug = 1 for more info.",
		}
		return nil, []error{err}
	}

	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: err.Error(),
		}}
	}

	if responseData.StatusCode != http.StatusOK {
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info.", responseData.StatusCode),
		}
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &response); err != nil {

		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur
	for _, seatBid := range response.SeatBid {
		for _, bid := range seatBid.Bid {
			b := &adapters.TypedBid{
				Bid:     &bid,
				BidType: "banner",
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}

	return bidResponse, nil
}

func setHeaders(raiHeaders *http.Header) {
	raiHeaders.Set("Content-Type", "application/json;charset=utf-8")
	raiHeaders.Set("Accept", "application/json")
	raiHeaders.Add("X-Openrtb-Version", "2.5")
}

func setImp(request *openrtb2.BidRequest, secure *int8, richaudienceRequest *richaudienceRequest) (raiImp openrtb2.Imp, err error) {
	//Imp: Id, tagId, Secure, Bidfloor, Bidfloorcur, Banner, test
	//Banner: Id, W, H, Format
	//Format: W, H

	for _, imp := range request.Imp {
		raiExt := parseImpExt(&imp)

		if raiExt != nil {
			if raiExt.Pid != "" {
				imp.TagID = raiExt.Pid
			}

			if raiExt.TestRa {
				richaudienceRequest.Test = int8(1)
			}

			if raiExt.BidFloor <= 0 {
				imp.BidFloor = 0.00001
			} else {
				imp.BidFloor = raiExt.BidFloor
			}

			if raiExt.BidFloorCur == "" {
				imp.BidFloorCur = "USD"
			} else {
				imp.BidFloorCur = raiExt.BidFloorCur
			}
		}

		if request.Site != nil && request.Site.Page != "" {
			pageURL, error := url.Parse(request.Site.Page)
			if error == nil && pageURL.Scheme == "https" {
				*secure = int8(1)
			}
		}
		imp.Secure = secure

		if imp.Banner == nil {
			err = &errortypes.BadInput{
				Message: "Banner Object not found",
			}
			return
		} else {
			raiBanner := *imp.Banner
			if raiBanner.W == nil && raiBanner.H == nil {
				if len(raiBanner.Format) == 0 {
					err = &errortypes.BadInput{
						Message: "Format Object not found",
					}
				} else {
					imp.Banner = &raiBanner
				}
			}
		}

		raiImp = imp

	}
	return
}

func setSite(request *openrtb2.BidRequest, richaudienceRequest *richaudienceRequest) {
	if request.Site.Domain == "" {
		raiUrl := strings.Split(request.Site.Page, "//")
		richaudienceRequest.Site.Domain = strings.Split(raiUrl[1], "/")[0]
	} else {
		richaudienceRequest.Site.Domain = request.Site.Domain
	}

	if request.Site.Page != "" {
		richaudienceRequest.Site.Page = request.Site.Page
	}
}

func setDevice(request *openrtb2.BidRequest, richaudienceRequest *richaudienceRequest) (err error) {

	if request.Device.DNT != nil {
		richaudienceRequest.Device.DNT = *request.Device.DNT
	} else {
		richaudienceRequest.Device.DNT = 0
	}

	if request.Device.Lmt != nil {
		richaudienceRequest.Device.Lmt = *request.Device.Lmt
	} else {
		richaudienceRequest.Device.Lmt = 0
	}

	if request.Device.UA != "" {
		richaudienceRequest.Device.UA = request.Device.UA
	}

	//request.Device.IP = "11.222.33.44"

	if request.Device.IP == "" {
		err = &errortypes.BadInput{
			Message: "Not found IP",
		}
	} else {
		richaudienceRequest.Device.IP = request.Device.IP
	}

	return err
}

func setUser(request *openrtb2.BidRequest, richaudienceRequest *richaudienceRequest) (err error) {
	if request.User != nil {
		if request.User.BuyerUID != "" && request.User.BuyerUID != "[PDID]" {
			richaudienceRequest.User.BuyerUID = request.User.BuyerUID
		} else {
			err = &errortypes.BadInput{
				Message: "Not found PDID",
			}
		}
		if request.User.Ext != nil {
			var extUser openrtb_ext.ExtUser
			if err := json.Unmarshal(request.User.Ext, &extUser); err != nil {
				fmt.Printf("%s", err)
			} else {
				if extUser.Eids != nil {
					richaudienceRequest.User.Ext.Eids = extUser.Eids
				}
				if extUser.Consent != "" {
					richaudienceRequest.User.Ext.GDPR = extUser.Consent
				}
			}
		}
	}

	return
}

//utils
func parseImpExt(imp *openrtb2.Imp) *openrtb_ext.ExtImpRichaudience {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		err = &errortypes.BadInput{
			Message: fmt.Sprintf("not found parameters ext in ImpID : %s", imp.ID),
		}
	}

	var richaudienceExt openrtb_ext.ExtImpRichaudience
	if err := json.Unmarshal(bidderExt.Bidder, &richaudienceExt); err != nil {
		err = &errortypes.BadInput{
			Message: fmt.Sprintf("invalid parameters ext in ImpID: %s", imp.ID),
		}
	}

	return &richaudienceExt
}
