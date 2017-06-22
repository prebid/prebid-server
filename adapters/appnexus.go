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
	"strings"

	"github.com/prebid/prebid-server/pbs"

	"golang.org/x/net/context/ctxhttp"

	"github.com/prebid/openrtb"
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

type KeyVal struct {
	Key    string   `json:"key"`
	Values []string `json:"value"`
}

type appnexusParams struct {
	PlacementId       int      `json:"placementId"`
	InvCode           string   `json:"invCode"`
	Member            int      `json:"member"`
	Keywords          []KeyVal `json:"keywords"`
	TrafficSourceCode string   `json:"trafficSourceCode"`
	Reserve           float64  `json:"reserve"`
	Position          int      `json:"position"`
}

type appnexusImpExtAppnexus struct {
	PlacementID       int    `json:"placement_id"`
	Keywords          string `json:"keywords"`
	TrafficSourceCode string `json:"traffic_source_code"`
}

type appnexusImpExt struct {
	Appnexus appnexusImpExtAppnexus `json:"appnexus"`
}

func (a *AppNexusAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	anReq := makeOpenRTBGeneric(req, bidder, a.FamilyName())

	uri := a.URI
	for i, unit := range bidder.AdUnits {
		var params appnexusParams
		err := json.Unmarshal(unit.Params, &params)
		if err != nil {
			return nil, err
		}

		if params.PlacementId == 0 && (params.InvCode == "" || params.Member == 0) {
			return nil, errors.New("No placement or member+invcode provided")
		}

		if params.InvCode != "" {
			anReq.Imp[i].TagID = params.InvCode
			if params.Member != 0 {
				// this assumes that the same member ID is used across all tags, which should be the case
				uri = fmt.Sprintf("%s?member=%d", a.URI, params.Member)
			}

		}
		if params.Reserve > 0 {
			anReq.Imp[i].BidFloor = params.Reserve // TODO: we need to factor in currency here if non-USD
		}
		if anReq.Imp[i].Banner != nil && params.Position > 0 {
			if params.Position == 1 {
				anReq.Imp[i].Banner.Pos = 1
			} else if params.Position == 2 {
				anReq.Imp[i].Banner.Pos = 3 // openRTB spec
			}
		}

		var buffer bytes.Buffer
		for i, kv := range params.Keywords {
			if i > 0 {
				buffer.WriteString(",")
			}
			buffer.WriteString(kv.Key)
			buffer.WriteString("=")
			buffer.WriteString(strings.Join(kv.Values, ","))
		}

		impExt := appnexusImpExt{Appnexus: appnexusImpExtAppnexus{
			PlacementID:       params.PlacementId,
			TrafficSourceCode: params.TrafficSourceCode,
			Keywords:          buffer.String(),
		}}
		anReq.Imp[i].Ext, err = json.Marshal(&impExt)
	}

	reqJSON, err := json.Marshal(anReq)
	if err != nil {
		return nil, err
	}

	debug := &pbs.BidderDebug{
		RequestURI: uri,
	}

	if req.IsDebug {
		debug.RequestBody = string(reqJSON)
		bidder.Debug = append(bidder.Debug, debug)
	}

	httpReq, err := http.NewRequest("POST", uri, bytes.NewBuffer(reqJSON))
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

	defer anResp.Body.Close()
	body, err := ioutil.ReadAll(anResp.Body)
	if err != nil {
		return nil, err
	}
	responseBody := string(body)

	if anResp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP status %d; body: %s", anResp.StatusCode, responseBody)
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

			pbid := pbs.PBSBid{
				BidID:       bidID,
				AdUnitCode:  bid.ImpID,
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

func NewAppNexusAdapter(config *HTTPAdapterConfig, externalURL string) *AppNexusAdapter {
	a := NewHTTPAdapter(config)

	redirect_uri := fmt.Sprintf("%s/setuid?bidder=adnxs&uid=$UID", externalURL)
	usersyncURL := "//ib.adnxs.com/getuid?"

	info := &pbs.UsersyncInfo{
		URL:         fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirect_uri)),
		Type:        "redirect",
		SupportCORS: false,
	}

	return &AppNexusAdapter{
		http:         a,
		URI:          "http://ib.adnxs.com/openrtb2",
		usersyncInfo: info,
	}
}
