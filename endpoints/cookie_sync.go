package endpoints

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
	"github.com/prebid/prebid-server/pbsmetrics"
	"github.com/prebid/prebid-server/usersync"
)

func NewCookieSyncEndpoint(syncers map[openrtb_ext.BidderName]usersync.Usersyncer, optOutCookie *config.Cookie, metrics pbsmetrics.MetricsEngine, pbsAnalytics analytics.PBSAnalyticsModule) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		//CookieSyncObject makes a log of requests and responses to  /cookie_sync endpoint
		co := analytics.CookieSyncObject{
			Status:       http.StatusOK,
			Errors:       make([]error, 0),
			BidderStatus: make([]*usersync.CookieSyncBidders, 0),
		}

		defer pbsAnalytics.LogCookieSyncObject(&co)

		metrics.RecordCookieSync(pbsmetrics.Labels{})
		userSyncCookie := pbs.ParsePBSCookieFromRequest(r, optOutCookie)
		if !userSyncCookie.AllowSyncs() {
			http.Error(w, "User has opted out", http.StatusUnauthorized)
			co.Status = http.StatusUnauthorized
			co.Errors = append(co.Errors, fmt.Errorf("user has opted out"))
			return
		}

		defer r.Body.Close()

		csReq := &cookieSyncRequest{}
		csReqRaw := map[string]json.RawMessage{}
		err := json.NewDecoder(r.Body).Decode(&csReqRaw)
		if err != nil {
			if glog.V(2) {
				glog.Infof("Failed to parse /cookie_sync request body: %v", err)
			}
			co.Status = http.StatusBadRequest
			co.Errors = append(co.Errors, fmt.Errorf("JSON parse failed"))
			http.Error(w, "JSON parse failed", http.StatusBadRequest)
			return
		}
		biddersOmitted := true
		if biddersRaw, ok := csReqRaw["bidders"]; ok {
			biddersOmitted = false
			err := json.Unmarshal(biddersRaw, &csReq.Bidders)
			if err != nil {
				if glog.V(2) {
					glog.Infof("Failed to parse /cookie_sync request body (bidders list): %v", err)
				}
				co.Status = http.StatusBadRequest
				co.Errors = append(co.Errors, fmt.Errorf("JSON parse failed (bidders"))
				http.Error(w, "JSON parse failed (bidders)", http.StatusBadRequest)
				return
			}
		}

		csResp := cookieSyncResponse{
			BidderStatus: make([]*usersync.CookieSyncBidders, 0, len(csReq.Bidders)),
		}

		if userSyncCookie.LiveSyncCount() == 0 {
			csResp.Status = "no_cookie"
		} else {
			csResp.Status = "ok"
		}

		// If at the end (After possibly reading stored bidder lists) there still are no bidders,
		// and "bidders" is not found in the JSON, sync all bidders
		if len(csReq.Bidders) == 0 && biddersOmitted {
			for bidder := range syncers {
				csReq.Bidders = append(csReq.Bidders, string(bidder))
			}
		}

		for _, bidder := range csReq.Bidders {
			if syncer, ok := syncers[openrtb_ext.BidderName(bidder)]; ok {
				if !userSyncCookie.HasLiveSync(syncer.FamilyName()) {
					b := usersync.CookieSyncBidders{
						BidderCode:   bidder,
						NoCookie:     true,
						UsersyncInfo: syncer.GetUsersyncInfo(),
					}
					csResp.BidderStatus = append(csResp.BidderStatus, &b)
				}
			}
		}

		if len(csResp.BidderStatus) > 0 {
			co.BidderStatus = append(co.BidderStatus, csResp.BidderStatus...)
		}

		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		//enc.SetIndent("", "  ")
		enc.Encode(csResp)
	}
}

type cookieSyncRequest struct {
	Bidders []string `json:"bidders"`
}

type cookieSyncResponse struct {
	Status       string                        `json:"status"`
	BidderStatus []*usersync.CookieSyncBidders `json:"bidder_status"`
}
