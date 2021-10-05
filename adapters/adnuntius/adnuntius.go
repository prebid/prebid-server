package adnuntius

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type QueryString map[string]string
type adapter struct {
	tzo      timezone
	endpoint string
}
type adnAdunit struct {
	AuId     string `json:"auId"`
	TargetId string `json:"targetId"`
}

type AdnResponse struct {
	AdUnits []struct {
		AuId       string
		TargetId   string
		Html       string
		ResponseId string
		Ads        []struct {
			Bid struct {
				Amount   float64
				Currency string
			}
			AdId            string
			CreativeWidth   string
			CreativeHeight  string
			CreativeId      string
			LineItemId      string
			Html            string
			DestinationUrls map[string]string
		}
	}
}
type adnMetaData struct {
	Usi string `json:"usi,omitempty"`
}
type adnRequest struct {
	AdUnits  []adnAdunit `json:"adUnits"`
	MetaData adnMetaData `json:"metaData,omitempty"`
	Context  string      `json:"context,omitempty"`
}

// Builder builds a new instance of the Adnuntius adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		tzo:      realTzo{},
		endpoint: config.Endpoint,
	}

	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var extRequests []*adapters.RequestData
	var errs []error

	extRequests, err := a.generateRequests(*request)
	if err != nil {
		errs = append(errs, err...)
	}
	return extRequests, errs
}

func setHeaders() http.Header {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	return headers
}

func makeEndpointUrl(ortbRequest openrtb2.BidRequest, a *adapter) (string, []error) {
	uri, err := url.Parse(a.endpoint)
	if err != nil {
		return "", []error{fmt.Errorf("failed to parse Adnuntius endpoint: %v", err)}
	}

	gdpr, consent, err := getGDPR(&ortbRequest)
	if err != nil {
		return "", []error{fmt.Errorf("failed to parse Adnuntius endpoint: %v", err)}
	}

	_, offset := a.tzo.Now().UTC().Local().Zone()
	tzo := -offset / 3600 * 60

	q := uri.Query()
	if gdpr != "" && consent != "" {
		q.Set("gdpr", gdpr)
		q.Set("consentString", consent)
	}
	q.Set("tzo", fmt.Sprint(tzo))
	q.Set("format", "json")

	url := a.endpoint + "?" + q.Encode()
	return url, nil
}

/*
	Generate the requests to Adnuntius to reduce the amount of requests going out.
*/
func (a *adapter) generateRequests(ortbRequest openrtb2.BidRequest) ([]*adapters.RequestData, []error) {
	var requestData []*adapters.RequestData
	var errors []error
	networkAdunitMap := make(map[string][]adnAdunit)
	headers := setHeaders()

	endpoint, err := makeEndpointUrl(ortbRequest, a)
	if err != nil {
		errors = append(errors, fmt.Errorf("failed to parse URL: %v", err))
	}

	for _, imp := range ortbRequest.Imp {
		if imp.Banner == nil {
			errors = append(errors, &errortypes.BadInput{
				Message: fmt.Sprintf("ignoring imp id=%s, Adnuntius supports only Banner", imp.ID),
			})
		}
		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: fmt.Sprintf("Error unmarshalling ExtImpBidder: %s", err.Error()),
			})
		}

		var adnuntiusExt openrtb_ext.ImpExtAdnunitus
		if err := json.Unmarshal(bidderExt.Bidder, &adnuntiusExt); err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: fmt.Sprintf("Error unmarshalling ExtImpBmtm: %s", err.Error()),
			})
		}

		network := "default"
		if adnuntiusExt.Network != "" {
			network = adnuntiusExt.Network
		}

		networkAdunitMap[network] = append(
			networkAdunitMap[network],
			adnAdunit{
				AuId:     adnuntiusExt.Auid,
				TargetId: fmt.Sprintf("%s-%s", adnuntiusExt.Auid, imp.ID),
			})
	}

	site := "unknown"
	if ortbRequest.Site != nil && ortbRequest.Site.Page != "" {
		site = ortbRequest.Site.Page
	}

	for _, networkAdunits := range networkAdunitMap {

		adnuntiusRequest := adnRequest{
			AdUnits: networkAdunits,
			Context: site,
		}

		ortbUser := ortbRequest.User
		if ortbUser != nil {
			ortbUserId := ortbRequest.User.ID
			if ortbUserId != "" {
				adnuntiusRequest.MetaData.Usi = ortbRequest.User.ID
			}
		}

		adnJson, err := json.Marshal(adnuntiusRequest)
		if err != nil {
			errors = append(errors, err)
		}

		if len(errors) > 0 {
			return nil, errors
		}
		requestData = append(requestData, &adapters.RequestData{
			Method:  http.MethodPost,
			Uri:     endpoint,
			Body:    adnJson,
			Headers: headers,
		})

	}

	return requestData, nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Status code: %d, Request malformed", response.StatusCode),
		}}
	}

	var adnResponse AdnResponse
	if err := json.Unmarshal(response.Body, &adnResponse); err != nil {
		return nil, []error{err}
	}

	bidResponse, bidErr := generateBidResponse(&adnResponse, request)
	if bidErr != nil {
		return nil, bidErr
	}

	return bidResponse, nil
}

func getGDPR(request *openrtb2.BidRequest) (string, string, error) {
	gdpr := ""
	var extRegs openrtb_ext.ExtRegs
	if request.Regs != nil {
		if err := json.Unmarshal(request.Regs.Ext, &extRegs); err != nil {
			return "", "", fmt.Errorf("failed to parse ExtRegs in Adnuntius GDPR check: %v", err)
		}
		if extRegs.GDPR != nil && (*extRegs.GDPR == 0 || *extRegs.GDPR == 1) {
			gdpr = strconv.Itoa(int(*extRegs.GDPR))
		}
	}

	consent := ""
	if request.User != nil && request.User.Ext != nil {
		var extUser openrtb_ext.ExtUser
		if err := json.Unmarshal(request.User.Ext, &extUser); err != nil {
			return "", "", fmt.Errorf("failed to parse ExtUser in Adnuntius GDPR check: %v", err)
		}
		consent = extUser.Consent
	}

	return gdpr, consent, nil
}

func generateBidResponse(adnResponse *AdnResponse, request *openrtb2.BidRequest) (*adapters.BidderResponse, []error) {
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(adnResponse.AdUnits))
	var currency string

	for i, adunit := range adnResponse.AdUnits {

		var bid openrtb2.Bid
		if len(adunit.Ads) > 0 {
			var adsErr []error
			ad := adunit.Ads[0]

			currency = ad.Bid.Currency

			creativeWidth, Werr := strconv.ParseInt(ad.CreativeWidth, 10, 64)
			if Werr != nil {
				adsErr = append(adsErr, Werr)
			}

			creativeHeight, Herr := strconv.ParseInt(ad.CreativeHeight, 10, 64)
			if Herr != nil {
				adsErr = append(adsErr, Herr)
			}

			adDomain := []string{}
			for _, url := range ad.DestinationUrls {
				domainArray := strings.Split(url, "/")
				domain := strings.Replace(domainArray[2], "www.", "", -1)
				adDomain = append(adDomain, domain)
			}

			if len(adsErr) > 0 {
				return nil, adsErr
			}

			bid = openrtb2.Bid{
				ID:      ad.AdId,
				ImpID:   request.Imp[i].ID,
				W:       creativeWidth,
				H:       creativeHeight,
				AdID:    ad.AdId,
				CID:     ad.LineItemId,
				CrID:    ad.CreativeId,
				Price:   ad.Bid.Amount * 1000,
				AdM:     adunit.Html,
				ADomain: adDomain,
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: "banner",
			})
		}

	}
	bidResponse.Currency = currency
	return bidResponse, nil
}
