package adpone

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

const adponeFamilyName = "adpone"

func NewadponeSyncer(urlTemplate *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer(
		adponeFamilyName,
		urlTemplate,
		adapters.SyncTypeRedirect,
	)
}
