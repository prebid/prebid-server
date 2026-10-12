# VASTlint monitors sampled inline VAST in final auction responses.

The `vastlint` analytics adapter copies one bounded video `adm` from a sampled
`/openrtb2/auction` response and sends it to a host-controlled VASTlint gRPC
service in the background. It does not wait for validation in the analytics
callback. It does not modify bids or markup. Collection and copying still use
PBS CPU and memory.

The adapter is disabled by default. A host must configure the validator and an
explicit account allowlist. Raw creative export requires `reportAnalytics`,
`transmitUfpd`, `transmitPreciseGeo`, `transmitUniqueRequestIds`, and `transmitTid`
permission for the `analytics` component named `vastlint`. Denial of any of
these activities prevents dispatch. Request scrubbing alone cannot remove
identifiers embedded in creative URLs. Hosts must authorize this data path and
operate the endpoint accordingly.

Maintainer: Aleks Sekowski, https://github.com/aleksUIX.
Shared maintenance email: dev@vastlint.org.

## Configure the validator and authorized accounts.

```json
{
  "analytics": {
    "vastlint": {
      "enabled": true,
      "endpoint": "validator.example.internal:443",
      "accounts": ["authorized-publisher-account"],
      "sample_percent": 1,
      "max_document_bytes": 65536,
      "queue_size": 32,
      "timeout_ms": 250,
      "max_age_ms": 1000,
      "max_events_per_second": 100,
      "max_bytes_per_second": 1048576
    }
  }
}
```

TLS uses system roots and the configured hostname. A trusted local validator can
use `allow_insecure: true`. The Rust gRPC server needs a host TLS terminator for
the TLS configuration above. No public endpoint is selected by default. Only
the host can change endpoints or resource limits.

Every setting also supports the standard `PBS_ANALYTICS_VASTLINT_*` environment
variables. For example, set `PBS_ANALYTICS_VASTLINT_ENDPOINT=localhost:50051` and
`PBS_ANALYTICS_VASTLINT_ACCOUNTS="account-one account-two"`. Account IDs in the
environment value are separated by spaces. Environment values override file
settings, including resource limits.

Zero resource settings inherit the defaults shown above. Accepted limits are
1 to 100 percent sampling, 1 to 131072 bytes per document, 1 to 128 outstanding
documents, up to 8 MiB of outstanding copied XML, 1 to 5000 ms RPC timeout,
1 to 10000 ms document age, 1 to 10000 events per second, and up to 16 MiB per
second of XML admission. Byte rate must admit at least one maximum-size
document. At most 256 account IDs of 1 to 128 bytes are accepted.

Token buckets bound document and byte admission. Each starts with one second of
capacity and refills at the configured rate, so short bursts are possible.
Sampling is per authorized successful auction, independent of the egress limits.
One document is selected at most. Selection favors earlier seats and bids and
must not be treated as an unbiased sample of all returned creatives.

## The scope is returned inline markup.

Video is identified by OpenRTB `mtype=2`, or `ext.prebid.type=video` when `mtype`
is absent. The callback scans at most 64 seats and 64 bids total. Empty seats
and non-video bids consume scan capacity. Extension decoding is capped at 4 KiB.
The document length is checked before examining its contents. Empty and URL-only
markup are counted and skipped. Other supplied video markup is submitted
verbatim, including malformed XML. No XML parsing runs in PBS.

The collector uses the typed OpenRTB response returned after exitpoint hooks.
An exitpoint replacement with a different body type is skipped. Replaced or
removed markup from the earlier auction response is never used as a fallback.
Existing analytics consumers keep their original response field. PBS-generated
wrappers can be observed. Markup removed by caching, including `returnCreative:
false`, is unavailable and counted as missing when the typed video bid remains.
The adapter does not fetch VAST URLs, NURLs, cache entries, or downstream wrappers.
AMP, the video endpoint, and other analytics event types are not monitored in
this version. Findings do not prove an original bidder defect or playback failure.

## Resource limits include work that is still running.

One worker sends documents sequentially. The outstanding slot cap includes
copying, queued, and in-flight documents. Capacity and rate admission happen
before cloning XML. The worker retains capacity until the actual RPC returns.
A full queue drops monitoring work. Stale documents are discarded. RPC deadlines
also respect the remaining document age. Service failures and malformed replies
are counted without altering auction results.

With the default configuration, outstanding owned XML is bounded by 2 MiB per
adapter instance. This is not a total RSS limit. Metadata, channel storage,
gRPC buffers, runtime allocations, and multiple PBS instances add memory.
The response decoder is capped at 64 KiB. The client requests `summary_only`
and requires the server's acknowledgement. Compact replies contain validity,
summary counts, and provenance. Detailed findings remain in server monitoring
and are omitted from the wire reply. Omitting fields only from the Go schema
would not reduce reply size. A server without compact-reply support is counted
as failed. Only the unary Validate wire contract is included. No native library
or new Go dependency version is added.

There are no application retries, disk spooling, or per-document goroutines.
gRPC manages its connection goroutines and may transparently retry calls known
not to have been processed. Shutdown stops admission, cancels validation, closes
the connection, and discards queued work. It waits up to the configured RPC
timeout plus 100 ms. Unexpected worker-call panics are contained and counted.

## Monitor findings on the validator and collection counters on PBS.

The validator exposes findings and verdicts on its existing Prometheus endpoint,
labelled with `caller="prebid-server"`. Scrape
`vastlint_grpc_findings_total` and `vastlint_grpc_verdicts_total` there. The service
also exposes `vastlint_grpc_active_workers` when built with the safety fixes.
Use separate validator instances or host monitoring configuration when multiple
PBS clusters need separate attribution.

PBS logs cumulative collection counters once per minute and on shutdown. The
counters distinguish authorized auctions, sampled auctions, scan truncation,
non-video bids, missing/URL/oversized markup, oversized extensions, capacity and
rate drops, expiration, submission attempts, checked responses, documents with
findings, failures, and shutdown drops. Enqueued documents are not reported as
checked. A checked response requires compact-mode acknowledgement, a verdict,
summary, engine identity, and catalog digest. The API also exposes
`Module.Stats()` for host diagnostics.

No XML, document identifiers, finding messages, or validator error strings are
included in these logs. Raw XML is transmitted to the configured validator.
Verdicts completed after the response are not appended to auction analytics
objects or hook tags.

Use a validator release containing strict XML parsing, parser resource ceilings,
hard CPU-worker permits held until actual work finishes after cancellation,
and acknowledged compact replies. The compatible source includes the
[validator changes](https://github.com/aleksUIX/vastlint/commit/46f86bfbc8a7ef6eb0904226d128c6659d9a70aa).
The local compatibility run uses an explicitly recorded source build. It does
not establish that older published images contain those fixes. A fixed release
and verified image digest are required before a host trial.

## Reproduce verification and code generation.

```sh
go test -race -cover ./analytics/vastlint ./analytics/build ./config
go test -bench BenchmarkFinalResponseCapture -benchmem ./analytics/vastlint
```

Against a fixed local Rust validator, run:

```sh
VASTLINT_TEST_ENDPOINT=127.0.0.1:50051 go test -tags vastlint_integration \
  -run TestRust -v ./analytics/vastlint
```

The collector tests cover unchanged markup, copied ownership, scan and byte
bounds, rate and capacity limits, cancellation, expiration, malformed responses,
panic containment, outage handling, and concurrent shutdown. Real Rust tests
also cover finding-heavy replies and the maximum admitted document size. Endpoint tests cover typed and custom exitpoint replacements.
Configuration tests load and override all settings through environment variables.
Analytics dispatch tests cover each required activity and preserve existing
adapters' request scrubbing behavior.

Generate the minimal client with `protoc`, `protoc-gen-go v1.34.1`, and
`protoc-gen-go-grpc v1.4.0` from this directory:

```sh
protoc -I api --go_out=api --go_opt=paths=source_relative \
  --go-grpc_out=api --go-grpc_opt=paths=source_relative api/vastlint.proto
```

The subset preserves the public service path and field numbers. Compatibility
must be tested against the actual Rust server, because local Go fakes do not
prove interoperability. It intentionally omits Fix, ListRules, streaming, and
finding-message fields.

This proposal uses gRPC dependencies already present in PBS. Maintainer review
should confirm that this host-controlled communication path is acceptable under
the Prebid module communication rules.
