package conversant

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type ConversantAdapter struct {
	URI string
}

func (c *ConversantAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	//Backend needs USD or it will reject the request
	if len(request.Cur) > 0 && request.Cur[0] != "USD" {
		request.Cur = []string{"USD"}
	}
	for i := 0; i < len(request.Imp); i++ {
		var bidderExt adapters.ExtImpBidder
		if err := jsonutil.Unmarshal(request.Imp[i].Ext, &bidderExt); err != nil {
			return nil, []error{&errortypes.BadInput{
				Message: fmt.Sprintf("Impression[%d] missing ext object", i),
			}}
		}

		var cnvrExt openrtb_ext.ExtImpConversant
		if err := jsonutil.Unmarshal(bidderExt.Bidder, &cnvrExt); err != nil {
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
		err := parseCnvrParams(&request.Imp[i], cnvrExt, reqInfo)
		if err != nil {
			return nil, err
		}
	}

	//create the request body
	data, err := json.Marshal(request)
	if err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: "Error in packaging request to JSON",
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
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}}, nil
}

func parseCnvrParams(imp *openrtb2.Imp, cnvrExt openrtb_ext.ExtImpConversant, reqInfo *adapters.ExtraRequestInfo) []error {
	imp.DisplayManager = "prebid-s2s"
	imp.DisplayManagerVer = "2.0.0"

	if imp.BidFloor <= 0 && cnvrExt.BidFloor > 0 {
		imp.BidFloor = cnvrExt.BidFloor
	}

	if len(cnvrExt.TagID) > 0 {
		imp.TagID = cnvrExt.TagID
	}

	// Take care not to override the global secure flag
	if (imp.Secure == nil || *imp.Secure == 0) && cnvrExt.Secure != nil {
		imp.Secure = cnvrExt.Secure
	}

	var position *adcom1.PlacementPosition
	if cnvrExt.Position != nil {
		position = adcom1.PlacementPosition(*cnvrExt.Position).Ptr()
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
			imp.Video.API = make([]adcom1.APIFramework, 0, len(cnvrExt.API))
			for _, api := range cnvrExt.API {
				imp.Video.API = append(imp.Video.API, adcom1.APIFramework(api))
			}
		}

		// Include protocols, mimes, and max duration if specified
		// These properties can also be specified in ad unit's video object,
		// but are overridden if the custom params object also contains them.

		if len(cnvrExt.Protocols) > 0 {
			imp.Video.Protocols = make([]adcom1.MediaCreativeSubtype, 0, len(cnvrExt.Protocols))
			for _, protocol := range cnvrExt.Protocols {
				imp.Video.Protocols = append(imp.Video.Protocols, adcom1.MediaCreativeSubtype(protocol))
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
	if imp.BidFloor > 0 && imp.BidFloorCur != "" && strings.ToUpper(imp.BidFloorCur) != "USD" {
		floor, err := reqInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, "USD")
		if err != nil {
			return []error{&errortypes.BadInput{
				Message: fmt.Sprintf("Unable to convert provided bid floor currency from %s to USD", imp.BidFloorCur),
			}}
		}
		imp.BidFloorCur = "USD"
		imp.BidFloor = floor
	}
	return nil
}

func (c *ConversantAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil // no bid response
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d", response.StatusCode),
		}}
	}

	var resp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &resp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("bad server response: %d. ", err),
		}}
	}

	if len(resp.SeatBid) == 0 {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Empty bid request",
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

func getBidType(impId string, imps []openrtb2.Imp) openrtb_ext.BidType {
	for _, imp := range imps {
		if imp.ID == impId {
			switch {
			case imp.Native != nil:
				return openrtb_ext.BidTypeNative
			case imp.Audio != nil:
				return openrtb_ext.BidTypeAudio
			case imp.Video != nil:
				return openrtb_ext.BidTypeVideo
			}
			break
		}
	}
	return openrtb_ext.BidTypeBanner
}

// Builder builds a new instance of the Conversant adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &ConversantAdapter{
		URI: config.Endpoint,
	}
	return bidder, nil
}
