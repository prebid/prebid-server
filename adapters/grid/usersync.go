package grid

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewGridSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("grid", 686, temp, adapters.SyncTypeRedirect)
}
