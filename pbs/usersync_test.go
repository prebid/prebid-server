package pbs

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOptOutCookie(t *testing.T) {
	cookie := &PBSCookie{
		uids:     make(map[string]uidWithExpiry),
		optOut:   true,
		birthday: timestamp(),
	}
	ensureConsistency(t, cookie)
}

func TestEmptyOptOutCookie(t *testing.T) {
	cookie := &PBSCookie{
		uids:     make(map[string]uidWithExpiry),
		optOut:   true,
		birthday: timestamp(),
	}
	ensureConsistency(t, cookie)
}

func TestEmptyCookie(t *testing.T) {
	cookie := &PBSCookie{
		uids:     make(map[string]uidWithExpiry, 0),
		optOut:   false,
		birthday: timestamp(),
	}
	ensureConsistency(t, cookie)
}

func TestCookieWithData(t *testing.T) {
	cookie := &PBSCookie{
		uids: map[string]uidWithExpiry{
			"adnxs":           newTempId("123"),
			"audienceNetwork": newTempId("456"),
		},
		optOut:   false,
		birthday: timestamp(),
	}
	ensureConsistency(t, cookie)
}

func TestRejectAudienceNetworkCookie(t *testing.T) {
	raw := &PBSCookie{
		uids: map[string]uidWithExpiry{
			"audienceNetwork": newTempId("0"),
		},
		optOut:   false,
		birthday: timestamp(),
	}
	parsed := ParsePBSCookie(raw.ToHTTPCookie())
	if parsed.HasLiveSync("audienceNetwork") {
		t.Errorf("Cookie serializing and deserializing should delete audienceNetwork values of 0")
	}

	err := parsed.TrySync("audienceNetwork", "0")
	if err == nil {
		t.Errorf("Cookie should reject audienceNetwork values of 0.")
	}
	if parsed.HasLiveSync("audienceNetwork") {
		t.Errorf("Cookie The cookie should have rejected the audienceNetwork sync.")
	}
}

func TestOptOutReset(t *testing.T) {
	cookie := &PBSCookie{
		uids: map[string]uidWithExpiry{
			"adnxs":           newTempId("123"),
			"audienceNetwork": newTempId("456"),
		},
		optOut:   false,
		birthday: timestamp(),
	}

	cookie.SetPreference(false)
	if cookie.AllowSyncs() {
		t.Error("After SetPreference(false), a cookie should not allow more user syncs.")
	}
	ensureConsistency(t, cookie)
}

func TestOptIn(t *testing.T) {
	cookie := &PBSCookie{
		uids:     make(map[string]uidWithExpiry),
		optOut:   true,
		birthday: timestamp(),
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
	parsed := ParsePBSCookie(&raw)
	ensureEmptyMap(t, parsed)
}

func TestParseCorruptedCookieJSON(t *testing.T) {
	cookieData := base64.URLEncoding.EncodeToString([]byte("bad json"))
	raw := http.Cookie{
		Name:  "uids",
		Value: cookieData,
	}
	parsed := ParsePBSCookie(&raw)
	ensureEmptyMap(t, parsed)
}

func TestParseNilSyncMap(t *testing.T) {
	cookieJSON := "{\"bday\":123,\"optout\":true}"
	cookieData := base64.URLEncoding.EncodeToString([]byte(cookieJSON))
	raw := http.Cookie{
		Name:  COOKIE_NAME,
		Value: cookieData,
	}
	parsed := ParsePBSCookie(&raw)
	ensureEmptyMap(t, parsed)
	ensureConsistency(t, parsed)
}

func TestCookieReadWrite(t *testing.T) {
	cookie := &PBSCookie{
		uids: map[string]uidWithExpiry{
			"adnxs":           newTempId("123"),
			"audienceNetwork": newTempId("456"),
		},
		optOut:   false,
		birthday: timestamp(),
	}

	received := writeThenRead(cookie)
	uid, exists, isLive := received.GetUID("adnxs")
	if !exists || !isLive || uid != "123" {
		t.Errorf("Received cookie should have the adnxs ID=123. Got %s", uid)
	}
	uid, exists, isLive = received.GetUID("audienceNetwork")
	if !exists || !isLive || uid != "456" {
		t.Errorf("Received cookie should have the audienceNetwork ID=456. Got %s", uid)
	}
	if received.LiveSyncCount() != 2 {
		t.Errorf("Expected 2 user syncs. Got %d", received.LiveSyncCount())
	}
}

func TestPopulatedLegacyCookieRead(t *testing.T) {
	legacyJson := `{"uids":{"adnxs":"123","audienceNetwork":"456"},"bday":"2017-08-03T21:04:52.629198911Z"}`
	var cookie PBSCookie
	json.Unmarshal([]byte(legacyJson), &cookie)

	if cookie.LiveSyncCount() != 0 {
		t.Errorf("Expected 0 user syncs. Got %d", cookie.LiveSyncCount())
	}
	if cookie.HasLiveSync("adnxs") {
		t.Errorf("Received cookie should act like it has no ID for adnxs.")
	}
	if cookie.HasLiveSync("audienceNetwork") {
		t.Errorf("Received cookie should act like it has no ID for audienceNetwork.")
	}
}

func TestEmptyLegacyCookieRead(t *testing.T) {
	legacyJson := `{"bday":"2017-08-29T18:54:18.393925772Z"}`
	var cookie PBSCookie
	json.Unmarshal([]byte(legacyJson), &cookie)

	if cookie.LiveSyncCount() != 0 {
		t.Errorf("Expected 0 user syncs. Got %d", cookie.LiveSyncCount())
	}
}

func TestNilCookie(t *testing.T) {
	var nilCookie *PBSCookie = nil

	if nilCookie.HasLiveSync("anything") {
		t.Error("nil cookies should respond with false when asked if they have a sync")
	}

	if nilCookie.LiveSyncCount() != 0 {
		t.Error("nil cookies shouldn't have any syncs.")
	}

	if nilCookie.AllowSyncs() {
		t.Error("nil cookies shouldn't allow syncs to take place.")
	}

	uid, hadUID, isLive := nilCookie.GetUID("anything")

	if uid != "" {
		t.Error("nil cookies should return empty strings for the UID.")
	}
	if hadUID {
		t.Error("nil cookies shouldn't claim to have a UID mapping.")
	}
	if isLive {
		t.Error("nil cookies shouldn't report live UID mappings.")
	}
}

func ensureEmptyMap(t *testing.T, cookie *PBSCookie) {
	if !cookie.AllowSyncs() {
		t.Error("Empty cookies should allow user syncs.")
	}
	if cookie.LiveSyncCount() != 0 {
		t.Errorf("Empty cookies shouldn't have any user syncs. Found %d.", cookie.LiveSyncCount())
	}
}

func ensureConsistency(t *testing.T, cookie *PBSCookie) {
	if cookie.AllowSyncs() {
		err := cookie.TrySync("pulsepoint", "1")
		if err != nil {
			t.Errorf("Cookie sync should succeed if the user has opted in.")
		}
		if !cookie.HasLiveSync("pulsepoint") {
			t.Errorf("The PBSCookie should have a usersync after a successful call to TrySync")
		}
		savedUID, hadSync, isLive := cookie.GetUID("pulsepoint")
		if !hadSync {
			t.Error("The GetUID function should properly report that it has a sync.")
		}
		if !isLive {
			t.Error("The GetUID function should properly report live syncs.")
		}
		if savedUID != "1" {
			t.Errorf("The PBSCookie isn't saving syncs correctly. Expected %s, got %s", "1", savedUID)
		}
		cookie.Unsync("pulsepoint")
		if cookie.HasLiveSync("pulsepoint") {
			t.Errorf("The PBSCookie should not have have a usersync after a call to Unsync")
		}
		if value, hadValue, isLive := cookie.GetUID("pulsepoint"); value != "" || hadValue || isLive {
			t.Error("PBSCookie.GetUID() should return empty strings if it doesn't have a sync")
		}
	} else {
		if cookie.LiveSyncCount() != 0 {
			t.Errorf("If the user opted out, the PBSCookie should have no user syncs. Got %d", cookie.LiveSyncCount())
		}

		err := cookie.TrySync("adnxs", "123")
		if err == nil {
			t.Error("TrySync should fail if the user has opted out of PBSCookie syncs, but it succeeded.")
		}
	}

	copiedCookie := ParsePBSCookie(cookie.ToHTTPCookie())
	if copiedCookie.AllowSyncs() != cookie.AllowSyncs() {
		t.Error("The PBSCookie interface shouldn't let modifications happen if the user has opted out")
	}
	if cookie.LiveSyncCount() != copiedCookie.LiveSyncCount() {
		t.Errorf("Incorrect sync count. Expected %d, got %d", copiedCookie.LiveSyncCount(), cookie.LiveSyncCount())
	}

	for family, uid := range copiedCookie.uids {
		if !cookie.HasLiveSync(family) {
			t.Errorf("Cookie is missing sync for family %s", family)
		}
		savedUID, hadSync, isLive := cookie.GetUID(family)
		if !hadSync {
			t.Error("The GetUID function should properly report that it has a sync.")
		}
		if !isLive {
			t.Error("The GetUID function should properly report live syncs.")
		}
		if savedUID != uid.UID {
			t.Errorf("Wrong UID saved for family %s. Expected %s, got %s", family, uid, savedUID)
		}
	}
}

func writeThenRead(cookie *PBSCookie) *PBSCookie {
	w := httptest.NewRecorder()
	cookie.SetCookieOnResponse(w, "mock-domain")
	writtenCookie := w.HeaderMap.Get("Set-Cookie")

	header := http.Header{}
	header.Add("Cookie", writtenCookie)
	request := http.Request{Header: header}
	return ParsePBSCookieFromRequest(&request)
}

func newTempId(uid string) uidWithExpiry {
	return uidWithExpiry{
		UID:     uid,
		Expires: time.Now().Add(10 * time.Minute),
	}
}
