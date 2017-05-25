package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/prebid/openrtb"
	"github.com/prebid/prebid-server/pbs"
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

type indexParams struct {
	SiteID int `json:"siteID"`
}

func (a *IndexAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	if req.App != nil {
		return nil, fmt.Errorf("Index doesn't support apps")
	}
	indexReq := makeOpenRTBGeneric(req, bidder, a.FamilyName())
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

	if !req.IsDebug {
		debug = nil // make this nil so DefaultOpenRTBResponse can ignore it
	}

	return DefaultOpenRTBResponse(ctx, a.http, httpReq, bidder, debug)
}

func NewIndexAdapter(config *HTTPAdapterConfig, externalURL string) *IndexAdapter {
	a := NewHTTPAdapter(config)
	redirect_uri := fmt.Sprintf("%s/setuid?bidder=indexExchange&uid=__UID__", externalURL)
	usersyncURI := "//ssum-sec.casalemedia.com/usermatchredir?s=184932&cb="

	return &IndexAdapter{
		http: a,
		URI:  "http://ssp-sandbox.casalemedia.com/bidder?p=184932",
		usersyncInfo: &pbs.UsersyncInfo{
			URL:         fmt.Sprintf("%s%s", usersyncURI, url.QueryEscape(redirect_uri)),
			Type:        "redirect",
			SupportCORS: false,
		},
	}
}
