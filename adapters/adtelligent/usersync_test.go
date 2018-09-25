package adtelligent

import (
	"testing"

	"github.com/prebid/prebid-server/config"

	"github.com/stretchr/testify/assert"
)

func TestAdtelligentSyncer(t *testing.T) {
	syncer := NewAdtelligentSyncer(&config.Configuration{ExternalURL: "localhost"})
	u := syncer.GetUsersyncInfo("0", "")
	assert.Equal(t, "//sync.adtelligent.com/csync?t=p&ep=0&redir=localhost%2Fsetuid%3Fbidder%3Dadtelligent%26gdpr%3D0%26gdpr_consent%3D%26uid%3D%7Buid%7D", u.URL)
	assert.Equal(t, "redirect", u.Type)
	assert.Equal(t, uint16(0), syncer.GDPRVendorID())
	assert.Equal(t, false, u.SupportCORS)
}
