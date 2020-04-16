package dmx

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
	"net/url"
)

type DmxAdapter struct {
	endpoint    string
	publisherId string
}

func NewDmxBidder(endpoint string) *DmxAdapter {
	return &DmxAdapter{endpoint: endpoint}
}

type dmxExt struct {
	Bidder dmxParams `json:"bidder"`
}

type dmxParams struct {
	TagId       string `json:"tagid,omitempty"`
	DmxId       string `json:"dmxid,omitempty"`
	MemberId    string `json:"memberid,omitempty"`
	PublisherId string `json:"publisher_id,omitempty"`
	SellerId    string `json:"seller_id,omitempty"`
}

func UserSellerOrPubId(str1, str2 string) string {
	if str1 != "" {
		return str1
	}
	return str2
}

func (adapter *DmxAdapter) MakeRequests(request *openrtb.BidRequest, req *adapters.ExtraRequestInfo) (reqsBidder []*adapters.RequestData, errs []error) {
	var dmxBidderStaticInfo dmxExt
	var imps []openrtb.Imp
	var rootExtInfo dmxExt
	var publisherId string
	var sellerId string

	var dmxReq = request

	if len(request.Imp) >= 1 {
		err := json.Unmarshal(request.Imp[0].Ext, &rootExtInfo)
		if err != nil {
			errs = append(errs, err)
		} else {
			publisherId = UserSellerOrPubId(rootExtInfo.Bidder.PublisherId, rootExtInfo.Bidder.MemberId)
			sellerId = rootExtInfo.Bidder.SellerId
		}
	}
	if err := json.Unmarshal(request.Ext, &dmxBidderStaticInfo); err != nil {
		errs = append(errs, err)
	}

	if dmxBidderStaticInfo.Bidder.PublisherId != "" && publisherId == "" {
		publisherId = dmxBidderStaticInfo.Bidder.PublisherId
	}

	if dmxReq.App != nil {
		dmxReq.Site = nil
		dmxReq.App.Publisher.ID = publisherId
	}
	if dmxReq.Site != nil {
		if dmxReq.Site.Publisher == nil {
			dmxReq.Site.Publisher = &openrtb.Publisher{ID: publisherId}
		} else {
			if dmxReq.Site.Publisher != nil {
				dmxReq.Site.Publisher.ID = adapter.publisherId
			} else {
				dmxReq.Site.Publisher.ID = publisherId
			}
		}
	}

	for _, inst := range request.Imp {
		var banner openrtb.Banner
		var ins openrtb.Imp

		if len(inst.Banner.Format) != 0 {
			banner = openrtb.Banner{
				W:      &inst.Banner.Format[0].W,
				H:      &inst.Banner.Format[0].H,
				Format: inst.Banner.Format,
			}

			const intVal int8 = 1
			var params dmxExt

			source := (*json.RawMessage)(&inst.Ext)
			if err := json.Unmarshal(*source, &params); err != nil {
				errs = append(errs, err)
			}
			if params.Bidder.PublisherId == "" && params.Bidder.MemberId == "" {
				var failedParams dmxParams
				if err := json.Unmarshal(inst.Ext, &failedParams); err != nil {
					errs = append(errs, err)
					return nil, errs
				}
				imps = fetchAlternativeParams(failedParams, inst, ins, imps, banner, intVal)
			} else {
				imps = fetchParams(params, inst, ins, imps, banner, intVal)
			}

		}

	}

	dmxReq.Imp = imps

	oJson, err := json.Marshal(request)

	if err != nil {
		errs = append(errs, err)
	}
	headers := http.Header{}
	headers.Add("Content-Type", "Application/json;charset=utf-8")
	reqBidder := &adapters.RequestData{
		Method:  "POST",
		Uri:     adapter.endpoint + addParams(sellerId), //adapter.endpoint,
		Body:    oJson,
		Headers: headers,
	}

	if dmxReq.User == nil {
		if dmxReq.App == nil {
			return nil, []error{errors.New("no user Id found and AppID, no request to DMX")}
		}
	}

	reqsBidder = append(reqsBidder, reqBidder)
	return
}

func (adapter *DmxAdapter) MakeBids(request *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var errs []error

	if http.StatusNoContent == response.StatusCode {
		return nil, nil
	}

	if http.StatusBadRequest == response.StatusCode {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code 400"),
		}}
	}

	if http.StatusOK != response.StatusCode {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected response no status code"),
		}}
	}

	var bidResp openrtb.BidResponse

	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidType, err := getMediaTypeForImp(sb.Bid[i].ImpID, request.Imp)
			if err != nil {
				errs = append(errs, err)
			} else {
				b := &adapters.TypedBid{
					Bid:     &sb.Bid[i],
					BidType: bidType,
				}
				bidResponse.Bids = append(bidResponse.Bids, b)
			}
		}
	}
	return bidResponse, errs

}

func fetchAlternativeParams(params dmxParams, inst openrtb.Imp, ins openrtb.Imp, imps []openrtb.Imp, banner openrtb.Banner, intVal int8) []openrtb.Imp {
	if params.TagId != "" {
		ins = openrtb.Imp{
			ID:     inst.ID,
			TagID:  params.TagId,
			Banner: &banner,
			Ext:    inst.Ext,
			Secure: &intVal,
		}
	}

	if params.DmxId != "" {
		ins = openrtb.Imp{
			ID:     inst.ID,
			TagID:  params.DmxId,
			Banner: &banner,
			Ext:    inst.Ext,
			Secure: &intVal,
		}
	}
	imps = append(imps, ins)
	return imps
}

func fetchParams(params dmxExt, inst openrtb.Imp, ins openrtb.Imp, imps []openrtb.Imp, banner openrtb.Banner, intVal int8) []openrtb.Imp {
	if params.Bidder.TagId != "" {
		ins = openrtb.Imp{
			ID:     inst.ID,
			TagID:  params.Bidder.TagId,
			Banner: &banner,
			Ext:    inst.Ext,
			Secure: &intVal,
		}
	}

	if params.Bidder.DmxId != "" {
		ins = openrtb.Imp{
			ID:     inst.ID,
			TagID:  params.Bidder.DmxId,
			Banner: &banner,
			Ext:    inst.Ext,
			Secure: &intVal,
		}
	}
	imps = append(imps, ins)
	return imps
}

func addParams(str string) string {
	if str != "" {
		return "?sellerid=" + url.QueryEscape(str)
	}
	return ""
}

func getMediaTypeForImp(impID string, imps []openrtb.Imp) (openrtb_ext.BidType, error) {
	mediaType := openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Banner == nil && imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			}
			return mediaType, nil
		}
	}

	// This shouldnt happen. Lets handle it just incase by returning an error.
	return "", &errortypes.BadInput{
		Message: fmt.Sprintf("Failed to find impression \"%s\" ", impID),
	}
}
