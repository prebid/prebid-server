package evolution

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewEvolutionSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("e_volution", temp, adapters.SyncTypeRedirect)
}
