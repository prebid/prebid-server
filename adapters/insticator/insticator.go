package insticator

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/errortypes"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

// type adapter struct {
// 	endpoint string
// }

type Ext struct {
	Insticator impInsticatorExt `json:"insticator"`
}

type insticatorUserExt struct {
	Eids    []openrtb2.EID  `json:"eids,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
	Consent string          `json:"consent,omitempty"`
}

type impInsticatorExt struct {
	AdUnitId    string `json:"adUnitId,omitempty"`
	PublisherId string `json:"publisherId,omitempty"`
}

type adapter struct {
	endpoint string
}

type reqExt struct {
	Insticator *reqInsticatorExt `json:"insticator,omitempty"`
}

type reqInsticatorExt struct {
	Caller []InsticatorCaller `json:"caller,omitempty"`
}

type InsticatorCaller struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

// CALLER Info used to track Prebid Server
// as one of the hops in the request to exchange
var CALLER = InsticatorCaller{"Prebid-Server", "n/a"}

type bidExt struct {
	Insticator bidInsticatorExt `json:"insticator,omitempty"`
}

type bidInsticatorExt struct {
	MediaType string `json:"mediaType,omitempty"`
}

// Placeholder for the actual openrtb2.Video struct
type Video struct {
	W              *int     `json:"w"`
	H              *int     `json:"h"`
	MIMEs          []string `json:"mimes"`
	Placement      int      `json:"placement"`
	Plcmt          int      `json:"plcmt"`
	MinDuration    *int     `json:"minduration"`
	MaxDuration    *int     `json:"maxduration"`
	Protocols      []int    `json:"protocols"`
	StartDelay     *int     `json:"startdelay"`
	Linearity      *int     `json:"linearity"`
	Skip           *int     `json:"skip"`
	SkipMin        *int     `json:"skipmin"`
	SkipAfter      *int     `json:"skipafter"`
	Sequence       *int     `json:"sequence"`
	Battr          []int    `json:"battr"`
	MaxExtended    *int     `json:"maxextended"`
	MinBitrate     *int     `json:"minbitrate"`
	MaxBitrate     *int     `json:"maxbitrate"`
	PlaybackMethod []int    `json:"playbackmethod"`
	PlaybackEnd    *int     `json:"playbackend"`
	Delivery       []int    `json:"delivery"`
	Pos            *int     `json:"pos"`
	API            []int    `json:"api"`
}

type BadInput struct {
	Message string
}

func (e *BadInput) Error() string {
	return e.Message
}

// Validation functions
func isInteger(value interface{}) bool {
	switch v := value.(type) {
	case int:
		return true
	case float64:
		return v == float64(int(v))
	case string:
		_, err := strconv.Atoi(v)
		return err == nil
	default:
		return false
	}
}

func isArrayOfNums(value interface{}) bool {
	switch v := value.(type) {
	case []int:
		return true
	case []float64:
		for _, num := range v {
			if num != float64(int(num)) {
				return false
			}
		}
		return true
	case []string:
		for _, str := range v {
			if _, err := strconv.Atoi(str); err != nil {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// Define valid values
var validLinearity = map[int]bool{1: true}
var validSkip = map[int]bool{0: true, 1: true}
var validPlaybackEnd = map[int]bool{1: true, 2: true, 3: true}
var validPos = map[int]bool{0: true, 1: true, 2: true, 3: true, 4: true, 5: true, 6: true, 7: true}

// Map parameters to validation functions
var optionalVideoParams = map[string]func(interface{}) bool{
	"minduration":    isInteger,
	"maxduration":    isInteger,
	"protocols":      isArrayOfNums,
	"startdelay":     isInteger,
	"linearity":      func(value interface{}) bool { return isInteger(value) && validLinearity[toInt(value)] },
	"skip":           func(value interface{}) bool { return isInteger(value) && validSkip[toInt(value)] },
	"skipmin":        isInteger,
	"skipafter":      isInteger,
	"sequence":       isInteger,
	"battr":          isArrayOfNums,
	"maxextended":    isInteger,
	"minbitrate":     isInteger,
	"maxbitrate":     isInteger,
	"playbackmethod": isArrayOfNums,
	"playbackend":    func(value interface{}) bool { return isInteger(value) && validPlaybackEnd[toInt(value)] },
	"delivery":       isArrayOfNums,
	"pos":            func(value interface{}) bool { return isInteger(value) && validPos[toInt(value)] },
	"api":            isArrayOfNums,
}

// Helper function to convert interface to int
func toInt(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return 0
}

// Builder builds a new insticatorance of the Foo adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

// getMediaTypeForImp figures out which media type this bid is for
func getMediaTypeForBid(bid *openrtb2.Bid) openrtb_ext.BidType {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo
	default:
		return openrtb_ext.BidTypeBanner
	}
}

func getBidType(ext bidExt) openrtb_ext.BidType {
	if ext.Insticator.MediaType == "video" {
		return openrtb_ext.BidTypeVideo
	}

	return openrtb_ext.BidTypeBanner
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {

	var errs []error
	var adapterRequests []*adapters.RequestData
	var groupedImps = make(map[string][]openrtb2.Imp)

	reqExt, err := makeReqExt(request)
	if err != nil {
		errs = append(errs, err)
	}

	request.Ext = reqExt

	for i := 0; i < len(request.Imp); i++ {
		if impCopy, err := makeImps(request.Imp[i]); err == nil {
			var impExt Ext
			// Populate site.publisher.id from imp extension only once
			if request.Site != nil && i == 0 {
				populatePublisherId(&impCopy, request)
			} else if request.App != nil && i == 0 {
				populatePublisherId(&impCopy, request)
			}

			// group together the imp hacing insticator adUnitId. However let's not block request creation.
			if err := json.Unmarshal(impCopy.Ext, &impExt); err == nil {
				impKey := impExt.Insticator.AdUnitId

				resolvedBidFloor, errFloor := resolveBidFloor(impCopy.BidFloor, impCopy.BidFloorCur, requestInfo)
				if errFloor != nil {
					errs = append(errs, errFloor)
				} else {
					if resolvedBidFloor > 0 {
						impCopy.BidFloor = resolvedBidFloor
						impCopy.BidFloorCur = "USD"
					}
				}

				groupedImps[impKey] = append(groupedImps[impKey], impCopy)
			} else {
				errs = append(errs, err)
			}
		} else {
			errs = append(errs, err)
		}
	}

	for _, impList := range groupedImps {
		if adapterReq, err := a.makeRequest(*request, impList); err == nil {
			adapterRequests = append(adapterRequests, adapterReq)
		} else {
			errs = append(errs, err)
		}
	}
	return adapterRequests, errs
}

func (a *adapter) makeRequest(request openrtb2.BidRequest, impList []openrtb2.Imp) (*adapters.RequestData, error) {
	request.Imp = impList

	reqJSON, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
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
			headers.Add("IP", request.Device.IP)
		}
	}

	return &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    reqJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}, nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode == http.StatusBadRequest {
		err := &errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info.", responseData.StatusCode),
		}
		return nil, []error{err}
	}

	if responseData.StatusCode != http.StatusOK {
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info.", responseData.StatusCode),
		}
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bid := &seatBid.Bid[i]
			bidType := getMediaTypeForBid(bid)
			b := &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}
	return bidResponse, nil
}

func makeImps(imp openrtb2.Imp) (openrtb2.Imp, error) {
	if imp.Banner == nil && imp.Video == nil {
		return openrtb2.Imp{}, &errortypes.BadInput{
			Message: fmt.Sprintf("Imp ID %s must have at least one of [Banner, Video] defined", imp.ID),
		}
	}

	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return openrtb2.Imp{}, &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	var insticatorExt openrtb_ext.ExtImpInsticator
	if err := json.Unmarshal(bidderExt.Bidder, &insticatorExt); err != nil {
		return openrtb2.Imp{}, &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	var impExt Ext
	impExt.Insticator.AdUnitId = insticatorExt.AdUnitId
	impExt.Insticator.PublisherId = insticatorExt.PublisherId

	impExtJSON, err := json.Marshal(impExt)
	if err != nil {
		return openrtb2.Imp{}, &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	imp.Ext = impExtJSON

	// Validate Video if it exists
	if imp.Video != nil {
		videoCopy, err := validateVideoParams(imp.Video)

		imp.Video = videoCopy

		if err != nil {
			return openrtb2.Imp{}, &errortypes.BadInput{
				Message: err.Error(),
			}
		}
	}
	return imp, nil
}

func validateVideoParams(video *openrtb2.Video) (*openrtb2.Video, error) {
	videoCopy := *video
	if (videoCopy.W == nil || *videoCopy.W == 0) ||
		(videoCopy.H == nil || *videoCopy.H == 0) ||
		videoCopy.MIMEs == nil {

		return nil, &errortypes.BadInput{
			Message: "One or more invalid or missing video field(s) w, h, mimes",
		}
	}

	// Validate optional parameters and remove invalid ones
	cleanedVideo, err := validateOptionalVideoParams(&videoCopy)
	if err != nil {
		return nil, err
	}

	return cleanedVideo, nil
}

func makeReqExt(request *openrtb2.BidRequest) ([]byte, error) {
	var reqExt reqExt

	if len(request.Ext) > 0 {
		if err := json.Unmarshal(request.Ext, &reqExt); err != nil {
			return nil, err
		}
	}

	if reqExt.Insticator == nil {
		reqExt.Insticator = &reqInsticatorExt{}
	}

	if reqExt.Insticator.Caller == nil {
		reqExt.Insticator.Caller = make([]InsticatorCaller, 0)
	}

	reqExt.Insticator.Caller = append(reqExt.Insticator.Caller, CALLER)

	return json.Marshal(reqExt)
}

func resolveBidFloor(bidFloor float64, bidFloorCur string, reqInfo *adapters.ExtraRequestInfo) (float64, error) {
	if bidFloor > 0 && bidFloorCur != "" && strings.ToUpper(bidFloorCur) != "USD" {
		floor, err := reqInfo.ConvertCurrency(bidFloor, bidFloorCur, "USD")
		return roundTo4Decimals(floor), err
	}

	return bidFloor, nil
}

// roundTo4Decimals function
func roundTo4Decimals(amount float64) float64 {
	return math.Round(amount*10000) / 10000
}

func validateOptionalVideoParams(video *openrtb2.Video) (*openrtb2.Video, error) {
	v := reflect.ValueOf(video).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Convert the field name to camelCase for matching in optionalVideoParams map
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" {
			jsonTag = toCamelCase(field.Name)
		}

		// Skip fields that are not in optionalVideoParams
		validator, exists := optionalVideoParams[jsonTag]
		if !exists {
			continue
		}

		// Check if the field value is zero/nil and skip if true
		if isZeroOrNil(fieldValue) {
			continue
		}

		// Validate the field value
		if !validator(fieldValue.Interface()) {
			// If invalid, set the field to zero value
			fieldValue.Set(reflect.Zero(fieldValue.Type()))
		}
	}
	return video, nil
}

// Helper function to convert field name to camelCase
func toCamelCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(string(s[0])) + s[1:]
}

// Helper function to check if a value is zero or nil
func isZeroOrNil(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.Ptr, reflect.Interface:
		return value.IsNil()
	case reflect.Slice, reflect.Array:
		return value.Len() == 0
	case reflect.Map:
		return len(value.MapKeys()) == 0
	default:
		return value.IsZero()
	}
}

// populate publisherId to site/app object from imp extension
func populatePublisherId(imp *openrtb2.Imp, request *openrtb2.BidRequest) {
	var ext Ext

	if request.Site != nil && request.Site.Publisher == nil {
		request.Site.Publisher = &openrtb2.Publisher{}
		if err := json.Unmarshal(imp.Ext, &ext); err == nil {
			request.Site.Publisher.ID = ext.Insticator.PublisherId
		} else {
			log.Printf("Error unmarshalling imp extension: %v", err)
		}
	}

	if request.App != nil && request.App.Publisher == nil {
		request.App.Publisher = &openrtb2.Publisher{}
		if err := json.Unmarshal(imp.Ext, &ext); err == nil {
			request.App.Publisher.ID = ext.Insticator.PublisherId
		} else {
			log.Printf("Error unmarshalling imp extension: %v", err)
		}
	}
}
