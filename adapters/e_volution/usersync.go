package evolution

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewEvolutionSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("evolution", temp, adapters.SyncTypeRedirect)
}
