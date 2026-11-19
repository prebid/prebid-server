# VASTlint video markup validation

Validates the VAST, VMAP or DAAST XML inside video bids as they come back from each bidder, before the auction picks winners. Findings go to analytics tags and to a small summary on the bid; in reject mode, bids that trip host-chosen rules are dropped; in async mode the hook returns at once and validation runs off the bid path. Everything runs inside the Prebid Server process: no external service, no network call.

## Maintainer

Contact: dev@vastlint.org (shared maintainers' address). Repository and issue tracker: https://github.com/aleksUIX/vastlint. Engine binding: https://github.com/aleksUIX/vastlint-go.

## Description

The rules are the [vastlint](https://vastlint.org) catalog (IAB VAST 2.0 through 4.3 plus the 4.4 working draft, VMAP 1.0, DAAST 1.0, SIMID, Open Measurement and the CTV Addendum 2024; 246 rules with stable ids, [reference](https://vastlint.org/docs/rules), [methodology](https://vastlint.org/docs/methodology/)). The engine is reached through the vastlint-go binding, two ways:

- Default build: the engine compiled to WebAssembly and executed by [wazero](https://github.com/tetratelabs/wazero), a pure Go runtime. Nothing to install; `CGO_ENABLED=0` works. The 650 KB module is embedded in vastlint-go, pinned by SHA-256, compiled at startup (about 0.1 s) and run in wazero's sandbox: it imports no host functions, so it has no file, network or clock access. The compiler backend needs amd64 or arm64; on other architectures the module refuses to start with a clear error and the native build below is the option.
- `go build -tags vastlint_cgo`: the native library `libvastlint`, found with `pkg-config vastlint`, installed from the [vastlint releases](https://github.com/aleksUIX/vastlint/releases) in the build stage. Same findings, native speed. Native calls run on a fixed set of pinned threads so the library's allocator state stays bounded. This build is exercised in the vastlint-go repository, not in Prebid Server's CI.

Both engines produce identical findings on the same fixtures; vastlint-go's test suite checks that on every fixture. The module is inert until a host enables it: Prebid Server only calls the builder for enabled modules, so hosts that do not enable it pay binary size and nothing else.

## Prerequisites

None for the default build. For the native build, `libvastlint` and `vastlint.h` on the build host, reachable through `pkg-config`.

## Configuration

```yaml
hooks:
  enabled: true
  modules:
    vastlint:
      validation:
        enabled: true
        mode: async                  # monitor | reject | async
        reject_on: []                # rule ids that drop a bid in reject mode; empty drops nothing
        rules:                       # optional per-rule levels: off, info, warning, error
          - VAST-2.0-adsystem-no-version=off
        max_document_bytes: 262144   # larger creatives are skipped and kept
        sample_percent: 100          # monitor and async modes; reject mode checks every bid
        annotate_bids: true          # write bid.ext.vastlint (monitor and reject modes)
        annotate_rules: true         # include rule ids in that summary
        cache_size: 33554432         # verdict cache in bytes; 32 MiB holds about 100k verdicts
        cache_ttl_seconds: 600
        async_queue: 4096            # async mode: creatives waiting for off-path validation
        async_queue_bytes: 67108864  # async mode: markup held by that queue, 64 MiB
        pool_size: 0                 # engine instances; 0 = GOMAXPROCS
        report_interval_seconds: 0   # module counters to the server log; 0 (default) is off
  host_execution_plan:
    endpoints:
      /openrtb2/auction:
        stages:
          raw_bidder_response:
            groups:
              - timeout: 20
                hook_sequence:
                  - module_code: vastlint.validation
                    hook_impl_code: vastlint-raw-bidder-response
```

The same in JSON (`pbs.json`):

```json
{
  "hooks": {
    "enabled": true,
    "modules": {
      "vastlint": {
        "validation": {
          "enabled": true,
          "mode": "async",
          "cache_size": 33554432,
          "rules": {"VAST-2.0-adsystem-no-version": "off"}
        }
      }
    },
    "host_execution_plan": {
      "endpoints": {
        "/openrtb2/auction": {
          "stages": {
            "raw_bidder_response": {
              "groups": [
                {
                  "timeout": 20,
                  "hook_sequence": [
                    {"module_code": "vastlint.validation", "hook_impl_code": "vastlint-raw-bidder-response"}
                  ]
                }
              ]
            }
          }
        }
      }
    }
  }
}
```

`rules` takes a list of `id=level` strings in YAML and environment configuration (those loaders lowercase keys and split them on dots, which would mangle an object keyed by rule id) and either a list or an object in JSON and account configuration. Rule ids are matched against the engine catalog case-insensitively. Unknown rule ids in `rules` or `reject_on`, and rule levels other than `error`, `warning`, `info` or `off`, fail startup.

Give the module its own group in the execution plan, after any module that also replaces the bid list (`prebid.ortb2blocking` does): hooks in one group run concurrently on the same payload and the later change set replaces the earlier one's bids.

### Account-level configuration

An account may override `mode`, `reject_on`, `rules`, `sample_percent`, `annotate_bids` and `annotate_rules`:

```json
{"hooks": {"modules": {"vastlint": {"validation": {"mode": "reject", "reject_on": ["VAST-2.0-parse-error", "VAST-2.0-inline-impression"]}}}}}
```

Account `rules` are merged over the host's per rule id; the other fields replace the host value. An account override that is invalid, or names a rule the engine does not have, is reported as a hook warning and the host config applies to that request.

## Modes

- `async` (recommended): the hook queues each eligible creative and returns at once; workers validate after the auction has moved on. Nothing on the bid changes (except that a `vastlint` key a bidder put in `ext` is removed, in every mode) and the hook never waits for the engine. Verdicts reach the analytics tags of the next sighting of the same creative through the cache, and the module's counters. The off-path workers use at most half of the engine pool (at least one, so with `pool_size: 1` the single instance is shared), and a job that waits more than 100 ms for an engine slot is counted under `timeouts` and dropped, so a backlog never pins the pool. `async_queue_bytes` must be at least `max_document_bytes`.
- `monitor`: validate on the bid path, annotate each checked bid with `bid.ext.vastlint`, report through analytics tags. Nothing is removed.
- `reject`: as monitor, and a bid whose findings include any rule in `reject_on` is removed from the bidder response. Only `reject_on` drops bids; an invalid document with no listed rule is kept and annotated. The module never rejects the whole response and never sets `Reject` on the hook result.

## Analytics tags

Each hook result carries one activity named `vast_validation` with one result per video bid, so analytics adapters can report per bidder and per rule:

| Field | Values |
|---|---|
| `status` (activity) | `success`; `error` when no eligible bid could be validated |
| `status` (result) | `success-modify` when the bid was annotated, `success-allow` when it was checked, queued or skipped, `success-block` when it was dropped, `error` when the engine failed on it |
| `values.action` | `checked`, `dropped`, `skipped_url`, `skipped_size`, `skipped_not_xml`, `skipped_not_utf8`, `sampled_out`, `engine_error`, `queued`, `queue_full` |
| `values.valid` | true when the creative has no error-level finding (checked and dropped bids) |
| `values.input_status` | `supported`, `unsupported` or `malformed` |
| `values.document_type`, `values.document_version` | `VAST`, `VMAP` or `DAAST` and the declared version, so a playlist in a video bid is not mistaken for an ad |
| `values.errors`, `values.warnings`, `values.infos` | finding counts by severity |
| `values.rules` | the distinct rule ids that fired |
| `values.cache` | `hit` or `miss`, when the cache is on |
| `appliedto` | `bidder`, `bidids`, `impids` (omitted when the bid has no imp id) |

## Bid extension

Kept bids in monitor and reject modes carry the same summary on the bid when `annotate_bids` is true:

```json
"ext": {"vastlint": {"valid": false, "input_status": "supported", "document_type": "VAST", "document_version": "4.2", "errors": 1, "warnings": 3, "infos": 0, "rules": ["VAST-2.0-inline-impression"], "engine_version": "0.16.2"}}
```

The key is set in place with `sjson`, so the bidder's other `ext` bytes are untouched; a bid whose `ext` is not a JSON object is left alone and noted in the debug messages. `engine_version` is the engine whose catalog produced the verdict. A `vastlint` key that a bidder put in `ext` itself is removed on every path, so a verdict in the response can only come from this module.

## Metrics and logs

The framework's module metrics (`modules_vastlint_validation_{called,duration,success_noops,success_updates,success_rejects,failed,execution_errors,timeouts}`, under the host's configured namespace) need no extra wiring. The module logs one line at startup with the engine kind, version and effective configuration. With `report_interval_seconds` set, it also logs, at that interval and only when the figures changed, the off-path totals (`vastlint async: submitted= validated= invalid= errors= timeouts= queue_full= queue_depth= queue_bytes= mean_us=`) and the cache hit rate (`vastlint cache: hits= misses= hit_rate_pct= entries=`); the same figures can be derived from the analytics tags, which is why it is off by default. Dropped bids appear in the debug trace when the request runs with debug enabled.

## Verdict cache

Production creatives repeat heavily, so the verdict cache serves most bids without engine time. The key is 128 bits of SHA-256 over the creative, the effective rule overrides and the engine version, so a different rule set or engine build never sees another's verdict; the value is the summary only, never markup. Cached verdicts behave exactly like fresh ones: they annotate bids and, in reject mode, drop them. About 300 bytes per entry, allocated up front by freecache (the same shape as the scope3 module's segment cache).

## Privacy

The module reads only the creative markup of video bids (`bid.adm`) and the bidder code. It does not read the request, user, device or geo objects, carries no identifiers, makes no network calls, and stores nothing but verdict summaries keyed by a hash of the markup. Markup is never logged. The activity controls for user data do not apply.

## Behavior under trouble

- Only bids with `BidType` video or `bid.mtype` 2, non-empty UTF-8 markup that starts with `<` after leading whitespace and a byte order mark are trimmed, and size within `max_document_bytes`, reach the engine. URL markup (any scheme case), oversized, non-XML and non-UTF-8 markup are counted and kept.
- The cost of one creative grows with its size and element count: about 0.1 ms per KB of dense markup on the wasm engine, so a 256 KiB creative built to be expensive can take 20 ms on its own. `max_document_bytes` and the group timeout are the host's bounds; a hook that overruns the group timeout is discarded whole by the framework (annotations, drops and analytics tags for that bidder response included), and the module stops validating remaining bids once the deadline has passed.
- Bids of one response are validated concurrently, up to eight at a time (or the engine pool size when smaller), and reassembled in the original order.
- An engine failure, a context deadline or a panic inside the engine binding is contained: the bid is kept, its result is tagged `engine_error`, and one hook warning per response reports the count. A hook group timeout leaves the whole bidder response untouched, as the framework defines; reject mode is therefore best effort under timeout.
- The async queue is bounded by count and by bytes; when full, the creative is counted as `queue_full` and skipped, so a burst cannot grow memory without limit.
- Memory: `pool_size` wasm instances (about 1.25 MiB each at start, recycled above 16 MiB), plus `cache_size`, plus up to `async_queue_bytes` of held markup in async mode.
- At process shutdown the module stops its workers, waits for the validation each has in flight, and closes the engine (`modules.Shutdowner`). A hook racing the shutdown gets `queue_full` or `engine_error`, never a panic.

## Development

```sh
go test ./modules/vastlint/validation/...                                   # wasm engine
PKG_CONFIG_PATH=/path/to/pkgconfig go test -tags vastlint_cgo ./modules/vastlint/validation/...  # native engine
go test -bench . -benchmem ./modules/vastlint/validation/                   # hook cost, cached and uncached
```

Most unit tests inject a fake engine; the builder, end-to-end, edge-case and `TestRace` tests run the embedded engine against the fixtures in `testdata/`. `integration_test.go` drives the module through Prebid Server's module builder, hook executor and configuration loader, including the YAML and JSON blocks of this README.
