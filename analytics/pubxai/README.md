# PubxAI Analytics

In order to use the pubxai analytics module, it needs to be configured by the host.

You can configure the server using the following environment variables:

```bash
export PBS_ANALYTICS_PUBXAI_ENABLED="true"
export PBS_ANALYTICS_PUBXAI_ENDPOINT="https://analytics.pbxai.com"
export PBS_ANALYTICS_PUBXAI_PUBLISHERID=<your pubxid here> # should be an UUIDv4
export PBS_ANALYTICS_PUBXAI_BUFFER_INTERVAL="5m"
export PBS_ANALYTICS_PUBXAI_BUFFER_SIZE="10KB"
export PBS_ANALYTICS_PUBXAI_SAMPLING_PERCENTAGE="100"
export PBS_ANALYTICS_PUBXAI_CONFIGURATION_REFRESH_INTERVAL="5h"
```

Or using the pbs configuration file and by appending the following block:

```yaml
analytics:
  pubxai:
    enabled: true
    publisherid: "your pubxid here" # should be an UUIDv4
    endpoint: "https://analytics.pbxai.com"
    buffer_interval: 5m
    buffer_size: 10kb
    sampling_percentage: 100
    configuration_refresh_interval: 5h
```