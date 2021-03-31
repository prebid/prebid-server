package lockerdome

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewLockerDomeSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("lockerdome", temp, adapters.SyncTypeRedirect)
}
