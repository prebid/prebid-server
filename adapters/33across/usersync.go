package ttx

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func New33AcrossSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("ttx", 58, temp, adapters.SyncTypeRedirect)
}
