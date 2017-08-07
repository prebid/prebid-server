package pbs

import (
	"encoding/base64"
	"net/http"
	"testing"
	"net/http/httptest"
)

func TestOptOutCookie(t *testing.T) {
	cookie := &cookieImpl{
		UIDs:     make(map[string]string, 0),
		OptOut:   true,
		Birthday: timestamp(),
	}
	ensureConsistency(t, cookie)
}

func TestEmptyCookie(t *testing.T) {
	cookie := &cookieImpl{
		UIDs:     make(map[string]string, 0),
		OptOut:   false,
		Birthday: timestamp(),
	}
	ensureConsistency(t, cookie)
}

func TestCookieWithData(t *testing.T) {
	cookie := &cookieImpl{
		UIDs: map[string]string{
			"adnxs":           "123",
			"audienceNetwork": "456",
		},
		OptOut:   false,
		Birthday: timestamp(),
	}
	ensureConsistency(t, cookie)
}

func TestRejectAudienceNetworkCookie(t *testing.T) {
	raw := &cookieImpl{
		UIDs: map[string]string{
			"audienceNetwork": "0",
		},
		OptOut:   false,
		Birthday: timestamp(),
	}
	parsed := ParseCookie(raw.ToHTTPCookie())
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
		UIDs: map[string]string{
			"adnxs":           "123",
			"audienceNetwork": "456",
		},
		OptOut:   false,
		Birthday: timestamp(),
	}

	cookie.SetPreference(false)
	if cookie.IsAllowed() {
		t.Error("After SetPreference(false), a cookie should not allow more user syncs.")
	}
	ensureConsistency(t, cookie)
}

func TestOptIn(t *testing.T) {
	cookie := &cookieImpl{
		UIDs:     map[string]string{},
		OptOut:   true,
		Birthday: timestamp(),
	}

	cookie.SetPreference(true)
	if !cookie.IsAllowed() {
		t.Error("After SetPreference(true), a cookie should allow more user syncs.")
	}
	ensureConsistency(t, cookie)
}

func TestParseCorruptedCookie(t *testing.T) {
	raw := http.Cookie{
		Name:  "uids",
		Value: "bad base64 encoding",
	}
	parsed := ParseCookie(&raw)
	ensureEmptyCookie(t, parsed)
}

func TestParseCorruptedCookieJSON(t *testing.T) {
	cookieData := base64.URLEncoding.EncodeToString([]byte("bad json"))
	raw := http.Cookie{
		Name:  "uids",
		Value: cookieData,
	}
	parsed := ParseCookie(&raw)
	ensureEmptyCookie(t, parsed)
}

func writeThenRead(t *testing.T, cookie UserSyncCookie) UserSyncCookie {
	w := httptest.NewRecorder()
	cookie.SetCookieOnResponse(w, "mock-domain")
	writtenCookie := w.HeaderMap.Get("Set-Cookie")

	header := http.Header{}
	header.Add("Cookie", writtenCookie)
	request := http.Request{Header: header}
	return ParseCookieFromRequest(&request)
}

func TestCookieReadWrite(t *testing.T) {
	cookie := &cookieImpl{
		UIDs: map[string]string{
			"adnxs":           "123",
			"audienceNetwork": "456",
		},
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

func ensureEmptyCookie(t *testing.T, cookie UserSyncCookie) {
	if !cookie.IsAllowed() {
		t.Error("Empty cookies should allow user syncs.")
	}
	if cookie.SyncCount() != 0 {
		t.Errorf("Empty cookies shouldn't have any user syncs. Found %d.", cookie.SyncCount())
	}
}

func ensureConsistency(t *testing.T, cookie UserSyncCookie) {
	if cookie.IsAllowed() {
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
	if cookieImpl.OptOut == cookie.IsAllowed() {
		t.Error("The cookieImpl interface shouldn't let modifications happen if the user has opted out")
	}
	if cookie.SyncCount() != len(cookieImpl.UIDs) {
		t.Errorf("Incorrect sync count. Expected %d, got %d", len(cookieImpl.UIDs), cookie.SyncCount())
	}

	for family, uid := range cookieImpl.UIDs {
		if !cookie.HasSync(family) {
			t.Errorf("Cookie is missing sync for family %s", family)
		}
		savedUID, hadSync := cookie.GetUID(family)
		if !hadSync {
			t.Error("The GetUID function should return true when it has a sync. Got false")
		}
		if savedUID != uid {
			t.Errorf("Wrong UID saved for family %s. Expected %s, got %s", family, uid, savedUID)
		}
	}
}
