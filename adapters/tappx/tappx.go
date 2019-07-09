package tappx

import (
	"encoding/json"
	"fmt"
	"text/template"
	//"strings"
	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
	//"github.com/prebid/prebid-server/config"
	"net/http"
	"strconv"
	"time"
)

const TAPPX_BIDDER_VERSION = "1.0"

type TappxAdapter struct {
	http             *adapters.HTTPAdapter
	endpointTemplate template.Template
}

func NewTappxAdapter(config *adapters.HTTPAdapterConfig, endpointTemplate string) adapters.Bidder {
	return NewTappxBidder(adapters.NewHTTPAdapter(config).Client, endpointTemplate)
}

func NewTappxBidder(client *http.Client, endpointTemplate string) *TappxAdapter {
	a := &adapters.HTTPAdapter{Client: client}
	template, err := template.New("endpointTemplate").Parse(endpointTemplate)
	if err != nil {
		glog.Fatal("Unable to parse endpoint url template")
		return nil
	}
	return &TappxAdapter{
		http:             a,
		endpointTemplate: *template,
	}
}

type tappxParams struct {
	Host     string `json:"host"`
	TappxKey string `json:"tappxkey"`
	Endpoint string `json:"endpoint"`
}

func (a *TappxAdapter) Name() string {
	return "tappx"
}

func (a *TappxAdapter) SkipNoCookies() bool {
	return false
}

func (a *TappxAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if(len(request.Imp) == 0){
		return nil, []error{&errortypes.BadInput{
			Message: "No impression in the bid request",
		}}
	}

	errs := make([]error, 0, len(request.Imp))

	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(request.Imp[0].Ext, &bidderExt); err != nil { // 3
		errs = append(errs, err)
		return nil, errs
	}

	var tappxExt openrtb_ext.ExtImpTappx
	if err := json.Unmarshal(bidderExt.Bidder, &tappxExt); err != nil { // 2
		errs = append(errs, err)
		return nil, errs
	}

	var test int
	test = int(request.Test)

	url, err := a.buildEndpointURL(&tappxExt, test)
	if url == "" {
		return nil, []error{err}
	}

	//fmt.Println(url)

	reqJSON, err := json.Marshal(request) // 1
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     url,
		Body:    reqJSON,
		Headers: headers,
	}}, errs
}

// Builds enpoint url based on adapter-specific pub settings from imp.ext
func (a *TappxAdapter) buildEndpointURL(params *openrtb_ext.ExtImpTappx, test int) (string, error) {
	reqHost, reqKey, reqEndpoint := "","",""
	if params.Host != "" {
		reqHost = params.Host
	}
	if params.Endpoint != "" {
		reqEndpoint = params.Endpoint
	}
	if params.TappxKey != "" {
		reqKey = params.TappxKey
	}

	if reqHost == "" {
		return "", &errortypes.BadInput{
			Message: "Tappx host undefined",
		}
	}

	endpointParams := macros.EndpointTemplateParams{Host: reqHost}
	host, err := macros.ResolveMacros(a.endpointTemplate, endpointParams)

	if err != nil {
		return "", &errortypes.BadInput{	
			Message: "Unable to parse endpoint url template",
		}
	}

	if reqKey == "" {
		return "", &errortypes.BadInput{
			Message: "Tappx key undefined",
		}
	}

	if reqEndpoint == "" {
		return "", &errortypes.BadInput{
			Message: "Tappx endpoint undefined",
		}
	}

	thisURI := host + params.Endpoint + "?tappxkey=" + params.TappxKey

	if(test == 0){
		t := time.Now().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
		thisURI = thisURI + "&ts=" + strconv.Itoa(int(t))
	}

	thisURI = thisURI + "&v=" + TAPPX_BIDDER_VERSION

	//fmt.Println(thisURI)

	return thisURI, nil
}

func (a *TappxAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{fmt.Errorf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode)}
	}

	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	var errs []error
	for _, sb := range bidResp.SeatBid {
		for i := 0; i < len(sb.Bid); i++ {
			bid := sb.Bid[i]
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: getMediaTypeForImp(bid.ImpID, internalRequest.Imp),
			})

		}
	}
	return bidResponse, errs
}

func getMediaTypeForImp(impId string, imps []openrtb.Imp) openrtb_ext.BidType {
	mediaType := openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			}
			return mediaType
		}
	}
	return mediaType
}
