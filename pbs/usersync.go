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

	"errors"
	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/ssl"
	"github.com/rcrowley/go-metrics"
)

// Recaptcha code from https://github.com/haisum/recaptcha/blob/master/recaptcha.go
const RECAPTCHA_URL = "https://www.google.com/recaptcha/api/siteverify"
const COOKIE_NAME = "uids"

const (
	USERSYNC_OPT_OUT     = "usersync.opt_outs"
	USERSYNC_BAD_REQUEST = "usersync.bad_requests"
	USERSYNC_SUCCESS     = "usersync.%s.sets"
)

type UserSyncDeps struct {
	Cookie_domain    string
	External_url     string
	Recaptcha_secret string
	Metrics          metrics.Registry
}

// ParseUserSyncMapFromRequest parses the UserSyncMap from an HTTP Request.
func ParseUserSyncMapFromRequest(r *http.Request) *PBSCookie {
	cookie, err := r.Cookie(COOKIE_NAME)
	if err != nil {
		return NewSyncMap()
	}

	return ParseUserSyncMap(cookie)
}

// ParseUserSyncMap parses the UserSync cookie from a raw HTTP cookie.
func ParseUserSyncMap(cookie *http.Cookie) *PBSCookie {
	return parseCookieImpl(cookie)
}

// NewSyncMap returns an empty UserSyncMap
func NewSyncMap() *PBSCookie {
	return &PBSCookie{
		uids:     make(map[string]string),
		birthday: timestamp(),
	}
}

// parseCookieImpl parses the PBSCookie from a raw HTTP cookie.
// This exists for testing. Callers should use ParseUserSyncMap.
func parseCookieImpl(cookie *http.Cookie) *PBSCookie {
	pc := PBSCookie{
		uids:     make(map[string]string),
		birthday: timestamp(),
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
	if pc.optOut || pc.uids == nil {
		pc.uids = make(map[string]string) // empty map
	}

	// Facebook sends us a sentinel value of 0 if the user isn't logged in.
	// As a result, we've stored  "0" as the UID for many users in the audienceNetwork so far.
	// Since users log in and out of facebook all the time, this will cause re-sync attempts until
	// we get a non-zero value.
	if pc.uids["audienceNetwork"] == "0" {
		delete(pc.uids, "audienceNetwork")
	}

	return &pc
}

type pbsCookieJson struct {
	UIDs     map[string]string `json:"uids,omitempty"`
	OptOut   bool              `json:"optout,omitempty"`
	Birthday *time.Time        `json:"bday,omitempty"`
}

// PBSCookie is cookie which stores the user sync info for all of our bidders.
//
// To get an instance of this from a request, use ParseUserSyncMapFromRequest.
// To write an instance onto a response, use SetCookieOnResponse.
type PBSCookie struct {
	uids     map[string]string
	optOut   bool
	birthday *time.Time
}

func (cookie *PBSCookie) AllowSyncs() bool {
	if cookie == nil {
		return false
	} else {
		return !cookie.optOut
	}
}

func (cookie *PBSCookie) SetPreference(allow bool) {
	if allow {
		cookie.optOut = false
	} else {
		cookie.optOut = true
		cookie.uids = make(map[string]string)
	}
}

func (cookie *PBSCookie) ToHTTPCookie() *http.Cookie {
	j, _ := json.Marshal(cookie)
	b64 := base64.URLEncoding.EncodeToString(j)

	return &http.Cookie{
		Name:    COOKIE_NAME,
		Value:   b64,
		Expires: time.Now().Add(180 * 24 * time.Hour),
	}
}

func (cookie *PBSCookie) GetUID(familyName string) (string, bool) {
	if cookie == nil {
		return "", false
	} else {
		uid, ok := cookie.uids[familyName]
		return uid, ok
	}
}

func (cookie *PBSCookie) SetCookieOnResponse(w http.ResponseWriter, domain string) {
	httpCookie := cookie.ToHTTPCookie()
	if domain != "" {
		httpCookie.Domain = domain
	}
	http.SetCookie(w, httpCookie)
}

func (cookie *PBSCookie) Unsync(familyName string) {
	delete(cookie.uids, familyName)
}

func (cookie *PBSCookie) HasSync(familyName string) bool {
	if cookie == nil {
		return false
	} else {
		_, ok := cookie.uids[familyName]
		return ok
	}
}

func (cookie *PBSCookie) SyncCount() int {
	if cookie == nil {
		return 0
	} else {
		return len(cookie.uids)
	}
}

func (cookie *PBSCookie) TrySync(familyName string, uid string) error {
	if !cookie.AllowSyncs() {
		return errors.New("The user has opted out of prebid server PBSCookie syncs.")
	}

	// At the moment, Facebook calls /setuid with a UID of 0 if the user isn't logged into Facebook.
	// They shouldn't be sending us a sentinel value... but since they are, we're refusing to save that ID.
	if familyName == "audienceNetwork" && uid == "0" {
		return errors.New("audienceNetwork uses a UID of 0 as \"not yet recognized\".")
	}

	cookie.uids[familyName] = uid
	return nil
}

func (cookie *PBSCookie) MarshalJSON() ([]byte, error) {
	return json.Marshal(pbsCookieJson{
		UIDs:     cookie.uids,
		OptOut:   cookie.optOut,
		Birthday: cookie.birthday,
	})
}

func (cookie *PBSCookie) UnmarshalJSON(b []byte) error {
	var cookieContract pbsCookieJson
	err := json.Unmarshal(b, &cookieContract)
	if err == nil {
		cookie.uids = cookieContract.UIDs
		cookie.birthday = cookieContract.Birthday
		cookie.optOut = cookieContract.OptOut
		return nil
	} else {
		return err
	}
}

func (deps *UserSyncDeps) GetUIDs(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	pc := ParseUserSyncMapFromRequest(r)
	pc.SetCookieOnResponse(w, deps.Cookie_domain)
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
	pc := ParseUserSyncMapFromRequest(r)
	if !pc.AllowSyncs() {
		w.WriteHeader(http.StatusUnauthorized)
		metrics.GetOrRegisterMeter(USERSYNC_OPT_OUT, deps.Metrics).Mark(1)
		return
	}

	query := getRawQueryMap(r.URL.RawQuery)
	bidder := query["bidder"]
	if bidder == "" {
		w.WriteHeader(http.StatusBadRequest)
		metrics.GetOrRegisterMeter(USERSYNC_BAD_REQUEST, deps.Metrics).Mark(1)
		return
	}

	uid := query["uid"]
	var err error = nil
	if uid == "" {
		pc.Unsync(bidder)
	} else {
		err = pc.TrySync(bidder, uid)
	}

	if err == nil {
		metrics.GetOrRegisterMeter(fmt.Sprintf(USERSYNC_SUCCESS, bidder), deps.Metrics).Mark(1)
	}

	pc.SetCookieOnResponse(w, deps.Cookie_domain)
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

	pc := ParseUserSyncMapFromRequest(r)
	pc.SetPreference(optout == "")

	pc.SetCookieOnResponse(w, deps.Cookie_domain)
	if optout == "" {
		http.Redirect(w, r, "https://ib.adnxs.com/optin", 301)
	} else {
		http.Redirect(w, r, "https://ib.adnxs.com/optout", 301)
	}
}

func timestamp() *time.Time {
	birthday := time.Now()
	return &birthday
}
