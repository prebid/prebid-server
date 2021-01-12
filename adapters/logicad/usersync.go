package logicad

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewLogicadSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("logicad", 0, temp, adapters.SyncTypeRedirect)
}
