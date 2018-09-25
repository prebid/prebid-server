package eplanning

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/stretchr/testify/assert"
)

func TestEPlanningSyncer(t *testing.T) {
	syncer := NewEPlanningSyncer(&config.Configuration{ExternalURL: "localhost", Adapters: map[string]config.Adapter{
		string(openrtb_ext.BidderEPlanning): {
			UserSyncURL: "http://sync.e-planning.net/um?uid",
		},
	}})
	u := syncer.GetUsersyncInfo("", "")
	assert.Equal(t, "http://sync.e-planning.net/um?uidlocalhost%2Fsetuid%3Fbidder%3Deplanning%26gdpr%3D%26gdpr_consent%3D%26uid%3D%24UID", u.URL)
	assert.Equal(t, "redirect", u.Type)
	assert.Equal(t, uint16(0), syncer.GDPRVendorID())
	assert.Equal(t, false, u.SupportCORS)
}
