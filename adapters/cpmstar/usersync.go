package cpmstar

import (
	"text/template"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
)

//NewCpmstarSyncer :
func NewCpmstarSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("cpmstar", temp, adapters.SyncTypeRedirect)
}
