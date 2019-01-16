package sonobi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
	"golang.org/x/net/context/ctxhttp"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strconv"
)

// SonobiAdapter - Sonobi SonobiAdapter definition
type SonobiAdapter struct {
	http *adapters.HTTPAdapter
	URI  string
}

// Name returns the name fo cookie stuff
func (a *SonobiAdapter) Name() string {
	return "sonobi"
}

//SkipNoCookies flag for skipping no cookies...
func (a *SonobiAdapter) SkipNoCookies() bool {
	return false
}

// NewSonobiAdapter create a new SovrnSonobiAdapter instance
func NewSonobiAdapter(config *adapters.HTTPAdapterConfig, endpoint string) *SonobiAdapter {
	return NewSonobiBidder(adapters.NewHTTPAdapter(config).Client, endpoint)
}

// NewSonobiBidder Initializes the Bidder
func NewSonobiBidder(client *http.Client, endpoint string) *SonobiAdapter {
	a := &adapters.HTTPAdapter{Client: client}

	return &SonobiAdapter{
		http: a,
		URI:  endpoint,
	}
}

type sonobiParams struct {
	TagID string `json:"TagID"`
}

// Call OpenRTB request to sonobi and parse the response into prebid server bids
func (a *SonobiAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	supportedMediaTypes := []pbs.MediaType{pbs.MEDIA_TYPE_BANNER}
	sReq, err := adapters.MakeOpenRTBGeneric(req, bidder, a.Name(), supportedMediaTypes)
	if err != nil {
		return nil, err
	}
	sonobiReq := openrtb.BidRequest{
		ID:   sReq.ID,
		Imp:  sReq.Imp,
		Site: sReq.Site,
		User: sReq.User,
		Regs: sReq.Regs,
	}

	// add tag ids to impressions
	for i, unit := range bidder.AdUnits {
		var params openrtb_ext.ExtImpSonobi
		err = json.Unmarshal(unit.Params, &params)
		if err != nil {
			return nil, err
		}

		// Fixes some segfaults. Since this is legacy code, I'm not looking into it too deeply
		if len(sonobiReq.Imp) <= i {
			break
		}
		sonobiReq.Imp[i].TagID = params.TagID
	}

	reqJSON, err := json.Marshal(sonobiReq)
	if err != nil {
		return nil, err
	}

	debug := &pbs.BidderDebug{
		RequestURI: a.URI,
	}
	httpReq, _ := http.NewRequest("POST", a.URI, bytes.NewReader(reqJSON))
	httpReq.Header.Set("Content-Type", "application/json")
	if sReq.Device != nil {
		addHeaderIfNonEmpty(httpReq.Header, "User-Agent", sReq.Device.UA)
		addHeaderIfNonEmpty(httpReq.Header, "X-Forwarded-For", sReq.Device.IP)
		addHeaderIfNonEmpty(httpReq.Header, "Accept-Language", sReq.Device.Language)
		addHeaderIfNonEmpty(httpReq.Header, "DNT", strconv.Itoa(int(sReq.Device.DNT)))
	}
	sResp, err := ctxhttp.Do(ctx, a.http.Client, httpReq)
	if err != nil {
		return nil, err
	}
	defer sResp.Body.Close()
	debug.StatusCode = sResp.StatusCode
	if sResp.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	body, err := ioutil.ReadAll(sResp.Body)
	if err != nil {
		return nil, err
	}
	responseBody := string(body)
	if sResp.StatusCode == http.StatusBadRequest {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("HTTP status %d; body: %s", sResp.StatusCode, responseBody),
		}
	}

	if sResp.StatusCode != http.StatusOK {
		return nil, &errortypes.BadServerResponse{
			Message: fmt.Sprintf("HTTP status %d; body: %s", sResp.StatusCode, responseBody),
		}
	}
	if req.IsDebug {
		debug.RequestBody = string(reqJSON)
		bidder.Debug = append(bidder.Debug, debug)
		debug.ResponseBody = responseBody
	}

	var bidResp openrtb.BidResponse
	err = json.Unmarshal(body, &bidResp)
	if err != nil {
		return nil, &errortypes.BadServerResponse{
			Message: err.Error(),
		}
	}
	bids := make(pbs.PBSBidSlice, 0)
	for _, sb := range bidResp.SeatBid {
		for _, bid := range sb.Bid {
			bidID := bidder.LookupBidID(bid.ImpID)
			if bidID == "" {
				return nil, &errortypes.BadServerResponse{
					Message: fmt.Sprintf("Unknown ad unit code '%s'", bid.ImpID),
				}
			}

			adm, _ := url.QueryUnescape(bid.AdM)
			pbid := pbs.PBSBid{
				BidID:       bidID,
				AdUnitCode:  bid.ImpID,
				BidderCode:  bidder.BidderCode,
				Price:       bid.Price,
				Adm:         adm,
				Creative_id: bid.CrID,
				Width:       bid.W,
				Height:      bid.H,
				DealId:      bid.DealID,
				NURL:        bid.NURL,
			}
			bids = append(bids, &pbid)
		}
	}

	sort.Sort(bids)
	return bids, nil
}

// MakeRequests Makes the OpenRTB request payload
func (a *SonobiAdapter) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
	var errs []error
	var sonobiExt openrtb_ext.ExtImpSonobi
	var err error
	// NOTE: sonobi only supports 1 impression. Only the first will be considered until Sonobi supports more than 1.

	var adapterRequests []*adapters.RequestData

	for _, imp := range request.Imp {
		// Sonobi doesn't allow multi-type imp. Banner takes priority over video.
		if imp.Banner != nil {
		} else if imp.Video != nil {
		} else {
			err := fmt.Errorf("Sonobi only supports banner and video imps. Ignoring imp id=%s", imp.ID)
			errs = append(errs, err)
		}

		// Make a copy as we don't want to change the original request
		reqCopy := *request
		reqCopy.Imp = append(make([]openrtb.Imp, 0), imp)

		for i := range reqCopy.Imp {
			var bidderExt adapters.ExtImpBidder
			if err = json.Unmarshal(reqCopy.Imp[i].Ext, &bidderExt); err != nil {
				errs = append(errs, err)
				continue
			}

			if err = json.Unmarshal(bidderExt.Bidder, &sonobiExt); err != nil {

				errs = append(errs, err)
				continue
			}
			reqCopy.Imp[i].TagID = sonobiExt.TagID
		}

		adapterReq, errors := a.makeRequest(&reqCopy)
		if len(sonobiExt.PubID) > 0 {
			if adapterReq != nil {
				adapterRequests = append(adapterRequests, adapterReq)
			}
			adapterReq.Uri = adapterReq.Uri + "=" + sonobiExt.PubID
		} else {
			err := fmt.Errorf("Missing PubID for imp id=%s", imp.ID)
			errs = append(errs, err)
		}
		errs = append(errs, errors...)
	}

	return adapterRequests, errs

}

// makeRequest helper method to crete the http request data
func (a *SonobiAdapter) makeRequest(request *openrtb.BidRequest) (*adapters.RequestData, []error) {

	var errs []error

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
		Uri:     a.URI,
		Body:    reqJSON,
		Headers: headers,
	}, errs
}

// MakeBids makes the bids
func (a *SonobiAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: getMediaTypeForImp(sb.Bid[i].ImpID, internalRequest.Imp),
			})
		}
	}
	return bidResponse, nil
}

func getMediaTypeForImp(impID string, imps []openrtb.Imp) openrtb_ext.BidType {
	mediaType := openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Banner == nil && imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			}
			return mediaType
		}
	}
	return mediaType
}

func addHeaderIfNonEmpty(headers http.Header, headerName string, headerValue string) {
	if len(headerValue) > 0 {
		headers.Add(headerName, headerValue)
	}
}
