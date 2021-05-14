package adxcg

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

func NewAdxcgSyncer(urlTemplate *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("adxcg", urlTemplate, adapters.SyncTypeRedirect)
}
