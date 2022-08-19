########################################################################################################################
## GLOBAL VARIABLES & GENERAL PURPOSE TARGETS
########################################################################################################################

PROJECT_NAME      := tpe_prebid_service
PROJECT_DNS_NAME  := tpe-prebid-service

SHELL := /bin/bash
MAKEFLAGS += --no-print-directory

# Tasks that interact w/ k8s should never be run in production. Localdev setups use the default kube config
# (~/.kube/config), whereas production kubeconfigs are kept in separate files.
KUBECONFIG :=

KUSTOMIZE_OVERLAY_DIR ?= deploy/kubernetes/overlays
DEPLOY_INFRAS := legacy-production-eks
FACETS := web-internal

GOLANG_VERSION := 1.16

# We track library dependencies in the repo, so we do not want *any* of the Golang-hosted package interactions
GOPROXY := direct
GOFLAGS := -mod=vendor
GOSUMDB := off

VERSION           := 0.0.1
GIT_SHA := $$(git rev-parse HEAD)
export PRODUCTION_IMAGE_TAG := $$(git rev-parse HEAD^)

BUILD             := $(shell date +%FT%T%z)
BUILD_FOLDER      := ./build
BUILD_FILE        := ${BUILD_FOLDER}/prebid-server

.PHONY: no-args
no-args:
# Do nothing by default. Ensure this is first in the list of tasks

.PHONY: print-%
print-%: ; @echo $*=$($*)
# Use to print the evaluated value of a make variable. e.g. `make print-SHELL`

# deps will clean out the vendor directory and use go mod for a fresh install
.PHONY: deps
deps:
	GOPROXY="https://proxy.golang.org" go mod vendor -v && go mod tidy -v

.PHONY: test
test:
# If there is no indentation, Make will treat it as a directive for itself; otherwise, it's regarded as a shell script.
# https://stackoverflow.com/a/4483467
ifeq "$(adapter)" ""
	@# TODO This script calls scripts/check_coverage.sh, which calls scripts/coverarge.sh, which references the upstream fork.
	@# That script needs to be updated to point to Tapjoy's fork, along with all similar references throughout the project.
	@#./validate.sh
else
	@# TODO This needs to be updated to point to Tapjoy's fork, along with all similar references throughout the project.
	@#go test github.com/prebid/prebid-server/adapters/$(adapter) -bench=.
endif
	@echo "`tput setaf 3`testing not yet implemented`tput sgr0`"

# build will ensure all of our tests pass and then build the go binary
.PHONY: build
build: LDFLAGS="-X main.Version=${VERSION} -X main.Build=${BUILD} -X main.GitSHA=${GIT_SHA}"
build: test
	go build -v -ldflags=${LDFLAGS} -o=${BUILD_FILE} -mod=vendor .

.PHONY: run
run: clean build
	${BUILD_FILE}

.PHONY: clean
clean:
	rm -f ${BUILD_FILE}

######################################################################################################################
## DEVELOPMENT-RELATED TARGETS
######################################################################################################################

.PHONY: dev
dev: TAIL_STDOUT ?= true
dev: dev-clean baseimage
	@make dev-manifest | kubectl apply -f -
	@test ${TAIL_STDOUT} == "false" || { clear; make dev-wait && make dev-logs; }

.PHONY: dev-clean
dev-clean:
dev-clean:
	@kubectl delete deployment,statefulset,svc,cm,secret --wait=false -l app.kubernetes.io/part-of=${PROJECT_NAME}
# Only remove the source code pv,pvc, persisting state/vendor data (see dev-clean-state & dev-clean-vendor)
	@kubectl delete pv,pvc --wait=false -l app.kubernetes.io/part-of=${PROJECT_NAME},app.kubernetes.io/component=app,volume=src

# TODO: Temporary for transition of k8s labels. Remove after 5/1/2022.
	@kubectl delete pod,deployment,statefulset,svc,cm,secret,pv,pvc --wait=false -l app=${PROJECT_NAME}

.PHONY: dev-clean-state
dev-clean-state: dev-clean
# Not relevant to localdev for this application

.PHONY: dev-clean-vendor
dev-clean-vendor: dev-clean
# Not relevant to localdev for this application

.PHONY: dev-clean-all
dev-clean-all: dev-clean dev-clean-state dev-clean-vendor

.PHONY: dev-wait
dev-wait: INSTANCE ?= web-internal
dev-wait: TRIES ?= 45
dev-wait:
	@i=0; \
	until kubectl get pod -l app.kubernetes.io/part-of=${PROJECT_NAME},app.kubernetes.io/instance=${INSTANCE} | grep -q "Running"; do \
		if test $${i} -eq ${TRIES}; then echo "Did not see a running pod after $${i} tries, bailing."; exit 1; fi; \
		((i+=1)); \
		sleep 1; \
	done

.PHONY: dev-inspect
dev-inspect: export INSPECT_PODS := true
dev-inspect: export TAIL_STDOUT := false
dev-inspect: CONTAINER ?= app
dev-inspect: INSTANCE ?= web-internal
dev-inspect:
# Use this target as an escape hatch if your app is crashing on turnup or you're iterating on the manifest
# The localdev entrypoint consumes this env var
	@make dev
	@CONTAINER=${CONTAINER} INSTANCE=${INSTANCE} make dev-shell

.PHONY: dev-list-pods
dev-list-pods:
	kubectl get pod -l app.kubernetes.io/part-of=${PROJECT_NAME} --watch || true

.PHONY: dev-describe-pods
dev-describe-pods:
	kubectl describe pod -l app.kubernetes.io/part-of=${PROJECT_NAME}

.PHONY: dev-logs
dev-logs: INSTANCE ?= web-internal
dev-logs: CONTAINER ?= app
dev-logs:
	kubectl logs -l app.kubernetes.io/part-of=${PROJECT_NAME},app.kubernetes.io/instance=${INSTANCE} --follow --container ${CONTAINER} || true

.PHONY: dev-shell
dev-shell: INSTANCE ?= web-internal
dev-shell: CONTAINER ?= app
dev-shell: CMD ?= bash
dev-shell:
	@CONTAINER=${CONTAINER} INSTANCE=${INSTANCE} make dev-wait
	kubectl exec --tty --stdin $$(kubectl get pod -l app.kubernetes.io/part-of=${PROJECT_NAME},app.kubernetes.io/instance=${INSTANCE} --output=jsonpath={.items[0].metadata.name}) --container ${CONTAINER} -- ${CMD}

.PHONY: dev-fqdn
dev-fqdn: INSTANCE ?= web-internal
dev-fqdn:
	@kubectl get svc -l app.kubernetes.io/part-of=${PROJECT_NAME},app.kubernetes.io/instance=${INSTANCE} -o=jsonpath={.items[].metadata.name}.{.items[0].metadata.namespace}.svc.cluster.local

.PHONY: dev-host
dev-host: dev-fqdn

.PHONY: dev-restart
dev-restart: COMPONENT ?= app
dev-restart:
	@kubectl delete pod -l app.kubernetes.io/part-of=${PROJECT_NAME},app.kubernetes.io/component=${COMPONENT} --wait=false
	@make dev-wait
	@make dev-logs

.PHONY: dev-mocks
dev-mocks: MOCK_HOST ?= tpe-prebid-service-mocks.default.svc.cluster.local
dev-mocks: SCENARIO ?=
dev-mocks:
	@test -z "${SCENARIO}" && { echo "SCENARIO must be set, and its value should be one of the following:"; ls testdata/scenarios; exit 1; } || true

	MOCK_HOST=${MOCK_HOST} ruby testdata/scenarios/${SCENARIO}/fake.rb

.PHONY: dev-manifest
# Literal variables
dev-manifest: CONFIGMAP_ENV_VARS := INSPECT_PODS
dev-manifest: SECRET_ENV_VARS := AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY PBS_MONITORING_NEWRELIC_LICENSE_KEY
dev-manifest: STORAGE_ENV_VARS := MINIKUBE_GATEWAY PWD
dev-manifest: export PBS_MONITORING_NEWRELIC_LICENSE_KEY ?= abcdefghijklmnopqrstuvwxyzabcdefghijklmn
dev-manifest:
# Kustomize config for localdev is lightly templated, but `kustomize` is dogmatic about "no templating". So we will
# stage the files to a temporary overlay location (to avoid creating changes that could get picked up via `git` commits)
# and do the substitutions necessary for localdev. While this goes against the kustomize template-free philosophy, it's
# super narrow, for development only, and we preferentially use the `kustomize` tooling to update values when possible
	@rm -rf ${KUSTOMIZE_OVERLAY_DIR}/tmp
	@cp -af ${KUSTOMIZE_OVERLAY_DIR}/localdev ${KUSTOMIZE_OVERLAY_DIR}/tmp

# Add workload environment variables to ConfigMap from local environment variables
	@cd ${KUSTOMIZE_OVERLAY_DIR}/tmp/web-internal &&\
	for envkey in ${CONFIGMAP_ENV_VARS}; do \
		envval=$$(printenv $${envkey} | tr -d '\n') ;\
		kustomize edit add configmap ${PROJECT_DNS_NAME}-env --from-literal $${envkey}=$${envval} ;\
	done

# Add workload environment variables to Secret from local environment variables
	@PBS_MONITORING_NEWRELIC_LICENSE_KEY=${PBS_MONITORING_NEWRELIC_LICENSE_KEY} \
	cd ${KUSTOMIZE_OVERLAY_DIR}/tmp/web-internal &&\
	for envkey in ${SECRET_ENV_VARS}; do \
		envval=$$(printenv $${envkey} | tr -d '\n') ;\
		kustomize edit add secret ${PROJECT_DNS_NAME}-env --from-literal $${envkey}=$${envval} ;\
	done

# Substitute storage.yaml w/ appropriate settings. Doing string substitution here because `kustomize` does not have CLI
# commands for modifying PersistentVolumes
	@for envkey in ${STORAGE_ENV_VARS}; do \
		envval=$$(printenv $${envkey} | tr -d '\n') ;\
		sed -i'' -e s+\\\$${$${envkey}}+$${envval}+g ${KUSTOMIZE_OVERLAY_DIR}/tmp/web-internal/storage.yaml ;\
	done

	@echo "---"
	@kustomize build ${KUSTOMIZE_OVERLAY_DIR}/tmp/mocks
	@echo "---"
	@kustomize build ${KUSTOMIZE_OVERLAY_DIR}/tmp/web-internal

########################################################################################################################
## ARTIFACT RELATED TARGETS
########################################################################################################################

ROOT_IMAGE := golang:${GOLANG_VERSION}
REGISTRY := localhost:5000/tapjoy
IMAGE_NAME := ${REGISTRY}/${PROJECT_NAME}

.PHONY: baseimage
baseimage: IMAGE_TAG ?= baseimage
baseimage: CACHE_DIR := .docker-build-cache
baseimage:
	@# The .docker-build-cache directory is a speed hack to avoid the Docker CLI unecessarily scanning the repo before build
	@mkdir -p ${CACHE_DIR}
	@cp Dockerfile ${CACHE_DIR}

	docker build \
		--build-arg ROOT_IMAGE=${ROOT_IMAGE} \
		--target baseimage \
		--tag ${IMAGE_NAME}:${IMAGE_TAG} \
		${CACHE_DIR}

	@rm -rf ${CACHE_DIR}

.PHONY: inspect-baseimage
inspect-baseimage: baseimage
	kubectl run ${PROJECT_DNS_NAME}-baseimage \
		--rm -it \
		--image=${IMAGE_NAME}:baseimage \
		--command -- bash -l

.PHONY: artifact
artifact:
	# NOTE: This must be run on a build box that has AWS credentials, either via env var or IAM instance profile
	docker build \
		--target artifact \
		--build-arg ROOT_IMAGE=${ROOT_IMAGE} \
		--tag ${IMAGE_NAME}:${PRODUCTION_IMAGE_TAG} \
		.

.PHONY: inspect-artifact
inspect-artifact:
# Artifact builds are very time-consuming, so we're not going to run artifact as a dependency of this task
	kubectl run ${PROJECT_DNS_NAME}-artifact \
		--rm -it \
		--image=${IMAGE_NAME}:${PRODUCTION_IMAGE_TAG} \
		--command -- bash -l

.PHONY: artifact-prep
artifact-prep: build
# All build-time steps needed for preparing a deployment artifact should be contained here
# This would generally be tasks like bundle installs, asset building, bundling GeoIP data and so on
## NOTE: Once slugs of a project are no longer deployed, this task can be moved to the Dockerfile

# Create shafile containing current git SHA
	echo ${GIT_SHA} > shafile

# Create pids directory
	mkdir pids

# Remove everything but the binary and supporting files needed in production.
### NOTES
## We are keeping the `.git` directory around for slug artifacts, as `slugforge` needs it to be there in order to
## properly name the slugs it builds.
##
## We are keeping the `static` and `stored_requests` directories because there is a dependency on them in the
## application config (`tpe_prebid_service/config/config.go`) and the compiled binary will not execute without them.
	rm -r `ls -A | grep -v -E "\.git|build|Makefile|Procfile|deploy|data|grace-shepherd|pids|db|bin|static|stored_requests"`

# Move the built binary and remove the build directory
# Ensure the binary exists where our deployment tooling expects and remove the build directory
	mv ${BUILD_FILE} ./${PROJECT_NAME} && rm -rf build

.PHONY: artifact-sha
artifact-sha:
# Reports back the git sha that should be for given artifact
# in non-kustomized environments this will be $GIT_SHA
# in kustomized environments this will be $PRODUCTION_IMAGE_TAG
	@echo ${PRODUCTION_IMAGE_TAG}

.PHONY: artifact-preflight
ifeq ($(DEPLOY_INFRAS),)
artifact-preflight:
	@echo "No preflight necessary until DEPLOY_INFRAS has members"
else
artifact-preflight:
# Last-mile setup that updates the manifest to set the tag of the image used by Kubernetes to the current git sha
# This should be done by the deploy mechanism (deployboard, not Harness)
	@if ! git show HEAD --summary | grep -q "Automated Kustomize edits" || ! git show HEAD | grep -q "${PRODUCTION_IMAGE_TAG}"; then \
		1>/dev/null git commit --allow-empty -m "Automated Kustomize edits. Refer to previous commit." &&\
		for infra in ${DEPLOY_INFRAS}; do \
			for facet in ${FACETS}; do \
				&>/dev/null pushd ${KUSTOMIZE_OVERLAY_DIR}/$${infra}/$${facet} &&\
				kustomize edit set image app=${IMAGE_NAME}:${PRODUCTION_IMAGE_TAG} &&\
				1>/dev/null git commit . --amend --no-edit &&\
				&>/dev/null popd ;\
			done ;\
		done ;\
	fi

	@test -z "$$(git show --name-only --format=tformat: | grep -v kustomization.yaml)" || { echo "Non-tooling changes detected in tooling commit. Tooling commits should only contain tooling-related changes. This usually happens when a developer amends a commit while hotfixing a branch they're trying to deploy."; exit 1; }
endif

.PHONY: artifact-publish
artifact-publish: artifact
	docker push ${IMAGE_NAME}:${PRODUCTION_IMAGE_TAG}
	@# We'll need to clean up after ourselves so long as legacy Jenkins is the builder component
	docker rmi ${IMAGE_NAME}:${PRODUCTION_IMAGE_TAG} || true
	docker rmi `docker images -q -f dangling=true` || true

.PHONY: slug-builder
slug-builder: IMAGE_TAG := $$(test -z "$$(git diff origin/master Dockerfile)" && printf baseimage || printf baseimage-${GIT_SHA})
slug-builder: WORKDIR := /go/src/github.com/tapjoy/${PROJECT_NAME}
slug-builder: fix-permissions
# Build the base image for build container. If the Dockerfile hasn't changed, the image will
# likely be in the Docker build cache already.
	IMAGE_TAG=${IMAGE_TAG} make baseimage

# Run the deployment artifact preparation steps
## - This Docker command will be run in the context of a `slugforge build` call by the Jenkins slug builder
	IMAGE_TAG=${IMAGE_TAG} docker run \
		--rm \
		--env AWS_ACCESS_KEY_ID --env AWS_SECRET_ACCESS_KEY --env AWS_SESSION_TOKEN \
		-v "$$(pwd)":${WORKDIR} \
		"${IMAGE_NAME}:${IMAGE_TAG}" \
		make artifact-prep

	make fix-permissions

.PHONY: fix-permissions
fix-permissions:
# The contents of vendor/bundle and other files created by package preparation steps (assets, etc.)
# during a containerized build workflow will be owned by root
# during the duration of the slug build and packaging process.
# If prep steps create any files with read-only permissions (data files in gems, etc.)
# The fpm run in slugforge will fail due to permissions issues.
# Explicitly change ownership of all files to the user running the build.
	sudo chown -R $$(id -u):$$(id -g) .

.PHONY: manifest
manifest:
	@for infra in ${DEPLOY_INFRAS}; do \
		for facet in ${FACETS}; do \
			&>/dev/null pushd deploy/kubernetes/overlays/$${infra}/$${facet} &&\
			echo "---" &&\
			kustomize build &&\
			&>/dev/null popd ;\
		done ;\
	done
