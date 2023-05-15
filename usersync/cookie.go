package usersync

import (
	"encoding/json"
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
	uids     map[string]UIDEntry
	optOut   bool
	birthday *time.Time
}

// UIDEntry bundles the UID with an Expiration date.
// After the expiration, the UID is no longer valid.
type UIDEntry struct {
	// UID is the ID given to a user by a particular bidder
	UID string `json:"uid"`
	// Expires is the time at which this UID should no longer apply.
	Expires time.Time `json:"expires"`
}

// ReadCookie will replace ParseCookieFromRequest
func ReadCookie(r *http.Request) *Cookie {
	//TODO: ParseCookieFromRequest OptOut Logic?

	cookieFromRequest, err := r.Cookie(uidCookieName)
	if err != nil {
		return NewCookie()
	}

	decoder := DecodeV1{}
	decodedCookie := decoder.Decode(cookieFromRequest.Value)

	return decodedCookie
}

// WriteCookie
func WriteCookie(cookie *Cookie, ttl time.Duration, w http.ResponseWriter, setSiteCookie bool, domain string) {
	encoder := EncoderV1{}
	b64 := encoder.Encode(cookie)

	httpCookie := &http.Cookie{
		Name:    uidCookieName,
		Value:   b64,
		Expires: time.Now().Add(ttl),
		Path:    "/",
	}

	if domain != "" {
		httpCookie.Domain = domain
	}

	if setSiteCookie {
		httpCookie.Secure = true
		httpCookie.SameSite = http.SameSiteNoneMode
	}

	w.Header().Add("Set-Cookie", httpCookie.String())
}

// NewCookie returns a new empty cookie.
func NewCookie() *Cookie {
	return &Cookie{
		uids:     make(map[string]UIDEntry),
		birthday: timestamp(),
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
		cookie.uids = make(map[string]UIDEntry)
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

func (cookie *Cookie) SetCookieOnResponse(w http.ResponseWriter, setSiteCookie bool, cfg *config.HostCookie, ttl time.Duration) {
	encoder := EncoderV1{}
	encodedCookie := encoder.Encode(cookie)

	isCookieTooBig := len(encodedCookie) > cfg.MaxCookieSizeBytes && cfg.MaxCookieSizeBytes > 0

	for isCookieTooBig && len(cookie.uids) > 0 {
		uidToDelete, err := ejector.Choose(cookie.uids)
		if err != nil {
			return err
		}
		delete(cookie.uids, uidToDelete)
		encodedCookie = encoder.Encode(cookie)
		isCookieTooBig = len(encodedCookie) > cfg.MaxCookieSizeBytes && cfg.MaxCookieSizeBytes > 0
	}
	WriteCookie(cookie, ttl, w, setSiteCookie, cfg.Domain)
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

// Sync tries to set the UID for some syncer key. It returns an error if the set didn't happen.
func (cookie *Cookie) Sync(key string, uid string) error {
	cookie.uids[key] = UIDEntry{
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
	LegacyUIDs map[string]string   `json:"uids,omitempty"`
	UIDs       map[string]UIDEntry `json:"tempUIDs,omitempty"`
	OptOut     bool                `json:"optout,omitempty"`
	Birthday   *time.Time          `json:"bday,omitempty"`
}

func (cookie *Cookie) MarshalJSON() ([]byte, error) {
	return json.Marshal(cookieJson{
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
func (cookie *Cookie) UnmarshalJSON(b []byte) error {
	var cookieContract cookieJson
	err := json.Unmarshal(b, &cookieContract)
	if err == nil {
		cookie.optOut = cookieContract.OptOut
		cookie.birthday = cookieContract.Birthday

		if cookie.optOut {
			cookie.uids = make(map[string]UIDEntry)
		} else {
			cookie.uids = cookieContract.UIDs

			if cookie.uids == nil {
				cookie.uids = make(map[string]UIDEntry, len(cookieContract.LegacyUIDs))
			}

			// Interpret "legacy" UIDs as having been expired already.
			// This should cause us to re-sync, since it would be time for a new one.
			for bidder, uid := range cookieContract.LegacyUIDs {
				if _, ok := cookie.uids[bidder]; !ok {
					cookie.uids[bidder] = UIDEntry{
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

func timestamp() *time.Time {
	birthday := time.Now()
	return &birthday
}
