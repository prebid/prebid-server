package synacormedia

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/stretchr/testify/assert"
)

func TestSynacormediaSyncer(t *testing.T) {
	syncURL := "//sync.technoratimedia.com/services?srv=cs&pid=70&cb=localhost%2Fsetuid%3Fbidder%3Dsynacormedia%26uid%3D%5BUSER_ID%5D"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewSynacorMediaSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{})

	assert.NoError(t, err)
	assert.Equal(t, "//sync.technoratimedia.com/services?srv=cs&pid=70&cb=localhost%2Fsetuid%3Fbidder%3Dsynacormedia%26uid%3D%5BUSER_ID%5D", syncInfo.URL)
	assert.Equal(t, "iframe", syncInfo.Type)
	assert.False(t, syncInfo.SupportCORS)
}
