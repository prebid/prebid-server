# Makefile

all:
	@echo ""
	@echo "  deps: install dependencies using glide"
	@echo "  test: test prebid-server (ignores imports)."
	@echo "  build: build prebid-server"
	@echo ""

.PHONY: deps test build

deps:
	-rm -rf vendor
	glide install

test:
	# filter out packages in /vendor
	go test $(shell go list ./... | grep -v /vendor/)

build:
	go build .
