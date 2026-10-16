# Hosts can monitor sampled VAST from final auction responses.

The optional `vastlint` analytics adapter exports bounded, copied inline video
markup to a host-controlled external validator. Collection uses the existing
analytics dispatch and activity controls. Validation runs in one background
worker. The callback performs bounded collection work and never waits for the
validator. Host-owned rule settings and Prometheus findings support aggregate
quality monitoring alongside PBS collection-health metrics.

Configuration, monitoring counters, resource limits, privacy permissions, and
verification commands are documented in the
[adapter README](../../analytics/vastlint/README.md).
The coverage includes returned inline `adm` at `/openrtb2/auction`, after
exitpoint hooks. Other replacement body types are skipped. Creatives
removed by caching and content behind URLs are outside that coverage.
