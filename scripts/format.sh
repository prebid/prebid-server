#!/bin/bash

# Build a list of all the top-level directories in the project.
for DIRECTORY in */ ; do
  GOGLOB="$GOGLOB ${DIRECTORY%/}"
done

GOFMT_LINES=`gofmt -s -l $GOGLOB | tr '\\\\' '/' | wc -l | xargs`

if [[ $GOFMT_LINES -ne 0 ]]; then
    FMT_FILES=`gofmt -s -l $GOGLOB | tr '\\\\' '/' | xargs`
    for FILE in $FMT_FILES; do
        echo "Running: gofmt -s -w $FILE"
        `gofmt -s -w $FILE`
    done
fi