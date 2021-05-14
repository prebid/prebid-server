package sovrn

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewSovrnSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("sovrn", temp, adapters.SyncTypeRedirect)
}
