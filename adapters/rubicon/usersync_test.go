package rubicon

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/stretchr/testify/assert"
)

func TestRubiconSyncer(t *testing.T) {
	syncer := NewRubiconSyncer(&config.Configuration{Adapters: map[string]config.Adapter{
		string(openrtb_ext.BidderRubicon): {
			UserSyncURL: "https://pixel.rubiconproject.com/exchange/sync.php?p=prebid&gdpr={{gdpr}}&gdpr_consent={{gdpr_consent}}",
		},
	}})
	u := syncer.GetUsersyncInfo("0", "")
	assert.Equal(t, "https://pixel.rubiconproject.com/exchange/sync.php?p=prebid&gdpr=0&gdpr_consent=", u.URL)
	assert.Equal(t, "redirect", u.Type)
	assert.Equal(t, uint16(52), syncer.GDPRVendorID())
	assert.Equal(t, false, u.SupportCORS)
	assert.Equal(t, "rubicon", syncer.FamilyName())
}
