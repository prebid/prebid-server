package lifestreet

import (
	"testing"

	"github.com/prebid/prebid-server/config"

	"github.com/stretchr/testify/assert"
)

func TestLifestreetSyncer(t *testing.T) {
	syncer := NewLifestreetSyncer(&config.Configuration{ExternalURL: "localhost"})
	u := syncer.GetUsersyncInfo("0", "")
	assert.Equal(t, "//ads.lfstmedia.com/idsync/137062?synced=1&ttl=1s&rurl=localhost%2Fsetuid%3Fbidder%3Dlifestreet%26gdpr%3D0%26gdpr_consent%3D%26uid%3D%24%24visitor_cookie%24%24", u.URL)
	assert.Equal(t, "redirect", u.Type)
	assert.Equal(t, uint16(67), syncer.GDPRVendorID())
	assert.Equal(t, false, u.SupportCORS)
}
