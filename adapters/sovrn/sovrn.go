package sovrn

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
	"golang.org/x/net/context/ctxhttp"
)

type SovrnAdapter struct {
	http *adapters.HTTPAdapter
	URI  string
}

// Name - export adapter name */
func (s *SovrnAdapter) Name() string {
	return "sovrn"
}

// FamilyName used for cookies and such
func (s *SovrnAdapter) FamilyName() string {
	return "sovrn"
}

func (s *SovrnAdapter) SkipNoCookies() bool {
	return false
}

// Call send bid requests to sovrn and receive responses
func (s *SovrnAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	supportedMediaTypes := []pbs.MediaType{pbs.MEDIA_TYPE_BANNER}
	sReq, err := adapters.MakeOpenRTBGeneric(req, bidder, s.FamilyName(), supportedMediaTypes)

	if err != nil {
		return nil, err
	}

	sovrnReq := openrtb.BidRequest{
		ID:   sReq.ID,
		Imp:  sReq.Imp,
		Site: sReq.Site,
		User: sReq.User,
		Regs: sReq.Regs,
	}

	// add tag ids to impressions
	for i, unit := range bidder.AdUnits {
		var params openrtb_ext.ExtImpSovrn
		err = json.Unmarshal(unit.Params, &params)
		if err != nil {
			return nil, err
		}

		// Fixes some segfaults. Since this is legacy code, I'm not looking into it too deeply
		if len(sovrnReq.Imp) <= i {
			break
		}
		sovrnReq.Imp[i].TagID = params.TagId
	}

	reqJSON, err := json.Marshal(sovrnReq)
	if err != nil {
		return nil, err
	}

	debug := &pbs.BidderDebug{
		RequestURI: s.URI,
	}

	httpReq, _ := http.NewRequest("POST", s.URI, bytes.NewReader(reqJSON))
	httpReq.Header.Set("Content-Type", "application/json")
	if sReq.Device != nil {
		addHeaderIfNonEmpty(httpReq.Header, "User-Agent", sReq.Device.UA)
		addHeaderIfNonEmpty(httpReq.Header, "X-Forwarded-For", sReq.Device.IP)
		addHeaderIfNonEmpty(httpReq.Header, "Accept-Language", sReq.Device.Language)
		addHeaderIfNonEmpty(httpReq.Header, "DNT", strconv.Itoa(int(sReq.Device.DNT)))
	}
	if sReq.User != nil {
		userID := strings.TrimSpace(sReq.User.BuyerUID)
		if len(userID) > 0 {
			httpReq.AddCookie(&http.Cookie{Name: "ljt_reader", Value: userID})
		}
	}
	sResp, err := ctxhttp.Do(ctx, s.http.Client, httpReq)
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
		return nil, &adapters.BadInputError{
			Message: fmt.Sprintf("HTTP status %d; body: %s", sResp.StatusCode, responseBody),
		}
	}

	if sResp.StatusCode != http.StatusOK {
		return nil, &adapters.BadServerResponseError{
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
		return nil, &adapters.BadServerResponseError{
			Message: err.Error(),
		}
	}

	bids := make(pbs.PBSBidSlice, 0)

	for _, sb := range bidResp.SeatBid {
		for _, bid := range sb.Bid {
			bidID := bidder.LookupBidID(bid.ImpID)
			if bidID == "" {
				return nil, &adapters.BadServerResponseError{
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

func (s *SovrnAdapter) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
	errs := make([]error, 0, len(request.Imp))

	for i := 0; i < len(request.Imp); i++ {
		_, err := preprocess(&request.Imp[i])
		if err != nil {
			errs = append(errs, err)
			request.Imp = append(request.Imp[:i], request.Imp[i+1:]...)
			i--
		}
	}

	// If all the requests were malformed, don't bother making a server call with no impressions.
	if len(request.Imp) == 0 {
		return nil, errs
	}

	reqJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json")
	if request.Device != nil {
		addHeaderIfNonEmpty(headers, "User-Agent", request.Device.UA)
		addHeaderIfNonEmpty(headers, "X-Forwarded-For", request.Device.IP)
		addHeaderIfNonEmpty(headers, "Accept-Language", request.Device.Language)
		addHeaderIfNonEmpty(headers, "DNT", strconv.Itoa(int(request.Device.DNT)))
	}

	if request.User != nil {
		userID := strings.TrimSpace(request.User.BuyerUID)
		if len(userID) > 0 {
			headers.Add("Cookie", fmt.Sprintf("%s=%s", "ljt_reader", userID))
		}
	}

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     s.URI,
		Body:    reqJSON,
		Headers: headers,
	}}, errs
}

func addHeaderIfNonEmpty(headers http.Header, headerName string, headerValue string) {
	if len(headerValue) > 0 {
		headers.Add(headerName, headerValue)
	}
}
func (s *SovrnAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&adapters.BadInputError{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&adapters.BadServerResponseError{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{&adapters.BadServerResponseError{
			Message: err.Error(),
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	for _, sb := range bidResp.SeatBid {
		for i := 0; i < len(sb.Bid); i++ {
			bid := sb.Bid[i]
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: openrtb_ext.BidTypeBanner,
			})
		}
	}

	return bidResponse, nil
}

func preprocess(imp *openrtb.Imp) (string, error) {
	// We currently only support banner impressions
	if imp.Native != nil || imp.Audio != nil || imp.Video != nil {
		return "", &adapters.BadInputError{
			Message: fmt.Sprintf("Sovrn doesn't support audio, video, or native Imps. Ignoring Imp ID=%s", imp.ID),
		}
	}

	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return "", &adapters.BadInputError{
			Message: err.Error(),
		}
	}

	var sovrnExt openrtb_ext.ExtImpSovrn
	if err := json.Unmarshal(bidderExt.Bidder, &sovrnExt); err != nil {
		return "", &adapters.BadInputError{
			Message: err.Error(),
		}
	}

	imp.TagID = sovrnExt.TagId
	imp.BidFloor = sovrnExt.BidFloor

	return imp.TagID, nil
}

// NewSovrnAdapter create a new SovrnAdapter instance
func NewSovrnAdapter(config *adapters.HTTPAdapterConfig, endpoint string) *SovrnAdapter {
	return NewSovrnBidder(adapters.NewHTTPAdapter(config).Client, endpoint)
}

func NewSovrnBidder(client *http.Client, endpoint string) *SovrnAdapter {
	a := &adapters.HTTPAdapter{Client: client}

	return &SovrnAdapter{
		http: a,
		URI:  endpoint,
	}
}
