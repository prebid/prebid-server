package adnuntius

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters"
	"github.com/prebid/prebid-server/v2/errortypes"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

func makeEndpointUrl(ortbRequest openrtb2.BidRequest, a *adapter, noCookies bool) (string, []error) {
	uri, err := url.Parse(a.endpoint)
	endpointUrl := a.endpoint
	if err != nil {
		return "", []error{fmt.Errorf("failed to parse Adnuntius endpoint: %v", err)}
	}

	gdpr, consent, err := getGDPR(&ortbRequest)
	if err != nil {
		return "", []error{fmt.Errorf("failed to parse Adnuntius endpoint: %v", err)}
	}

	if !noCookies {
		var deviceExt extDeviceAdnuntius
		if ortbRequest.Device != nil && ortbRequest.Device.Ext != nil {
			if err := json.Unmarshal(ortbRequest.Device.Ext, &deviceExt); err != nil {
				return "", []error{fmt.Errorf("failed to parse Adnuntius endpoint: %v", err)}
			}
		}

		if deviceExt.NoCookies {
			noCookies = true
		}
	}

	_, offset := a.time.Now().Zone()
	tzo := -offset / minutesInHour

	q := uri.Query()
	if gdpr != "" {
		endpointUrl = a.extraInfo
		q.Set("gdpr", gdpr)
	}

	if consent != "" {
		q.Set("consentString", consent)
	}

	if noCookies {
		q.Set("noCookies", "true")
	}

	q.Set("tzo", fmt.Sprint(tzo))
	q.Set("format", "prebid")

	url := endpointUrl + "?" + q.Encode()
	return url, nil
}

func getImpSizes(imp openrtb2.Imp) [][]int64 {

	if len(imp.Banner.Format) > 0 {
		sizes := make([][]int64, len(imp.Banner.Format))
		for i, format := range imp.Banner.Format {
			sizes[i] = []int64{format.W, format.H}
		}

		return sizes
	}

	if imp.Banner.W != nil && imp.Banner.H != nil {
		size := make([][]int64, 1)
		size[0] = []int64{*imp.Banner.W, *imp.Banner.H}
		return size
	}

	return nil
}

func setHeaders(ortbRequest openrtb2.BidRequest) http.Header {

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	if ortbRequest.Device != nil {
		if ortbRequest.Device.IP != "" {
			headers.Add("X-Forwarded-For", ortbRequest.Device.IP)
		}
		if ortbRequest.Device.UA != "" {
			headers.Add("user-agent", ortbRequest.Device.UA)
		}
	}
	return headers
}

func getSiteExtAsKv(request *openrtb2.BidRequest) (siteExt, error) {
	var extSite siteExt
	if request.Site != nil && request.Site.Ext != nil {
		if err := json.Unmarshal(request.Site.Ext, &extSite); err != nil {
			return extSite, fmt.Errorf("failed to parse ExtSite in Adnuntius: %v", err)
		}
	}
	return extSite, nil
}

func getGDPR(request *openrtb2.BidRequest) (string, string, error) {

	gdpr := ""
	var extRegs openrtb_ext.ExtRegs
	if request.Regs != nil && request.Regs.Ext != nil {
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

/*
Generate the requests to Adnuntius to reduce the amount of requests going out.
*/
func (a *adapter) generateRequests(ortbRequest openrtb2.BidRequest) ([]*adapters.RequestData, []error) {
	var requestData []*adapters.RequestData
	networkAdunitMap := make(map[string][]adnAdunit)
	headers := setHeaders(ortbRequest)
	var noCookies bool = false

	for _, imp := range ortbRequest.Imp {
		if imp.Banner == nil {
			return nil, []error{&errortypes.BadInput{
				Message: fmt.Sprintf("ignoring imp id=%s, Adnuntius supports only Banner", imp.ID),
			}}
		}

		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			return nil, []error{&errortypes.BadInput{
				Message: fmt.Sprintf("Error unmarshalling ExtImpBidder: %s", err.Error()),
			}}
		}

		var adnuntiusExt openrtb_ext.ImpExtAdnunitus
		if err := json.Unmarshal(bidderExt.Bidder, &adnuntiusExt); err != nil {
			return nil, []error{&errortypes.BadInput{
				Message: fmt.Sprintf("Error unmarshalling ExtImpValues: %s", err.Error()),
			}}
		}

		if adnuntiusExt.NoCookies {
			noCookies = true
		}

		network := defaultNetwork
		if adnuntiusExt.Network != "" {
			network = adnuntiusExt.Network
		}

		adUnit := adnAdunit{
			AuId:       adnuntiusExt.Auid,
			TargetId:   fmt.Sprintf("%s-%s", adnuntiusExt.Auid, imp.ID),
			Dimensions: getImpSizes(imp),
		}
		if adnuntiusExt.MaxDeals > 0 {
			adUnit.MaxDeals = adnuntiusExt.MaxDeals
		}
		networkAdunitMap[network] = append(
			networkAdunitMap[network],
			adUnit)
	}

	endpoint, err := makeEndpointUrl(ortbRequest, a, noCookies)
	if err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("failed to parse URL: %s", err),
		}}
	}

	site := defaultSite
	if ortbRequest.Site != nil && ortbRequest.Site.Page != "" {
		site = ortbRequest.Site.Page
	}

	extSite, erro := getSiteExtAsKv(&ortbRequest)
	if erro != nil {
		return nil, []error{fmt.Errorf("failed to parse site Ext: %v", err)}
	}

	for _, networkAdunits := range networkAdunitMap {

		adnuntiusRequest := adnRequest{
			AdUnits:   networkAdunits,
			Context:   site,
			KeyValues: extSite.Data,
		}

		var extUser openrtb_ext.ExtUser
		if ortbRequest.User != nil && ortbRequest.User.Ext != nil {
			if err := json.Unmarshal(ortbRequest.User.Ext, &extUser); err != nil {
				return nil, []error{fmt.Errorf("failed to parse Ext User: %v", err)}
			}
		}

		// Will change when our adserver can accept multiple user IDS
		if extUser.Eids != nil && len(extUser.Eids) > 0 {
			if len(extUser.Eids[0].UIDs) > 0 {
				adnuntiusRequest.MetaData.Usi = extUser.Eids[0].UIDs[0].ID
			}
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
			return nil, []error{&errortypes.BadInput{
				Message: fmt.Sprintf("Error unmarshalling adnuntius request: %s", err.Error()),
			}}
		}

		requestData = append(requestData, &adapters.RequestData{
			Method:  http.MethodPost,
			Uri:     endpoint,
			Body:    adnJson,
			Headers: headers,
			ImpIDs:  openrtb_ext.GetImpIDs(ortbRequest.Imp),
		})

	}

	return requestData, nil
}
