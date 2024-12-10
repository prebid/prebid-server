package jsonutil

import (
	"testing"

	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

func Test_ParseIntoString(t *testing.T) {
	tests := []struct {
		name string
		b    []byte
		want *string
	}{
		{
			name: "empty",
		},
		{
			name: "quoted_1",
			b:    []byte(`"1"`),
			want: ptrutil.ToPtr("1"),
		},
		{
			name: "unquoted_1",
			b:    []byte(`1`),
			want: ptrutil.ToPtr("1"),
		},
		{
			name: "null",
			b:    []byte(`null`),
		},
		{
			name: "quoted_null",
			b:    []byte(`"null"`),
			want: ptrutil.ToPtr("null"),
		},
		{
			name: "quoted_hello",
			b:    []byte(`"hello"`),
			want: ptrutil.ToPtr("hello"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got *string
			ParseIntoString(tt.b, &got)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_ParseIntoStringPanic(t *testing.T) {
	assert.Panics(t, func() {
		ParseIntoString([]byte(`"123"`), nil)
	})
}
