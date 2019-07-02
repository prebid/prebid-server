package tappx

import (
	"encoding/json"
	"fmt"
	"net/http"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/mxmCherry/openrtb"
)

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
	TappxKey	string            `json:"tappxkey"`
	Endpoint    string            `json:"endpoint"`
}

func (a *TappxAdapter) Name() string {
	return "tappx"
}

func (a *TappxAdapter) SkipNoCookies() bool {
	return false
}

/*func (a *TappxAdapter) Call(ctx context.Context, request *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {

}*/

func (a *TappxAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	errs := make([]error, 0, len(request.Imp))

	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(request.Imp[0].Ext, &bidderExt); err != nil {
		//fmt.Println("ERROR")
	}

	var tappxExt openrtb_ext.ExtImpTappx
	if err := json.Unmarshal(bidderExt.Bidder, &tappxExt); err != nil {
		//fmt.Println("ERROR")
	}

	if(tappxExt.TappxKey != ""){
		//fmt.Println("ERROR")
	}
	if(tappxExt.Endpoint != ""){
		//fmt.Println("ERROR")
	}	

	reqJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	thisURI := a.URL + tappxExt.Endpoint + "?test=1&appkey=" + tappxExt.TappxKey

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
