# Makefile

all: deps test build-modules build

.PHONY: deps test build-modules build image format

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
	go test github.com/prebid/prebid-server/v3/adapters/$(adapter) -bench=.
endif

# build-modules generates modules/builder.go file which provides a list of all available modules
build-modules:
	go generate modules/modules.go

# build will ensure all of our tests pass and then build the go binary
build: test
	go build -mod=vendor ./...

# image will build a docker image
image:
	docker build -t prebid-server .

# format runs format
format:
	./scripts/format.sh -f true

# formatcheck runs format for diagnostics, without modifying the code
formatcheck:
	./scripts/format.sh -f false
