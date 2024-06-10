# Prebid Server Go EKS

This project contains a fork of prebid server go and terraform/helm/github actions required to deploy it

## Build/Run

You can run `build-and-push-ecr.yml` workflow to build your image and push it to adthrive ecr.

## Test

To test the new image you need to get image tag from ecr and update the tag at `charts/values.yml`

## Deploy AWS Resources

You can run `cmi-development-terraform-deployment.yml` workflow to deploy AWS Resources for EKS.

## Deploy Kubernetes Resources

You can run `cmi-development-helm-release.yml` workflow to deploy helm resources to the cluster located at cmi-development account.

## Monitoring Tools

You can check available monitoring tools [here](https://cafemedia.atlassian.net/wiki/spaces/AAC/pages/3168436347/PBS+Monitoring+Tools+CMI+development).


### Data Dog Setup for Application & Cluster Monitoring

- [DataDog Docs](https://docs.datadoghq.com/containers/kubernetes/installation/?tab=operator)
- [DataDog Kubernetes](https://docs.datadoghq.com/containers/kubernetes/prometheus/?tab=kubernetesadv2)
