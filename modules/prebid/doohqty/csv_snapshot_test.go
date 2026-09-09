package doohqty

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"testing/iotest"
	"time"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseImpressionValueCSV(t *testing.T) {
	values, warnings, err := parseImpressionValueCSV(testAccountID, strings.NewReader(` key , path , multiplier , sourcetype , vendor
screen-1,dooh.id,12.5,1,measurement.example
tag-1,imp.tagid,8.5,2,
bad,site.id,1,,
screen-1,dooh.id,13,1,measurement.example
screen-2,dooh.id,not-number,,
screen-3,dooh.id,5,1,
`))

	require.NoError(t, err)
	assert.Equal(t, 12.5, values[lookupKey{AccountID: testAccountID, Path: lookupPathDOOHID, Key: "screen-1"}].Multiplier)
	assert.Equal(t, 8.5, values[lookupKey{AccountID: testAccountID, Path: lookupPathImpTagID, Key: "tag-1"}].Multiplier)
	assert.Equal(t, adcom1.MultiplierPublisherProvided, values[lookupKey{AccountID: testAccountID, Path: lookupPathImpTagID, Key: "tag-1"}].SourceType)
	require.Len(t, warnings, 4)
	assert.Contains(t, warnings[0], `lookup path "site.id" is not supported`)
	assert.Contains(t, warnings[1], "duplicate value")
	assert.Contains(t, warnings[2], "multiplier is invalid")
	assert.Contains(t, warnings[3], "vendor is required")
}

func TestParseImpressionValueCSVHeaderRequired(t *testing.T) {
	_, _, err := parseImpressionValueCSV(testAccountID, strings.NewReader("path,key\n"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "CSV header must include path, key, and multiplier columns")
}

func TestCSVSnapshotSourceLookupLoadsAsyncSnapshot(t *testing.T) {
	client := &http.Client{Transport: doohQtyRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		assert.Equal(t, "text/csv", r.Header.Get("Accept"))
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("path,key,multiplier,sourcetype,vendor\ndooh.id,screen-1,10,2,\n")),
		}, nil
	})}

	cfg := defaultModuleConfig()
	cfg.Source.Type = sourceTypeCSVSnapshot
	cfg.Source.Endpoint = "https://values.example.com/dooh-qty.csv"
	cfg.Source.SyncRateSeconds = 300
	source := newCSVSnapshotSource(context.Background(), client)
	defer source.Shutdown()
	lookup := lookupKey{AccountID: testAccountID, Path: lookupPathDOOHID, Key: "screen-1"}

	values, warnings := source.Lookup(cfg, testAccountID, []lookupKey{lookup})

	assert.Empty(t, values)
	require.NotEmpty(t, warnings)
	assert.Contains(t, warnings[0], "CSV snapshot is loading")

	source.wg.Wait()
	values, warnings = source.Lookup(cfg, testAccountID, []lookupKey{lookup})

	require.Empty(t, warnings)
	assert.Equal(t, 10.0, values[lookup].Multiplier)
}

func TestCSVSnapshotSourceClosed(t *testing.T) {
	source := newCSVSnapshotSource(context.Background(), http.DefaultClient)
	source.Shutdown()

	values, warnings := source.Lookup(defaultModuleConfig(), testAccountID, nil)

	assert.Empty(t, values)
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "CSV source is shutting down")
}

func TestMaxBytesReaderAllowsExactLimit(t *testing.T) {
	data, err := io.ReadAll(newMaxBytesReader(strings.NewReader("0123456789"), 10))

	require.NoError(t, err)
	assert.Equal(t, "0123456789", string(data))
}

func TestMaxBytesReaderFailsPastLimit(t *testing.T) {
	_, err := io.ReadAll(newMaxBytesReader(strings.NewReader("0123456789X"), 10))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "CSV snapshot exceeds the 10 byte limit")
}

func TestParseImpressionValueCSVFailsOnTruncatedBody(t *testing.T) {
	body := "path,key,multiplier,sourcetype,vendor\ndooh.id,screen-1,12.5,2,\ndooh.id,screen-2,7,2,\n"

	values, _, err := parseImpressionValueCSV(testAccountID, strings.NewReader(body))
	require.NoError(t, err)
	require.Len(t, values, 2)

	values, _, err = parseImpressionValueCSV(testAccountID, newMaxBytesReader(strings.NewReader(body), 70))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "CSV snapshot exceeds the 70 byte limit")
	assert.Empty(t, values)
}

func TestParseImpressionValueCSVFailsOnReadError(t *testing.T) {
	body := io.MultiReader(
		strings.NewReader("path,key,multiplier,sourcetype,vendor\n"),
		iotest.ErrReader(errors.New("connection reset by peer")),
	)

	_, _, err := parseImpressionValueCSV(testAccountID, body)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection reset by peer")
}

func TestCSVSnapshotSourceOversizedRefreshKeepsLastSnapshot(t *testing.T) {
	const goodCSV = "path,key,multiplier,sourcetype,vendor\ndooh.id,screen-1,10,2,\n"

	oversizedCSV := goodCSV
	for i := 0; i < 20; i++ {
		oversizedCSV += "dooh.id,screen-1,99,2,\n"
	}

	var mu sync.Mutex
	calls := 0
	client := &http.Client{Transport: doohQtyRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		mu.Lock()
		defer mu.Unlock()
		calls++

		body := goodCSV
		if calls > 1 {
			body = oversizedCSV
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	})}

	cfg := defaultModuleConfig()
	cfg.Source.Type = sourceTypeCSVSnapshot
	cfg.Source.Endpoint = "https://values.example.com/dooh-qty.csv"
	cfg.Source.SyncRateSeconds = 0
	source := newCSVSnapshotSource(context.Background(), client)
	source.maxBodyBytes = int64(len(goodCSV))
	defer source.Shutdown()
	lookup := lookupKey{AccountID: testAccountID, Path: lookupPathDOOHID, Key: "screen-1"}

	source.Lookup(cfg, testAccountID, []lookupKey{lookup})
	source.wg.Wait()

	values, warnings := source.Lookup(cfg, testAccountID, []lookupKey{lookup})
	require.Empty(t, warnings)
	require.Equal(t, 10.0, values[lookup].Multiplier)
	source.wg.Wait()

	values, warnings = source.Lookup(cfg, testAccountID, []lookupKey{lookup})

	assert.Equal(t, 10.0, values[lookup].Multiplier)
	require.NotEmpty(t, warnings)
	assert.Contains(t, warnings[0], "CSV snapshot exceeds")
	assert.Contains(t, warnings[0], "using last successful snapshot")
}

func TestCSVSnapshotSourceUsesSyncTimeout(t *testing.T) {
	var deadline time.Time
	var hasDeadline bool
	client := &http.Client{Transport: doohQtyRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		deadline, hasDeadline = r.Context().Deadline()
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("path,key,multiplier,sourcetype,vendor\ndooh.id,screen-1,10,2,\n")),
		}, nil
	})}

	cfg := defaultModuleConfig()
	cfg.Source.Type = sourceTypeCSVSnapshot
	cfg.Source.Endpoint = "https://values.example.com/dooh-qty.csv"
	cfg.TimeoutMS = 50
	source := newCSVSnapshotSource(context.Background(), client)
	defer source.Shutdown()

	start := time.Now()
	source.Lookup(cfg, testAccountID, []lookupKey{{AccountID: testAccountID, Path: lookupPathDOOHID, Key: "screen-1"}})
	source.wg.Wait()

	require.True(t, hasDeadline)
	assert.Greater(t, deadline.Sub(start), time.Second, "CSV download must not inherit the auction-path timeout_ms")
}
