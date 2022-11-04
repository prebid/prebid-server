package hookstage

type InvocationContext struct {
}

type HookResult[T any] struct {
	Reject bool
}
