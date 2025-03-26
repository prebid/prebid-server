package connatix

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

const (
	maxImpsPerReq = 1
)

// Builder builds a new instance of the Connatix adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if request.Device == nil || (request.Device.IP == "" && request.Device.IPv6 == "") {
		return nil, []error{&errortypes.BadInput{
			Message: "Device IP is required",
		}}
	}

	// connatix adapter expects imp.displaymanagerver to be populated in openrtb2 request
	// but some SDKs will put it in imp.ext.prebid instead
	displayManagerVer := buildDisplayManagerVer(request)

	var errs []error

	validImps := []openrtb2.Imp{}

	for i := range request.Imp {
		impExtIncoming, err := validateAndBuildImpExt(&request.Imp[i])
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if err := buildRequestImp(&request.Imp[i], impExtIncoming, displayManagerVer, reqInfo); err != nil {
			errs = append(errs, err)
			continue
		}

		validImps = append(validImps, request.Imp[i])
	}

	// Divide imps to several requests
	requests, errors := splitRequests(validImps, request, a.endpoint)
	return requests, append(errs, errors...)
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(response) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
	}

	var connatixResponse openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &connatixResponse); err != nil {
		return nil, []error{err}
	}

	var errs []error
	bidderResponse := adapters.NewBidderResponseWithBidsCapacity(1)
	for _, sb := range connatixResponse.SeatBid {
		for i := range sb.Bid {
			bid := sb.Bid[i]
			var bidExt bidExt
			var bidType openrtb_ext.BidType

			if err := jsonutil.Unmarshal(bid.Ext, &bidExt); err != nil {
				bidType = openrtb_ext.BidTypeBanner
			} else {
				bidType = getBidType(bidExt)
			}

			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidType,
			})
		}
	}

	bidderResponse.Currency = "USD"

	return bidderResponse, errs
}

func validateAndBuildImpExt(imp *openrtb2.Imp) (impExtIncoming, error) {
	var ext impExtIncoming
	if err := jsonutil.Unmarshal(imp.Ext, &ext); err != nil {
		return impExtIncoming{}, err
	}

	return ext, nil
}

func splitRequests(imps []openrtb2.Imp, request *openrtb2.BidRequest, uri string) ([]*adapters.RequestData, []error) {
	var errs []error
	// Initial capacity for future array of requests, memory optimization.
	// Let's say there are 35 impressions and limit impressions per request equals to 10.
	// In this case we need to create 4 requests with 10, 10, 10 and 5 impressions.
	// With this formula initial capacity=(35+10-1)/10 = 4
	initialCapacity := (len(imps) + maxImpsPerReq - 1) / maxImpsPerReq
	resArr := make([]*adapters.RequestData, 0, initialCapacity)
	startInd := 0
	impsLeft := len(imps) > 0

	headers := http.Header{}
	headers.Add("Content-Type", "application/json")
	headers.Add("Accept", "application/json")

	if request.Device != nil {
		if len(request.Device.UA) > 0 {
			headers.Add("User-Agent", request.Device.UA)
		}

		if len(request.Device.IPv6) > 0 {
			headers.Add("X-Forwarded-For", request.Device.IPv6)
		}

		if len(request.Device.IP) > 0 {
			headers.Add("X-Forwarded-For", request.Device.IP)
		}
	}

	for impsLeft {
		endInd := startInd + maxImpsPerReq
		if endInd >= len(imps) {
			endInd = len(imps)
			impsLeft = false
		}
		impsForReq := imps[startInd:endInd]
		request.Imp = impsForReq

		reqJSON, err := jsonutil.Marshal(request)
		if err != nil {
			errs = append(errs, err)
			return nil, errs
		}

		endpoint, err := url.Parse(uri)
		if err != nil {
			errs = append(errs, err)
			return nil, errs
		}

		if request.User != nil {
			userID := strings.TrimSpace(request.User.BuyerUID)

			if len(userID) > 0 {
				queryParams := url.Values{}

				if strings.HasPrefix(userID, "1-") {
					queryParams.Add("dc", "us-east-2")
				} else if strings.HasPrefix(userID, "2-") {
					queryParams.Add("dc", "us-west-2")
				} else if strings.HasPrefix(userID, "3-") {
					queryParams.Add("dc", "eu-west-1")
				}

				endpoint.RawQuery = queryParams.Encode()
			}
		}

		resArr = append(resArr, &adapters.RequestData{
			Method:  "POST",
			Uri:     endpoint.String(),
			Body:    reqJSON,
			Headers: headers,
			ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
		})
		startInd = endInd
	}
	return resArr, errs
}

func buildRequestImp(imp *openrtb2.Imp, ext impExtIncoming, displayManagerVer string, reqInfo *adapters.ExtraRequestInfo) error {
	if imp.Banner != nil {
		bannerCopy := *imp.Banner

		if bannerCopy.W == nil && bannerCopy.H == nil && len(bannerCopy.Format) > 0 {
			firstFormat := bannerCopy.Format[0]
			bannerCopy.W = &(firstFormat.W)
			bannerCopy.H = &(firstFormat.H)
		}
		imp.Banner = &bannerCopy
	}

	// Populate imp.displaymanagerver if the SDK failed to do it.
	if len(imp.DisplayManagerVer) == 0 && len(displayManagerVer) > 0 {
		imp.DisplayManagerVer = displayManagerVer
	}

	// Check if imp comes with bid floor amount defined in a foreign currency
	if imp.BidFloor > 0 && imp.BidFloorCur != "" && !strings.EqualFold(imp.BidFloorCur, "USD") {
		// Convert to US dollars
		convertedValue, err := reqInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, "USD")
		if err != nil {
			return err
		}
		// Update after conversion. All imp elements inside request.Imp are shallow copies
		// therefore, their non-pointer values are not shared memory and are safe to modify.
		imp.BidFloorCur = "USD"
		imp.BidFloor = convertedValue
	}

	impExt := impExt{
		Connatix: impExtConnatix{
			PlacementId:           ext.Bidder.PlacementId,
			ViewabilityPercentage: ext.Bidder.ViewabilityPercentage,
		},
	}

	var err error
	imp.Ext, err = json.Marshal(impExt)

	return err
}

func buildDisplayManagerVer(req *openrtb2.BidRequest) string {
	if req.App == nil {
		return ""
	}

	source, err := jsonparser.GetString(req.App.Ext, openrtb_ext.PrebidExtKey, "source")
	if err != nil {
		return ""
	}

	version, err := jsonparser.GetString(req.App.Ext, openrtb_ext.PrebidExtKey, "version")
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%s-%s", source, version)
}

func getBidType(ext bidExt) openrtb_ext.BidType {
	if ext.Cnx.MediaType == "video" {
		return openrtb_ext.BidTypeVideo
	}

	return openrtb_ext.BidTypeBanner
}
