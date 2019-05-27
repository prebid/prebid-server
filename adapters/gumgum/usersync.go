package gumgum

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewGumGumSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("gumgum", 61, temp, adapters.SyncTypeIframe)
}
