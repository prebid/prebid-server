package hookstage

import "github.com/prebid/prebid-server/hooks/hookanalytics"

type InvocationContext struct {
}

type HookResult[T any] struct {
	Reject        bool
	AnalyticsTags hookanalytics.Analytics
}
