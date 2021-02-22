package improvedigital

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewImprovedigitalSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("improvedigital", temp, adapters.SyncTypeRedirect)
}
