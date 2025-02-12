package pbs

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/server/ssl"
	"github.com/prebid/prebid-server/v3/usersync"
)

// Recaptcha code from https://github.com/haisum/recaptcha/blob/master/recaptcha.go
const RECAPTCHA_URL = "https://www.google.com/recaptcha/api/siteverify"

type UserSyncDeps struct {
	ExternalUrl      string
	RecaptchaSecret  string
	HostCookieConfig *config.HostCookie
	PriorityGroups   [][]string
}

// Struct for parsing json in google's response
type googleResponse struct {
	Success    bool
	ErrorCodes []string `json:"error-codes"`
}

func (deps *UserSyncDeps) VerifyRecaptcha(response string) error {
	ts := &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
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
	encoder := usersync.Base64Encoder{}
	decoder := usersync.Base64Decoder{}

	if rr == "" {
		http.Redirect(w, r, fmt.Sprintf("%s/static/optout.html", deps.ExternalUrl), http.StatusMovedPermanently)
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

	// Read Cookie
	pc := usersync.ReadCookie(r, decoder, deps.HostCookieConfig)
	usersync.SyncHostCookie(r, pc, deps.HostCookieConfig)
	pc.SetOptOut(optout != "")

	// Write Cookie
	encodedCookie, err := encoder.Encode(pc)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	usersync.WriteCookie(w, encodedCookie, deps.HostCookieConfig, false)

	if optout == "" {
		http.Redirect(w, r, deps.HostCookieConfig.OptInURL, http.StatusMovedPermanently)
	} else {
		http.Redirect(w, r, deps.HostCookieConfig.OptOutURL, http.StatusMovedPermanently)
	}
}
