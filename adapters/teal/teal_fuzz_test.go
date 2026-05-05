package teal

import (
	"encoding/json"
	"testing"
	"unicode/utf8"

	"github.com/prebid/openrtb/v20/openrtb2"
)

// Fuzz tests for the Teal adapter helpers. These never assert specific outputs —
// they only enforce the panic-free + invariant contract on arbitrary input.
//
// Run locally with:
//
//	go test -fuzz=FuzzParseImpExt -fuzztime=30s ./adapters/teal/...
//	go test -fuzz=FuzzMergeBidsPBSFlag -fuzztime=30s ./adapters/teal/...
//	go test -fuzz=FuzzModifyImp -fuzztime=30s ./adapters/teal/...
//
// In CI, run them as standard tests (the seed corpus only) by using `go test`.
// History: FuzzMergeBidsPBSFlag and FuzzModifyImp originally surfaced a
// nil-map panic on JSON `null` input — fixed in Iter 3 by routing both call
// sites through decodeJSONObject, which guarantees a non-nil receiver. The
// pinning unit test is `TestMergeBidsPBSFlag_NullInputHandledAsEmpty` /
// `TestModifyImp_NullImpExtHandledAsEmpty`.

// fuzzSeedExtCorpus is shared between FuzzParseImpExt and FuzzModifyImp.
// Each entry is chosen for a distinct historical JSON-parser edge case.
var fuzzSeedExtCorpus = [][]byte{
	// 1. Canonical happy-path imp.ext.
	[]byte(`{"bidder":{"account":"a","placement":"p"}}`),
	// 2. Empty object — boundary case.
	[]byte(`{}`),
	// 3. Empty bidder block.
	[]byte(`{"bidder":{}}`),
	// 4. Top-level array — wrong shape entirely.
	[]byte(`[]`),
	// 5. JSON null — neither object nor array. Iter 2 panic seed.
	[]byte(`null`),
	// 6. Number at top level.
	[]byte(`42`),
	// 7. String at top level.
	[]byte(`"plaintext"`),
	// 8. Nested arrays / objects with deep keys.
	[]byte(`{"bidder":{"account":"a","extra":{"deep":[1,2,3,{"x":"y"}]}}}`),
	// 9. Unicode escape sequences in account / placement.
	[]byte(`{"bidder":{"account":"ab","placement":"  "}}`),
	// 10. Account explicitly null (typed-as-string field with null value).
	[]byte(`{"bidder":{"account":null}}`),
	// 11. Placement explicitly null (allowed — same as absent).
	[]byte(`{"bidder":{"account":"a","placement":null}}`),
	// 12. Trailing garbage after a valid object.
	[]byte(`{"bidder":{"account":"a"}}xyz`),
	// 13. Bare empty bytes.
	[]byte(``),
	// 14. Single open brace — truncated JSON.
	[]byte(`{`),
	// 15. Embedded prebid object with unrelated sub-object.
	[]byte(`{"bidder":{"account":"a","placement":"p"},"prebid":{"foo":"bar"}}`),
	// 16. Pathological depth — nested objects (jsoniter has a default depth limit).
	[]byte(`{"a":{"a":{"a":{"a":{"a":{"a":{}}}}}}}`),
}

// FuzzParseImpExt — parseImpExt must never panic on arbitrary input. When it
// returns nil error, the resulting *ExtImpTeal must be non-nil. When it
// returns an error, the message must contain the impression-id template.
func FuzzParseImpExt(f *testing.F) {
	for _, seed := range fuzzSeedExtCorpus {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		imp := &openrtb2.Imp{ID: "fuzz-imp", Ext: data}
		ext, err := parseImpExt(imp)
		if err != nil {
			// On error: ext must be nil; error message must contain the impid.
			if ext != nil {
				t.Fatalf("parseImpExt returned both non-nil ext and non-nil err for input %q", string(data))
			}
			if !contains(err.Error(), "Error parsing imp.ext for impression") {
				t.Fatalf("parseImpExt error must use the canonical template; got %q", err.Error())
			}
			return
		}
		// On success: ext must be non-nil. Account may be empty (validation
		// is parseImpExt's caller's responsibility).
		if ext == nil {
			t.Fatalf("parseImpExt returned nil ext with nil err for input %q", string(data))
		}
	})
}

// FuzzMergeBidsPBSFlag — for any object-shaped input the function either
// returns an error (when the input was invalid JSON) or returns a marshalable
// object whose "bids" key is exactly {"pbs":1}.
func FuzzMergeBidsPBSFlag(f *testing.F) {
	seeds := [][]byte{
		// Empty input — the no-existing branch.
		nil,
		[]byte(`{}`),
		[]byte(`{"foo":1}`),
		[]byte(`{"bids":{"old":true}}`), // overwrite branch
		[]byte(`{"bids":42}`),           // non-object existing bids — still overwritten
		[]byte(`{"prebid":{"server":{"ttl":3600}}}`),
		[]byte(`null`),  // Iter 2 panic seed; Iter 3 fix expects empty-handling.
		[]byte(`[]`),    // wrong shape — should error
		[]byte(`"abc"`), // string at top level — wrong shape, should error
		// Pathological keys.
		[]byte(`{"":""}`),
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		out, err := mergeBidsPBSFlag(json.RawMessage(data))
		if err != nil {
			if out != nil {
				t.Fatalf("mergeBidsPBSFlag returned non-nil out alongside err for input %q", string(data))
			}
			return
		}
		// Success path: out must be a valid JSON object with bids:{"pbs":1}.
		var parsed map[string]json.RawMessage
		if jerr := json.Unmarshal(out, &parsed); jerr != nil {
			t.Fatalf("mergeBidsPBSFlag produced unmarshalable output %q for input %q", string(out), string(data))
		}
		bids, ok := parsed["bids"]
		if !ok {
			t.Fatalf("mergeBidsPBSFlag output missing bids key; out=%q input=%q", string(out), string(data))
		}
		if string(bids) != `{"pbs":1}` {
			t.Fatalf("mergeBidsPBSFlag bids must equal {\"pbs\":1}; got %q for input %q", string(bids), string(data))
		}
	})
}

// FuzzModifyImp — modifyImp must never panic. With a non-nil placement the
// returned imp's ext (when err is nil) must round-trip to JSON, and the
// storedrequest.id path must equal the placement string.
func FuzzModifyImp(f *testing.F) {
	for _, seed := range fuzzSeedExtCorpus {
		f.Add(seed, "fuzz-placement")
	}
	// Edge-case placements.
	f.Add([]byte(`{}`), "")
	f.Add([]byte(`{}`), "  ")
	f.Add([]byte(`{}`), "\t\n")
	f.Add([]byte(`{}`), "very-long-"+string(make([]byte, 1024)))

	f.Fuzz(func(t *testing.T, extData []byte, placement string) {
		imp := &openrtb2.Imp{ID: "fuzz-imp", Ext: extData}
		got, err := modifyImp(imp, &placement)
		if err != nil {
			if got != nil {
				t.Fatalf("modifyImp returned non-nil imp alongside err for input %q", string(extData))
			}
			return
		}
		if got == nil {
			t.Fatalf("modifyImp returned nil imp + nil err for input %q", string(extData))
		}
		// Verify storedrequest.id is set when err is nil.
		var ext map[string]json.RawMessage
		if jerr := json.Unmarshal(got.Ext, &ext); jerr != nil {
			t.Fatalf("modifyImp produced unmarshalable ext %q (input %q)", string(got.Ext), string(extData))
		}
		var prebid map[string]json.RawMessage
		if jerr := json.Unmarshal(ext["prebid"], &prebid); jerr != nil {
			t.Fatalf("modifyImp prebid not a JSON object: %q (input %q)", string(ext["prebid"]), string(extData))
		}
		var sr map[string]json.RawMessage
		if jerr := json.Unmarshal(prebid["storedrequest"], &sr); jerr != nil {
			t.Fatalf("modifyImp storedrequest not a JSON object: %q (input %q)", string(prebid["storedrequest"]), string(extData))
		}
		var id string
		if jerr := json.Unmarshal(sr["id"], &id); jerr != nil {
			t.Fatalf("modifyImp storedrequest.id not a JSON string: %q", string(sr["id"]))
		}
		// Only assert round-trip identity for valid-UTF-8 placements. Go's
		// encoding/json replaces invalid UTF-8 byte sequences with U+FFFD,
		// which is the standard library's documented behavior — not a Teal
		// adapter bug. A non-UTF-8 placement that survived Marshal will not
		// equal its source bytes after a round-trip.
		if utf8.ValidString(placement) && id != placement {
			t.Fatalf("modifyImp storedrequest.id=%q, want %q (input ext=%q)", id, placement, string(extData))
		}
	})
}

// contains is a tiny strings.Contains stand-in to avoid importing strings into
// the fuzz file (keeps the fuzz harness lean).
func contains(haystack, needle string) bool {
	if len(needle) == 0 {
		return true
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
