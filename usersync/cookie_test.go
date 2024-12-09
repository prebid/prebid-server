package usersync

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

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
			name:         "simple-cookie",
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
			name:         "empty-cookie",
			givenRequest: httptest.NewRequest("POST", "http://www.prebid.com", nil),
			givenCookie:  &Cookie{},
			expectedCookie: &Cookie{
				uids:   map[string]UIDEntry{},
				optOut: false,
			},
		},
		{
			name:         "nil-cookie",
			givenRequest: httptest.NewRequest("POST", "http://www.prebid.com", nil),
			givenCookie:  nil,
			expectedCookie: &Cookie{
				uids:   map[string]UIDEntry{},
				optOut: false,
			},
		},
		{
			name:         "corruptted-http-cookie",
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
				httpCookie, err := ToHTTPCookie(test.givenCookie)
				assert.NoError(t, err)
				test.givenRequest.AddCookie(httpCookie)
			} else if test.givenCookie == nil && test.givenHttpCookie != nil {
				test.givenRequest.AddCookie(test.givenHttpCookie)
			}
			actualCookie := ReadCookie(test.givenRequest, Base64Decoder{}, &config.HostCookie{})
			assert.Equal(t, test.expectedCookie.uids, actualCookie.uids)
			assert.Equal(t, test.expectedCookie.optOut, actualCookie.optOut)
		})
	}
}

func TestWriteCookie(t *testing.T) {
	encoder := Base64Encoder{}
	decoder := Base64Decoder{}

	testCases := []struct {
		name               string
		givenCookie        *Cookie
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
			givenSetSiteCookie: false,
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
		{
			name: "simple-cookie-opt-out",
			givenCookie: &Cookie{
				uids: map[string]UIDEntry{
					"adnxs": {
						UID:     "UID",
						Expires: time.Time{},
					},
				},
				optOut: true,
			},
			givenSetSiteCookie: true,
			expectedCookie: &Cookie{
				uids:   map[string]UIDEntry{},
				optOut: true,
			},
		},
		{
			name: "cookie-multiple-uids",
			givenCookie: &Cookie{
				uids: map[string]UIDEntry{
					"adnxs": {
						UID:     "UID",
						Expires: time.Time{},
					},
					"rubicon": {
						UID:     "UID2",
						Expires: time.Time{},
					},
				},
				optOut: false,
			},
			givenSetSiteCookie: true,
			expectedCookie: &Cookie{
				uids: map[string]UIDEntry{
					"adnxs": {
						UID:     "UID",
						Expires: time.Time{},
					},
					"rubicon": {
						UID:     "UID2",
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
			encodedCookie, err := encoder.Encode(test.givenCookie)
			assert.NoError(t, err)
			WriteCookie(w, encodedCookie, &config.HostCookie{}, test.givenSetSiteCookie)
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
			name: "simple-sync",
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
			name: "dont-allow-syncs",
			givenCookie: &Cookie{
				uids:   map[string]UIDEntry{},
				optOut: true,
			},
			givenSyncerKey: "adnxs",
			givenUID:       "123",
			expectedCookie: &Cookie{
				uids: map[string]UIDEntry{},
			},
			expectedError: errors.New("the user has opted out of prebid server cookie syncs"),
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
			name: "one-uid",
			givenCookie: &Cookie{
				uids: map[string]UIDEntry{
					"adnxs": {
						UID: "123",
					},
				},
			},
			expectedLen: 1,
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
			assert.Len(t, uids, test.expectedLen)
			for key, value := range uids {
				assert.Equal(t, test.givenCookie.uids[key].UID, value)
			}

		})
	}
}

func TestWriteCookieUserAgent(t *testing.T) {
	encoder := Base64Encoder{}

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
			encodedCookie, err := encoder.Encode(test.givenCookie)
			assert.NoError(t, err)
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

func TestPrepareCookieForWrite(t *testing.T) {
	encoder := Base64Encoder{}
	decoder := Base64Decoder{}

	mainCookie := &Cookie{
		uids: map[string]UIDEntry{
			"mainUID": newTempId("1234567890123456789012345678901234567890123456", 7),
			"2":       newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 6),
			"3":       newTempId("123456789012345678901234567896123456789012345678", 5),
			"4":       newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 4),
			"5":       newTempId("12345678901234567890123456789012345678901234567890", 3),
			"6":       newTempId("abcdefghij", 2),
			"7":       newTempId("abcdefghijklmnopqrstuvwxy", 1),
		},
		optOut: false,
	}

	errorCookie := &Cookie{
		uids: map[string]UIDEntry{
			"syncerNotPriority": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 7),
			"2":                 newTempId("1234567890123456789012345678901234567890123456", 7), // Priority Element
		},
		optOut: false,
	}

	ejector := &PriorityBidderEjector{
		PriorityGroups: [][]string{
			{"mainUID"},
			{"2", "3"},
			{"4", "5", "6"},
			{"7"},
		},
		SyncersByBidder: map[string]Syncer{
			"mainUID": fakeSyncer{
				key: "mainUID",
			},
			"2": fakeSyncer{
				key: "2",
			},
			"3": fakeSyncer{
				key: "3",
			},
			"4": fakeSyncer{
				key: "4",
			},
			"5": fakeSyncer{
				key: "5",
			},
			"6": fakeSyncer{
				key: "6",
			},
			"mistmatchedBidder": fakeSyncer{
				key: "7",
			},
		},
		TieEjector: &OldestEjector{},
	}

	testCases := []struct {
		name                     string
		givenMaxCookieSize       int
		givenCookieToSend        *Cookie
		givenIsSyncerPriority    bool
		expectedRemainingUidKeys []string
		expectedError            error
	}{
		{
			name:                  "no-uids-ejected",
			givenMaxCookieSize:    2000,
			givenCookieToSend:     mainCookie,
			givenIsSyncerPriority: true,
			expectedRemainingUidKeys: []string{
				"mainUID", "2", "3", "4", "5", "6", "7",
			},
		},
		{
			name:               "invalid-max-size",
			givenMaxCookieSize: -100,
			givenCookieToSend:  mainCookie,
			expectedRemainingUidKeys: []string{
				"mainUID", "2", "3", "4", "5", "6", "7",
			},
		},
		{
			name:                  "syncer-is-not-priority",
			givenMaxCookieSize:    100,
			givenCookieToSend:     errorCookie,
			givenIsSyncerPriority: false,
			expectedError:         errors.New("syncer key is not a priority, and there are only priority elements left"),
		},
		{
			name:                  "no-uids-ejected-2",
			givenMaxCookieSize:    0,
			givenCookieToSend:     mainCookie,
			givenIsSyncerPriority: true,
			expectedRemainingUidKeys: []string{
				"mainUID", "2", "3", "4", "5", "6", "7",
			},
		},
		{
			name:                  "one-uid-ejected",
			givenMaxCookieSize:    900,
			givenCookieToSend:     mainCookie,
			givenIsSyncerPriority: true,
			expectedRemainingUidKeys: []string{
				"mainUID", "2", "3", "4", "5", "6",
			},
		},
		{
			name:                  "four-uids-ejected",
			givenMaxCookieSize:    500,
			givenCookieToSend:     mainCookie,
			givenIsSyncerPriority: true,
			expectedRemainingUidKeys: []string{
				"mainUID", "2", "3",
			},
		},
		{
			name:                  "all-but-one-uids-ejected",
			givenMaxCookieSize:    300,
			givenCookieToSend:     mainCookie,
			givenIsSyncerPriority: true,
			expectedRemainingUidKeys: []string{
				"mainUID",
			},
		},
		{
			name:                     "only-main-uid-left",
			givenMaxCookieSize:       100,
			givenCookieToSend:        mainCookie,
			expectedError:            errors.New("uid that's trying to be synced is bigger than MaxCookieSize"),
			expectedRemainingUidKeys: []string{},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ejector.IsSyncerPriority = test.givenIsSyncerPriority
			encodedCookie, err := test.givenCookieToSend.PrepareCookieForWrite(&config.HostCookie{MaxCookieSizeBytes: test.givenMaxCookieSize}, encoder, ejector)

			if test.expectedError != nil {
				assert.Equal(t, test.expectedError, err)
			} else {
				assert.NoError(t, err)
				decodedCookie := decoder.Decode(encodedCookie)

				for _, key := range test.expectedRemainingUidKeys {
					_, ok := decodedCookie.uids[key]
					assert.Equal(t, true, ok)
				}
				assert.Equal(t, len(decodedCookie.uids), len(test.expectedRemainingUidKeys))
			}
		})
	}
}

func TestSyncHostCookie(t *testing.T) {
	testCases := []struct {
		name            string
		givenCookie     *Cookie
		givenUID        string
		givenHostCookie *config.HostCookie
		expectedCookie  *Cookie
		expectedError   error
	}{
		{
			name: "simple-sync",
			givenCookie: &Cookie{
				uids: map[string]UIDEntry{},
			},
			givenHostCookie: &config.HostCookie{
				Family:     "syncer",
				CookieName: "adnxs",
			},
			expectedCookie: &Cookie{
				uids: map[string]UIDEntry{
					"syncer": {
						UID: "some-user-id",
					},
				},
			},
		},
		{
			name: "uids-already-present",
			givenCookie: &Cookie{
				uids: map[string]UIDEntry{
					"some-syncer": {
						UID: "some-other-user-id",
					},
				},
			},
			givenHostCookie: &config.HostCookie{
				Family:     "syncer",
				CookieName: "adnxs",
			},
			expectedCookie: &Cookie{
				uids: map[string]UIDEntry{
					"syncer": {
						UID: "some-user-id",
					},
					"some-syncer": {
						UID: "some-other-user-id",
					},
				},
			},
		},
		{
			name: "host-already-synced",
			givenCookie: &Cookie{
				uids: map[string]UIDEntry{
					"syncer": {
						UID: "some-user-id",
					},
				},
			},
			givenHostCookie: &config.HostCookie{
				Family:     "syncer",
				CookieName: "adnxs",
			},
			expectedCookie: &Cookie{
				uids: map[string]UIDEntry{
					"syncer": {
						UID: "some-user-id",
					},
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "http://www.prebid.com", nil)
			r.AddCookie(&http.Cookie{
				Name:  test.givenHostCookie.CookieName,
				Value: "some-user-id",
			})

			SyncHostCookie(r, test.givenCookie, test.givenHostCookie)
			for key, value := range test.expectedCookie.uids {
				assert.Equal(t, value.UID, test.givenCookie.uids[key].UID)
			}
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

func TestReadCookieOptOut(t *testing.T) {
	optOutCookieName := "optOutCookieName"
	optOutCookieValue := "optOutCookieValue"
	decoder := Base64Decoder{}

	cookie := Cookie{
		uids: map[string]UIDEntry{
			"foo": newTempId("fooID", 1),
			"bar": newTempId("barID", 2),
		},
		optOut: false,
	}

	existingCookie, _ := ToHTTPCookie(&cookie)

	testCases := []struct {
		description          string
		givenExistingCookies []*http.Cookie
		expectedEmpty        bool
		expectedSetOptOut    bool
	}{
		{
			description: "Opt Out Cookie",
			givenExistingCookies: []*http.Cookie{
				existingCookie,
				{Name: optOutCookieName, Value: optOutCookieValue}},
			expectedEmpty:     true,
			expectedSetOptOut: true,
		},
		{
			description: "No Opt Out Cookie",
			givenExistingCookies: []*http.Cookie{
				existingCookie},
			expectedEmpty:     false,
			expectedSetOptOut: false,
		},
		{
			description: "Opt Out Cookie - Wrong Value",
			givenExistingCookies: []*http.Cookie{
				existingCookie,
				{Name: optOutCookieName, Value: "wrong"}},
			expectedEmpty:     false,
			expectedSetOptOut: false,
		},
		{
			description: "Opt Out Cookie - Wrong Name",
			givenExistingCookies: []*http.Cookie{
				existingCookie,
				{Name: "wrong", Value: optOutCookieValue}},
			expectedEmpty:     false,
			expectedSetOptOut: false,
		},
		{
			description: "Opt Out Cookie - No Host Cookies",
			givenExistingCookies: []*http.Cookie{
				{Name: optOutCookieName, Value: optOutCookieValue}},
			expectedEmpty:     true,
			expectedSetOptOut: true,
		},
	}

	for _, test := range testCases {
		req := httptest.NewRequest("POST", "http://www.prebid.com", nil)

		for _, c := range test.givenExistingCookies {
			req.AddCookie(c)
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

func TestOptIn(t *testing.T) {
	cookie := &Cookie{
		uids:   make(map[string]UIDEntry),
		optOut: true,
	}

	cookie.SetOptOut(false)
	if !cookie.AllowSyncs() {
		t.Error("After SetOptOut(false), a cookie should allow more user syncs.")
	}
	ensureConsistency(t, cookie)
}

func TestOptOutReset(t *testing.T) {
	cookie := newSampleCookie()

	cookie.SetOptOut(true)
	if cookie.AllowSyncs() {
		t.Error("After SetOptOut(true), a cookie should not allow more user syncs.")
	}
	ensureConsistency(t, cookie)
}

func TestOptOutCookie(t *testing.T) {
	cookie := &Cookie{
		uids:   make(map[string]UIDEntry),
		optOut: true,
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
	decoder := Base64Decoder{}

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
	httpCookie, err := ToHTTPCookie(cookie)
	assert.NoError(t, err)
	copiedCookie := decoder.Decode(httpCookie.Value)
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

func ToHTTPCookie(cookie *Cookie) (*http.Cookie, error) {
	encoder := Base64Encoder{}
	encodedCookie, err := encoder.Encode(cookie)
	if err != nil {
		return nil, nil
	}

	return &http.Cookie{
		Name:    uidCookieName,
		Value:   encodedCookie,
		Expires: time.Now().Add((90 * 24 * time.Hour)),
		Path:    "/",
	}, nil
}
