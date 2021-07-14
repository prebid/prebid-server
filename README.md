[![Build](https://img.shields.io/github/workflow/status/prebid/prebid-server/Validate/master?style=flat-square)](https://github.com/prebid/prebid-server/actions/workflows/validate.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/prebid/prebid-server?style=flat-square)](https://goreportcard.com/report/github.com/prebid/prebid-server)
![Go Version](https://img.shields.io/github/go-mod/go-version/prebid/prebid-server?style=flat-square)

# Prebid Server

Prebid Server is an open source implementation of Server-Side Header Bidding.
It is managed by [Prebid.org](http://prebid.org/overview/what-is-prebid-org.html),
and upholds the principles from the [Prebid Code of Conduct](http://prebid.org/wrapper_code_of_conduct.html).

This project does not support the same set of Bidders as Prebid.js, although there is overlap.
The current set can be found in the [adapters](./adapters) package. If you don't see the one you want, feel free to [contribute it](https://docs.prebid.org/prebid-server/developers/add-new-bidder-go.html).

For more information, see:

- [What is Prebid?](https://prebid.org/overview/intro.html)
- [Prebid Server Overview](https://docs.prebid.org/prebid-server/overview/prebid-server-overview.html)
- [Current Bidders](http://prebid.org/dev-docs/pbs-bidders.html)

Please consider [registering your Prebid Server](https://docs.prebid.org/prebid-server/hosting/pbs-hosting.html#optional-registration) to get on the mailing list for updates, etc.

## Installation

First install [Go](https://golang.org/doc/install) version 1.15 or newer.

Note that prebid-server is using [Go modules](https://blog.golang.org/using-go-modules).
We officially support the most recent two major versions of the Go runtime. However, if you'd like to use a version <1.13 and are inside GOPATH `GO111MODULE` needs to be set to `GO111MODULE=on`.

Download and prepare Prebid Server:

```bash
cd YOUR_DIRECTORY
git clone https://github.com/prebid/prebid-server src/github.com/prebid/prebid-server
cd src/github.com/prebid/prebid-server
```

Run the automated tests:

```bash
./validate.sh
```

Or just run the server locally:

```bash
go build .
./prebid-server
```

Load the landing page in your browser at `http://localhost:8000/`.
For the full API reference, see [the endpoint documentation](https://docs.prebid.org/prebid-server/endpoints/pbs-endpoint-overview.html)

## Go Modules

The packages within this repository are intended to be used as part of the Prebid Server compiled binary. If you
choose to import Prebid Server packages in other projects, please understand we make no promises on the stability
of exported types.

## Contributing

Want to [add an adapter](https://docs.prebid.org/prebid-server/developers/add-new-bidder-go.html)? Found a bug? Great!

Report bugs, request features, and suggest improvements [on Github](https://github.com/prebid/prebid-server/issues).

Or better yet, [open a pull request](https://github.com/prebid/prebid-server/compare) with the changes you'd like to see.

## IDE Recommendations

The quickest way to start developing Prebid Server in a reproducible environment isolated from your host OS
is by using Visual Studio Code with [Remote Container Setup](devcontainer.md).
