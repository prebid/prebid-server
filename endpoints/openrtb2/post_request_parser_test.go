package openrtb2

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prebid/prebid-server/v4/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gzipBytes compresses the given payload so tests can exercise the
// Content-Encoding negotiation path of readRequestBody.
func gzipBytes(t *testing.T, payload []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, err := w.Write(payload)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return buf.Bytes()
}

// errReader fails on Read to simulate a broken request body.
type errReader struct {
	err error
}

func (r *errReader) Read(_ []byte) (int, error) { return 0, r.err }
func (r *errReader) Close() error               { return nil }

func TestReadRequestBody(t *testing.T) {
	body := []byte(`{"id":"some-request-id","imp":[{"id":"imp-id"}]}`)

	testCases := []struct {
		name            string
		body            io.Reader
		contentEncoding string
		cfg             *config.Configuration
		expectedBody    []byte
		expectedErr     string
	}{
		{
			name:         "plain body within limit",
			body:         bytes.NewReader(body),
			cfg:          &config.Configuration{MaxRequestSize: 1024},
			expectedBody: body,
		},
		{
			name:         "empty body",
			body:         bytes.NewReader(nil),
			cfg:          &config.Configuration{MaxRequestSize: 1024},
			expectedBody: []byte{},
		},
		{
			name:            "gzip body when gzip enabled",
			body:            bytes.NewReader(gzipBytes(t, body)),
			contentEncoding: "gzip",
			cfg: &config.Configuration{
				MaxRequestSize: 1024,
				Compression:    config.Compression{Request: config.CompressionInfo{GZIP: true}},
			},
			expectedBody: body,
		},
		{
			name:            "gzip body when gzip disabled",
			body:            bytes.NewReader(gzipBytes(t, body)),
			contentEncoding: "gzip",
			cfg: &config.Configuration{
				MaxRequestSize: 1024,
				Compression:    config.Compression{Request: config.CompressionInfo{GZIP: false}},
			},
			expectedErr: "Content-Encoding of type gzip is not supported",
		},
		{
			name:            "unsupported content encoding",
			body:            bytes.NewReader(body),
			contentEncoding: "deflate",
			cfg: &config.Configuration{
				MaxRequestSize: 1024,
				Compression:    config.Compression{Request: config.CompressionInfo{GZIP: true}},
			},
			expectedErr: "Content-Encoding of type deflate is not supported",
		},
		{
			name:            "gzip enabled but body is not valid gzip",
			body:            bytes.NewReader(body),
			contentEncoding: "gzip",
			cfg: &config.Configuration{
				MaxRequestSize: 1024,
				Compression:    config.Compression{Request: config.CompressionInfo{GZIP: true}},
			},
			expectedErr: "gzip: invalid header",
		},
		{
			name:        "body larger than max size",
			body:        bytes.NewReader(body),
			cfg:         &config.Configuration{MaxRequestSize: 10},
			expectedErr: "request size exceeded max size of 10 bytes.",
		},
		{
			// LimitedReader hits N == 0, but the underlying reader is already at EOF,
			// so a body of exactly MaxRequestSize bytes is accepted.
			name:         "body exactly at max size",
			body:         bytes.NewReader(body),
			cfg:          &config.Configuration{MaxRequestSize: int64(len(body))},
			expectedBody: body,
		},
		{
			name:         "body one byte below max size boundary",
			body:         bytes.NewReader(body),
			cfg:          &config.Configuration{MaxRequestSize: int64(len(body)) + 1},
			expectedBody: body,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			httpReq := httptest.NewRequest(http.MethodPost, "/openrtb2/auction", test.body)
			if test.contentEncoding != "" {
				httpReq.Header.Set("Content-Encoding", test.contentEncoding)
			}

			result, err := readRequestBody(httpReq, test.cfg)

			if test.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.expectedErr)
				assert.Nil(t, result)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expectedBody, result)
		})
	}
}

func TestReadRequestBodyReadError(t *testing.T) {
	readErr := errors.New("some read error")
	httpReq := httptest.NewRequest(http.MethodPost, "/openrtb2/auction", nil)
	httpReq.Body = &errReader{err: readErr}

	result, err := readRequestBody(httpReq, &config.Configuration{MaxRequestSize: 1024})

	require.Error(t, err)
	assert.Equal(t, readErr, err)
	assert.Nil(t, result)
}

func TestReadRequestBodyGzipRoundTripLargePayload(t *testing.T) {
	payload := []byte(`{"id":"` + strings.Repeat("a", 5000) + `"}`)
	compressed := gzipBytes(t, payload)

	httpReq := httptest.NewRequest(http.MethodPost, "/openrtb2/auction", bytes.NewReader(compressed))
	httpReq.Header.Set("Content-Encoding", "gzip")

	cfg := &config.Configuration{
		MaxRequestSize: int64(len(payload)) + 1,
		Compression:    config.Compression{Request: config.CompressionInfo{GZIP: true}},
	}

	result, err := readRequestBody(httpReq, cfg)

	require.NoError(t, err)
	assert.Equal(t, payload, result)
}

func TestReadRequestBodyDrainsOversizedBody(t *testing.T) {
	// An oversized body must be fully drained so the connection can be reused.
	body := strings.Repeat("x", 500)
	httpReq := httptest.NewRequest(http.MethodPost, "/openrtb2/auction", strings.NewReader(body))

	result, err := readRequestBody(httpReq, &config.Configuration{MaxRequestSize: 100})

	require.Error(t, err)
	assert.Nil(t, result)

	remaining, readErr := io.ReadAll(httpReq.Body)
	require.NoError(t, readErr)
	assert.Empty(t, remaining, "request body should be fully drained")
}
