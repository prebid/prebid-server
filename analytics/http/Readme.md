# Http Analytics

This module sends selected analytics events to a http endpoint.

Please make sure you take a look at the possible configuration and the filter options

## Configuration

```yaml
analytics:
    http:
        # Required: enable the module
        enabled: true
        endpoint: 
            # Required: url where the endpoint post data to
            url: "https://my-rest-endpoint.com"
            # Required: timeout for the request
            timeout: "2s"
            # Optional: enables gzip compression for the payload
            gzip: false
            # Optional: additional headers send in every request
            additional_headers:
                X-My-header: "some-thing"
        buffer: # Flush events when (first condition reached)
            # Size of the buffer in bytes
            size: "2MB" # greater than 2MB
            count : 100 # greater than 100 events
            timeout: "15m" # greater than 15 minutes
        auction: 
            enabled: false # enable auction tracking
            sample_rate: 1 # sample rate 0-1.0 to sample the event
            filter: "RequestWrapper.BidRequest.App.ID == '123'" # Optional filter
        video:
            sample_rate: 1 # Sample rate, f 0-1 set sample rate, 1 is 100%
            filter: "" 
        amp:
            sample_rate: 0.5 # 50% of the events are sampled
            filter: "" 
        setuid:
            sample_rate: 0.25 # 25% of the events are sampled
            filter: "" 
        cookie_sync:
            sample_rate: 0 # events are not sampled
            filter: "" 
        notification:
            sample_rate: 1
            filter: "" 

```

### Sample Rate

The sample rate has to be between `0.0` (never sample) and `1.0` (always sample). The sample rate is always evaluated and defaults to 0

### Filter

The module uses [github.com/antonmedv/expr](github.com/antonmedv/expr) for complexer filter options. The analytics object is always passed into the expression.

#### Samples:

- Auction: `RequestWrapper.BidRequest.App.ID == '123'`
- Video: `VideoRequest.Video.MinDuration > 200`
