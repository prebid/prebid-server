package doohqty

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLookupPathValue(t *testing.T) {
	request := newDOOHRequest(
		&openrtb2.DOOH{
			ID:        "screen-id",
			Name:      "screen-name",
			Publisher: &openrtb2.Publisher{ID: "publisher-id"},
		},
		openrtb2.Imp{ID: "imp-id", TagID: "tag-id"},
	)
	imp := request.GetImp()[0]

	assert.Equal(t, "screen-id", lookupPathValue(request, imp, lookupPathDOOHID))
	assert.Equal(t, "screen-name", lookupPathValue(request, imp, lookupPathDOOHName))
	assert.Equal(t, "publisher-id", lookupPathValue(request, imp, lookupPathDOOHPublisherID))
	assert.Equal(t, "imp-id", lookupPathValue(request, imp, lookupPathImpID))
	assert.Equal(t, "tag-id", lookupPathValue(request, imp, lookupPathImpTagID))
	assert.Empty(t, lookupPathValue(request, imp, "unsupported"))
	assert.Empty(t, lookupPathValue(nil, imp, lookupPathDOOHID))
}

func TestResolveImpressionLookupUsesFirstAvailablePath(t *testing.T) {
	request := newDOOHRequest(&openrtb2.DOOH{Name: "screen-name"}, openrtb2.Imp{ID: "imp-id"})

	lookup, ok := resolveImpressionLookup(request, request.GetImp()[0], testAccountID, []string{lookupPathDOOHID, lookupPathDOOHName, lookupPathImpID})

	require.True(t, ok)
	assert.Equal(t, lookupKey{AccountID: testAccountID, Path: lookupPathDOOHName, Key: "screen-name"}, lookup)
}

func TestResolveImpressionLookupsWarnsForUnresolvedImpression(t *testing.T) {
	request := newDOOHRequest(&openrtb2.DOOH{}, openrtb2.Imp{})

	assignments, uniqueLookups, warnings := resolveImpressionLookups(request, testAccountID, []string{lookupPathDOOHID, lookupPathImpID})

	assert.Empty(t, assignments)
	assert.Empty(t, uniqueLookups)
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "no DOOH qty lookup key resolved for imp index 0")
}
