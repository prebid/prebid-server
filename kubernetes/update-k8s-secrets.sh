#!/bin/bash

# This script can be used to manually update the k8s secrets when needed.

set -e
cd "$( dirname "${BASH_SOURCE[0]}" )"

echo "Select Environment"
options=("dev" "prod")
select opt in "${options[@]}"
do
  tput sgr0
  case $opt in
    "dev")
      ENV='dev'
      gcloud config set project newscorp-newsiq-dev
      if [[ ! -e .secrets-repo-dev ]]; then
        git clone keybase://team/prebid.dev/secrets .secrets-repo-dev
      else
        pushd .secrets-repo-dev
        git pull
        popd
      fi
      break
      ;;
    "prod")
      ENV='prod'
      gcloud config set project newscorp-newsiq
      if [[ ! -e .secrets-repo-prod ]]; then
        git clone keybase://team/prebid.prod/secrets .secrets-repo-prod
      else
        pushd .secrets-repo-prod
        git pull
        popd
      fi
      break
      ;;
    *) echo invalid option;;
  esac
done

gcloud config set container/cluster  kubernetes-prebid-cloudops
gcloud container clusters get-credentials kubernetes-prebid-cloudops --zone us-east1-c

echo "Update prebid-environment..."

kubectl --namespace=prebid create secret generic prebid-environment \
    --from-file=environment=.secrets-repo-${ENV}/others/environment \
    --dry-run -o yaml | kubectl apply -f -

echo "Done."