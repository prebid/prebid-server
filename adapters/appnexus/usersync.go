package appnexus

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewAppnexusSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("adnxs", 32, temp, adapters.SyncTypeRedirect)
}
