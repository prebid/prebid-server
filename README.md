[![Build Status](https://travis-ci.org/prebid/prebid-server.svg?branch=master)](https://travis-ci.org/prebid/prebid-server)

# Prebid Server

Prebid Server is an open source implementation of Server-Side Header Bidding.
It is managed by [Prebid.org](http://prebid.org/overview/what-is-prebid-org.html),
and upholds the principles from the [Prebid Code of Conduct](http://prebid.org/wrapper_code_of_conduct.html).

For more information, see:

- [A Beginner's Guide to Header Bidding](http://adprofs.co/beginners-guide-to-header-bidding/)
- [Server-side Header Bidding Explained](http://www.adopsinsider.com/header-bidding/server-side-header-bidding/)

If you're familiar with [Prebid.JS](https://github.com/prebid/Prebid.js), see the [getting started guide](http://prebid.org/dev-docs/get-started-with-prebid-server.html).

## Installation

First install [Go 1.9.1](https://golang.org/doc/install) and [Glide](https://github.com/Masterminds/glide#install).

Then download and prepare Prebid Server:

```bash
cd $GOPATH
git clone https://github.com/prebid/prebid-server src/github.com/prebid/prebid-server
cd src/github.com/prebid/prebid-server
glide install
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

Want to add an adapter? Found a bug? Great! This project is in its infancy, and many things
can be improved.

Report bugs, request features, and suggest improvements [on Github](https://github.com/prebid/prebid-server/issues).

Or better yet, [open a pull request](https://github.com/prebid/prebid-server/compare) with the changes you'd like to see.
