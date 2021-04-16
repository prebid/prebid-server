package adformOpenRTB

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewAdformOpenRTBSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("adformOpenRTB", temp, adapters.SyncTypeRedirect)
}
