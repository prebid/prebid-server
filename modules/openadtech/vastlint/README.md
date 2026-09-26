# openadtech.vastlint

Checks video `adm` that contains a VAST document and counts revenue-impact findings on the Prebid Server Prometheus `/metrics` scrape. The check runs in-process through [vastlint-go](https://github.com/aleksUIX/vastlint-go). It does not fetch wrapper chains and it does not call the hosted validator.

The module is off until a host enables it. Counting does not remove bids. Set `reject_revenue` to drop a video bid when a revenue-impact rule fires.

Build Prebid Server with cgo. A binary built with `CGO_ENABLED=0` still compiles, and enabling the module on that binary fails at startup.

## Configuration

```yaml
hooks:
  enabled: true
  modules:
    openadtech:
      vastlint:
        enabled: true
        reject_revenue: false
  host_execution_plan:
    endpoints:
      /openrtb2/auction:
        stages:
          raw_bidder_response:
            groups:
              - timeout: 5
                hook_sequence:
                  - module_code: openadtech.vastlint
                    hook_impl_code: vastlint-raw-bidder-response
      /openrtb2/video:
        stages:
          raw_bidder_response:
            groups:
              - timeout: 5
                hook_sequence:
                  - module_code: openadtech.vastlint
                    hook_impl_code: vastlint-raw-bidder-response
```

| Field | Default | Effect |
|---|---|---|
| `enabled` | false | Host opt-in. False skips the module. |
| `reject_revenue` | false | When true, a video bid with a revenue-impact finding is removed. Other bids in the response stay. |

An account execution plan can pass `{"reject_revenue": true}` in the account module config. That value replaces the host flag for that account.

## Metrics

`vastlint_findings_total{caller,rule_id,revenue_impact}`

`caller` is the bidder name. `revenue_impact` is `true` for the twelve vastlint rules that mark lost impressions, broken measurement, or zero fill. The series is registered on the Prometheus registry only after the module is enabled. The namespace and subsystem follow `metrics.prometheus`.

Banner bids, empty `adm`, and tag URLs are skipped. A validator error leaves the bid in place and records the error on the hook outcome.

## Maintainer

[vastlint](https://github.com/aleksUIX/vastlint)
