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
	"github.com/prebid/prebid-server/pbs"
	"golang.org/x/net/context/ctxhttp"
)

// SovrnAdapter adapter to send/receive bid requests/responses to/from sovrn
type SovrnAdapter struct {
	http         *adapters.HTTPAdapter
	URI          string
	usersyncInfo *pbs.UsersyncInfo
}

type sovrnParams struct {
	TagID int `json:"tagid"`
}

// Name - export adapter name */
func (a *SovrnAdapter) Name() string {
	return "sovrn"
}

// FamilyName used for cookies and such
func (a *SovrnAdapter) FamilyName() string {
	return "sovrn"
}

// GetUsersyncInfo get the UsersyncInfo object defining sovrn user sync parameters
func (a *SovrnAdapter) GetUsersyncInfo() *pbs.UsersyncInfo {
	return a.usersyncInfo
}

// Call send bid requests to sovrn and receive responses
func (s *SovrnAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	supportedMediaTypes := []pbs.MediaType{pbs.MEDIA_TYPE_BANNER}
	sReq, err := adapters.MakeOpenRTBGeneric(req, bidder, s.FamilyName(), supportedMediaTypes, true)

	if err != nil {
		return nil, err
	}

	sovrnReq := openrtb.BidRequest{
		ID:   sReq.ID,
		Imp:  sReq.Imp,
		Site: sReq.Site,
	}

	// add tag ids to impressions
	for i, unit := range bidder.AdUnits {
		var params sovrnParams
		err = json.Unmarshal(unit.Params, &params)
		if err != nil {
			return nil, err
		}
		sovrnReq.Imp[i].Secure = sReq.Imp[i].Secure
		sovrnReq.Imp[i].TagID = strconv.Itoa(params.TagID)
		sovrnReq.Imp[i].Banner.Format = nil
	}

	reqJSON, err := json.Marshal(sovrnReq)
	if err != nil {
		return nil, err
	}

	debug := &pbs.BidderDebug{
		RequestURI: s.URI,
	}

	if req.IsDebug {
		debug.RequestBody = string(reqJSON)
		bidder.Debug = append(bidder.Debug, debug)
	}

	httpReq, _ := http.NewRequest("POST", s.URI, bytes.NewReader(reqJSON))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", sReq.Device.UA)
	httpReq.Header.Set("Referer", sReq.Site.Ref)
	httpReq.Header.Set("X-Forwarded-For", sReq.Device.IP)
	httpReq.Header.Set("Accept-Language", sReq.Device.Language)
	httpReq.Header.Set("DNT", strconv.Itoa(int(sReq.Device.DNT)))

	userID := strings.TrimSpace(sReq.User.ID)
	if len(userID) > 0 {
		httpReq.AddCookie(&http.Cookie{Name: "ljt_reader", Value: userID})
	}
	sResp, err := ctxhttp.Do(ctx, s.http.Client, httpReq)
	if err != nil {
		return nil, err
	}

	debug.StatusCode = sResp.StatusCode

	if sResp.StatusCode == 204 {
		return nil, nil
	}

	defer sResp.Body.Close()
	body, err := ioutil.ReadAll(sResp.Body)
	if err != nil {
		return nil, err
	}
	responseBody := string(body)

	if sResp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP status %d; body: %s", sResp.StatusCode, responseBody)
	}

	if req.IsDebug {
		debug.ResponseBody = responseBody
	}

	var bidResp openrtb.BidResponse
	err = json.Unmarshal(body, &bidResp)
	if err != nil {
		return nil, err
	}

	bids := make(pbs.PBSBidSlice, 0)

	numBids := 0
	for _, sb := range bidResp.SeatBid {
		for _, bid := range sb.Bid {
			numBids++

			bidID := bidder.LookupBidID(bid.ImpID)
			if bidID == "" {
				return nil, fmt.Errorf("Unknown ad unit code '%s'", bid.ImpID)
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

// SkipNoCookies whether or not to send bids to sovrn in the absence of cookies
func (a *SovrnAdapter) SkipNoCookies() bool {
	return false
}

// NewSovrnAdapter create a new SovrnAdapter instance
func NewSovrnAdapter(config *adapters.HTTPAdapterConfig, endpoint string, usersyncURL string, externalURL string) *SovrnAdapter {
	a := adapters.NewHTTPAdapter(config)

	redirectURI := fmt.Sprintf("%s/setuid?bidder=sovrn&uid=[SOVRNID]", externalURL)

	info := &pbs.UsersyncInfo{
		URL:         fmt.Sprintf("%slocation=%s", usersyncURL, url.QueryEscape(redirectURI)),
		Type:        "redirect",
		SupportCORS: false,
	}

	return &SovrnAdapter{
		http:         a,
		URI:          endpoint,
		usersyncInfo: info,
	}
}
