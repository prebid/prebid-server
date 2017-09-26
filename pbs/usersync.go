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

// DEFAULT_TTL is the default amount of time which a cookie is considered valid.
const DEFAULT_TTL = 14 * 24 * time.Hour

// customBidderTTLs stores rules about how long a particular UID sync is valid for each bidder.
// If a bidder does a cookie sync *without* listing a rule here, then the DEFAULT_TTL will be used.
var customBidderTTLs = map[string]time.Duration{}

const (
	USERSYNC_OPT_OUT     = "usersync.opt_outs"
	USERSYNC_BAD_REQUEST = "usersync.bad_requests"
	USERSYNC_SUCCESS     = "usersync.%s.sets"
)

// PBSCookie is the cookie used in Prebid Server.
//
// To get an instance of this from a request, use ParsePBSCookieFromRequest.
// To write an instance onto a response, use SetCookieOnResponse.
type PBSCookie struct {
	uids     map[string]uidWithExpiry
	optOut   bool
	birthday *time.Time
}

// uidWithExpiry bundles the UID with an Expiration date.
// After the expiration, the UID is no longer valid.
type uidWithExpiry struct {
	// UID is the ID given to a user by a particular bidder
	UID string `json:"uid"`
	// Expires is the time at which this UID should no longer apply.
	Expires time.Time `json:"expires"`
}

type UserSyncDeps struct {
	Cookie_domain    string
	External_url     string
	Recaptcha_secret string
	Metrics          metrics.Registry
}

// ParsePBSCookieFromRequest parses the UserSyncMap from an HTTP Request.
func ParsePBSCookieFromRequest(r *http.Request) *PBSCookie {
	cookie, err := r.Cookie(COOKIE_NAME)
	if err != nil {
		return NewPBSCookie()
	}

	return ParsePBSCookie(cookie)
}

// ParsePBSCookie parses the UserSync cookie from a raw HTTP cookie.
func ParsePBSCookie(cookie *http.Cookie) *PBSCookie {
	pc := NewPBSCookie()

	j, err := base64.URLEncoding.DecodeString(cookie.Value)
	if err != nil {
		// corrupted cookie; we should reset
		return pc
	}
	err = json.Unmarshal(j, pc)

	// The error on Unmarshal here isn't terribly important.
	// If the cookie has been corrupted, we should reset to an empty one anyway.
	return pc
}

// NewPBSCookie returns an empty PBSCookie
func NewPBSCookie() *PBSCookie {
	return &PBSCookie{
		uids:     make(map[string]uidWithExpiry),
		birthday: timestamp(),
	}
}

// AllowSyncs is true if the user lets bidders sync cookies, and false otherwise.
func (cookie *PBSCookie) AllowSyncs() bool {
	if cookie == nil {
		return false
	} else {
		return !cookie.optOut
	}
}

// SetPreference is used to change whether or not we're allowed to sync cookies for this user.
func (cookie *PBSCookie) SetPreference(allow bool) {
	if allow {
		cookie.optOut = false
	} else {
		cookie.optOut = true
		cookie.uids = make(map[string]uidWithExpiry)
	}
}

// Gets an HTTP cookie containing all the data from this UserSyncMap. This is a snapshot--not a live view.
func (cookie *PBSCookie) ToHTTPCookie() *http.Cookie {
	j, _ := json.Marshal(cookie)
	b64 := base64.URLEncoding.EncodeToString(j)

	return &http.Cookie{
		Name:    COOKIE_NAME,
		Value:   b64,
		Expires: time.Now().Add(180 * 24 * time.Hour),
	}
}

// GetUID Gets this user's ID for the given family.
// The first returned value is the user's ID.
// The second returned value is true if we had a value stored, and false if we didn't.
// The third returned value is true if that value is "active", and false if it's expired.
//
// If no value was stored, then the "isActive" return value will be false.
func (cookie *PBSCookie) GetUID(familyName string) (string, bool, bool) {
	if cookie != nil {
		if uid, ok := cookie.uids[familyName]; ok {
			return uid.UID, true, time.Now().Before(uid.Expires)
		}
	}
	return "", false, false
}

// SetCookieOnResponse is a shortcut for "ToHTTPCookie(); cookie.setDomain(domain); setCookie(w, cookie)"
func (cookie *PBSCookie) SetCookieOnResponse(w http.ResponseWriter, domain string) {
	httpCookie := cookie.ToHTTPCookie()
	if domain != "" {
		httpCookie.Domain = domain
	}
	http.SetCookie(w, httpCookie)
}

// Unsync removes the user's ID for the given family from this cookie.
func (cookie *PBSCookie) Unsync(familyName string) {
	delete(cookie.uids, familyName)
}

// HasLiveSync returns true if we have an active UID for the given family, and false otherwise.
func (cookie *PBSCookie) HasLiveSync(familyName string) bool {
	_, _, isLive := cookie.GetUID(familyName)
	return isLive
}

// LiveSyncCount returns the number of families which have active UIDs for this user.
func (cookie *PBSCookie) LiveSyncCount() int {
	now := time.Now()
	numSyncs := 0
	if cookie != nil {
		for _, value := range cookie.uids {
			if now.Before(value.Expires) {
				numSyncs++
			}
		}
	}
	return numSyncs
}

// TrySync tries to set the UID for some family name. It returns an error if the set didn't happen.
func (cookie *PBSCookie) TrySync(familyName string, uid string) error {
	if !cookie.AllowSyncs() {
		return errors.New("The user has opted out of prebid server PBSCookie syncs.")
	}

	// At the moment, Facebook calls /setuid with a UID of 0 if the user isn't logged into Facebook.
	// They shouldn't be sending us a sentinel value... but since they are, we're refusing to save that ID.
	if familyName == "audienceNetwork" && uid == "0" {
		return errors.New("audienceNetwork uses a UID of 0 as \"not yet recognized\".")
	}

	cookie.uids[familyName] = uidWithExpiry{
		UID:     uid,
		Expires: getExpiry(familyName),
	}

	return nil
}

// pbsCookieJson defines the JSON contract for the cookie data's storage format.
//
// This exists so that PBSCookie (which is public) can have private fields, and the rest of
// PBS doesn't have to worry about the cookie data storage format.
type pbsCookieJson struct {
	LegacyUIDs map[string]string        `json:"uids,omitempty"`
	UIDs       map[string]uidWithExpiry `json:"tempUIDs,omitempty"`
	OptOut     bool                     `json:"optout,omitempty"`
	Birthday   *time.Time               `json:"bday,omitempty"`
}

func (cookie *PBSCookie) MarshalJSON() ([]byte, error) {
	return json.Marshal(pbsCookieJson{
		UIDs:     cookie.uids,
		OptOut:   cookie.optOut,
		Birthday: cookie.birthday,
	})
}

// UnmarshalJSON holds some transition code.
//
// "Legacy" cookies had UIDs *without* expiration dates, and recognized "0" as a legitimate UID for audienceNetwork.
// "Current" cookies always include UIDs with expiration dates, and never allow "0" for audienceNetwork.
//
// This Unmarshal method interprets both data formats, and does some conversions on legacy data to make it current.
// If you're seeing this message after March 2018, it's safe to assume that all the legacy cookies have been
// updated and remove the legacy logic.
func (cookie *PBSCookie) UnmarshalJSON(b []byte) error {
	var cookieContract pbsCookieJson
	err := json.Unmarshal(b, &cookieContract)
	if err == nil {
		cookie.optOut = cookieContract.OptOut
		cookie.birthday = cookieContract.Birthday

		if cookie.optOut {
			cookie.uids = make(map[string]uidWithExpiry)
		} else {
			cookie.uids = cookieContract.UIDs

			if cookie.uids == nil {
				cookie.uids = make(map[string]uidWithExpiry, len(cookieContract.LegacyUIDs))
			}

			// Interpret "legacy" UIDs as having been expired already.
			// This should cause us to re-sync, since it would be time for a new one.
			for bidder, uid := range cookieContract.LegacyUIDs {
				if _, ok := cookie.uids[bidder]; !ok {
					cookie.uids[bidder] = uidWithExpiry{
						UID:     uid,
						Expires: time.Now().Add(-5 * time.Minute),
					}
				}
			}

			// Any "0" values from audienceNetwork really meant "no ID available." This happens if they've never
			// logged into Facebook. However... once we know a user's ID, we stop trying to re-sync them until the
			// expiration date has passed.
			//
			// Since users may log into facebook later, this is a bad strategy.
			// Since "0" is a fake ID for this bidder, we'll just treat it like it doesn't exist.
			if id, ok := cookie.uids["audienceNetwork"]; ok && id.UID == "0" {
				delete(cookie.uids, "audienceNetwork")
			}
		}
	}
	return err
}

func (deps *UserSyncDeps) GetUIDs(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	pc := ParsePBSCookieFromRequest(r)
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
	pc := ParsePBSCookieFromRequest(r)
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
		if glog.V(2) {
			glog.Infof("Optout failed recaptcha: %v", err)
		}
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	pc := ParsePBSCookieFromRequest(r)
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
	ttl := DEFAULT_TTL
	if customTTL, ok := customBidderTTLs[familyName]; ok {
		ttl = customTTL
	}
	return time.Now().Add(ttl)
}

func timestamp() *time.Time {
	birthday := time.Now()
	return &birthday
}
