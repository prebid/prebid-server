package huaweiads

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewSyncer(template *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("huaweiads", template, adapters.SyncTypeRedirect)
}