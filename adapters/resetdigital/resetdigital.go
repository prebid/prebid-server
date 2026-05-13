package resetdigital

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"golang.org/x/text/currency"
)

type adapter struct {
	endpoint    *template.Template
	endpointUri string
}

type resetDigitalRequest struct {
	Site resetDigitalSite  `json:"site"`
	Imps []resetDigitalImp `json:"imps"`
	User *resetDigitalUser `json:"user,omitempty"`
}
type resetDigitalSite struct {
	Domain   string `json:"domain"`
	Referrer string `json:"referrer"`
}
type resetDigitalUser struct {
	Ext resetDigitalUserExt `json:"ext"`
}
type resetDigitalUserExt struct {
	EIDs []openrtb2.EID `json:"eids,omitempty"`
}
type resetDigitalImp struct {
	ZoneID     resetDigitalImpZone    `json:"zone_id"`
	BidID      string                 `json:"bid_id"`
	ImpID      string                 `json:"imp_id"`
	Ext        resetDigitalImpExt     `json:"ext"`
	MediaTypes resetDigitalMediaTypes `json:"media_types"`
}
type resetDigitalImpZone struct {
	PlacementID string `json:"placementId"`
}
type resetDigitalImpExt struct {
	Gpid string `json:"gpid"`
}
type resetDigitalMediaTypes struct {
	Banner resetDigitalMediaType `json:"banner,omitempty"`
	Video  resetDigitalMediaType `json:"video,omitempty"`
	Audio  resetDigitalMediaType `json:"audio,omitempty"`
}
type resetDigitalMediaType struct {
	Sizes [][]int64 `json:"sizes,omitempty"`
	Mimes []string  `json:"mimes,omitempty"`
}
type resetDigitalBidResponse struct {
	Bids []resetDigitalBid `json:"bids"`
}
type resetDigitalBid struct {
	BidID string  `json:"bid_id"`
	ImpID string  `json:"imp_id"`
	CPM   float64 `json:"cpm"`
	CID   string  `json:"cid,omitempty"`
	CrID  string  `json:"crid,omitempty"`
	AdID  string  `json:"adid"`
	W     string  `json:"w,omitempty"`
	H     string  `json:"h,omitempty"`
	Seat  string  `json:"seat"`
	HTML  string  `json:"html"`
}

const liveRampEIDSource = "liveramp.com"

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	template, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}
	bidder := &adapter{
		endpoint: template,
	}
	return bidder, nil
}

type adapter struct {
	endpoint string
}

func Builder(_ openrtb_ext.BidderName, cfg config.Adapter, _ config.Server) (adapters.Bidder, error) {
	return &adapter{
		endpoint: cfg.Endpoint,
	}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {

	if len(request.Imp) != 1 {
		return nil, []error{&errortypes.BadInput{
			Message: "ResetDigital adapter supports only one impression per request",
		}}
	}

	imp := request.Imp[0]
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Error parsing bidderExt from imp.ext: %v", err),
		}}
	}

	if eids := getLiveRampEIDs(requestData.User); len(eids) > 0 {
		reqData.User = &resetDigitalUser{
			Ext: resetDigitalUserExt{
				EIDs: eids,
			},
		}
	}

	rdImp := resetDigitalImp{
		BidID: requestData.ID,
		ImpID: imp.ID,
	}

	uri := fmt.Sprintf("%s?pid=%s", a.endpoint, resetDigitalExt.PlacementID)

	reqHeaders := baseHeaders.Clone()

	reqs := []*adapters.RequestData{
		{
			Method:  http.MethodPost,
			Uri:     uri,
			Body:    reqBody,
			Headers: reqHeaders,
			ImpIDs:  []string{imp.ID},
		},
	}

	return reqs, nil
}

func getLiveRampEIDs(user *openrtb2.User) []openrtb2.EID {
	if user == nil {
		return nil
	}

	eids := make([]openrtb2.EID, 0, len(user.EIDs))
	eids = appendLiveRampEIDs(eids, user.EIDs)

	if len(user.Ext) > 0 {
		var userExt openrtb_ext.ExtUser
		if err := json.Unmarshal(user.Ext, &userExt); err == nil {
			eids = appendLiveRampEIDs(eids, userExt.Eids)
		}
	}

	return eids
}

func appendLiveRampEIDs(dst []openrtb2.EID, src []openrtb2.EID) []openrtb2.EID {
	for _, eid := range src {
		if eid.Source == liveRampEIDSource {
			dst = append(dst, eid)
		}
	}

	return dst
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}

	if responseData.StatusCode >= http.StatusBadRequest && responseData.StatusCode < http.StatusInternalServerError {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", responseData.StatusCode),
		}}
	}

	if responseData.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", responseData.StatusCode),
		}}
	}

	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Bad server response: %s", err),
		}}
	}

	if len(bidResp.SeatBid) == 0 {
		return nil, nil
	}

	return parseBidResponse(request, &bidResp)
}

func parseBidResponse(request *openrtb2.BidRequest, bidResp *openrtb2.BidResponse) (*adapters.BidderResponse, []error) {
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(bidResp.SeatBid))
	var errs []error

	if bidResp.Cur != "" {
		bidResponse.Currency = bidResp.Cur
	} else {
		cur := validateAndFilterCurrencies(request.Cur)
		bidResponse.Currency = cur[0]
	}

	for _, seatBid := range bidResp.SeatBid {
		for i := range seatBid.Bid {
			if seatBid.Bid[i].Price <= 0 {
				errs = append(errs, &errortypes.Warning{
					Message: fmt.Sprintf("price %f <= 0 filtered out", seatBid.Bid[i].Price),
				})
				continue
			}

			bidType, err := getBidType(seatBid.Bid[i], request)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			seat := openrtb_ext.BidderName(bidderSeat)
			if seatBid.Seat != "" {
				seat = openrtb_ext.BidderName(seatBid.Seat)
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
				Seat:    seat,
			})
		}
	}

	return bidResponse, errs
}

func getBidType(bid openrtb2.Bid, request *openrtb2.BidRequest) (openrtb_ext.BidType, error) {
	if bid.MType > 0 {
		switch bid.MType {
		case openrtb2.MarkupBanner:
			return openrtb_ext.BidTypeBanner, nil
		case openrtb2.MarkupVideo:
			return openrtb_ext.BidTypeVideo, nil
		case openrtb2.MarkupAudio:
			return openrtb_ext.BidTypeAudio, nil
		case openrtb2.MarkupNative:
			return openrtb_ext.BidTypeNative, nil
		}
	}
	if request.Imp[0].ID != bid.ImpID {
		return "", fmt.Errorf("no matching impression found for ImpID: %s", bid.ImpID)
	}
	return getMediaType(request.Imp[0]), nil
}

func validateAndFilterCurrencies(currencies []string) []string {
	valid := make([]string, 0, len(currencies))
	for _, s := range currencies {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		s = strings.ToUpper(s)
		if u, err := currency.ParseISO(s); err == nil {
			valid = append(valid, u.String())
		}
	}
	if len(valid) == 0 {
		return []string{currencyUSD}
	}
	return valid
}

func getMediaType(imp openrtb2.Imp) openrtb_ext.BidType {
	switch {
	case imp.Video != nil:
		return openrtb_ext.BidTypeVideo
	case imp.Audio != nil:
		return openrtb_ext.BidTypeAudio
	case imp.Native != nil:
		return openrtb_ext.BidTypeNative
	default:
		return openrtb_ext.BidTypeBanner
	}
}
