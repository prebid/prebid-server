# Makefile

all:
	@echo ""
	@echo "  install: install dep (assumes go is installed)"
	@echo "  deps: grab dependencies using dep"
	@echo "  test: test prebid-server (via validate.sh)"
	@echo "  build: build prebid-server"
	@echo "  image: build docker image"
	@echo ""

.PHONY: install deps test build image

# install dep https://golang.github.io/dep/ (assumes go is already installed)
install:
	export DEP_RELEASE_TAG=v0.4.1
	curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

# deps will clean out the vendor directory and use dep for a fresh install
deps:
	-rm -rf vendor
	dep ensure

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
	go build .

# image will build a docker image
image: build
	docker build -t prebid-server .
