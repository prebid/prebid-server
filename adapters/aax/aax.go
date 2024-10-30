package aax

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
)

type adapter struct {
	endpoint string
}

type aaxResponseBidExt struct {
	AdCodeType string `json:"adCodeType"`
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error

	reqJson, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    reqJson,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}}, errs
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var errs []error

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

	var bidResp openrtb2.BidResponse

	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponse()

	for _, seatBid := range bidResp.SeatBid {
		for i := range seatBid.Bid {
			bidType, err := getMediaTypeForImp(seatBid.Bid[i], internalRequest.Imp)
			if err != nil {
				errs = append(errs, err)
			} else {
				b := &adapters.TypedBid{
					Bid:     &seatBid.Bid[i],
					BidType: bidType,
				}
				bidResponse.Bids = append(bidResponse.Bids, b)
			}
		}
	}
	return bidResponse, errs
}

// Builder builds a new instance of the Aax adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	url := buildEndpoint(config.Endpoint, config.ExtraAdapterInfo)
	return &adapter{
		endpoint: url,
	}, nil
}

func getMediaTypeForImp(bid openrtb2.Bid, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	var bidExt aaxResponseBidExt
	err := jsonutil.Unmarshal(bid.Ext, &bidExt)
	if err == nil {
		switch bidExt.AdCodeType {
		case "banner":
			return openrtb_ext.BidTypeBanner, nil
		case "native":
			return openrtb_ext.BidTypeNative, nil
		case "video":
			return openrtb_ext.BidTypeVideo, nil
		}
	}

	var mediaType openrtb_ext.BidType
	var typeCnt = 0
	for _, imp := range imps {
		if imp.ID == bid.ImpID {
			if imp.Banner != nil {
				typeCnt += 1
				mediaType = openrtb_ext.BidTypeBanner
			}
			if imp.Native != nil {
				typeCnt += 1
				mediaType = openrtb_ext.BidTypeNative
			}
			if imp.Video != nil {
				typeCnt += 1
				mediaType = openrtb_ext.BidTypeVideo
			}
		}
	}
	if typeCnt == 1 {
		return mediaType, nil
	}
	return mediaType, fmt.Errorf("unable to fetch mediaType in multi-format: %s", bid.ImpID)
}

func buildEndpoint(aaxUrl, hostUrl string) string {

	if len(hostUrl) == 0 {
		return aaxUrl
	}
	urlObject, err := url.Parse(aaxUrl)
	if err != nil {
		return aaxUrl
	}
	values := urlObject.Query()
	values.Add("src", hostUrl)
	urlObject.RawQuery = values.Encode()
	return urlObject.String()
}
