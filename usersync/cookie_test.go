package usersync

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

// TODO: Add a one off test with a single t.Run()
// TODO: Add corrupted JSON
func TestReadCookie(t *testing.T) {
	testCases := []struct {
		name            string
		givenRequest    *http.Request
		givenHttpCookie *http.Cookie
		givenCookie     *Cookie
		givenDecoder    Decoder
		expectedCookie  *Cookie
	}{
		{
			name:         "SimpleCookie",
			givenRequest: httptest.NewRequest("POST", "http://www.prebid.com", nil),
			givenCookie: &Cookie{
				uids: map[string]UIDEntry{
					"adnxs": {
						UID:     "UID",
						Expires: time.Time{},
					},
				},
				optOut: false,
			},
			expectedCookie: &Cookie{
				uids: map[string]UIDEntry{
					"adnxs": {
						UID: "UID",
					},
				},
				optOut: false,
			},
		},
		{
			name:         "EmptyCookie",
			givenRequest: httptest.NewRequest("POST", "http://www.prebid.com", nil),
			givenCookie:  &Cookie{},
			expectedCookie: &Cookie{
				uids:   map[string]UIDEntry{},
				optOut: false,
			},
		},
		{
			name:         "CorruptedHttpCookie",
			givenRequest: httptest.NewRequest("POST", "http://www.prebid.com", nil),
			givenHttpCookie: &http.Cookie{
				Name:  "uids",
				Value: "bad base64 encoding",
			},
			givenCookie: nil,
			expectedCookie: &Cookie{
				uids:   map[string]UIDEntry{},
				optOut: false,
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			if test.givenCookie != nil {
				test.givenRequest.AddCookie(test.givenCookie.ToHTTPCookie())
			} else {
				test.givenRequest.AddCookie(test.givenHttpCookie)
			}
			actualCookie := ReadCookie(test.givenRequest, DecodeV1{}, &config.HostCookie{})
			assert.Equal(t, test.expectedCookie.uids, actualCookie.uids)
			assert.Equal(t, test.expectedCookie.optOut, actualCookie.optOut)
		})
	}
}

func TestWriteCookie(t *testing.T) {
	encoder := EncoderV1{}
	decoder := DecodeV1{}

	testCases := []struct {
		name               string
		givenCookie        *Cookie
		givenHostCookie    config.HostCookie
		givenSetSiteCookie bool
		expectedCookie     *Cookie
	}{
		{
			name: "simple-cookie",
			givenCookie: &Cookie{
				uids: map[string]UIDEntry{
					"adnxs": {
						UID:     "UID",
						Expires: time.Time{},
					},
				},
				optOut: false,
			},
			givenHostCookie:    config.HostCookie{},
			givenSetSiteCookie: true,
			expectedCookie: &Cookie{
				uids: map[string]UIDEntry{
					"adnxs": {
						UID:     "UID",
						Expires: time.Time{},
					},
				},
				optOut: false,
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			// Write Cookie
			w := httptest.NewRecorder()
			encodedCookie := encoder.Encode(test.givenCookie)
			WriteCookie(w, encodedCookie, &test.givenHostCookie, test.givenSetSiteCookie)
			writtenCookie := w.Header().Get("Set-Cookie")

			// Read Cookie
			header := http.Header{}
			header.Add("Cookie", writtenCookie)
			r := &http.Request{Header: header}
			actualCookie := ReadCookie(r, decoder, &config.HostCookie{})

			assert.Equal(t, test.expectedCookie, actualCookie)
		})
	}
}

func TestSync(t *testing.T) {
	testCases := []struct {
		name           string
		givenCookie    *Cookie
		givenSyncerKey string
		givenUID       string
		expectedCookie *Cookie
		expectedError  error
	}{
		{
			name: "simple-syncer",
			givenCookie: &Cookie{
				uids: map[string]UIDEntry{},
			},
			givenSyncerKey: "adnxs",
			givenUID:       "123",
			expectedCookie: &Cookie{
				uids: map[string]UIDEntry{
					"adnxs": {
						UID: "123",
					},
				},
			},
		},
		{
			name: "audienceNetwork",
			givenCookie: &Cookie{
				uids: map[string]UIDEntry{},
			},
			givenSyncerKey: string(openrtb_ext.BidderAudienceNetwork),
			givenUID:       "0",
			expectedError:  errors.New("audienceNetwork uses a UID of 0 as \"not yet recognized\""),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := test.givenCookie.Sync(test.givenSyncerKey, test.givenUID)
			if test.expectedError != nil {
				assert.Equal(t, test.expectedError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expectedCookie.uids[test.givenSyncerKey].UID, test.givenCookie.uids[test.givenSyncerKey].UID)
			}
		})
	}
}

func TestGetUIDs(t *testing.T) {
	testCases := []struct {
		name           string
		givenCookie    *Cookie
		expectedCookie *Cookie
		expectedLen    int
	}{
		{
			name: "two-uids",
			givenCookie: &Cookie{
				uids: map[string]UIDEntry{
					"adnxs": {
						UID: "123",
					},
					"rubicon": {
						UID: "456",
					},
				},
			},
			expectedLen: 2,
		},
		{
			name:        "empty",
			givenCookie: &Cookie{},
			expectedLen: 0,
		},
		{
			name:        "nil",
			givenCookie: nil,
			expectedLen: 0,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			uids := test.givenCookie.GetUIDs()
			assert.Len(t, uids, test.expectedLen, "GetUIDs should return user IDs for all bidders")
			for key, value := range uids {
				assert.Equal(t, test.givenCookie.uids[key].UID, value)
			}

		})
	}
}

func TestWriteCookieUserAgent(t *testing.T) {
	encoder := EncoderV1{}

	testCases := []struct {
		name                string
		givenUserAgent      string
		givenCookie         *Cookie
		givenHostCookie     config.HostCookie
		givenSetSiteCookie  bool
		expectedContains    string
		expectedNotContains string
	}{
		{
			name:           "same-site-none",
			givenUserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/75.0.3770.142 Safari/537.36",
			givenCookie: &Cookie{
				uids: map[string]UIDEntry{
					"adnxs": {
						UID:     "UID",
						Expires: time.Time{},
					},
				},
				optOut: false,
			},
			givenHostCookie:    config.HostCookie{},
			givenSetSiteCookie: true,
			expectedContains:   "; Secure;",
		},
		{
			name:           "older-chrome-version",
			givenUserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/65.0.3770.142 Safari/537.36",
			givenCookie: &Cookie{
				uids: map[string]UIDEntry{
					"adnxs": {
						UID:     "UID",
						Expires: time.Time{},
					},
				},
				optOut: false,
			},
			givenHostCookie:     config.HostCookie{},
			givenSetSiteCookie:  true,
			expectedNotContains: "SameSite=none",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			// Set Up
			req := httptest.NewRequest("GET", "http://www.prebid.com", nil)
			req.Header.Set("User-Agent", test.givenUserAgent)

			// Write Cookie
			w := httptest.NewRecorder()
			encodedCookie := encoder.Encode(test.givenCookie)
			WriteCookie(w, encodedCookie, &test.givenHostCookie, test.givenSetSiteCookie)
			writtenCookie := w.Header().Get("Set-Cookie")

			if test.expectedContains == "" {
				assert.NotContains(t, writtenCookie, test.expectedNotContains)
			} else {
				assert.Contains(t, writtenCookie, test.expectedContains)
			}
		})
	}
}

// TODO: Fix it so it doesn't randomly fail
func TestPrepareCookieForWrite(t *testing.T) {
	encoder := EncoderV1{}
	decoder := DecodeV1{}
	cookieToSend := &Cookie{
		uids: map[string]UIDEntry{
			"k1": newTempId("12345678901234567890123456789012345678901234567890", 7),
			"k2": newTempId("abcdefghijklmnopqrstuvwxyz", 1),
			"k3": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 6),
			"k4": newTempId("12345678901234567890123456789612345678901234567890", 5),
			"k5": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 4),
			"k6": newTempId("12345678901234567890123456789012345678901234567890", 3),
			"k7": newTempId("abcdefghijklmnopqrstuvwxyz", 2),
		},
		optOut: false,
	}

	testCases := []struct {
		name                     string
		givenMaxCookieSize       int
		expectedRemainingUidKeys map[string]bool
	}{
		{
			name:               "no-uids-ejected",
			givenMaxCookieSize: 2000,
			expectedRemainingUidKeys: map[string]bool{
				"k1": true, "k2": true, "k3": true, "k4": true, "k5": true, "k6": true, "k7": true,
			},
		},
		{
			name:               "no-uids-ejected-2",
			givenMaxCookieSize: 0,
			expectedRemainingUidKeys: map[string]bool{
				"k1": true, "k2": true, "k3": true, "k4": true, "k5": true, "k6": true, "k7": true,
			},
		},
		{
			name:               "two-uids-ejected",
			givenMaxCookieSize: 800,
			expectedRemainingUidKeys: map[string]bool{
				"k1": true, "k3": true, "k4": true, "k5": true, "k6": true,
			},
		},
		{
			name:               "four-uids-ejected",
			givenMaxCookieSize: 500,
			expectedRemainingUidKeys: map[string]bool{
				"k1": true, "k3": true, "k4": true,
			},
		},
		{
			name:               "all-but-one-uids-ejected",
			givenMaxCookieSize: 200,
			expectedRemainingUidKeys: map[string]bool{
				"k1": true,
			},
		},
		{
			name:                     "all-uids-ejected",
			givenMaxCookieSize:       100,
			expectedRemainingUidKeys: map[string]bool{},
		},
		{
			name:                     "invalid-max-size",
			givenMaxCookieSize:       -100,
			expectedRemainingUidKeys: map[string]bool{},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			freshCookie := cookieToSend
			encodedCookie := freshCookie.PrepareCookieForWrite(&config.HostCookie{MaxCookieSizeBytes: test.givenMaxCookieSize}, 90*24*time.Hour, encoder)
			decodedCookie := decoder.Decode(encodedCookie)

			for key := range decodedCookie.uids {
				_, ok := test.expectedRemainingUidKeys[key]
				assert.Equal(t, true, ok)
			}
			assert.Equal(t, len(decodedCookie.uids), len(test.expectedRemainingUidKeys))
		})
	}
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

func TestReadCookieOptOut(t *testing.T) {
	optOutCookieName := "optOutCookieName"
	optOutCookieValue := "optOutCookieValue"
	decoder := DecodeV1{}

	existingCookie := *(&Cookie{
		uids: map[string]UIDEntry{
			"foo": newTempId("fooID", 1),
			"bar": newTempId("barID", 2),
		},
		optOut:   false,
		birthday: timestamp(),
	}).ToHTTPCookie()

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

		parsed := ReadCookie(req, decoder, &config.HostCookie{
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
		uids:     make(map[string]UIDEntry),
		optOut:   true,
		birthday: timestamp(),
	}

	cookie.SetOptOut(false)
	if !cookie.AllowSyncs() {
		t.Error("After SetOptOut(false), a cookie should allow more user syncs.")
	}
	ensureConsistency(t, cookie)
}

func TestOptOutCookie(t *testing.T) {
	cookie := &Cookie{
		uids:     make(map[string]UIDEntry),
		optOut:   true,
		birthday: timestamp(),
	}
	ensureConsistency(t, cookie)
}

func newTempId(uid string, offset int) UIDEntry {
	return UIDEntry{
		UID:     uid,
		Expires: time.Now().Add(time.Duration(offset) * time.Minute).UTC(),
	}
}

func newSampleCookie() *Cookie {
	return &Cookie{
		uids: map[string]UIDEntry{
			"adnxs":   newTempId("123", 10),
			"rubicon": newTempId("456", 10),
		},
		optOut: false,
	}
}

func ensureConsistency(t *testing.T, cookie *Cookie) {
	decoder := DecodeV1{}

	if cookie.AllowSyncs() {
		err := cookie.Sync("pulsepoint", "1")
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

		err := cookie.Sync("adnxs", "123")
		if err == nil {
			t.Error("TrySync should fail if the user has opted out of PBSCookie syncs, but it succeeded.")
		}
	}

	copiedCookie := decoder.Decode(cookie.ToHTTPCookie().Value)
	if copiedCookie.AllowSyncs() != cookie.AllowSyncs() {
		t.Error("The PBSCookie interface shouldn't let modifications happen if the user has opted out")
	}

	if cookie.optOut {
		assert.Equal(t, 0, len(copiedCookie.uids), "Incorrect sync count on reparsed cookie.")
	} else {
		assert.Equal(t, len(cookie.uids), len(copiedCookie.uids), "Incorrect sync count on reparsed cookie.")
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
