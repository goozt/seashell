PWD=$(shell pwd)
GO_SYSTEM=$(shell go version 2>/dev/null)

install: init-env.sh
	@bash init-env.sh

build: install
ifeq ($(findstring go1.18,$(GO_SYSTEM)),go1.18)
	@go build -o bin/seashell . && echo "system go"
else
	@GOROOT="$(PWD)/tmp/go" \
	$(PWD)/bin/go build -o bin/seashell .
endif
	@echo "Build complete!"
