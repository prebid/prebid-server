package pubmatic

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
	adapters.Init("pubmatic", a)
}

func NewAdapter() *Adapter {
	return &Adapter{
		URI:  "http://openbid-useast.pubmatic.com/translator?",
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
	redirect_uri := fmt.Sprintf("%s/setuid?bidder=pubmatic&uid=", externalURL)
	usersyncURL := "//ads.pubmatic.com/AdServer/js/user_sync.html?predirect="

	a.usersyncInfo = &pbs.UsersyncInfo{
		URL:         fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirect_uri)),
		Type:        "iframe",
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
	return "Pubmatic"
}

// used for cookies and such
func (a *Adapter) FamilyName() string {
	return "pubmatic"
}

func (a *Adapter) GetUsersyncInfo() *pbs.UsersyncInfo {
	return a.usersyncInfo
}

type pubmaticParams struct {
	PublisherId string `json:"publisherId"`
	AdSlot      string `json:"adSlot"`
}

func (a *Adapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	pbReq := openrtb_util.MakeOpenRTBGeneric(req, bidder, a.FamilyName())
	for i, unit := range bidder.AdUnits {
		var params pubmaticParams
		err := json.Unmarshal(unit.Params, &params)
		if err != nil {
			return nil, err
		}
		if params.PublisherId == "" {
			return nil, errors.New("Missing publisherId param")
		}
		if params.AdSlot == "" {
			return nil, errors.New("Missing adSlot param")
		}
		pbReq.Imp[i].Banner.Format = nil // pubmatic doesn't support
		pbReq.Imp[i].TagID = params.AdSlot
		if pbReq.Site != nil {
			pbReq.Site.Publisher = &openrtb.Publisher{ID: params.PublisherId}
		}
		if pbReq.App != nil {
			pbReq.App.Publisher = &openrtb.Publisher{ID: params.PublisherId}
		}
	}

	reqJSON, err := json.Marshal(pbReq)

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
	httpReq.AddCookie(&http.Cookie{
		Name:  "KADUSERCOOKIE",
		Value: req.GetUserID(a.FamilyName()),
	})

	pbResp, err := ctxhttp.Do(ctx, a.http.Client, httpReq)
	if err != nil {
		return nil, err
	}

	debug.StatusCode = pbResp.StatusCode

	if pbResp.StatusCode == 204 {
		return nil, nil
	}

	if pbResp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP status: %d", pbResp.StatusCode)
	}

	defer pbResp.Body.Close()
	body, err := ioutil.ReadAll(pbResp.Body)
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
				return nil, errors.New(fmt.Sprintf("Unknown ad unit code '%s'", bid.ImpID))
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
