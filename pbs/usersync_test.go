package pbs

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOptOutCookie(t *testing.T) {
	cookie := &cookieImpl{
		UIDs:          nil,
		TemporaryUIDs: make(map[string]temporaryUid),
		OptOut:        true,
		Birthday:      timestamp(),
	}
	ensureConsistency(t, cookie)
}

func TestEmptyOptOutCookie(t *testing.T) {
	cookie := &cookieImpl{
		UIDs:          nil,
		TemporaryUIDs: make(map[string]temporaryUid),
		OptOut:        true,
		Birthday:      timestamp(),
	}
	ensureConsistency(t, cookie)
}

func TestEmptyCookie(t *testing.T) {
	cookie := &cookieImpl{
		UIDs:          nil,
		TemporaryUIDs: make(map[string]temporaryUid),
		OptOut:        false,
		Birthday:      timestamp(),
	}
	ensureConsistency(t, cookie)
}

func TestCookieWithData(t *testing.T) {
	cookie := &cookieImpl{
		UIDs: nil,
		TemporaryUIDs: attachTTLs(map[string]string{
			"adnxs":           "123",
			"audienceNetwork": "456",
		}),
		OptOut:   false,
		Birthday: timestamp(),
	}
	ensureConsistency(t, cookie)
}

func TestRejectAudienceNetworkCookie(t *testing.T) {
	raw := &cookieImpl{
		// Starting with UIDs here because Facebook IDs of 0 should be impossible in TemporaryUIDs
		UIDs: map[string]string{
			"audienceNetwork": "0",
		},
		OptOut:   false,
		Birthday: timestamp(),
	}
	parsed := ParseUserSyncMap(raw.ToHTTPCookie())
	if parsed.HasSync("audienceNetwork") {
		t.Errorf("Cookie serializing and deserializing should delete audienceNetwork values of 0")
	}

	err := parsed.TrySync("audienceNetwork", "0")
	if err == nil {
		t.Errorf("Cookie should reject audienceNetwork values of 0.")
	}
	if parsed.HasSync("audienceNetwork") {
		t.Errorf("Cookie The cookie should have rejected the audienceNetwork sync.")
	}
}

func TestOptOutReset(t *testing.T) {
	cookie := &cookieImpl{
		UIDs: nil,
		TemporaryUIDs: attachTTLs(map[string]string{
			"adnxs":           "123",
			"audienceNetwork": "456",
		}),
		OptOut:   false,
		Birthday: timestamp(),
	}

	cookie.SetPreference(false)
	if cookie.AllowSyncs() {
		t.Error("After SetPreference(false), a cookie should not allow more user syncs.")
	}
	ensureConsistency(t, cookie)
}

func TestOptIn(t *testing.T) {
	cookie := &cookieImpl{
		UIDs:          nil,
		TemporaryUIDs: make(map[string]temporaryUid),
		OptOut:        true,
		Birthday:      timestamp(),
	}

	cookie.SetPreference(true)
	if !cookie.AllowSyncs() {
		t.Error("After SetPreference(true), a cookie should allow more user syncs.")
	}
	ensureConsistency(t, cookie)
}

func TestParseCorruptedCookie(t *testing.T) {
	raw := http.Cookie{
		Name:  "uids",
		Value: "bad base64 encoding",
	}
	parsed := ParseUserSyncMap(&raw)
	ensureEmptyMap(t, parsed)
}

func TestParseCorruptedCookieJSON(t *testing.T) {
	cookieData := base64.URLEncoding.EncodeToString([]byte("bad json"))
	raw := http.Cookie{
		Name:  "uids",
		Value: cookieData,
	}
	parsed := ParseUserSyncMap(&raw)
	ensureEmptyMap(t, parsed)
}

func TestParseNilSyncMap(t *testing.T) {
	cookieJSON := "{\"bday\":123,\"optout\":true}"
	cookieData := base64.URLEncoding.EncodeToString([]byte(cookieJSON))
	raw := http.Cookie{
		Name:  COOKIE_NAME,
		Value: cookieData,
	}
	parsed := ParseUserSyncMap(&raw)
	ensureEmptyMap(t, parsed)
	ensureConsistency(t, parsed)
}

func writeThenRead(t *testing.T, cookie UserSyncMap) UserSyncMap {
	w := httptest.NewRecorder()
	cookie.SetCookieOnResponse(w, "mock-domain")
	writtenCookie := w.HeaderMap.Get("Set-Cookie")

	header := http.Header{}
	header.Add("Cookie", writtenCookie)
	request := http.Request{Header: header}
	return ParseUserSyncMapFromRequest(&request)
}

func TestCookieReadWrite(t *testing.T) {
	cookie := &cookieImpl{
		UIDs: nil,
		TemporaryUIDs: attachTTLs(map[string]string{
			"adnxs":           "123",
			"audienceNetwork": "456",
		}),
		OptOut:   false,
		Birthday: timestamp(),
	}

	received := writeThenRead(t, cookie)
	uid, exists := received.GetUID("adnxs")
	if !exists || uid != "123" {
		t.Errorf("Received cookie should have the adnxs ID=123. Got %s", uid)
	}
	uid, exists = received.GetUID("audienceNetwork")
	if !exists || uid != "456" {
		t.Errorf("Received cookie should have the audienceNetwork ID=456. Got %s", uid)
	}
	if received.SyncCount() != 2 {
		t.Errorf("Expected 2 user syncs. Got %d", received.SyncCount())
	}
}

func TestLegacyCookieReadWrite(t *testing.T) {
	cookie := &cookieImpl{
		UIDs: map[string]string{
			"adnxs":           "123",
			"audienceNetwork": "456",
		},
		TemporaryUIDs: make(map[string]temporaryUid),
		OptOut:        false,
		Birthday:      timestamp(),
	}

	received := writeThenRead(t, cookie)
	if received.SyncCount() != 0 {
		t.Errorf("Expected 0 user syncs. Got %d", received.SyncCount())
	}
	if received.HasSync("adnxs") {
		t.Errorf("Received cookie should act like it has no ID for adnxs.")
	}
	if received.HasSync("audienceNetwork") {
		t.Errorf("Received cookie should act like it has no ID for audienceNetwork.")
	}
}

func ensureEmptyMap(t *testing.T, cookie UserSyncMap) {
	if !cookie.AllowSyncs() {
		t.Error("Empty cookies should allow user syncs.")
	}
	if cookie.SyncCount() != 0 {
		t.Errorf("Empty cookies shouldn't have any user syncs. Found %d.", cookie.SyncCount())
	}
}

func ensureConsistency(t *testing.T, cookie UserSyncMap) {
	if cookie.AllowSyncs() {
		err := cookie.TrySync("pulsepoint", "1")
		if err != nil {
			t.Errorf("Cookie sync should succeed if the user has opted in.")
		}
		if !cookie.HasSync("pulsepoint") {
			t.Errorf("The cookieImpl should have a usersync after a successful call to TrySync")
		}
		savedUID, hadSync := cookie.GetUID("pulsepoint")
		if !hadSync {
			t.Error("The GetUID function should return true when it has a sync. Got false")
		}
		if savedUID != "1" {
			t.Errorf("The cookieImpl isn't saving syncs correctly. Expected %s, got %s", "1", savedUID)
		}
		cookie.Unsync("pulsepoint")
		if cookie.HasSync("pulsepoint") {
			t.Errorf("The cookieImpl should not have have a usersync after a call to Unsync")
		}
		if value, hadValue := cookie.GetUID("pulsepoint"); value != "" || hadValue {
			t.Error("cookieImpl.GetUID() should return empty strings if it doesn't have a sync")
		}
	} else {
		if cookie.SyncCount() != 0 {
			t.Errorf("If the user opted out, the cookieImpl should have no user syncs. Got %d", cookie.SyncCount())
		}

		err := cookie.TrySync("adnxs", "123")
		if err == nil {
			t.Error("TrySync should fail if the user has opted out of cookieImpl syncs, but it succeeded.")
		}
	}

	cookieImpl := parseCookieImpl(cookie.ToHTTPCookie())
	if cookieImpl.OptOut == cookie.AllowSyncs() {
		t.Error("The cookieImpl interface shouldn't let modifications happen if the user has opted out")
	}
	if cookie.SyncCount() != len(cookieImpl.TemporaryUIDs) {
		t.Errorf("Incorrect sync count. Expected %d, got %d", len(cookieImpl.TemporaryUIDs), cookie.SyncCount())
	}

	for family, uid := range cookieImpl.TemporaryUIDs {
		if !cookie.HasSync(family) {
			t.Errorf("Cookie is missing sync for family %s", family)
		}
		savedUID, hadSync := cookie.GetUID(family)
		if !hadSync {
			t.Error("The GetUID function should return true when it has a sync. Got false")
		}
		if savedUID != uid.UID {
			t.Errorf("Wrong UID saved for family %s. Expected %s, got %s", family, uid.UID, savedUID)
		}
	}
}

func attachTTLs(uids map[string]string) map[string]temporaryUid {
	idMap := make(map[string]temporaryUid, len(uids))
	for familyName, uid := range uids {
		idMap[familyName] = temporaryUid{
			UID:     uid,
			Expires: getExpiry(familyName),
		}
	}
	return idMap
}
