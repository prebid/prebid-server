#!/usr/bin/env bash

set -eux

# "$@" refers to the command-line argument(s) given when this script is called.
# The intended usage of these cli args in our entrypoint scripts is to either call a function
# defined in this script (e.g. `deploy/docker-entrypoint.sh prepare_and_run bash`), or to simply
# pass through the argument supplied. (e.g. `deploy/docker-entrypoint.sh bash`)
ACTION="$@"

# BEGIN FUNCTIONS

# Call the functions that do the necessary work to run your application in a given container,
# and then execute one of the following:
## - CLI arguments passed to the invocation of the `prepare_and_run` function
## - if no CLI arguments provided, run whatever has been defined as the default run target
function prepare_and_run() {
  echo "Container setup finished, running '$@'" &&\
  exec bash -c "$*"
}

# END FUNCTIONS

# This is the actual "entry point", by which we mean the invocation that will directly result in PID 1 on the container.
## We will run whatever was passed to the container run command.
## In local development, commands will usually start w/ calls to functions in this script (e.g. `prepare_and_run rake console`)
[[ "${INSPECT_PODS:-}" ]] && exec tail -f /dev/null || ${ACTION}
