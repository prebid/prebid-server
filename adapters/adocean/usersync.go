package adocean

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewAdOceanSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("adocean", 328, temp, adapters.SyncTypeRedirect)
}
