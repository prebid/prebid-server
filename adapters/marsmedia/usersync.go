package marsmedia

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewMarsmediaSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("marsmedia", 0, temp, adapters.SyncTypeRedirect)
}
