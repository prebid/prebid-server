package endpoints

import (
	"net/http"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbsmetrics"
	"github.com/prebid/prebid-server/usersync"
)

func NewSetUIDEndpoint(cfg config.HostCookie, pbsanalytics analytics.PBSAnalyticsModule, metrics pbsmetrics.MetricsEngine) httprouter.Handle {
	cookieTTL := time.Duration(cfg.TTL) * 24 * time.Hour
	return httprouter.Handle(func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		so := analytics.SetUIDObject{
			Status: http.StatusOK,
			Errors: make([]error, 0),
		}

		defer pbsanalytics.LogSetUIDObject(&so)

		pc := usersync.ParsePBSCookieFromRequest(r, &cfg.OptOutCookie)
		if !pc.AllowSyncs() {
			w.WriteHeader(http.StatusUnauthorized)
			metrics.RecordUserIDSet(pbsmetrics.UserLabels{Action: pbsmetrics.RequestActionOptOut})
			so.Status = http.StatusUnauthorized
			return
		}

		query := getRawQueryMap(r.URL.RawQuery)
		bidder := query["bidder"]
		if bidder == "" {
			w.WriteHeader(http.StatusBadRequest)
			metrics.RecordUserIDSet(pbsmetrics.UserLabels{Action: pbsmetrics.RequestActionErr})
			so.Status = http.StatusBadRequest
			return
		}
		so.Bidder = bidder

		uid := query["uid"]
		so.UID = uid

		var err error = nil
		if uid == "" {
			pc.Unsync(bidder)
		} else {
			err = pc.TrySync(bidder, uid)
		}

		if err == nil {
			labels := pbsmetrics.UserLabels{
				Action: pbsmetrics.RequestActionSet,
				Bidder: openrtb_ext.BidderName(bidder),
			}
			metrics.RecordUserIDSet(labels)
			so.Success = true
		}

		pc.SetCookieOnResponse(w, cfg.Domain, cookieTTL)
	})
}

func getRawQueryMap(query string) map[string]string {
	m := make(map[string]string)
	for _, kv := range strings.SplitN(query, "&", -1) {
		if len(kv) == 0 {
			continue
		}
		pair := strings.SplitN(kv, "=", 2)
		if len(pair) == 2 {
			m[pair[0]] = pair[1]
		}
	}
	return m
}
