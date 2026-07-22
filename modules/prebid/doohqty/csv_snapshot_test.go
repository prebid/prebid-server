package doohqty

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

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
