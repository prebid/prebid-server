# Makefile

all: deps test build

.PHONY: deps test build image

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

# build will ensure all of our tests pass and then build the go binary
build: test
	go build -mod=vendor ./...

# image will build a docker image
image:
	docker build -t prebid-server .
