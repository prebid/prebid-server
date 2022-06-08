package endpoints

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	accountService "github.com/prebid/prebid-server/account"
	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/gdpr"
	"github.com/prebid/prebid-server/metrics"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/ccpa"
	gdprPrivacy "github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/prebid/prebid-server/stored_requests"
	"github.com/prebid/prebid-server/usersync"
)

var (
	errCookieSyncOptOut                            = errors.New("User has opted out")
	errCookieSyncBody                              = errors.New("Failed to read request body")
	errCookieSyncGDPRConsentMissing                = errors.New("gdpr_consent is required if gdpr=1")
	errCookieSyncGDPRConsentMissingSignalAmbiguous = errors.New("gdpr_consent is required. gdpr is not specified and is assumed to be 1 by the server. set gdpr=0 to exempt this request")
	errCookieSyncInvalidBiddersType                = errors.New("invalid bidders type. must either be a string '*' or a string array of bidders")
	errCookieSyncAccountBlocked                    = errors.New("account is disabled, please reach out to the prebid server host")
	errCookieSyncAccountInvalid                    = errors.New("account must be valid if provided, please reach out to the prebid server host")
)

var cookieSyncBidderFilterAllowAll = usersync.NewUniformBidderFilter(usersync.BidderFilterModeInclude)

func NewCookieSyncEndpoint(
	syncersByBidder map[string]usersync.Syncer,
	config *config.Configuration,
	gdprPermissions gdpr.Permissions,
	metrics metrics.MetricsEngine,
	pbsAnalytics analytics.PBSAnalyticsModule,
	accountsFetcher stored_requests.AccountFetcher,
	bidders map[string]openrtb_ext.BidderName) HTTPRouterHandler {

	bidderHashSet := make(map[string]struct{}, len(bidders))
	for _, bidder := range bidders {
		bidderHashSet[string(bidder)] = struct{}{}
	}

	return &cookieSyncEndpoint{
		chooser: usersync.NewChooser(syncersByBidder),
		config:  config,
		privacyConfig: usersyncPrivacyConfig{
			gdprConfig:      config.GDPR,
			gdprPermissions: gdprPermissions,
			ccpaEnforce:     config.CCPA.Enforce,
			bidderHashSet:   bidderHashSet,
		},
		metrics:         metrics,
		pbsAnalytics:    pbsAnalytics,
		accountsFetcher: accountsFetcher,
	}
}

type cookieSyncEndpoint struct {
	chooser         usersync.Chooser
	config          *config.Configuration
	privacyConfig   usersyncPrivacyConfig
	metrics         metrics.MetricsEngine
	pbsAnalytics    analytics.PBSAnalyticsModule
	accountsFetcher stored_requests.AccountFetcher
}

func (c *cookieSyncEndpoint) Handle(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	request, privacyPolicies, err := c.parseRequest(r)
	if err != nil {
		c.writeParseRequestErrorMetrics(err)
		c.handleError(w, err, http.StatusBadRequest)
		return
	}

	cookie := usersync.ParseCookieFromRequest(r, &c.config.HostCookie)

	result := c.chooser.Choose(request, cookie)
	switch result.Status {
	case usersync.StatusBlockedByUserOptOut:
		c.metrics.RecordCookieSync(metrics.CookieSyncOptOut)
		c.handleError(w, errCookieSyncOptOut, http.StatusUnauthorized)
	case usersync.StatusBlockedByGDPR:
		c.metrics.RecordCookieSync(metrics.CookieSyncGDPRHostCookieBlocked)
		c.handleResponse(w, request.SyncTypeFilter, cookie, privacyPolicies, nil)
	case usersync.StatusOK:
		c.metrics.RecordCookieSync(metrics.CookieSyncOK)
		c.writeBidderMetrics(result.BiddersEvaluated)
		c.handleResponse(w, request.SyncTypeFilter, cookie, privacyPolicies, result.SyncersChosen)
	}
}

func (c *cookieSyncEndpoint) parseRequest(r *http.Request) (usersync.Request, privacy.Policies, error) {
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return usersync.Request{}, privacy.Policies{}, errCookieSyncBody
	}

	request := cookieSyncRequest{}
	if err := json.Unmarshal(body, &request); err != nil {
		return usersync.Request{}, privacy.Policies{}, fmt.Errorf("JSON parsing failed: %s", err.Error())
	}

	if request.Account == "" {
		request.Account = metrics.PublisherUnknown
	}
	account, fetchErrs := accountService.GetAccount(context.Background(), c.config, c.accountsFetcher, request.Account)
	if len(fetchErrs) > 0 {
		return usersync.Request{}, privacy.Policies{}, combineErrors(fetchErrs)
	}

	var gdprString string
	if request.GDPR != nil {
		gdprString = strconv.Itoa(*request.GDPR)
	}
	gdprSignal, err := gdpr.SignalParse(gdprString)
	if err != nil {
		return usersync.Request{}, privacy.Policies{}, err
	}

	if request.GDPRConsent == "" {
		if gdprSignal == gdpr.SignalYes {
			return usersync.Request{}, privacy.Policies{}, errCookieSyncGDPRConsentMissing
		}

		if gdprSignal == gdpr.SignalAmbiguous && gdpr.SignalNormalize(gdprSignal, c.privacyConfig.gdprConfig.DefaultValue) == gdpr.SignalYes {
			return usersync.Request{}, privacy.Policies{}, errCookieSyncGDPRConsentMissingSignalAmbiguous
		}
	}

	request = c.setLimit(request, account.CookieSync)
	request = c.setCooperativeSync(request, account.CookieSync)

	privacyPolicies := privacy.Policies{
		GDPR: gdprPrivacy.Policy{
			Signal:  gdprString,
			Consent: request.GDPRConsent,
		},
		CCPA: ccpa.Policy{
			Consent: request.USPrivacy,
		},
	}

	ccpaParsedPolicy := ccpa.ParsedPolicy{}
	if request.USPrivacy != "" {
		parsedPolicy, err := privacyPolicies.CCPA.Parse(c.privacyConfig.bidderHashSet)
		if err != nil {
			privacyPolicies.CCPA.Consent = ""
		}
		if c.privacyConfig.ccpaEnforce {
			ccpaParsedPolicy = parsedPolicy
		}
	}

	syncTypeFilter, err := parseTypeFilter(request.FilterSettings)
	if err != nil {
		return usersync.Request{}, privacy.Policies{}, err
	}

	rx := usersync.Request{
		Bidders: request.Bidders,
		Cooperative: usersync.Cooperative{
			Enabled:        (request.CooperativeSync != nil && *request.CooperativeSync) || (request.CooperativeSync == nil && c.config.UserSync.Cooperative.EnabledByDefault),
			PriorityGroups: c.config.UserSync.Cooperative.PriorityGroups,
		},
		Limit: request.Limit,
		Privacy: usersyncPrivacy{
			gdprPermissions:  c.privacyConfig.gdprPermissions,
			gdprSignal:       gdprSignal,
			gdprConsent:      request.GDPRConsent,
			ccpaParsedPolicy: ccpaParsedPolicy,
		},
		SyncTypeFilter: syncTypeFilter,
	}
	return rx, privacyPolicies, nil
}

func (c *cookieSyncEndpoint) writeParseRequestErrorMetrics(err error) {
	switch err {
	case errCookieSyncAccountBlocked:
		c.metrics.RecordCookieSync(metrics.CookieSyncAccountBlocked)
	case errCookieSyncAccountInvalid:
		c.metrics.RecordCookieSync(metrics.CookieSyncAccountInvalid)
	default:
		c.metrics.RecordCookieSync(metrics.CookieSyncBadRequest)
	}
}

func (c *cookieSyncEndpoint) setLimit(request cookieSyncRequest, cookieSyncConfig config.CookieSync) cookieSyncRequest {
	if request.Limit <= 0 && cookieSyncConfig.DefaultLimit != nil {
		request.Limit = *cookieSyncConfig.DefaultLimit
	}
	if cookieSyncConfig.MaxLimit != nil && (request.Limit <= 0 || request.Limit > *cookieSyncConfig.MaxLimit) {
		request.Limit = *cookieSyncConfig.MaxLimit
	}
	if request.Limit < 0 {
		request.Limit = 0
	}

	return request
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
		case errortypes.BlacklistedAcctErrorCode:
			return errCookieSyncAccountBlocked
		case errortypes.AcctRequiredErrorCode:
			return errCookieSyncAccountInvalid
		}

		errorStrings = append(errorStrings, err.Error())
	}
	combinedErrors := strings.Join(errorStrings, " ")
	return errors.New(combinedErrors)
}

func (c *cookieSyncEndpoint) writeBidderMetrics(biddersEvaluated []usersync.BidderEvaluation) {
	for _, bidder := range biddersEvaluated {
		switch bidder.Status {
		case usersync.StatusOK:
			c.metrics.RecordSyncerRequest(bidder.SyncerKey, metrics.SyncerCookieSyncOK)
		case usersync.StatusBlockedByGDPR:
			c.metrics.RecordSyncerRequest(bidder.SyncerKey, metrics.SyncerCookieSyncPrivacyBlocked)
		case usersync.StatusBlockedByCCPA:
			c.metrics.RecordSyncerRequest(bidder.SyncerKey, metrics.SyncerCookieSyncPrivacyBlocked)
		case usersync.StatusAlreadySynced:
			c.metrics.RecordSyncerRequest(bidder.SyncerKey, metrics.SyncerCookieSyncAlreadySynced)
		case usersync.StatusTypeNotSupported:
			c.metrics.RecordSyncerRequest(bidder.SyncerKey, metrics.SyncerCookieSyncTypeNotSupported)
		}
	}
}

func (c *cookieSyncEndpoint) handleResponse(w http.ResponseWriter, tf usersync.SyncTypeFilter, co *usersync.Cookie, p privacy.Policies, s []usersync.SyncerChoice) {
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
		sync, err := syncerChoice.Syncer.GetSync(syncTypes, p)
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

	c.pbsAnalytics.LogCookieSyncObject(&analytics.CookieSyncObject{
		Status:       http.StatusOK,
		BidderStatus: mapBidderStatusToAnalytics(response.BidderStatus),
	})

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.Encode(response)
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

type cookieSyncRequest struct {
	Bidders         []string                         `json:"bidders"`
	GDPR            *int                             `json:"gdpr"`
	GDPRConsent     string                           `json:"gdpr_consent"`
	USPrivacy       string                           `json:"us_privacy"`
	Limit           int                              `json:"limit"`
	CooperativeSync *bool                            `json:"coopSync"`
	FilterSettings  *cookieSyncRequestFilterSettings `json:"filterSettings"`
	Account         string                           `json:"account"`
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

type usersyncPrivacyConfig struct {
	gdprConfig      config.GDPR
	gdprPermissions gdpr.Permissions
	ccpaEnforce     bool
	bidderHashSet   map[string]struct{}
}

type usersyncPrivacy struct {
	gdprPermissions  gdpr.Permissions
	gdprSignal       gdpr.Signal
	gdprConsent      string
	ccpaParsedPolicy ccpa.ParsedPolicy
}

func (p usersyncPrivacy) GDPRAllowsHostCookie() bool {
	allowCookie, err := p.gdprPermissions.HostCookiesAllowed(context.Background(), p.gdprSignal, p.gdprConsent)
	return err == nil && allowCookie
}

func (p usersyncPrivacy) GDPRAllowsBidderSync(bidder string) bool {
	allowSync, err := p.gdprPermissions.BidderSyncAllowed(context.Background(), openrtb_ext.BidderName(bidder), p.gdprSignal, p.gdprConsent)
	return err == nil && allowSync
}

func (p usersyncPrivacy) CCPAAllowsBidderSync(bidder string) bool {
	enforce := p.ccpaParsedPolicy.CanEnforce() && p.ccpaParsedPolicy.ShouldEnforce(bidder)
	return !enforce
}
