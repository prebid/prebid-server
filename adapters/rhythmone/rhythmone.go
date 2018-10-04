package rhythmone

import (
	"encoding/json"
	"fmt"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
)

type RhythmoneAdapter struct {
	http     *adapters.HTTPAdapter
	endPoint string
}

// used for cookies and such
func (a *RhythmoneAdapter) Name() string {
	return "rhythmone"
}

func (a *RhythmoneAdapter) SkipNoCookies() bool {
	return false
}

func (a *RhythmoneAdapter) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
	errs := make([]error, 0, len(request.Imp))
	if len(request.Imp) == 0 {
		err := &errortypes.BadInput{
			Message: "No imp data provided in the bid request.",
		}
		errs = append(errs, err)
		return nil, errs
	}

	validImpDataExists := false

	for _, imp := range request.Imp {
		if imp.Banner != nil {
			validImpDataExists = true
		} else if imp.Video != nil {
			validImpDataExists = true
		} else {
			err := &errortypes.BadInput{
				Message: fmt.Sprintf("RhythmOne only supports banner and video imps. Ignoring imp id=%s", imp.ID),
			}
			errs = append(errs, err)
		}
	}

	if !validImpDataExists {
		err := &errortypes.BadInput{
			Message: fmt.Sprintf("No valid imp data in the bid request"),
		}
		errs = append(errs, err)
		return nil, errs
	}

	var uri string
	request, uri, errs = a.preProcess(request, errs)
	if request != nil {
		reqJSON, err := json.Marshal(request)
		if err != nil {
			errs = append(errs, err)
			return nil, errs
		}
		if uri != "" {
			headers := http.Header{}
			headers.Add("Content-Type", "application/json;charset=utf-8")
			headers.Add("Accept", "application/json")
			return []*adapters.RequestData{{
				Method:  "POST",
				Uri:     uri,
				Body:    reqJSON,
				Headers: headers,
			}}, errs
		}
	}
	return nil, errs
}

func (a *RhythmoneAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}
	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("bad server response: %d. ", err),
		}}
	}

	var errs []error
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: getMediaTypeForImp(sb.Bid[i].ImpID, internalRequest.Imp),
			})
		}
	}
	return bidResponse, errs
}

func getMediaTypeForImp(impId string, imps []openrtb.Imp) openrtb_ext.BidType {
	mediaType := openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Banner != nil {
				mediaType = openrtb_ext.BidTypeBanner
			} else if imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			}
			if imp.Banner != nil && imp.Video != nil {
				mediaType = openrtb_ext.BidTypeBanner
			}
			return mediaType
		}
	}
	return mediaType
}

func NewRhythmoneAdapter(config *adapters.HTTPAdapterConfig, endpoint string) *RhythmoneAdapter {
	return NewRhythmoneBidder(adapters.NewHTTPAdapter(config).Client, endpoint)
}

func NewRhythmoneBidder(client *http.Client, endpoint string) *RhythmoneAdapter {
	a := &adapters.HTTPAdapter{Client: client}
	return &RhythmoneAdapter{
		http:     a,
		endPoint: endpoint,
	}
}

func (a *RhythmoneAdapter) preProcess(req *openrtb.BidRequest, errors []error) (*openrtb.BidRequest, string, []error) {
	numRequests := len(req.Imp)

	requestImpCopy := req.Imp
	var uri string = ""
	for i := 0; i < numRequests; i++ {
		imp := requestImpCopy[i]

		if imp.Audio != nil {
			continue
		}
		if imp.Native != nil {
			continue
		}
		if imp.Banner != nil || imp.Video != nil {

			var bidderExt adapters.ExtImpBidder
			err := json.Unmarshal(imp.Ext, &bidderExt)

			if err != nil {
				err = &errortypes.BadInput{
					Message: fmt.Sprintf("ext data not provided in imp id=%s. Abort all Request", imp.ID),
				}
				errors = append(errors, err)
				return nil, "", errors
			}
			var rhythmoneExt openrtb_ext.ExtImpRhythmone
			err = json.Unmarshal(bidderExt.Bidder, &rhythmoneExt)
			if err != nil {
				err = &errortypes.BadInput{
					Message: fmt.Sprintf("placementId | zone | path not provided in imp id=%s. Abort all Request", imp.ID),
				}
				errors = append(errors, err)
				return nil, "", errors
			}

			if rhythmoneExt.PlacementId == "" || rhythmoneExt.Zone == "" || rhythmoneExt.Path == "" {
				err = &errortypes.BadInput{
					Message: fmt.Sprintf("placementId | zone | path empty in imp id=%s. Abort all Request", imp.ID),
				}
				errors = append(errors, err)
				return nil, "", errors
			}
			rhythmoneExt.S2S = true
			rhythmoneExtCopy, err := json.Marshal(&rhythmoneExt)
			if err != nil {
				errors = append(errors, err)
				return nil, "", errors
			}
			bidderExtCopy := openrtb_ext.ExtBid{
				Bidder: rhythmoneExtCopy,
			}

			impExtCopy, err := json.Marshal(&bidderExtCopy)
			if err != nil {
				errors = append(errors, err)
				return nil, "", errors
			}
			imp.Ext = impExtCopy
			req.Imp[i] = imp
			if uri == "" {
				uri = fmt.Sprintf("%s/%s/0/%s?z=%s&s2s=%s", a.endPoint, rhythmoneExt.PlacementId, rhythmoneExt.Path, rhythmoneExt.Zone, "true")
			}
		}
	}

	return req, uri, errors
}
