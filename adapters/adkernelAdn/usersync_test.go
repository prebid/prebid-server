package adkernelAdn

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/stretchr/testify/assert"
)

func TestAdkernelAdnSyncer(t *testing.T) {
	syncer := NewAdkernelAdnSyncer(&config.Configuration{ExternalURL: "https://localhost:8888", Adapters: map[string]config.Adapter{
		string(openrtb_ext.BidderAdkernelAdn): {
			UserSyncURL: "https://tag.adkernel.com/syncr?gdpr={{gdpr}}&gdpr_consent={{gdpr_consent}}&r=",
		},
	}})
	u := syncer.GetUsersyncInfo("1", "BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw")
	assert.Equal(t, "https://tag.adkernel.com/syncr?gdpr=1&gdpr_consent=BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw&r=https%3A%2F%2Flocalhost%3A8888%2Fsetuid%3Fbidder%3DadkernelAdn%26uid%3D%7BUID%7D", u.URL)
	assert.Equal(t, "redirect", u.Type)
	assert.Equal(t, adkernelGDPRVendorID, syncer.GDPRVendorID())
	assert.Equal(t, false, u.SupportCORS)
}
