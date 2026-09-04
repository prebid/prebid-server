# Sample

The Sample describes several demos of quickly spinning up different Prebid Server instances with various preset configurations. These samples are intended for audiences with little knowledge about Prebid Server and plan to play around with it locally and see how it works.

# Installation

In the Sample, we use `docker` and `docker-compose` to instantiate examples; with docker providing a unified setup and interface,  you can spin up a demo server instance locally with only one command without knowing all the complexities.
The docker image used in `docker-compose.yml` is the `Dockerfile` residing in the root level of the repository. 

> [!NOTE] 
> Depending on what version of Docker is installed on your system, `docker-compose` might be considered deprecated.
>
> - Run `docker-compose` for docker compose version 1.X.
> - Run `docker compose` (without the dash) for docker compose versions 2.X and 3.X. 
>
> For more details please refer to the [History and development of Docker compose](https://docs.docker.com/compose/intro/history/). 

## Option 1 - Standard Docker Engine
Install `docker` and `docker-compose` via the [official docker page](https://docs.docker.com/compose/install/#scenario-one-install-docker-desktop). If you cannot use the official docker engine due to restrictions of its license, see the option below about using Podman instead of Docker. 

## Option 2 - Podman
From MacOS, you can use [podman](https://podman.io/) with these additional steps:

```sh
$ brew install podman docker-compose
$ podman machine init
$ podman machine set --rootful
$ podman machine start
```

# Examples

## Common File & Structures
All required files for each example are stored in a folder that follows the name pattern <number>_<name>. The `<number>` suggests its order and `<name`>` describes its title.

The following files will be present for every example and are exclusively catered to that example.
1. `app.yaml` - the prebid server app config.
2. `pbjs.html` - the HTML file with `Prebid JS` integration and communicates with the Prebid Server. It also provides a detailed explanation of the example.
3. `*.json` - additional files required to support the example. e.g. stored request and stored response.

## Common steps 

```sh
#1 - To get to the sample folder if you are on the root repository directory.
$ cd sample 

#2a - This command builds a new image, you should execute this command whenever the repository source code changes.
$ docker-compose build

#2b - Optionally you could run `docker-compose build --no-cache` if you want to build an completely new image without using cache but results in slower time to build it.
$ docker-compose build --no-cache

#3a - Spin up a corresponding sample in a container - see Steps for details 
$ docker-compose up <number>_<name>

#3b - Optionally you could use `--force-recreate` flag if you want to recreate the container every time you spin up the container.
$ docker-compose up <number>_<name> --force-recreate
```

### Detailed Steps
1. To prevent `app.yaml` from being overwritten by other config files. Ensure that `pbs.yaml` or `pbs.json` config file **MUST NOT** be present in the root directory of the repository.

2. Bring up an instance by running `docker-compose up <number>_<name>` in the `sample` folder.

3. Wait patiently until you see ` Admin server starting on: :6060` and `Main server starting on: :8000` in the command line output. This marks the Prebid Server instance finishing its initialization and is ready to serve the auction traffic.

4. you can copy the URL `http://localhost:8000/status` and paste it into your browser. You should see `ok` in the response which is another way to tell the Prebid Server that the main auction server is up and running.

5. Open a new tab in your browser and turn on the console UI. If you are using Chrome, you can right-click on the page and click `inspect`. Once the console UI is on, click on the `Network` tab to inspect the traffic later.

6. Copy the URL `http://localhost:8000/static/pbjs.html?pbjs_debug=true` into your browser. It starts the example immediately with debugging information from `Prebid JS`, and you can inspect the request and response between `Prebid JS` and `Prebid Server`.

7. After playing with the example, type `docker-compose down`. This is to shut down the existing Sample so you can start the next one you want to select.

## 002_intentiq_identity

Runs the `intentiq.identity` module, which enriches `user.eids` at the `processed_auction_request`
stage from the IntentIQ Bid Enhancement S2S API.

Before you start:

Put a real partner token in `partner_id` in `002_intentiq_identity/app.yaml`. Ask your IntentIQ
  account manager for one. Without it the resolution call fails and the hook fails open, so the
  auction still completes — just with `user.eids` untouched.

```sh
# From the repository root, work in the sample folder.
cd sample

# Start the server together with the Valkey it caches into. First run builds the image.
docker-compose up --build 002_intentiq_identity

# In a second terminal, wait until the server reports ready.
curl -s http://localhost:8000/status                       # -> ok

# Run an auction. The bid comes from stored_response.json, so no bidder account is needed.
curl -s -X POST http://localhost:8000/openrtb2/auction \
  -H 'Content-Type: application/json' \
  -d @002_intentiq_identity/request.json \
  | jq '{eids: .ext.debug.resolvedrequest.user.eids, trace: .ext.trace["iiq-identity"]}'

# Run it a second time: the trace now reports a cache hit instead of another S2S call.

# The module's own metrics, on prebid-server's metrics endpoint next to the core ones.
curl -s http://localhost:9090/metrics | grep iiq_identity_

# Stop the server and Valkey.
docker-compose down
```

`eids` is what the module resolved: `request.json` sends `"test": 1`, so the response carries
`ext.debug.resolvedrequest` — the request as bidders received it. `trace` is the module's own flow
trace, present because `trace_enabled` is on in `app.yaml`; it names the cache result, the S2S call and
what was added to `user.eids`.

The example enables the two-layer cache with Valkey as the shared L2, which is why `docker-compose up`
starts a `valkey` service and waits for it to be healthy before the server: with the cache on, a
backend that is unreachable at startup fails the module build by design rather than quietly running
uncached. `iiq_identity_cache_lookup_total` splits by `result` and `layer`, so it shows the second
request being served from L1 or L2.

See [modules/intentiq/identity/README.md](../modules/intentiq/identity/README.md) for the full
configuration reference, including the other cache backends and impression reporting.
