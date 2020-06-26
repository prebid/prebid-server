package aja

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewAJASyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("aja", 0, temp, adapters.SyncTypeRedirect)
}
