########################################################################################################################
## GLOBAL VARIABLES & GENERAL PURPOSE TARGETS
########################################################################################################################

REGISTRY := localhost:5000/tapjoy
GIT_SHA := `git rev-parse HEAD`
PROJECT_NAME := tpe_prebid_service
BUILD_FOLDER := ./build
BUILD_FILE   := ${BUILD_FOLDER}/prebid-server
.PHONY: no-args
no-args:
# Do nothing by default. Ensure this is first in the list of tasks

all: deps test build

.PHONY: deps test build

# deps will clean out the vendor directory and use go mod for a fresh install
deps:
	GOPROXY="https://proxy.golang.org" go mod vendor -v && go mod tidy -v
	
# test will ensure that all of our dependencies are available and run validate.sh
test: deps
# If there is no indentation, Make will treat it as a directive for itself; otherwise, it's regarded as a shell script.
# https://stackoverflow.com/a/4483467
ifeq "$(adapter)" ""
	./validate.sh
else
	go test github.com/prebid/prebid-server/adapters/$(adapter) -bench=.
endif

build:
	go build -v -o=${BUILD_FOLDER} -mod=vendor ./...

run: clean build
	${BUILD_FILE}

clean:
	rm -f ${BUILD_FILE}

########################################################################################################################
## ARTIFACT RELATED TARGETS
########################################################################################################################

GO_IMAGE ?= "golang:1.13"

.PHONY: baseimage
baseimage: CACHE_DIR=.docker-build-cache
baseimage:
	# The .docker-build-cache directory is a speed hack to avoid the Docker CLI unecessarily scanning the repo before build
	@mkdir -p ${CACHE_DIR}
	@cp Dockerfile ${CACHE_DIR}

	docker build \
		--build-arg GO_IMAGE=${GO_IMAGE} \
		--target baseimage \
		--tag ${REGISTRY}/${PROJECT_NAME}:baseimage \
		${CACHE_DIR}

	@rm -rf ${CACHE_DIR}

.PHONY: dev
dev: export GOPATH ?= "${HOME}/go"
dev: dev-deps dev-clean baseimage
	@${GOPATH}/bin/envtpl deploy/local/manifest.yaml | kubectl apply -f -

.PHONY: dev-deps
dev-deps: export GOPATH ?= "${HOME}/go"
dev-deps:
# Checks for template parser and installs it if necessary
	@${GOPATH}/bin/envtpl --version &>/dev/null 2>&1 || GOPATH=${GOPATH} go get -v github.com/subfuzion/envtpl/...

.PHONY: dev-clean
dev-clean: export GOPATH ?= "${HOME}/go"
dev-clean: dev-deps
	@${GOPATH}/bin/envtpl deploy/local/manifest.yaml | kubectl delete --ignore-not-found -f -
