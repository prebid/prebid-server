package ix

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func NewIxSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("ix", 10, temp, adapters.SyncTypeRedirect)
}
