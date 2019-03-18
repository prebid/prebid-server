package ttx

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func Test33AcrossSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("https://ssc-cms.33across.com/ps/?ri=123&ru=localhost%2Fsetuid%3Fbidder%3Dttx%26uid%3D33XUSERID33X"))
	syncer := New33AcrossSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("", "")
	assert.NoError(t, err)
	assert.Equal(t, "https://ssc-cms.33across.com/ps/?ri=123&ru=localhost%2Fsetuid%3Fbidder%3Dttx%26uid%3D33XUSERID33X", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 58, syncer.GDPRVendorID())
	assert.False(t, syncInfo.SupportCORS)
}
