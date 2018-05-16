package gdpr

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestVendorFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(mockServer(map[int]string{1: "{}"})))
	defer server.Close()
}

func testURLMaker(client *httptest.Server) func(uint16) string {
	url := client.URL
	return func(version uint16) string {
		return url + "?version=" + strconv.Itoa(int(version))
	}
}

// mockServer returns a handler which returns the given response for each global vendor list version.
//
// If the "version" query param doesn't exist, it returns a 400.
//
// If the "version" query param points to a version which doesn't exist, it returns a 403.
// Don't ask why... that's just what the official page is doing. See https://vendorlist.consensu.org/v-9999/vendorlist.json
func mockServer(responses map[int]string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		version := req.URL.Query().Get("version")
		versionInt, err := strconv.Atoi(version)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Request had invalid version: " + version))
			return
		}
		response, ok := responses[versionInt]
		if !ok {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("Version not found: " + version))
			return
		}
		w.Write([]byte(response))
	}
}
