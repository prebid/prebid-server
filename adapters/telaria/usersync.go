package telaria

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewTelariaSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("telaria", 202, temp, adapters.SyncTypeRedirect)
}
