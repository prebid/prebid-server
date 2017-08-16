package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/prebid/openrtb"
	"github.com/prebid/prebid-server/pbs"
	"golang.org/x/net/context/ctxhttp"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

type SovrnAdapter struct {
	http         *HTTPAdapter
	URI          string
	usersyncInfo *pbs.UsersyncInfo
}

type sovrnParams struct {
	TagId int `json:"tagid"`
}

/* Name - export adapter name */
func (a *SovrnAdapter) Name() string {
	return "sovrn"
}

// used for cookies and such
func (a *SovrnAdapter) FamilyName() string {
	return "sovrn"
}

func (a *SovrnAdapter) GetUsersyncInfo() *pbs.UsersyncInfo {
	return a.usersyncInfo
}

func (s *SovrnAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	sReq := makeOpenRTBGeneric(req, bidder, s.FamilyName())

	// Sovrn only needs a few things from the entire RTB request
	sovrnReq := openrtb.BidRequest{
		ID:   sReq.ID,
		Imp:  sReq.Imp,
		Site: sReq.Site,
	}

	// add tag ids to impressions
	for i, unit := range bidder.AdUnits {
		var params sovrnParams
		err := json.Unmarshal(unit.Params, &params)
		if err != nil {
			return nil, err
		}
		sovrnReq.Imp[i].TagID = strconv.Itoa(params.TagId)
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

	httpReq, err := http.NewRequest("POST", s.URI, bytes.NewReader(reqJSON))
	httpReq.Header.Set("Content-Type", "application/json")
	userId := strings.TrimSpace(sReq.User.ID)
	if len(userId) > 0 {
		httpReq.AddCookie(&http.Cookie{Name: "ljt_reader", Value: userId})
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
func (a *SovrnAdapter) SkipNoCookies() bool {
	return false
}
func NewSovrnAdapter(config *HTTPAdapterConfig, endpoint string, usersyncURL string, externalURL string) *SovrnAdapter {
	a := NewHTTPAdapter(config)

	redirect_uri := fmt.Sprintf("%s/setuid?bidder=sovrn&uid=$UID", externalURL)
	//usersyncURL := "http://ap.lijit.dev:9080/userSync?"

	info := &pbs.UsersyncInfo{
		URL:         fmt.Sprintf("%sredir=%s", usersyncURL, url.QueryEscape(redirect_uri)),
		Type:        "redirect",
		SupportCORS: false,
	}

	return &SovrnAdapter{
		http:         a,
		URI:          endpoint,
		usersyncInfo: info,
	}
}
