package logger

import (
	"context"
	"fmt"
	"strings"
)

// ctxOrBg returns the context if it is not nil, otherwise returns the background context.
func ctxOrBg(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

// convertArgsToString converts args to a formatted string with ", " separator
func convertToString(msg any, args ...any) string {
	var parts []string

	// Convert msg to string
	parts = append(parts, fmt.Sprintf("%v", msg))

	// Convert each arg to string and append
	for _, arg := range args {
		parts = append(parts, fmt.Sprintf("%v", arg))
	}

	return strings.Join(parts, ", ")
}
