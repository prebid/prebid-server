#!/bin/bash

# We still want to exit the pod if sync is failing, so we can get alerted
set -ex

while true; do
    ./sync.sh
    sleep 60
done
