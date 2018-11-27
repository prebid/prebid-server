package beachfront

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/stretchr/testify/assert"
)

func TestBeachfrontSyncer(t *testing.T) {
	syncer := NewBeachfrontSyncer(&config.Configuration{ExternalURL: "localhost", Adapters: map[string]config.Adapter{
		string(openrtb_ext.BidderBeachfront): {
			UserSyncURL: "localhost",
		},
	}})
	u := syncer.GetUsersyncInfo("0", "")
	assert.Equal(t, "localhost", u.URL)
	assert.Equal(t, "iframe", u.Type)
	assert.Equal(t, uint16(0), syncer.GDPRVendorID())
	assert.Equal(t, false, u.SupportCORS)
}

func TestBeachfrontNullSyncer(t *testing.T) {
	syncer := NewBeachfrontSyncer(&config.Configuration{ExternalURL: "localhost", Adapters: map[string]config.Adapter{
		string(openrtb_ext.BidderBeachfront): {
			PlatformID: "142",
		},
	}})
	u := syncer.GetUsersyncInfo("0", "")
	assert.Equal(t, "", u.URL)
	assert.Equal(t, "iframe", u.Type)
	assert.Equal(t, uint16(0), syncer.GDPRVendorID())
	assert.Equal(t, false, u.SupportCORS)
}
