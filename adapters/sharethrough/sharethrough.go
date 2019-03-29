package sharethrough

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"html/template"
	"net/http"
	"net/url"
)

const hbEndpoint = "http://dumb-waiter.sharethrough.com/header-bid/v1"

func NewSharethroughBidder(endpoint string) *SharethroughAdapter {
	return &SharethroughAdapter{URI: endpoint}
}

// SharethroughAdapter converts the Sharethrough Adserver response into a
// prebid server compatible format
type SharethroughAdapter struct {
	URI string
}

// Name returns the adapter name as a string
func (s SharethroughAdapter) Name() string {
	return "sharethrough"
}

type params struct {
	BidID        string `json:"bidId"`
	PlacementKey string `json:"placement_key"`
	HBVersion    string `json:"hbVersion"`
	StrVersion   string `json:"strVersion"`
	HBSource     string `json:"hbSource"`
}

func (s SharethroughAdapter) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
	fmt.Println("in sharethrough adapter")
	pKeys := make([]string, 0, len(request.Imp))
	errs := make([]error, 0, len(request.Imp))
	headers := http.Header{}
	var potentialRequests []*adapters.RequestData

	headers.Add("Content-Type", "text/plain;charset=utf-8")
	headers.Add("Accept", "application/json")

	for i := 0; i < len(request.Imp); i++ {
		pKey, err := preprocess(&request.Imp[i])
		if pKey != "" {
			pKeys = append(pKeys, pKey)
		}

		// If the preprocessing failed, the server won't be able to bid on this Imp. Delete it, and note the error.
		if err != nil {
			errs = append(errs, err)
			request.Imp = append(request.Imp[:i], request.Imp[i+1:]...)
			i--
			continue
		}

		//hbURI := generateHBUri(pKey, "testBidID-"+strconv.Itoa(i))
		potentialRequests = append(potentialRequests, &adapters.RequestData{
			Method:  "POST",
			Uri:     s.URI + "?pkey=" + pKey,
			Body:    nil,
			Headers: headers,
		})
	}

	return potentialRequests, errs
}

func (s SharethroughAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	fmt.Printf("internal request: %v\n", internalRequest)
	fmt.Printf("external request: %v\n", externalRequest)

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
	var strBidResp openrtb_ext.ExtImpSharethroughResponse
	if err := json.Unmarshal(response.Body, &strBidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponse()

	br, _ := butlerToOpenRTBResponse(externalRequest, strBidResp)
	fmt.Printf("br code: %v\n", br)
	var errs []error
	for _, sb := range bidResp.SeatBid {
		for i := 0; i < len(sb.Bid); i++ {
			bid := sb.Bid[i]
			if bidType, err := getMediaTypeForBid(&bid); err == nil {
				bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
					Bid:     &bid,
					BidType: bidType,
				})
			} else {
				errs = append(errs, err)
			}
		}
	}
	for _, bid := range bidResponse.Bids {
		fmt.Printf("bidResponse.Bids: %+v\n", bid)
	}
	if len(errs) > 0 {
		for _, err := range errs {
			fmt.Printf("error: %s\n", err)
		}
	}
	return bidResponse, errs
}

func butlerToOpenRTBResponse(btlrReq *adapters.RequestData, strResp openrtb_ext.ExtImpSharethroughResponse) (*adapters.BidderResponse, []error) {
	var errs []error
	bidResponse := adapters.NewBidderResponse()

	bidResponse.Currency = "USD"
	typedBid := &adapters.TypedBid{BidType: openrtb_ext.BidTypeNative}
	creative := strResp.Creatives[0]

	btlrUrl, err := url.Parse(btlrReq.Uri)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}
	pkey := btlrUrl.Query().Get("pkey")

	bid := &openrtb.Bid{
		ID:    strResp.BidID,
		ImpID: strResp.AdServerRequestID, // MAYBE?
		Price: creative.CPM,
		// NURL: creative.Beacons.WinNotification[0] // what do we do with other notification URLs ???
		CID:    creative.Metadata.CampaignKey,
		CrID:   creative.Metadata.CreativeKey,
		DealID: creative.Metadata.DealID,
		AdM:    getAdMarkup(strResp, pkey),
	}

	typedBid.Bid = bid
	bidResponse.Bids = append(bidResponse.Bids, typedBid)

	return bidResponse, errs
}

func getAdMarkup(strResp openrtb_ext.ExtImpSharethroughResponse, pkey string) string {
	strRespId := fmt.Sprintf("str_response_%s", strResp.BidID)
	//b64EncodedJson := base64.NewEncoding(json.Mar)
	//tmpl := `
	//	<div data-str-native-key="{{pkey}}" data-stx-response-name="{{strRespId}}"></div>
	//  	<script>var {{strRespId}} = "${b64EncodeUnicode(JSON.stringify(body))}"</script>
	//`
	//
	//let adMarkup = `
	//  <div data-str-native-key="${req.data.placement_key}" data-stx-response-name="${strRespId}">
	//  </div>
	//  <script>var ${strRespId} = "${b64EncodeUnicode(JSON.stringify(body))}"</script>
	//`
	//
	//if (req.strData.stayInIframe) {
	//	// Don't break out of iframe
	//	adMarkup = adMarkup + `<script src="//native.sharethrough.com/assets/sfp.js"></script>`
	//} else {
	//	// Break out of iframe
	//	adMarkup = adMarkup + `
	//    <script src="//native.sharethrough.com/assets/sfp-set-targeting.js"></script>
	//    <script>
	//      (function() {
	//        if (!(window.STR && window.STR.Tag) && !(window.top.STR && window.top.STR.Tag)) {
	//          var sfp_js = document.createElement('script');
	//          sfp_js.src = "//native.sharethrough.com/assets/sfp.js";
	//          sfp_js.type = 'text/javascript';
	//          sfp_js.charset = 'utf-8';
	//          try {
	//              window.top.document.getElementsByTagName('body')[0].appendChild(sfp_js);
	//          } catch (e) {
	//            console.log(e);
	//          }
	//        }
	//      })()
	//  </script>`
	//}
}

func getMediaTypeForBid(bid *openrtb.Bid) (openrtb_ext.BidType, error) {
	return openrtb_ext.BidTypeNative, nil
	// var impExt struct {
	// 	Sharethrough struct {
	// 		BidType int `json:"bid_type"`
	// 	} `json:"sharethrough"`
	// }
	// if err := json.Unmarshal(bid.Ext, &impExt); err != nil {
	// 	return "", err
	// }
	// switch impExt.Sharethrough.BidType {
	// case 0:
	// 	return openrtb_ext.BidTypeBanner, nil
	// case 1:
	// 	return openrtb_ext.BidTypeVideo, nil
	// case 2:
	// 	return openrtb_ext.BidTypeNative, nil
	// default:
	// 	return "", fmt.Errorf("Unrecognized bid_ad_type in response from sharethrough: %d", impExt.Sharethrough.BidType)
	// }
}

func preprocess(imp *openrtb.Imp) (pKey string, err error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return "", err
	}

	var sharethroughExt openrtb_ext.ExtImpSharethrough
	if err := json.Unmarshal(bidderExt.Bidder, &sharethroughExt); err != nil {
		return "", err
	}

	return sharethroughExt.PlacementKey, nil
}

func generateHBUri(pKey string, bidID string) string {
	return "http://localhost:8000/bid"
}

// func generateHBUri(pKey string, bidID string) string {
// 	v := url.Values{}
// 	v.Set("placement_key", pKey)
// 	v.Set("bidId", bidID)
// 	v.Set("hbVersion", "test-version")
// 	v.Set("hbSource", "prebid-server")

// 	return hbEndpoint + "?" + v.Encode()
// }
