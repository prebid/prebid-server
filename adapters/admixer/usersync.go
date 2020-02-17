package admixer

import (
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
	"text/template"
)

func NewAdmixerSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("admixer", 511, temp, adapters.SyncTypeRedirect)
}
