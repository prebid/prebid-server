package marsmedia

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewMarsmediaSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("marsmedia", temp, adapters.SyncTypeRedirect)
}
