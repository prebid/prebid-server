[![Build Status](https://travis-ci.org/prebid/prebid-server.svg?branch=master)](https://travis-ci.org/prebid/prebid-server)

# prebid-server
Server side component to offload prebid processing to the cloud

# Discussion group for development / adapter testing
http://redditadops.slack.com channel headerbidding-dev

# Current Status of Adapters (Sep 20, 2017)
Working on live sites:
- AppNexus Web
- Audience Network (Facebook) Web
- Rubicon Web
- Index Exchange Web

In testing:
- Pubmatic web

Under development/ testing
- PulsePoint Web

# How it works
The client (typically prebid.js) sends a JSON request to Prebid Server at `/auction`. See static/pbs_request.json for the format.
Prebid Server forms OpenRTB requests, sends them to the appropriate adapters, concatenates the responses, and returns them
to the client.

A few key points:
 * No ranking or decisioning is performed by Prebid Server. It just proxies requests.
 * No ad quality management (malware, viruses, deceptive creatives) is performed by Prebid Server
 * Prebid Server does no fraud scanning and does nothing to prevent bad traffic.

# User synching
Prebid Server provides a `/setuid` endpoint that allows adapters to push in their user IDs. These are stored in a cookie named,
creatively, `uids`. To see stored cookies, call `/getuids`. To set an optout cookie, call `/optout`. When an adapter doesn't
have a synched cookie, a `no_cookie` response is returned with a usersync URL that the client should call via asynchronous pixel
or equivalent. If Prebid Server doesn't have a cookie set, a preemptive `no_cookie` response is returned to allow the client
to ask for user consent and drop a cookie.

# Logging
Prebid Server does no server-side logging. It can stream metrics to an InfluxDB endpoint, which are aggregated as a time series.
Prebid Server has no user profiling or user-data collection capabilities.

# Usage
## Without Docker
### Prerequisites
* [Go](https://www.golang.org)
* [Glide](https://glide.sh/)

# Hosted version
AppNexus is hosting a version (generally bleeding-edge of this repo including some in-flight pull requests) at https://prebid.adnxs.com.

### Getting
1. Install glide: https://github.com/Masterminds/glide#install
2. `cd $GOPATH`
3. `git clone https://github.com/prebid/prebid-server src/github.com/prebid/prebid-server`
4. `cd src/github.com/prebid/prebid-server`
5. `glide install`
6. `./validate.sh`

### Running
To compile a binary and run locally:
```
$ make build
$ ./prebid-server -v 1 -logtostderr
```

## With Docker
### Prerequisites
* [Docker](https://www.docker.com)

### Compiling an alpine binary
The Dockerfile for prebid-server copies the binary in the root directory to the
docker container, and must be specifically be compiled for the target
architecture (alpine).

```
$ docker run --rm -v "$PWD":/go/src/github.com/prebid/prebid-server \
-w /go/src/github.com/prebid/prebid-server \
billyteves/alpine-golang-glide:1.2.0 \
/bin/bash -c 'glide install; go build -v'
```

The above command will run a container with the necessary dependencies (alpine,
go 1.8, glide) and compile an alpine compatible binary.

### Build prebid-server docker container
```
# make image
```

### Run container
This command will run a prebid-server container in interactive mode and map the
`8000` port to your machine's `8000` port so that you can visit `http://localhost:8000`
and see prebid-server's index page.

```
$ docker run --rm -it -p 8000:8000 prebid-server
```

# Data integration
Prebid Server has three primary data objects that it needs to manage:
 * Accounts represent publishers, and are used for metrics aggregation and terms of service adherence. Requests without an
 active account will be rejected.
 * Domains are compared to the HTTP Referer header; all unknown/unapproved domains will be rejected.
 * Bundles are compared to the `app.bundle` value; all unknown/unapproved bundles will be rejected.
 * Configs are used for server-side configuration of adapters, primarily for use with mobile apps where managing configs
 client-side is ineffective.

# Up Next
 * Limit adapters to one bid per ad unit
 * NURL support
 * Think through how deals work
 * Video
 * Native
