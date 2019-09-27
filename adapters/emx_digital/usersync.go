package emx_digital

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewEMXDigitalSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("emx_digital", 183, temp, adapters.SyncTypeIframe)
}
