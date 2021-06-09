package beintoo

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewBeintooSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("Beintoo", temp, adapters.SyncTypeIframe)
}
