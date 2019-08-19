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
			"audienceNetwork": newTempId("0"),
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

	received := writeThenRead(cookie)
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

func TestTrimBigCookie(t *testing.T) {
	var cookie *PBSCookie
	var bigCookieLen int
	cookie, bigCookieLen = newBigCookie()

	var received *PBSCookie = writeThenRead(cookie)

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

func newTempId(uid string) uidWithExpiry {
	return uidWithExpiry{
		UID:     uid,
		Expires: time.Now().Add(10 * time.Minute),
	}
}

func newSampleCookie() *PBSCookie {
	return &PBSCookie{
		uids: map[string]uidWithExpiry{
			"adnxs":   newTempId("123"),
			"rubicon": newTempId("456"),
		},
		optOut:   false,
		birthday: timestamp(),
	}
}

func newBigCookie() (*PBSCookie, int) {
	var bigCookie *PBSCookie = &PBSCookie{
		uids: map[string]uidWithExpiry{
			"key1": newTempId("12345678901234567890123456789012345678901234567890"),
			"key2": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key3": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key4": newTempId("12345678901234567890123456789012345678901234567890"),
			"key5": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key6": newTempId("12345678901234567890123456789012345678901234567890"),
			"key7": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key8": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key9": newTempId("12345678901234567890123456789012345678901234567890"),

			"key10": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key11": newTempId("12345678901234567890123456789012345678901234567890"),
			"key12": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key13": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key14": newTempId("12345678901234567890123456789012345678901234567890"),
			"key15": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key16": newTempId("12345678901234567890123456789012345678901234567890"),
			"key17": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key18": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key19": newTempId("12345678901234567890123456789012345678901234567890"),

			"key20": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key21": newTempId("12345678901234567890123456789012345678901234567890"),
			"key22": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key23": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key24": newTempId("12345678901234567890123456789012345678901234567890"),
			"key25": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key26": newTempId("12345678901234567890123456789012345678901234567890"),
			"key27": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key28": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key29": newTempId("12345678901234567890123456789012345678901234567890"),

			"key30": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key31": newTempId("12345678901234567890123456789012345678901234567890"),
			"key32": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key33": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key34": newTempId("12345678901234567890123456789012345678901234567890"),
			"key35": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key36": newTempId("12345678901234567890123456789012345678901234567890"),
			"key37": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key38": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key39": newTempId("12345678901234567890123456789012345678901234567890"),

			"key40": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key41": newTempId("12345678901234567890123456789012345678901234567890"),
			"key42": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key43": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key44": newTempId("12345678901234567890123456789012345678901234567890"),
			"key45": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key46": newTempId("12345678901234567890123456789012345678901234567890"),
			"key47": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key48": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key49": newTempId("12345678901234567890123456789012345678901234567890"),

			"key50": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key51": newTempId("12345678901234567890123456789012345678901234567890"),
			"key52": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key53": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key54": newTempId("12345678901234567890123456789012345678901234567890"),
			"key55": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key56": newTempId("12345678901234567890123456789012345678901234567890"),
			"key57": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key58": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key59": newTempId("12345678901234567890123456789012345678901234567890"),

			"key60": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key61": newTempId("12345678901234567890123456789012345678901234567890"),
			"key62": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key63": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key64": newTempId("12345678901234567890123456789012345678901234567890"),
			"key65": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key66": newTempId("12345678901234567890123456789012345678901234567890"),
			"key67": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key68": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key69": newTempId("12345678901234567890123456789012345678901234567890"),

			"key70": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key71": newTempId("12345678901234567890123456789012345678901234567890"),
			"key72": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key73": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key74": newTempId("12345678901234567890123456789012345678901234567890"),
			"key75": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key76": newTempId("12345678901234567890123456789012345678901234567890"),
			"key77": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key78": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key79": newTempId("12345678901234567890123456789012345678901234567890"),

			"key80": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key81": newTempId("12345678901234567890123456789012345678901234567890"),
			"key82": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key83": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key84": newTempId("12345678901234567890123456789012345678901234567890"),
			"key85": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key86": newTempId("12345678901234567890123456789012345678901234567890"),
			"key87": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key88": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key89": newTempId("12345678901234567890123456789012345678901234567890"),

			"key90": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key91": newTempId("12345678901234567890123456789012345678901234567890"),
			"key92": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key93": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key94": newTempId("12345678901234567890123456789012345678901234567890"),
			"key95": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key96": newTempId("12345678901234567890123456789012345678901234567890"),
			"key97": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key98": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key99": newTempId("12345678901234567890123456789012345678901234567890"),

			"key101": newTempId("12345678901234567890123456789012345678901234567890"),
			"key102": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key103": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key104": newTempId("12345678901234567890123456789012345678901234567890"),
			"key105": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key106": newTempId("12345678901234567890123456789012345678901234567890"),
			"key107": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key108": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key109": newTempId("12345678901234567890123456789012345678901234567890"),

			"key110": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key111": newTempId("12345678901234567890123456789012345678901234567890"),
			"key112": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key113": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key114": newTempId("12345678901234567890123456789012345678901234567890"),
			"key115": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key116": newTempId("12345678901234567890123456789012345678901234567890"),
			"key117": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key118": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key119": newTempId("12345678901234567890123456789012345678901234567890"),

			"key120": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key121": newTempId("12345678901234567890123456789012345678901234567890"),
			"key122": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key123": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key124": newTempId("12345678901234567890123456789012345678901234567890"),
			"key125": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key126": newTempId("12345678901234567890123456789012345678901234567890"),
			"key127": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key128": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key129": newTempId("12345678901234567890123456789012345678901234567890"),

			"key130": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key131": newTempId("12345678901234567890123456789012345678901234567890"),
			"key132": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key133": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key134": newTempId("12345678901234567890123456789012345678901234567890"),
			"key135": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key136": newTempId("12345678901234567890123456789012345678901234567890"),
			"key137": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key138": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key139": newTempId("12345678901234567890123456789012345678901234567890"),

			"key140": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key141": newTempId("12345678901234567890123456789012345678901234567890"),
			"key142": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key143": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key144": newTempId("12345678901234567890123456789012345678901234567890"),
			"key145": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key146": newTempId("12345678901234567890123456789012345678901234567890"),
			"key147": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key148": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key149": newTempId("12345678901234567890123456789012345678901234567890"),

			"key150": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key151": newTempId("12345678901234567890123456789012345678901234567890"),
			"key152": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key153": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key154": newTempId("12345678901234567890123456789012345678901234567890"),
			"key155": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key156": newTempId("12345678901234567890123456789012345678901234567890"),
			"key157": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key158": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key159": newTempId("12345678901234567890123456789012345678901234567890"),

			"key160": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key161": newTempId("12345678901234567890123456789012345678901234567890"),
			"key162": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key163": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key164": newTempId("12345678901234567890123456789012345678901234567890"),
			"key165": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key166": newTempId("12345678901234567890123456789012345678901234567890"),
			"key167": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key168": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key169": newTempId("12345678901234567890123456789012345678901234567890"),

			"key170": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key171": newTempId("12345678901234567890123456789012345678901234567890"),
			"key172": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key173": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key174": newTempId("12345678901234567890123456789012345678901234567890"),
			"key175": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key176": newTempId("12345678901234567890123456789012345678901234567890"),
			"key177": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key178": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key179": newTempId("12345678901234567890123456789012345678901234567890"),

			"key180": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key181": newTempId("12345678901234567890123456789012345678901234567890"),
			"key182": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key183": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key184": newTempId("12345678901234567890123456789012345678901234567890"),
			"key185": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key186": newTempId("12345678901234567890123456789012345678901234567890"),
			"key187": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key188": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key189": newTempId("12345678901234567890123456789012345678901234567890"),

			"key190": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key191": newTempId("12345678901234567890123456789012345678901234567890"),
			"key192": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key193": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key194": newTempId("12345678901234567890123456789012345678901234567890"),
			"key195": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key196": newTempId("12345678901234567890123456789012345678901234567890"),
			"key197": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key198": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key199": newTempId("12345678901234567890123456789012345678901234567890"),

			"key200": newTempId("12345678901234567890123456789012345678901234567890"),
			"key201": newTempId("12345678901234567890123456789012345678901234567890"),
			"key202": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key203": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key204": newTempId("12345678901234567890123456789012345678901234567890"),
			"key205": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key206": newTempId("12345678901234567890123456789012345678901234567890"),
			"key207": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key208": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key209": newTempId("12345678901234567890123456789012345678901234567890"),

			"key210": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key211": newTempId("12345678901234567890123456789012345678901234567890"),
			"key212": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key213": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key214": newTempId("12345678901234567890123456789012345678901234567890"),
			"key215": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key216": newTempId("12345678901234567890123456789012345678901234567890"),
			"key217": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key218": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key219": newTempId("12345678901234567890123456789012345678901234567890"),

			"key220": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key221": newTempId("12345678901234567890123456789012345678901234567890"),
			"key222": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key223": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key224": newTempId("12345678901234567890123456789012345678901234567890"),
			"key225": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key226": newTempId("12345678901234567890123456789012345678901234567890"),
			"key227": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key228": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key229": newTempId("12345678901234567890123456789012345678901234567890"),

			"key230": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key231": newTempId("12345678901234567890123456789012345678901234567890"),
			"key232": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key233": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key234": newTempId("12345678901234567890123456789012345678901234567890"),
			"key235": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key236": newTempId("12345678901234567890123456789012345678901234567890"),
			"key237": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key238": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key239": newTempId("12345678901234567890123456789012345678901234567890"),

			"key240": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key241": newTempId("12345678901234567890123456789012345678901234567890"),
			"key242": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key243": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key244": newTempId("12345678901234567890123456789012345678901234567890"),
			"key245": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key246": newTempId("12345678901234567890123456789012345678901234567890"),
			"key247": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key248": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key249": newTempId("12345678901234567890123456789012345678901234567890"),

			"key250": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key251": newTempId("12345678901234567890123456789012345678901234567890"),
			"key252": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key253": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key254": newTempId("12345678901234567890123456789012345678901234567890"),
			"key255": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key256": newTempId("12345678901234567890123456789012345678901234567890"),
			"key257": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key258": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key259": newTempId("12345678901234567890123456789012345678901234567890"),

			"key260": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key261": newTempId("12345678901234567890123456789012345678901234567890"),
			"key262": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key263": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key264": newTempId("12345678901234567890123456789012345678901234567890"),
			"key265": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key266": newTempId("12345678901234567890123456789012345678901234567890"),
			"key267": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key268": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key269": newTempId("12345678901234567890123456789012345678901234567890"),

			"key270": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key271": newTempId("12345678901234567890123456789012345678901234567890"),
			"key272": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key273": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key274": newTempId("12345678901234567890123456789012345678901234567890"),
			"key275": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key276": newTempId("12345678901234567890123456789012345678901234567890"),
			"key277": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key278": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key279": newTempId("12345678901234567890123456789012345678901234567890"),

			"key280": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key281": newTempId("12345678901234567890123456789012345678901234567890"),
			"key282": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key283": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key284": newTempId("12345678901234567890123456789012345678901234567890"),
			"key285": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key286": newTempId("12345678901234567890123456789012345678901234567890"),
			"key287": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key288": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key289": newTempId("12345678901234567890123456789012345678901234567890"),

			"key290": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key291": newTempId("12345678901234567890123456789012345678901234567890"),
			"key292": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key293": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key294": newTempId("12345678901234567890123456789012345678901234567890"),
			"key295": newTempId("aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ"),
			"key296": newTempId("12345678901234567890123456789012345678901234567890"),
			"key297": newTempId("abcdefghijklmnopqrstuvwxyz"),
			"key298": newTempId("ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			"key299": newTempId("12345678901234567890123456789012345678901234567890"),
		},
		optOut:   false,
		birthday: timestamp(),
	}
	return bigCookie, len(bigCookie.uids)
}

func writeThenRead(cookie *PBSCookie) *PBSCookie {
	w := httptest.NewRecorder()
	cookie.SetCookieOnResponse(w, "mock-domain", 90*24*time.Hour)
	writtenCookie := w.HeaderMap.Get("Set-Cookie")

	header := http.Header{}
	header.Add("Cookie", writtenCookie)
	request := http.Request{Header: header}
	return ParsePBSCookieFromRequest(&request, &config.HostCookie{})
}
