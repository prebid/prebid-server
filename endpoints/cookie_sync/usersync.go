package cookie_sync

import (
	"encoding/json"
	"net/http"

	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/pbs"
	syncers "github.com/prebid/prebid-server/usersync"
	metrics "github.com/rcrowley/go-metrics"
)

// NewEndpoint implements /cookie_sync
func NewEndpoint(syncers map[openrtb_ext.BidderName]syncers.Usersyncer, optOutCookie *config.Cookie, metric metrics.Meter) httprouter.Handle {
	return (&cookieSyncDeps{syncers, optOutCookie, metric}).CookieSync
}

type cookieSyncDeps struct {
	syncs        map[openrtb_ext.BidderName]syncers.Usersyncer
	optOutCookie *config.Cookie
	metric       metrics.Meter
}

func (deps *cookieSyncDeps) CookieSync(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	deps.metric.Mark(1)
	userSyncCookie := pbs.ParsePBSCookieFromRequest(r, deps.optOutCookie)
	if !userSyncCookie.AllowSyncs() {
		http.Error(w, "User has opted out", http.StatusUnauthorized)
		return
	}

	defer r.Body.Close()

	csReq := &cookieSyncRequest{}
	err := json.NewDecoder(r.Body).Decode(&csReq)
	if err != nil {
		if glog.V(2) {
			glog.Infof("Failed to parse /cookie_sync request body: %v", err)
		}
		http.Error(w, "JSON parse failed", http.StatusBadRequest)
		return
	}

	csResp := cookieSyncResponse{
		UUID:         csReq.UUID,
		BidderStatus: make([]*pbs.PBSBidder, 0, len(csReq.Bidders)),
	}

	if userSyncCookie.LiveSyncCount() == 0 {
		csResp.Status = "no_cookie"
	} else {
		csResp.Status = "ok"
	}

	for _, bidder := range csReq.Bidders {
		if syncer, ok := deps.syncs[openrtb_ext.BidderName(bidder)]; ok {
			if !userSyncCookie.HasLiveSync(syncer.FamilyName()) {
				b := pbs.PBSBidder{
					BidderCode:   bidder,
					NoCookie:     true,
					UsersyncInfo: syncer.GetUsersyncInfo(),
				}
				csResp.BidderStatus = append(csResp.BidderStatus, &b)
			}
		}
	}

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	//enc.SetIndent("", "  ")
	enc.Encode(csResp)
}

type cookieSyncRequest struct {
	UUID    string   `json:"uuid"`
	Bidders []string `json:"bidders"`
}

type cookieSyncResponse struct {
	UUID         string           `json:"uuid"`
	Status       string           `json:"status"`
	BidderStatus []*pbs.PBSBidder `json:"bidder_status"`
}
