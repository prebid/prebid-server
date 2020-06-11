# Pubstack Analytics

In order to use the pubstack analytics module, it needs to be configured by the host.

You can configure the server using the following environment variables:

```bash
export PBS_ANALYTICS_PUBSTACK_ENABLED="true"
export PBS_ANALYTICS_PUBSTACK_ENDPOINT="https://openrtb.preview.pubstack.io/v1/openrtb2"
export PBS_ANALYTICS_PUBSTACK_SCOPEID=<your scopeId here> # should be an UUIDv4
```

Or using the pbs configuration file and by appending the following block:

```yaml
analytics:
    pubstack:
      enabled: true
      endpoint: "https://openrtb.preview.pubstack.io/v1/openrtb2"
      scopeid: <your scopeId here> # The scopeId provided by the Pubstack Support Team
```