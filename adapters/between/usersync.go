package between

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

// NewBetweenSyncer returns "between" syncer
func NewBetweenSyncer(template *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("between", 724, template, adapters.SyncTypeRedirect)
}
