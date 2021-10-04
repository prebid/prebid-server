package richaudience

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/metrics"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type adapter struct {
	endpoint string
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
	Eids    []openrtb_ext.ExtUserEid `json:"eids,omitempty"`
	Consent string                   `json:"consent,omitempty"`
}

type richaudienceDevice struct {
	IP   string `json:"ip,omitempty"`
	IPv6 string `json:"ipv6,omitempty"`
	Lmt  int8   `json:"lmt,omitempty"`
	DNT  int8   `json:"dnt,omitempty"`
	UA   string `json:"ua,omitempty"`
}
type richaudienceSite struct {
	Domain string `json:"domain,omitempty"`
	Page   string `json:"page,omitempty"`
}

// Builder builds a new instance of the RichAudience adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {

	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	richaudienceRequest := richaudienceRequest{}

	raiHeaders := http.Header{}

	raiErrors := make([]error, 0, len(request.Imp))

	setHeaders(&raiHeaders)

	richaudienceRequest.ID = request.ID

	resImps, err := setImp(request, &richaudienceRequest)

	if err != nil {
		return nil, []error{err}
	}

	richaudienceRequest.Imp = resImps

	setSite(request, &richaudienceRequest)

	err = setDevice(request, &richaudienceRequest)
	if err != nil {
		return nil, []error{err}
	}

	err = setUser(request, &richaudienceRequest, requestInfo)
	if err != nil {
		return nil, []error{err}
	}

	req, err := json.Marshal(richaudienceRequest)
	if err != nil {
		return nil, []error{err}
	}

	requestData := &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    req,
		Headers: raiHeaders,
	}

	return []*adapters.RequestData{requestData}, raiErrors
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

	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &bidResp); err != nil {

		return nil, []error{&errortypes.BadServerResponse{
			Message: err.Error(),
		}}
	}

	var response openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			b := &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: openrtb_ext.BidTypeBanner,
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

func setImp(request *openrtb2.BidRequest, richaudienceRequest *richaudienceRequest) (resImps []openrtb2.Imp, err error) {
	for _, imp := range request.Imp {
		var secure = int8(0)
		raiExt, errImp := parseImpExt(&imp)

		if errImp != nil {
			return nil, errImp
		}

		if raiExt != nil {
			if raiExt.Pid != "" {
				imp.TagID = raiExt.Pid
			}

			if raiExt.Test {
				richaudienceRequest.Test = int8(1)
			}

			if raiExt.BidFloorCur == "" {
				imp.BidFloorCur = "USD"
			} else {
				imp.BidFloorCur = raiExt.BidFloorCur
			}
		}

		if request.Site != nil && request.Site.Page != "" {
			pageURL, err := url.Parse(request.Site.Page)
			if err == nil && pageURL.Scheme == "https" {
				secure = int8(1)
			}
		}
		imp.Secure = &secure

		if imp.Banner.W == nil && imp.Banner.H == nil {
			if len(imp.Banner.Format) == 0 {
				err = &errortypes.BadInput{
					Message: "Format Object not found",
				}
				return nil, err
			}
		}

		resImps = append(resImps, imp)

	}
	return resImps, nil
}

func setSite(request *openrtb2.BidRequest, richaudienceRequest *richaudienceRequest) {
	if request.Site != nil {
		if request.Site.Page != "" {
			if request.Site.Domain == "" {
				pageURL, err := url.Parse(request.Site.Page)
				if err == nil {
					richaudienceRequest.Site.Domain = pageURL.Host
				}
			} else {
				richaudienceRequest.Site.Domain = request.Site.Domain
			}

			richaudienceRequest.Site.Page = request.Site.Page
		}
	}
}

func setDevice(request *openrtb2.BidRequest, richaudienceRequest *richaudienceRequest) (err error) {

	if request.Device.IP == "" && request.Device.IPv6 == "" {
		err = &errortypes.BadInput{
			Message: "request.Device.IP is required",
		}
		return err
	} else if request.Device.IP != "" {
		richaudienceRequest.Device.IP = request.Device.IP
	} else if request.Device.IPv6 != "" {
		richaudienceRequest.Device.IPv6 = request.Device.IPv6
	}

	if request.Device.DNT != nil {
		richaudienceRequest.Device.DNT = *request.Device.DNT
	}

	if request.Device.Lmt != nil {
		richaudienceRequest.Device.Lmt = *request.Device.Lmt
	}

	if request.Device.UA != "" {
		richaudienceRequest.Device.UA = request.Device.UA
	}

	return err
}

func setUser(request *openrtb2.BidRequest, richaudienceRequest *richaudienceRequest, requestInfo *adapters.ExtraRequestInfo) (err error) {
	if request.User != nil {
		if requestInfo.PbsEntryPoint != metrics.ReqTypeAMP {
			if request.User.BuyerUID != "" && request.User.BuyerUID != "[PDID]" {
				richaudienceRequest.User.BuyerUID = request.User.BuyerUID
			} else {
				err = &errortypes.BadInput{
					Message: "request.ext.user.buyerUID is required",
				}
				return err
			}
		}

		if request.User.Ext != nil {
			var extUser openrtb_ext.ExtUser
			if err := json.Unmarshal(request.User.Ext, &extUser); err != nil {
				return err
			} else {
				if extUser.Eids != nil {
					richaudienceRequest.User.Ext.Eids = extUser.Eids
				}
				if extUser.Consent != "" {
					richaudienceRequest.User.Ext.Consent = extUser.Consent
				}
			}
		}
	}

	return nil
}

func parseImpExt(imp *openrtb2.Imp) (*openrtb_ext.ExtImpRichaudience, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		err = &errortypes.BadInput{
			Message: fmt.Sprintf("not found parameters ext in ImpID : %s", imp.ID),
		}
		return nil, err
	}

	var richaudienceExt openrtb_ext.ExtImpRichaudience
	if err := json.Unmarshal(bidderExt.Bidder, &richaudienceExt); err != nil {
		err = &errortypes.BadInput{
			Message: fmt.Sprintf("invalid parameters ext in ImpID: %s", imp.ID),
		}
		return nil, err
	}

	return &richaudienceExt, nil
}
