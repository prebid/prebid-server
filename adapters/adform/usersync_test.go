package adform

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/stretchr/testify/assert"
)

func TestAdformSyncer(t *testing.T) {
	syncer := NewAdformSyncer(&config.Configuration{ExternalURL: "localhost", Adapters: map[string]config.Adapter{
		string(openrtb_ext.BidderAdform): {
			UserSyncURL: "//cm.adform.net?return_url=",
		},
	}})
	u := syncer.GetUsersyncInfo("1", "BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw")
	assert.Equal(t, "//cm.adform.net?return_url=localhost%2Fsetuid%3Fbidder%3Dadform%26gdpr%3D1%26gdpr_consent%3DBONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw%26uid%3D%24UID", u.URL)
	assert.Equal(t, "redirect", u.Type)
	assert.Equal(t, uint16(50), syncer.GDPRVendorID())
	assert.Equal(t, false, u.SupportCORS)
}
