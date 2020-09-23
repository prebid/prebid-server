package conversant

import (
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
)

type ConversantAdapter struct {
	URI string
}

func (c ConversantAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	for i := 0; i < len(request.Imp); i++ {
		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(request.Imp[i].Ext, &bidderExt); err != nil {
			return nil, []error{&errortypes.BadInput{
				Message: fmt.Sprintf("Impression[%d] missing ext object", i),
			}}
		}

		var cnvrExt openrtb_ext.ExtImpConversant
		if err := json.Unmarshal(bidderExt.Bidder, &cnvrExt); err != nil {
			return nil, []error{&errortypes.BadInput{
				Message: fmt.Sprintf("Impression[%d] missing ext.bidder object", i),
			}}
		}

		if cnvrExt.SiteID == "" {
			return nil, []error{&errortypes.BadInput{
				Message: fmt.Sprintf("Impression[%d] requires ext.bidder.site_id", i),
			}}
		}

		if i == 0 {
			if request.Site != nil {
				tmpSite := *request.Site
				request.Site = &tmpSite
				request.Site.ID = cnvrExt.SiteID
			} else if request.App != nil {
				tmpApp := *request.App
				request.App = &tmpApp
				request.App.ID = cnvrExt.SiteID
			}
		}
		parseCnvrParams(&request.Imp[i], cnvrExt)
	}

	//create the request body
	data, err := json.Marshal(request)
	if err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Error in packaging request to JSON"),
		}}
	}
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     c.URI,
		Body:    data,
		Headers: headers,
	}}, nil
}

func parseCnvrParams(imp *openrtb.Imp, cnvrExt openrtb_ext.ExtImpConversant) {
	imp.DisplayManager = "prebid-s2s"
	imp.DisplayManagerVer = "2.0.0"
	imp.BidFloor = cnvrExt.BidFloor
	imp.TagID = cnvrExt.TagID

	// Take care not to override the global secure flag
	if (imp.Secure == nil || *imp.Secure == 0) && cnvrExt.Secure != nil {
		imp.Secure = cnvrExt.Secure
	}

	var position *openrtb.AdPosition
	if cnvrExt.Position != nil {
		position = openrtb.AdPosition(*cnvrExt.Position).Ptr()
	}
	if imp.Banner != nil {
		tmpBanner := *imp.Banner
		imp.Banner = &tmpBanner
		imp.Banner.Pos = position

	} else if imp.Video != nil {
		tmpVideo := *imp.Video
		imp.Video = &tmpVideo
		imp.Video.Pos = position

		if len(cnvrExt.API) > 0 {
			imp.Video.API = make([]openrtb.APIFramework, 0, len(cnvrExt.API))
			for _, api := range cnvrExt.API {
				imp.Video.API = append(imp.Video.API, openrtb.APIFramework(api))
			}
		}

		// Include protocols, mimes, and max duration if specified
		// These properties can also be specified in ad unit's video object,
		// but are overridden if the custom params object also contains them.

		if len(cnvrExt.Protocols) > 0 {
			imp.Video.Protocols = make([]openrtb.Protocol, 0, len(cnvrExt.Protocols))
			for _, protocol := range cnvrExt.Protocols {
				imp.Video.Protocols = append(imp.Video.Protocols, openrtb.Protocol(protocol))
			}
		}

		if len(cnvrExt.MIMEs) > 0 {
			imp.Video.MIMEs = make([]string, len(cnvrExt.MIMEs))
			copy(imp.Video.MIMEs, cnvrExt.MIMEs)
		}

		if cnvrExt.MaxDuration != nil {
			imp.Video.MaxDuration = *cnvrExt.MaxDuration
		}
	}
}

func (c ConversantAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil // no bid response
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d", response.StatusCode),
		}}
	}

	var resp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &resp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("bad server response: %d. ", err),
		}}
	}

	if len(resp.SeatBid) == 0 {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Empty bid request"),
		}}
	}

	bids := resp.SeatBid[0].Bid
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(bids))
	for i := 0; i < len(bids); i++ {
		bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
			Bid:     &bids[i],
			BidType: getBidType(bids[i].ImpID, internalRequest.Imp),
		})
	}
	return bidResponse, nil
}

func getBidType(impId string, imps []openrtb.Imp) openrtb_ext.BidType {
	bidType := openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Video != nil {
				bidType = openrtb_ext.BidTypeVideo
			}
			break
		}
	}
	return bidType
}

func NewConversantBidder(endpoint string) *ConversantAdapter {
	return &ConversantAdapter{URI: endpoint}
}
