package logger

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSlogLogger_CtxOrBg tests the ctxOrBg utility function
func TestSlogLogger_CtxOrBg(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
	}{
		{
			name: "valid context",
			ctx:  context.Background(),
		},
		{
			name: "nil context",
			ctx:  nil,
		},
		{
			name: "context with value",
			ctx:  context.WithValue(context.Background(), "key", "value"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ctxOrBg(tt.ctx)
			assert.NotNil(t, result)

			if tt.ctx == nil {
				assert.Equal(t, context.Background(), result)
			} else {
				assert.Equal(t, tt.ctx, result)
			}
		})
	}
}

// BenchmarkCtxOrBg benchmarks the ctxOrBg function
func BenchmarkCtxOrBg(b *testing.B) {
	ctx := context.Background()

	b.Run("ValidContext", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ctxOrBg(ctx)
		}
	})

	b.Run("NilContext", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ctxOrBg(nil)
		}
	})
}

func TestConvertToString(t *testing.T) {
	tests := []struct {
		name     string
		msg      any
		args     []any
		expected string
	}{
		{
			name:     "msg only - string",
			msg:      "Hello",
			args:     nil,
			expected: "Hello",
		},
		{
			name:     "msg only - integer",
			msg:      42,
			args:     nil,
			expected: "42",
		},
		{
			name:     "msg only - boolean",
			msg:      true,
			args:     nil,
			expected: "true",
		},
		{
			name:     "msg only - nil",
			msg:      nil,
			args:     nil,
			expected: "<nil>",
		},
		{
			name:     "msg with one arg",
			msg:      "Hello",
			args:     []any{"world"},
			expected: "Hello, world",
		},
		{
			name:     "msg with multiple args - mixed types",
			msg:      "Count",
			args:     []any{123, true, "test"},
			expected: "Count, 123, true, test",
		},
		{
			name:     "msg with multiple args - all strings",
			msg:      "First",
			args:     []any{"Second", "Third", "Fourth"},
			expected: "First, Second, Third, Fourth",
		},
		{
			name:     "msg with multiple args - all numbers",
			msg:      1,
			args:     []any{2, 3.14, 42},
			expected: "1, 2, 3.14, 42",
		},
		{
			name:     "msg with nil args",
			msg:      "Message",
			args:     []any{nil, "test", nil},
			expected: "Message, <nil>, test, <nil>",
		},
		{
			name:     "empty string msg with args",
			msg:      "",
			args:     []any{"arg1", "arg2"},
			expected: ", arg1, arg2",
		},
		{
			name:     "msg with empty string args",
			msg:      "Message",
			args:     []any{"", "test", ""},
			expected: "Message, , test, ",
		},
		{
			name:     "complex types",
			msg:      "Data",
			args:     []any{[]int{1, 2, 3}, map[string]int{"a": 1}},
			expected: "Data, [1 2 3], map[a:1]",
		},
		{
			name:     "struct as msg",
			msg:      struct{ Name string }{Name: "Test"},
			args:     []any{"extra"},
			expected: "{Test}, extra",
		},
		{
			name:     "zero values",
			msg:      0,
			args:     []any{false, ""},
			expected: "0, false, ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToString(tt.msg, tt.args...)
			if result != tt.expected {
				t.Errorf("convertToString(%v, %v) = %q, want %q", tt.msg, tt.args, result, tt.expected)
			}
		})
	}
}

func TestConvertToStringEdgeCases(t *testing.T) {
	// Test with large number of arguments
	t.Run("many arguments", func(t *testing.T) {
		args := make([]any, 100)
		for i := range args {
			args[i] = i
		}
		result := convertToString("start", args...)

		// Check that it starts correctly
		if !strings.HasPrefix(result, "start, 0, 1, 2") {
			t.Errorf("Result should start with 'start, 0, 1, 2', got: %s", result[:20])
		}

		// Check that it ends correctly
		if !strings.HasSuffix(result, "98, 99") {
			t.Errorf("Result should end with '98, 99', got: %s", result[len(result)-10:])
		}
	})

	// Test with pointer types
	t.Run("pointer types", func(t *testing.T) {
		str := "test"
		num := 42
		result := convertToString("pointers", &str, &num)
		// The exact format might vary, but should contain the addresses
		if !strings.Contains(result, "pointers") {
			t.Errorf("Result should contain 'pointers', got: %s", result)
		}
	})
}

func BenchmarkConvertToString(b *testing.B) {
	b.Run("single arg", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			convertToString("message", "arg")
		}
	})

	b.Run("multiple args", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			convertToString("message", "arg1", 123, true, "arg4")
		}
	})

	b.Run("many args", func(b *testing.B) {
		args := []any{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
		for i := 0; i < b.N; i++ {
			convertToString("message", args...)
		}
	})
}
