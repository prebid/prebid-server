package usersync

import "github.com/prebid/prebid-server/pbs"

type Usersyncer interface {
	// GetUsersyncInfo returns basic info the browser needs in order to run a user sync.
	// The returned UsersyncInfo object must not be mutated by callers.
	//
	// For more information about user syncs, see http://clearcode.cc/2015/12/cookie-syncing/
	GetUsersyncInfo() *pbs.UsersyncInfo
	// FamilyName identifies the space of cookies for this usersyncer.
	// For example, if this Usersyncer syncs with adnxs.com, then this
	// should return "adnxs".
	FamilyName() string
}

type syncer struct {
	familyName string
	syncInfo   *pbs.UsersyncInfo
}

func (s *syncer) GetUsersyncInfo() *pbs.UsersyncInfo {
	return s.syncInfo
}

func (s *syncer) FamilyName() string {
	return s.familyName
}
