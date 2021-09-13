package endpoints

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"

	"github.com/buger/jsonparser"
	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/gdpr"
	"github.com/prebid/prebid-server/metrics"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/ccpa"
	gdprPrivacy "github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/prebid/prebid-server/usersync"
)

func NewCookieSyncEndpoint(
	syncers map[openrtb_ext.BidderName]usersync.Usersyncer,
	cfg *config.Configuration,
	syncPermissions gdpr.Permissions,
	metrics metrics.MetricsEngine,
	pbsAnalytics analytics.PBSAnalyticsModule,
	bidderMap map[string]openrtb_ext.BidderName) httprouter.Handle {

	bidderLookup := make(map[string]struct{})
	for k := range bidderMap {
		bidderLookup[k] = struct{}{}
	}

	deps := &cookieSyncDeps{
		syncers:         syncers,
		hostCookie:      &cfg.HostCookie,
		gDPR:            &cfg.GDPR,
		syncPermissions: syncPermissions,
		metrics:         metrics,
		pbsAnalytics:    pbsAnalytics,
		enforceCCPA:     cfg.CCPA.Enforce,
		bidderLookup:    bidderLookup,
	}
	return deps.Endpoint
}

type cookieSyncDeps struct {
	syncers         map[openrtb_ext.BidderName]usersync.Usersyncer
	hostCookie      *config.HostCookie
	gDPR            *config.GDPR
	syncPermissions gdpr.Permissions
	metrics         metrics.MetricsEngine
	pbsAnalytics    analytics.PBSAnalyticsModule
	enforceCCPA     bool
	bidderLookup    map[string]struct{}
}

func (deps *cookieSyncDeps) Endpoint(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	//CookieSyncObject makes a log of requests and responses to  /cookie_sync endpoint
	co := analytics.CookieSyncObject{
		Status:       http.StatusOK,
		Errors:       make([]error, 0),
		BidderStatus: make([]*usersync.CookieSyncBidders, 0),
	}

	defer deps.pbsAnalytics.LogCookieSyncObject(&co)

	deps.metrics.RecordCookieSync()
	userSyncCookie := usersync.ParsePBSCookieFromRequest(r, deps.hostCookie)
	if !userSyncCookie.AllowSyncs() {
		http.Error(w, "User has opted out", http.StatusUnauthorized)
		co.Status = http.StatusUnauthorized
		co.Errors = append(co.Errors, fmt.Errorf("user has opted out"))
		return
	}

	defer r.Body.Close()
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		co.Status = http.StatusBadRequest
		co.Errors = append(co.Errors, errors.New("Failed to read request body"))
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	biddersJSON, err := parseBidders(bodyBytes)
	if err != nil {
		co.Status = http.StatusBadRequest
		co.Errors = append(co.Errors, errors.New("Failed to check request.bidders in request body. Was your JSON well-formed?"))
		http.Error(w, "Failed to check request.bidders in request body. Was your JSON well-formed?", http.StatusBadRequest)
		return
	}

	parsedReq := &cookieSyncRequest{}
	if err := parseRequest(parsedReq, bodyBytes, deps.gDPR.DefaultValue); err != nil {
		co.Status = http.StatusBadRequest
		co.Errors = append(co.Errors, err)
		http.Error(w, co.Errors[len(co.Errors)-1].Error(), co.Status)
		return
	}

	if len(biddersJSON) == 0 {
		parsedReq.Bidders = make([]string, 0, len(deps.syncers))
		for bidder := range deps.syncers {
			parsedReq.Bidders = append(parsedReq.Bidders, string(bidder))
		}
	}
	setSiteCookie := siteCookieCheck(r.UserAgent())
	needSyncupForSameSite := false
	if setSiteCookie {
		_, err1 := r.Cookie(usersync.SameSiteCookieName)
		if err1 == http.ErrNoCookie {
			needSyncupForSameSite = true
		}
	}

	parsedReq.filterExistingSyncs(deps.syncers, userSyncCookie, needSyncupForSameSite)

	adapterSyncs := make(map[openrtb_ext.BidderName]bool)
	// assume all bidders will be privacy blocked
	for _, b := range parsedReq.Bidders {
		adapterSyncs[openrtb_ext.BidderName(b)] = true
	}

	privacyPolicy := privacy.Policies{
		GDPR: gdprPrivacy.Policy{
			Signal:  gdprToString(parsedReq.GDPR),
			Consent: parsedReq.Consent,
		},
		CCPA: ccpa.Policy{
			Consent: parsedReq.USPrivacy,
		},
	}

	parsedReq.filterForGDPR(deps.syncPermissions)

	if deps.enforceCCPA {
		parsedReq.filterForCCPA(deps.bidderLookup)
	}

	// surviving bidders are not privacy blocked
	for _, b := range parsedReq.Bidders {
		adapterSyncs[openrtb_ext.BidderName(b)] = false
	}
	for b, g := range adapterSyncs {
		deps.metrics.RecordAdapterCookieSync(b, g)
	}
	parsedReq.filterToLimit()

	csResp := cookieSyncResponse{
		Status:       cookieSyncStatus(userSyncCookie.LiveSyncCount()),
		BidderStatus: make([]*usersync.CookieSyncBidders, 0, len(parsedReq.Bidders)),
	}
	for i := 0; i < len(parsedReq.Bidders); i++ {
		bidder := parsedReq.Bidders[i]
		syncInfo, err := deps.syncers[openrtb_ext.BidderName(bidder)].GetUsersyncInfo(privacyPolicy)
		if err == nil {
			newSync := &usersync.CookieSyncBidders{
				BidderCode:   bidder,
				NoCookie:     true,
				UsersyncInfo: syncInfo,
			}
			csResp.BidderStatus = append(csResp.BidderStatus, newSync)
		} else {
			glog.Errorf("Failed to get usersync info for %s: %v", bidder, err)
		}
	}

	if len(csResp.BidderStatus) > 0 {
		co.BidderStatus = append(co.BidderStatus, csResp.BidderStatus...)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.Encode(csResp)
}

func parseRequest(parsedReq *cookieSyncRequest, bodyBytes []byte, gdprDefaultValue string) error {
	if err := json.Unmarshal(bodyBytes, parsedReq); err != nil {
		return fmt.Errorf("JSON parsing failed: %s", err.Error())
	}

	if parsedReq.GDPR != nil && *parsedReq.GDPR == 1 && parsedReq.Consent == "" {
		return errors.New("gdpr_consent is required if gdpr=1")
	}
	// If GDPR is ambiguous, lets untangle it here.
	if parsedReq.GDPR == nil {
		var gdpr = new(int)
		*gdpr = 1
		if gdprDefaultValue == "0" {
			*gdpr = 0
		}
		parsedReq.GDPR = gdpr
	}
	return nil
}

func gdprToString(gdpr *int) string {
	if gdpr == nil {
		return ""
	}
	return strconv.Itoa(*gdpr)
}

func parseBidders(request []byte) ([]byte, error) {
	value, valueType, _, err := jsonparser.Get(request, "bidders")
	if err == nil && valueType != jsonparser.NotExist {
		return value, nil
	} else if err != jsonparser.KeyPathNotFoundError {
		return nil, err
	}
	return nil, nil
}

func cookieSyncStatus(syncCount int) string {
	if syncCount == 0 {
		return "no_cookie"
	}
	return "ok"
}

type cookieSyncRequest struct {
	Bidders   []string `json:"bidders"`
	GDPR      *int     `json:"gdpr"`
	Consent   string   `json:"gdpr_consent"`
	USPrivacy string   `json:"us_privacy"`
	Limit     int      `json:"limit"`
}

func (req *cookieSyncRequest) filterExistingSyncs(valid map[openrtb_ext.BidderName]usersync.Usersyncer, cookie *usersync.PBSCookie, needSyncupForSameSite bool) {
	for i := 0; i < len(req.Bidders); i++ {
		thisBidder := req.Bidders[i]
		if syncer, isValid := valid[openrtb_ext.BidderName(thisBidder)]; !isValid || (cookie.HasLiveSync(syncer.FamilyName()) && !needSyncupForSameSite) {
			req.Bidders = append(req.Bidders[:i], req.Bidders[i+1:]...)
			i--
		}
	}
}

func (req *cookieSyncRequest) filterForGDPR(permissions gdpr.Permissions) {
	if req.GDPR != nil && *req.GDPR == 0 {
		return
	}

	// At this point we know the gdpr signal is Yes because the upstream call to parseRequest already denormalized the signal if it was ambiguous
	if allowSync, err := permissions.HostCookiesAllowed(context.Background(), gdpr.SignalYes, req.Consent); err != nil || !allowSync {
		req.Bidders = nil
		return
	}

	for i := 0; i < len(req.Bidders); i++ {
		if allowSync, err := permissions.BidderSyncAllowed(context.Background(), openrtb_ext.BidderName(req.Bidders[i]), gdpr.SignalYes, req.Consent); err != nil || !allowSync {
			req.Bidders = append(req.Bidders[:i], req.Bidders[i+1:]...)
			i--
		}
	}
}

func (req *cookieSyncRequest) filterForCCPA(bidderMap map[string]struct{}) {
	ccpaPolicy := &ccpa.Policy{Consent: req.USPrivacy}
	ccpaParsedPolicy, err := ccpaPolicy.Parse(bidderMap)

	if err == nil {
		for i := 0; i < len(req.Bidders); i++ {
			if ccpaParsedPolicy.ShouldEnforce(req.Bidders[i]) {
				req.Bidders = append(req.Bidders[:i], req.Bidders[i+1:]...)
				i--
			}
		}
	}
}

// filterToLimit will enforce a max limit on cookiesyncs supplied, picking a random subset of syncs to get to the limit if over.
func (req *cookieSyncRequest) filterToLimit() {
	if req.Limit <= 0 {
		return
	}
	if req.Limit >= len(req.Bidders) {
		return
	}

	// Modified Fisher and Yates' shuffle. We don't need the bidder list shuffled, so we stop shuffling once the final values beyond limit have been set.
	// We also don't bother saving the values that should go into the entries beyond limit, as they will be discarded.
	for i := len(req.Bidders) - 1; i >= req.Limit; i-- {
		j := rand.Intn(i + 1)
		if i != j {
			req.Bidders[j] = req.Bidders[i]
			// Don't complete the swap as the new value for req.Bidders[i] will be discarded below, and will never again be accessed as part of the swapping.
		}
	}
	req.Bidders = req.Bidders[:req.Limit]
	return
}

type cookieSyncResponse struct {
	Status       string                        `json:"status"`
	BidderStatus []*usersync.CookieSyncBidders `json:"bidder_status"`
}
