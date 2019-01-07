package beachfront

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestBeachfrontSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("localhost"))
	syncer := NewBeachfrontSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("", "")
	assert.NoError(t, err)
	assert.Equal(t, "localhost", syncInfo.URL)
	assert.Equal(t, "iframe", syncInfo.Type)
	assert.EqualValues(t, 0, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
