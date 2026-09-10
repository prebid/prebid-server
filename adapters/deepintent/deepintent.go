package deepintent

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

const displayManager string = "di_prebid"
const displayManagerVer string = "2.0.0"

// DeepintentAdapter struct
type DeepintentAdapter struct {
	URI string
}

// Builder builds a new instance of the Deepintent adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &DeepintentAdapter{
		URI: config.Endpoint,
	}
	return bidder, nil
}

// MakeRequests which creates request object for Deepintent DSP
func (d *DeepintentAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	var deepintentExt openrtb_ext.ExtImpDeepintent
	var err error

	var adapterRequests []*adapters.RequestData

	reqCopy := *request
	for _, imp := range request.Imp {
		reqCopy.Imp = []openrtb2.Imp{imp}

		var bidderExt adapters.ExtImpBidder
		if err = jsonutil.Unmarshal(reqCopy.Imp[0].Ext, &bidderExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: fmt.Sprintf("Impression id=%s has an Error: %s", imp.ID, err.Error()),
			})
			continue
		}

		if err = jsonutil.Unmarshal(bidderExt.Bidder, &deepintentExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: fmt.Sprintf("Impression id=%s, has invalid Ext", imp.ID),
			})
			continue
		}

		reqCopy.Imp[0].TagID = deepintentExt.TagID
		reqCopy.Imp[0].DisplayManager = displayManager
		reqCopy.Imp[0].DisplayManagerVer = displayManagerVer

		adapterReq, errors := d.preprocess(reqCopy)
		if errors != nil {
			errs = append(errs, errors...)
		}
		if adapterReq != nil {
			adapterRequests = append(adapterRequests, adapterReq)
		}

	}
	return adapterRequests, errs
}

// MakeBids makes the bids
func (d *DeepintentAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var errs []error

	if response.StatusCode == http.StatusNoContent {
		return nil, nil
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

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidType, err := getMediaTypeForBid(&sb.Bid[i])
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

func (d *DeepintentAdapter) preprocess(request openrtb2.BidRequest) (*adapters.RequestData, []error) {

	var errs []error
	impsCount := len(request.Imp)
	resImps := make([]openrtb2.Imp, 0, impsCount)

	for _, imp := range request.Imp {

		if err := validateImp(&imp); err != nil {
			errs = append(errs, err)
			continue
		}
		resImps = append(resImps, imp)
	}
	request.Imp = resImps
	if errs != nil {
		return nil, errs
	}
	reqJSON, err := json.Marshal(request)

	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	return &adapters.RequestData{
		Method:  "POST",
		Uri:     d.URI,
		Body:    reqJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}, errs
}

func validateImp(imp *openrtb2.Imp) error {
	if imp.Banner == nil && imp.Video == nil {
		return &errortypes.BadInput{
			Message: fmt.Sprintf("DeepIntent only supports Banner and Video media types. Ignoring ImpID=%s", imp.ID),
		}
	}

	if imp.Banner != nil {
		return buildImpBanner(imp)
	}

	return nil
}

func buildImpBanner(imp *openrtb2.Imp) error {
	if imp.Banner.W == nil && imp.Banner.H == nil {
		bannerCopy := *imp.Banner
		banner := &bannerCopy

		if len(banner.Format) == 0 {
			return &errortypes.BadInput{
				Message: "At least one size is required",
			}
		}
		format := banner.Format[0]
		banner.W = &format.W
		banner.H = &format.H
		imp.Banner = banner
	}

	return nil
}

func getMediaTypeForBid(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unsupported MType %d for bid %s", bid.MType, bid.ImpID),
		}
	}
}
