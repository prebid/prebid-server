package appnexus

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"golang.org/x/net/context/ctxhttp"

	"github.com/prebid/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/openrtb_util"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/pbs"
)

// init will register the Adapter with our global exchanges
func init() {
	var a = NewAdapter()
	adapters.Init("appnexus", a)
}

func NewAdapter() *Adapter {
	return &Adapter{
		URI:  "http://ib.adnxs.com/openrtb2",
		http: pbs.NewHTTPAdapter(pbs.DefaultHTTPAdapterConfig),
	}
}

// Adapter
type Adapter struct {
	http         *pbs.HTTPAdapter
	URI          string
	usersyncInfo *pbs.UsersyncInfo
}

// Use will set a shared HTTPAdapter (optional)
func (a *Adapter) Use(http *pbs.HTTPAdapter) {
	a.http = http
}

// Configure is required. It accepts an external url (required) and optional *config.Adapter
// After Configure is run, the adapter will be registered as an active PBS exchange
func (a *Adapter) Configure(externalURL string, config *config.Adapter) {
	redirect_uri := fmt.Sprintf("%s/setuid?bidder=adnxs&uid=$UID", externalURL)
	usersyncURL := "//ib.adnxs.com/getuid?"

	a.usersyncInfo = &pbs.UsersyncInfo{
		URL:         fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirect_uri)),
		Type:        "redirect",
		SupportCORS: false,
	}

	// if no configs are provided then we'll use the default values provided in init()
	if config == nil {
		return
	}
	return
}

// Name needs to be unique per adapter
func (a *Adapter) Name() string {
	return "AppNexus"
}

// used for cookies and such
func (a *Adapter) FamilyName() string {
	return "adnxs"
}

func (a *Adapter) GetUsersyncInfo() *pbs.UsersyncInfo {
	return a.usersyncInfo
}

type appnexusParams struct {
	PlacementId int    `json:"placementId"`
	InvCode     string `json:"invCode"`
	Member      string `json:"member"`
}

type appnexusImpExtAppnexus struct {
	PlacementID int `json:"placement_id"`
}

type appnexusImpExt struct {
	Appnexus appnexusImpExtAppnexus `json:"appnexus"`
}

func (a *Adapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	anReq := openrtb_util.MakeOpenRTBGeneric(req, bidder, a.FamilyName())
	for i, unit := range bidder.AdUnits {
		var params appnexusParams
		err := json.Unmarshal(unit.Params, &params)
		if err != nil {
			return nil, err
		}

		if params.PlacementId == 0 {
			return nil, errors.New("Missing placementId param")
		}

		impExt := appnexusImpExt{Appnexus: appnexusImpExtAppnexus{PlacementID: params.PlacementId}}
		anReq.Imp[i].Ext, err = json.Marshal(&impExt)
		// TODO: support member + invCode
	}

	reqJSON, err := json.Marshal(anReq)
	if err != nil {
		return nil, err
	}

	debug := &pbs.BidderDebug{
		RequestURI: a.URI,
	}

	if req.IsDebug {
		debug.RequestBody = string(reqJSON)
		bidder.Debug = append(bidder.Debug, debug)
	}

	httpReq, err := http.NewRequest("POST", a.URI, bytes.NewBuffer(reqJSON))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Add("Content-Type", "application/json;charset=utf-8")
	httpReq.Header.Add("Accept", "application/json")

	anResp, err := ctxhttp.Do(ctx, a.http.Client, httpReq)
	if err != nil {
		return nil, err
	}

	debug.StatusCode = anResp.StatusCode

	if anResp.StatusCode == 204 {
		return nil, nil
	}

	if anResp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP status code %d", anResp.StatusCode)
	}

	defer anResp.Body.Close()
	body, err := ioutil.ReadAll(anResp.Body)
	if err != nil {
		return nil, err
	}

	if req.IsDebug {
		debug.ResponseBody = string(body)
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
			bids = append(bids, &pbs.PBSBid{
				BidID:       bidID,
				AdUnitCode:  bid.ImpID,
				BidderCode:  bidder.BidderCode,
				Price:       bid.Price,
				Adm:         bid.AdM,
				Creative_id: bid.CrID,
				Width:       bid.W,
				Height:      bid.H,
				DealId:      bid.DealID,
			})
		}
	}

	return bids, nil
}
