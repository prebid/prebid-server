package openx

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewOpenxSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("openx", 69, temp, adapters.SyncTypeRedirect)
}
