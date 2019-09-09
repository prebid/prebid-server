package sharethrough

import (
	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
	"text/template"
)

func NewSharethroughSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("sharethrough", 80, temp, adapters.SyncTypeRedirect)
}
