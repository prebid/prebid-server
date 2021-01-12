package adtelligent

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewAdtelligentSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("adtelligent", 410, temp, adapters.SyncTypeRedirect)
}
