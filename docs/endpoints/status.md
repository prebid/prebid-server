## `GET /status`

This endpoint will return a 2xx response whenever Prebid Server is ready to serve requests.
Its exact response can be [configured](../developers/configuration.md) with the `status_response`
config option. For example, in `pbs.yaml`:

```yaml
status_response: "ok"
```
