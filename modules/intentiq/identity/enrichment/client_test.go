package enrichment

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetParsesResponse(t *testing.T) {
	var consent string
	srv := bodyServer(t, `{"data":{"eids":[{"source":"intentiq.com","uids":[{"id":"x"}]}]},`+
		`"cttl":60,"abTestUuid":"ab-1","tc":120088}`, &consent)

	got, err := NewClient(srv.Client()).Get(t.Context(), srv.URL, "")
	require.NoError(t, err)

	assert.Equal(t, "ab-1", got.AbTestUUID)
	require.NotNil(t, got.Tc)
	assert.Equal(t, int64(120088), *got.Tc)
	assert.Equal(t, 60*time.Second, got.CTTL())

	eids := got.Eids()
	require.Len(t, eids, 1)
	assert.Equal(t, "intentiq.com", eids[0].Source)
	require.Len(t, eids[0].UIDs, 1)
	assert.Equal(t, "x", eids[0].UIDs[0].ID)
}

func TestGetEmptyDataIsSuccess(t *testing.T) {
	var consent string
	srv := bodyServer(t, `{"data":"","cttl":30}`, &consent)

	got, err := NewClient(srv.Client()).Get(t.Context(), srv.URL, "")
	require.NoError(t, err)
	assert.Nil(t, got.Eids())
	assert.Equal(t, 30*time.Second, got.CTTL())
}

func TestGetSendsConsentHeader(t *testing.T) {
	var consent string
	srv := bodyServer(t, `{"data":""}`, &consent)
	c := NewClient(srv.Client())

	_, err := c.Get(t.Context(), srv.URL, "CONSENT-STRING")
	require.NoError(t, err)
	assert.Equal(t, "CONSENT-STRING", consent)

	_, err = c.Get(t.Context(), srv.URL, "")
	require.NoError(t, err)
	assert.Empty(t, consent, "no consent -> header not set")
}

func TestGetNon2xxCarriesStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := NewClient(srv.Client()).Get(t.Context(), srv.URL, "")

	var apiErr *Error
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, KindStatus, apiErr.Kind)
	assert.Equal(t, 500, apiErr.Status)
	kind, status := ErrorLabels(err)
	assert.Equal(t, "status", kind)
	assert.Equal(t, "500", status)
}

func TestGetUnreachableIsTransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	client := srv.Client()
	url := srv.URL
	srv.Close() // nothing is listening now

	_, err := NewClient(client).Get(t.Context(), url, "")

	var apiErr *Error
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, KindTransport, apiErr.Kind)
	assert.Zero(t, apiErr.Status, "no response arrived -> no status")
}

func TestGetDeadlineIsTimeoutError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	_, err := NewClient(srv.Client()).Get(ctx, srv.URL, "")

	var apiErr *Error
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, KindTimeout, apiErr.Kind)
	assert.ErrorIs(t, err, context.DeadlineExceeded, "the cause stays reachable through Unwrap")
}

func TestErrorString(t *testing.T) {
	assert.Equal(t, "status: boom", (&Error{Kind: KindStatus, Err: errors.New("boom")}).Error())
	assert.Equal(t, "status", (&Error{Kind: KindStatus}).Error(), "no cause -> kind alone")
	assert.Equal(t, "unknown", (&Error{}).Error(), "an error must never stringify to empty")

	cause := errors.New("boom")
	assert.Equal(t, cause, (&Error{Kind: KindParse, Err: cause}).Unwrap())
}

// A non-2xx must leave the connection reusable, so the pool does not churn during an error burst.
func TestGetReusesConnectionAfterNon2xx(t *testing.T) {
	peers := make(chan string, 2)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peers <- r.RemoteAddr
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"data":""}`))
	}))
	defer srv.Close()

	c := NewClient(srv.Client())
	for range 2 {
		_, err := c.Get(t.Context(), srv.URL, "")
		require.Error(t, err)
	}
	assert.Equal(t, <-peers, <-peers, "second request should reuse the first connection")
}

func TestResponseCTTL(t *testing.T) {
	assert.Equal(t, time.Duration(0), Response{}.CTTL(), "absent cttl -> 0")
	v := int64(30)
	assert.Equal(t, 30*time.Second, Response{Cttl: &v}.CTTL())
}

func TestClassifyAPIError(t *testing.T) {
	kind, status := ErrorLabels(&Error{Kind: KindStatus, Status: 500, Err: errors.New("boom")})
	assert.Equal(t, "status", kind)
	assert.Equal(t, "500", status)

	kind, status = ErrorLabels(&Error{Kind: KindTransport, Err: errors.New("boom")})
	assert.Equal(t, "transport", kind)
	assert.Empty(t, status, "no response received -> no status code")

	// A deadline that did not come from the client still classifies as a timeout.
	kind, status = ErrorLabels(fmt.Errorf("wrapped: %w", context.DeadlineExceeded))
	assert.Equal(t, "timeout", kind)
	assert.Empty(t, status)

	kind, _ = ErrorLabels(errors.New("unknown"))
	assert.Equal(t, "transport", kind, "unrecognised error -> transport")
}

func TestResponseEidsLenient(t *testing.T) {
	assert.Nil(t, Response{Data: []byte(`""`)}.Eids())
	assert.Nil(t, Response{Data: nil}.Eids())
	assert.Nil(t, Response{Data: []byte(`  `)}.Eids())

	got := Response{Data: []byte(`{"eids":[{"source":"intentiq.com","uids":[{"id":"x"}]}]}`)}.Eids()
	require.Len(t, got, 1)
	assert.Equal(t, "intentiq.com", got[0].Source)
}

func bodyServer(t *testing.T, body string, gotConsent *string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*gotConsent = r.Header.Get(GDPRConsentHeader)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// The status code says the call failed; the snippet says why, which is what reaches the logs.
func TestGetNon2xxCapturesBodySnippet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("invalid partner token"))
	}))
	defer srv.Close()

	_, err := NewClient(srv.Client()).Get(t.Context(), srv.URL, "")

	var apiErr *Error
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, "invalid partner token", apiErr.ResponseSnippet)
}

// A multi-line error page must not spread one log event over many lines.
func TestGetSnippetIsSingleLine(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("<html>\n  <body>\n    502 Bad Gateway\n  </body>\n</html>"))
	}))
	defer srv.Close()

	_, err := NewClient(srv.Client()).Get(t.Context(), srv.URL, "")

	var apiErr *Error
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, "<html> <body> 502 Bad Gateway </body> </html>", apiErr.ResponseSnippet)
	assert.NotContains(t, apiErr.ResponseSnippet, "\n")
}

// The snippet is destined for the logs, so a huge error body must not be copied there whole.
func TestGetSnippetIsCapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(strings.Repeat("x", 10_000)))
	}))
	defer srv.Close()

	_, err := NewClient(srv.Client()).Get(t.Context(), srv.URL, "")

	var apiErr *Error
	require.ErrorAs(t, err, &apiErr)
	assert.Len(t, apiErr.ResponseSnippet, maxSnippetBytes)
}

// A successful resolution carries no snippet to log.
func TestGetSuccessHasNoSnippet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"eids":[]}}`))
	}))
	defer srv.Close()

	resp, err := NewClient(srv.Client()).Get(t.Context(), srv.URL, "")

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.Status)
}
