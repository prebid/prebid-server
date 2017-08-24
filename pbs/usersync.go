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
	"github.com/prebid/prebid-server/ssl"
	metrics "github.com/rcrowley/go-metrics"
)

var cookie_domain string
var optout_url string
var optin_url string
var external_url string
var recaptcha_secret string

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

func SetUIDCookie(w http.ResponseWriter, pc *PBSCookie) {
	j, _ := json.Marshal(pc)
	b64 := base64.URLEncoding.EncodeToString(j)

	hc := http.Cookie{
		Name:    "uids",
		Value:   b64,
		Expires: time.Now().Add(180 * 24 * time.Hour),
	}
	if cookie_domain != "" {
		hc.Domain = cookie_domain
	}
	http.SetCookie(w, &hc)
}

func GetUIDs(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	pc := ParseUIDCookie(r)
	SetUIDCookie(w, pc)
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

func SetUID(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
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

	SetUIDCookie(w, pc)
}

// Recaptcha code from https://github.com/haisum/recaptcha/blob/master/recaptcha.go
var recaptchaURL = "https://www.google.com/recaptcha/api/siteverify"

// Struct for parsing json in google's response
type googleResponse struct {
	Success    bool
	ErrorCodes []string `json:"error-codes"`
}

func VerifyRecaptcha(response string) error {
	ts := &http.Transport{
		TLSClientConfig: &tls.Config{RootCAs: ssl.GetRootCAPool()},
	}

	client := &http.Client{
		Transport: ts,
	}
	resp, err := client.PostForm(recaptchaURL,
		url.Values{"secret": {recaptcha_secret}, "response": {response}})
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

func OptOut(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	optout := r.FormValue("optout")
	rr := r.FormValue("g-recaptcha-response")

	if rr == "" {
		http.Redirect(w, r, fmt.Sprintf("%s/static/optout.html", external_url), 301)
		return
	}

	err := VerifyRecaptcha(rr)
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

	SetUIDCookie(w, pc)
	if optout == "" {
		http.Redirect(w, r, optin_url, 301)
	} else {
		http.Redirect(w, r, optout_url, 301)
	}
}

// split this for testability
func InitUsersyncHandlers(router *httprouter.Router, metricsRegistry metrics.Registry, cdomain string, optout string, optin string,
	xternal_url string, captcha_secret string) {
	cookie_domain = cdomain
	optout_url = optout
	optin_url = optin
	external_url = xternal_url
	recaptcha_secret = captcha_secret

	router.GET("/getuids", GetUIDs)
	router.GET("/setuid", SetUID)
	router.POST("/optout", OptOut)
	router.GET("/optout", OptOut)
}
