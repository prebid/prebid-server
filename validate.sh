#!/bin/bash

set -e

die() { echo "$@" 1>&2 ; exit 1;  }

# check there are no formatting issues
GOFMT_LINES=`gofmt -l *.go pbs adapters | wc -l | xargs`
test $GOFMT_LINES -eq 0 || die "gofmt needs to be run, ${GOFMT_LINES} files have issues"

go test $(go list ./... | grep -v /vendor/)
