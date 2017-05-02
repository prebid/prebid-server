package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/prebid/openrtb"
	"github.com/prebid/prebid-server/pbs"
	"golang.org/x/net/context/ctxhttp"
	"io/ioutil"
	"net/http"
	"net/url"
)

type FacebookAdapter struct {
	http         *HTTPAdapter
	URI          string
	usersyncInfo *pbs.UsersyncInfo
	platformJSON openrtb.RawJSON
}

/* Name - export adapter name */
func (a *FacebookAdapter) Name() string {
	return "audienceNetwork"
}

// used for cookies and such
func (a *FacebookAdapter) FamilyName() string {
	return "audienceNetwork"
}

// Facebook likes to parallelize to minimize latency
func (a *FacebookAdapter) SplitAdUnits() bool {
	return true
}

func (a *FacebookAdapter) GetUsersyncInfo() *pbs.UsersyncInfo {
	return a.usersyncInfo
}

type facebookParams struct {
	AppId       int `json:"appId"`
	PlacementId int `json:"placementId"`
}

func (a *FacebookAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	if len(bidder.AdUnits) != 1 {
		return nil, fmt.Errorf("Facebook ad units not split properly")
	}

	fbReq := makeOpenRTBGeneric(req, bidder, a.FamilyName())
	fbReq.Ext = a.platformJSON

	unit := bidder.AdUnits[0]
	var params facebookParams
	err := json.Unmarshal(unit.Params, &params)
	if err != nil {
		return nil, err
	}
	if params.AppId == 0 {
		return nil, errors.New("Missing appId param")
	}
	fbReq.Site.Publisher = &openrtb.Publisher{ID: fmt.Sprintf("%d", params.AppId)}
	if params.PlacementId == 0 {
		return nil, errors.New("Missing placementId param")
	}
	fbReq.Imp[0].TagID = fmt.Sprintf("%d", params.PlacementId)

	reqJSON, err := json.Marshal(fbReq)
	if err != nil {
		return nil, err
	}

	if req.IsDebug {
		bidder.Debug.RequestURI = a.URI
		bidder.Debug.RequestBody = string(reqJSON)
	}

	httpReq, err := http.NewRequest("POST", a.URI, bytes.NewBuffer(reqJSON))
	httpReq.Header.Add("Content-Type", "application/json")
	httpReq.Header.Add("Accept", "application/json")

	anResp, err := ctxhttp.Do(ctx, a.http.Client, httpReq)
	if err != nil {
		return nil, err
	}

	if req.IsDebug {
		bidder.Debug.StatusCode = anResp.StatusCode
	}

	if anResp.StatusCode == 204 {
		return nil, nil
	}

	if anResp.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("HTTP status code %d", anResp.StatusCode))
	}

	defer anResp.Body.Close()
	body, err := ioutil.ReadAll(anResp.Body)
	if err != nil {
		return nil, err
	}

	if req.IsDebug {
		bidder.Debug.ResponseBody = string(body)
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
				return nil, errors.New(fmt.Sprintf("Unknown ad unit code '%s'", bid.ImpID))
			}

			pbid := pbs.PBSBid{
				BidID:       bidID,
				AdUnitCode:  bid.ImpID,
				BidderCode:  bidder.BidderCode,
				Price:       bid.Price / 100, // convert from cents to dollars
				Adm:         bid.AdM,
				Creative_id: bid.CrID,
				Width:       bid.W,
				Height:      bid.H,
				DealId:      bid.DealID,
			}
			bids = append(bids, &pbid)
		}
	}

	return bids, nil
}

func NewFacebookAdapter(config *HTTPAdapterConfig, partnerID string, externalURL string) *FacebookAdapter {
	a := NewHTTPAdapter(config)

	redirect_uri := fmt.Sprintf("%s/setuid?bidder=audienceNetwork&uid=$UID", externalURL)
	usersyncURL := fmt.Sprintf("https://www.facebook.com/audiencenetwork/idsync/?partner=%s&callback=", partnerID)
	info := &pbs.UsersyncInfo{
		URL:         fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirect_uri)),
		Type:        "redirect",
		SupportCORS: false,
	}

	return &FacebookAdapter{
		http:         a,
		URI:          "https://an.facebook.com/placementbid.ortb",
		usersyncInfo: info,
		platformJSON: openrtb.RawJSON(fmt.Sprintf("{\"platformid\": %s}", partnerID)),
	}
}
