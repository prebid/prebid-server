package openx

import (
	"testing"

	"github.com/prebid/prebid-server/config"

	"github.com/stretchr/testify/assert"
)

func TestOpenxSyncer(t *testing.T) {
	syncer := NewOpenxSyncer(&config.Configuration{ExternalURL: "localhost"})
	u := syncer.GetUsersyncInfo("", "")
	assert.Equal(t, "https://rtb.openx.net/sync/prebid?r=localhost%2Fsetuid%3Fbidder%3Dopenx%26gdpr%3D%26gdpr_consent%3D%26uid%3D%24%7BUID%7D", u.URL)
	assert.Equal(t, "redirect", u.Type)
	assert.Equal(t, uint16(69), syncer.GDPRVendorID())
	assert.Equal(t, false, u.SupportCORS)
}
