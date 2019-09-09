package synacormedia

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"text/template"
)

func TestSynacormediaSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("//sync.technoratimedia.com/services?srv=cs&pid=70&cb=localhost%2Fsetuid%3Fbidder%3Dsynacormedia%26uid%3D%5BUSER_ID%5D"))
	syncer := NewSynacorMediaSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("", "")
	assert.NoError(t, err)
	assert.Equal(t, "//sync.technoratimedia.com/services?srv=cs&pid=70&cb=localhost%2Fsetuid%3Fbidder%3Dsynacormedia%26uid%3D%5BUSER_ID%5D", syncInfo.URL)
	assert.Equal(t, "iframe", syncInfo.Type)
	assert.EqualValues(t, 0, syncer.GDPRVendorID())
	assert.False(t, syncInfo.SupportCORS)
}
