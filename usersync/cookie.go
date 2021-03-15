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

const (
	// DEFAULT_TTL is the default amount of time which a cookie is considered valid.
	DEFAULT_TTL         = 14 * 24 * time.Hour
	UID_COOKIE_NAME     = "uids"
	SameSiteCookieName  = "SSCookie"
	SameSiteCookieValue = "1"
	SameSiteAttribute   = "; SameSite=None"
)

// customBidderTTLs stores rules about how long a particular UID sync is valid for each bidder.
// If a bidder does a cookie sync *without* listing a rule here, then the DEFAULT_TTL will be used.
var customBidderTTLs = map[string]time.Duration{}

// bidderToFamilyNames maps the BidderName to Adapter.Name() for the early adapters.
// If a mapping isn't listed here, then we assume that the two are the same.
var bidderToFamilyNames = map[openrtb_ext.BidderName]string{
	openrtb_ext.BidderAppnexus: "adnxs",
}

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

// ParsePBSCookieFromRequest parses the UserSyncMap from an HTTP Request.
func ParsePBSCookieFromRequest(r *http.Request, cookie *config.HostCookie) *PBSCookie {
	if cookie.OptOutCookie.Name != "" {
		optOutCookie, err1 := r.Cookie(cookie.OptOutCookie.Name)
		if err1 == nil && optOutCookie.Value == cookie.OptOutCookie.Value {
			pc := NewPBSCookie()
			pc.SetPreference(false)
			return pc
		}
	}
	var parsed *PBSCookie
	uidCookie, err2 := r.Cookie(UID_COOKIE_NAME)
	if err2 == nil {
		parsed = ParsePBSCookie(uidCookie)
	} else {
		parsed = NewPBSCookie()
	}
	// Fixes #582
	if uid, _, _ := parsed.GetUID(cookie.Family); uid == "" && cookie.CookieName != "" {
		if hostCookie, err := r.Cookie(cookie.CookieName); err == nil {
			parsed.TrySync(cookie.Family, hostCookie.Value)
		}
	}
	return parsed
}

// ParsePBSCookie parses the UserSync cookie from a raw HTTP cookie.
func ParsePBSCookie(uidCookie *http.Cookie) *PBSCookie {
	pc := NewPBSCookie()

	j, err := base64.URLEncoding.DecodeString(uidCookie.Value)
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

// NewPBSCookie returns an empty PBSCookie with optOut enabled
func NewPBSCookieWithOptOut() *PBSCookie {
	return &PBSCookie{
		uids:     make(map[string]uidWithExpiry),
		optOut:   true,
		birthday: timestamp(),
	}
}

// AllowSyncs is true if the user lets bidders sync cookies, and false otherwise.
func (cookie *PBSCookie) AllowSyncs() bool {
	return cookie != nil && !cookie.optOut
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
func (cookie *PBSCookie) ToHTTPCookie(ttl time.Duration) *http.Cookie {
	j, _ := json.Marshal(cookie)
	b64 := base64.URLEncoding.EncodeToString(j)

	return &http.Cookie{
		Name:    UID_COOKIE_NAME,
		Value:   b64,
		Expires: time.Now().Add(ttl),
		Path:    "/",
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

// GetUIDs returns this user's ID for all the bidders
func (cookie *PBSCookie) GetUIDs() map[string]string {
	uids := make(map[string]string)
	if cookie != nil {
		// Extract just the uid for each bidder
		for bidderName, uidWithExpiry := range cookie.uids {
			uids[bidderName] = uidWithExpiry.UID
		}
	}
	return uids
}

// GetId wraps GetUID, letting callers fetch the ID given an OpenRTB BidderName.
func (cookie *PBSCookie) GetId(bidderName openrtb_ext.BidderName) (id string, exists bool) {
	if familyName, ok := bidderToFamilyNames[bidderName]; ok {
		id, exists, _ = cookie.GetUID(familyName)
	} else {
		id, exists, _ = cookie.GetUID(string(bidderName))
	}
	return
}

// SetCookieOnResponse is a shortcut for "ToHTTPCookie(); cookie.setDomain(domain); setCookie(w, cookie)"
func (cookie *PBSCookie) SetCookieOnResponse(w http.ResponseWriter, setSiteCookie bool, cfg *config.HostCookie, ttl time.Duration) {
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

	var uidsCookieStr string
	var sameSiteCookie *http.Cookie
	if setSiteCookie {
		httpCookie.Secure = true
		uidsCookieStr = httpCookie.String()
		uidsCookieStr += SameSiteAttribute
		sameSiteCookie = &http.Cookie{
			Name:    SameSiteCookieName,
			Value:   SameSiteCookieValue,
			Expires: time.Now().Add(ttl),
			Path:    "/",
			Secure:  true,
		}
		sameSiteCookieStr := sameSiteCookie.String()
		sameSiteCookieStr += SameSiteAttribute
		w.Header().Add("Set-Cookie", sameSiteCookieStr)
	} else {
		uidsCookieStr = httpCookie.String()
	}
	w.Header().Add("Set-Cookie", uidsCookieStr)
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
	if familyName == string(openrtb_ext.BidderAudienceNetwork) && uid == "0" {
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
			if id, ok := cookie.uids[string(openrtb_ext.BidderAudienceNetwork)]; ok && id.UID == "0" {
				delete(cookie.uids, string(openrtb_ext.BidderAudienceNetwork))
			}
		}
	}
	return err
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
