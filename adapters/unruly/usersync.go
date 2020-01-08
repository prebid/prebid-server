package unruly

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewUnrulySyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("unruly", 162, temp, adapters.SyncTypeIframe)
}
