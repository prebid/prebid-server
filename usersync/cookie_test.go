package usersync

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prebid/prebid-server/config"
	"github.com/stretchr/testify/assert"
)

func TestOptOutCookie(t *testing.T) {
	cookie := &Cookie{
		uids:     make(map[string]uidWithExpiry),
		optOut:   true,
		birthday: timestamp(),
	}
	ensureConsistency(t, cookie)
}

func TestEmptyOptOutCookie(t *testing.T) {
	cookie := &Cookie{
		uids:     make(map[string]uidWithExpiry),
		optOut:   true,
		birthday: timestamp(),
	}
	ensureConsistency(t, cookie)
}

func TestEmptyCookie(t *testing.T) {
	cookie := &Cookie{
		uids:     make(map[string]uidWithExpiry),
		optOut:   false,
		birthday: timestamp(),
	}
	ensureConsistency(t, cookie)
}

func TestCookieWithData(t *testing.T) {
	cookie := newSampleCookie()
	ensureConsistency(t, cookie)
}

func TestBidderNameGets(t *testing.T) {
	cookie := newSampleCookie()
	id, exists, _ := cookie.GetUID("adnxs")
	if !exists {
		t.Errorf("Cookie missing expected Appnexus ID")
	}
	if id != "123" {
		t.Errorf("Bad appnexus id. Expected %s, got %s", "123", id)
	}

	id, exists, _ = cookie.GetUID("rubicon")
	if !exists {
		t.Errorf("Cookie missing expected Rubicon ID")
	}
	if id != "456" {
		t.Errorf("Bad rubicon id. Expected %s, got %s", "456", id)
	}
}

func TestRejectAudienceNetworkCookie(t *testing.T) {
	raw := &Cookie{
		uids: map[string]uidWithExpiry{
			"audienceNetwork": newTempId("0", 10),
		},
		optOut:   false,
		birthday: timestamp(),
	}
	parsed := ParseCookie(raw.ToHTTPCookie(90 * 24 * time.Hour))
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
	cookie := newSampleCookie()

	cookie.SetOptOut(true)
	if cookie.AllowSyncs() {
		t.Error("After SetOptOut(true), a cookie should not allow more user syncs.")
	}
	ensureConsistency(t, cookie)
}

func TestOptIn(t *testing.T) {
	cookie := &Cookie{
		uids:     make(map[string]uidWithExpiry),
		optOut:   true,
		birthday: timestamp(),
	}

	cookie.SetOptOut(false)
	if !cookie.AllowSyncs() {
		t.Error("After SetOptOut(false), a cookie should allow more user syncs.")
	}
	ensureConsistency(t, cookie)
}

func TestParseCorruptedCookie(t *testing.T) {
	raw := http.Cookie{
		Name:  "uids",
		Value: "bad base64 encoding",
	}
	parsed := ParseCookie(&raw)
	ensureEmptyMap(t, parsed)
}

func TestParseCorruptedCookieJSON(t *testing.T) {
	cookieData := base64.URLEncoding.EncodeToString([]byte("bad json"))
	raw := http.Cookie{
		Name:  "uids",
		Value: cookieData,
	}
	parsed := ParseCookie(&raw)
	ensureEmptyMap(t, parsed)
}

func TestParseNilSyncMap(t *testing.T) {
	cookieJSON := "{\"bday\":123,\"optout\":true}"
	cookieData := base64.URLEncoding.EncodeToString([]byte(cookieJSON))
	raw := http.Cookie{
		Name:  uidCookieName,
		Value: cookieData,
	}
	parsed := ParseCookie(&raw)
	ensureEmptyMap(t, parsed)
	ensureConsistency(t, parsed)
}

func TestParseOtherCookie(t *testing.T) {
	req := httptest.NewRequest("POST", "http://www.prebid.com", nil)
	otherCookieName := "other"
	id := "some-user-id"
	req.AddCookie(&http.Cookie{
		Name:  otherCookieName,
		Value: id,
	})
	parsed := ParseCookieFromRequest(req, &config.HostCookie{
		Family:     "adnxs",
		CookieName: otherCookieName,
	})
	val, _, _ := parsed.GetUID("adnxs")
	if val != id {
		t.Errorf("Bad cookie value. Expected %s, got %s", id, val)
	}
}

func TestParseCookieFromRequestOptOut(t *testing.T) {
	optOutCookieName := "optOutCookieName"
	optOutCookieValue := "optOutCookieValue"

	existingCookie := *(&Cookie{
		uids: map[string]uidWithExpiry{
			"foo": newTempId("fooID", 1),
			"bar": newTempId("barID", 2),
		},
		optOut:   false,
		birthday: timestamp(),
	}).ToHTTPCookie(24 * time.Hour)

	testCases := []struct {
		description          string
		givenExistingCookies []http.Cookie
		expectedEmpty        bool
		expectedSetOptOut    bool
	}{
		{
			description: "Opt Out Cookie",
			givenExistingCookies: []http.Cookie{
				existingCookie,
				{Name: optOutCookieName, Value: optOutCookieValue}},
			expectedEmpty:     true,
			expectedSetOptOut: true,
		},
		{
			description: "No Opt Out Cookie",
			givenExistingCookies: []http.Cookie{
				existingCookie},
			expectedEmpty:     false,
			expectedSetOptOut: false,
		},
		{
			description: "Opt Out Cookie - Wrong Value",
			givenExistingCookies: []http.Cookie{
				existingCookie,
				{Name: optOutCookieName, Value: "wrong"}},
			expectedEmpty:     false,
			expectedSetOptOut: false,
		},
		{
			description: "Opt Out Cookie - Wrong Name",
			givenExistingCookies: []http.Cookie{
				existingCookie,
				{Name: "wrong", Value: optOutCookieValue}},
			expectedEmpty:     false,
			expectedSetOptOut: false,
		},
		{
			description: "Opt Out Cookie - No Host Cookies",
			givenExistingCookies: []http.Cookie{
				{Name: optOutCookieName, Value: optOutCookieValue}},
			expectedEmpty:     true,
			expectedSetOptOut: true,
		},
	}

	for _, test := range testCases {
		req := httptest.NewRequest("POST", "http://www.prebid.com", nil)

		for _, c := range test.givenExistingCookies {
			req.AddCookie(&c)
		}

		parsed := ParseCookieFromRequest(req, &config.HostCookie{
			Family: "foo",
			OptOutCookie: config.Cookie{
				Name:  optOutCookieName,
				Value: optOutCookieValue,
			},
		})

		if test.expectedEmpty {
			assert.Empty(t, parsed.uids, test.description+":empty")
		} else {
			assert.NotEmpty(t, parsed.uids, test.description+":not-empty")
		}
		assert.Equal(t, parsed.optOut, test.expectedSetOptOut, test.description+":opt-out")
	}
}

func TestCookieReadWrite(t *testing.T) {
	cookie := newSampleCookie()

	received := writeThenRead(cookie, 0)
	uid, exists, isLive := received.GetUID("adnxs")
	if !exists || !isLive || uid != "123" {
		t.Errorf("Received cookie should have the adnxs ID=123. Got %s", uid)
	}

	uid, exists, isLive = received.GetUID("rubicon")
	if !exists || !isLive || uid != "456" {
		t.Errorf("Received cookie should have the rubicon ID=456. Got %s", uid)
	}

	assert.True(t, received.HasAnyLiveSyncs(), "Has Live Syncs")
	assert.Len(t, received.uids, 2, "Sync Count")
}

func TestPopulatedLegacyCookieRead(t *testing.T) {
	legacyJson := `{"uids":{"adnxs":"123","audienceNetwork":"456"},"bday":"2017-08-03T21:04:52.629198911Z"}`
	var cookie Cookie
	json.Unmarshal([]byte(legacyJson), &cookie)

	if cookie.HasAnyLiveSyncs() {
		t.Error("Expected 0 user syncs. Found at least 1.")
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
	var cookie Cookie
	json.Unmarshal([]byte(legacyJson), &cookie)

	if cookie.HasAnyLiveSyncs() {
		t.Error("Expected 0 user syncs. Found at least 1.")
	}
}

func TestNilCookie(t *testing.T) {
	var nilCookie *Cookie

	if nilCookie.HasLiveSync("anything") {
		t.Error("nil cookies should respond with false when asked if they have a sync")
	}

	if nilCookie.HasAnyLiveSyncs() {
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

func TestGetUIDs(t *testing.T) {
	cookie := newSampleCookie()
	uids := cookie.GetUIDs()

	assert.Len(t, uids, 2, "GetUIDs should return user IDs for all bidders")
	assert.Equal(t, "123", uids["adnxs"], "GetUIDs should return the correct user ID for each bidder")
	assert.Equal(t, "456", uids["rubicon"], "GetUIDs should return the correct user ID for each bidder")
}

func TestGetUIDsWithEmptyCookie(t *testing.T) {
	cookie := &Cookie{}
	uids := cookie.GetUIDs()

	assert.Len(t, uids, 0, "GetUIDs shouldn't return any user syncs for an empty cookie")
}

func TestGetUIDsWithNilCookie(t *testing.T) {
	var cookie *Cookie
	uids := cookie.GetUIDs()

	assert.Len(t, uids, 0, "GetUIDs shouldn't return any user syncs for a nil cookie")
}

func TestTrimCookiesClosestExpirationDates(t *testing.T) {
	cookieToSend := &Cookie{
		uids: map[string]uidWithExpiry{
			"k1": newTempId("12345678901234567890123456789012345678901234567890", 7),
			"k2": newTempId("abcdefghijklmnopqrstuvwxyz", 1),
			"k3": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 6),
			"k4": newTempId("12345678901234567890123456789612345678901234567890", 5),
			"k5": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 4),
			"k6": newTempId("12345678901234567890123456789012345678901234567890", 3),
			"k7": newTempId("abcdefghijklmnopqrstuvwxyz", 2),
		},
		optOut:   false,
		birthday: timestamp(),
	}

	type aTest struct {
		maxCookieSize int
		expKeys       []string
	}
	testCases := []aTest{
		{maxCookieSize: 2000, expKeys: []string{"k1", "k2", "k3", "k4", "k5", "k6", "k7"}}, //1 don't trim, set
		{maxCookieSize: 0, expKeys: []string{"k1", "k2", "k3", "k4", "k5", "k6", "k7"}},    //2 unlimited size: don't trim, set
		{maxCookieSize: 800, expKeys: []string{"k1", "k5", "k4", "k3"}},                    //3 trim to size and set
		{maxCookieSize: 500, expKeys: []string{"k1", "k3"}},                                //4 trim to size and set
		{maxCookieSize: 200, expKeys: []string{}},                                          //5 insufficient size, trim to zero length and set
		{maxCookieSize: -100, expKeys: []string{}},                                         //6 invalid size, trim to zero length and set
	}
	for i := range testCases {
		processedCookie := writeThenRead(cookieToSend, testCases[i].maxCookieSize)

		actualKeys := make([]string, 0, 7)
		for key := range processedCookie.uids {
			actualKeys = append(actualKeys, key)
		}

		assert.ElementsMatch(t, testCases[i].expKeys, actualKeys, "[Test %d]", i+1)
	}
}

func ensureEmptyMap(t *testing.T, cookie *Cookie) {
	if !cookie.AllowSyncs() {
		t.Error("Empty cookies should allow user syncs.")
	}
	if cookie.HasAnyLiveSyncs() {
		t.Error("Empty cookies shouldn't have any user syncs. Found at least 1.")
	}
}

func ensureConsistency(t *testing.T, cookie *Cookie) {
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
		if cookie.HasAnyLiveSyncs() {
			t.Error("If the user opted out, the PBSCookie should have no user syncs.")
		}

		err := cookie.TrySync("adnxs", "123")
		if err == nil {
			t.Error("TrySync should fail if the user has opted out of PBSCookie syncs, but it succeeded.")
		}
	}

	copiedCookie := ParseCookie(cookie.ToHTTPCookie(90 * 24 * time.Hour))
	if copiedCookie.AllowSyncs() != cookie.AllowSyncs() {
		t.Error("The PBSCookie interface shouldn't let modifications happen if the user has opted out")
	}

	assert.Equal(t, len(cookie.uids), len(copiedCookie.uids), "Incorrect sync count on reparsed cookie.")

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

func newTempId(uid string, offset int) uidWithExpiry {
	return uidWithExpiry{
		UID:     uid,
		Expires: time.Now().Add(time.Duration(offset) * time.Minute).UTC(),
	}
}

func newSampleCookie() *Cookie {
	return &Cookie{
		uids: map[string]uidWithExpiry{
			"adnxs":   newTempId("123", 10),
			"rubicon": newTempId("456", 10),
		},
		optOut:   false,
		birthday: timestamp(),
	}
}

func writeThenRead(cookie *Cookie, maxCookieSize int) *Cookie {
	w := httptest.NewRecorder()
	hostCookie := &config.HostCookie{Domain: "mock-domain", MaxCookieSizeBytes: maxCookieSize}
	cookie.SetCookieOnResponse(w, false, hostCookie, 90*24*time.Hour)
	writtenCookie := w.HeaderMap.Get("Set-Cookie")

	header := http.Header{}
	header.Add("Cookie", writtenCookie)
	request := http.Request{Header: header}
	return ParseCookieFromRequest(&request, hostCookie)
}

func TestSetCookieOnResponseForSameSiteNone(t *testing.T) {
	cookie := newSampleCookie()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "http://www.prebid.com", nil)
	ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/75.0.3770.142 Safari/537.36"
	req.Header.Set("User-Agent", ua)
	hostCookie := &config.HostCookie{Domain: "mock-domain", MaxCookieSizeBytes: 0}
	cookie.SetCookieOnResponse(w, true, hostCookie, 90*24*time.Hour)
	writtenCookie := w.HeaderMap.Get("Set-Cookie")
	t.Log("Set-Cookie is: ", writtenCookie)
	if !strings.Contains(writtenCookie, "; Secure;") {
		t.Error("Set-Cookie should contain Secure")
	}
}

func TestSetCookieOnResponseForOlderChromeVersion(t *testing.T) {
	cookie := newSampleCookie()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "http://www.prebid.com", nil)
	ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/65.0.3770.142 Safari/537.36"
	req.Header.Set("User-Agent", ua)
	hostCookie := &config.HostCookie{Domain: "mock-domain", MaxCookieSizeBytes: 0}
	cookie.SetCookieOnResponse(w, false, hostCookie, 90*24*time.Hour)
	writtenCookie := w.HeaderMap.Get("Set-Cookie")
	t.Log("Set-Cookie is: ", writtenCookie)
	if strings.Contains(writtenCookie, "SameSite=none") {
		t.Error("Set-Cookie should not contain SameSite=none")
	}
}
