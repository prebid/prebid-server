package facebook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

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
	adapters.Init("audienceNetwork", a)
}

func NewAdapter() *Adapter {
	return &Adapter{
		URI:  "https://an.facebook.com/placementbid.ortb",
		http: pbs.NewHTTPAdapter(pbs.DefaultHTTPAdapterConfig),
	}
}

type Adapter struct {
	http         *pbs.HTTPAdapter
	URI          string
	usersyncInfo *pbs.UsersyncInfo
	platformJSON openrtb.RawJSON
}

// Use will set a shared Use(http *pbs.HTTPAdapter) (optional)
func (a *Adapter) Use(http *pbs.HTTPAdapter) {
	a.http = http
}

// Configure is required. We require both an ExternalURl and *config.Adapter (so we can configre the PlatformID and UserSync URL)
func (a *Adapter) Configure(externalURL string, config *config.Adapter) {
	a.usersyncInfo = &pbs.UsersyncInfo{
		URL:         config.UserSyncURL,
		Type:        "redirect",
		SupportCORS: false,
	}
	a.platformJSON = openrtb.RawJSON(fmt.Sprintf("{\"platformid\": %s}", config.PlatformID))
}

/* Name - export adapter name */
func (a *Adapter) Name() string {
	return "facebook"
}

// used for cookies and such
func (a *Adapter) FamilyName() string {
	return "audienceNetwork"
}

// Facebook likes to parallelize to minimize latency
func (a *Adapter) SplitAdUnits() bool {
	return true
}

func (a *Adapter) GetUsersyncInfo() *pbs.UsersyncInfo {
	return a.usersyncInfo
}

type facebookParams struct {
	PlacementId string `json:"placementId"`
}

type fbResult struct {
	statusCode   int
	responseBody string
	bid          *pbs.PBSBid
	Error        error
}

func (a *Adapter) CallOne(ctx context.Context, req *pbs.PBSRequest, reqJSON bytes.Buffer) (result fbResult, err error) {
	httpReq, err := http.NewRequest("POST", a.URI, &reqJSON)
	if err != nil {
		return
	}
	httpReq.Header.Add("Content-Type", "application/json")
	httpReq.Header.Add("Accept", "application/json")

	anResp, err := ctxhttp.Do(ctx, a.http.Client, httpReq)
	if err != nil {
		return
	}

	result.statusCode = anResp.StatusCode

	defer anResp.Body.Close()
	body, err := ioutil.ReadAll(anResp.Body)
	if err != nil {
		return
	}
	result.responseBody = string(body)

	if anResp.StatusCode != 200 {
		err = fmt.Errorf("HTTP status %d; body: %s", anResp.StatusCode, result.responseBody)
		return
	}

	var bidResp openrtb.BidResponse
	err = json.Unmarshal(body, &bidResp)
	if err != nil {
		return
	}
	if len(bidResp.SeatBid) == 0 {
		return
	}
	if len(bidResp.SeatBid[0].Bid) == 0 {
		return
	}
	bid := bidResp.SeatBid[0].Bid[0]

	result.bid = &pbs.PBSBid{
		AdUnitCode: bid.ImpID,
		Price:      bid.Price / 100, // convert from cents to dollars
		Adm:        bid.AdM,
		Width:      300, // hard code as it's all FB supports
		Height:     250, // hard code as it's all FB supports
	}
	return
}

func (a *Adapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	requests := make([]bytes.Buffer, len(bidder.AdUnits))
	for i, unit := range bidder.AdUnits {
		fbReq := openrtb_util.MakeOpenRTBGeneric(req, bidder, a.FamilyName())
		fbReq.Ext = a.platformJSON

		// only grab this ad unit
		fbReq.Imp = fbReq.Imp[i : i+1]

		var params facebookParams
		err := json.Unmarshal(unit.Params, &params)
		if err != nil {
			return nil, err
		}
		if params.PlacementId == "" {
			return nil, errors.New("Missing placementId param")
		}
		s := strings.Split(params.PlacementId, "_")
		if len(s) != 2 {
			return nil, fmt.Errorf("Invalid placementId param '%s'", params.PlacementId)
		}
		if fbReq.Site != nil {
			fbReq.Site.Publisher = &openrtb.Publisher{ID: s[0]}
		}
		if fbReq.App != nil {
			fbReq.App.Publisher = &openrtb.Publisher{ID: s[0]}
		}
		fbReq.Imp[0].TagID = params.PlacementId

		err = json.NewEncoder(&requests[i]).Encode(fbReq)
		if err != nil {
			return nil, err
		}
	}

	ch := make(chan fbResult)
	for i, _ := range bidder.AdUnits {
		go func(bidder *pbs.PBSBidder, reqJSON bytes.Buffer) {
			result, err := a.CallOne(ctx, req, reqJSON)
			result.Error = err
			if result.bid != nil {
				result.bid.BidderCode = bidder.BidderCode
				result.bid.BidID = bidder.LookupBidID(result.bid.AdUnitCode)
				if result.bid.BidID == "" {
					result.Error = fmt.Errorf("Unknown ad unit code '%s'", result.bid.AdUnitCode)
					result.bid = nil
				}
			}
			ch <- result
		}(bidder, requests[i])
	}

	var err error

	bids := make(pbs.PBSBidSlice, 0)
	for i := 0; i < len(bidder.AdUnits); i++ {
		result := <-ch
		if result.bid != nil {
			bids = append(bids, result.bid)
		}
		if req.IsDebug {
			debug := &pbs.BidderDebug{
				RequestURI:   a.URI,
				RequestBody:  requests[i].String(),
				StatusCode:   result.statusCode,
				ResponseBody: result.responseBody,
			}
			bidder.Debug = append(bidder.Debug, debug)
		}
		if result.Error != nil {
			err = result.Error
		}
	}

	if len(bids) == 0 {
		return nil, err
	}
	return bids, nil
}
