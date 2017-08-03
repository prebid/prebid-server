package pbs

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/metrics"
	"github.com/prebid/prebid-server/ssl"
)

type UserSyncDeps struct {
	Cookie_domain    string
	External_url     string
	Recaptcha_secret string
	Metrics          metrics.PBSMetrics
}

type PBSCookie struct {
	UIDs     map[string]string `json:"uids,omitempty"`
	OptOut   bool              `json:"optout,omitempty"`
	Birthday *time.Time        `json:"bday,omitempty"`
}

func ParseUIDCookie(r *http.Request) *PBSCookie {
	t := time.Now()
	pc := PBSCookie{
		UIDs:     make(map[string]string),
		Birthday: &t,
	}

	cookie, err := r.Cookie("uids")
	if err != nil {
		return &pc
	}
	j, err := base64.URLEncoding.DecodeString(cookie.Value)
	if err != nil {
		// corrupted cookie; we should reset
		return &pc
	}
	err = json.Unmarshal(j, &pc)
	if err != nil {
		// corrupted cookie; we should reset
		return &pc
	}
	if pc.OptOut || pc.UIDs == nil {
		pc.UIDs = make(map[string]string) // empty map
	}
	return &pc
}

func (deps *UserSyncDeps) SetUIDCookie(w http.ResponseWriter, pc *PBSCookie) {
	j, _ := json.Marshal(pc)
	b64 := base64.URLEncoding.EncodeToString(j)

	hc := http.Cookie{
		Name:    "uids",
		Value:   b64,
		Expires: time.Now().Add(180 * 24 * time.Hour),
	}
	if deps.Cookie_domain != "" {
		hc.Domain = deps.Cookie_domain
	}
	http.SetCookie(w, &hc)
}

func (deps *UserSyncDeps) GetUIDs(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	pc := ParseUIDCookie(r)
	deps.SetUIDCookie(w, pc)
	json.NewEncoder(w).Encode(pc)
	return
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

func (deps *UserSyncDeps) SetUID(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	pc := ParseUIDCookie(r)
	if pc.OptOut {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	query := getRawQueryMap(r.URL.RawQuery)
	bidder := query["bidder"]
	if bidder == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	uid := query["uid"]
	if uid == "" {
		delete(pc.UIDs, bidder)
	} else {
		pc.UIDs[bidder] = uid
	}

	deps.Metrics.DoneUserSync(bidder)
	deps.SetUIDCookie(w, pc)
}

// Recaptcha code from https://github.com/haisum/recaptcha/blob/master/recaptcha.go
var recaptchaURL = "https://www.google.com/recaptcha/api/siteverify"

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
	resp, err := client.PostForm(recaptchaURL,
		url.Values{"secret": {deps.Recaptcha_secret}, "response": {response}})
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
		http.Redirect(w, r, fmt.Sprintf("%s/static/optout.html", deps.External_url), 301)
		return
	}

	err := deps.VerifyRecaptcha(rr)
	if err != nil {
		glog.Infof("Optout failed recaptcha: %v", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	pc := ParseUIDCookie(r)
	if optout == "" {
		pc.OptOut = false
	} else {
		pc.OptOut = true
		pc.UIDs = nil
	}

	deps.SetUIDCookie(w, pc)
	if optout == "" {
		http.Redirect(w, r, "https://ib.adnxs.com/optin", 301)
	} else {
		http.Redirect(w, r, "https://ib.adnxs.com/optout", 301)
	}
}
