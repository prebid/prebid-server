package grid

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewGridSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("grid", temp, adapters.SyncTypeRedirect)
}
