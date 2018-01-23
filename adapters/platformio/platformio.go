package platformio

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/pbs"
	"golang.org/x/net/context/ctxhttp"
)

type PlatformioAdapter struct {
	http         *adapters.HTTPAdapter
	URI          string
	usersyncInfo *pbs.UsersyncInfo
}

func (a *PlatformioAdapter) Name() string {
	return "platformio"
}

func (a *PlatformioAdapter) FamilyName() string {
	return "platformio"
}

func (a *PlatformioAdapter) GetUsersyncInfo() *pbs.UsersyncInfo {
	return a.usersyncInfo
}

func (a *PlatformioAdapter) SkipNoCookies() bool {
	return false
}

// parameters for platformio adapter.
type PlatformioParams struct {
	PublisherId int     `json:"pubId"`
	TagId       int     `json:"placementId"`
	AdSize      string  `json:"size"`
	SiteId      int     `json:"siteId"`
	BidFloor    float64 `json:"bidFloor"`
}

func (a *PlatformioAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	mediaTypes := []pbs.MediaType{pbs.MEDIA_TYPE_BANNER, pbs.MEDIA_TYPE_VIDEO}
	pltfrmReq, err := adapters.MakeOpenRTBGeneric(req, bidder, a.FamilyName(), mediaTypes, true)

	if err != nil {
		return nil, err
	}

	for i, unit := range bidder.AdUnits {
		var params PlatformioParams
		err := json.Unmarshal(unit.Params, &params)
		if err != nil {
			return nil, err
		}
		if params.PublisherId == 0 {
			return nil, fmt.Errorf("Missing PublisherId param pubId")
		}
		if params.TagId == 0 {
			return nil, fmt.Errorf("Missing TagId param placementId")
		}
		if params.AdSize == "" {
			return nil, fmt.Errorf("Missing AdSize param size")
		}
		if params.SiteId == 0 {
			return nil, fmt.Errorf("Missing SiteId param siteId")
		}

		pltfrmReq.Imp[i].TagID = strconv.Itoa(params.TagId)
		pltfrmReq.Imp[i].BidFloor = params.BidFloor
		publisher := &openrtb.Publisher{ID: strconv.Itoa(params.PublisherId)}
		siteid := strconv.Itoa(params.SiteId)

		if pltfrmReq.Site != nil {
			siteCopy := *pltfrmReq.Site
			siteCopy.Publisher = publisher
			siteCopy.ID = siteid
			pltfrmReq.Site = &siteCopy
		} else {
			appCopy := *pltfrmReq.App
			appCopy.Publisher = publisher
			pltfrmReq.App = &appCopy
		}
		if pltfrmReq.Imp[i].Banner != nil {
			var size = strings.Split(strings.ToLower(params.AdSize), "x")
			if len(size) == 2 {
				width, err := strconv.Atoi(size[0])
				if err == nil {
					pltfrmReq.Imp[i].Banner.W = openrtb.Uint64Ptr(uint64(width))
				} else {
					return nil, fmt.Errorf("Invalid Width param %s", size[0])
				}
				height, err := strconv.Atoi(size[1])
				if err == nil {
					pltfrmReq.Imp[i].Banner.H = openrtb.Uint64Ptr(uint64(height))
				} else {
					return nil, fmt.Errorf("Invalid Height param %s", size[1])
				}
			} else {
				return nil, fmt.Errorf("Invalid AdSize param %s", params.AdSize)
			}
		}
	}
	reqJSON, err := json.Marshal(pltfrmReq)
	debug := &pbs.BidderDebug{
		RequestURI: a.URI,
	}

	if req.IsDebug {
		debug.RequestBody = string(reqJSON)
		bidder.Debug = append(bidder.Debug, debug)
	}

	httpReq, err := http.NewRequest("POST", a.URI, bytes.NewBuffer(reqJSON))
	httpReq.Header.Add("Content-Type", "application/json;charset=utf-8")
	httpReq.Header.Add("Accept", "application/json")

	ppResp, err := ctxhttp.Do(ctx, a.http.Client, httpReq)
	if err != nil {
		return nil, err
	}

	debug.StatusCode = ppResp.StatusCode

	if ppResp.StatusCode == 204 {
		return nil, nil
	}

	if ppResp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP status: %d", ppResp.StatusCode)
	}

	defer ppResp.Body.Close()
	body, err := ioutil.ReadAll(ppResp.Body)
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

	for _, sb := range bidResp.SeatBid {
		for _, bid := range sb.Bid {
			bidID := bidder.LookupBidID(bid.ImpID)
			if bidID == "" {
				return nil, errors.New(fmt.Sprintf("Unknown ad unit code '%s'", bid.ImpID))
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
			}
			bids = append(bids, &pbid)
		}
	}

	return bids, nil
}

func NewPlatformioAdapter(config *adapters.HTTPAdapterConfig, uri string, usersyncURL string, externalURL string) *PlatformioAdapter {
	a := adapters.NewHTTPAdapter(config)
	redirect_uri := fmt.Sprintf("%s/setuid?bidder=platformio&uid=%s", externalURL, "%%USER_ALIAS%%")

	info := &pbs.UsersyncInfo{
		URL:         fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirect_uri)),
		Type:        "redirect",
		SupportCORS: false,
	}

	return &PlatformioAdapter{
		http:         a,
		URI:          uri,
		usersyncInfo: info,
	}
}
