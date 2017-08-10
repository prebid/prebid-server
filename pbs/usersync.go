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

// customBidderTTLs stores rules about how long a particular UID sync is valid for each bidder.
// If a bidder does a cookie sync *without* listing a rule here, then the UID's TTL will be 7 days.
var customBidderTTLs = map[string]time.Duration{}

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
	// HasSync returns true if we have an active UID for the given family, and false otherwise.
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
		TemporaryUIDs: make(map[string]temporaryUid),
		Birthday:      timestamp(),
	}
}

// parseCookieImpl parses the cookieImpl from a raw HTTP cookie.
// This exists for testing. Callers should use ParseUserSyncMap.
func parseCookieImpl(cookie *http.Cookie) *cookieImpl {
	pc := cookieImpl{
		TemporaryUIDs: make(map[string]temporaryUid),
		Birthday:      timestamp(),
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
	if pc.OptOut || pc.TemporaryUIDs == nil {
		pc.TemporaryUIDs = make(map[string]temporaryUid)
	}

	// Facebook sends us a sentinel value of 0 if the user isn't logged in.
	// As a result, we've stored  "0" as the UID for many users in the audienceNetwork so far.
	// Since users log in and out of facebook all the time, this will cause re-sync attempts until
	// we get a non-zero value.
	//
	// If you're seeing this message after February 2018, this block of logic is safe to delete.
	if pc.UIDs["audienceNetwork"] == "0" {
		delete(pc.UIDs, "audienceNetwork")
	}

	// This exists to help migrate a "legacy" cookie format onto the new one. Originally, cookies did not
	// allow per-bidder expiration dates. Now, they do.
	// This block attaches TTLs for each bidder. It uses a short(ish) TTL so that they re-sync soon, since there's
	// no record of how long ago this UID was generated.
	//
	// If you're seeing this message after February 2018, this block of logic is safe to delete.
	for bidder, uid := range pc.UIDs {
		pc.TemporaryUIDs[bidder] = temporaryUid{
			UID:     uid,
			Expires: time.Now().Add(5 * time.Minute),
		}
	}

	pc.UIDs = nil

	return &pc
}

type temporaryUid struct {
	// uid is the ID given to a user by a particular bidder
	UID string `json:"uid"`
	// Expires is the time at which this UID should no longer apply.
	Expires time.Time `json:"expires"`
}

type cookieImpl struct {
	// UIDs *should not be used* outside of the parseCookieImpl function.
	// They exist for legacy reasons, but should be nil everywhere else.
	// If you're seeing this message after February 2018, they are safe to delete.
	UIDs map[string]string `json:"uids,omitempty"`
	// TemporaryUIDs stores a mapping from various bidders' FamilyNames to the UIDs which recognize them.
	TemporaryUIDs map[string]temporaryUid `json:"user_ids,omitempty"`
	// OptOut is true if the user has opted not to let prebid-server sync user IDs, and false otherwise.
	OptOut bool `json:"optout,omitempty"`
	// Birthday is the time
	Birthday *time.Time `json:"bday,omitempty"`
}

func (cookie *cookieImpl) AllowSyncs() bool {
	return !cookie.OptOut
}

func (cookie *cookieImpl) SetPreference(allow bool) {
	if allow {
		cookie.OptOut = false
	} else {
		cookie.OptOut = true
		cookie.TemporaryUIDs = make(map[string]temporaryUid)
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
	if value, ok := cookie.TemporaryUIDs[familyName]; ok {
		if time.Now().Before(value.Expires) {
			return value.UID, true
		}
	}
	return "", false
}

func (cookie *cookieImpl) SetCookieOnResponse(w http.ResponseWriter, domain string) {
	httpCookie := cookie.ToHTTPCookie()
	if domain != "" {
		httpCookie.Domain = domain
	}
	http.SetCookie(w, httpCookie)
}

func (cookie *cookieImpl) Unsync(familyName string) {
	delete(cookie.TemporaryUIDs, familyName)
}

func (cookie *cookieImpl) HasSync(familyName string) bool {
	_, ok := cookie.GetUID(familyName)
	return ok
}

func (cookie *cookieImpl) SyncCount() int {
	return len(cookie.TemporaryUIDs)
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

	cookie.TemporaryUIDs[familyName] = temporaryUid{
		UID:     uid,
		Expires: getExpiry(familyName),
	}

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

// getExpiry gets an expiry date for the cookie, assuming it was generated right now.
func getExpiry(familyName string) time.Time {
	ttl := 7 * 24 * time.Hour
	if customTTL, ok := customBidderTTLs[familyName]; ok {
		ttl = customTTL
	}
	now := time.Now()
	return now.Add(ttl)
}

func timestamp() *time.Time {
	birthday := time.Now()
	return &birthday
}
