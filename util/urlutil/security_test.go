package urlutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSafeHost(t *testing.T) {
	testCases := []struct {
		name string
		host string
		want bool
	}{
		{
			name: "hostname",
			host: "example.com",
			want: true,
		},
		{
			name: "subdomain",
			host: "api-us",
			want: true,
		},
		{
			name: "hostname with port",
			host: "example.com:8080",
			want: true,
		},
		{
			name: "empty",
			host: "",
			want: false,
		},
		{
			name: "path injection",
			host: "127.0.0.1:6060/debug/pprof",
			want: false,
		},
		{
			name: "fragment injection",
			host: "127.0.0.1:6060#",
			want: false,
		},
		{
			name: "query injection",
			host: "example.com?x=1",
			want: false,
		},
		{
			name: "userinfo injection",
			host: "example.com@127.0.0.1",
			want: false,
		},
		{
			name: "scheme injection",
			host: "http://127.0.0.1",
			want: false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result := IsSafeHost(test.host)
			assert.Equal(t, test.want, result)
		})
	}
}
