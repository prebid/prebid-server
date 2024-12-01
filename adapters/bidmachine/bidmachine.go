package bidmachine

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"text/template"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"

	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	endpoint *template.Template
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json")
	headers.Add("Accept", "application/json")
	headers.Add("X-Openrtb-Version", "2.5")

	impressions := request.Imp
	result := make([]*adapters.RequestData, 0, len(impressions))
	errs := make([]error, 0, len(impressions))

	for _, impression := range impressions {
		if impression.Banner != nil {
			banner := impression.Banner
			if banner.W == nil && banner.H == nil {
				if banner.Format == nil {
					errs = append(errs, &errortypes.BadInput{
						Message: "Impression with id: " + impression.ID + " has following error: Banner width and height is not provided and banner format is missing. At least one is required",
					})
					continue
				}
				if len(banner.Format) == 0 {
					errs = append(errs, &errortypes.BadInput{
						Message: "Impression with id: " + impression.ID + " has following error: Banner width and height is not provided and banner format array is empty. At least one is required",
					})
					continue
				}
			}

		}

		var bidderExt adapters.ExtImpBidder
		err := jsonutil.Unmarshal(impression.Ext, &bidderExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		var impressionExt openrtb_ext.ExtImpBidmachine
		err = jsonutil.Unmarshal(bidderExt.Bidder, &impressionExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		url, err := a.buildEndpointURL(impressionExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if bidderExt.Prebid != nil && bidderExt.Prebid.IsRewardedInventory != nil && *bidderExt.Prebid.IsRewardedInventory == 1 {
			if impression.Banner != nil && !hasRewardedBattr(impression.Banner.BAttr) {
				bannerCopy := *impression.Banner
				bannerCopy.BAttr = copyBAttrWithRewardedInventory(bannerCopy.BAttr)
				impression.Banner = &bannerCopy
			}
			if impression.Video != nil && !hasRewardedBattr(impression.Video.BAttr) {
				videoCopy := *impression.Video
				videoCopy.BAttr = copyBAttrWithRewardedInventory(videoCopy.BAttr)
				impression.Video = &videoCopy
			}
		}
		request.Imp = []openrtb2.Imp{impression}
		body, err := json.Marshal(request)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		result = append(result, &adapters.RequestData{
			Method:  "POST",
			Uri:     url,
			Body:    body,
			Headers: headers,
			ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
		})
	}

	request.Imp = impressions

	return result, errs
}

func hasRewardedBattr(attr []adcom1.CreativeAttribute) bool {
	for i := 0; i < len(attr); i++ {
		if attr[i] == adcom1.AttrHasSkipButton {
			return true
		}
	}
	return false
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, _ *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var errs []error

	switch responseData.StatusCode {
	case http.StatusNoContent:
		return nil, nil
	case http.StatusServiceUnavailable:
		fallthrough
	case http.StatusBadRequest:
		fallthrough
	case http.StatusUnauthorized:
		fallthrough
	case http.StatusForbidden:
		return nil, []error{&errortypes.BadInput{
			Message: "unexpected status code: " + strconv.Itoa(responseData.StatusCode) + " " + string(responseData.Body),
		}}
	case http.StatusOK:
		break
	default:
		return nil, []error{&errortypes.BadServerResponse{
			Message: "unexpected status code: " + strconv.Itoa(responseData.StatusCode) + " " + string(responseData.Body),
		}}
	}

	var bidResponse openrtb2.BidResponse
	err := jsonutil.Unmarshal(responseData.Body, &bidResponse)
	if err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: err.Error(),
		}}
	}

	response := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))

	for _, seatBid := range bidResponse.SeatBid {
		for _, bid := range seatBid.Bid {
			thisBid := bid
			bidType := GetMediaTypeForImp(bid.ImpID, request.Imp)
			if bidType == UndefinedMediaType {
				errs = append(errs, &errortypes.BadServerResponse{
					Message: "ignoring bid id=" + bid.ID + ", request doesn't contain any valid impression with id=" + bid.ImpID,
				})
				continue
			}
			response.Bids = append(response.Bids, &adapters.TypedBid{
				Bid:     &thisBid,
				BidType: bidType,
			})
		}
	}

	return response, errs
}

// Builder builds a new instance of the Bidmachine adapter for the given bidder with the given config.
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

const UndefinedMediaType = openrtb_ext.BidType("")

func (a *adapter) buildEndpointURL(params openrtb_ext.ExtImpBidmachine) (string, error) {
	endpointParams := macros.EndpointTemplateParams{Host: params.Host}
	uriString, errMacros := macros.ResolveMacros(a.endpoint, endpointParams)
	if errMacros != nil {
		return "", &errortypes.BadInput{
			Message: "Failed to resolve host macros",
		}
	}
	uri, errUrl := url.Parse(uriString)
	if errUrl != nil || uri.Scheme == "" || uri.Host == "" {
		return "", &errortypes.BadInput{
			Message: "Failed to create final URL with provided host",
		}
	}
	uri.Path = path.Join(uri.Path, params.Path)
	uri.Path = path.Join(uri.Path, params.SellerID)
	return uri.String(), nil
}

func copyBAttrWithRewardedInventory(src []adcom1.CreativeAttribute) []adcom1.CreativeAttribute {
	dst := make([]adcom1.CreativeAttribute, len(src))
	copy(dst, src)
	dst = append(dst, adcom1.AttrHasSkipButton)
	return dst
}

func GetMediaTypeForImp(impID string, imps []openrtb2.Imp) openrtb_ext.BidType {
	mediaType := openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Banner == nil && imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			}
			return mediaType
		}
	}
	return UndefinedMediaType
}
