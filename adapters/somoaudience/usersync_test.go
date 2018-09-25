package somoaudience

import (
	"testing"

	"github.com/prebid/prebid-server/config"

	"github.com/stretchr/testify/assert"
)

func TestSomoaudienceSyncer(t *testing.T) {
	syncer := NewSomoaudienceSyncer(&config.Configuration{ExternalURL: "localhost"})
	u := syncer.GetUsersyncInfo("", "")
	assert.Equal(t, "//publisher-east.mobileadtrading.com/usersync?ru=localhost%2Fsetuid%3Fbidder%3Dsomoaudience%26gdpr%3D%26gdpr_consent%3D%26uid%3D%24%7BUID%7D", u.URL)
	assert.Equal(t, "redirect", u.Type)
	assert.Equal(t, uint16(341), syncer.GDPRVendorID())
	assert.Equal(t, false, u.SupportCORS)
}
