#!/bin/bash

set -e

AUTOFMT=true
COVERAGE=false
VET=true

while true; do
  case "$1" in
     --nofmt ) AUTOFMT=false; shift ;;
     --cov ) COVERAGE=true; shift ;;
     --novet ) VET=false; shift ;;
     * ) break ;;
  esac
done

die() { echo -e "$@" 1>&2 ; exit 1;  }

# Build a list of all the top-level directories in the project.
GOGLOB="*.go"
for DIRECTORY in */ ; do
  GOGLOB="$GOGLOB ${DIRECTORY%/}"
done
GOGLOB="${GOGLOB/ docs/}"
GOGLOB="${GOGLOB/ vendor/}"

# Check that there are no formatting issues
GOFMT_LINES=`gofmt -l $GOGLOB | wc -l | xargs`
if $AUTOFMT; then
  # if there are files with formatting issues, they will be automatically corrected using the gofmt -w <file> command
  if [[ $GOFMT_LINES -ne 0 ]]; then
    FMT_FILES=`gofmt -l $GOGLOB | xargs`
    for FILE in $FMT_FILES; do
        echo "Running: gofmt -w $FILE"
        `gofmt -w $FILE`
    done
  fi
else
  test $GOFMT_LINES -eq 0 || die "gofmt needs to be run, ${GOFMT_LINES} files have issues.  Below is a list of files to review:\n`gofmt -l *.go pbs adapters`"
fi

if $COVERAGE; then
  ./scripts/check_coverage.sh
else
  go test $(go list ./... | grep -v /vendor/)
fi

if $VET; then
  for SOURCE in $GOGLOB ; do
    # default call for wildcards and directories
    COMMAND="go tool vet -source $SOURCE"
    if [ -f $SOURCE ]; then
      # file
      COMMAND="go vet -source $SOURCE"
    fi
    echo "Running: $COMMAND"
    `$COMMAND`
  done
fi
