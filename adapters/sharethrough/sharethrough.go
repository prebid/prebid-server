package sharethrough

import (
	"bytes"
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
	"strconv"
)

const hbSource = "prebid-server"
const strVersion = "1.0.0"

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

func (s SharethroughAdapter) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
	//fmt.Printf("in sharethrough adapter\nrequest: %+v\n", request)
	errs := make([]error, 0, len(request.Imp))
	headers := http.Header{}
	var potentialRequests []*adapters.RequestData

	headers.Add("Content-Type", "text/plain;charset=utf-8")
	headers.Add("Accept", "application/json")

	for i := 0; i < len(request.Imp); i++ {
		imp := request.Imp[i]

		fmt.Printf("processing imp")

		var extBtlrParams openrtb_ext.ExtImpSharethroughExt
		if err := json.Unmarshal(imp.Ext, &extBtlrParams); err != nil {
			return nil, []error{err}
		}

		var extUser struct {
			Consent string `json:"consent"`
		}
		if err := json.Unmarshal(request.User.Ext, &extUser); err != nil {
			extUser.Consent = ""
		}
		fmt.Println(extUser)

		// todo: get gdpr from Regs
		//var extRegs struct{ Gdpr int }
		//if err := json.Unmarshal(request.Regs.Ext, &extRegs); err != nil {
		//	extRegs.Gdpr = 0
		//}

		pKey := extBtlrParams.Bidder.Pkey

		potentialRequests = append(potentialRequests, &adapters.RequestData{
			Method: "POST",
			Uri: generateHBUri(s.URI, hbUriParams{
				Pkey:  pKey,
				BidID: imp.ID,
				//ConsentRequired: !(extRegs.Gdpr == 0),
				ConsentString: extUser.Consent,
				Iframe:        extBtlrParams.Bidder.Iframe,
				IframeWidth:   extBtlrParams.Bidder.IframeSize[0],
				IframeHeight:  extBtlrParams.Bidder.IframeSize[1],
			}),
			Body:    nil,
			Headers: headers,
		})
	}

	return potentialRequests, errs
}

func (s SharethroughAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{fmt.Errorf("unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode)}
	}

	var strBidResp openrtb_ext.ExtImpSharethroughResponse
	if err := json.Unmarshal(response.Body, &strBidResp); err != nil {
		return nil, []error{err}
	}

	br, bidderResponseErr := butlerToOpenRTBResponse(externalRequest, strBidResp)

	return br, bidderResponseErr
}

func butlerToOpenRTBResponse(btlrReq *adapters.RequestData, strResp openrtb_ext.ExtImpSharethroughResponse) (*adapters.BidderResponse, []error) {
	var errs []error
	bidResponse := adapters.NewBidderResponse()

	bidResponse.Currency = "USD"
	typedBid := &adapters.TypedBid{BidType: openrtb_ext.BidTypeNative}
	creative := strResp.Creatives[0]

	btlrParams, _ := parseHBUri(btlrReq.Uri)

	bid := &openrtb.Bid{
		AdID:   strResp.AdServerRequestID,
		ID:     strResp.BidID,
		ImpID:  btlrParams.BidID,
		Price:  creative.CPM,
		CID:    creative.Metadata.CampaignKey,
		CrID:   creative.Metadata.CreativeKey,
		DealID: creative.Metadata.DealID,
		AdM:    getAdMarkup(strResp, btlrParams),
		// NURL: creative.Beacons.WinNotification[0] // what do we do with other notification URLs ???
	}

	typedBid.Bid = bid
	bidResponse.Bids = append(bidResponse.Bids, typedBid)

	return bidResponse, errs
}

func getAdMarkup(strResp openrtb_ext.ExtImpSharethroughResponse, params *hbUriParams) string {
	strRespId := fmt.Sprintf("str_response_%s", strResp.BidID)
	jsonPayload, err := json.Marshal(strResp)

	if err != nil {
		//handle error
		fmt.Printf("ERROR: %s\n", err)
	}

	tmplBody := `
		<div data-str-native-key="{{.Pkey}}" data-stx-response-name="{{.StrRespId}}"></div>
	 	<script>var {{.StrRespId}} = "{{.B64EncodedJson}}"</script>
	`

	if params.Iframe {
		tmplBody = tmplBody + `
			<script src="//native.sharethrough.com/assets/sfp.js"></script>
		`
	} else {
		tmplBody = tmplBody + `
			<script src="//native.sharethrough.com/assets/sfp-set-targeting.js"></script>
	    	<script>
		     (function() {
		       if (!(window.STR && window.STR.Tag) && !(window.top.STR && window.top.STR.Tag)) {
		         var sfp_js = document.createElement('script');
		         sfp_js.src = "//native.sharethrough.com/assets/sfp.js";
		         sfp_js.type = 'text/javascript';
		         sfp_js.charset = 'utf-8';
		         try {
		             window.top.document.getElementsByTagName('body')[0].appendChild(sfp_js);
		         } catch (e) {
		           console.log(e);
		         }
		       }
		     })()
		   </script>
	`

	}

	tmpl, err := template.New("sfpjs").Parse(tmplBody)
	if err != nil {
		// handle error
		fmt.Printf("ERROR TEMPLATE: %s\n", err)
	}

	var buf []byte
	templatedBuf := bytes.NewBuffer(buf)

	b64EncodedJson := base64.StdEncoding.EncodeToString(jsonPayload)
	err = tmpl.Execute(templatedBuf, struct {
		Pkey           string
		StrRespId      template.JS
		B64EncodedJson string
	}{
		params.Pkey,
		template.JS(strRespId),
		b64EncodedJson,
	})

	if err != nil {
		// handle error
		fmt.Printf("ERROR TEMPLATE Execute: %s\n", err)

	}

	return templatedBuf.String()
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

type hbUriParams struct {
	Pkey               string
	BidID              string
	ConsentRequired    bool
	ConsentString      string
	InstantPlayCapable bool
	Iframe             bool
	IframeHeight       int
	IframeWidth        int
}

func generateHBUri(baseUrl string, params hbUriParams) string {
	v := url.Values{}
	v.Set("placement_key", params.Pkey)
	v.Set("bidId", params.BidID)
	v.Set("consent_required", fmt.Sprintf("%t", params.ConsentRequired))

	if params.ConsentRequired {
		v.Set("consent_string", params.ConsentString)
	}

	v.Set("instant_play_capable", fmt.Sprintf("%t", params.InstantPlayCapable))
	v.Set("stayInIframe", fmt.Sprintf("%t", params.Iframe))
	v.Set("iframeHeight", string(params.IframeHeight))
	v.Set("iframeWidth", string(params.IframeWidth))

	v.Set("hbVersion", "test-version") // todo: figure out the version dynamically
	v.Set("hbSource", hbSource)
	v.Set("strVersion", strVersion)

	return baseUrl + "?" + v.Encode()
}

func parseHBUri(uri string) (*hbUriParams, error) {
	btlrUrl, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	params := btlrUrl.Query()
	iframeHeight, _ := strconv.ParseInt(params.Get("iframeHeight"), 10, 64)
	iframeWidth, _ := strconv.ParseInt(params.Get("iframeWidth"), 10, 64)

	return &hbUriParams{
		Pkey:         params.Get("placement_key"),
		BidID:        params.Get("bidId"),
		Iframe:       params.Get("stayInIframe") == "true",
		IframeHeight: int(iframeHeight),
		IframeWidth:  int(iframeWidth),
	}, nil
}
