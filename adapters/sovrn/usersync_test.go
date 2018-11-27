package sovrn

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/stretchr/testify/assert"
)

func TestSovrnSyncer(t *testing.T) {
	syncer := NewSovrnSyncer(&config.Configuration{ExternalURL: "external.com", Adapters: map[string]config.Adapter{
		string(openrtb_ext.BidderSovrn): {
			UserSyncURL: "//ap.lijit.com/pixel?",
		},
	}})
	u := syncer.GetUsersyncInfo("0", "")
	assert.Equal(t, "//ap.lijit.com/pixel?redir=external.com%2Fsetuid%3Fbidder%3Dsovrn%26gdpr%3D0%26gdpr_consent%3D%26uid%3D%24UID", u.URL)
	assert.Equal(t, "redirect", u.Type)
	assert.Equal(t, uint16(13), syncer.GDPRVendorID())
	assert.Equal(t, false, u.SupportCORS)
}
