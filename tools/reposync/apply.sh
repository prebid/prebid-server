#!/bin/bash

set -ex

#kubectl create secret generic reposync -n reposync --dry-run -oyaml --from-file=kubernetes/secrets/reposync/ | kubectl apply -f -

kubectl apply -f kubernetes/deployment.yaml
