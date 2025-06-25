package nexx360

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/version"
)

type adapter struct {
	endpoint string
}

type Ext struct {
	Nexx360 json.RawMessage `json:"nexx360"`
}

type Nexx360Caller struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

type ReqExt struct {
	Nexx360 *ReqNexx360Ext `json:"nexx360,omitempty"`
}

type ReqNexx360Ext struct {
	Caller []Nexx360Caller `json:"caller,omitempty"`
}

type Nexx360ResBidExt struct {
	BidType string `json:"bidType,omitempty"`
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	return &adapter{endpoint: config.Endpoint}, nil
}

// CALLER Info used to track Prebid Server
// as one of the hops in the request to exchange

func getVersion() string {
	if version.Ver != "" {
		return version.Ver
	}
	return "n/a"
}

var CALLER = Nexx360Caller{"Prebid-Server", getVersion()}

func processImps(impList []openrtb2.Imp) (imp []openrtb2.Imp, tagId string, placement string, error error) {
	var imps []openrtb2.Imp
	for idx, imp := range impList {
		var bidderExt adapters.ExtImpBidder
		if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
			return nil, "", "", &errortypes.BadInput{
				Message: err.Error(),
			}
		}

		impExt := Ext{
			Nexx360: bidderExt.Bidder,
		}

		impExtJSON, err := json.Marshal(impExt)
		if err != nil {
			return nil, "", "", &errortypes.BadInput{
				Message: err.Error(),
			}
		}

		impCopy := imp
		impCopy.Ext = impExtJSON
		imps = append(imps, impCopy)
		if idx == 0 {
			var nexx360Ext openrtb_ext.ExtImpNexx360
			if err := jsonutil.Unmarshal(bidderExt.Bidder, &nexx360Ext); err != nil {
				return nil, "", "", &errortypes.BadInput{
					Message: err.Error(),
				}
			}
			tagId = nexx360Ext.TagId
			placement = nexx360Ext.Placement
		}
	}

	return imps, tagId, placement, nil
}

func makeReqExt() ([]byte, error) {
	reqExt := ReqExt{
		Nexx360: &ReqNexx360Ext{
			Caller: []Nexx360Caller{CALLER},
		},
	}

	return json.Marshal(reqExt)
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var imp, tagId, placement, err = processImps(request.Imp)
	if err != nil {
		return nil, []error{err}
	}

	request.Imp = imp

	urlBuilder, err := url.Parse(a.endpoint)
	if err != nil {
		return nil, []error{err}
	}

	query := url.Values{}

	if placement != "" {
		query.Add("placement", placement)
	}

	if tagId != "" {
		query.Add("tag_id", tagId)
	}
	urlBuilder.RawQuery = query.Encode()

	uri := urlBuilder.String()

	reqExt, err := makeReqExt()
	if err != nil {
		return nil, []error{err}
	}
	request.Ext = reqExt

	// Last Step
	reqJSON, err := json.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json")

	adapter := &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     uri,
		Body:    reqJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}

	return []*adapters.RequestData{adapter}, nil
}

// MakeBids make the bids for the bid response.
func (a *adapter) MakeBids(request *openrtb2.BidRequest, externalRequest *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected http status code: %d", responseData.StatusCode),
		}}
	}

	var response openrtb2.BidResponse

	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	var Bids []*adapters.TypedBid
	var errors []error
	for _, seatBid := range response.SeatBid {
		for i, bid := range seatBid.Bid {

			bidType, err := getBidType(bid)
			if err != nil {
				errors = append(errors, err)
				continue
			}
			Bids = append(Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			})
		}
	}
	if len(Bids) == 0 {
		return nil, nil
	}
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(Bids))
	bidResponse.Bids = Bids
	if response.Cur != "" {
		bidResponse.Currency = response.Cur
	}

	return bidResponse, errors
}

func getBidType(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	var bidExt Nexx360ResBidExt
	err := jsonutil.Unmarshal(bid.Ext, &bidExt)
	if err == nil {
		if bidExt.BidType == "video" {
			return openrtb_ext.BidTypeVideo, nil
		}
		if bidExt.BidType == "audio" {
			return openrtb_ext.BidTypeAudio, nil
		}
		if bidExt.BidType == "native" {
			return openrtb_ext.BidTypeNative, nil
		}
		if bidExt.BidType == "banner" {
			return openrtb_ext.BidTypeBanner, nil
		}
	}
	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("unable to fetch mediaType in multi-format: %s", bid.ImpID),
	}
}
