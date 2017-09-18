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
	"strconv"
	"strings"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/pbs"
	"golang.org/x/net/context/ctxhttp"
)

type PulsePointAdapter struct {
	http         *HTTPAdapter
	URI          string
	usersyncInfo *pbs.UsersyncInfo
}

// adapter name
func (a *PulsePointAdapter) Name() string {
	return "pulsepoint"
}

// used for cookies and such
func (a *PulsePointAdapter) FamilyName() string {
	return "pulsepoint"
}

func (a *PulsePointAdapter) GetUsersyncInfo() *pbs.UsersyncInfo {
	return a.usersyncInfo
}

func (a *PulsePointAdapter) SkipNoCookies() bool {
	return false
}

// parameters for pulsepoint adapter.
type PulsepointParams struct {
	PublisherId int    `json:"cp"`
	TagId       int    `json:"ct"`
	AdSize      string `json:"cf"`
}

func (a *PulsePointAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	mediaTypes := []pbs.MediaType{pbs.MEDIA_TYPE_BANNER, pbs.MEDIA_TYPE_VIDEO}
	ppReq, err := makeOpenRTBGeneric(req, bidder, a.FamilyName(), mediaTypes, true)

	if err != nil {
		return nil, err
	}

	for i, unit := range bidder.AdUnits {
		var params PulsepointParams
		err := json.Unmarshal(unit.Params, &params)
		if err != nil {
			return nil, err
		}
		if params.PublisherId == 0 {
			return nil, fmt.Errorf("Missing PublisherId param cp")
		}
		if params.TagId == 0 {
			return nil, fmt.Errorf("Missing TagId param ct")
		}
		if params.AdSize == "" {
			return nil, fmt.Errorf("Missing AdSize param cf")
		}
		ppReq.Imp[i].TagID = strconv.Itoa(params.TagId)
		publisher := &openrtb.Publisher{ID: strconv.Itoa(params.PublisherId)}
		if ppReq.Site != nil {
			ppReq.Site.Publisher = publisher
		} else {
			ppReq.App.Publisher = publisher
		}
		if ppReq.Imp[i].Banner != nil {
			var size = strings.Split(strings.ToLower(params.AdSize), "x")
			if len(size) == 2 {
				width, err := strconv.Atoi(size[0])
				if err == nil {
					ppReq.Imp[i].Banner.W = uint64(width)
				} else {
					return nil, fmt.Errorf("Invalid Width param %s", size[0])
				}
				height, err := strconv.Atoi(size[1])
				if err == nil {
					ppReq.Imp[i].Banner.H = uint64(height)
				} else {
					return nil, fmt.Errorf("Invalid Height param %s", size[1])
				}
			} else {
				return nil, fmt.Errorf("Invalid AdSize param %s", params.AdSize)
			}
		}
	}
	reqJSON, err := json.Marshal(ppReq)
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

func NewPulsePointAdapter(config *HTTPAdapterConfig, uri string, externalURL string) *PulsePointAdapter {
	a := NewHTTPAdapter(config)
	redirect_uri := fmt.Sprintf("%s/setuid?bidder=pulsepoint&uid=%s", externalURL, "%%VGUID%%")
	usersyncURL := "//bh.contextweb.com/rtset?pid=561205&ev=1&rurl="

	info := &pbs.UsersyncInfo{
		URL:         fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirect_uri)),
		Type:        "redirect",
		SupportCORS: false,
	}

	return &PulsePointAdapter{
		http:         a,
		URI:          uri,
		usersyncInfo: info,
	}
}
