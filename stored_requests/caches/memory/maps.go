package memory

import (
	"encoding/json"
	"sync"

	"github.com/coocood/freecache"
	"github.com/golang/glog"
)

// This file contains an interface and some wrapper types for various types of "map-like" structures
// so that we can mix and match them inside the Cache implementation in cache.go.

// Interface which abstracts the common operations of sync.Map and the freecache.Cache
type mapLike interface {
	Get(id string) (json.RawMessage, bool)
	Set(id string, value json.RawMessage)
	Delete(id string)
}

// sync.Map wrapper which implements the interface
type pbsSyncMap struct {
	*sync.Map
}

func (m *pbsSyncMap) Get(id string) (json.RawMessage, bool) {
	val, ok := m.Map.Load(id)
	if ok {
		return val.(json.RawMessage), ok
	} else {
		return nil, ok
	}
}

func (m *pbsSyncMap) Set(id string, value json.RawMessage) {
	m.Map.Store(id, value)
}

func (m *pbsSyncMap) Delete(id string) {
	m.Map.Delete(id)
}

// lruCache wrapper which implements the interface
type pbsLRUCache struct {
	*freecache.Cache
	ttlSeconds int
}

func (m *pbsLRUCache) Get(id string) (json.RawMessage, bool) {
	val, err := m.Cache.Get([]byte(id))
	if err == nil {
		return val, true
	}
	if err != freecache.ErrNotFound {
		glog.Errorf("unexpected error from freecache: %v", err)
	}
	return val, false
}

func (m *pbsLRUCache) Set(id string, value json.RawMessage) {
	if err := m.Cache.Set([]byte(id), value, m.ttlSeconds); err != nil {
		glog.Errorf("error saving value in freecache: %v", err)
	}
}

func (m *pbsLRUCache) Delete(id string) {
	m.Cache.Del([]byte(id))
}
