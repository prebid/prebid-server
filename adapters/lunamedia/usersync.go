package lunamedia

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewLunaMediaSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("lunamedia", 0, temp, adapters.SyncTypeIframe)
}
