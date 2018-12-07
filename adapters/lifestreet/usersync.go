package lifestreet

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewLifestreetSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("lifestreet", 67, temp, adapters.SyncTypeRedirect)
}
