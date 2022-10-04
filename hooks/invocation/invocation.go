package invocation

import "time"

type Action int

const (
	Nop Action = iota
	Update
	Reject
)

type Context struct {
	Endpoint string
	Timeout  time.Duration
	// todo: think on adding next fields
	// debugEnabled
	// moduleContext
}

type HookResult[T any] struct {
	Action    Action // Action indicates result caused by hook invocation
	Mutations []Mutation[T]
	// todo: think on adding next fields
	// analyticTags
	// errors
	// warnings
	// debug
	// moduleContext - arbitrary data the hook wishes to pass to downstream hooks of the same module
}

type HookResponse[T any] struct {
	Result HookResult[T]
	Err    error
}
