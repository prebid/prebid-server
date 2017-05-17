package register

import "github.com/prebid/prebid-server/adapters"

var Adapters map[string]adapters.Adapter

func init() {
	Adapters = map[string]adapters.Adapter{}
}

func Add(name string, adapter adapters.Adapter) {
	Adapters[name] = adapter
}
