package brightroll

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type BrightrollAdapter struct {
	URI       string
	extraInfo ExtraInfo
}

type ExtraInfo struct {
	Accounts []Account `json:"accounts"`
}

type Account struct {
	ID       string   `json:"id"`
	Badv     []string `json:"badv"`
	Bcat     []string `json:"bcat"`
	Battr    []int8   `json:"battr"`
	BidFloor float64  `json:"bidfloor"`
}

func (a *BrightrollAdapter) MakeRequests(requestIn *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {

	request := *requestIn
	errs := make([]error, 0, len(request.Imp))
	if len(request.Imp) == 0 {
		err := &errortypes.BadInput{
			Message: "No impression in the bid request",
		}
		errs = append(errs, err)
		return nil, errs
	}

	errors := make([]error, 0, 1)

	var bidderExt adapters.ExtImpBidder
	err := json.Unmarshal(request.Imp[0].Ext, &bidderExt)
	if err != nil {
		err = &errortypes.BadInput{
			Message: "ext.bidder not provided",
		}
		errors = append(errors, err)
		return nil, errors
	}
	var brightrollExt openrtb_ext.ExtImpBrightroll
	err = json.Unmarshal(bidderExt.Bidder, &brightrollExt)
	if err != nil {
		err = &errortypes.BadInput{
			Message: "ext.bidder.publisher not provided",
		}
		errors = append(errors, err)
		return nil, errors
	}
	if brightrollExt.Publisher == "" {
		err = &errortypes.BadInput{
			Message: "publisher is empty",
		}
		errors = append(errors, err)
		return nil, errors
	}

	var account *Account
	for _, a := range a.extraInfo.Accounts {
		if a.ID == brightrollExt.Publisher {
			account = &a
			break
		}
	}

	if account == nil {
		err = &errortypes.BadInput{
			Message: "Invalid publisher",
		}
		errors = append(errors, err)
		return nil, errors
	}

	validImpExists := false
	for i := 0; i < len(request.Imp); i++ {
		//Brightroll supports only banner and video impressions as of now
		if request.Imp[i].Banner != nil {
			bannerCopy := *request.Imp[i].Banner
			if bannerCopy.W == nil && bannerCopy.H == nil && len(bannerCopy.Format) > 0 {
				firstFormat := bannerCopy.Format[0]
				bannerCopy.W = &(firstFormat.W)
				bannerCopy.H = &(firstFormat.H)
			}

			if len(account.Battr) > 0 {
				bannerCopy.BAttr = getBlockedCreativetypes(account.Battr)
			}
			request.Imp[i].Banner = &bannerCopy
			validImpExists = true
		} else if request.Imp[i].Video != nil {
			validImpExists = true
			if brightrollExt.Publisher == "adthrive" {
				videoCopy := *request.Imp[i].Video
				if len(account.Battr) > 0 {
					videoCopy.BAttr = getBlockedCreativetypes(account.Battr)
				}
				request.Imp[i].Video = &videoCopy
			}
		}
		if validImpExists && request.Imp[i].BidFloor == 0 && account.BidFloor > 0 {
			request.Imp[i].BidFloor = account.BidFloor
		}
	}
	if !validImpExists {
		err := &errortypes.BadInput{
			Message: fmt.Sprintf("No valid impression in the bid request"),
		}
		errs = append(errs, err)
		return nil, errs
	}

	request.AT = 1 //Defaulting to first price auction for all prebid requests

	if len(account.Bcat) > 0 {
		request.BCat = account.Bcat
	}

	if len(account.Badv) > 0 {
		request.BAdv = account.Badv
	}
	reqJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}
	thisURI := a.URI
	thisURI = thisURI + "?publisher=" + brightrollExt.Publisher
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("x-openrtb-version", "2.5")

	if request.Device != nil {
		addHeaderIfNonEmpty(headers, "User-Agent", request.Device.UA)
		addHeaderIfNonEmpty(headers, "X-Forwarded-For", request.Device.IP)
		addHeaderIfNonEmpty(headers, "Accept-Language", request.Device.Language)
		if request.Device.DNT != nil {
			addHeaderIfNonEmpty(headers, "DNT", strconv.Itoa(int(*request.Device.DNT)))
		}
	}

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     thisURI,
		Body:    reqJSON,
		Headers: headers,
	}}, errors
}

func (a *BrightrollAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {

	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. ", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("bad server response: %d. ", err),
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(bidResp.SeatBid[0].Bid))
	sb := bidResp.SeatBid[0]
	for i := 0; i < len(sb.Bid); i++ {
		bid := sb.Bid[i]
		bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
			Bid:     &bid,
			BidType: getMediaTypeForImp(bid.ImpID, internalRequest.Imp),
		})
	}
	return bidResponse, nil
}

func getBlockedCreativetypes(attr []int8) []openrtb2.CreativeAttribute {
	var creativeAttr []openrtb2.CreativeAttribute
	for i := 0; i < len(attr); i++ {
		creativeAttr = append(creativeAttr, openrtb2.CreativeAttribute(attr[i]))
	}
	return creativeAttr
}

//Adding header fields to request header
func addHeaderIfNonEmpty(headers http.Header, headerName string, headerValue string) {
	if len(headerValue) > 0 {
		headers.Add(headerName, headerValue)
	}
}

// getMediaTypeForImp figures out which media type this bid is for.
func getMediaTypeForImp(impId string, imps []openrtb2.Imp) openrtb_ext.BidType {
	mediaType := openrtb_ext.BidTypeBanner //default type
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			}
			return mediaType
		}
	}
	return mediaType
}

// Builder builds a new instance of the Brightroll adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	extraInfo, err := getExtraInfo(config.ExtraAdapterInfo)
	if err != nil {
		return nil, err
	}

	bidder := &BrightrollAdapter{
		URI:       config.Endpoint,
		extraInfo: extraInfo,
	}
	return bidder, nil
}

func getExtraInfo(v string) (ExtraInfo, error) {
	if len(v) == 0 {
		return getDefaultExtraInfo(), nil
	}

	var extraInfo ExtraInfo
	if err := json.Unmarshal([]byte(v), &extraInfo); err != nil {
		return extraInfo, fmt.Errorf("invalid extra info: %v", err)
	}

	return extraInfo, nil
}

func getDefaultExtraInfo() ExtraInfo {
	return ExtraInfo{
		Accounts: []Account{},
	}
}
