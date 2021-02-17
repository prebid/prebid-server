package deepintent

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

// NewDeepintentSyncer returns deepintent syncer
func NewDeepintentSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("deepintent", 541, temp, adapters.SyncTypeRedirect)
}
