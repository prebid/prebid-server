package ttx

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/stretchr/testify/assert"
)

func Test33AcrossSyncer(t *testing.T) {
	syncURL := "https://ic.tynt.com/r/d?m=xch&rt=html&ri=123&ru=%2Fsetuid%3Fbidder%3D33across%26uid%3D33XUSERID33X&id=zzz000000000002zzz"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := New33AcrossSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{})

	assert.NoError(t, err)
	assert.Equal(t, "https://ic.tynt.com/r/d?m=xch&rt=html&ri=123&ru=%2Fsetuid%3Fbidder%3D33across%26uid%3D33XUSERID33X&id=zzz000000000002zzz", syncInfo.URL)
	assert.Equal(t, "iframe", syncInfo.Type)
	assert.EqualValues(t, 58, syncer.GDPRVendorID())
	assert.False(t, syncInfo.SupportCORS)
}
