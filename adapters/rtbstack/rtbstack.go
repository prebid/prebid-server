package rtbstack

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

const endpointMacro = "http://{{.Host}}"

type adapter struct {
	endpoint string
}

// impCtx represents the context containing an OpenRTB impression and its corresponding RTBStack extension configuration.
type impCtx struct {
	imp         openrtb2.Imp
	rtbStackExt *openrtb_ext.ExtImpRTBStack
}

// extImpRTBStack is used for imp->ext when sending to rtb-stack backend
type extImpRTBStack struct {
	TagId        string                 `json:"tagid"`
	CustomParams map[string]interface{} `json:"customParams,omitempty"`
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{
			Message: "No impressions in request",
		}}
	}

	var errs []error
	var validImps []*impCtx

	for _, imp := range request.Imp {
		ext, err := preprocessImp(&imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		validImps = append(validImps, &impCtx{
			imp:         imp,
			rtbStackExt: ext,
		})
	}

	if len(validImps) == 0 {
		return nil, errs
	}

	request.Imp = nil
	for _, v := range validImps {
		request.Imp = append(request.Imp, v.imp)
	}

	endpoint := a.buildEndpointURL(validImps[0].rtbStackExt)

	var newRequest openrtb2.BidRequest
	newRequest = *request

	if request.Site != nil && request.Site.Domain == "" {
		newSite := *request.Site
		newSite.Domain = request.Site.Page
		newRequest.Site = &newSite
	}

	reqJSON, err := json.Marshal(newRequest)
	if err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: "Error parsing reqJSON object",
		}}
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     endpoint,
		Body:    reqJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}}, []error{}
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{fmt.Errorf("Unexpected status code: %d", response.StatusCode)}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	if len(bidResp.SeatBid) == 0 || len(bidResp.SeatBid[0].Bid) == 0 {
		return nil, nil
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	for _, sb := range bidResp.SeatBid {
		for i := 0; i < len(sb.Bid); i++ {
			bid := sb.Bid[i]
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: getMediaTypeForImp(sb.Bid[i].ImpID, internalRequest.Imp),
			})

		}
	}
	return bidResponse, []error{}
}

func (a *adapter) buildEndpointURL(ext *openrtb_ext.ExtImpRTBStack) string {
	return strings.Replace(a.endpoint, endpointMacro, ext.Endpoint, -1)
}

func preprocessImp(
	imp *openrtb2.Imp,
) (*openrtb_ext.ExtImpRTBStack, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{Message: err.Error()}
	}

	var impExt openrtb_ext.ExtImpRTBStack
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &impExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "Wrong RTBStack bidder ext",
		}
	}

	imp.TagID = impExt.TagId

	// create new imp->ext without odd params
	newExt := extImpRTBStack{
		TagId:        impExt.TagId,
		CustomParams: impExt.CustomParams,
	}

	// simplify content from imp->ext->bidder to imp->ext
	newImpExtForRTBStack, err := json.Marshal(newExt)
	if err != nil {
		return nil, &errortypes.BadInput{Message: err.Error()}
	}
	imp.Ext = newImpExtForRTBStack

	return &impExt, nil
}

func getMediaTypeForImp(impID string, imps []openrtb2.Imp) openrtb_ext.BidType {
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Banner != nil {
				return openrtb_ext.BidTypeBanner
			} else if imp.Video != nil {
				return openrtb_ext.BidTypeVideo
			} else if imp.Native != nil {
				return openrtb_ext.BidTypeNative
			} else if imp.Audio != nil {
				return openrtb_ext.BidTypeAudio
			}
		}
	}
	return openrtb_ext.BidTypeBanner
}
