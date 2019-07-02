package tappx

import (
	"encoding/json"
	"fmt"
	//"text/template"
	"strings"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
	"strconv"
	"time"
)

const TAPPX_BIDDER_VERSION = "1.0"

type TappxAdapter struct {
	http *adapters.HTTPAdapter
	URL  string
}

func NewTappxAdapter(config *adapters.HTTPAdapterConfig, endpoint string) *TappxAdapter {
	return NewTappxBidder(adapters.NewHTTPAdapter(config).Client, endpoint)
}

func NewTappxBidder(client *http.Client, endpoint string) *TappxAdapter {
	a := &adapters.HTTPAdapter{Client: client}

	return &TappxAdapter{
		http: a,
		URL:  endpoint,
	}
}

type tappxParams struct {
	Host string `json:"host"`
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
	errs := make([]error, 0, len(request.Imp))

	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(request.Imp[0].Ext, &bidderExt); err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	var tappxExt openrtb_ext.ExtImpTappx
	if err := json.Unmarshal(bidderExt.Bidder, &tappxExt); err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	if tappxExt.TappxKey == "" {
		return nil, []error{&errortypes.BadInput{
			Message: "Tappx key undefined",
		}}
	}
	if tappxExt.Endpoint == "" {
		return nil, []error{&errortypes.BadInput{
			Message: "Endpoint undefined",
		}}
	}
	if tappxExt.Host == "" {
		return nil, []error{&errortypes.BadInput{
			Message: "Host undefined",
		}}
	}

	reqJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	t := time.Now().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))

	url := strings.Replace(a.URL, "you.new.the.tappx.host.com", tappxExt.Host, -1)

	thisURI := url + tappxExt.Endpoint + "?appkey=" + tappxExt.TappxKey + "&ts=" + strconv.Itoa(int(t)) + "&v=" + TAPPX_BIDDER_VERSION

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     thisURI,
		Body:    reqJSON,
		Headers: headers,
	}}, errs
}

/*func (a *TappxAdapter) buildEndpointURL(endpoint string) (string, error) {
	reqHost := ""
	if endpoint != "" {
		reqHost = endpoint
	}
	endpointParams := macros.EndpointTemplateParams{Host: reqHost}
	return macros.ResolveMacros(a.Host, endpointParams)
}*/

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
