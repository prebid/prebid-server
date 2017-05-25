package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/prebid/openrtb"
	"github.com/prebid/prebid-server/pbs"
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

// parameters for pulsepoint adapter.
type PulsepointParams struct {
	PublisherId int    `json:"cp"`
	TagId       int    `json:"ct"`
	AdSize      string `json:"cf"`
}

func (a *PulsePointAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	ppReq := makeOpenRTBGeneric(req, bidder, a.FamilyName())
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

func NewPulsePointAdapter(config *HTTPAdapterConfig, uri string, externalURL string) *PulsePointAdapter {
	a := NewHTTPAdapter(config)
	redirect_uri := fmt.Sprintf("%s/setuid?bidder=pulsepoint&uid=%s", externalURL, "%%VGUID%%")
	usersyncURL := "//bh.contextweb.com/rtset?pid=561205&ev=1&rurl="

	return &PulsePointAdapter{
		http: a,
		URI:  uri,
		usersyncInfo: &pbs.UsersyncInfo{
			URL:         fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirect_uri)),
			Type:        "redirect",
			SupportCORS: false,
		},
	}
}
