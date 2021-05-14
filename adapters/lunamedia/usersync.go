package lunamedia

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewLunaMediaSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("lunamedia", temp, adapters.SyncTypeIframe)
}
