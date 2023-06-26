package endpoints

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	gpplib "github.com/prebid/go-gpp"
	gppConstants "github.com/prebid/go-gpp/constants"
	accountService "github.com/prebid/prebid-server/account"
	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/gdpr"
	"github.com/prebid/prebid-server/metrics"
	gppPrivacy "github.com/prebid/prebid-server/privacy/gpp"
	"github.com/prebid/prebid-server/stored_requests"
	"github.com/prebid/prebid-server/usersync"
	"github.com/prebid/prebid-server/util/httputil"
	stringutil "github.com/prebid/prebid-server/util/stringutil"
)

const (
	chromeStr       = "Chrome/"
	chromeiOSStr    = "CriOS/"
	chromeMinVer    = 67
	chromeStrLen    = len(chromeStr)
	chromeiOSStrLen = len(chromeiOSStr)
)

func NewSetUIDEndpoint(cfg *config.Configuration, syncersByBidder map[string]usersync.Syncer, gdprPermsBuilder gdpr.PermissionsBuilder, tcf2CfgBuilder gdpr.TCF2ConfigBuilder, pbsanalytics analytics.PBSAnalyticsModule, accountsFetcher stored_requests.AccountFetcher, metricsEngine metrics.MetricsEngine) httprouter.Handle {
	cookieTTL := time.Duration(cfg.HostCookie.TTL) * 24 * time.Hour

	// convert map of syncers by bidder to map of syncers by key
	// - its safe to assume that if multiple bidders map to the same key, the syncers are interchangeable.
	syncersByKey := make(map[string]usersync.Syncer, len(syncersByBidder))
	for _, v := range syncersByBidder {
		syncersByKey[v.Key()] = v
	}

	return httprouter.Handle(func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		so := analytics.SetUIDObject{
			Status: http.StatusOK,
			Errors: make([]error, 0),
		}

		defer pbsanalytics.LogSetUIDObject(&so)

		pc := usersync.ParseCookieFromRequest(r, &cfg.HostCookie)
		if !pc.AllowSyncs() {
			w.WriteHeader(http.StatusUnauthorized)
			metricsEngine.RecordSetUid(metrics.SetUidOptOut)
			so.Status = http.StatusUnauthorized
			return
		}

		query := r.URL.Query()

		syncer, err := getSyncer(query, syncersByKey)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			metricsEngine.RecordSetUid(metrics.SetUidSyncerUnknown)
			so.Errors = []error{err}
			so.Status = http.StatusBadRequest
			return
		}
		so.Bidder = syncer.Key()

		responseFormat, err := getResponseFormat(query, syncer)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			metricsEngine.RecordSetUid(metrics.SetUidBadRequest)
			so.Errors = []error{err}
			so.Status = http.StatusBadRequest
			return
		}

		accountID := query.Get("account")
		if accountID == "" {
			accountID = metrics.PublisherUnknown
		}
		account, fetchErrs := accountService.GetAccount(context.Background(), cfg, accountsFetcher, accountID, metricsEngine)
		if len(fetchErrs) > 0 {
			w.WriteHeader(http.StatusBadRequest)
			err := combineErrors(fetchErrs)
			w.Write([]byte(err.Error()))
			switch err {
			case errCookieSyncAccountBlocked:
				metricsEngine.RecordSetUid(metrics.SetUidAccountBlocked)
			case errCookieSyncAccountConfigMalformed:
				metricsEngine.RecordSetUid(metrics.SetUidAccountConfigMalformed)
			case errCookieSyncAccountInvalid:
				metricsEngine.RecordSetUid(metrics.SetUidAccountInvalid)
			default:
				metricsEngine.RecordSetUid(metrics.SetUidBadRequest)
			}
			so.Errors = []error{err}
			so.Status = http.StatusBadRequest
			return
		}

		gdprRequestInfo, err := extractGDPRInfo(query)
		if err != nil {
			// Only exit if non-warning
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			metricsEngine.RecordSetUid(metrics.SetUidBadRequest)
			so.Errors = []error{err}
			so.Status = http.StatusBadRequest
			return
		}

		tcf2Cfg := tcf2CfgBuilder(cfg.GDPR.TCF2, account.GDPR)

		if shouldReturn, status, body := preventSyncsGDPR(gdprRequestInfo, gdprPermsBuilder, tcf2Cfg); shouldReturn {
			w.WriteHeader(status)
			w.Write([]byte(body))
			switch status {
			case http.StatusBadRequest:
				metricsEngine.RecordSetUid(metrics.SetUidBadRequest)
			case http.StatusUnavailableForLegalReasons:
				metricsEngine.RecordSetUid(metrics.SetUidGDPRHostCookieBlocked)
			}
			so.Errors = []error{errors.New(body)}
			so.Status = status
			return
		}

		uid := query.Get("uid")
		so.UID = uid

		if uid == "" {
			pc.Unsync(syncer.Key())
			metricsEngine.RecordSetUid(metrics.SetUidOK)
			metricsEngine.RecordSyncerSet(syncer.Key(), metrics.SyncerSetUidCleared)
			so.Success = true
		} else if err = pc.TrySync(syncer.Key(), uid); err == nil {
			metricsEngine.RecordSetUid(metrics.SetUidOK)
			metricsEngine.RecordSyncerSet(syncer.Key(), metrics.SyncerSetUidOK)
			so.Success = true
		}

		setSiteCookie := siteCookieCheck(r.UserAgent())
		pc.SetCookieOnResponse(w, setSiteCookie, &cfg.HostCookie, cookieTTL)

		switch responseFormat {
		case "i":
			w.Header().Add("Content-Type", httputil.Pixel1x1PNG.ContentType)
			w.Header().Add("Content-Length", strconv.Itoa(len(httputil.Pixel1x1PNG.Content)))
			w.WriteHeader(http.StatusOK)
			// signal, consent, error := parseGDPRFromGPP(query)
			w.Write(httputil.Pixel1x1PNG.Content)
		case "b":
			w.Header().Add("Content-Type", "text/html")
			w.Header().Add("Content-Length", "0")
			w.WriteHeader(http.StatusOK)
		}
	})
}

// extractGDPRInfo looks for the GDPR consent string and GDPR signal in the GPP query params
// first and the 'gdpr' and 'gdpr_consent' query params second. If found in both, throws a
// warning
//
// WRONG! Write the following functions instead:
// signal, consent, error := parseGDPRFromGPP(query)
// legacySignal, legacyConsent, err := parseLegacyGDPRFields(query)
//
// if valid signal or valid consent is found in both, then warning
//
// We could probably make them pointers instead:
// var legacySignal, legacyConsent, signal, consent *string
// err := parseGDPRFromGPP(query, signal, consent)
// err := parseLegacyGDPRFields(query, legacySignal, legacyConsent)
func extractGDPRInfo(query url.Values) (gdpr.RequestInfo, error) {

	reqInfo, err := parseGDPRFromGPP(query)
	if err != nil {
		return gdpr.RequestInfo{}, err
	}

	legacySignal, legacyConsent, err := parseLegacyGDPRFields(query, reqInfo.GDPRSignal, reqInfo.Consent)
	if err != nil {
		if !errortypes.IsWarning(err) {
			return gdpr.RequestInfo{}, err
		}
	} else {
		reqInfo = gdpr.RequestInfo{
			Consent:    legacyConsent,
			GDPRSignal: legacySignal,
		}
	}

	return reqInfo, nil
}

// signal, consent, error := parseGDPRFromGPP(query)
func parseGDPRFromGPP(query url.Values) (gdpr.RequestInfo, error) {
	var gdprSignal gdpr.Signal = gdpr.SignalAmbiguous
	var gdprConsent string = ""
	var err error

	// Signal from gpp_sid list takes precedence
	gdprSignal, err = parseSignalFromGppSidStr(query.Get("gpp_sid"))
	if err != nil {
		return gdpr.RequestInfo{GDPRSignal: gdpr.SignalAmbiguous}, err
	}

	gdprConsent, err = parseConsentFromGppStr(query.Get("gpp"))
	if err != nil {
		return gdpr.RequestInfo{GDPRSignal: gdpr.SignalAmbiguous}, err
	}

	if gdprConsent == "" && gdprSignal == gdpr.SignalYes {
		return gdpr.RequestInfo{GDPRSignal: gdpr.SignalAmbiguous}, errors.New("GDPR consent is required when gdpr signal equals 1")
	}

	return gdpr.RequestInfo{
		Consent:    gdprConsent,
		GDPRSignal: gdprSignal,
	}, nil
}

func parseLegacyGDPRFields(query url.Values, gppGDPRSignal gdpr.Signal, gppGDPRConsent string) (gdpr.Signal, string, error) {
	var gdprSignal gdpr.Signal
	var gdprConsent string
	var warning *errortypes.Warning

	if gdprQuerySignal := query.Get("gdpr"); len(gdprQuerySignal) > 0 {
		if gppGDPRSignal == gdpr.SignalAmbiguous {
			if gdprQuerySignal != "" && gdprQuerySignal != "0" && gdprQuerySignal != "1" {
				return gdpr.SignalAmbiguous, "", errors.New("the gdpr query param must be either 0 or 1. You gave " + gdprQuerySignal)
			}

			if i, err := strconv.Atoi(gdprQuerySignal); err == nil {
				gdprSignal = gdpr.Signal(i)
			}
		} else {
			warning = &errortypes.Warning{
				Message:     "'gpp_sid' signal value will be used over the one found in the deprecated 'gdpr' field.",
				WarningCode: errortypes.UnknownWarningCode,
			}
		}
	}

	gdprConsent = query.Get("gdpr_consent")
	if len(gdprConsent) > 0 && len(gppGDPRConsent) > 0 {
		warning = &errortypes.Warning{
			Message:     "'gpp' signal value will be used over the one found in the deprecated 'gdpr_consent' field.",
			WarningCode: errortypes.UnknownWarningCode,
		}
	}

	return gdprSignal, gdprConsent, warning
}

// getGDPRSignal looks for a GDPR signal value in the "gpp_sid" and "gdpr" URL query
// fields returns gdpr.SignalAmbiguous as default value. If values are found in both
// "gpp_sid" and "gdpr" fields, the "gpp_sid" value will be returned along with a
// warning
func getGDPRSignal(query url.Values) (gdpr.Signal, error) {
	var gdprSignal gdpr.Signal = gdpr.SignalAmbiguous
	var err error

	// Signal from gpp_sid list takes precedence
	gdprSignal, err = parseSignalFromGppSidStr(query.Get("gpp_sid"))
	if err != nil {
		return gdpr.SignalAmbiguous, err
	}

	// Signal from gdpr field, if any
	if gdprQuerySignal := query.Get("gdpr"); len(gdprQuerySignal) > 0 {
		if gdprSignal == gdpr.SignalAmbiguous {
			if gdprQuerySignal != "" && gdprQuerySignal != "0" && gdprQuerySignal != "1" {
				return gdpr.SignalAmbiguous, errors.New("the gdpr query param must be either 0 or 1. You gave " + gdprQuerySignal)
			}

			if i, err := strconv.Atoi(gdprQuerySignal); err == nil {
				gdprSignal = gdpr.Signal(i)
			}
		} else {
			err = &errortypes.Warning{
				Message:     "'gpp_sid' signal value will be used over the one found in the 'gdpr' field.",
				WarningCode: errortypes.UnknownWarningCode,
			}
		}
	}

	return gdprSignal, err
}

// parseSignalFromGPPSID returns gdpr.SignalAmbiguous if strSID is empty or malformed
func parseSignalFromGppSidStr(strSID string) (gdpr.Signal, error) {
	gdprSignal := gdpr.SignalAmbiguous

	if len(strSID) > 0 {
		gppSID, err := stringutil.StrToInt8Slice(strSID)
		if err != nil {
			return gdpr.SignalAmbiguous, fmt.Errorf("Error parsing gpp_sid %s", err.Error())
		}

		if len(gppSID) > 0 {
			gdprSignal = gdpr.SignalNo
			if gppPrivacy.IsSIDInList(gppSID, gppConstants.SectionTCFEU2) {
				gdprSignal = gdpr.SignalYes
			}
		}
	}

	return gdprSignal, nil
}

func parseConsentFromGppStr(gppQueryValue string) (string, error) {
	var gdprConsent string

	if len(gppQueryValue) > 0 {
		gpp, err := gpplib.Parse(gppQueryValue)
		if err != nil {
			return "", err
		}

		if i := gppPrivacy.IndexOfSID(gpp, gppConstants.SectionTCFEU2); i >= 0 {
			gdprConsent = gpp.Sections[i].GetValue()
		}
	}

	return gdprConsent, nil
}

func getSyncer(query url.Values, syncersByKey map[string]usersync.Syncer) (usersync.Syncer, error) {
	key := query.Get("bidder")

	if key == "" {
		return nil, errors.New(`"bidder" query param is required`)
	}

	syncer, syncerExists := syncersByKey[key]
	if !syncerExists {
		return nil, errors.New("The bidder name provided is not supported by Prebid Server")
	}

	return syncer, nil
}

// getResponseFormat reads the format query parameter or falls back to the syncer's default.
// Returns either "b" (iframe), "i" (redirect), or an empty string "" (legacy behavior of an
// empty response body with no content type).
func getResponseFormat(query url.Values, syncer usersync.Syncer) (string, error) {
	format, formatProvided := query["f"]
	formatEmpty := len(format) == 0 || format[0] == ""

	if !formatProvided || formatEmpty {
		switch syncer.DefaultSyncType() {
		case usersync.SyncTypeIFrame:
			return "b", nil
		case usersync.SyncTypeRedirect:
			return "i", nil
		default:
			return "", nil
		}
	}

	if !strings.EqualFold(format[0], "b") && !strings.EqualFold(format[0], "i") {
		return "", errors.New(`"f" query param is invalid. must be "b" or "i"`)
	}
	return strings.ToLower(format[0]), nil
}

// siteCookieCheck scans the input User Agent string to check if browser is Chrome and browser version is greater than the minimum version for adding the SameSite cookie attribute
func siteCookieCheck(ua string) bool {
	result := false

	index := strings.Index(ua, chromeStr)
	criOSIndex := strings.Index(ua, chromeiOSStr)
	if index != -1 {
		result = checkChromeBrowserVersion(ua, index, chromeStrLen)
	} else if criOSIndex != -1 {
		result = checkChromeBrowserVersion(ua, criOSIndex, chromeiOSStrLen)
	}

	return result
}

func checkChromeBrowserVersion(ua string, index int, chromeStrLength int) bool {
	result := false
	vIndex := index + chromeStrLength
	dotIndex := strings.Index(ua[vIndex:], ".")
	if dotIndex == -1 {
		dotIndex = len(ua[vIndex:])
	}
	version, _ := strconv.Atoi(ua[vIndex : vIndex+dotIndex])
	if version >= chromeMinVer {
		result = true
	}
	return result
}

func preventSyncsGDPR(gdprRequestInfo gdpr.RequestInfo, permsBuilder gdpr.PermissionsBuilder, tcf2Cfg gdpr.TCF2ConfigReader) (shouldReturn bool, status int, body string) {
	perms := permsBuilder(tcf2Cfg, gdprRequestInfo)

	allowed, err := perms.HostCookiesAllowed(context.Background())
	if err != nil {
		if _, ok := err.(*gdpr.ErrorMalformedConsent); ok {
			return true, http.StatusBadRequest, "gdpr_consent was invalid. " + err.Error()
		}

		// We can't distinguish between requests for a new version of the global vendor list, and requests
		// which are malformed (version number is much too large). Since we try to fetch new versions as we
		// receive requests, PBS *should* self-correct quickly, allowing us to assume most of the errors
		// caught here will be malformed strings.
		return true, http.StatusBadRequest, "No global vendor list was available to interpret this consent string. If this is a new, valid version, it should become available soon."
	}

	if allowed {
		return false, 0, ""
	}

	return true, http.StatusUnavailableForLegalReasons, "The gdpr_consent string prevents cookies from being saved"
}
