# Pubstack Analytics

In order to use the pubstack analytics module. One should configure it first.
Configuration of the module is made in the same fashion as other prebid server configuration.

You can either configure the server using the following environment variables:

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
      scopeid: <your scopeId here> # should be an UUIDv4
```