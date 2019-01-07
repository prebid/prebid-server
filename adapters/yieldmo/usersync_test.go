package yieldmo

import (
	"testing"

	"github.com/prebid/prebid-server/config"

	"github.com/stretchr/testify/assert"
)

var yieldmo = NewYieldmoSyncer(&config.Configuration{
	ExternalURL: "external-url.com",
	Adapters: map[string]config.Adapter{
		"yieldmo": {
			UserSyncURL: "https://ads.yieldmo.com/pbsync?",
		},
	},
})

func TestYieldmoSyncerWithGdpr(t *testing.T) {
	syncInfo := yieldmo.GetUsersyncInfo("1", "")
	assert.Equal(t, "https://ads.yieldmo.com/pbsync?gdpr=1&gdpr_consent=&redirectUri=external-url.com%2Fsetuid%3Fbidder%3Dyieldmo%26gdpr%3D1%26gdpr_consent%3D%26uid%3D%24UID", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.False(t, syncInfo.SupportCORS)
}

func TestYieldmoSyncerWithoutGdpr(t *testing.T) {
	syncInfo := yieldmo.GetUsersyncInfo("0", "")
	assert.Equal(t, "https://ads.yieldmo.com/pbsync?gdpr=0&gdpr_consent=&redirectUri=external-url.com%2Fsetuid%3Fbidder%3Dyieldmo%26gdpr%3D0%26gdpr_consent%3D%26uid%3D%24UID", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.False(t, syncInfo.SupportCORS)
}
