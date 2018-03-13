[![Build Status](https://travis-ci.org/prebid/prebid-server.svg?branch=master)](https://travis-ci.org/prebid/prebid-server)

# Prebid Server

Prebid Server is an open source implementation of Server-Side Header Bidding.
It is managed by [Prebid.org](http://prebid.org/overview/what-is-prebid-org.html),
and upholds the principles from the [Prebid Code of Conduct](http://prebid.org/wrapper_code_of_conduct.html).

For more information, see:

- [What is Prebid?](http://prebid.org/overview/intro.html)
- [Getting started with Prebid Server](http://prebid.org/dev-docs/get-started-with-prebid-server.html)

## Installation

First install [Go 1.9.1](https://golang.org/doc/install) or later and [dep](https://golang.github.io/dep/docs/installation.html). Note that dep requires an explicit GOPATH to be set.

```bash
export GOPATH=$(go env GOPATH)
mkdir -p $GOPATH
```

Then download and prepare Prebid Server:

```bash
cd $GOPATH
git clone https://github.com/prebid/prebid-server src/github.com/prebid/prebid-server
cd src/github.com/prebid/prebid-server
dep ensure
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
