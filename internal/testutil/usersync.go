package testutil

import (
	"testing"

	"github.com/prebid/prebid-server/usersync"
)

type usersyncTest struct {
	t          *testing.T
	usersyncer usersync.Usersyncer
	syncInfo   *usersync.UsersyncInfo
}

// Assert common values match.
func (u *usersyncTest) Assert(url, syncType string, vendor uint16, supportCORS bool) {
	if url != u.syncInfo.URL {
		u.t.Errorf("expected %s, got %s", url, u.syncInfo.URL)
	}
	if syncType != u.syncInfo.Type {
		u.t.Errorf("expected %s, got %s", syncType, u.syncInfo.Type)
	}
	if vendor != u.usersyncer.GDPRVendorID() {
		u.t.Errorf("expected %d, got %d", vendor, u.usersyncer.GDPRVendorID())
	}
	if supportCORS != u.syncInfo.SupportCORS {
		u.t.Errorf("should have been %v", supportCORS)
	}
}

// AssertFamilyName matches user sync's FamilyName with provided value.
func (u *usersyncTest) AssertFamilyName(familyName string) {
	if familyName != u.usersyncer.FamilyName() {
		u.t.Errorf("expected %s, got %s", familyName, u.usersyncer.FamilyName())
	}
}

func UsersyncTest(t *testing.T, usersyncer usersync.Usersyncer, syncInfo *usersync.UsersyncInfo) *usersyncTest {
	return &usersyncTest{
		t:          t,
		usersyncer: usersyncer,
		syncInfo:   syncInfo,
	}
}
