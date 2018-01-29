package usersync

import "github.com/prebid/prebid-server/pbs"

// Usersyncer is the interface for objects which have usersync info.
// This can be returned to the browser so that the Prebid Server host knows which IDs
// to send to each Bidder.s
type Usersyncer interface {
	GetUsersyncInfo() *pbs.UsersyncInfo
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
