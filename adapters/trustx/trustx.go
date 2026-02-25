package trustx

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type impExt struct {
	Prebid *openrtb_ext.ExtImpPrebid `json:"prebid,omitempty"`
	Bidder json.RawMessage           `json:"bidder"`
	Data   *impExtData               `json:"data,omitempty"`
	Gpid   string                    `json:"gpid,omitempty"`
}

type impExtData struct {
	PbAdslot string              `json:"pbadslot,omitempty"`
	AdServer *impExtDataAdServer `json:"adserver,omitempty"`
}

type impExtDataAdServer struct {
	Name   string `json:"name,omitempty"`
	AdSlot string `json:"adslot,omitempty"`
}

type bidExt struct {
	Bidder bidExtBidder `json:"bidder,omitempty"`
}

type bidExtBidder struct {
	TrustX bidExtBidderTrustX `json:"trustx,omitempty"`
}

type bidExtBidderTrustX struct {
	NetworkName string `json:"networkName,omitempty"`
}

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the TRUSTX adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	for i := range request.Imp {
		err := setImpExtData(&request.Imp[i])
		if err != nil {
			errs = append(errs, err)
			return nil, errs
		}
	}

	body, err := jsonutil.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	reqs := make([]*adapters.RequestData, 0, 1)
	requestData := &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     a.endpoint,
		Body:    body,
		Headers: getHeaders(request),
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}
	reqs = append(reqs, requestData)

	return reqs, errs
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(response) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
	}

	var resp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &resp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Bad Server Response",
		}}
	}

	var bidErrors []error
	bidderResponse := adapters.NewBidderResponseWithBidsCapacity(1)
	for i := range resp.SeatBid {
		seatBid := &resp.SeatBid[i]
		for j := range seatBid.Bid {
			bid := &seatBid.Bid[j]
			typedBid, err := getTypedBid(bid)
			if err != nil {
				bidErrors = append(bidErrors, err)
				continue
			}
			bidderResponse.Bids = append(bidderResponse.Bids, typedBid)
		}
	}

	return bidderResponse, bidErrors
}

func setImpExtData(imp *openrtb2.Imp) error {
	var ext impExt
	if err := jsonutil.Unmarshal(imp.Ext, &ext); err != nil {
		//if unmarshalling fails, proceed with the request
		return nil
	}
	if ext.Data != nil && ext.Data.AdServer != nil && ext.Data.AdServer.AdSlot != "" {
		ext.Gpid = ext.Data.AdServer.AdSlot
		extJSON, err := jsonutil.Marshal(ext)
		if err != nil {
			return err
		}
		imp.Ext = extJSON
	}
	return nil
}

func getHeaders(request *openrtb2.BidRequest) http.Header {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json;charset=utf-8")
	headers.Set("Accept", "application/json")
	headers.Set("X-Openrtb-Version", "2.6")

	if request.Site != nil {
		if request.Site.Ref != "" {
			headers.Set("Referer", request.Site.Ref)
		}
		if request.Site.Domain != "" {
			headers.Set("Origin", request.Site.Domain)
		}
	}

	if request.Device != nil {
		if len(request.Device.IP) > 0 {
			headers.Set("X-Forwarded-For", request.Device.IP)
		}

		if len(request.Device.IPv6) > 0 {
			headers.Set("X-Forwarded-For", request.Device.IPv6)
		}

		if len(request.Device.UA) > 0 {
			headers.Set("User-Agent", request.Device.UA)
		}
	}

	return headers
}

func getTypedBid(bid *openrtb2.Bid) (*adapters.TypedBid, error) {
	var bidType openrtb_ext.BidType
	switch bid.MType {
	case openrtb2.MarkupBanner:
		bidType = openrtb_ext.BidTypeBanner
	case openrtb2.MarkupVideo:
		bidType = openrtb_ext.BidTypeVideo
	default:
		return nil, &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unsupported MType: %v", bid.MType),
		}
	}

	var extBidPrebidVideo *openrtb_ext.ExtBidPrebidVideo
	if bidType == openrtb_ext.BidTypeVideo {
		extBidPrebidVideo = &openrtb_ext.ExtBidPrebidVideo{}
		if len(bid.Cat) > 0 {
			extBidPrebidVideo.PrimaryCategory = bid.Cat[0]
		}
		if bid.Dur > 0 {
			extBidPrebidVideo.Duration = int(bid.Dur)
		}
	}
	return &adapters.TypedBid{
		Bid:      bid,
		BidType:  bidType,
		BidVideo: extBidPrebidVideo,
		BidMeta:  getBidMeta(bid.Ext),
	}, nil
}

func getBidMeta(ext json.RawMessage) *openrtb_ext.ExtBidPrebidMeta {
	if ext == nil {
		return nil
	}
	var be bidExt

	if err := jsonutil.Unmarshal(ext, &be); err != nil {
		return nil
	}
	var bidMeta *openrtb_ext.ExtBidPrebidMeta
	if be.Bidder.TrustX.NetworkName != "" {
		bidMeta = &openrtb_ext.ExtBidPrebidMeta{
			NetworkName: be.Bidder.TrustX.NetworkName,
		}
	}
	return bidMeta
}
