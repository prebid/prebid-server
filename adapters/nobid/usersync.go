package nobid

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewNoBidSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("nobid", temp, adapters.SyncTypeRedirect)
}
