package optimization

import (
	"sync"
	"time"

	"github.com/prebid/prebid-server/v3/modules/prebid/optimization/rulesengine"
)

// Is sync.Map the best choice for our use case? Would it better to use a go map with mutex?
// https://pkg.go.dev/sync/atomic#Pointer

// TTL expiration check every 5 min
// When TTL expires, perform raw JSON hash diff to determine if tree rebuild is needed

type hash string
type stage string

type cacheObject struct {
	timestamp    time.Time
	hashedConfig hash
	ruleSets     map[stage][]cacheRuleSet
}
type cacheRuleSet struct {
	name        string
	modelGroups []cacheModelGroup
}
type cacheModelGroup struct {
	weight       int
	version      string
	analyticsKey string
	defaults     []rulesengine.ResultFunction
	root         rulesengine.Node
}

func NewCacheObject(tree rulesengine.Tree) (cacheObject, error) {
	return cacheObject{}, nil
}

type cacher interface {
	Get(string) *cacheObject
	Set(string, cacheObject)
	Delete(id string)
}

type cache struct {
	*sync.Map
}

func (c *cache) Get(id string) (data *cacheObject) {
	if val, ok := c.Map.Load(id); ok {
		return val.(*cacheObject)
	}
	return nil
}

func (c *cache) Set(id string, data cacheObject) {
	c.Map.Store(id, data)
}

func (c *cache) Delete(id string) {
	c.Map.Delete(id)
}
