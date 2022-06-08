package sovrn

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
)

type SovrnAdapter struct {
	URI string
}

func (s *SovrnAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json")
	if request.Device != nil {
		addHeaderIfNonEmpty(headers, "User-Agent", request.Device.UA)
		addHeaderIfNonEmpty(headers, "X-Forwarded-For", request.Device.IP)
		addHeaderIfNonEmpty(headers, "Accept-Language", request.Device.Language)
		if request.Device.DNT != nil {
			addHeaderIfNonEmpty(headers, "DNT", strconv.Itoa(int(*request.Device.DNT)))
		}
	}

	if request.User != nil {
		userID := strings.TrimSpace(request.User.BuyerUID)
		if len(userID) > 0 {
			headers.Add("Cookie", fmt.Sprintf("%s=%s", "ljt_reader", userID))
		}
	}

	errs := make([]error, 0, len(request.Imp))
	var err error
	validImps := make([]openrtb2.Imp, 0, len(request.Imp))

	for _, imp := range request.Imp {
		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		var sovrnExt openrtb_ext.ExtImpSovrn
		if err := json.Unmarshal(bidderExt.Bidder, &sovrnExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		tagId := getTagId(sovrnExt)
		if tagId == "" {
			errs = append(errs, &errortypes.BadInput{
				Message: "Missing required parameter 'tagid'",
			})
			continue
		}

		imp.TagID = tagId

		if imp.BidFloor == 0 && sovrnExt.BidFloor > 0 {
			imp.BidFloor = sovrnExt.BidFloor
		}

		// Validate video params if appropriate
		video := imp.Video
		if video != nil {
			if video.MIMEs == nil ||
				video.MinDuration == 0 ||
				video.MaxDuration == 0 ||
				video.Protocols == nil {
				errs = append(errs, &errortypes.BadInput{
					Message: "Missing required video parameter",
				})
				continue
			}
		}

		validImps = append(validImps, imp)
	}

	if len(validImps) == 0 {
		return nil, errs
	}

	request.Imp = validImps

	reqJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     s.URI,
		Body:    reqJSON,
		Headers: headers,
	}}, errs
}

func addHeaderIfNonEmpty(headers http.Header, headerName string, headerValue string) {
	if len(headerValue) > 0 {
		headers.Add(headerName, headerValue)
	}
}

func (s *SovrnAdapter) MakeBids(request *openrtb2.BidRequest, bidderRequest *adapters.RequestData, bidderResponse *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if bidderResponse.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if bidderResponse.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", bidderResponse.StatusCode),
		}}
	}

	if bidderResponse.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", bidderResponse.StatusCode),
		}}
	}

	var bidResponse openrtb2.BidResponse
	if err := json.Unmarshal(bidderResponse.Body, &bidResponse); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: err.Error(),
		}}
	}

	response := adapters.NewBidderResponseWithBidsCapacity(5)
	errs := make([]error, 0)

	for _, sb := range bidResponse.SeatBid {
		for _, bid := range sb.Bid {
			adm, err := url.QueryUnescape(bid.AdM)
			if err == nil {
				bid.AdM = adm

				bidType := openrtb_ext.BidTypeBanner

				impIdx, impIdErr := getImpIdx(bid.ImpID, request)
				if impIdErr != nil {
					errs = append(errs, impIdErr)
					continue
				} else if request.Imp[impIdx].Video != nil {
					bidType = openrtb_ext.BidTypeVideo
				}

				response.Bids = append(response.Bids, &adapters.TypedBid{
					Bid:     &bid,
					BidType: bidType,
				})
			}
		}
	}

	return response, errs
}

func getTagId(sovrnExt openrtb_ext.ExtImpSovrn) string {
	if len(sovrnExt.Tagid) > 0 {
		return sovrnExt.Tagid
	} else {
		return sovrnExt.TagId
	}
}

func getImpIdx(impId string, request *openrtb2.BidRequest) (int, error) {
	for idx, imp := range request.Imp {
		if imp.ID == impId {
			return idx, nil
		}
	}

	return -1, &errortypes.BadInput{
		Message: fmt.Sprintf("Imp ID %s in bid didn't match with any imp in the original request", impId),
	}
}

// Builder builds a new instance of the Sovrn adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &SovrnAdapter{
		URI: config.Endpoint,
	}
	return bidder, nil
}
