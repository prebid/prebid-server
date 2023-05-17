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
      # Required properties
      enabled: true
      endpoint: "https://openrtb.preview.pubstack.io/v1/openrtb2"
      scopeid: "<scopeId>" # The scopeId provided by the Pubstack Support Team
      # Optional properties (advanced configuration)
      configuration_refresh_delay: "2h" # Dynamic configuration delay
      buffers: # Flush events to Pubstack when (first condition reached)
        size: "2MB" # greater than 2MB
        count : 100 # greater than 100 events
        timeout: "15m" # greater than 15 minutes
```