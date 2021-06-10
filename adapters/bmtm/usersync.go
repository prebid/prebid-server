package bmtm

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewBmtmSyncer(template *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("bmtm", template, adapters.SyncTypeRedirect)
}
