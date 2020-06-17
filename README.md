[![Build Status](https://travis-ci.org/prebid/prebid-server.svg?branch=master)](https://travis-ci.org/prebid/prebid-server)
[![Go Report Card](https://goreportcard.com/badge/github.com/prebid/prebid-server?style=flat-square)](https://goreportcard.com/report/github.com/prebid/prebid-server)

# Prebid Server

Prebid Server is an open source implementation of Server-Side Header Bidding.
It is managed by [Prebid.org](http://prebid.org/overview/what-is-prebid-org.html),
and upholds the principles from the [Prebid Code of Conduct](http://prebid.org/wrapper_code_of_conduct.html).

This project does not support the same set of Bidders as Prebid.js, although there is overlap.
The current set can be found in the [adapters](./adapters) package. If you don't see the one you want, feel free to [contribute it](docs/developers/add-new-bidder.md).

For more information, see:

- [What is Prebid?](http://prebid.org/overview/intro.html)
- [Getting started with Prebid Server](http://prebid.org/dev-docs/get-started-with-prebid-server.html)
- [Current Bidders](http://prebid.org/dev-docs/prebid-server-bidders.html)

## Installation

First install [Go](https://golang.org/doc/install) version 1.13 or newer.

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
For the full API reference, see [docs/endpoints](docs/endpoints)


## Contributing

Want to [add an adapter](docs/developers/add-new-bidder.md)? Found a bug? Great!
This project is in its infancy, and many things can be improved.


Report bugs, request features, and suggest improvements [on Github](https://github.com/prebid/prebid-server/issues).

Or better yet, [open a pull request](https://github.com/prebid/prebid-server/compare) with the changes you'd like to see.
