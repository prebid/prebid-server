package between

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewBetweenSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("between", 729, temp, adapters.SyncTypeRedirect)
}
