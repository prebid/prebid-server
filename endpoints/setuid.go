package endpoints

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

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

const uidCookieName = "uids"

func NewSetUIDEndpoint(cfg *config.Configuration, syncersByBidder map[string]usersync.Syncer, gdprPermsBuilder gdpr.PermissionsBuilder, tcf2CfgBuilder gdpr.TCF2ConfigBuilder, pbsanalytics analytics.PBSAnalyticsModule, accountsFetcher stored_requests.AccountFetcher, metricsEngine metrics.MetricsEngine) httprouter.Handle {
	encoder := usersync.Base64Encoder{}
	decoder := usersync.Base64Decoder{}

	return httprouter.Handle(func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		so := analytics.SetUIDObject{
			Status: http.StatusOK,
			Errors: make([]error, 0),
		}

		defer pbsanalytics.LogSetUIDObject(&so)

		cookie := usersync.ReadCookie(r, decoder, &cfg.HostCookie)
		if !cookie.AllowSyncs() {
			w.WriteHeader(http.StatusUnauthorized)
			metricsEngine.RecordSetUid(metrics.SetUidOptOut)
			so.Status = http.StatusUnauthorized
			return
		}
		usersync.SyncHostCookie(r, cookie, &cfg.HostCookie)

		query := r.URL.Query()

		syncer, err := getSyncer(query, syncersByBidder)
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
			if !errortypes.IsWarning(err) {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(err.Error()))
				metricsEngine.RecordSetUid(metrics.SetUidBadRequest)
				so.Errors = []error{err}
				so.Status = http.StatusBadRequest
				return
			}
			w.Write([]byte("Warning: " + err.Error()))
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
			cookie.Unsync(syncer.Key())
			metricsEngine.RecordSetUid(metrics.SetUidOK)
			metricsEngine.RecordSyncerSet(syncer.Key(), metrics.SyncerSetUidCleared)
			so.Success = true
		} else if err = cookie.Sync(syncer.Key(), uid); err == nil {
			metricsEngine.RecordSetUid(metrics.SetUidOK)
			metricsEngine.RecordSyncerSet(syncer.Key(), metrics.SyncerSetUidOK)
			so.Success = true
		}

		setSiteCookie := siteCookieCheck(r.UserAgent())

		// Write Cookie
		encodedCookie, err := cookie.PrepareCookieForWrite(&cfg.HostCookie, encoder)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			metricsEngine.RecordSetUid(metrics.SetUidBadRequest)
			so.Errors = []error{err}
			so.Status = http.StatusBadRequest
			return
		}
		usersync.WriteCookie(w, encodedCookie, &cfg.HostCookie, setSiteCookie)

		switch responseFormat {
		case "i":
			w.Header().Add("Content-Type", httputil.Pixel1x1PNG.ContentType)
			w.Header().Add("Content-Length", strconv.Itoa(len(httputil.Pixel1x1PNG.Content)))
			w.WriteHeader(http.StatusOK)
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
// warning. Can also throw a parsing or validation error
func extractGDPRInfo(query url.Values) (reqInfo gdpr.RequestInfo, err error) {

	reqInfo, err = parseGDPRFromGPP(query)
	if err != nil {
		return gdpr.RequestInfo{GDPRSignal: gdpr.SignalAmbiguous}, err
	}

	legacySignal, legacyConsent, err := parseLegacyGDPRFields(query, reqInfo.GDPRSignal, reqInfo.Consent)
	isWarning := errortypes.IsWarning(err)

	if err != nil && !isWarning {
		return gdpr.RequestInfo{GDPRSignal: gdpr.SignalAmbiguous}, err
	}

	// If no GDPR data in the GPP fields, use legacy instead
	if reqInfo.Consent == "" && reqInfo.GDPRSignal == gdpr.SignalAmbiguous {
		reqInfo.GDPRSignal = legacySignal
		reqInfo.Consent = legacyConsent
	}

	if reqInfo.Consent == "" && reqInfo.GDPRSignal == gdpr.SignalYes {
		return gdpr.RequestInfo{GDPRSignal: gdpr.SignalAmbiguous}, errors.New("GDPR consent is required when gdpr signal equals 1")
	}

	return reqInfo, err
}

// parseGDPRFromGPP parses and validates the "gpp_sid" and "gpp" query fields.
func parseGDPRFromGPP(query url.Values) (gdpr.RequestInfo, error) {
	var gdprSignal gdpr.Signal = gdpr.SignalAmbiguous
	var gdprConsent string = ""
	var err error

	gdprSignal, err = parseSignalFromGppSidStr(query.Get("gpp_sid"))
	if err != nil {
		return gdpr.RequestInfo{GDPRSignal: gdpr.SignalAmbiguous}, err
	}

	gdprConsent, err = parseConsentFromGppStr(query.Get("gpp"))
	if err != nil {
		return gdpr.RequestInfo{GDPRSignal: gdpr.SignalAmbiguous}, err
	}

	return gdpr.RequestInfo{
		Consent:    gdprConsent,
		GDPRSignal: gdprSignal,
	}, nil
}

// parseLegacyGDPRFields parses and validates the "gdpr" and "gdpr_consent" query fields which
// are considered deprecated in favor of the "gpp" and "gpp_sid". The parsed and validated GDPR
// values contained in "gpp" and "gpp_sid" are passed in the parameters gppGDPRSignal and
// gppGDPRConsent. If the GPP parameters come with non-default values, this function discards
// "gdpr" and "gdpr_consent" and returns a warning.
func parseLegacyGDPRFields(query url.Values, gppGDPRSignal gdpr.Signal, gppGDPRConsent string) (gdpr.Signal, string, error) {
	var gdprSignal gdpr.Signal = gdpr.SignalAmbiguous
	var gdprConsent string
	var warning error

	if gdprQuerySignal := query.Get("gdpr"); len(gdprQuerySignal) > 0 {
		if gppGDPRSignal == gdpr.SignalAmbiguous {
			switch gdprQuerySignal {
			case "0":
				fallthrough
			case "1":
				if zeroOrOne, err := strconv.Atoi(gdprQuerySignal); err == nil {
					gdprSignal = gdpr.Signal(zeroOrOne)
				}
			default:
				return gdpr.SignalAmbiguous, "", errors.New("the gdpr query param must be either 0 or 1. You gave " + gdprQuerySignal)
			}
		} else {
			warning = &errortypes.Warning{
				Message:     "'gpp_sid' signal value will be used over the one found in the deprecated 'gdpr' field.",
				WarningCode: errortypes.UnknownWarningCode,
			}
		}
	}

	if gdprLegacyConsent := query.Get("gdpr_consent"); len(gdprLegacyConsent) > 0 {
		if len(gppGDPRConsent) > 0 {
			warning = &errortypes.Warning{
				Message:     "'gpp' value will be used over the one found in the deprecated 'gdpr_consent' field.",
				WarningCode: errortypes.UnknownWarningCode,
			}
		} else {
			gdprConsent = gdprLegacyConsent
		}
	}
	return gdprSignal, gdprConsent, warning
}

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

func getSyncer(query url.Values, syncersByBidder map[string]usersync.Syncer) (usersync.Syncer, error) {
	bidder := query.Get("bidder")

	if bidder == "" {
		return nil, errors.New(`"bidder" query param is required`)
	}

	syncer, syncerExists := syncersByBidder[bidder]
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
