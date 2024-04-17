package trustedstack

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/errortypes"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

type adapter struct {
	endpoint string
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
	}}, errs
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var errs []error

	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
	}

	var bidResp openrtb2.BidResponse

	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponse()

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidType, err := getMediaTypeForImp(&sb.Bid[i], internalRequest.Imp)
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

// Builder builds a new instance of the Trustedstack adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	url := buildEndpoint(config.Endpoint, server.ExternalUrl)
	return &adapter{
		endpoint: url,
	}, nil
}

func getMediaTypeForImp(bid *openrtb2.Bid, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	mediaType, err := getBidMediaTypeFromMtype(bid)
	if err == nil {
		return mediaType, nil
	}
	mediaType = openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == bid.ImpID {
			switch {
			case imp.Banner == nil && imp.Video != nil && imp.Audio == nil && imp.Native == nil:
				mediaType = openrtb_ext.BidTypeVideo
			case imp.Banner == nil && imp.Video == nil && imp.Audio != nil && imp.Native == nil:
				mediaType = openrtb_ext.BidTypeAudio
			case imp.Banner == nil && imp.Video == nil && imp.Audio == nil && imp.Native != nil:
				mediaType = openrtb_ext.BidTypeNative
			}
			return mediaType, nil
		}
	}

	return "", &errortypes.BadInput{
		Message: fmt.Sprintf("Failed to find impression \"%s\"", bid.ImpID),
	}
}

func getBidMediaTypeFromMtype(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupAudio:
		return openrtb_ext.BidTypeAudio, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	default:
		return "", fmt.Errorf("unable to fetch mediaType for imp: %s", bid.ImpID)
	}
}

func buildEndpoint(trustedstackUrl, hostUrl string) string {

	if len(hostUrl) == 0 {
		return trustedstackUrl
	}
	urlObject, err := url.Parse(trustedstackUrl)
	if err != nil {
		return trustedstackUrl
	}
	values := urlObject.Query()
	values.Add("src", hostUrl)
	urlObject.RawQuery = values.Encode()
	return urlObject.String()
}
