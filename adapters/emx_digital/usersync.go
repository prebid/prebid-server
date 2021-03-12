package emx_digital

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewEMXDigitalSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("emx_digital", temp, adapters.SyncTypeIframe)
}
