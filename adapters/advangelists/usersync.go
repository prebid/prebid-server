package advangelists

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewAdvangelistsSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("advangelists", 0, temp, adapters.SyncTypeIframe)
}
