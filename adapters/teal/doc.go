// Package teal implements the Teal openrtb2 bidder for prebid-server.
//
// Teal exchanges openrtb2 bid requests through a single passthrough endpoint:
// https://a.bids.ws/openrtb2/auction. The adapter ports prebid-server-java's
// org.prebid.server.bidder.teal.TealBidder semantically 1:1 — every behavioral
// branch the Java unit tests exercise has a matching Go unit test in
// teal_test.go.
//
// # Per-imp params
//
// Each impression carries Teal-specific params under imp.ext.bidder:
//
//	{
//	  "account":   "publisher-account-id",  // required, non-blank
//	  "placement": "placement-name"          // optional, non-blank when present
//	}
//
// Account is required and non-blank (org.apache.commons.lang3.StringUtils.isBlank
// semantics — empty / whitespace-only fail validation). Placement is optional;
// when present it must also be non-blank. Validation errors mirror Java
// PreBidException messages verbatim:
//
//   - "account parameter failed validation"
//   - "placement parameter failed validation"
//   - "Error parsing imp.ext for impression {impID}"
//
// # Three novel request mutations
//
// MakeRequests applies three mutations that prepare the request body for
// Teal's exchange. They are not standard in the prebid-server-go adapter
// corpus, so each is documented inline below; teal_test.go also pins each
// mutation with a dedicated test case.
//
// M1 — Per-imp imp.ext.prebid.storedrequest.id injection. When placement is
// non-nil and non-blank, the adapter ensures imp.ext.prebid is an object,
// ensures imp.ext.prebid.storedrequest is an object, and writes
// imp.ext.prebid.storedrequest.id = placement. Existing siblings under
// imp.ext, imp.ext.prebid, and imp.ext.prebid.storedrequest are preserved.
// If imp.ext.prebid (or storedrequest) exists but is not a JSON object it is
// replaced with a fresh object — mirrors Java's getOrCreate(parent, field)
// helper which uses putObject when the existing child is not an ObjectNode.
//
// M2 — First-imp-account-wins propagation to Site.Publisher.ID and
// App.Publisher.ID. The first imp whose ExtImpTeal validates contributes its
// account value to BOTH Site.Publisher.ID (when site is non-nil) AND
// App.Publisher.ID (when app is non-nil). Subsequent imps' accounts are
// ignored — matches Java's `account = account == null ? ext.getAccount() :
// account`. If publisher is missing on either side, a fresh Publisher with
// the account ID is created.
//
// M3 — Request.Ext.bids stamp. The adapter unconditionally adds a top-level
// "bids" property to request.ext: {"bids":{"pbs":1}}. Existing top-level
// fields under request.ext are preserved; an existing "bids" key is
// overwritten. JSON literal `null` and absent request.ext are both treated
// as empty (mirrors Java's ObjectUtils.defaultIfNull pattern).
//
// # Mediatype resolution
//
// getBidType walks the request's imps to find the matching ImpID, then
// returns the first mediatype it sees in this priority order:
//
//	banner > video > audio > native
//
// Default is banner — used when no imp matches the bid's ImpID, or when the
// matching imp has no mediatype declared. Note this priority is INVERTED from
// the prebid-server-go default (which favors video first); the Java side
// drives the order so we mirror it for fidelity.
//
// # Validation flow
//
// MakeRequests collects per-imp errors but does NOT fail the whole batch on
// single-imp failure. The flow is:
//
//  1. For each imp: parse ext.bidder → validate → mutate → append to
//     surviving imps. On any per-imp error, append to errs and continue.
//  2. If zero imps survived, return (nil, errs) — no HTTP request issued.
//  3. Otherwise apply M1/M2/M3, marshal, return one RequestData + collected
//     errors.
//
// All per-imp validation errors are wrapped in errortypes.BadInput so the
// prebid-server core categorizes them as input issues rather than server
// faults.
//
// # Bid response handling
//
// MakeBids uses the canonical adapters helpers:
//
//   - status 204 → IsResponseStatusCodeNoContent → returns (nil, nil)
//   - status 4xx/5xx → CheckResponseStatusCodeForErrors → returns one error
//   - status 200 → jsonutil.Unmarshal into BidResponse → emit TypedBid per bid
//
// Currency comes from BidResponse.Cur and is set on the BidderResponse once
// (not per-bid).
//
// # Cross-language fidelity surface
//
// Every assertion in TealBidderTest.java has a matching Go test in
// teal_test.go. Behavioral parity is also verified by the JSON-driven
// adapterstest framework against fixtures under tealtest/exemplary/ and
// tealtest/supplemental/.
//
// Four intentional or stdlib-driven cross-language differences are documented:
//
//   - Java uses `placement != null` checks against a String field. Go uses
//     `*string` (Placement *string in ExtImpTeal) so it can distinguish
//     absent (nil) from present-empty (non-nil, "" or whitespace).
//   - Java's URL validation goes through Apache Commons; Go uses
//     net/url.ParseRequestURI which has slightly different lenience
//     boundaries on edge cases like trailing dots in hostnames. Both reject
//     "invalid_url" and require an absolute URL with scheme + host.
//   - Whitespace classification: Go's unicode.IsSpace returns true for
//     U+00A0 (non-breaking space), U+2007 (figure space), U+202F (narrow
//     no-break space) and treats them as blank; Java's Character.isWhitespace
//     and Apache Commons StringUtils.isBlank return false for these. Inputs
//     with NBSP-only account or placement therefore pass Java validation but
//     fail Go validation. This is a strict-MORE-than-Java behavior — there
//     is no input Java rejects that Go accepts.
//   - JSON object key ordering: Go's encoding/json sorts map keys
//     alphabetically when marshaling. Java's Jackson ObjectNode preserves
//     insertion order. M1 and M3 emit objects via map[string]json.RawMessage,
//     so byte-level wire output differs from Java even when the logical
//     object is identical. Receivers compare logically (e.g., adapterstest
//     uses jsondiff structural comparison), so this divergence is invisible
//     in practice.
//
// # Performance characteristics
//
// On Apple M1 Max with the realistic single-imp banner fixture:
//
//   - MakeRequests: ~8.8μs / 11.3KB / 151 allocs per call
//   - MakeBids:     ~675ns / 840B / 14 allocs per call
//   - getBidType:   ~27ns / 0 allocs (zero-allocation hot path)
//
// MakeRequests' allocation profile is dominated by the marshal/unmarshal
// round-trips through json.RawMessage maps required for M1 (per-imp
// storedrequest injection) and M3 (request.ext bids stamp).
//
// # Fuzzing
//
// teal_fuzz_test.go ships three fuzz harnesses that have collectively
// executed 1M+ exploratory inputs without surfacing new panic classes:
//
//   - FuzzParseImpExt: imp.ext bytes → never-panic + on-error message contract
//   - FuzzMergeBidsPBSFlag: request.ext bytes → success path always yields a
//     marshalable map containing "bids":{"pbs":1}
//   - FuzzModifyImp: imp.ext bytes → round-trip identity for valid placements
//
// To run: `go test -fuzz=FuzzMergeBidsPBSFlag -fuzztime=30s`.
package teal
