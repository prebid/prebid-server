########################################################################################################################
## GLOBAL VARIABLES & GENERAL PURPOSE TARGETS
########################################################################################################################

PROJECT_NAME  := tpe_prebid_service
VERSION       := 0.0.1
GIT_SHA       := $(shell git rev-parse HEAD)
BUILD         := $(shell date +%FT%T%z)
BUILD_FOLDER  := ./build
BUILD_FILE    := ${BUILD_FOLDER}/prebid-server

# Do nothing by default. Ensure this is first in the list of tasks
.PHONY: no-args
no-args:

# deps will clean out the vendor directory and use go mod for a fresh install
.PHONY: deps
deps:
	GOPROXY="https://proxy.golang.org" go mod vendor -v && go mod tidy -v

# test will ensure that all of our dependencies are available and run validate.sh
.PHONY: test
test: deps
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
dev: PATH := "${GOPATH}/bin:${PATH}"
dev: dev-deps dev-clean baseimage
	@envtpl deploy/local/manifest.yaml | kubectl apply -f -

.PHONY: dev-deps
dev-deps:
	@# Checks for template parser and installs it if necessary
	@which envtpl &>/dev/null 2>&1 || go get -v github.com/subfuzion/envtpl/...

.PHONY: dev-clean
dev-clean: PATH := "${GOPATH}/bin:${PATH}"
dev-clean: dev-deps
	@envtpl deploy/local/manifest.yaml | kubectl delete --ignore-not-found -f -

########################################################################################################################
## ARTIFACT RELATED TARGETS
########################################################################################################################

GO_IMAGE := golang:1.13
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
		--build-arg GO_IMAGE=${GO_IMAGE} \
		--target baseimage \
		--tag ${IMAGE_NAME}:${IMAGE_TAG} \
		${CACHE_DIR}

	@rm -rf ${CACHE_DIR}

.PHONY: artifact
artifact:
	docker build \
		--target artifact \
		--tag ${IMAGE_NAME}:${GIT_SHA} \
		.

.PHONY: artifact-prep
artifact-prep: build
	@# All build-time steps needed for preparing a deployment artifact should be contained here
	@# This would generally be tasks like bundle installs, asset building, bundling GeoIP data and so on
	@## NOTE: Once slugs of a project are no longer deployed, this task can be moved to the Dockerfile

	@# Create shafile containing current git SHA
	echo ${GIT_SHA} > shafile

	@# Remove everything but the binary and supporting files needed in production.
	@### NOTES
	@## We are keeping the `.git` directory around for slug artifacts, as `slugforge` needs it to be there in order to
	@## properly name the slugs it builds.
	@##
	@## We are keeping the `static` and `stored_requests` directories because there is a dependency on them in the
	@## application config (`tpe_prebid_service/config/config.go`) and the compiled binary will not execute without them.
	rm -r `ls -A | grep -v -E "\.git|build|Makefile|Procfile|deploy|data|grace-shepherd|pids|db|bin|static|stored_requests"`

	@# Move the built binary and remove the build directory
	@# Ensure the binary exists where our deployment tooling expects and remove the build directory
	mv ${BUILD_FILE} ./${PROJECT_NAME} && rm -rf build

.PHONY: artifact-publish
artifact-publish: artifact
	docker push ${IMAGE_NAME}:${GIT_SHA}
	@# We'll need to clean up after ourselves so long as legacy Jenkins is the builder component
	docker rmi ${IMAGE_NAME}:${GIT_SHA}
	docker rmi `docker images -q -f dangling=true`
