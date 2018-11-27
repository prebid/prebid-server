package rhythmone

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/stretchr/testify/assert"
)

func TestRhythmoneSyncer(t *testing.T) {
	syncer := NewRhythmoneSyncer(&config.Configuration{ExternalURL: "localhost", Adapters: map[string]config.Adapter{
		string(openrtb_ext.BidderRhythmone): {
			UserSyncURL: "https://sync.1rx.io/usersync2/rmphb?gdpr={{gdpr}}&gdpr_consent={{gdpr_consent}}&redir=",
		},
	}})
	u := syncer.GetUsersyncInfo("1", "BOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA")
	assert.Equal(t, "https://sync.1rx.io/usersync2/rmphb?gdpr=1&gdpr_consent=BOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA&redir=localhost%2Fsetuid%3Fbidder%3Drhythmone%26gdpr%3D1%26gdpr_consent%3DBOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA%26uid%3D%5BRX_UUID%5D", u.URL)
	assert.Equal(t, "redirect", u.Type)
	assert.Equal(t, uint16(36), syncer.GDPRVendorID())
	assert.Equal(t, false, u.SupportCORS)
}
