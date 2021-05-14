package admixer

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewAdmixerSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("admixer", temp, adapters.SyncTypeRedirect)
}
