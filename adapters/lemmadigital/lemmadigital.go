package lemmadigital

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/errortypes"
	"github.com/prebid/prebid-server/v2/macros"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

// HOST constants
const (
	EMEA         = "emea"
	USES         = "uses"
	USCT         = "usct"
	USWS         = "ucws"
	SG           = "sg"
	DOOH_US      = "doohus"
	DOOH_SG      = "doohsg"
	DEFAULT_HOST = SG
)

var (
	validHosts = map[string]bool{
		USES: true,
		USWS: true,
		USCT: true,
		SG:   true,
		EMEA: true,
	}
	validDoohHosts = map[string]bool{
		DOOH_US: true,
		DOOH_SG: true,
	}
)

type ExtraInfo struct {
	Host     string `json:"host,omitempty"`
	DoohHost string `json:"dooh_host,omitempty"`
}

type adapter struct {
	endpoint  *template.Template
	extraInfo ExtraInfo
}

// Builder builds a new instance of the Lemmadigital adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	template, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	extraInfo, err := parseExtraInfo(config.ExtraAdapterInfo)
	if err != nil {
		return nil, err
	}

	bidder := &adapter{
		endpoint:  template,
		extraInfo: extraInfo,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if len(request.Imp) == 0 {
		return nil, []error{errors.New("Impression array should not be empty")}
	}

	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(request.Imp[0].Ext, &bidderExt); err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Invalid imp.ext for impression index %d. Error Infomation: %s", 0, err.Error()),
		}}
	}

	var impExt openrtb_ext.ImpExtLemmaDigital
	if err := json.Unmarshal(bidderExt.Bidder, &impExt); err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Invalid imp.ext.bidder for impression index %d. Error Infomation: %s", 0, err.Error()),
		}}
	}

	endpoint, err := a.buildEndpointURL(impExt, nil != request.DOOH)
	if err != nil {
		return nil, []error{err}
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	requestData := &adapters.RequestData{
		Method: "POST",
		Uri:    endpoint,
		Body:   requestJSON,
		ImpIDs: openrtb_ext.GetImpIDs(request.Imp),
	}

	return []*adapters.RequestData{requestData}, nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidType := openrtb_ext.BidTypeBanner
	if nil != request.Imp[0].Video {
		bidType = openrtb_ext.BidTypeVideo
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	if len(response.Cur) > 0 {
		bidResponse.Currency = response.Cur
	}
	if len(response.SeatBid) > 0 {
		for i := range response.SeatBid[0].Bid {
			b := &adapters.TypedBid{
				Bid:     &response.SeatBid[0].Bid[i],
				BidType: bidType,
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}

	return bidResponse, nil
}

func (a *adapter) buildEndpointURL(params openrtb_ext.ImpExtLemmaDigital, isDooh bool) (string, error) {
	host := a.extraInfo.Host
	if isDooh {
		host = a.extraInfo.DoohHost
	}
	endpointParams := macros.EndpointTemplateParams{PublisherID: strconv.Itoa(params.PublisherId),
		AdUnit: strconv.Itoa(params.AdId), Host: host}
	return macros.ResolveMacros(a.endpoint, endpointParams)
}

func parseExtraInfo(v string) (ExtraInfo, error) {
	var extraInfo ExtraInfo
	if len(v) == 0 {
		extraInfo.defaultHost()
		return extraInfo, nil
	}

	if err := json.Unmarshal([]byte(v), &extraInfo); err != nil {
		return extraInfo, fmt.Errorf("invalid extra info: %v", err)
	}

	if _, ok := validHosts[extraInfo.Host]; !ok {
		return extraInfo, fmt.Errorf("invalid host in extra info: %s", extraInfo.Host)
	}

	if extraInfo.DoohHost == "" {
		extraInfo.assignDoohHost()
	}

	if _, ok := validDoohHosts[extraInfo.DoohHost]; !ok {
		return extraInfo, fmt.Errorf("invalid dooh host: %s", extraInfo.DoohHost)
	}

	return extraInfo, nil
}

func (ei *ExtraInfo) defaultHost() {
	ei.Host = DEFAULT_HOST
}

func (ei *ExtraInfo) assignDoohHost() {
	var doohHost string
	switch ei.Host {
	case USES, USCT, USWS:
		doohHost = DOOH_US
	default:
		doohHost = DOOH_SG
	}
	ei.DoohHost = doohHost
}
