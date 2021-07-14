package gamma

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type GammaAdapter struct {
	URI string
}

type gammaBid struct {
	openrtb2.Bid        //base
	VastXML      string `json:"vastXml,omitempty"`
	VastURL      string `json:"vastUrl,omitempty"`
}

type gammaSeatBid struct {
	Bid   []gammaBid      `json:"bid"`
	Group int8            `json:"group,omitempty"`
	Ext   json.RawMessage `json:"ext,omitempty"`
}
type gammaBidResponse struct {
	ID         string                    `json:"id"`
	SeatBid    []gammaSeatBid            `json:"seatbid,omitempty"`
	BidID      string                    `json:"bidid,omitempty"`
	Cur        string                    `json:"cur,omitempty"`
	CustomData string                    `json:"customdata,omitempty"`
	NBR        *openrtb2.NoBidReasonCode `json:"nbr,omitempty"`
	Ext        json.RawMessage           `json:"ext,omitempty"`
}

func checkParams(gammaExt openrtb_ext.ExtImpGamma) error {
	if gammaExt.PartnerID == "" {
		return &errortypes.BadInput{
			Message: "PartnerID is empty",
		}
	}
	if gammaExt.ZoneID == "" {
		return &errortypes.BadInput{
			Message: "ZoneID is empty",
		}
	}
	if gammaExt.WebID == "" {
		return &errortypes.BadInput{
			Message: "WebID is empty",
		}
	}
	return nil
}
func (a *GammaAdapter) makeRequest(request *openrtb2.BidRequest, imp openrtb2.Imp) (*adapters.RequestData, []error) {
	var errors []error

	var bidderExt adapters.ExtImpBidder
	err := json.Unmarshal(imp.Ext, &bidderExt)
	if err != nil {
		err = &errortypes.BadInput{
			Message: "ext.bidder not provided",
		}
		errors = append(errors, err)
		return nil, errors
	}
	var gammaExt openrtb_ext.ExtImpGamma
	err = json.Unmarshal(bidderExt.Bidder, &gammaExt)
	if err != nil {
		err = &errortypes.BadInput{
			Message: "ext.bidder.publisher not provided",
		}
		errors = append(errors, err)
		return nil, errors
	}
	err = checkParams(gammaExt)
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	thisURI := a.URI
	thisURI = thisURI + "?id=" + gammaExt.PartnerID
	thisURI = thisURI + "&zid=" + gammaExt.ZoneID
	thisURI = thisURI + "&wid=" + gammaExt.WebID
	thisURI = thisURI + "&bidid=" + imp.ID
	thisURI = thisURI + "&hb=pbmobile"
	if request.Device != nil {
		if request.Device.IP != "" {
			thisURI = thisURI + "&device_ip=" + request.Device.IP
		}
		if request.Device.Model != "" {
			thisURI = thisURI + "&device_model=" + request.Device.Model
		}
		if request.Device.OS != "" {
			thisURI = thisURI + "&device_os=" + request.Device.OS
		}
		if request.Device.UA != "" {
			thisURI = thisURI + "&device_ua=" + url.QueryEscape(request.Device.UA)
		}
		if request.Device.IFA != "" {
			thisURI = thisURI + "&device_ifa=" + request.Device.IFA
		}
	}
	if request.App != nil {
		if request.App.ID != "" {
			thisURI = thisURI + "&app_id=" + request.App.ID
		}
		if request.App.Bundle != "" {
			thisURI = thisURI + "&app_bundle=" + request.App.Bundle
		}
		if request.App.Name != "" {
			thisURI = thisURI + "&app_name=" + request.App.Name
		}
	}
	headers := http.Header{}
	headers.Add("Accept", "*/*")
	headers.Add("x-openrtb-version", "2.5")
	if request.Device != nil {
		addHeaderIfNonEmpty(headers, "User-Agent", request.Device.UA)
		addHeaderIfNonEmpty(headers, "X-Forwarded-For", request.Device.IP)
		addHeaderIfNonEmpty(headers, "Accept-Language", request.Device.Language)
		if request.Device.DNT != nil {
			addHeaderIfNonEmpty(headers, "DNT", strconv.Itoa(int(*request.Device.DNT)))
		}
	}
	headers.Add("Connection", "keep-alive")
	headers.Add("cache-control", "no-cache")
	headers.Add("Accept-Encoding", "gzip, deflate")

	return &adapters.RequestData{
		Method:  "GET",
		Uri:     thisURI,
		Headers: headers,
	}, errors
}
func (a *GammaAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	errs := make([]error, 0, len(request.Imp))
	if len(request.Imp) == 0 {
		err := &errortypes.BadInput{
			Message: "No impressions in the bid request",
		}
		errs = append(errs, err)
		return nil, errs
	}
	var invalidImpIndex = make([]int, 0, 0)

	for i := 0; i < len(request.Imp); i++ {
		if request.Imp[i].Banner != nil {
			bannerCopy := *request.Imp[i].Banner
			if bannerCopy.W == nil && bannerCopy.H == nil && len(bannerCopy.Format) > 0 {
				firstFormat := bannerCopy.Format[0]
				bannerCopy.W = &(firstFormat.W)
				bannerCopy.H = &(firstFormat.H)
			}
			request.Imp[i].Banner = &bannerCopy
		} else if request.Imp[i].Video == nil {
			err := &errortypes.BadInput{
				Message: fmt.Sprintf("Gamma only supports banner and video media types. Ignoring imp id=%s", request.Imp[i].ID),
			}
			errs = append(errs, err)
			invalidImpIndex = append(invalidImpIndex, i)
		}
	}

	var adapterRequests []*adapters.RequestData
	if len(invalidImpIndex) == 0 {
		for _, imp := range request.Imp {
			adapterReq, errors := a.makeRequest(request, imp)
			if adapterReq != nil {
				adapterRequests = append(adapterRequests, adapterReq)
			}
			errs = append(errs, errors...)
		}
	} else if len(request.Imp) == len(invalidImpIndex) {
		//only true if every Imp was not a Banner or a Video
		err := &errortypes.BadInput{
			Message: fmt.Sprintf("No valid impression in the bid request"),
		}
		errs = append(errs, err)
		return nil, errs
	} else {
		var j int = 0
		for i := 0; i < len(request.Imp); i++ {
			if j < len(invalidImpIndex) && i == invalidImpIndex[j] {
				j++
			} else {
				adapterReq, errors := a.makeRequest(request, request.Imp[i])
				if adapterReq != nil {
					adapterRequests = append(adapterRequests, adapterReq)
				}
				errs = append(errs, errors...)
			}
		}
	}

	return adapterRequests, errs
}

func convertBid(gBid gammaBid, mediaType openrtb_ext.BidType) *openrtb2.Bid {
	var bid openrtb2.Bid
	bid = gBid.Bid

	if mediaType == openrtb_ext.BidTypeVideo {
		//Return inline VAST XML Document (Section 6.4.2)
		if len(gBid.VastXML) > 0 {
			if len(gBid.VastURL) > 0 {
				bid.NURL = gBid.VastURL
			}
			bid.AdM = gBid.VastXML
		} else {
			return nil
		}
	} else {
		if len(gBid.Bid.AdM) == 0 {
			return nil
		}
	}
	return &bid
}

func (a *GammaAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var gammaResp gammaBidResponse
	if err := json.Unmarshal(response.Body, &gammaResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("bad server response: %d. ", err),
		}}
	}

	//(Section 7.1 No-Bid Signaling)
	if len(gammaResp.SeatBid) == 0 {
		return nil, nil
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(gammaResp.SeatBid[0].Bid))
	errs := make([]error, 0, len(gammaResp.SeatBid[0].Bid))
	for _, sb := range gammaResp.SeatBid {
		for i := range sb.Bid {
			mediaType := getMediaTypeForImp(gammaResp.ID, internalRequest.Imp)
			bid := convertBid(sb.Bid[i], mediaType)
			if bid != nil {
				bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
					Bid:     bid,
					BidType: mediaType,
				})
			} else {
				err := &errortypes.BadServerResponse{
					Message: fmt.Sprintf("Missing Ad Markup. Run with request.debug = 1 for more info"),
				}
				errs = append(errs, err)
			}
		}
	}
	return bidResponse, errs
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

// Builder builds a new instance of the Gamma adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &GammaAdapter{
		URI: config.Endpoint,
	}
	return bidder, nil
}
