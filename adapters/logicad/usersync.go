package logicad

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewLogicadSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("logicad", temp, adapters.SyncTypeRedirect)
}
