#!/bin/bash

set -ex

docker build -t gcr.io/newscorp-newsiq-dev/prebid-reposync:latest .
gcloud docker -- push gcr.io/newscorp-newsiq-dev/prebid-reposync:latest
