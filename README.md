[![Build](https://img.shields.io/github/actions/workflow/status/prebid/prebid-server/validate.yml?branch=master&style=flat-square)](https://github.com/prebid/prebid-server/actions/workflows/validate.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/prebid/prebid-server?style=flat-square)](https://goreportcard.com/report/github.com/prebid/prebid-server)
![Go Version](https://img.shields.io/github/go-mod/go-version/prebid/prebid-server?style=flat-square)

<br />
<br />
<p align="center"><img alt="Prebid Server Logo" src="/static/pbs-logo.svg" style="width:80%; max-width:600px;"></p>
<br />

<a href="https://prebid.org/product-suite/prebid-server/">Prebid Server</a> is an open-source solution for running real-time advertising auctions in the cloud. This project is part of the <a href="https://prebid.org/">Prebid.org</a> ecosystem, seamlessly integrating with  <a href="https://prebid.org/product-suite/prebidjs/">Prebid.js</a> and the <a href="https://prebid.org/product-suite/prebid-mobile/">Prebid Mobile SDKs</a> to deliver world-class header bidding for any ad format and for any type of digital media.

## Getting Started
- <a href="https://docs.prebid.org/prebid-server/overview/prebid-server-overview.html">What is Prebid Server?</a>
- <a href="https://docs.prebid.org/overview/intro-to-header-bidding.html">Intro to Header Bidding</a>
- <a href="https://docs.prebid.org/overview/intro.html#header-bidding-with-prebid">Header Bidding with Prebid</a>
- <a href="https://docs.prebid.org/prebid-server/endpoints/pbs-endpoint-overview.html">API Endpoints</a>

## Required Configuration

If you're hosting Prebid Server or developing locally, you must set a default GDPR value. This configuration determines whether GDPR is enforced when no regulatory signal is available in the request, where a value of `0` disables enforcement by default and a value of `1` enables it.

This configuration is required since there is no consensus on a good default. Please see the [configuration guide](docs/developers/configuration.md) for details on how to set the default GDPR value.


## Hosting Prebid Server
> [!NOTE]
> Please consider [registering your Prebid Server host](https://docs.prebid.org/prebid-server/hosting/pbs-hosting.html#optional-registration) to join the mailing list for updates and feedback.

The quickest way to host Prebid Server is to deploy our [official Docker image](https://hub.docker.com/r/prebid/prebid-server). If you're hosting the container within Kubernetes, you can configure Prebid Server with environment variables [using a pod file](https://kubernetes.io/docs/tasks/inject-data-application/define-interdependent-environment-variables/) or [using a config map](https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/#configure-all-key-value-pairs-in-a-configmap-as-container-environment-variables). Alternatively, you can deploy with a configuration [file using a config map](https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/#populate-a-volume-with-data-stored-in-a-configmap) which Prebid Server will load from the path `/etc/config`.

For deploying from a fork, you can either create a custom Docker container using the command `docker build -t prebid-server .` or compile a standalone binary using `go build .`. Ensure that you include the `/static` directory, as Prebid Server reads from it during startup.

## Developing Locally

Prebid Server requires [Go](https://golang.org/doc/install) version 1.19 or newer. Prebid Server can be developed on any operating system supported by Go, however our helper scripts are written for Bash.

Follow these steps to begin developing:

1. Clone the repository.
``` bash
git clone git@github.com:prebid/prebid-server.git
cd prebid-server
```

3. Download dependencies.
``` bash
go mod tidy
```

3. Verify the automated tests pass:
```bash
./validate.sh
```

4. Run the server locally:

```bash
go build .
./prebid-server
```

5. Load the landing page in your browser at `http://localhost:8000/`.

### IDE Recommendation

The quickest way to start developing Prebid Server in a reproducible environment isolated from your host OS is by using Visual Studio Code with [Remote Container Setup](devcontainer.md). This is a recommendation, not a requirement. This approach is useful especially if you are developing on Windows, since the Remote Container will run within WSL providing you with the capability to execute bash scripts.

## Importing Prebid Server

Prebid Server is not intended to be imported by other projects. Go Modules is used to manage dependencies, which also makes it possible to import Prebid Server packages. This is not supported. We offer no guarantees regarding the stability of packages and do not adhere to semantic versioning guidelines.

## Contributing
> [!IMPORTANT]
> All contributions must follow the [Prebid Code of Conduct](https://prebid.org/code-of-conduct/) and the [Prebid Module Rules](https://docs.prebid.org/dev-docs/module-rules.html).

### Bid Adapter
  allows prebid server to relay a bid request to your SSP and collect bids. you should only contribute an adapter for your own company. contributions from third parties are not permitted. follow the instructions here. click here to see a list of curently supported bidders.

### Analytics Module
 allows prebid server to collect analytics. 

### Auction Module
  extends the behavior of prebid server in many ways, such as bid filters, a/b testing, etc. follow our instructions here.

### Feature
also proposals
 all are welcome to contribute to this project. feel free to pick up an issue which is in the "ready for dev" state, before working on it, please post a comment to avoid double work. if you have a question about the specs, 

### Bug Fix
 please open an issue to detail the bug and or your feature proposal. a member of the core development team will review and discuss next steps after either verifying the bug or discussing the feature. if you want to open an exploratory PR, please mark it as a draft.

