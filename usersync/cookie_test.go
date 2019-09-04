package usersync

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
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
	cookie := newSampleCookie()
	ensureConsistency(t, cookie)
}

func TestBidderNameGets(t *testing.T) {
	cookie := newSampleCookie()
	id, exists := cookie.GetId(openrtb_ext.BidderAppnexus)
	if !exists {
		t.Errorf("Cookie missing expected Appnexus ID")
	}
	if id != "123" {
		t.Errorf("Bad appnexus id. Expected %s, got %s", "123", id)
	}

	id, exists = cookie.GetId(openrtb_ext.BidderRubicon)
	if !exists {
		t.Errorf("Cookie missing expected Rubicon ID")
	}
	if id != "456" {
		t.Errorf("Bad rubicon id. Expected %s, got %s", "456", id)
	}
}

func TestRejectAudienceNetworkCookie(t *testing.T) {
	raw := &PBSCookie{
		uids: map[string]uidWithExpiry{
			"audienceNetwork": newTempId("0", 10),
		},
		optOut:   false,
		birthday: timestamp(),
	}
	parsed := ParsePBSCookie(raw.ToHTTPCookie(90 * 24 * time.Hour))
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
		Name:  UID_COOKIE_NAME,
		Value: cookieData,
	}
	parsed := ParsePBSCookie(&raw)
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
	parsed := ParsePBSCookieFromRequest(req, &config.HostCookie{
		Family:     "adnxs",
		CookieName: otherCookieName,
	})
	val, _, _ := parsed.GetUID("adnxs")
	if val != id {
		t.Errorf("Bad cookie value. Expected %s, got %s", id, val)
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
	var nilCookie *PBSCookie

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

	uid, hadUID = nilCookie.GetId("anything")

	if uid != "" {
		t.Error("nil cookies should return empty strings for the UID.")
	}
	if hadUID {
		t.Error("nil cookies shouldn't claim to have a UID mapping.")
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
	cookie := &PBSCookie{}
	uids := cookie.GetUIDs()

	assert.Len(t, uids, 0, "GetUIDs shouldn't return any user syncs for an empty cookie")
}

func TestGetUIDsWithNilCookie(t *testing.T) {
	var cookie *PBSCookie
	uids := cookie.GetUIDs()

	assert.Len(t, uids, 0, "GetUIDs shouldn't return any user syncs for a nil cookie")
}

func TestUnlimitedSizeCookie(t *testing.T) {
	var cookie *PBSCookie
	var bigCookieLen int
	cookie, bigCookieLen = newBigCookie()
	//cookie.maxSizeBytes = 0 //When equal to zero, unlimited size

	var received *PBSCookie = writeThenRead(cookie, 0)

	assert.Equal(t, bigCookieLen, len(received.uids), "Cookie bigger than 32 KB in size was not supposed to be reduced in size")
}

func TestTrimBigCookie(t *testing.T) {
	var cookie *PBSCookie
	var bigCookieLen int
	var maxCookieSize int = 1 << 15 // 32768 bytes = 32 KB
	cookie, bigCookieLen = newBigCookie()

	var received *PBSCookie = writeThenRead(cookie, maxCookieSize)

	assert.Equal(t, bigCookieLen > len(received.uids), true, "Cookie bigger than 32 KB in size was not reduced according to date")
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

	copiedCookie := ParsePBSCookie(cookie.ToHTTPCookie(90 * 24 * time.Hour))
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

func newTempId(uid string, offset int) uidWithExpiry {
	return uidWithExpiry{
		UID:     uid,
		Expires: time.Now().Add(time.Duration(offset) * time.Minute),
	}
}

func newSampleCookie() *PBSCookie {
	return &PBSCookie{
		uids: map[string]uidWithExpiry{
			"adnxs":   newTempId("123", 10),
			"rubicon": newTempId("456", 10),
		},
		optOut:   false,
		birthday: timestamp(),
	}
}

func newBigCookie() (*PBSCookie, int) {
	var bigCookie *PBSCookie = &PBSCookie{
		uids: map[string]uidWithExpiry{
			"key1": newTempId("12345678901234567890123456789012345678901234567890", 3),
			"key2": newTempId("abcdefghijklmnopqrstuvwxyz", 1),
			"key3": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 6),
			"key4": newTempId("12345678901234567890123456789012345678901234567890", 6),
			"key5": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 1),
			"key6": newTempId("12345678901234567890123456789012345678901234567890", 7),
			"key7": newTempId("abcdefghijklmnopqrstuvwxyz", 2),
			"key8": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 6),
			"key9": newTempId("12345678901234567890123456789012345678901234567890", 1),

			"key10": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 2),
			"key11": newTempId("12345678901234567890123456789012345678901234567890", 3),
			"key12": newTempId("abcdefghijklmnopqrstuvwxyz", 4),
			"key13": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 7),
			"key14": newTempId("12345678901234567890123456789012345678901234567890", 3),
			"key15": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 6),
			"key16": newTempId("12345678901234567890123456789012345678901234567890", 5),
			"key17": newTempId("abcdefghijklmnopqrstuvwxyz", 3),
			"key18": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 4),
			"key19": newTempId("12345678901234567890123456789012345678901234567890", 3),

			"key20": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 4),
			"key21": newTempId("12345678901234567890123456789012345678901234567890", 1),
			"key22": newTempId("abcdefghijklmnopqrstuvwxyz", 7),
			"key23": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 4),
			"key24": newTempId("12345678901234567890123456789012345678901234567890", 5),
			"key25": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 6),
			"key26": newTempId("12345678901234567890123456789012345678901234567890", 4),
			"key27": newTempId("abcdefghijklmnopqrstuvwxyz", 1),
			"key28": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 4),
			"key29": newTempId("12345678901234567890123456789012345678901234567890", 6),

			"key30": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 6),
			"key31": newTempId("12345678901234567890123456789012345678901234567890", 7),
			"key32": newTempId("abcdefghijklmnopqrstuvwxyz", 6),
			"key33": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 2),
			"key34": newTempId("12345678901234567890123456789012345678901234567890", 4),
			"key35": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 4),
			"key36": newTempId("12345678901234567890123456789012345678901234567890", 7),
			"key37": newTempId("abcdefghijklmnopqrstuvwxyz", 1),
			"key38": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 3),
			"key39": newTempId("12345678901234567890123456789012345678901234567890", 3),

			"key40": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 1),
			"key41": newTempId("12345678901234567890123456789012345678901234567890", 7),
			"key42": newTempId("abcdefghijklmnopqrstuvwxyz", 5),
			"key43": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 7),
			"key44": newTempId("12345678901234567890123456789012345678901234567890", 3),
			"key45": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 3),
			"key46": newTempId("12345678901234567890123456789012345678901234567890", 4),
			"key47": newTempId("abcdefghijklmnopqrstuvwxyz", 1),
			"key48": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 6),
			"key49": newTempId("12345678901234567890123456789012345678901234567890", 4),

			"key50": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 2),
			"key51": newTempId("12345678901234567890123456789012345678901234567890", 4),
			"key52": newTempId("abcdefghijklmnopqrstuvwxyz", 3),
			"key53": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 1),
			"key54": newTempId("12345678901234567890123456789012345678901234567890", 5),
			"key55": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 3),
			"key56": newTempId("12345678901234567890123456789012345678901234567890", 7),
			"key57": newTempId("abcdefghijklmnopqrstuvwxyz", 7),
			"key58": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 3),
			"key59": newTempId("12345678901234567890123456789012345678901234567890", 5),

			"key60": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 4),
			"key61": newTempId("12345678901234567890123456789012345678901234567890", 7),
			"key62": newTempId("abcdefghijklmnopqrstuvwxyz", 2),
			"key63": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 2),
			"key64": newTempId("12345678901234567890123456789012345678901234567890", 4),
			"key65": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 4),
			"key66": newTempId("12345678901234567890123456789012345678901234567890", 3),
			"key67": newTempId("abcdefghijklmnopqrstuvwxyz", 7),
			"key68": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 2),
			"key69": newTempId("12345678901234567890123456789012345678901234567890", 3),

			"key70": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 1),
			"key71": newTempId("12345678901234567890123456789012345678901234567890", 3),
			"key72": newTempId("abcdefghijklmnopqrstuvwxyz", 3),
			"key73": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 1),
			"key74": newTempId("12345678901234567890123456789012345678901234567890", 5),
			"key75": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 7),
			"key76": newTempId("12345678901234567890123456789012345678901234567890", 5),
			"key77": newTempId("abcdefghijklmnopqrstuvwxyz", 5),
			"key78": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 5),
			"key79": newTempId("12345678901234567890123456789012345678901234567890", 2),

			"key80": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 2),
			"key81": newTempId("12345678901234567890123456789012345678901234567890", 6),
			"key82": newTempId("abcdefghijklmnopqrstuvwxyz", 2),
			"key83": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 7),
			"key84": newTempId("12345678901234567890123456789012345678901234567890", 6),
			"key85": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 4),
			"key86": newTempId("12345678901234567890123456789012345678901234567890", 3),
			"key87": newTempId("abcdefghijklmnopqrstuvwxyz", 3),
			"key88": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 7),
			"key89": newTempId("12345678901234567890123456789012345678901234567890", 2),

			"key90": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 7),
			"key91": newTempId("12345678901234567890123456789012345678901234567890", 4),
			"key92": newTempId("abcdefghijklmnopqrstuvwxyz", 1),
			"key93": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 1),
			"key94": newTempId("12345678901234567890123456789012345678901234567890", 5),
			"key95": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 6),
			"key96": newTempId("12345678901234567890123456789012345678901234567890", 2),
			"key97": newTempId("abcdefghijklmnopqrstuvwxyz", 4),
			"key98": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 6),
			"key99": newTempId("12345678901234567890123456789012345678901234567890", 4),

			"key101": newTempId("12345678901234567890123456789012345678901234567890", 5),
			"key102": newTempId("abcdefghijklmnopqrstuvwxyz", 3),
			"key103": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 1),
			"key104": newTempId("12345678901234567890123456789012345678901234567890", 6),
			"key105": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 7),
			"key106": newTempId("12345678901234567890123456789012345678901234567890", 7),
			"key107": newTempId("abcdefghijklmnopqrstuvwxyz", 2),
			"key108": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 3),
			"key109": newTempId("12345678901234567890123456789012345678901234567890", 4),

			"key110": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 5),
			"key111": newTempId("12345678901234567890123456789012345678901234567890", 4),
			"key112": newTempId("abcdefghijklmnopqrstuvwxyz", 2),
			"key113": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 6),
			"key114": newTempId("12345678901234567890123456789012345678901234567890", 5),
			"key115": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 1),
			"key116": newTempId("12345678901234567890123456789012345678901234567890", 4),
			"key117": newTempId("abcdefghijklmnopqrstuvwxyz", 4),
			"key118": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 1),
			"key119": newTempId("12345678901234567890123456789012345678901234567890", 2),

			"key120": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 7),
			"key121": newTempId("12345678901234567890123456789012345678901234567890", 7),
			"key122": newTempId("abcdefghijklmnopqrstuvwxyz", 1),
			"key123": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 1),
			"key124": newTempId("12345678901234567890123456789012345678901234567890", 2),
			"key125": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 1),
			"key126": newTempId("12345678901234567890123456789012345678901234567890", 6),
			"key127": newTempId("abcdefghijklmnopqrstuvwxyz", 6),
			"key128": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 2),
			"key129": newTempId("12345678901234567890123456789012345678901234567890", 2),

			"key130": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 6),
			"key131": newTempId("12345678901234567890123456789012345678901234567890", 5),
			"key132": newTempId("abcdefghijklmnopqrstuvwxyz", 3),
			"key133": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 4),
			"key134": newTempId("12345678901234567890123456789012345678901234567890", 6),
			"key135": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 3),
			"key136": newTempId("12345678901234567890123456789012345678901234567890", 5),
			"key137": newTempId("abcdefghijklmnopqrstuvwxyz", 1),
			"key138": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 3),
			"key139": newTempId("12345678901234567890123456789012345678901234567890", 3),

			"key140": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 1),
			"key141": newTempId("12345678901234567890123456789012345678901234567890", 1),
			"key142": newTempId("abcdefghijklmnopqrstuvwxyz", 6),
			"key143": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 6),
			"key144": newTempId("12345678901234567890123456789012345678901234567890", 7),
			"key145": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 2),
			"key146": newTempId("12345678901234567890123456789012345678901234567890", 4),
			"key147": newTempId("abcdefghijklmnopqrstuvwxyz", 4),
			"key148": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 2),
			"key149": newTempId("12345678901234567890123456789012345678901234567890", 1),

			"key150": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 2),
			"key151": newTempId("12345678901234567890123456789012345678901234567890", 2),
			"key152": newTempId("abcdefghijklmnopqrstuvwxyz", 3),
			"key153": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 5),
			"key154": newTempId("12345678901234567890123456789012345678901234567890", 1),
			"key155": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 2),
			"key156": newTempId("12345678901234567890123456789012345678901234567890", 2),
			"key157": newTempId("abcdefghijklmnopqrstuvwxyz", 1),
			"key158": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 6),
			"key159": newTempId("12345678901234567890123456789012345678901234567890", 2),

			"key160": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 4),
			"key161": newTempId("12345678901234567890123456789012345678901234567890", 5),
			"key162": newTempId("abcdefghijklmnopqrstuvwxyz", 5),
			"key163": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 1),
			"key164": newTempId("12345678901234567890123456789012345678901234567890", 1),
			"key165": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 1),
			"key166": newTempId("12345678901234567890123456789012345678901234567890", 5),
			"key167": newTempId("abcdefghijklmnopqrstuvwxyz", 7),
			"key168": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 3),
			"key169": newTempId("12345678901234567890123456789012345678901234567890", 5),

			"key170": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 7),
			"key171": newTempId("12345678901234567890123456789012345678901234567890", 5),
			"key172": newTempId("abcdefghijklmnopqrstuvwxyz", 4),
			"key173": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 1),
			"key174": newTempId("12345678901234567890123456789012345678901234567890", 2),
			"key175": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 6),
			"key176": newTempId("12345678901234567890123456789012345678901234567890", 6),
			"key177": newTempId("abcdefghijklmnopqrstuvwxyz", 5),
			"key178": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 7),
			"key179": newTempId("12345678901234567890123456789012345678901234567890", 1),

			"key180": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 4),
			"key181": newTempId("12345678901234567890123456789012345678901234567890", 3),
			"key182": newTempId("abcdefghijklmnopqrstuvwxyz", 1),
			"key183": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 4),
			"key184": newTempId("12345678901234567890123456789012345678901234567890", 1),
			"key185": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 1),
			"key186": newTempId("12345678901234567890123456789012345678901234567890", 5),
			"key187": newTempId("abcdefghijklmnopqrstuvwxyz", 1),
			"key188": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 7),
			"key189": newTempId("12345678901234567890123456789012345678901234567890", 1),

			"key190": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 6),
			"key191": newTempId("12345678901234567890123456789012345678901234567890", 7),
			"key192": newTempId("abcdefghijklmnopqrstuvwxyz", 5),
			"key193": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 3),
			"key194": newTempId("12345678901234567890123456789012345678901234567890", 1),
			"key195": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 6),
			"key196": newTempId("12345678901234567890123456789012345678901234567890", 4),
			"key197": newTempId("abcdefghijklmnopqrstuvwxyz", 1),
			"key198": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 3),
			"key199": newTempId("12345678901234567890123456789012345678901234567890", 6),

			"key200": newTempId("12345678901234567890123456789012345678901234567890", 5),
			"key201": newTempId("12345678901234567890123456789012345678901234567890", 7),
			"key202": newTempId("abcdefghijklmnopqrstuvwxyz", 3),
			"key203": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 6),
			"key204": newTempId("12345678901234567890123456789012345678901234567890", 3),
			"key205": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 6),
			"key206": newTempId("12345678901234567890123456789012345678901234567890", 2),
			"key207": newTempId("abcdefghijklmnopqrstuvwxyz", 7),
			"key208": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 7),
			"key209": newTempId("12345678901234567890123456789012345678901234567890", 5),

			"key210": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 7),
			"key211": newTempId("12345678901234567890123456789012345678901234567890", 6),
			"key212": newTempId("abcdefghijklmnopqrstuvwxyz", 2),
			"key213": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 4),
			"key214": newTempId("12345678901234567890123456789012345678901234567890", 3),
			"key215": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 5),
			"key216": newTempId("12345678901234567890123456789012345678901234567890", 2),
			"key217": newTempId("abcdefghijklmnopqrstuvwxyz", 1),
			"key218": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 5),
			"key219": newTempId("12345678901234567890123456789012345678901234567890", 7),

			"key220": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 2),
			"key221": newTempId("12345678901234567890123456789012345678901234567890", 7),
			"key222": newTempId("abcdefghijklmnopqrstuvwxyz", 4),
			"key223": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 6),
			"key224": newTempId("12345678901234567890123456789012345678901234567890", 7),
			"key225": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 7),
			"key226": newTempId("12345678901234567890123456789012345678901234567890", 6),
			"key227": newTempId("abcdefghijklmnopqrstuvwxyz", 2),
			"key228": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 6),
			"key229": newTempId("12345678901234567890123456789012345678901234567890", 4),

			"key230": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 2),
			"key231": newTempId("12345678901234567890123456789012345678901234567890", 2),
			"key232": newTempId("abcdefghijklmnopqrstuvwxyz", 2),
			"key233": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 6),
			"key234": newTempId("12345678901234567890123456789012345678901234567890", 6),
			"key235": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 4),
			"key236": newTempId("12345678901234567890123456789012345678901234567890", 7),
			"key237": newTempId("abcdefghijklmnopqrstuvwxyz", 4),
			"key238": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 7),
			"key239": newTempId("12345678901234567890123456789012345678901234567890", 3),

			"key240": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 6),
			"key241": newTempId("12345678901234567890123456789012345678901234567890", 7),
			"key242": newTempId("abcdefghijklmnopqrstuvwxyz", 2),
			"key243": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 1),
			"key244": newTempId("12345678901234567890123456789012345678901234567890", 7),
			"key245": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 5),
			"key246": newTempId("12345678901234567890123456789012345678901234567890", 7),
			"key247": newTempId("abcdefghijklmnopqrstuvwxyz", 6),
			"key248": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 6),
			"key249": newTempId("12345678901234567890123456789012345678901234567890", 7),

			"key250": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 6),
			"key251": newTempId("12345678901234567890123456789012345678901234567890", 2),
			"key252": newTempId("abcdefghijklmnopqrstuvwxyz", 3),
			"key253": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 1),
			"key254": newTempId("12345678901234567890123456789012345678901234567890", 3),
			"key255": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 1),
			"key256": newTempId("12345678901234567890123456789012345678901234567890", 3),
			"key257": newTempId("abcdefghijklmnopqrstuvwxyz", 5),
			"key258": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 6),
			"key259": newTempId("12345678901234567890123456789012345678901234567890", 3),

			"key260": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 2),
			"key261": newTempId("12345678901234567890123456789012345678901234567890", 2),
			"key262": newTempId("abcdefghijklmnopqrstuvwxyz", 5),
			"key263": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 7),
			"key264": newTempId("12345678901234567890123456789012345678901234567890", 7),
			"key265": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 5),
			"key266": newTempId("12345678901234567890123456789012345678901234567890", 2),
			"key267": newTempId("abcdefghijklmnopqrstuvwxyz", 6),
			"key268": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 3),
			"key269": newTempId("12345678901234567890123456789012345678901234567890", 1),

			"key270": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 5),
			"key271": newTempId("12345678901234567890123456789012345678901234567890", 7),
			"key272": newTempId("abcdefghijklmnopqrstuvwxyz", 3),
			"key273": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 5),
			"key274": newTempId("12345678901234567890123456789012345678901234567890", 3),
			"key275": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 2),
			"key276": newTempId("12345678901234567890123456789012345678901234567890", 1),
			"key277": newTempId("abcdefghijklmnopqrstuvwxyz", 6),
			"key278": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 3),
			"key279": newTempId("12345678901234567890123456789012345678901234567890", 1),

			"key280": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 2),
			"key281": newTempId("12345678901234567890123456789012345678901234567890", 2),
			"key282": newTempId("abcdefghijklmnopqrstuvwxyz", 2),
			"key283": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 6),
			"key284": newTempId("12345678901234567890123456789012345678901234567890", 6),
			"key285": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 4),
			"key286": newTempId("12345678901234567890123456789012345678901234567890", 7),
			"key287": newTempId("abcdefghijklmnopqrstuvwxyz", 4),
			"key288": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 7),
			"key289": newTempId("12345678901234567890123456789012345678901234567890", 3),

			"key290": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 6),
			"key291": newTempId("12345678901234567890123456789012345678901234567890", 7),
			"key292": newTempId("abcdefghijklmnopqrstuvwxyz", 2),
			"key293": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 1),
			"key294": newTempId("12345678901234567890123456789012345678901234567890", 7),
			"key295": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ", 5),
			"key296": newTempId("12345678901234567890123456789012345678901234567890", 7),
			"key297": newTempId("abcdefghijklmnopqrstuvwxyz", 6),
			"key298": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 6),
			"key299": newTempId("12345678901234567890123456789012345678901234567890", 7),
		},
		optOut:   false,
		birthday: timestamp(),
	}
	return bigCookie, len(bigCookie.uids)

}

func writeThenRead(cookie *PBSCookie, maxCookieSize int) *PBSCookie {
	w := httptest.NewRecorder()
	hostCookie := &config.HostCookie{Domain: "mock-domain", MaxCookieSizeBytes: maxCookieSize}
	cookie.SetCookieOnResponse(w, hostCookie, 90*24*time.Hour)
	writtenCookie := w.HeaderMap.Get("Set-Cookie")

	header := http.Header{}
	header.Add("Cookie", writtenCookie)
	request := http.Request{Header: header}
	return ParsePBSCookieFromRequest(&request, hostCookie)
}
