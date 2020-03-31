package ninthdecimal

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewNinthdecimalSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("ninthdecimal", 61, temp, adapters.SyncTypeIframe)
}
