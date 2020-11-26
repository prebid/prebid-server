package adot

import (
	"encoding/json"
	"fmt"
	"github.com/buger/jsonparser"
	"net/http"
	"strconv"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type AdotAdapter struct {
	endpoint string
}

// MakeRequests makes the HTTP requests which should be made to fetch bids.
func (a *AdotAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errors = make([]error, 0)

	reqJSON, err := json.Marshal(request)
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}
	reqJSON = addParallaxIfNecessary(reqJSON)

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    reqJSON,
		Headers: headers,
	}}, errors
}

// MakeBids unpacks the server's response into Bids.
func (a *AdotAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			if bidType, err := getMediaTypeForBid(&sb.Bid[i], internalRequest); err == nil {
				bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
					Bid:     &sb.Bid[i],
					BidType: bidType,
				})
			}
		}
	}
	return bidResponse, nil

}

// getMediaTypeForBid determines which type of bid.
func getMediaTypeForBid(bid *openrtb.Bid, internalRequest *openrtb.BidRequest) (openrtb_ext.BidType, error) {

	impID := bid.ImpID

	for _, imp := range internalRequest.Imp {
		if imp.ID == impID {
			if imp.Banner != nil {
				return openrtb_ext.BidTypeBanner, nil
			} else if imp.Video != nil {
				return openrtb_ext.BidTypeVideo, nil
			} else if imp.Audio != nil {
				return openrtb_ext.BidTypeAudio, nil
			} else if imp.Native != nil {
				return openrtb_ext.BidTypeNative, nil
			}
		}
	}

	return "", fmt.Errorf("Unrecognized bid type in response from adot")
}

func addParallaxIfNecessary(reqJSON []byte) []byte {
	var adotJSON []byte
	var err error

	adotRequest, parallaxError := addParallaxInRequest(reqJSON)
	if parallaxError == nil {
		adotJSON, err = json.Marshal(adotRequest)
		if err != nil {
			adotJSON = reqJSON
		}
	} else {
		adotJSON = reqJSON
	}

	return adotJSON
}

func addParallaxInRequest(data []byte) (map[string]interface{}, error) {
	var adotRequest map[string]interface{}

	if err := json.Unmarshal(data, &adotRequest); err != nil {
		return adotRequest, err
	}

	imps := adotRequest["imp"].([]interface{})
	for i, impObj := range imps {
		castedImps := impObj.(map[string]interface{})
		if isParallaxInExt(castedImps) {
			if impByte, err := json.Marshal(impObj); err == nil {
				if val, err := jsonparser.Set(impByte, jsonparser.StringToBytes(strconv.FormatBool(true)), "banner", "parallax"); err == nil {
					_ = json.Unmarshal(val, &imps[i])
				}
			}
		}
	}
	return adotRequest, nil
}

func isParallaxInExt(impObj map[string]interface{}) bool {
	isParallaxInExt := false

	isParallaxByte, err := getParallaxByte(impObj)
	if err == nil {
		isParallaxInt, err := jsonparser.GetInt(isParallaxByte)
		if err != nil {
			isParallaxBool, err := jsonparser.GetBoolean(isParallaxByte)
			if err == nil {
				return isParallaxBool
			}
		}
		isParallaxInExt = isParallaxInt == 1
	}

	return isParallaxInExt
}

func getParallaxByte(impObj map[string]interface{}) ([]byte, error) {
	impByte, err := json.Marshal(impObj)
	if err != nil {
		return nil, err
	}

	isExtByte, _, _, err := jsonparser.Get(impByte, "ext")
	if err != nil {
		return nil, err
	}

	isParallaxByte, _, _, err := jsonparser.Get(isExtByte, "bidder", "parallax")
	return isParallaxByte, err
}

// NewGridBidder configure bidder endpoint
func NewAdotAdapter(endpoint string) *AdotAdapter {
	return &AdotAdapter{
		endpoint: endpoint,
	}
}
