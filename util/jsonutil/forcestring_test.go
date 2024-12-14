package jsonutil

import (
	"testing"

	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

func Test_ParseIntoString(t *testing.T) {
	tests := []struct {
		name    string
		b       []byte
		want    *string
		wantErr bool
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
			err := ParseIntoString(tt.b, &got)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_ParseIntoNilStringError(t *testing.T) {
	assert.Error(t, ParseIntoString([]byte(`"123"`), nil))
}
