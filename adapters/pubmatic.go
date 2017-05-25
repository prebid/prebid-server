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

type PubmaticAdapter struct {
	http         *HTTPAdapter
	URI          string
	usersyncInfo *pbs.UsersyncInfo
}

/* Name - export adapter name */
func (a *PubmaticAdapter) Name() string {
	return "Pubmatic"
}

// used for cookies and such
func (a *PubmaticAdapter) FamilyName() string {
	return "pubmatic"
}

func (a *PubmaticAdapter) GetUsersyncInfo() *pbs.UsersyncInfo {
	return a.usersyncInfo
}

type pubmaticParams struct {
	PublisherId string `json:"publisherId"`
	AdSlot      string `json:"adSlot"`
}

func (a *PubmaticAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	pbReq := makeOpenRTBGeneric(req, bidder, a.FamilyName())
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
	httpReq.AddCookie(&http.Cookie{
		Name:  "KADUSERCOOKIE",
		Value: req.GetUserID(a.FamilyName()),
	})

	if !req.IsDebug {
		debug = nil // make this nil so DefaultOpenRTBResponse can ignore it
	}

	return DefaultOpenRTBResponse(ctx, a.http, httpReq, bidder, debug)
}

func NewPubmaticAdapter(config *HTTPAdapterConfig, uri string, externalURL string) *PubmaticAdapter {
	a := NewHTTPAdapter(config)
	redirect_uri := fmt.Sprintf("%s/setuid?bidder=pubmatic&uid=", externalURL)
	usersyncURL := "//ads.pubmatic.com/AdServer/js/user_sync.html?predirect="

	return &PubmaticAdapter{
		http: a,
		URI:  uri,
		usersyncInfo: &pbs.UsersyncInfo{
			URL:         fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirect_uri)),
			Type:        "iframe",
			SupportCORS: false,
		},
	}
}
