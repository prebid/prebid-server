package ninthdecimal

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewNinthDecimalSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("ninthdecimal", temp, adapters.SyncTypeIframe)
}
