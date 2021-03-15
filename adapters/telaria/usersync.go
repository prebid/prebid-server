package telaria

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewTelariaSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("telaria", temp, adapters.SyncTypeRedirect)
}
