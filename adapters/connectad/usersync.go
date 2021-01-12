package connectad

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewConnectAdSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("connectad", 138, temp, adapters.SyncTypeIframe)
}
