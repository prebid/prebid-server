package brightroll

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewBrightrollSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("brightroll", temp, adapters.SyncTypeRedirect)
}
