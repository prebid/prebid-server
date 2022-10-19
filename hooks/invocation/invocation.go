package invocation

import (
	"time"
)

type Context struct {
	Endpoint string
	Timeout  time.Duration
	// todo: think on adding next fields
	// debugEnabled
	// moduleContext
}

type HookResult[T any] struct {
	Reject    bool
	Mutations []Mutation[T]
	// todo: think on adding next fields
	// analyticTags
	// errors
	// warnings
	// debug
	// moduleContext - arbitrary data the hook wishes to pass to downstream hooks of the same module
}
