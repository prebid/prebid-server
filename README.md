# [![Prebid Server Logo](https://github.com/prebid/prebid-server/blob/master/static/pbs-logo.svg?raw=true)](https://prebid.org/product-suite/prebid-server/)

[![Build](https://img.shields.io/github/workflow/status/prebid/prebid-server/Validate/master?style=flat-square)](https://github.com/prebid/prebid-server/actions/workflows/validate.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/prebid/prebid-server?style=flat-square)](https://goreportcard.com/report/github.com/prebid/prebid-server)
![Go Version](https://img.shields.io/github/go-mod/go-version/prebid/prebid-server?style=flat-square)

Prebid Server is an open-source solution for server-to-server header bidding managed by [Prebid](https://prebid.org). Prebid Server supports a variety of use cases including Mobile Web, AMP, Server Side Wed with Prebid.js and Prebid Mobile SDKs, and Long Form Video/CTV.
Utilizing Prebid Server can reduce latency between bid request and ad selection, and speed the presentation of your site and ads.

## Documentation
Please explore both our [Marketing Website](https://prebid.org/) and [Technical Docs](https://prebid.org/) website. We are fully open source and you can contribute here and here.

Highlights:
- [Prebid & Header Bidding Overview](https://docs.prebid.org/overview/intro.html)
- [Prebid Server Overview](https://docs.prebid.org/prebid-server/overview/prebid-server-overview.html)
- [Prebid Server API Reference](https://docs.prebid.org/prebid-server/overview/prebid-server-overview.html)
- [Bidders](http://prebid.org/dev-docs/pbs-bidders.html)

## Host

Please consider [registering your Prebid Server](https://docs.prebid.org/prebid-server/hosting/pbs-hosting.html#optional-registration) to get on the mailing list for updates, etc.

use our official docker image or build your own. you must specify a gdpr.default-value to `1` if you want to require by default or `0` if you wish to ignore by default. Otherwise pbs will run out of the box with as many bidders enabled as possible. 

please view our configuration guide for further setup guidance.

## Develop

### VS Code

The quickest way to start developing Prebid Server in a reproducible environment isolated from your host OS is by using Visual Studio Code with [Remote Container Setup](devcontainer.md).

## Contribute
> All contributions must follow the [Prebid Code of Conduct](http://prebid.org/wrapper_code_of_conduct.html).


This project does not support the same set of Bidders as Prebid.js, although there is overlap.
The current set can be found in the [adapters](./adapters) package. If you don't see the one you want, feel free to [contribute it](https://docs.prebid.org/prebid-server/developers/add-new-bidder-go.html).





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

