package usersync

import (
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
// To get an instance of this from a request, use ReadCookie.
// To write an instance onto a response, use WriteCookie.
type Cookie struct {
	uids     map[string]UIDEntry
	optOut   bool
	birthday *time.Time
}

// UIDEntry bundles the UID with an Expiration date.
type UIDEntry struct {
	// UID is the ID given to a user by a particular bidder
	UID string `json:"uid"`
	// Expires is the time at which this UID should no longer apply.
	Expires time.Time `json:"expires"`
}

// NewCookie returns a new empty cookie.
func NewCookie() *Cookie {
	return &Cookie{
		uids:     make(map[string]UIDEntry),
		birthday: timestamp(),
	}
}

// ReadCookie reads the cookie from the request
func ReadCookie(r *http.Request, decoder Decoder, host *config.HostCookie) *Cookie {
	if hostOptOutCookie := checkHostCookieOptOut(r, host); hostOptOutCookie != nil {
		return hostOptOutCookie
	}

	// Read cookie from request
	cookieFromRequest, err := r.Cookie(uidCookieName)
	if err != nil {
		return NewCookie()
	}
	decodedCookie := decoder.Decode(cookieFromRequest.Value)

	return decodedCookie
}

// PrepareCookieForWrite ejects UIDs as long as the cookie is too full
func (cookie *Cookie) PrepareCookieForWrite(cfg *config.HostCookie, ttl time.Duration, encoder Encoder) string {
	encodedCookie := encoder.Encode(cookie)

	isCookieTooBig := len(encodedCookie) > cfg.MaxCookieSizeBytes && cfg.MaxCookieSizeBytes > 0

	for isCookieTooBig && len(cookie.uids) > 0 {
		uidToDelete := getOldestUID(cookie)
		delete(cookie.uids, uidToDelete)

		encodedCookie = encoder.Encode(cookie)
		isCookieTooBig = len(encodedCookie) > cfg.MaxCookieSizeBytes && cfg.MaxCookieSizeBytes > 0
	}
	return encodedCookie
}

// WriteCookie sets the prepared cookie onto the header
func WriteCookie(w http.ResponseWriter, encodedCookie string, cfg *config.HostCookie, setSiteCookie bool) {
	ttl := cfg.TTLDuration()

	httpCookie := &http.Cookie{
		Name:    uidCookieName,
		Value:   encodedCookie,
		Expires: time.Now().Add(ttl),
		Path:    "/",
	}

	if cfg.Domain != "" {
		httpCookie.Domain = cfg.Domain
	}

	if setSiteCookie {
		httpCookie.Secure = true
		httpCookie.SameSite = http.SameSiteNoneMode
	}

	w.Header().Add("Set-Cookie", httpCookie.String())
}

// Sync tries to set the UID for some syncer key. It returns an error if the set didn't happen.
func (cookie *Cookie) Sync(key string, uid string) error {
	if !cookie.AllowSyncs() {
		return errors.New("the user has opted out of prebid server cookie syncs")
	}

	if checkAudienceNetwork(key, uid) {
		return errors.New("audienceNetwork uses a UID of 0 as \"not yet recognized\"")
	}

	// Sync
	cookie.uids[key] = UIDEntry{
		UID:     uid,
		Expires: time.Now().Add(uidTTL),
	}

	return nil
}

// getOldestUID grabs the oldest UID from the cookie, so that it can be ejected.
// this will be replaced with more complex ejection framework in a future PR
func getOldestUID(cookie *Cookie) string {
	oldestElem := ""
	var oldestDate int64 = math.MaxInt64
	for key, value := range cookie.uids {
		timeUntilExpiration := time.Until(value.Expires)
		if timeUntilExpiration < time.Duration(oldestDate) {
			oldestElem = key
			oldestDate = int64(timeUntilExpiration)
		}
	}
	return oldestElem
}

func SyncHostCookie(r *http.Request, requestCookie *Cookie, host *config.HostCookie) {
	if uid, _, _ := requestCookie.GetUID(host.Family); uid == "" && host.CookieName != "" {
		if hostCookie, err := r.Cookie(host.CookieName); err == nil {
			requestCookie.Sync(host.Family, hostCookie.Value)
		}
	}
}

func checkHostCookieOptOut(r *http.Request, host *config.HostCookie) *Cookie {
	if host.OptOutCookie.Name != "" {
		optOutCookie, err := r.Cookie(host.OptOutCookie.Name)
		if err == nil && optOutCookie.Value == host.OptOutCookie.Value {
			hostOptOut := NewCookie()
			hostOptOut.SetOptOut(true)
			return hostOptOut
		}
	}
	return nil
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

func (cookie *Cookie) ToHTTPCookie() *http.Cookie {
	encoder := EncoderV1{}
	encodedCookie := encoder.Encode(cookie)

	return &http.Cookie{
		Name:    uidCookieName,
		Value:   encodedCookie,
		Expires: time.Now().Add((90 * 24 * time.Hour)),
		Path:    "/",
	}
}

func checkAudienceNetwork(key string, uid string) bool {
	return key == string(openrtb_ext.BidderAudienceNetwork) && uid == "0"
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
