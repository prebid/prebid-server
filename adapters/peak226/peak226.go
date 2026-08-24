package peak226

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/macros"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

const (
	defaultRegion = "us"
	currencyUSD   = "USD"
	// zeroIFA is the sentinel value the OS reports for device.ifa when the user has not
	// granted app tracking permission (e.g. iOS ATT declined). It is not a real device ID.
	zeroIFA = "00000000-0000-0000-0000-000000000000"
)

type adapter struct {
	endpoint *template.Template
}

// Builder builds a new instance of the Peak226 adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	endpointTemplate, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &adapter{
		endpoint: endpointTemplate,
	}
	return bidder, nil
}

// peak226ImpCtx pairs a processed impression with the publisher ID declared on it, so that
// each region group can resolve its own effective publisher ID independently of the others.
type peak226ImpCtx struct {
	imp         openrtb2.Imp
	publisherID string
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error

	impsByRegion := make(map[string][]peak226ImpCtx)
	var regionOrder []string

	for _, imp := range request.Imp {
		var bidderExt adapters.ExtImpBidder
		if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: fmt.Sprintf("imp #%s: %s", imp.ID, err.Error()),
			})
			continue
		}

		var peak226Ext openrtb_ext.ImpExtPeak226
		if err := jsonutil.Unmarshal(bidderExt.Bidder, &peak226Ext); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: fmt.Sprintf("imp #%s: %s", imp.ID, err.Error()),
			})
			continue
		}

		imp.TagID = peak226Ext.PlacementID

		if err := stripBidderExt(&imp); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: fmt.Sprintf("imp #%s: %s", imp.ID, err.Error()),
			})
			continue
		}

		if imp.BidFloor > 0 && imp.BidFloorCur != "" && !strings.EqualFold(imp.BidFloorCur, currencyUSD) {
			convertedValue, err := reqInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, currencyUSD)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			imp.BidFloor = convertedValue
			imp.BidFloorCur = currencyUSD
		}

		region := peak226Ext.Region
		if region == "" {
			region = defaultRegion
		}

		if _, ok := impsByRegion[region]; !ok {
			regionOrder = append(regionOrder, region)
		}
		impsByRegion[region] = append(impsByRegion[region], peak226ImpCtx{
			imp:         imp,
			publisherID: peak226Ext.PublisherID,
		})
	}

	if len(regionOrder) == 0 {
		return nil, errs
	}

	device := sanitizeDevice(request.Device)
	requests := make([]*adapters.RequestData, 0, len(regionOrder))

	for _, region := range regionOrder {
		group := impsByRegion[region]

		imps := make([]openrtb2.Imp, 0, len(group))
		var publisherID string
		for _, ctx := range group {
			imps = append(imps, ctx.imp)
			if publisherID == "" && ctx.publisherID != "" {
				publisherID = ctx.publisherID
			}
		}

		requestCopy := *request
		requestCopy.Imp = imps
		requestCopy.Device = device
		setPublisherID(&requestCopy, publisherID)

		endpoint, err := macros.ResolveMacros(a.endpoint, macros.EndpointTemplateParams{Region: region})
		if err != nil {
			errs = append(errs, err)
			continue
		}

		requestJSON, err := jsonutil.Marshal(&requestCopy)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		headers := http.Header{}
		headers.Add("Content-Type", "application/json;charset=utf-8")
		headers.Add("Accept", "application/json")

		requests = append(requests, &adapters.RequestData{
			Method:  http.MethodPost,
			Uri:     endpoint,
			Body:    requestJSON,
			Headers: headers,
			ImpIDs:  openrtb_ext.GetImpIDs(requestCopy.Imp),
		})
	}

	if len(requests) == 0 {
		return nil, errs
	}

	return requests, errs
}

// stripBidderExt removes only the "bidder" key from imp.ext, preserving non-bidder
// signals such as gpid, data and tid that the Prebid.js adapter also forwards. When
// nothing else remains, imp.Ext is cleared so the imp serializes without an empty "ext".
func stripBidderExt(imp *openrtb2.Imp) error {
	if len(imp.Ext) == 0 {
		imp.Ext = nil
		return nil
	}

	var ext map[string]json.RawMessage
	if err := jsonutil.Unmarshal(imp.Ext, &ext); err != nil {
		return err
	}

	delete(ext, "bidder")

	if len(ext) == 0 {
		imp.Ext = nil
		return nil
	}

	updatedExt, err := jsonutil.Marshal(ext)
	if err != nil {
		return err
	}
	imp.Ext = updatedExt

	return nil
}

// setPublisherID mirrors the Prebid.js adapter's behavior of writing the publisherId
// param onto app.publisher.id when the request is an app request, or site.publisher.id otherwise.
func setPublisherID(request *openrtb2.BidRequest, publisherID string) {
	if publisherID == "" {
		return
	}

	if request.App != nil {
		appCopy := *request.App
		appCopy.Publisher = clonePublisher(appCopy.Publisher, publisherID)
		request.App = &appCopy
		return
	}

	var siteCopy openrtb2.Site
	if request.Site != nil {
		siteCopy = *request.Site
	}
	siteCopy.Publisher = clonePublisher(siteCopy.Publisher, publisherID)
	request.Site = &siteCopy
}

// sanitizeDevice clears device.ifa when it's the all-zero sentinel value reported by the
// OS when app tracking permission was declined, so it's never forwarded as if it were a
// real device ID. There is no cookie-sync equivalent for app traffic; device.ifa and
// user.eids (from an in-app ID SDK, if the app integrates one) are the identity signals
// available for app-originated requests, and both already pass through unmodified otherwise.
func sanitizeDevice(device *openrtb2.Device) *openrtb2.Device {
	if device == nil || device.IFA != zeroIFA {
		return device
	}
	deviceCopy := *device
	deviceCopy.IFA = ""
	return &deviceCopy
}

func clonePublisher(publisher *openrtb2.Publisher, id string) *openrtb2.Publisher {
	if publisher == nil {
		return &openrtb2.Publisher{ID: id}
	}
	publisherCopy := *publisher
	publisherCopy.ID = id
	return &publisherCopy
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(response) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("bad server response: %s", err.Error()),
		}}
	}

	if len(bidResp.SeatBid) == 0 {
		return adapters.NewBidderResponse(), nil
	}

	var errs []error
	bidderResponse := adapters.NewBidderResponseWithBidsCapacity(len(bidResp.SeatBid[0].Bid))
	if bidResp.Cur != "" {
		bidderResponse.Currency = bidResp.Cur
	}

	for _, seatBid := range bidResp.SeatBid {
		for i := range seatBid.Bid {
			bid := seatBid.Bid[i]

			bidType, err := getMediaTypeForBid(bid)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			resolveMacros(&bid)

			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidType,
			})
		}
	}

	return bidderResponse, errs
}

// resolveMacros substitutes the OpenRTB ${AUCTION_PRICE} macro in adm and nurl with the
// bid price. peak226 always returns the macro in adm and relies on the demand-side adapter
// to expand it, so leaving it unresolved would render the literal macro text in the creative
// and report the wrong price on the win notice.
func resolveMacros(bid *openrtb2.Bid) {
	if bid == nil {
		return
	}
	price := strconv.FormatFloat(bid.Price, 'f', -1, 64)
	bid.AdM = strings.Replace(bid.AdM, "${AUCTION_PRICE}", price, -1)
	bid.NURL = strings.Replace(bid.NURL, "${AUCTION_PRICE}", price, -1)
}

func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	}

	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("unrecognized bid type for impression %s", bid.ImpID),
	}
}
