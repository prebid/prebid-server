package endpoints

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/gdpr"
	"github.com/prebid/prebid-server/metrics"
	"github.com/prebid/prebid-server/usersync"
	"github.com/prebid/prebid-server/util/httputil"
)

const (
	chromeStr       = "Chrome/"
	chromeiOSStr    = "CriOS/"
	chromeMinVer    = 67
	chromeStrLen    = len(chromeStr)
	chromeiOSStrLen = len(chromeiOSStr)
)

func NewSetUIDEndpoint(cfg config.HostCookie, syncers map[string]usersync.Syncer, perms gdpr.Permissions, pbsanalytics analytics.PBSAnalyticsModule, metricsEngine metrics.MetricsEngine) httprouter.Handle {
	cookieTTL := time.Duration(cfg.TTL) * 24 * time.Hour

	return httprouter.Handle(func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		so := analytics.SetUIDObject{
			Status: http.StatusOK,
			Errors: make([]error, 0),
		}

		defer pbsanalytics.LogSetUIDObject(&so)

		pc := usersync.ParseCookieFromRequest(r, &cfg)
		if !pc.AllowSyncs() {
			w.WriteHeader(http.StatusUnauthorized)
			metricsEngine.RecordSetUid(metrics.SetUidOptOut)
			so.Status = http.StatusUnauthorized
			return
		}

		query := r.URL.Query()

		syncerKey, err := getSyncerKey(query, syncers)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			metricsEngine.RecordSetUid(metrics.SetUidSyncerUnknown)
			so.Status = http.StatusBadRequest
			return
		}
		so.Bidder = syncerKey

		responseFormat, err := getResponseFormat(query, syncers[syncerKey])
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			metricsEngine.RecordSetUid(metrics.SetUidBadRequest)
			so.Status = http.StatusBadRequest
			return
		}

		if shouldReturn, status, body := preventSyncsGDPR(query.Get("gdpr"), query.Get("gdpr_consent"), perms); shouldReturn {
			w.WriteHeader(status)
			w.Write([]byte(body))
			switch status {
			case http.StatusBadRequest:
				metricsEngine.RecordSetUid(metrics.SetUidBadRequest)
			case http.StatusUnavailableForLegalReasons:
				metricsEngine.RecordSetUid(metrics.SetUidGDPRHostCookieBlocked)
			}
			so.Status = status
			return
		}

		uid := query.Get("uid")
		so.UID = uid

		if uid == "" {
			pc.Unsync(syncerKey)
			metricsEngine.RecordSetUid(metrics.SetUidOK)
			metricsEngine.RecordSyncerSet(syncerKey, metrics.SyncerSetUidCleared)
			so.Success = true
		} else if err = pc.TrySync(syncerKey, uid); err == nil {
			metricsEngine.RecordSetUid(metrics.SetUidOK)
			metricsEngine.RecordSyncerSet(syncerKey, metrics.SyncerSetUidOK)
			so.Success = true
		}

		setSiteCookie := siteCookieCheck(r.UserAgent())
		pc.SetCookieOnResponse(w, setSiteCookie, &cfg, cookieTTL)

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

func getSyncerKey(query url.Values, syncers map[string]usersync.Syncer) (string, error) {
	key := query.Get("bidder")

	if key == "" {
		return "", errors.New(`"bidder" query param is required`)
	}

	if _, ok := syncers[key]; !ok {
		return "", errors.New("The bidder name provided is not supported by Prebid Server")
	}

	return key, nil
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

func preventSyncsGDPR(gdprEnabled string, gdprConsent string, perms gdpr.Permissions) (shouldReturn bool, status int, body string) {

	if gdprEnabled != "" && gdprEnabled != "0" && gdprEnabled != "1" {
		return true, http.StatusBadRequest, "the gdpr query param must be either 0 or 1. You gave " + gdprEnabled
	}

	if gdprEnabled == "1" && gdprConsent == "" {
		return true, http.StatusBadRequest, "gdpr_consent is required when gdpr=1"
	}

	gdprSignal := gdpr.SignalAmbiguous

	if i, err := strconv.Atoi(gdprEnabled); err == nil {
		gdprSignal = gdpr.Signal(i)
	}

	allowed, err := perms.HostCookiesAllowed(context.Background(), gdprSignal, gdprConsent)
	if err != nil {
		if _, ok := err.(*gdpr.ErrorMalformedConsent); ok {
			return true, http.StatusBadRequest, "gdpr_consent was invalid. " + err.Error()
		}

		// We can't really distinguish between requests that are for a new version of the global vendor list, and
		// ones which are simply malformed (version number is much too large).
		// Since we try to fetch new versions as requests come in for them, PBS *should* self-correct
		// rather quickly, meaning that most of these will be malformed strings.
		return true, http.StatusBadRequest, "No global vendor list was available to interpret this consent string. If this is a new, valid version, it should become available soon."
	}

	if allowed {
		return false, 0, ""
	}

	return true, http.StatusUnavailableForLegalReasons, "The gdpr_consent string prevents cookies from being saved"
}
