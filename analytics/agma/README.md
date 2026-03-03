# agma Analytics

In order to use the Agma Analytics Adapter, please adjust the accounts / endpoint with the data provided by agma (https://www.agma-mmc.de).

## Configuration

```yaml
analytics:
    agma:
        # Required: enable the module
        enabled: true
        # Required: set the accounts you want to track
        accounts:
        - code: "my-code" # Required: provied by agma
          publisher_id: "123" # Required: Exchange specific publisher_id, can be an empty string accounts are not used
          site_app_id: "openrtb2-site.id-or-app.id-or-app.bundle" # optional: scope to the publisher with an openrtb2 Site object id or App object id/bundle
        # Optional properties (advanced configuration)
        endpoint: 
            url: "https://go.pbs.agma-analytics.de/v1/prebid-server" # Check with agma if your site needs an extra url
            timeout: "2s"
            gzip: true
        buffers: # Flush events when (first condition reached)
            # Size of the buffer in bytes
            size: "2MB" # greater than 2MB (size using SI standard eg. "44kB", "17MB")
            count : 100 # greater than 100 events
            timeout: "15m" # greater than 15 minutes (parsed as golang duration)

```
