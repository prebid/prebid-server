package pbs

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PubMatic-OpenWrap/prebid-server/analytics"
	"github.com/PubMatic-OpenWrap/prebid-server/config"
	"github.com/PubMatic-OpenWrap/prebid-server/pbsmetrics"
	"github.com/PubMatic-OpenWrap/prebid-server/ssl"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
)

// Recaptcha code from https://github.com/haisum/recaptcha/blob/master/recaptcha.go
const RECAPTCHA_URL = "https://www.google.com/recaptcha/api/siteverify"

const (
	USERSYNC_OPT_OUT     = "usersync.opt_outs"
	USERSYNC_BAD_REQUEST = "usersync.bad_requests"
	USERSYNC_SUCCESS     = "usersync.%s.sets"
)

// uidWithExpiry bundles the UID with an Expiration date.
// After the expiration, the UID is no longer valid.
type uidWithExpiry struct {
	// UID is the ID given to a user by a particular bidder
	UID string `json:"uid"`
	// Expires is the time at which this UID should no longer apply.
	Expires time.Time `json:"expires"`
}

type UserSyncDeps struct {
	ExternalUrl      string
	RecaptchaSecret  string
	HostCookieConfig *config.HostCookie
	MetricsEngine    pbsmetrics.MetricsEngine
	PBSAnalytics     analytics.PBSAnalyticsModule
}

// Struct for parsing json in google's response
type googleResponse struct {
	Success    bool
	ErrorCodes []string `json:"error-codes"`
}

func (deps *UserSyncDeps) VerifyRecaptcha(response string) error {
	ts := &http.Transport{
		TLSClientConfig: &tls.Config{RootCAs: ssl.GetRootCAPool()},
	}

	client := &http.Client{
		Transport: ts,
	}
	resp, err := client.PostForm(RECAPTCHA_URL,
		url.Values{"secret": {deps.RecaptchaSecret}, "response": {response}})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var gr = googleResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return err
	}
	if !gr.Success {
		return fmt.Errorf("Captcha verify failed: %s", strings.Join(gr.ErrorCodes, ", "))
	}
	return nil
}

func (deps *UserSyncDeps) OptOut(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	optout := r.FormValue("optout")
	rr := r.FormValue("g-recaptcha-response")

	if rr == "" {
		http.Redirect(w, r, fmt.Sprintf("%s/static/optout.html", deps.ExternalUrl), 301)
		return
	}

	err := deps.VerifyRecaptcha(rr)
	if err != nil {
		if glog.V(2) {
			glog.Infof("Opt Out failed recaptcha: %v", err)
		}
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	pc := usersync.ParsePBSCookieFromRequest(r, deps.HostCookieConfig)
	pc.SetPreference(optout == "")

	pc.SetCookieOnResponse(w, false, "", deps.HostCookieConfig, deps.HostCookieConfig.TTLDuration())

	if optout == "" {
		http.Redirect(w, r, deps.HostCookieConfig.OptInURL, 301)
	} else {
		http.Redirect(w, r, deps.HostCookieConfig.OptOutURL, 301)
	}
}
