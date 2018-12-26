package gumgum

import (
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"

	"testing"
)

func TestGumGumSyncer(t *testing.T) {
	syncer := NewGumGumSyncer(&config.Configuration{ExternalURL: "localhost", Adapters: map[string]config.Adapter{
		string(openrtb_ext.BidderGumGum): {
			UserSyncURL: "https://rtb.gumgum.com/usync/prbds2s?gdpr={{gdpr}}&gdpr_consent={{gdpr_consent}}&r=",
		},
	}})
	u := syncer.GetUsersyncInfo("1", "BOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA")
	assert.Equal(t, "https://rtb.gumgum.com/usync/prbds2s?gdpr=1&gdpr_consent=BOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA&r=localhost%2Fsetuid%3Fbidder%3Dgumgum%26gdpr%3D1%26gdpr_consent%3DBOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA%26uid%3D", u.URL)
	assert.Equal(t, "iframe", u.Type)
	assert.Equal(t, uint16(61), syncer.GDPRVendorID())
	assert.Equal(t, false, u.SupportCORS)
}
