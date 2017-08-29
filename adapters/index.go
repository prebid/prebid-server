package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/prebid/prebid-server/pbs"

	"golang.org/x/net/context/ctxhttp"

	"github.com/prebid/openrtb"
)

type IndexAdapter struct {
	http         *HTTPAdapter
	URI          string
	usersyncInfo *pbs.UsersyncInfo
}

/* Name - export adapter name */
func (a *IndexAdapter) Name() string {
	return "indexExchange"
}

// used for cookies and such
func (a *IndexAdapter) FamilyName() string {
	return "indexExchange"
}

func (a *IndexAdapter) GetUsersyncInfo() *pbs.UsersyncInfo {
	return a.usersyncInfo
}

func (a *IndexAdapter) SkipNoCookies() bool {
	return false
}

type indexParams struct {
	SiteID int `json:"siteID"`
}

func (a *IndexAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	if req.App != nil {
		return nil, fmt.Errorf("Index doesn't support apps")
	}
	mediaTypes := []pbs.MediaType{pbs.MEDIA_TYPE_BANNER, pbs.MEDIA_TYPE_VIDEO}
	indexReq, err := makeOpenRTBGeneric(req, bidder, a.FamilyName(), mediaTypes, true)

	if err != nil {
		return pbs.PBSBidSlice{}, err
	}

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

			pbid := pbs.PBSBid{
				BidID:       bidID,
				AdUnitCode:  bidder.AdUnits[i].Code, // todo: check this
				BidderCode:  bidder.BidderCode,
				Price:       bid.Price,
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

func NewIndexAdapter(config *HTTPAdapterConfig, uri string, externalURL string) *IndexAdapter {
	a := NewHTTPAdapter(config)
	redirect_uri := fmt.Sprintf("%s/setuid?bidder=indexExchange&uid=__UID__", externalURL)
	usersyncURI := "//ssum-sec.casalemedia.com/usermatchredir?s=184932&cb="

	info := &pbs.UsersyncInfo{
		URL:         fmt.Sprintf("%s%s", usersyncURI, url.QueryEscape(redirect_uri)),
		Type:        "redirect",
		SupportCORS: false,
	}

	return &IndexAdapter{
		http:         a,
		URI:          uri,
		usersyncInfo: info,
	}
}
