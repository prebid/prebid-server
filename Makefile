# Makefile

all:
	@echo ""
	@echo "  deps: grab dependencies using go modules"
	@echo "  test: test prebid-server (via validate.sh)"
	@echo "  build: build prebid-server"
	@echo "  image: build docker image"
	@echo ""

.PHONY: install deps test build image

# deps will clean out the vendor directory and use go mod for a fresh install
deps:
	rm -rf vendor
	GOPROXY="https://proxy.golang.org" go mod vendor -v && go mod tidy -v
	
# test will ensure that all of our dependencies are available and run validate.sh
test: deps
	./validate.sh

	# TODO: when adapters are in their own packages we can enable adapter-specific testing by passing the "adapter" argument
	#ifeq ($(adapter),"all")
	#	./validate.sh
	#else
	#	go test github.com/prebid/prebid-server/adapters/$(adapter) -bench=.
	#endif

# build will ensure all of our tests pass and then build the go binary
build: test
	go build -mod=vendor ./...

# image will build a docker image
image: build
	docker build -t prebid-server .
