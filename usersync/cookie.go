package usersync

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"time"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const uidCookieName = "uids"

// uidTTL is the default amount of time a uid stored within a cookie is considered valid. This is
// separate from the cookie ttl.
const uidTTL = 14 * 24 * time.Hour

// Cookie is the cookie used in Prebid Server.
//
// To get an instance of this from a request, use ParseCookieFromRequest.
// To write an instance onto a response, use SetCookieOnResponse.
type Cookie struct {
	uids   map[string]uidWithExpiry
	optOut bool
}

// uidWithExpiry bundles the UID with an Expiration date.
// After the expiration, the UID is no longer valid.
type uidWithExpiry struct {
	// UID is the ID given to a user by a particular bidder
	UID string `json:"uid"`
	// Expires is the time at which this UID should no longer apply.
	Expires time.Time `json:"expires"`
}

// ParseCookieFromRequest parses the UserSyncMap from an HTTP Request.
func ParseCookieFromRequest(r *http.Request, cookie *config.HostCookie) *Cookie {
	if cookie.OptOutCookie.Name != "" {
		optOutCookie, err1 := r.Cookie(cookie.OptOutCookie.Name)
		if err1 == nil && optOutCookie.Value == cookie.OptOutCookie.Value {
			pc := NewCookie()
			pc.SetOptOut(true)
			return pc
		}
	}
	var parsed *Cookie
	uidCookie, err2 := r.Cookie(uidCookieName)
	if err2 == nil {
		parsed = ParseCookie(uidCookie)
	} else {
		parsed = NewCookie()
	}
	// Fixes #582
	if uid, _, _ := parsed.GetUID(cookie.Family); uid == "" && cookie.CookieName != "" {
		if hostCookie, err := r.Cookie(cookie.CookieName); err == nil {
			parsed.TrySync(cookie.Family, hostCookie.Value)
		}
	}
	return parsed
}

// ParseCookie parses the UserSync cookie from a raw HTTP cookie.
func ParseCookie(httpCookie *http.Cookie) *Cookie {
	jsonValue, err := base64.URLEncoding.DecodeString(httpCookie.Value)
	if err != nil {
		// corrupted cookie; we should reset
		return NewCookie()
	}

	var cookie Cookie
	if err = json.Unmarshal(jsonValue, &cookie); err != nil {
		// corrupted cookie; we should reset
		return NewCookie()
	}

	return &cookie
}

// NewCookie returns a new empty cookie.
func NewCookie() *Cookie {
	return &Cookie{
		uids: make(map[string]uidWithExpiry),
	}
}

// AllowSyncs is true if the user lets bidders sync cookies, and false otherwise.
func (cookie *Cookie) AllowSyncs() bool {
	return cookie != nil && !cookie.optOut
}

// SetOptOut is used to change whether or not we're allowed to sync cookies for this user.
func (cookie *Cookie) SetOptOut(optOut bool) {
	cookie.optOut = optOut

	if optOut {
		cookie.uids = make(map[string]uidWithExpiry)
	}
}

// Gets an HTTP cookie containing all the data from this UserSyncMap. This is a snapshot--not a live view.
func (cookie *Cookie) ToHTTPCookie(ttl time.Duration) *http.Cookie {
	j, _ := json.Marshal(cookie)
	b64 := base64.URLEncoding.EncodeToString(j)

	return &http.Cookie{
		Name:    uidCookieName,
		Value:   b64,
		Expires: time.Now().Add(ttl),
		Path:    "/",
	}
}

// GetUID Gets this user's ID for the given syncer key.
// The first returned value is the user's ID.
// The second returned value is true if we had a value stored, and false if we didn't.
// The third returned value is true if that value is "active", and false if it's expired.
//
// If no value was stored, then the "isActive" return value will be false.
func (cookie *Cookie) GetUID(key string) (string, bool, bool) {
	if cookie != nil {
		if uid, ok := cookie.uids[key]; ok {
			return uid.UID, true, time.Now().Before(uid.Expires)
		}
	}
	return "", false, false
}

// GetUIDs returns this user's ID for all the bidders
func (cookie *Cookie) GetUIDs() map[string]string {
	uids := make(map[string]string)
	if cookie != nil {
		// Extract just the uid for each bidder
		for bidderName, uidWithExpiry := range cookie.uids {
			uids[bidderName] = uidWithExpiry.UID
		}
	}
	return uids
}

// SetCookieOnResponse is a shortcut for "ToHTTPCookie(); cookie.setDomain(domain); setCookie(w, cookie)"
func (cookie *Cookie) SetCookieOnResponse(w http.ResponseWriter, setSiteCookie bool, cfg *config.HostCookie, ttl time.Duration) {
	httpCookie := cookie.ToHTTPCookie(ttl)
	var domain string = cfg.Domain

	if domain != "" {
		httpCookie.Domain = domain
	}

	var currSize int = len([]byte(httpCookie.String()))
	for cfg.MaxCookieSizeBytes > 0 && currSize > cfg.MaxCookieSizeBytes && len(cookie.uids) > 0 {
		var oldestElem string = ""
		var oldestDate int64 = math.MaxInt64
		for key, value := range cookie.uids {
			timeUntilExpiration := time.Until(value.Expires)
			if timeUntilExpiration < time.Duration(oldestDate) {
				oldestElem = key
				oldestDate = int64(timeUntilExpiration)
			}
		}
		delete(cookie.uids, oldestElem)
		httpCookie = cookie.ToHTTPCookie(ttl)
		if domain != "" {
			httpCookie.Domain = domain
		}
		currSize = len([]byte(httpCookie.String()))
	}

	if setSiteCookie {
		httpCookie.Secure = true
		httpCookie.SameSite = http.SameSiteNoneMode
	}
	w.Header().Add("Set-Cookie", httpCookie.String())
}

// Unsync removes the user's ID for the given syncer key from this cookie.
func (cookie *Cookie) Unsync(key string) {
	delete(cookie.uids, key)
}

// HasLiveSync returns true if we have an active UID for the given syncer key, and false otherwise.
func (cookie *Cookie) HasLiveSync(key string) bool {
	_, _, isLive := cookie.GetUID(key)
	return isLive
}

// HasAnyLiveSyncs returns true if this cookie has at least one active sync.
func (cookie *Cookie) HasAnyLiveSyncs() bool {
	now := time.Now()
	if cookie != nil {
		for _, value := range cookie.uids {
			if now.Before(value.Expires) {
				return true
			}
		}
	}
	return false
}

// TrySync tries to set the UID for some syncer key. It returns an error if the set didn't happen.
func (cookie *Cookie) TrySync(key string, uid string) error {
	if !cookie.AllowSyncs() {
		return errors.New("The user has opted out of prebid server cookie syncs.")
	}

	// At the moment, Facebook calls /setuid with a UID of 0 if the user isn't logged into Facebook.
	// They shouldn't be sending us a sentinel value... but since they are, we're refusing to save that ID.
	if key == string(openrtb_ext.BidderAudienceNetwork) && uid == "0" {
		return errors.New("audienceNetwork uses a UID of 0 as \"not yet recognized\".")
	}

	cookie.uids[key] = uidWithExpiry{
		UID:     uid,
		Expires: time.Now().Add(uidTTL),
	}

	return nil
}

// cookieJson defines the JSON contract for the cookie data's storage format.
//
// This exists so that Cookie (which is public) can have private fields, and the rest of
// the code doesn't have to worry about the cookie data storage format.
type cookieJson struct {
	UIDs   map[string]uidWithExpiry `json:"tempUIDs,omitempty"`
	OptOut bool                     `json:"optout,omitempty"`
}

func (cookie *Cookie) MarshalJSON() ([]byte, error) {
	return json.Marshal(cookieJson{
		UIDs:   cookie.uids,
		OptOut: cookie.optOut,
	})
}

func (cookie *Cookie) UnmarshalJSON(b []byte) error {
	var cookieContract cookieJson
	if err := json.Unmarshal(b, &cookieContract); err != nil {
		return err
	}

	cookie.optOut = cookieContract.OptOut

	if cookie.optOut {
		cookie.uids = nil
	} else {
		cookie.uids = cookieContract.UIDs
	}

	if cookie.uids == nil {
		cookie.uids = make(map[string]uidWithExpiry)
	}

	// Audience Network / Facebook Handling
	if id, ok := cookie.uids[string(openrtb_ext.BidderAudienceNetwork)]; ok && id.UID == "0" {
		delete(cookie.uids, string(openrtb_ext.BidderAudienceNetwork))
	}

	return nil
}
