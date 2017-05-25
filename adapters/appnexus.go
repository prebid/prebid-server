package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/prebid/prebid-server/pbs"
)

type AppNexusAdapter struct {
	http         *HTTPAdapter
	URI          string
	usersyncInfo *pbs.UsersyncInfo
}

/* Name - export adapter name */
func (a *AppNexusAdapter) Name() string {
	return "AppNexus"
}

// used for cookies and such
func (a *AppNexusAdapter) FamilyName() string {
	return "adnxs"
}

func (a *AppNexusAdapter) GetUsersyncInfo() *pbs.UsersyncInfo {
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

func (a *AppNexusAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	anReq := makeOpenRTBGeneric(req, bidder, a.FamilyName())
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

	if !req.IsDebug {
		debug = nil // make this nil so DefaultOpenRTBResponse can ignore it
	}

	return DefaultOpenRTBResponse(ctx, a.http, httpReq, bidder, debug)
}

func NewAppNexusAdapter(config *HTTPAdapterConfig, externalURL string) *AppNexusAdapter {
	a := NewHTTPAdapter(config)

	redirect_uri := fmt.Sprintf("%s/setuid?bidder=adnxs&uid=$UID", externalURL)
	usersyncURL := "//ib.adnxs.com/getuid?"

	return &AppNexusAdapter{
		http: a,
		URI:  "http://ib.adnxs.com/openrtb2",
		usersyncInfo: &pbs.UsersyncInfo{
			URL:         fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirect_uri)),
			Type:        "redirect",
			SupportCORS: false,
		},
	}
}
