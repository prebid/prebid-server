[![Build](https://img.shields.io/github/actions/workflow/status/prebid/prebid-server/validate.yml?branch=master&style=flat-square)](https://github.com/prebid/prebid-server/actions/workflows/validate.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/prebid/prebid-server?style=flat-square)](https://goreportcard.com/report/github.com/prebid/prebid-server)
![Go Version](https://img.shields.io/github/go-mod/go-version/prebid/prebid-server?style=flat-square)

<br />

<h1>
    <div align="center">
        <img alt="Prebid Server Logo" src="https://github.com/prebid/prebid-server/blob/master/static/pbs-logo.svg?raw=true" style="width:80%; max-width:600px">
    </div>
</h1>

<a href="https://prebid.org/product-suite/prebid-server/">Prebid Server</a> is an open-source solution for running real-time advertising auctions in the cloud. This project is part of the <a href="https://prebid.org/">Prebid</a> ecosystem, closely integrating with  <a href="https://prebid.org/product-suite/prebidjs/">Prebid.js</a> and the <a href="https://prebid.org/product-suite/prebid-mobile/">Prebid Mobile SDKs</a> to deliver world-class header bidding for any ad format and any type of digital media.

## Getting Started
- <a href="https://deploy-preview-4516--prebid-docs-preview.netlify.app/prebid-server/overview/prebid-server-overview.html">What is Prebid Server?</a>
- <a href="https://docs.prebid.org/overview/intro-to-header-bidding.html">Intro to Header Bidding</a>
- <a href="https://docs.prebid.org/overview/intro.html#header-bidding-with-prebid">Header Bidding with Prebid</a>
- <a href="https://docs.prebid.org/prebid-server/endpoints/pbs-endpoint-overview.html">API Documentation</a>

## Hosting Prebid Server

Please consider [registering your Prebid Server](https://docs.prebid.org/prebid-server/hosting/pbs-hosting.html#optional-registration) to get on the mailing list for updates, etc.

use our official docker image or build your own. you must specify a gdpr.default-value to `1` if you want to require by default or `0` if you wish to ignore by default. Otherwise pbs will run out of the box with as many bidders enabled as possible. 

please view our configuration guide for further setup guidance.

## Running Locally

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

## Use As a Library

The packages within this repository are intended to be used as part of the Prebid Server compiled binary. If you
choose to import Prebid Server packages in other projects, please understand we make no promises on the stability
of exported types.


## Contributing
> All contributions must follow the [Prebid Code of Conduct](http://prebid.org/wrapper_code_of_conduct.html).


This project does not support the same set of Bidders as Prebid.js, although there is overlap.
The current set can be found in the [adapters](./adapters) package. If you don't see the one you want, feel free to [contribute it](https://docs.prebid.org/prebid-server/developers/add-new-bidder-go.html).

Want to [add an adapter](https://docs.prebid.org/prebid-server/developers/add-new-bidder-go.html)? Found a bug? Great!

Report bugs, request features, and suggest improvements [on Github](https://github.com/prebid/prebid-server/issues).

Or better yet, [open a pull request](https://github.com/prebid/prebid-server/compare) with the changes you'd like to see.









### VS Code

The quickest way to start developing Prebid Server in a reproducible environment isolated from your host OS is by using Visual Studio Code with [Remote Container Setup](devcontainer.md).







## Resources