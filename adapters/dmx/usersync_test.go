package dmx

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestDmxSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("https://dmx.districtm.io/s/v1/img/s/10007"))
	syncer := NewDmxSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("", "")
	assert.NoError(t, err)
	assert.Equal(t, "https://dmx.districtm.io/s/v1/img/s/10007", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 144, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
