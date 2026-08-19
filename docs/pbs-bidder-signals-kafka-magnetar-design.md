# PBS Bidder Signals Logging: Kafka to Magnetar Design

## TL;DR

We need a production logging path for sampled Prebid Server (PBS) bidder-level signals that can be joined with Impbus data and used for revenue-signal analysis.

Agreed direction:

1. Impbus decides which PSP/PBS requests are sampled and sends a flag plus join metadata to PBS.
2. PBS captures full bidder request and response bodies, plus structured lifecycle metadata, only for those sampled requests.
3. PBS publishes one Kafka event per sampled PBS auction/response, containing an array of bidder exchanges.
4. Spark/Magnetar reads Kafka, explodes the bidder array, and writes PBS sub-funnel datasets that join to the existing Impbus bidder `443` row.
5. On-prem deployments use Kafka + MirrorMaker to replicate into the Azure-side ingestion path.
6. Azure-based DCs should also explore a direct-to-Magnetar publishing path, potentially via a dedicated StatefulSet/exporter, if Magnetar supports it cleanly.
7. The existing Impbus unified logging architecture already logs the Impbus -> PBS request as bidder `443`; the new PBS data should hang under that parent row rather than duplicate it.

We are not using Courier/Packrat as the primary PBS logging path because it is tightly coupled to Impbus/RTP logging infrastructure, Packrat/an_message/protobuf framing, and C/C++ client assumptions. We are also not returning full bidder request/response bodies in the PSP response because it would bloat the hot auction response and increase timeout/network risk.

## Problem

Impbus and PSP Health need better visibility into what happened inside PBS for sampled PSP traffic:

- Which bidders were considered.
- Which bidders were filtered before adapter execution.
- Which bidder requests were built and sent.
- What full request body PBS sent to each bidder.
- What full response body each bidder returned.
- Whether bidder returned no-bid, timeout, HTTP error, malformed response, bid filtered, or accepted bid.
- How to join these signals with Impbus revenue and bid landscape data.

The core requirement for the signals project is not just compact status codes. We want full bidder request and response bodies for sampled traffic, because revenue-signal discovery may depend on details inside those payloads.

Expected sampling:

```text
sampled traffic: 5% of PSP/PBS requests
max bidders per sampled PBS request: 8
```

At high PBS QPS, this is a large data pipeline and must not run through the auction response path.

## Goals

- Capture full per-bidder request and response bodies for sampled PBS auctions.
- Capture structured bidder lifecycle metadata to avoid parsing full bodies for common analysis.
- Keep PBS auction latency safe.
- Avoid using the PSP HTTP response as the transport for large payloads.
- Produce Magnetar datasets that join cleanly to existing Impbus auction-funnel datasets.
- Preserve join keys with Impbus logs.
- Support on-prem and Azure-based PBS deployments.
- Make the pipeline replayable and observable.

## Non-goals

- Do not build a PBS-specific Courier clone.
- Do not require PBS to speak Packrat/an_message framing.
- Do not log all traffic.
- Do not block auction processing when logging fails.
- Do not put full bidder bodies into `seatnonbid` or regular PBS response extensions.

## Alternatives considered

### A. Courier / Packrat

Rejected as the primary design.

Courier/Packrat is production-proven for Impbus logs, but it is not a clean fit for PBS:

- Packrat uses RTP-specific protobuf/an_message framing.
- Canonical Packrat client appears C/C++ oriented.
- PBS is Go.
- Onboarding requires schema registration, message ids, framing, Courier config, storage mapping, close signals, and operational ownership.
- It couples PBS to Impbus/RTP-specific logging internals.
- It is overkill if the desired shape is Kafka -> Spark -> Magnetar.

### B. Return extra data in PBS response to Impbus

Useful for compact signals, but rejected for full payload logging.

This approach would work for small structured diagnostics:

```json
{
  "ext": {
    "prebid": {
      "seatnonbid": []
    },
    "xandr": {
      "pbs_signals": {}
    }
  }
}
```

But it is not suitable for full bidder request/response bodies:

- Large response bloat.
- Higher network bandwidth between PBS and Impbus.
- Higher serialization cost in the hot path.
- Higher risk of Impbus -> PBS timeout.
- Lost data if Impbus times out and never receives the PBS response.

### C. Kafka -> Spark -> Magnetar

Selected approach.

This gives a dedicated analytics transport without overloading the PSP response.

```text
PBS -> Kafka -> Spark/Magnetar -> partitioned dataset
```

## Proposed architecture

```text
Existing Impbus auction-funnel logging
  supplyrequests
    -> bidderrequests
       bidder_id = 443              // Impbus treats PBS as bidder 443
       request_id = requestId available in the PBS request
          |
          v
PBS sampled signals logging
  PBS
    -> Kafka topic, one event per sampled PBS auction
    -> Spark/Magnetar
    -> pbsrequests
    -> pbsbidderrequests
    -> pbsbidderresponses
```

The existing Impbus unified logging architecture already logs the Impbus -> PBS request as a bidder request for bidder `443`. The new PBS logs are a child/sub-funnel under that row.

Join model:

```text
pbs.requestId = bidderrequests.requestId
AND pbs.auctionId64 = bidderrequests.auctionId64
AND bidderrequests.bidderId = 443
```

This assumes the PSP path has exactly one bidder `443` invocation per Impbus auction. Use `event_hour_utc` for partition pruning, not as the identity join. Avoid joining primarily on timestamps, request bodies, or hashes.

## Request contract: Impbus to PBS

Impbus should make the sampling decision before calling PBS. PBS does not need to know the sampling rate or sampling reason; it only needs to know whether to emit the signals payload and what identifiers to stamp on it for partitioning and joins.

If the request is selected for signals logging, Impbus includes an internal extension:

```json
{
  "id": "5532570462804727104",
  "source": {
    "tid": "58d36325-9a26-430d-bfcc-733f3a03687c",
    "ext": {
      "xandr": {
        "auction_timestamp_ms": 1786605000000,
        "pbs_signals_enabled": true,
        "auction_id_64": "2812461878875191178"
      }
    }
  }
}
```

Required fields:

| Field | Purpose |
|---|---|
| `auction_timestamp_ms` | Hour partition alignment with Impbus |
| `pbs_signals_enabled` | Tells PBS to capture/publish full bidder exchanges |
| `auction_id_64` | Auction/impression id for lower-level joins |

PBS should not independently sample the same request again for this logging path. Impbus owns sample selection. PBS only honors `pbs_signals_enabled`.

## PBS logging behavior

### Event grain

PBS publishes **one Kafka event per sampled PBS auction/response**.

The event contains all bidder exchanges in a `bidders[]` array.

We intentionally do not publish one Kafka record per bidder from PBS.

Reason:

```text
90K QPS * 5% sample = 4.5K sampled PBS auction events/sec
90K QPS * 5% sample * 8 bidders = 36K per-bidder events/sec
```

One event per auction greatly reduces Kafka record count, producer overhead, and partition churn. Spark can explode the event into per-bidder rows downstream.

### What PBS captures

For each sampled auction:

- Request-level metadata.
- PBS metadata.
- One bidder exchange per bidder.

For each bidder:

- Bidder name/id.
- Imp id.
- Lifecycle status.
- Reason code.
- HTTP status.
- Timing.
- Full bidder request body.
- Full bidder response body, if present.
- Request/response byte sizes and hashes.

### PBS event schema

Use protobuf or Avro for production. JSON below is illustrative.

Kafka topic:

```text
pbs_auction_bidder_exchange_v1
```

Kafka key:

```text
<request_id>|<auction_id_64>
```

Kafka value:

```json
{
  "schema_version": 1,
  "auction_timestamp_ms": 1786605000000,
  "event_date_utc": "2026-08-13",
  "event_hour_utc": "07",

  "request_id": "5532570462804727104",
  "source_tid": "58d36325-9a26-430d-bfcc-733f3a03687c",
  "auction_id_64": "2812461878875191178",
  "impbus_bidder_id": 443,
  "pbs_request_id": "pbs-generated-request-id",

  "pbs_account": "280",
  "pbs_cluster": "prd-prebid-server-krs-aks",
  "pbs_datacenter": "zks1",
  "pbs_version": "4.6.0p",

  "pbs_request_received_ms": 1786605000000,
  "pbs_elapsed_ms": 812,

  "bidders": [
    {
      "bidder": "rubicon",
      "bidder_id": 123,
      "demand_partner_id": 456,
      "pbs_bidder_request_id": "pbs-generated-bidder-request-id",
      "imp_id": "2812461878875191178",

      "was_candidate": true,
      "was_sent_to_adapter": true,
      "response_seen": true,
      "bid_seen": true,
      "bid_accepted": false,

      "lifecycle_status": "response_received",
      "final_status": "bid_filtered",
      "stage": "bid_validation",
      "reason_code": "BELOW_FLOOR",

      "http_status": 200,
      "bidder_start_offset_ms": 43,
      "bidder_elapsed_ms": 101,
      "adapter_timeout_ms": 900,

      "bid_count": 1,
      "accepted_bid_count": 0,
      "rejected_bid_count": 1,

      "bidder_request_body": "{ complete OpenRTB request sent to bidder }",
      "bidder_response_body": "{ complete raw response returned by bidder }",

      "bidder_request_body_bytes": 4821,
      "bidder_response_body_bytes": 1332,
      "bidder_request_body_sha256": "abc123",
      "bidder_response_body_sha256": "def456"
    }
  ]
}
```

Identifier rules:

- `request_id` is already present in the OpenRTB request and should be persisted by PBS.
- `pbs_request_id` is generated by PBS and identifies one PBS auction invocation.
- `pbs_bidder_request_id` is generated by PBS and identifies one actual PBS adapter request/attempt.
- `pbs_bidder_request_id` should represent an actual adapter attempt, not merely a configured bidder in the original request.
- Bidders filtered before adapter execution still need rows, but should have `was_sent_to_adapter=false` and no `pbs_bidder_request_id` unless a request was actually built/attempted.

### Lifecycle status taxonomy

Use stable enums:

```text
CANDIDATE
FILTERED_PRE_ADAPTER
REQUEST_BUILD_ERROR
REQUEST_BUILT
REQUEST_SENT
HTTP_ERROR
TIMEOUT
NO_BID
BID_RETURNED
BID_FILTERED
BID_ACCEPTED
```

Reason codes:

```text
GEOSCOPE_FILTERED
RULES_ENGINE_FILTERED
PRIVACY_FILTERED
REQUEST_BUILD_ERROR
BIDDER_HTTP_4XX
BIDDER_HTTP_5XX
BIDDER_TIMEOUT
NO_BID
BELOW_FLOOR
CREATIVE_REJECTED
CATEGORY_MAPPING_REJECTED
DSA_REJECTED
BID_ACCEPTED
```

## Magnetar dataset model

The existing Impbus auction-funnel produces datasets such as:

```text
supplyrequests
supplyimpressions
bidderrequests
bidderimpressions
bidderbids
```

Prebid Server appears in those datasets as bidder id `443`. The new PBS data should add a child funnel under that existing row.

### `pbsrequests`

Grain: one PBS auction invocation received from Impbus.

Suggested fields:

```text
requestId
auctionId64
impbusBidderId = 443
pbsRequestId
auctionTimestampMs
eventDateUtc
eventHourUtc
pbsDatacenter
pbsCluster
pbsVersion
sellerMemberId
publisherId
samplingPct
requestBody
requestBodyEncoding
schemaVersion
```

`samplingPct` is optional dataset metadata. It does not need to be sent to PBS in the request; it can be populated downstream from Impbus sampling configuration if required for reporting or scaled estimates.

### `pbsbidderrequests`

Grain: one PBS bidder candidate or adapter request attempt.

Suggested fields:

```text
requestId
auctionId64
pbsRequestId
pbsBidderRequestId
pbsBidderCode
pbsBidderId
impId
attemptNumber
endpoint
requestStartTimestamp
requestBody
requestBodyBytes
requestBodySha256
timeoutMs
wasSentToAdapter
filteredBeforeExecution
filterReason
schemaVersion
```

For bidders filtered before adapter execution:

```text
filteredBeforeExecution = true
wasSentToAdapter = false
filterReason = GEOSCOPE_FILTERED / RULES_ENGINE_FILTERED / PRIVACY_FILTERED
requestBody = null unless PBS built one
```

### `pbsbidderresponses`

Grain: one response/outcome for one PBS bidder request/attempt.

Suggested fields:

```text
requestId
auctionId64
pbsRequestId
pbsBidderRequestId
pbsBidderCode
pbsBidderId
responseTimestamp
httpStatus
responseBody
responseBodyBytes
responseBodySha256
responseTimeMs
timeout
networkError
adapterError
noBid
bidCount
acceptedBidCount
rejectedBidCount
finalStatus
reasonCode
schemaVersion
```

Failures should be explicit rows. A timeout, HTTP error, malformed response, connection error, adapter panic, or pre-adapter filter must not disappear just because no valid bid was produced.

### Optional `pbsrawevents`

Spark may also land the original Kafka event for replay/debug:

```text
eventId
eventTimestamp
requestId
auctionId64
pbsRequestId
rawPayload
schemaVersion
producerVersion
```

## PBS implementation details

### Capture points

PBS should capture at these points:

1. Original bidder candidate list.
2. Pre-adapter filter decisions.
3. Adapter request build result.
4. Raw HTTP request body sent to bidder.
5. Raw HTTP response body received from bidder.
6. Adapter response parse result.
7. Bid validation / filtering result.
8. Final accepted/rejected status.

### Producer model

PBS must publish asynchronously.

```text
auction goroutine
  -> if pbs_signals_enabled:
       build in-memory event
       try enqueue to bounded publisher queue
  -> return auction response

publisher goroutines
  -> read queue
  -> serialize/compress
  -> publish to Kafka
```

### Failure policy in PBS

Logging must be lossy.

| Failure | Action |
|---|---|
| Queue full | Drop event, increment metric |
| Kafka unavailable | Retry briefly, then drop |
| Serialization failure | Drop event, increment metric |
| Oversized event | Split event or drop according to config |
| PBS shutdown | Flush best effort with short timeout |

PBS must never block the auction path for Kafka.

### PBS metrics

Add metrics:

```text
pbs_signals_events_created
pbs_signals_events_enqueued
pbs_signals_events_sent
pbs_signals_events_dropped_queue_full
pbs_signals_events_dropped_kafka_error
pbs_signals_events_dropped_oversized
pbs_signals_event_bytes
pbs_signals_queue_depth
pbs_signals_kafka_publish_latency_ms
```

## Kafka design

### Topic

```text
pbs_auction_bidder_exchange_v1
```

### Producer key

```text
request_id|auction_id_64
```

All bidder data for the same sampled PBS auction should be in one event. Use the OpenRTB request id plus `auction_id_64` as the Kafka key.

### Topic sizing

Expected event rate:

```text
pbs_qps = 90,000/sec
sample_rate = 5%
sampled_events = 4,500 events/sec
```

Worst-case bidder payloads per event:

```text
max_bidders = 8
```

Estimated bytes:

| Avg request+response per bidder | Approx Kafka volume |
|---:|---:|
| 5 KB | ~180 MB/sec |
| 10 KB | ~360 MB/sec |
| 20 KB | ~720 MB/sec |
| 50 KB | ~1.8 GB/sec |

This requires explicit Kafka and Magnetar capacity review.

### Oversized events

Because one auction event can contain up to 8 full bodies, configure max event size.

Policy options:

1. Preferred: split oversized event into per-bidder chunks.
2. Fallback: drop oversized event with metric.

If chunking is used:

```text
event_id = auction_id_64
chunk_index = 0..N
chunk_count = N
```

Spark must reassemble or process chunks as per-bidder rows.

## On-prem and Azure routing

### On-prem DCs

```text
PBS on-prem
  -> on-prem Kafka topic
  -> MirrorMaker
  -> Azure-side Kafka/EventHub path
  -> Spark/Magnetar
```

Required work:

- Create on-prem topic.
- Configure ACLs.
- Add MirrorMaker whitelist.
- Confirm topic replication lag.
- Monitor source and destination lag.

### Azure-based DCs

Preferred:

```text
PBS in Azure
  -> Azure-side Kafka/MT Kafka directly
  -> Spark/Magnetar
```

Also explore:

```text
PBS in Azure
  -> local StatefulSet/exporter
  -> direct Magnetar writer
```

This direct path is an investigation item, not the baseline. It may reduce Kafka/MirrorMaker hops, but it must answer:

- Can AKS workload authenticate to Magnetar?
- Can it write to the target MT/ABFS/HDFS location?
- What retry/backpressure model is acceptable?
- Does it create operational coupling between PBS and Magnetar?

## Spark / Magnetar processing

### Baseline

Use Spark on Magnetar to read the Kafka topic and write a partitioned dataset.

If Magnetar supports long-running Spark Structured Streaming for this scenario:

```text
Kafka -> Spark Structured Streaming -> raw Magnetar dataset
```

If long-running streaming is not approved:

```text
Kafka -> landing/consumer layer -> hourly Spark batch -> Magnetar dataset
```

### Processing steps

Spark job:

1. Read Kafka events.
2. Parse protobuf/Avro payload.
3. Validate required fields.
4. Explode `bidders[]`.
5. Produce normalized rows for `pbsrequests`, `pbsbidderrequests`, and `pbsbidderresponses`.
6. Write parquet/Delta/Iceberg partitioned by:

```text
event_date_utc
event_hour_utc
impbus_bidder_id=443 for pbsrequests
pbs_bidder_id for pbsbidderrequests / pbsbidderresponses
```

### Output layout

```text
pbs_bidder_exchange/
  event_date_utc=2026-08-13/
    event_hour_utc=07/
      pbs_bidder_id=123/
        part-00000.parquet
      pbs_bidder_id=456/
        part-00000.parquet
```

### Job frequency

If using Spark Structured Streaming:

```text
trigger interval: 3-5 minutes
raw writes: every trigger
compaction: hourly, with 15-30 minute delay
late repair: every 6-24 hours
```

If using hourly batch:

```text
hourly batch at HH+15 or HH+30
process previous complete hour
late repair job handles late Kafka/MirrorMaker data
```

### Raw and curated layers

Use two layers:

```text
raw/
  streaming or micro-batch output
  may have many files

curated/
  hourly compacted output
  fewer/larger files
  query target for analytics
```

## Error handling in Spark

| Failure | Action |
|---|---|
| Bad schema | Write to dead-letter dataset |
| Missing required join key | Dead-letter |
| Missing bidder id | Dead-letter or `bidder_id=unknown`, depending on analysis requirements |
| Duplicate event | Dedupe by event id + bidder id + body hashes |
| Spark job fails before checkpoint | Reprocess same Kafka offsets |
| Spark job fails after checkpoint | Resume from committed offsets |
| Magnetar write fails | Retry; if persistent, fail job and alert |
| Kafka backlog grows | Scale executors or reduce source rate |

Dead-letter layout:

```text
pbs_bidder_exchange_deadletter/
  event_date_utc=2026-08-13/
    event_hour_utc=07/
      part-00000.parquet
```

Dead-letter columns:

```text
kafka_topic
kafka_partition
kafka_offset
kafka_key
raw_payload
error_type
error_message
ingest_time
```

## Join with Impbus

Required join fields:

```text
requestId
auctionId64
pbsRequestId
pbsBidderRequestId
pbsBidderId
event_hour_utc
auction_timestamp_ms
```

Preferred join:

```sql
pbs.requestId = bidderrequests.requestId
AND
pbs.auctionId64 = bidderrequests.auctionId64
AND bidderrequests.bidderId = 443
```

Then navigate to the broader auction:

```text
bidderrequests.requestId = supplyrequests.requestId
```

Secondary joins:

```text
pbs.requestId = supplyrequests.requestId
pbs.auctionId64 = bidderimpressions.auctionId64 / impression id equivalent
pbs.pbsBidderId = PBS bidder dimension
```

Avoid using timestamps as the primary join. Use `event_hour_utc` for partition pruning.

## Capacity and rollout

Start conservative. Full body logging at 5% and 90K QPS can be TB/hour scale.

Recommended rollout:

```text
Phase 0: schema + Kafka topic + Spark skeleton with synthetic data
Phase 1: 0.1% sampled traffic in one low-volume region
Phase 2: 1% sampled traffic
Phase 3: 5% sampled traffic after capacity sign-off
```

Measure:

```text
avg event size
p95/p99 event size
Kafka MB/sec
Kafka lag
MirrorMaker lag
Spark input rows/sec
Spark processing latency
Magnetar write MB/sec
file count per partition
dead-letter count
PBS dropped events
```

## Open questions

1. Does Magnetar approve Spark Structured Streaming for this production path, or prefer hourly batch/Flink?
2. Should Azure PBS publish directly to Azure-side Kafka/MT Kafka?
3. Can Azure PBS publish directly to Magnetar through a StatefulSet/exporter?
4. What is the maximum allowed Kafka message size?
5. What retention is acceptable for full request/response bodies?
6. What access controls are required for full bidder payloads?
7. Confirm the exact documented field names for `requestId` and `auctionId64` in the flattened Impbus auction-funnel datasets.

## Recommendation

Use Kafka as the transport and Magnetar Spark as the dataset builder:

```text
PBS -> Kafka -> Spark -> Magnetar
```

Publish one event per sampled PBS auction, containing all bidder exchanges. Spark explodes to one row per bidder and writes a dataset partitioned by hour and bidder id.

For on-prem, use MirrorMaker to replicate the Kafka topic into the Azure-side path. For Azure-based DCs, publish directly to Azure-side Kafka/MT Kafka if possible, and separately investigate a direct StatefulSet-to-Magnetar writer only if the platform owners confirm it is supported and operationally safe.
