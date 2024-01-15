package openrtb_ext

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xeipuuv/gojsonschema"
)

// TestMain does the expensive setup so we don't keep re-reading the files in static/bidder-params for each test.
func TestMain(m *testing.M) {
	bidderParams, err := NewBidderParamsValidator("../static/bidder-params")
	if err != nil {
		os.Exit(1)
	}
	validator = bidderParams
	os.Exit(m.Run())
}

var validator BidderParamValidator

// TestBidderParamSchemas makes sure that the validator.Schema() function
// returns valid JSON for all known CoreBidderNames.
func TestBidderParamSchemas(t *testing.T) {
	for _, bidderName := range CoreBidderNames() {
		schema := validator.Schema(bidderName)
		if schema == "" {
			t.Errorf("No schema exists for bidder %s. Does static/bidder-params/%s.json exist?", bidderName, bidderName)
		}

		if _, err := gojsonschema.NewBytesLoader([]byte(schema)).LoadJSON(); err != nil {
			t.Errorf("static/bidder-params/%s.json does not have a valid json-schema. %v", bidderName, err)
		}
	}
}

func TestBidderNamesValid(t *testing.T) {
	for _, bidder := range CoreBidderNames() {
		isReserved := IsBidderNameReserved(string(bidder))
		assert.False(t, isReserved, "bidder %v conflicts with a reserved name", bidder)
	}
}

// TestBidderUniquenessGatekeeping acts as a gatekeeper of bidder name uniqueness. If this test fails
// when you're building a new adapter, please consider choosing a different bidder name to maintain the
// current uniqueness threshold, or else start a discussion in the PR.
func TestBidderUniquenessGatekeeping(t *testing.T) {
	// Get List Of Bidders
	// - Exclude duplicates of adapters for the same bidder, as it's unlikely a publisher will use both.
	var bidders []string
	for _, bidder := range CoreBidderNames() {
		if bidder != BidderSilverPush && bidder != BidderTripleliftNative && bidder != BidderAdkernelAdn && bidder != "freewheel-ssp" && bidder != "yahooAdvertising" {
			bidders = append(bidders, string(bidder))
		}
	}

	currentThreshold := 6
	measuredThreshold := minUniquePrefixLength(bidders)

	assert.NotZero(t, measuredThreshold, "BidderMap contains duplicate bidder name values.")
	assert.LessOrEqual(t, measuredThreshold, currentThreshold)
}

// minUniquePrefixLength measures the minimun amount of characters needed to uniquely identify
// one of the strings, or returns 0 if there are duplicates.
func minUniquePrefixLength(b []string) int {
	targetingKeyMaxLength := 20
	for prefixLength := 1; prefixLength <= targetingKeyMaxLength; prefixLength++ {
		if uniqueForPrefixLength(b, prefixLength) {
			return prefixLength
		}
	}
	return 0
}

func uniqueForPrefixLength(b []string, prefixLength int) bool {
	m := make(map[string]struct{})

	if prefixLength <= 0 {
		return false
	}

	for i, n := range b {
		ns := string(n)

		if len(ns) > prefixLength {
			ns = ns[0:prefixLength]
		}

		m[ns] = struct{}{}

		if len(m) != i+1 {
			return false
		}
	}

	return true
}
