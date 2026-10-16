# Hosts can monitor sampled VAST quality without waiting for validation.

The optional `vastlint` analytics adapter copies at most one inline video `adm`
from a sampled `/openrtb2/auction` response. A bounded worker sends it to a
host-controlled VASTlint validator. Hosts aggregate collection health through
PBS metrics and effective rule findings through the validator's Prometheus
endpoint. Auctions never wait for validation, metrics delivery, application
retries, or validator availability. Bids, prices, markup, and auction outcomes
are unchanged.

Collection still consumes synchronous CPU: activity checks, account lookup,
sampling, bounded privacy checks and candidate scanning, admission, and one
bounded string copy. Deferred analytics runs before the HTTP handler returns.
PBS request-duration metrics alone do not measure this complete cost. Use
client-side latency measurements when evaluating a deployment.

Maintainer: Aleks Sekowski, https://github.com/aleksUIX.
Shared maintenance email: dev@vastlint.org.

## Enable monitoring for authorized accounts.

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
  },
  "metrics": {
    "prometheus": {"port": 9091, "namespace": "prebid", "subsystem": "server"}
  }
}
```

Disabled by default. Only host configuration chooses the endpoint, accounts,
limits, and sample rate. Standard `PBS_ANALYTICS_VASTLINT_*` environment variables
override file settings. `PBS_ANALYTICS_VASTLINT_ACCOUNTS="account-one account-two"`
uses space-separated IDs. No public VASTlint endpoint is selected and no samples
are contributed to VASTLint for product improvement.

TLS uses system roots, hostname verification, and TLS 1.2 or later. Place the
Rust service behind a host TLS terminator for remote connections. Use
`allow_insecure: true` only for a trusted loopback or equivalently isolated link.
Do not expose its unauthenticated gRPC or metrics ports to the public internet.

## Raw creative collection has a deliberately narrow privacy policy.

All of `reportAnalytics`, `transmitUfpd`, `transmitPreciseGeo`,
`transmitUniqueRequestIds`, and `transmitTid` must permit the `analytics`
component named `vastlint`. Every activity denial prevents dispatch.

The initial request policy additionally requires explicit `regs.gdpr=0`, or
its legacy `regs.ext.gdpr=0` equivalent without contradictory flags. A non-serialized
conflict flag survives OpenRTB normalization and analytics request copies. It suppresses
COPPA, nonzero DNT/LMT, any supplied GPP, GPP sections, US Privacy, GPC, consent,
or either additional-consent representation (`ConsentedProvidersSettings` and
`consented_providers_settings`). Missing GDPR applicability, invalid JSON, and
privacy extensions above 4 KiB also suppress collection. This does not interpret
consent strings or support GDPR-applicable traffic, even with a consent string.
A request with a large user extension is conservatively excluded.

An account allowlist and activity permission are not complete consent enforcement.
The host must authorize the affected accounts and this raw-creative data path.
Request scrubbing cannot remove identifiers embedded in creative URLs. XML goes
only to the configured validator, where validation is ephemeral. The integration
does not fetch URLs, fire tracking beacons, persist XML, or log markup. Collection
suppressions are counted, so coverage gaps remain visible.

## Sampling covers returned inline markup within fixed scan limits.

Only successfully encoded and written typed final responses are eligible.
Collection follows exitpoint replacement, including pointer and value responses.
Custom replacement body types are excluded. The earlier response is never a
fallback. Legacy analytics consumers retain their existing `Response` field.

Video is `mtype=2`, or `ext.prebid.type=video` when `mtype` is absent. A sampled
auction scans at most 64 seats and 64 bids total, including empty seats and
non-video bids. Bid-extension decoding is limited to 4 KiB. Reservoir sampling
chooses uniformly among eligible candidates inside this bounded set. Truncation,
privacy exclusions, auction-level sampling, and resource drops mean this is not
an unbiased sample of every creative or impression in production.

Examples of coverage:

* Returned inline VAST, including malformed XML: eligible within limits.
* PBS-generated VAST wrapper: eligible, but its downstream content is not fetched.
* Cached-away `adm`, including `returnCreative:false`: counted as `missing_markup`
  when a typed video bid remains. Missing markup alone cannot prove caching.
* URL-only `adm`: `url_markup`. NURLs and cache entries are not fetched.
* Oversized `adm`: `oversized_markup`, checked before inspecting content.
* Banner, native, or audio bid: `non_video`.
* Custom exitpoint body or failed encoding/write: `response_excluded`.

AMP, `/openrtb2/video`, pre-cache markup, and downstream wrapper chains are
outside this version's scope. Findings do not prove playback failure or an
original bidder defect.

## Work remains bounded under failure and overload.

One worker processes documents sequentially. The outstanding cap includes
copying, queueing, and in-flight RPCs. Capacity and event/byte admission precede
XML allocation. Default owned XML is at most 2 MiB per adapter instance. This is
not a total RSS limit: transport buffers and runtime allocations add memory.

Full capacity or rate exhaustion drops monitoring. Queued documents expire.
RPC deadlines also respect remaining document age. There is no disk spool,
application retry, or per-document goroutine. The gRPC library owns connection
maintenance and may transparently retry calls known not to have been processed.
The HTTP handler never waits for any of that work.

Replies are capped at 64 KiB. The collector requires compact acknowledgement,
input classification, a bounded profile identifier, policy/catalog digests,
engine identity, verdict, and summary. Old or incompatible services produce an
`incompatible_response` count. Worker-call panics are contained and counted.
Shutdown stops admission, cancels RPCs, closes the connection, discards queued
work, and waits at most the configured timeout plus 100 ms.

## Rule policy lives on the validator.

Set `VASTLINT_RULES_FILE` to a host-owned JSON file:

```json
{
  "profile": "publisher-video",
  "rules": {
    "VAST-2.0-inline-impression": "error",
    "VAST-2.0-url-cdata": "info",
    "VAST-2.0-mediafile-https": "off"
  }
}
```

Accepted levels: `off`, `info`, `warning`, `error`. Omitted rules use catalog
defaults. Remove an override and restart to restore its default. Configuration
is immutable during a process lifetime. Invalid IDs, levels, profile names, or
unknown top-level settings fail startup. Parser resource ceilings always apply;
the parser-error rule must remain an error. Configured host policy rejects
per-request rule/version overrides. PBS sends neither.

`vastlint_grpc_policy_info` exposes profile, policy digest, engine version, and
catalog digest. Severity labels reflect effective overrides. Policy changes
require a restart; compare this identity before interpreting a change in traffic.
Use scrape labels for environment and cluster attribution. No creative ID,
request ID, account ID, or URL is a metric label.

## Collection health and creative quality have different denominators.

PBS exports `prebid_server_vastlint_collection_total{event="..."}` with the
example namespace/subsystem above. The worker flushes atomic counter deltas on a one-second ticker between RPCs
and on shutdown through PBS's existing metrics engine. A running RPC can delay
flushing up to its deadline. Abrupt process
termination can lose the final unflushed interval. Influx uses the existing
`vastlint.<event>` counters. `Stats()` remains available for local diagnostics.

Auction counters are mutually staged: `activity_denied` counts denied auction
dispatches; `response_excluded` counts other unsuccessful or untyped responses;
`account_excluded` counts missing/unlisted accounts; `auctions` counts the remaining
authorized successful responses before sampling. `sample_excluded + sampled`
partitions `auctions`. `privacy_suppressed` is a subset of `sampled`.

Bid counters apply only to privacy-eligible sampled auctions: `non_video`,
`missing_markup`, `url_markup`, `oversized_markup`, and `oversized_extension`
count excluded scanned bids. `candidates` counts eligible scanned bids.
`scan_limited` and `no_candidate` count auctions, not bids; they can overlap.
Each auction with a candidate attempts to admit exactly one document.

Document counters: `capacity_dropped`, `rate_dropped`, `expired`, and
`shutdown_dropped` describe collection loss. `submitted` counts attempted RPCs;
`checked` counts compatible replies. `failed` aggregates `timeout`,
`transport_failed`, `incompatible_response`, and `worker_panicked`, plus canceled
in-flight calls at shutdown. `shutdown_dropped` includes canceled in-flight work,
so it overlaps `failed` in that case. Do not sum overlapping counters into a rate.

`unsupported` is a subset of `checked`. `with_errors`, `with_warnings`, and
`with_infos` count checked, non-unsupported documents with at least one finding
at each severity; these overlap. `with_findings` is their union, not an error or
playback-failure rate. A denominator for PBS error findings is
`checked - unsupported`. Pending work and time-window boundaries can temporarily
separate submission and completion totals.

On the validator, `vastlint_grpc_documents_total{caller="prebid-server",outcome}`
partitions completed validations into `malformed`, `unsupported`, `error`,
`warning`, `info`, or `clean`, in that precedence order. Supported documents use
the highest effective severity. Exclude `unsupported` from quality denominators.
`vastlint_grpc_rule_documents_total` counts each affected document once per rule
and severity. `vastlint_grpc_findings_total` counts individual occurrences.
Unsupported inputs do not contribute to these rule-quality counters.

Use the [validator deployment examples](https://github.com/aleksUIX/vastlint/tree/grpc-v0.15.0-rc.1/deploy/prebid)
for the pinned release candidate, Prometheus scraping, alerts, a Grafana dashboard,
health probes, resource limits, and rollback.
Metrics describe the received sample, not population prevalence. Collector
coverage and losses must remain visible alongside quality charts.

## Reproduce verification and keep the wire contract synchronized.

```sh
go test -race ./analytics/vastlint ./analytics/build ./metrics/...
go test ./endpoints/openrtb2 ./openrtb_ext ./router
go test -run '^$' -bench BenchmarkVastlintFullDispatch -benchmem ./analytics/build
VASTLINT_TEST_ENDPOINT=127.0.0.1:50051 VASTLINT_TEST_METRICS_URL=http://127.0.0.1:9090/metrics \
  go test -tags vastlint_integration -run TestRust ./analytics/vastlint
```

The VASTLint repository's `scripts/prebid-compatibility.py` runs the actual Go
client against five host profiles, verifies public protobuf field compatibility,
checks metrics, and covers malformed/unsupported inputs, compact and oversized
replies, cancellation, and recovery. It accepts a local binary or digest-pinned
image. The tagged `vastlint_operational` endpoint harness measures the complete
HTTP path in fresh processes; it is a synthetic load experiment, not a production
capacity claim.

Generate the small client using `protoc-gen-go v1.34.1` and
`protoc-gen-go-grpc v1.4.0`:

```sh
protoc -I api --go_out=api --go_opt=paths=source_relative \
  --go-grpc_out=api --go-grpc_opt=paths=source_relative api/vastlint.proto
```

The public service path and field numbers are preserved. Fix, ListRules, streaming,
and detailed finding fields are intentionally absent from this monitoring client.
The gRPC dependencies already exist in PBS; no dependency version is added.
Maintainer agreement is still needed on gRPC communication, typed `FinalResponse`,
and the optional creative-data permission interface. The PR remains draft while
those design questions are open.
