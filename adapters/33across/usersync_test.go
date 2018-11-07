package ttx

import (
	"testing"

	"github.com/prebid/prebid-server/config"

	"github.com/stretchr/testify/assert"
)

func Test33AcrossSyncer(t *testing.T) {
	ttx := New33AcrossSyncer(&config.Configuration{
		ExternalURL: "localhost",
		Adapters: map[string]config.Adapter{
			"33across": {
				UserSyncURL: "https://ssc-cms.33across.com/ps",
				PartnerId:   "123",
			},
		},
	})
	syncInfo := ttx.GetUsersyncInfo("", "")
	assert.Equal(t, "https://ssc-cms.33across.com/ps/?ri=123&ru=localhost%2Fsetuid%3Fbidder%3Dttx%26uid%3D33XUSERID33X", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.False(t, syncInfo.SupportCORS)
}
