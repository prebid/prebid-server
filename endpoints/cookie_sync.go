package endpoints

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	gpplib "github.com/prebid/go-gpp"
	gppConstants "github.com/prebid/go-gpp/constants"
	accountService "github.com/prebid/prebid-server/v3/account"
	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/gdpr"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/privacy"
	"github.com/prebid/prebid-server/v3/privacy/ccpa"
	gppPrivacy "github.com/prebid/prebid-server/v3/privacy/gpp"
	"github.com/prebid/prebid-server/v3/stored_requests"
	"github.com/prebid/prebid-server/v3/usersync"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	stringutil "github.com/prebid/prebid-server/v3/util/stringutil"
	"github.com/prebid/prebid-server/v3/util/timeutil"
)

const receiveCookieDeprecation = "receive-cookie-deprecation"

var (
	errCookieSyncOptOut                            = errors.New("User has opted out")
	errCookieSyncBody                              = errors.New("Failed to read request body")
	errCookieSyncGDPRConsentMissing                = errors.New("gdpr_consent is required if gdpr=1")
	errCookieSyncGDPRConsentMissingSignalAmbiguous = errors.New("gdpr_consent is required. gdpr is not specified and is assumed to be 1 by the server. set gdpr=0 to exempt this request")
	errCookieSyncInvalidBiddersType                = errors.New("invalid bidders type. must either be a string '*' or a string array of bidders")
	errCookieSyncAccountBlocked                    = errors.New("account is disabled, please reach out to the prebid server host")
	errCookieSyncAccountConfigMalformed            = errors.New("account config is malformed and could not be read")
	errCookieSyncAccountInvalid                    = errors.New("account must be valid if provided, please reach out to the prebid server host")
	errSyncerIsNotPriority                         = errors.New("syncer key is not a priority, and there are only priority elements left")
)

var cookieSyncBidderFilterAllowAll = usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude)

func NewCookieSyncEndpoint(
	syncersByBidder map[string]usersync.Syncer,
	config *config.Configuration,
	gdprPermsBuilder gdpr.PermissionsBuilder,
	tcf2CfgBuilder gdpr.TCF2ConfigBuilder,
	metrics metrics.MetricsEngine,
	analyticsRunner analytics.Runner,
	accountsFetcher stored_requests.AccountFetcher,
	bidders map[string]openrtb_ext.BidderName) HTTPRouterHandler {

	bidderHashSet := make(map[string]struct{}, len(bidders))
	for _, bidder := range bidders {
		bidderHashSet[string(bidder)] = struct{}{}
	}

	return &cookieSyncEndpoint{
		chooser: usersync.NewChooser(syncersByBidder, bidderHashSet, config.BidderInfos),
		config:  config,
		privacyConfig: usersyncPrivacyConfig{
			gdprConfig:             config.GDPR,
			gdprPermissionsBuilder: gdprPermsBuilder,
			tcf2ConfigBuilder:      tcf2CfgBuilder,
			ccpaEnforce:            config.CCPA.Enforce,
			bidderHashSet:          bidderHashSet,
		},
		metrics:         metrics,
		pbsAnalytics:    analyticsRunner,
		accountsFetcher: accountsFetcher,
		time:            &timeutil.RealTime{},
	}
}

type cookieSyncEndpoint struct {
	chooser         usersync.Chooser
	config          *config.Configuration
	privacyConfig   usersyncPrivacyConfig
	metrics         metrics.MetricsEngine
	pbsAnalytics    analytics.Runner
	accountsFetcher stored_requests.AccountFetcher
	time            timeutil.Time
}

func (c *cookieSyncEndpoint) Handle(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	request, privacyMacros, account, err := c.parseRequest(r)
	c.setCookieDeprecationHeader(w, r, account)
	if err != nil {
		c.writeParseRequestErrorMetrics(err)
		c.handleError(w, err, http.StatusBadRequest)
		return
	}
	decoder := usersync.Base64Decoder{}

	cookie := usersync.ReadCookie(r, decoder, &c.config.HostCookie)
	usersync.SyncHostCookie(r, cookie, &c.config.HostCookie)

	result := c.chooser.Choose(request, cookie)

	switch result.Status {
	case usersync.StatusBlockedByUserOptOut:
		c.metrics.RecordCookieSync(metrics.CookieSyncOptOut)
		c.handleError(w, errCookieSyncOptOut, http.StatusUnauthorized)
	case usersync.StatusBlockedByPrivacy:
		c.metrics.RecordCookieSync(metrics.CookieSyncGDPRHostCookieBlocked)
		c.handleResponse(w, request.SyncTypeFilter, cookie, privacyMacros, nil, result.BiddersEvaluated, request.Debug)
	case usersync.StatusOK:
		c.metrics.RecordCookieSync(metrics.CookieSyncOK)
		c.writeSyncerMetrics(result.BiddersEvaluated)
		c.handleResponse(w, request.SyncTypeFilter, cookie, privacyMacros, result.SyncersChosen, result.BiddersEvaluated, request.Debug)
	}
}

func (c *cookieSyncEndpoint) parseRequest(r *http.Request) (usersync.Request, macros.UserSyncPrivacy, *config.Account, error) {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return usersync.Request{}, macros.UserSyncPrivacy{}, nil, errCookieSyncBody
	}

	request := cookieSyncRequest{}
	if err := jsonutil.UnmarshalValid(body, &request); err != nil {
		return usersync.Request{}, macros.UserSyncPrivacy{}, nil, fmt.Errorf("JSON parsing failed: %s", err.Error())
	}

	if request.Account == "" {
		request.Account = metrics.PublisherUnknown
	}
	account, fetchErrs := accountService.GetAccount(context.Background(), c.config, c.accountsFetcher, request.Account, c.metrics)
	if len(fetchErrs) > 0 {
		return usersync.Request{}, macros.UserSyncPrivacy{}, nil, combineErrors(fetchErrs)
	}

	request = c.setLimit(request, account.CookieSync)
	request = c.setCooperativeSync(request, account.CookieSync)

	privacyMacros, gdprSignal, privacyPolicies, err := extractPrivacyPolicies(request, c.privacyConfig.gdprConfig.DefaultValue)
	if err != nil {
		return usersync.Request{}, macros.UserSyncPrivacy{}, account, err
	}

	ccpaParsedPolicy := ccpa.ParsedPolicy{}
	if request.USPrivacy != "" {
		parsedPolicy, err := ccpa.Policy{Consent: request.USPrivacy}.Parse(c.privacyConfig.bidderHashSet)
		if err != nil {
			privacyMacros.USPrivacy = ""
		}
		if c.privacyConfig.ccpaEnforce {
			ccpaParsedPolicy = parsedPolicy
		}
	}

	activityControl := privacy.NewActivityControl(&account.Privacy)

	syncTypeFilter, err := parseTypeFilter(request.FilterSettings)
	if err != nil {
		return usersync.Request{}, macros.UserSyncPrivacy{}, account, err
	}

	gdprRequestInfo := gdpr.RequestInfo{
		Consent:    privacyMacros.GDPRConsent,
		GDPRSignal: gdprSignal,
	}

	tcf2Cfg := c.privacyConfig.tcf2ConfigBuilder(c.privacyConfig.gdprConfig.TCF2, account.GDPR)
	gdprPerms := c.privacyConfig.gdprPermissionsBuilder(tcf2Cfg, gdprRequestInfo)

	limit := math.MaxInt
	if request.Limit != nil {
		limit = *request.Limit
	}

	rx := usersync.Request{
		Bidders: request.Bidders,
		Cooperative: usersync.Cooperative{
			Enabled:        (request.CooperativeSync != nil && *request.CooperativeSync) || (request.CooperativeSync == nil && c.config.UserSync.Cooperative.EnabledByDefault),
			PriorityGroups: c.config.UserSync.PriorityGroups,
		},
		Debug: request.Debug,
		Limit: limit,
		Privacy: usersyncPrivacy{
			gdprPermissions:  gdprPerms,
			ccpaParsedPolicy: ccpaParsedPolicy,
			activityControl:  activityControl,
			activityRequest:  privacy.NewRequestFromPolicies(privacyPolicies),
			gdprSignal:       gdprSignal,
		},
		SyncTypeFilter: syncTypeFilter,
		GPPSID:         request.GPPSID,
	}
	return rx, privacyMacros, account, nil
}

func extractPrivacyPolicies(request cookieSyncRequest, usersyncDefaultGDPRValue string) (macros.UserSyncPrivacy, gdpr.Signal, privacy.Policies, error) {
	// GDPR
	gppSID, err := stringutil.StrToInt8Slice(request.GPPSID)
	if err != nil {
		return macros.UserSyncPrivacy{}, gdpr.SignalNo, privacy.Policies{}, err
	}

	gdprSignal, gdprString, err := extractGDPRSignal(request.GDPR, gppSID)
	if err != nil {
		return macros.UserSyncPrivacy{}, gdpr.SignalNo, privacy.Policies{}, err
	}

	var gpp gpplib.GppContainer
	if len(request.GPP) > 0 {
		var errs []error
		gpp, errs = gpplib.Parse(request.GPP)
		if len(errs) > 0 {
			return macros.UserSyncPrivacy{}, gdpr.SignalNo, privacy.Policies{}, errs[0]
		}
	}

	gdprConsent := request.GDPRConsent
	if i := gppPrivacy.IndexOfSID(gpp, gppConstants.SectionTCFEU2); i >= 0 {
		gdprConsent = gpp.Sections[i].GetValue()
	}

	if gdprConsent == "" {
		if gdprSignal == gdpr.SignalYes {
			return macros.UserSyncPrivacy{}, gdpr.SignalNo, privacy.Policies{}, errCookieSyncGDPRConsentMissing
		}

		if gdprSignal == gdpr.SignalAmbiguous && gdpr.SignalNormalize(gdprSignal, usersyncDefaultGDPRValue) == gdpr.SignalYes {
			return macros.UserSyncPrivacy{}, gdpr.SignalNo, privacy.Policies{}, errCookieSyncGDPRConsentMissingSignalAmbiguous
		}
	}

	// CCPA
	ccpaString, err := ccpa.SelectCCPAConsent(request.USPrivacy, gpp, gppSID)
	if err != nil {
		return macros.UserSyncPrivacy{}, gdpr.SignalNo, privacy.Policies{}, err
	}

	privacyMacros := macros.UserSyncPrivacy{
		GDPR:        gdprString,
		GDPRConsent: gdprConsent,
		USPrivacy:   ccpaString,
		GPP:         request.GPP,
		GPPSID:      request.GPPSID,
	}

	privacyPolicies := privacy.Policies{
		GPPSID: gppSID,
	}

	return privacyMacros, gdprSignal, privacyPolicies, nil
}

func extractGDPRSignal(requestGDPR *int, gppSID []int8) (gdpr.Signal, string, error) {
	if len(gppSID) > 0 {
		if gppPrivacy.IsSIDInList(gppSID, gppConstants.SectionTCFEU2) {
			return gdpr.SignalYes, strconv.Itoa(int(gdpr.SignalYes)), nil
		}
		return gdpr.SignalNo, strconv.Itoa(int(gdpr.SignalNo)), nil
	}

	if requestGDPR == nil {
		return gdpr.SignalAmbiguous, "", nil
	}

	gdprSignal, err := gdpr.IntSignalParse(*requestGDPR)
	if err != nil {
		return gdpr.SignalAmbiguous, strconv.Itoa(*requestGDPR), err
	}
	return gdprSignal, strconv.Itoa(*requestGDPR), nil
}

func (c *cookieSyncEndpoint) writeParseRequestErrorMetrics(err error) {
	switch err {
	case errCookieSyncAccountBlocked:
		c.metrics.RecordCookieSync(metrics.CookieSyncAccountBlocked)
	case errCookieSyncAccountConfigMalformed:
		c.metrics.RecordCookieSync(metrics.CookieSyncAccountConfigMalformed)
	case errCookieSyncAccountInvalid:
		c.metrics.RecordCookieSync(metrics.CookieSyncAccountInvalid)
	default:
		c.metrics.RecordCookieSync(metrics.CookieSyncBadRequest)
	}
}

func (c *cookieSyncEndpoint) setLimit(request cookieSyncRequest, cookieSyncConfig config.CookieSync) cookieSyncRequest {
	limit := getEffectiveLimit(request.Limit, cookieSyncConfig.DefaultLimit)
	maxLimit := getEffectiveMaxLimit(cookieSyncConfig.MaxLimit)
	if maxLimit < limit {
		request.Limit = &maxLimit
	} else {
		request.Limit = &limit
	}
	return request
}

func getEffectiveLimit(reqLimit *int, defaultLimit *int) int {
	limit := reqLimit

	if limit == nil {
		limit = defaultLimit
	}

	if limit != nil && *limit > 0 {
		return *limit
	}

	return math.MaxInt
}

func getEffectiveMaxLimit(maxLimit *int) int {
	limit := maxLimit

	if limit != nil && *limit > 0 {
		return *limit
	}

	return math.MaxInt
}

func (c *cookieSyncEndpoint) setCooperativeSync(request cookieSyncRequest, cookieSyncConfig config.CookieSync) cookieSyncRequest {
	if request.CooperativeSync == nil && cookieSyncConfig.DefaultCoopSync != nil {
		request.CooperativeSync = cookieSyncConfig.DefaultCoopSync
	}

	return request
}

func parseTypeFilter(request *cookieSyncRequestFilterSettings) (usersync.SyncTypeFilter, error) {
	syncTypeFilter := usersync.SyncTypeFilter{
		IFrame:   cookieSyncBidderFilterAllowAll,
		Redirect: cookieSyncBidderFilterAllowAll,
	}

	if request != nil {
		if filter, err := parseBidderFilter(request.IFrame); err == nil {
			syncTypeFilter.IFrame = filter
		} else {
			return usersync.SyncTypeFilter{}, fmt.Errorf("error parsing filtersettings.iframe: %v", err)
		}

		if filter, err := parseBidderFilter(request.Redirect); err == nil {
			syncTypeFilter.Redirect = filter
		} else {
			return usersync.SyncTypeFilter{}, fmt.Errorf("error parsing filtersettings.image: %v", err)
		}
	}

	return syncTypeFilter, nil
}

func parseBidderFilter(filter *cookieSyncRequestFilter) (usersync.BidderFilter, error) {
	if filter == nil {
		return cookieSyncBidderFilterAllowAll, nil
	}

	var mode usersync.BidderFilterMode
	switch filter.Mode {
	case "include":
		mode = usersync.BidderFilterModeInclude
	case "exclude":
		mode = usersync.BidderFilterModeExclude
	default:
		return nil, fmt.Errorf("invalid filter value '%s'. must be either 'include' or 'exclude'", filter.Mode)
	}

	switch v := filter.Bidders.(type) {
	case string:
		if v == "*" {
			return usersync.NewUniformBidderFilter(mode), nil
		}
		return nil, fmt.Errorf("invalid bidders value `%s`. must either be '*' or a string array", v)
	case []interface{}:
		bidders := make([]string, len(v))
		for i, x := range v {
			if bidder, ok := x.(string); ok {
				bidders[i] = bidder
			} else {
				return nil, errCookieSyncInvalidBiddersType
			}
		}
		return usersync.NewSpecificBidderFilter(bidders, mode), nil
	default:
		return nil, errCookieSyncInvalidBiddersType
	}
}

func (c *cookieSyncEndpoint) handleError(w http.ResponseWriter, err error, httpStatus int) {
	http.Error(w, err.Error(), httpStatus)
	c.pbsAnalytics.LogCookieSyncObject(&analytics.CookieSyncObject{
		Status:       httpStatus,
		Errors:       []error{err},
		BidderStatus: []*analytics.CookieSyncBidder{},
	})
}

func combineErrors(errs []error) error {
	var errorStrings []string
	for _, err := range errs {
		// preserve knowledge of special account errors
		switch errortypes.ReadCode(err) {
		case errortypes.AccountDisabledErrorCode:
			return errCookieSyncAccountBlocked
		case errortypes.AcctRequiredErrorCode:
			return errCookieSyncAccountInvalid
		case errortypes.MalformedAcctErrorCode:
			return errCookieSyncAccountConfigMalformed
		}

		errorStrings = append(errorStrings, err.Error())
	}
	combinedErrors := strings.Join(errorStrings, " ")
	return errors.New(combinedErrors)
}

func (c *cookieSyncEndpoint) writeSyncerMetrics(biddersEvaluated []usersync.BidderEvaluation) {
	for _, bidder := range biddersEvaluated {
		switch bidder.Status {
		case usersync.StatusOK:
			c.metrics.RecordSyncerRequest(bidder.SyncerKey, metrics.SyncerCookieSyncOK)
		case usersync.StatusBlockedByPrivacy:
			c.metrics.RecordSyncerRequest(bidder.SyncerKey, metrics.SyncerCookieSyncPrivacyBlocked)
		case usersync.StatusAlreadySynced:
			c.metrics.RecordSyncerRequest(bidder.SyncerKey, metrics.SyncerCookieSyncAlreadySynced)
		case usersync.StatusRejectedByFilter:
			c.metrics.RecordSyncerRequest(bidder.SyncerKey, metrics.SyncerCookieSyncRejectedByFilter)
		}
	}
}

func (c *cookieSyncEndpoint) handleResponse(w http.ResponseWriter, tf usersync.SyncTypeFilter, co *usersync.Cookie, m macros.UserSyncPrivacy, s []usersync.SyncerChoice, biddersEvaluated []usersync.BidderEvaluation, debug bool) {
	status := "no_cookie"
	if co.HasAnyLiveSyncs() {
		status = "ok"
	}

	response := cookieSyncResponse{
		Status:       status,
		BidderStatus: make([]cookieSyncResponseBidder, 0, len(s)),
	}

	for _, syncerChoice := range s {
		syncTypes := tf.ForBidder(syncerChoice.Bidder)
		sync, err := syncerChoice.Syncer.GetSync(syncTypes, m)
		if err != nil {
			glog.Errorf("Failed to get usersync info for %s: %v", syncerChoice.Bidder, err)
			continue
		}

		response.BidderStatus = append(response.BidderStatus, cookieSyncResponseBidder{
			BidderCode: syncerChoice.Bidder,
			NoCookie:   true,
			UsersyncInfo: cookieSyncResponseSync{
				URL:         sync.URL,
				Type:        string(sync.Type),
				SupportCORS: sync.SupportCORS,
			},
		})
	}

	if debug {
		biddersSeen := make(map[string]struct{})
		var debugInfo []cookieSyncResponseDebug
		for _, bidderEval := range biddersEvaluated {
			var debugResponse cookieSyncResponseDebug
			debugResponse.Bidder = bidderEval.Bidder
			if bidderEval.Status == usersync.StatusDuplicate && biddersSeen[bidderEval.Bidder] == struct{}{} {
				debugResponse.Error = getDebugMessage(bidderEval.Status) + " synced as " + bidderEval.SyncerKey
				debugInfo = append(debugInfo, debugResponse)
			} else if bidderEval.Status != usersync.StatusOK {
				debugResponse.Error = getDebugMessage(bidderEval.Status)
				debugInfo = append(debugInfo, debugResponse)
			}
			biddersSeen[bidderEval.Bidder] = struct{}{}
		}
		response.Debug = debugInfo
	}

	c.pbsAnalytics.LogCookieSyncObject(&analytics.CookieSyncObject{
		Status:       http.StatusOK,
		BidderStatus: mapBidderStatusToAnalytics(response.BidderStatus),
	})

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.Encode(response)
}

func (c *cookieSyncEndpoint) setCookieDeprecationHeader(w http.ResponseWriter, r *http.Request, account *config.Account) {
	if rcd, err := r.Cookie(receiveCookieDeprecation); err == nil && rcd != nil {
		return
	}
	if account == nil || !account.Privacy.PrivacySandbox.CookieDeprecation.Enabled {
		return
	}
	cookie := &http.Cookie{
		Name:     receiveCookieDeprecation,
		Value:    "1",
		Secure:   true,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteNoneMode,
		Expires:  c.time.Now().Add(time.Second * time.Duration(account.Privacy.PrivacySandbox.CookieDeprecation.TTLSec)),
	}
	setCookiePartitioned(w, cookie)
}

// setCookiePartitioned temporary substitute for http.SetCookie(w, cookie) until it supports Partitioned cookie type. Refer https://github.com/golang/go/issues/62490
func setCookiePartitioned(w http.ResponseWriter, cookie *http.Cookie) {
	if v := cookie.String(); v != "" {
		w.Header().Add("Set-Cookie", v+"; Partitioned;")
	}
}

func mapBidderStatusToAnalytics(from []cookieSyncResponseBidder) []*analytics.CookieSyncBidder {
	to := make([]*analytics.CookieSyncBidder, len(from))
	for i, b := range from {
		to[i] = &analytics.CookieSyncBidder{
			BidderCode: b.BidderCode,
			NoCookie:   b.NoCookie,
			UsersyncInfo: &analytics.UsersyncInfo{
				URL:         b.UsersyncInfo.URL,
				Type:        b.UsersyncInfo.Type,
				SupportCORS: b.UsersyncInfo.SupportCORS,
			},
		}
	}
	return to
}

func getDebugMessage(status usersync.Status) string {
	switch status {
	case usersync.StatusAlreadySynced:
		return "Already in sync"
	case usersync.StatusBlockedByPrivacy:
		return "Rejected by privacy"
	case usersync.StatusBlockedByUserOptOut:
		return "Status blocked by user opt out"
	case usersync.StatusDuplicate:
		return "Duplicate bidder"
	case usersync.StatusUnknownBidder:
		return "Unsupported bidder"
	case usersync.StatusUnconfiguredBidder:
		return "No sync config"
	case usersync.StatusRejectedByFilter:
		return "Rejected by request filter"
	case usersync.StatusBlockedByDisabledUsersync:
		return "Sync disabled by config"
	}
	return ""
}

type cookieSyncRequest struct {
	Bidders         []string                         `json:"bidders"`
	GDPR            *int                             `json:"gdpr"`
	GDPRConsent     string                           `json:"gdpr_consent"`
	USPrivacy       string                           `json:"us_privacy"`
	Limit           *int                             `json:"limit"`
	GPP             string                           `json:"gpp"`
	GPPSID          string                           `json:"gpp_sid"`
	CooperativeSync *bool                            `json:"coopSync"`
	FilterSettings  *cookieSyncRequestFilterSettings `json:"filterSettings"`
	Account         string                           `json:"account"`
	Debug           bool                             `json:"debug"`
}

type cookieSyncRequestFilterSettings struct {
	IFrame   *cookieSyncRequestFilter `json:"iframe"`
	Redirect *cookieSyncRequestFilter `json:"image"`
}

type cookieSyncRequestFilter struct {
	Bidders interface{} `json:"bidders"`
	Mode    string      `json:"filter"`
}

type cookieSyncResponse struct {
	Status       string                     `json:"status"`
	BidderStatus []cookieSyncResponseBidder `json:"bidder_status"`
	Debug        []cookieSyncResponseDebug  `json:"debug,omitempty"`
}

type cookieSyncResponseBidder struct {
	BidderCode   string                 `json:"bidder"`
	NoCookie     bool                   `json:"no_cookie,omitempty"`
	UsersyncInfo cookieSyncResponseSync `json:"usersync,omitempty"`
}

type cookieSyncResponseSync struct {
	URL         string `json:"url,omitempty"`
	Type        string `json:"type,omitempty"`
	SupportCORS bool   `json:"supportCORS,omitempty"`
}

type cookieSyncResponseDebug struct {
	Bidder string `json:"bidder"`
	Error  string `json:"error,omitempty"`
}

type usersyncPrivacyConfig struct {
	gdprConfig             config.GDPR
	gdprPermissionsBuilder gdpr.PermissionsBuilder
	tcf2ConfigBuilder      gdpr.TCF2ConfigBuilder
	ccpaEnforce            bool
	bidderHashSet          map[string]struct{}
}

type usersyncPrivacy struct {
	gdprPermissions  gdpr.Permissions
	ccpaParsedPolicy ccpa.ParsedPolicy
	activityControl  privacy.ActivityControl
	activityRequest  privacy.ActivityRequest
	gdprSignal       gdpr.Signal
}

func (p usersyncPrivacy) GDPRAllowsHostCookie() bool {
	allowCookie, err := p.gdprPermissions.HostCookiesAllowed(context.Background())
	return err == nil && allowCookie
}

func (p usersyncPrivacy) GDPRAllowsBidderSync(bidder string) bool {
	allowSync, err := p.gdprPermissions.BidderSyncAllowed(context.Background(), openrtb_ext.BidderName(bidder))
	return err == nil && allowSync
}

func (p usersyncPrivacy) CCPAAllowsBidderSync(bidder string) bool {
	enforce := p.ccpaParsedPolicy.CanEnforce() && p.ccpaParsedPolicy.ShouldEnforce(bidder)
	return !enforce
}

func (p usersyncPrivacy) ActivityAllowsUserSync(bidder string) bool {
	return p.activityControl.Allow(
		privacy.ActivitySyncUser,
		privacy.Component{Type: privacy.ComponentTypeBidder, Name: bidder},
		p.activityRequest)
}

func (p usersyncPrivacy) GDPRInScope() bool {
	return p.gdprSignal == gdpr.SignalYes
}
