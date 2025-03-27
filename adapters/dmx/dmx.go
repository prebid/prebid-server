package dmx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type DmxAdapter struct {
	endpoint string
}

// Builder builds a new instance of the DistrictM DMX adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &DmxAdapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

type dmxExt struct {
	Bidder dmxParams `json:"bidder"`
}

type dmxPubExt struct {
	Dmx dmxPubExtId `json:"dmx,omitempty"`
}

type dmxPubExtId struct {
	Id string `json:"id,omitempty"`
}

type dmxParams struct {
	TagId       string  `json:"tagid,omitempty"`
	DmxId       string  `json:"dmxid,omitempty"`
	MemberId    string  `json:"memberid,omitempty"`
	PublisherId string  `json:"publisher_id,omitempty"`
	SellerId    string  `json:"seller_id,omitempty"`
	Bidfloor    float64 `json:"bidfloor,omitempty"`
}

var protocols = []adcom1.MediaCreativeSubtype{2, 3, 5, 6, 7, 8}

func UserSellerOrPubId(str1, str2 string) string {
	if str1 != "" {
		return str1
	}
	return str2
}

func (adapter *DmxAdapter) MakeRequests(request *openrtb2.BidRequest, req *adapters.ExtraRequestInfo) (reqsBidder []*adapters.RequestData, errs []error) {
	var imps []openrtb2.Imp
	var rootExtInfo dmxExt
	var publisherId string
	var sellerId string
	var userExt openrtb_ext.ExtUser
	var reqCopy openrtb2.BidRequest = *request
	var dmxReq *openrtb2.BidRequest = &reqCopy
	var dmxRawPubId dmxPubExt

	if request.User == nil {
		if request.App == nil {
			return nil, []error{errors.New("No user id or app id found. Could not send request to DMX.")}
		}
	}

	if len(request.Imp) >= 1 {
		err := jsonutil.Unmarshal(request.Imp[0].Ext, &rootExtInfo)
		if err != nil {
			errs = append(errs, err)
		} else {
			publisherId = UserSellerOrPubId(rootExtInfo.Bidder.PublisherId, rootExtInfo.Bidder.MemberId)
			sellerId = rootExtInfo.Bidder.SellerId
		}
	}

	hasNoID := true
	if request.App != nil {
		appCopy := *request.App
		appPublisherCopy := *request.App.Publisher
		dmxReq.App = &appCopy
		dmxReq.App.Publisher = &appPublisherCopy
		if dmxReq.App.Publisher.ID == "" {
			dmxReq.App.Publisher.ID = publisherId
		}
		dmxRawPubId.Dmx.Id = UserSellerOrPubId(rootExtInfo.Bidder.PublisherId, rootExtInfo.Bidder.MemberId)
		ext, err := json.Marshal(dmxRawPubId)
		if err != nil {
			errs = append(errs, fmt.Errorf("unable to marshal ext, %v", err))
			return nil, errs
		}
		dmxReq.App.Publisher.Ext = ext
		if dmxReq.App.ID != "" {
			hasNoID = false
		}
		if hasNoID {
			if idfa, valid := getIdfa(request); valid {
				dmxReq.App.ID = idfa
				hasNoID = false
			}
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
			if dmxReq.Site.Publisher.ID == "" {
				dmxReq.Site.Publisher.ID = publisherId
			}
			dmxRawPubId.Dmx.Id = UserSellerOrPubId(rootExtInfo.Bidder.PublisherId, rootExtInfo.Bidder.MemberId)
			ext, err := json.Marshal(dmxRawPubId)
			if err != nil {
				errs = append(errs, fmt.Errorf("unable to marshal ext, %v", err))
				return nil, errs
			}
			dmxReq.Site.Publisher.Ext = ext
		} else {
			dmxReq.Site.Publisher = &openrtb2.Publisher{ID: publisherId}
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
			hasNoID = false
		}
		if dmxReq.User.Ext != nil {
			if err := jsonutil.Unmarshal(dmxReq.User.Ext, &userExt); err == nil {
				if len(userExt.Eids) > 0 {
					hasNoID = false
				}
			}
		}
	}

	for _, inst := range dmxReq.Imp {
		var ins openrtb2.Imp
		var params dmxExt
		const intVal int8 = 1
		source := (*json.RawMessage)(&inst.Ext)
		if err := jsonutil.Unmarshal(*source, &params); err != nil {
			errs = append(errs, err)
		}
		if isDmxParams(params.Bidder) {
			if inst.Banner != nil {
				if len(inst.Banner.Format) != 0 {
					bannerCopy := *inst.Banner
					if params.Bidder.PublisherId != "" || params.Bidder.MemberId != "" {
						imps = fetchParams(params, inst, ins, imps, &bannerCopy, nil, intVal)
					} else {
						return nil, []error{errors.New("Missing Params for auction to be send")}
					}
				}
			}

			if inst.Video != nil {
				videoCopy := *inst.Video
				if params.Bidder.PublisherId != "" || params.Bidder.MemberId != "" {
					imps = fetchParams(params, inst, ins, imps, nil, &videoCopy, intVal)
				} else {
					return nil, []error{errors.New("Missing Params for auction to be send")}
				}
			}
		}

	}

	dmxReq.Imp = imps

	if hasNoID {
		return nil, []error{errors.New("This request contained no identifier")}
	}

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
		ImpIDs:  openrtb_ext.GetImpIDs(dmxReq.Imp),
	}

	reqsBidder = append(reqsBidder, reqBidder)
	return
}

func (adapter *DmxAdapter) MakeBids(request *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var errs []error

	if http.StatusNoContent == response.StatusCode {
		return nil, nil
	}

	if http.StatusBadRequest == response.StatusCode {
		return nil, []error{&errortypes.BadInput{
			Message: "Unexpected status code 400",
		}}
	}

	if http.StatusOK != response.StatusCode {
		return nil, []error{&errortypes.BadInput{
			Message: "Unexpected response no status code",
		}}
	}

	var bidResp openrtb2.BidResponse

	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
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

func fetchParams(params dmxExt, inst openrtb2.Imp, ins openrtb2.Imp, imps []openrtb2.Imp, banner *openrtb2.Banner, video *openrtb2.Video, intVal int8) []openrtb2.Imp {
	tempImp := inst
	if params.Bidder.Bidfloor != 0 {
		tempImp.BidFloor = params.Bidder.Bidfloor
	}
	if params.Bidder.TagId != "" {
		tempImp.TagID = params.Bidder.TagId
		tempImp.Secure = &intVal
	}

	if params.Bidder.DmxId != "" {
		tempImp.TagID = params.Bidder.DmxId
		tempImp.Secure = &intVal
	}
	if banner != nil {
		if banner.H == nil || banner.W == nil {
			banner.H = &banner.Format[0].H
			banner.W = &banner.Format[0].W
		}
		tempImp.Banner = banner
	}

	if video != nil {
		video.Protocols = checkProtocols(video)
		tempImp.Video = video
	}

	if tempImp.TagID == "" {
		return imps
	}
	imps = append(imps, tempImp)
	return imps
}

func addParams(str string) string {
	if str != "" {
		return "?sellerid=" + url.QueryEscape(str)
	}
	return ""
}

func getMediaTypeForImp(impID string, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
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

func videoImpInsertion(bid *openrtb2.Bid) string {
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

func getIdfa(request *openrtb2.BidRequest) (string, bool) {
	if request.Device == nil {
		return "", false
	}

	device := request.Device

	if device.IFA != "" {
		return device.IFA, true
	}
	return "", false
}
func checkProtocols(imp *openrtb2.Video) []adcom1.MediaCreativeSubtype {
	if len(imp.Protocols) > 0 {
		return imp.Protocols
	}
	return protocols
}
