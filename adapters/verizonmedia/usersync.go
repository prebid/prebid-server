package verizonmedia

import (
	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
	"text/template"
)

func NewVerizonMediaSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("verizonmedia", 25, temp, adapters.SyncTypeRedirect)
}
