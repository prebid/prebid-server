package source

import (
	"context"
	"encoding/json"
)

// NilSource reports every key as not found.
type NilSource[K comparable] struct{}

func (NilSource[K]) Fetch(context.Context, K) (json.RawMessage, bool, error) {
	return nil, false, nil
}
