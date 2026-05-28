package urlutil

import "testing"

func TestIsSafeHost(t *testing.T) {
	testCases := []struct {
		name string
		host string
		want bool
	}{
		{name: "hostname", host: "example.com", want: true},
		{name: "subdomain", host: "api-us", want: true},
		{name: "hostname with port", host: "example.com:8080", want: true},
		{name: "empty", host: "", want: false},
		{name: "path injection", host: "127.0.0.1:6060/debug/pprof", want: false},
		{name: "fragment injection", host: "127.0.0.1:6060#", want: false},
		{name: "query injection", host: "example.com?x=1", want: false},
		{name: "userinfo injection", host: "example.com@127.0.0.1", want: false},
		{name: "scheme injection", host: "http://127.0.0.1", want: false},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			if got := IsSafeHost(test.host); got != test.want {
				t.Fatalf("IsSafeHost(%q) = %t, want %t", test.host, got, test.want)
			}
		})
	}
}

func TestIsSafePath(t *testing.T) {
	testCases := []struct {
		name string
		path string
		want bool
	}{
		{name: "relative path", path: "auction/rtb/v2", want: true},
		{name: "single segment", path: "endpoint", want: true},
		{name: "empty", path: "", want: false},
		{name: "query injection", path: "auction?x=1", want: false},
		{name: "fragment injection", path: "auction#fragment", want: false},
		{name: "scheme relative", path: "//127.0.0.1/admin", want: false},
		{name: "absolute url", path: "http://127.0.0.1/admin", want: false},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			if got := IsSafePath(test.path); got != test.want {
				t.Fatalf("IsSafePath(%q) = %t, want %t", test.path, got, test.want)
			}
		})
	}
}

func TestIsSafePathSegment(t *testing.T) {
	testCases := []struct {
		name    string
		segment string
		want    bool
	}{
		{name: "segment", segment: "seller-123", want: true},
		{name: "empty", segment: "", want: false},
		{name: "slash", segment: "seller/123", want: false},
		{name: "query", segment: "seller?x=1", want: false},
		{name: "fragment", segment: "seller#x", want: false},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			if got := IsSafePathSegment(test.segment); got != test.want {
				t.Fatalf("IsSafePathSegment(%q) = %t, want %t", test.segment, got, test.want)
			}
		})
	}
}
