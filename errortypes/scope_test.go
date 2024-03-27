package errortypes

import (
	"errors"
	"testing"
)

func TestReadScope(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want Scope
	}{
		{
			name: "scope-debug",
			err:  &DebugWarning{Message: "scope is debug"},
			want: ScopeDebug,
		},
		{
			name: "scope-any",
			err:  &Warning{Message: "scope is any"},
			want: ScopeAny,
		},
		{
			name: "default-error",
			err:  errors.New("default error"),
			want: ScopeAny,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ReadScope(tt.err); got != tt.want {
				t.Errorf("ReadScope() = %v, want %v", got, tt.want)
			}
		})
	}
}
