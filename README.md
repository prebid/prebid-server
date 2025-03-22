[![Build](https://img.shields.io/github/actions/workflow/status/prebid/prebid-server/validate.yml?branch=master&style=flat-square)](https://github.com/prebid/prebid-server/actions/workflows/validate.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/prebid/prebid-server?style=flat-square)](https://goreportcard.com/report/github.com/prebid/prebid-server)
![Go Version](https://img.shields.io/github/go-mod/go-version/prebid/prebid-server?style=flat-square)

<br />
<br />
<p align="center"><img alt="Prebid Server Logo" src="/static/pbs-logo.svg" style="width:80%; max-width:600px;"></p>
<br />

<a href="https://prebid.org/product-suite/prebid-server/">Prebid Server</a> is an open-source solution for running real-time advertising auctions in the cloud. This project is part of the <a href="https://prebid.org/">Prebid</a> ecosystem, seamlessly integrating with  <a href="https://prebid.org/product-suite/prebidjs/">Prebid.js</a> and the <a href="https://prebid.org/product-suite/prebid-mobile/">Prebid Mobile SDKs</a> to deliver world-class header bidding for any ad format and for any type of digital media.

## Getting Started
- [What is Prebid Server](https://docs.prebid.org/prebid-server/overview/prebid-server-overview.html)
- [Intro to Header Bidding](https://docs.prebid.org/overview/intro-to-header-bidding.html)
- [Header Bidding with Prebid](https://docs.prebid.org/overview/intro.html#header-bidding-with-prebid)
- [API Endpoints](https://docs.prebid.org/prebid-server/endpoints/pbs-endpoint-overview.html)
  
## Configuring

When hosting Prebid Server or developing locally, **you must set a default GDPR value**. This configuration determines whether GDPR is enabled when no regulatory signal is available in the request, where a value of `"0"` disables it by default and a value of `"1"` enables it. This is required as there is no consensus on a good default.

Refer to the [configuration guide](docs/developers/configuration.md) for additional information and a list of available configuration options.

## Hosting Prebid Server
> [!NOTE]
> Please consider [registering as a Prebid Server host](https://docs.prebid.org/prebid-server/hosting/pbs-hosting.html#optional-registration) to join the mailing list for updates and feedback.

The quickest way to host Prebid Server is to deploy our [official Docker image](https://hub.docker.com/r/prebid/prebid-server). If you're hosting the container with Kubernetes, you can configure Prebid Server with environment variables [using a pod file](https://kubernetes.io/docs/tasks/inject-data-application/define-interdependent-environment-variables/) or [using a config map](https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/#configure-all-key-value-pairs-in-a-configmap-as-container-environment-variables). Alternatively, you can use a configuration file [embedded in a config map](https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/#populate-a-volume-with-data-stored-in-a-configmap) which Prebid Server will read from at the path `/etc/config`.

For deploying a fork, you can create a custom Docker container using the command:
``` bash
docker build --platform linux/amd64 -t prebid-server .
```
or compile a standalone binary using the command:
``` bash
go build .
```
**Note:** if building from source there are a couple dependencies to be aware of: 
1. *Compile-time*. Some modules ship native code that requires `cgo` (comes with the `go` compiler) being enabled - by default it is and environment variable `CGO_ENABLED=1` do NOT set it to `0`.
2. *Compile-time*. `cgo` depends on the C-compiler, which usually is `gcc`, but can be changed by setting the value of `CC` env var, f.e. `CC=clang`.  On ubuntu `gcc` can be installed via `sudo apt-get install gcc`. 
3. *Runtime*. Some modules require `libatomic`.  On ubuntu it is installed by running `sudo apt-get install libatomic1`.  `libatomic1` is a dependency of `gcc`, so if you are building with `gcc` and running on the same machine, it is likely that `libatomic1` is already installed.   

Ensure that you deploy the `/static` directory, as Prebid Server requires those files at startup.

## Developing

Prebid Server requires [Go](https://go.dev) version 1.23 or newer. You can develop on any operating system that Go supports; however, please note that our helper scripts are written in bash.

1. Clone The Repository
``` bash
git clone git@github.com:prebid/prebid-server.git
cd prebid-server
```

3. Download Dependencies
``` bash
go mod download
```

3. Verify Tests Pass
```bash
./validate.sh
```

4. Run The Server
```bash
go run .
```

By default, Prebid Server will attach to port 8000. To confirm the server is running, visit `http://localhost:8000/` in your web browser.

### Code Style
To maintain consistency in the project's code, please:
 
- Follow the recommendations set by [Effective Go](https://go.dev/doc/effective_go). This article provides a comprehensive guide on how to write idiomatic Go code, covering topics such as naming and formatting. Many IDEs will automatically format your code upon save. If you need to manaully format your code, either run the bash script or execute the make step:
   ```
   ./scripts/format.sh -f true
   ```
   ```
   make format
   ```

- Prefer small functions with descriptive names instead of complex functions with comments. This approach helps make the code more readable, maintainable, and testable.

- Do not discard errors. You should implement appropriate error handling, such as gracefully falling back to a default behavior or bubbling up an error.
   
### IDE Recommendation

An option for developing Prebid Server in a reproducible environment isolated from your host OS is using Visual Studio Code with [Remote Container Setup](devcontainer.md). This is a recommendation, not a requirement. This approach is especially useful if you are developing on Windows, since the Remote Container runs within WSL providing you with the capability to execute bash scripts.

## Importing Prebid Server

Prebid Server is not currently intended to be imported by other projects. Go Modules is used to manage dependencies, which also makes it possible to import Prebid Server packages. This is not supported. We offer no guarantees regarding the stability of packages and do not adhere to semantic versioning guidelines.

## Contributing
> [!IMPORTANT]
> All contributions must follow the [Prebid Code of Conduct](https://prebid.org/code-of-conduct/) and the [Prebid Module Rules](https://docs.prebid.org/dev-docs/module-rules.html).

### Bid Adapter
Bid Adapters transform OpenRTB requests and responses for communicating with a bidding server. This may be as simple as a passthrough or as complex as mapping to a custom data model. We invite you to contribute an adapter for your company. Consult our guide on [building a bid adapter](https://docs.prebid.org/prebid-server/developers/add-new-bidder-go.html) for more information.

### Analytics Module
Analytics Modules enable business intelligence tools to collect data from Prebid Server to provide publishers and hosts with valuable insights into their header bidding traffic. We welcome you to contribute a module for your platform. Refer to our guide on [building an analytics module](https://docs.prebid.org/prebid-server/developers/pbs-build-an-analytics-adapter.html) for further information.

### Auction Module
Auction Modules allow hosts to extend the behavior of Prebid Server at specfic spots in the auction pipeline using existing modules or by developing custom functionality. Auction Modules may provide creative validation, traffic optimization, and real time data services among many other potential uses. We welcome vendors and community members to contribute modules that publishers and hosts may find useful. Consult our guide on [building an auction module](https://docs.prebid.org/prebid-server/developers/add-a-module.html) for more information.

### Feature
We welcome everyone to contribute to this project by implementing a specification or by proposing a new feature. Please review the [prioritized project board](https://github.com/orgs/prebid/projects/4), where you can select an issue labeled "Ready For Dev". To avoid redundant effort, kindly leave a comment on the issue stating your intention to take it on. To propose a feature, [open a new issue](https://github.com/prebid/prebid-server/issues/new/choose) with as much detail as possible for consideration by the Prebid Server Committee.

### Bug Fix
Bug reports may be submitted by [opening a new issue](https://github.com/prebid/prebid-server/issues/new/choose) and describing the error in detail with the steps to reproduce and example data. A member of the core development team will validate the bug and discuss next steps. You're encouraged to open an exploratory draft pull request to either demonstrate the bug by adding a test or offering a potential fix.
The quickest way to start developing Prebid Server in a reproducible environment isolated from your host OS
is by using Visual Studio Code with [Remote Container Setup](devcontainer.md).

## Learning Materials

To understand more about how Prebid Server in Go works and quickly spins up sample instances, refer to the `sample` folder which describes various structured and integrated examples. The examples are designed to run on any platform that supports `docker` container.
