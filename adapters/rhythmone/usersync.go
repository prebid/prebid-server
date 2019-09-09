package rhythmone

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewRhythmoneSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("rhythmone", 36, temp, adapters.SyncTypeRedirect)
}
