#!/bin/bash

# Fetches static resources (keys) that should really never change.
# But it's good to know where they come from!

set -ex

curl https://packages.cloud.google.com/apt/doc/apt-key.gpg > files/apt/apt-key.gpg
ssh-keyscan github.com > files/ssh/known_hosts
