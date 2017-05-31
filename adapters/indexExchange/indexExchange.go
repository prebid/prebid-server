package indexExchange

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
	adapters.Init("indexExchange", a)
}

func NewAdapter() *Adapter {
	return &Adapter{
		URI:  "http://ssp-sandbox.casalemedia.com/bidder?p=184932",
		http: pbs.NewHTTPAdapter(pbs.DefaultHTTPAdapterConfig),
	}
}

type Adapter struct {
	http         *pbs.HTTPAdapter
	URI          string
	usersyncInfo *pbs.UsersyncInfo
}

// Use will set a shared Use(http *pbs.HTTPAdapter) (optional)
func (a *Adapter) Use(http *pbs.HTTPAdapter) {
	a.http = http
}

// Configure is required. It accepts an external url (required) and optional *config.Adapter
// After Configure is run, the adapter will be registered as an active PBS exchange
func (a *Adapter) Configure(externalURL string, config *config.Adapter) {
	redirect_uri := fmt.Sprintf("%s/setuid?bidder=indexExchange&uid=__UID__", externalURL)
	usersyncURL := "//ssum-sec.casalemedia.com/usermatchredir?s=184932&cb="

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

/* Name - export adapter name */
func (a *Adapter) Name() string {
	return "indexExchange"
}

// used for cookies and such
func (a *Adapter) FamilyName() string {
	return "indexExchange"
}

func (a *Adapter) GetUsersyncInfo() *pbs.UsersyncInfo {
	return a.usersyncInfo
}

type indexParams struct {
	SiteID int `json:"siteID"`
}

func (a *Adapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	if req.App != nil {
		return nil, fmt.Errorf("Index doesn't support apps")
	}
	indexReq := openrtb_util.MakeOpenRTBGeneric(req, bidder, a.FamilyName())
	for i, unit := range bidder.AdUnits {
		var params indexParams
		err := json.Unmarshal(unit.Params, &params)
		if err != nil {
			return nil, fmt.Errorf("unmarshal params '%s' failed: %v", unit.Params, err)
		}
		if params.SiteID == 0 {
			return nil, errors.New("Missing siteID param")
		}

		indexReq.Imp[i].TagID = unit.Code
		// Index spec says "adunit path representing ad server inventory" but we don't have this
		// ext is DFP div ID and KV pairs if avail
		//indexReq.Imp[i].Ext = openrtb.RawJSON("{}")
		indexReq.Site.Publisher = &openrtb.Publisher{ID: fmt.Sprintf("%d", params.SiteID)}
	}
	// spec also asks for publisher id if set
	// ext object on request for prefetch

	j, _ := json.Marshal(indexReq)

	debug := &pbs.BidderDebug{
		RequestURI: a.URI,
	}

	if req.IsDebug {
		debug.RequestBody = string(j)
		bidder.Debug = append(bidder.Debug, debug)
	}

	httpReq, err := http.NewRequest("POST", a.URI, bytes.NewBuffer(j))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Add("Content-Type", "application/json;charset=utf-8")
	httpReq.Header.Add("Accept", "application/json")

	ixResp, err := ctxhttp.Do(ctx, a.http.Client, httpReq)
	if err != nil {
		return nil, err
	}

	debug.StatusCode = ixResp.StatusCode

	if ixResp.StatusCode == 204 {
		return nil, nil
	}

	if ixResp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP status: %d", ixResp.StatusCode)
	}

	defer ixResp.Body.Close()
	body, err := ioutil.ReadAll(ixResp.Body)
	if err != nil {
		return nil, err
	}

	if req.IsDebug {
		debug.ResponseBody = string(body)
	}

	var bidResp openrtb.BidResponse
	err = json.Unmarshal(body, &bidResp)
	if err != nil {
		return nil, fmt.Errorf("Error parsing response: %v", err)
	}

	bids := make(pbs.PBSBidSlice, 0)

	numBids := 0
	for _, sb := range bidResp.SeatBid {
		for i, bid := range sb.Bid {
			numBids++

			bidID := bidder.LookupBidID(bid.ImpID)
			if bidID == "" {
				return nil, fmt.Errorf("Unknown ad unit code '%s'", bid.ImpID)
			}

			bids = append(bids, &pbs.PBSBid{
				BidID:       bidID,
				AdUnitCode:  bidder.AdUnits[i].Code, // todo: check this
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
