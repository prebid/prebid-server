package ix

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/stretchr/testify/assert"
)

func TestIxSyncer(t *testing.T) {
	syncer := NewIxSyncer(&config.Configuration{ExternalURL: "localhost", Adapters: map[string]config.Adapter{
		string(openrtb_ext.BidderIx): {
			UserSyncURL: "//ssum-sec.casalemedia.com/usermatchredir?s=184932&cb=localhost%2Fsetuid%3Fbidder%3Dix%26gdpr%3D{{gdpr}}%26gdpr_consent%3D{{gdpr_consent}}%26uid%3D",
		},
	}})
	u := syncer.GetUsersyncInfo("", "")
	assert.Equal(t, "//ssum-sec.casalemedia.com/usermatchredir?s=184932&cb=localhost%2Fsetuid%3Fbidder%3Dix%26gdpr%3D%26gdpr_consent%3D%26uid%3D", u.URL)
	assert.Equal(t, "redirect", u.Type)
	assert.Equal(t, uint16(10), syncer.GDPRVendorID())
	assert.Equal(t, false, u.SupportCORS)
}
