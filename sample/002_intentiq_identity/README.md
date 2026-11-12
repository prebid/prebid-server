# IntentIQ identity module: local test

Runs Prebid Server natively with the `intentiq.identity` module, a Redis cache,
a stored bid response, and `mock_iiq.py` standing in for the IntentIQ S2S and
reporting endpoints.

## Run

From the repository root, in separate terminals:

```sh
docker run --rm --name iiq-local-redis -p 6379:6379 redis:7-alpine
python3 sample/002_intentiq_identity/mock_iiq.py 9999
```

PBS reads `pbs.yaml` from its working directory and `static/` relative to it,
so run it from a directory that links both:

```sh
mkdir -p /tmp/pbs-iiq && cd /tmp/pbs-iiq
ln -sfn "$OLDPWD/static" static && ln -sfn "$OLDPWD/sample" sample
cp sample/002_intentiq_identity/app.yaml pbs.yaml
(cd "$OLDPWD" && go build -o /tmp/pbs-iiq/pbs-local .) && ./pbs-local -logtostderr
```

## Check

```sh
curl -s localhost:8000/openrtb2/auction -H 'Content-Type: application/json' \
  -d @sample/002_intentiq_identity/request.json | jq '.ext.trace["iiq-identity"], .ext.debug.resolvedrequest.user.eids'
curl -s localhost:9190/metrics | grep '^iiq_identity_'
docker exec iiq-local-redis redis-cli --scan
```

- `user.eids` keeps `pubcid.org` and gains `intentiq.com`.
- The first request resolves through the mock (`[resolve]` in its output); later
  ones come from `cache l1`, or `cache l2` after a PBS restart.
- Every bid produces a `[report]` line in the mock output.
- Redis keys expire after the mock's `cttl` (600 s).

## Real IntentIQ endpoints

Replace `partner_id`, `api_endpoint`, and `reports_endpoint` in `pbs.yaml` with
the values from the module README. Set `metrics.prometheus.port: 0` to check
the module still starts with Prometheus disabled.
