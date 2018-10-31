package pubmatic

import (
	"testing"

	"github.com/prebid/prebid-server/config"

	"github.com/stretchr/testify/assert"
)

func TestPubmaticSyncer(t *testing.T) {
	syncer := NewPubmaticSyncer(&config.Configuration{ExternalURL: "localhost"})
	u := syncer.GetUsersyncInfo("1", "BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw")
	assert.Equal(t, "//ads.pubmatic.com/AdServer/js/user_sync.html?predirect=localhost%2Fsetuid%3Fbidder%3Dpubmatic%26gdpr%3D1%26gdpr_consent%3DBONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw%26uid%3D", u.URL)
	assert.Equal(t, "iframe", u.Type)
	assert.Equal(t, uint16(76), syncer.GDPRVendorID())
	assert.Equal(t, false, u.SupportCORS)
}
