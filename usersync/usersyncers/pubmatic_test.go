package usersyncers

import (
	"testing"
)

func TestPubmaticSyncer(t *testing.T) {
	pubmatic := NewPubmaticSyncer("localhost")
	info := pubmatic.GetUsersyncInfo("1", "BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw")
	assertStringsMatch(t, "//ads.pubmatic.com/AdServer/js/user_sync.html?predirect=localhost%2Fsetuid%3Fbidder%3Dpubmatic%26gdpr%3D1%26gdpr_consent%3DBONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw%26uid%3D", info.URL)
	assertStringsMatch(t, "iframe", info.Type)
	if info.SupportCORS != false {
		t.Fatalf("should have been false")
	}
	if pubmatic.GDPRVendorID() != 76 {
		t.Errorf("Wrong Appnexus GDPR VendorID. Got %d", pubmatic.GDPRVendorID())
	}
}
