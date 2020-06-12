package adman

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestAdmanSyncer(t *testing.T) {
	syncURL := "https://sync.admanmedia.com/pbs.gif?redir=http%3A%2F%2Flocalhost%3A8000%2Fsetuid%3Fbidder%3Dadman%26uid%3D%5BUID%5D"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewAdmanSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal: "0",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "https://sync.admanmedia.com/pbs.gif?redir=http%3A%2F%2Flocalhost%3A8000%2Fsetuid%3Fbidder%3Dadman%26uid%3D%5BUID%5D", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 149, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
