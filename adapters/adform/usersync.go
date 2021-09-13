package adform

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewAdformSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("adform", temp, adapters.SyncTypeRedirect)
}
