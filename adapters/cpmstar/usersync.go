package cpmstar

import (
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

//NewCpmstarSyncer :
func NewCpmstarSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("cpmstar", 0, temp, adapters.SyncTypeRedirect)
}
