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
)

type DmxAdapter struct {
	endpoint    string
	publisherId string
}

func NewDmxBidder(endpoint string, publisher_id string) *DmxAdapter {
	return &DmxAdapter{endpoint: endpoint, publisherId: publisher_id}
}

type dmxExt struct {
	Bidder dmxParams `json:"bidder"`
}

type dmxParams struct {
	TagId       string `json:"tagid,omitempty"`
	DmxId       string `json:"dmxid,omitempty"`
	MemberId    string `json:"memberid,omitempty"`
	PublisherId string `json:"publisher_id"`
}

func ReturnPubId(str1, str2 string) string {
	if str1 != "" {
		return str1
	}
	if str2 != "" {
		return str2
	}
	return ""

}

func (adapter *DmxAdapter) MakeRequests(request *openrtb.BidRequest, req *adapters.ExtraRequestInfo) (reqsBidder []*adapters.RequestData, errs []error) {
	var dmxBidderStaticInfo dmxExt
	var imps []openrtb.Imp
	var rootExtInfo dmxExt
	var publisherId string
	if len(request.Imp) < 1 {
		errs = append(errs, errors.New("imp is empty no auction possible"))
		return nil, errs
	}

	if len(request.Imp) >= 1 {
		err := json.Unmarshal(request.Imp[0].Ext, &rootExtInfo)
		if err != nil {
			errs = append(errs, err)
		} else {
			publisherId = ReturnPubId(rootExtInfo.Bidder.PublisherId, rootExtInfo.Bidder.MemberId)

		}
	}
	if err := json.Unmarshal(request.Ext, &dmxBidderStaticInfo); err != nil {
		errs = append(errs, err)
	}

	if dmxBidderStaticInfo.Bidder.PublisherId != "" && publisherId == "" {
		request.Site.Publisher = &openrtb.Publisher{ID: dmxBidderStaticInfo.Bidder.PublisherId}
	} else {
		if request.Site.Publisher != nil {
			request.Site.Publisher.ID = adapter.publisherId
		}
	}

	if request.App != nil {
		request.Site = nil
		request.App.Publisher = &openrtb.Publisher{ID: publisherId}
	}
	if request.Site != nil {

		request.Site.Publisher.ID = publisherId
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

			var intVal int8
			intVal = 1
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
				imps = fetchFailedParams(failedParams, inst, ins, imps, banner, intVal)
			} else {
				imps = fetchParams(params, inst, ins, imps, banner, intVal)
			}

		}

	}

	request.Imp = imps

	oJson, _ := json.Marshal(request)
	headers := http.Header{}
	headers.Add("Content-Type", "Application/json;charset=utf-8")
	reqBidder := &adapters.RequestData{
		Method:  "POST",
		Uri:     adapter.endpoint, //adapter.endpoint,
		Body:    oJson,
		Headers: headers,
	}

	if request.User == nil {
		if request.App == nil {
			return nil, []error{errors.New("no user Id found and AppID, no request to DMX")}
		}
	}

	reqsBidder = append(reqsBidder, reqBidder)
	return
}

func (adapter *DmxAdapter) MakeBids(request *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var errs []error

	if http.StatusNoContent == response.StatusCode {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("No content to be return"),
		}}
	}

	if http.StatusBadRequest == response.StatusCode {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Bad formated request"),
		}}
	}

	if http.StatusOK != response.StatusCode {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Something is really wrong"),
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

func fetchFailedParams(params dmxParams, inst openrtb.Imp, ins openrtb.Imp, imps []openrtb.Imp, banner openrtb.Banner, intVal int8) []openrtb.Imp {
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
