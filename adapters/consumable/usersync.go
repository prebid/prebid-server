package consumable

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewConsumableSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer(
		"consumable",
		temp,
		adapters.SyncTypeRedirect)
}
