[![Build](https://img.shields.io/github/actions/workflow/status/prebid/prebid-server/validate.yml?branch=master&style=flat-square)](https://github.com/prebid/prebid-server/actions/workflows/validate.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/prebid/prebid-server?style=flat-square)](https://goreportcard.com/report/github.com/prebid/prebid-server)
![Go Version](https://img.shields.io/github/go-mod/go-version/prebid/prebid-server?style=flat-square)

<br />
<p align="center"><img alt="Prebid Server Logo" src="/static/pbs-logo.svg" style="width:80%; max-width:600px;"></p>

<a href="https://prebid.org/product-suite/prebid-server/">Prebid Server</a> is an open-source solution for running real-time advertising auctions in the cloud. This project is part of the <a href="https://prebid.org/">Prebid.org</a> ecosystem, closely integrating with  <a href="https://prebid.org/product-suite/prebidjs/">Prebid.js</a> and the <a href="https://prebid.org/product-suite/prebid-mobile/">Prebid Mobile SDKs</a> to deliver world-class header bidding for any ad format and any type of digital media.

## Getting Started
- <a href="https://docs.prebid.org/prebid-server/overview/prebid-server-overview.html">What is Prebid Server?</a>
- <a href="https://docs.prebid.org/overview/intro-to-header-bidding.html">Intro to Header Bidding</a>
- <a href="https://docs.prebid.org/overview/intro.html#header-bidding-with-prebid">Header Bidding with Prebid</a>
- <a href="https://docs.prebid.org/prebid-server/endpoints/pbs-endpoint-overview.html">API Endpoints</a>

## Hosting Prebid Server

use our official docker image or build your own. you must specify a gdpr.default-value to `1` if you want to require by default or `0` if you wish to ignore by default. Otherwise pbs will run out of the box with as many bidders enabled as possible. 

please view our configuration guide for further setup guidance.

Please consider [registering your Prebid Server](https://docs.prebid.org/prebid-server/hosting/pbs-hosting.html#optional-registration) to get on the mailing list for updates, etc.

## Running Locally

Prebid Server requires [Go](https://golang.org/doc/install) version 1.19 or newer. Helper scripts are written for Bash, but Prebid Server can run on any operating system supported by Go.

For developing logcally, clone this repository. Prebid Server uses Go modules, so it's recommended to clone the repository outside of the GOPATH. You can then download all dependencies using:

``` bash
go mod tidy
```

Run the automated tests:

```bash
./validate.sh
```

Run the server locally:

```bash
go build .
./prebid-server
```

Load the landing page in your browser at `http://localhost:8000/`.

## Importing Prebid Server

This repository is not intended to be imported by other projects. This is not a supported way to use Prebid Server and we make no gaurantees about the stability of internal packages. Prebid Server uses Go modules to manage its depnendecies and follows the tag convention, but does not ahere to semantic versioning guidelines. 

## Contributing
> [!IMPORTANT]
> All contributions must follow the [Prebid Code of Conduct](http://prebid.org/wrapper_code_of_conduct.html)

- Contribute An Adapter
  allows prebid server to relay a bid request to your SSP and collect bids. you should only contribute an adapter for your own company. contributions from third parties are not permitted. follow the instructions here. click here to see a list of curently supported bidders.

- Contribute An Analytics Module
 allows prebid server to collect analytics. 

- Contribute A Module
  extends the behavior of prebid server in many ways, such as bid filters, a/b testing, etc. follow our instructions here.

- Implement A Feature
 all are welcome to contribute to this project. feel free to pick up an issue which is in the "ready for dev" state, before working on it, please post a comment to avoid double work. if you have a question about the specs, 

- Fix A Bug or Suggest A Feature
 please open an issue to detail the bug and or your feature proposal. a member of the core development team will review and discuss next steps after either verifying the bug or discussing the feature. if you want to open an exploratory PR, please mark it as a draft.

### IDE Recommendation

The quickest way to start developing Prebid Server in a reproducible environment isolated from your host OS is by using Visual Studio Code with [Remote Container Setup](devcontainer.md). This is a recommendation, not a requirement. This is useful especially if you are developing on Windows as the Remote Container will run within WSL giving you the ability to run the bash scripts.

