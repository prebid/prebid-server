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
	"strings"
)

type DmxAdapter struct {
	endpoint string
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
	var imps []openrtb.Imp
	var rootExtInfo dmxExt
	var publisherId string
	var sellerId string
	var userExt openrtb_ext.ExtUser
	var anyHasId = false
	var reqCopy openrtb.BidRequest = *request
	var dmxReq *openrtb.BidRequest = &reqCopy

	if request.User == nil {
		if request.App == nil {
			return nil, []error{errors.New("No user id or app id found. Could not send request to DMX.")}
		}
	}

	if len(request.Imp) >= 1 {
		err := json.Unmarshal(request.Imp[0].Ext, &rootExtInfo)
		if err != nil {
			errs = append(errs, err)
		} else {
			publisherId = UserSellerOrPubId(rootExtInfo.Bidder.PublisherId, rootExtInfo.Bidder.MemberId)
			sellerId = rootExtInfo.Bidder.SellerId
		}
	}

	if request.App != nil {
		appCopy := *request.App
		appPublisherCopy := *request.App.Publisher
		dmxReq.App = &appCopy
		dmxReq.App.Publisher = &appPublisherCopy
		if dmxReq.App.ID != "" {
			anyHasId = true
		}
	} else {
		dmxReq.App = nil
	}

	if request.Site != nil {
		siteCopy := *request.Site
		sitePublisherCopy := *request.Site.Publisher
		dmxReq.Site = &siteCopy
		dmxReq.Site.Publisher = &sitePublisherCopy
		if dmxReq.Site.Publisher != nil {
			dmxReq.Site.Publisher.ID = publisherId
		} else {
			dmxReq.Site.Publisher = &openrtb.Publisher{ID: publisherId}
		}
	} else {
		dmxReq.Site = nil
	}

	if request.User != nil {
		userCopy := *request.User
		dmxReq.User = &userCopy
	} else {
		dmxReq.User = nil
	}

	if dmxReq.User != nil {
		if dmxReq.User.ID != "" {
			anyHasId = true
		}
		if dmxReq.User.Ext != nil {
			if err := json.Unmarshal(dmxReq.User.Ext, &userExt); err == nil {
				if len(userExt.Eids) > 0 || (userExt.DigiTrust != nil && userExt.DigiTrust.ID != "") {
					anyHasId = true
				}
			}
		}
	}

	if anyHasId == false {
		return nil, []error{errors.New("This request contained no identifier")}
	}

	for _, inst := range dmxReq.Imp {
		var banner *openrtb.Banner
		var video *openrtb.Video
		var ins openrtb.Imp
		var params dmxExt
		const intVal int8 = 1
		source := (*json.RawMessage)(&inst.Ext)
		if err := json.Unmarshal(*source, &params); err != nil {
			errs = append(errs, err)
		}
		if isDmxParams(params.Bidder) {
			if inst.Banner != nil {
				if len(inst.Banner.Format) != 0 {
					banner = inst.Banner
					if params.Bidder.PublisherId != "" || params.Bidder.MemberId != "" {
						imps = fetchParams(params, inst, ins, imps, banner, nil, intVal)
					} else {
						return nil, []error{errors.New("Missing Params for auction to be send")}
					}
				}
			}

			if inst.Video != nil {
				video = inst.Video
				if params.Bidder.PublisherId != "" || params.Bidder.MemberId != "" {
					imps = fetchParams(params, inst, ins, imps, nil, video, intVal)
				} else {
					return nil, []error{errors.New("Missing Params for auction to be send")}
				}
			}
		}

	}

	dmxReq.Imp = imps

	oJson, err := json.Marshal(dmxReq)

	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	reqBidder := &adapters.RequestData{
		Method:  "POST",
		Uri:     adapter.endpoint + addParams(sellerId), //adapter.endpoint,
		Body:    oJson,
		Headers: headers,
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
				if b.BidType == openrtb_ext.BidTypeVideo {
					b.Bid.AdM = videoImpInsertion(b.Bid)
				}
				bidResponse.Bids = append(bidResponse.Bids, b)
			}
		}
	}
	return bidResponse, errs

}

func fetchParams(params dmxExt, inst openrtb.Imp, ins openrtb.Imp, imps []openrtb.Imp, banner *openrtb.Banner, video *openrtb.Video, intVal int8) []openrtb.Imp {
	if params.Bidder.TagId != "" {
		ins = openrtb.Imp{
			ID:     inst.ID,
			TagID:  params.Bidder.TagId,
			Ext:    inst.Ext,
			Secure: &intVal,
		}
	}

	if params.Bidder.DmxId != "" {
		ins = openrtb.Imp{
			ID:     inst.ID,
			TagID:  params.Bidder.DmxId,
			Ext:    inst.Ext,
			Secure: &intVal,
		}
	}
	if banner != nil {
		ins.Banner = banner
	}

	if video != nil {
		ins.Video = video
	}

	if ins.TagID == "" {
		return imps
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

func videoImpInsertion(bid *openrtb.Bid) string {
	adm := bid.AdM
	nurl := bid.NURL
	search := "</Impression>"
	imp := "</Impression><Impression><![CDATA[%s]]></Impression>"
	wrapped_nurl := fmt.Sprintf(imp, nurl)
	results := strings.Replace(adm, search, wrapped_nurl, 1)
	return results
}

func isDmxParams(t interface{}) bool {
	switch t.(type) {
	case dmxParams:
		return true
	default:
		return false
	}
}
