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
	"github.com/prebid/prebid-server/metrics"
	"github.com/prebid/prebid-server/ssl"
)

// Recaptcha code from https://github.com/haisum/recaptcha/blob/master/recaptcha.go
const RECAPTCHA_URL = "https://www.google.com/recaptcha/api/siteverify"
const COOKIE_NAME = "uids"

type UserSyncDeps struct {
	Cookie_domain    string
	External_url     string
	Recaptcha_secret string
	Metrics          metrics.PBSMetrics
}

// UserSyncMap is cookie which stores the user sync info for all of our bidders.
//
// To get an instance of this from a request, use ParseUserSyncMapFromRequest.
// To write an instance onto a response, use SetCookieOnResponse.
type UserSyncMap interface {
	// AllowSyncs is true if the user lets bidders sync cookies, and false otherwise.
	AllowSyncs() bool
	// SetPreference is used to change whether or not we're allowed to sync cookies for this user.
	SetPreference(allow bool)
	// Gets an HTTP cookie containing all the data from this UserSyncMap. This is a snapshot--not a live view.
	ToHTTPCookie() *http.Cookie
	// SetCookieOnResponse is a shortcut for "ToHTTPCookie(); cookie.setDomain(domain); setCookie(w, cookie)"
	SetCookieOnResponse(w http.ResponseWriter, domain string)
	// GetUID Gets this user's ID for the given family, if present. If not present, this returns ("", false).
	GetUID(familyName string) (string, bool)
	// Unsync removes the user's ID for the given family from this cookie.
	Unsync(familyName string)
	// TrySync tries to set the UID for some family name. It returns an error if the set didn't happen.
	TrySync(familyName string, uid string) error
	// HasSync returns true if we have a UID for the given family, and false otherwise.
	HasSync(familyName string) bool
	// SyncCount returns the number of families which have UIDs for this user.
	SyncCount() int
}

// ParseUserSyncMapFromRequest parses the UserSyncMap from an HTTP Request.
func ParseUserSyncMapFromRequest(r *http.Request) UserSyncMap {
	cookie, err := r.Cookie(COOKIE_NAME)
	if err != nil {
		return NewSyncMap()
	}

	return ParseUserSyncMap(cookie)
}

// ParseUserSyncMap parses the UserSync cookie from a raw HTTP cookie.
func ParseUserSyncMap(cookie *http.Cookie) UserSyncMap {
	return parseCookieImpl(cookie)
}

// NewSyncMap returns an empty UserSyncMap
func NewSyncMap() UserSyncMap {
	return &cookieImpl{
		UIDs:     make(map[string]string),
		Birthday: timestamp(),
	}
}

// parseCookieImpl parses the cookieImpl from a raw HTTP cookie.
// This exists for testing. Callers should use ParseUserSyncMap.
func parseCookieImpl(cookie *http.Cookie) *cookieImpl {
	pc := cookieImpl{
		UIDs:     make(map[string]string),
		Birthday: timestamp(),
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

	// Facebook sends us a sentinel value of 0 if the user isn't logged in.
	// As a result, we've stored  "0" as the UID for many users in the audienceNetwork so far.
	// Since users log in and out of facebook all the time, this will cause re-sync attempts until
	// we get a non-zero value.
	if pc.UIDs["audienceNetwork"] == "0" {
		delete(pc.UIDs, "audienceNetwork")
	}

	return &pc
}

type cookieImpl struct {
	UIDs     map[string]string `json:"uids,omitempty"`
	OptOut   bool              `json:"optout,omitempty"`
	Birthday *time.Time        `json:"bday,omitempty"`
}

func (cookie *cookieImpl) AllowSyncs() bool {
	return !cookie.OptOut
}

func (cookie *cookieImpl) SetPreference(allow bool) {
	if allow {
		cookie.OptOut = false
	} else {
		cookie.OptOut = true
		cookie.UIDs = make(map[string]string)
	}
}

func (cookie *cookieImpl) ToHTTPCookie() *http.Cookie {
	j, _ := json.Marshal(cookie)
	b64 := base64.URLEncoding.EncodeToString(j)

	return &http.Cookie{
		Name:    COOKIE_NAME,
		Value:   b64,
		Expires: time.Now().Add(180 * 24 * time.Hour),
	}
}

func (cookie *cookieImpl) GetUID(familyName string) (string, bool) {
	uid, ok := cookie.UIDs[familyName]
	return uid, ok
}

func (cookie *cookieImpl) SetCookieOnResponse(w http.ResponseWriter, domain string) {
	httpCookie := cookie.ToHTTPCookie()
	if domain != "" {
		httpCookie.Domain = domain
	}
	http.SetCookie(w, httpCookie)
}

func (cookie *cookieImpl) Unsync(familyName string) {
	delete(cookie.UIDs, familyName)
}

func (cookie *cookieImpl) HasSync(familyName string) bool {
	_, ok := cookie.UIDs[familyName]
	return ok
}

func (cookie *cookieImpl) SyncCount() int {
	return len(cookie.UIDs)
}

func (cookie *cookieImpl) TrySync(familyName string, uid string) error {
	if !cookie.AllowSyncs() {
		return errors.New("The user has opted out of prebid server cookieImpl syncs.")
	}

	// At the moment, Facebook calls /setuid with a UID of 0 if the user isn't logged into Facebook.
	// They shouldn't be sending us a sentinel value... but since they are, we're refusing to save that ID.
	if familyName == "audienceNetwork" && uid == "0" {
		return errors.New("audienceNetwork uses a UID of 0 as \"not yet recognized\".")
	}

	cookie.UIDs[familyName] = uid
	return nil
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
		pc.Unsync(bidder)
	} else {
		pc.TrySync(bidder, uid)
	}

	deps.Metrics.DoneUserSync(bidder)
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
