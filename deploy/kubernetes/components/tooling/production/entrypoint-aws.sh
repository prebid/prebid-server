#!/usr/bin/env bash
#
# This entrypoint is intended to setup the execution environment based on infrastructure-specific things that don't
# make sense to hardcode, e.g. AWS region.

################################################################################
# ENVIRONMENT
################################################################################

set -eu

################################################################################
# FUNCTIONS
################################################################################

function get_aws_metadata_by_path() {
  curl -s http://169.254.169.254/latest/meta-data/${1}
}

function nr_host() {
  # e.g. `web-5678cb4f4b-zcml8`
  echo ${MY_NODE_NAME} | awk -F. '{print $1}'
}

################################################################################
# PIPELINES
################################################################################

# For cloud libraries/tooling
export AWS_AVAILABILITY_ZONE=$(get_aws_metadata_by_path placement/availability-zone)
export AWS_DEFAULT_REGION=$(echo ${AWS_AVAILABILITY_ZONE} | sed 's/.$//')

# For the app
export AVAILABILITY_ZONE=${AWS_AVAILABILITY_ZONE}
export NEW_RELIC_PROCESS_HOST_DISPLAY_NAME="$(echo ${AWS_AVAILABILITY_ZONE}: $(nr_host))"

# If the method of supplying args changes, the need for quotes may change.
exec bash -l -c "$*"
