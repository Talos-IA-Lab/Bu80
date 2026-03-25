ROOT_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
LOCAL_GO := $(ROOT_DIR).tools/go/bin/go
LOCAL_GOFMT := $(ROOT_DIR).tools/go/bin/gofmt

ifeq ($(wildcard $(LOCAL_GO)),)
GO := go
else
GO := $(LOCAL_GO)
endif

ifeq ($(wildcard $(LOCAL_GOFMT)),)
GOFMT := gofmt
else
GOFMT := $(LOCAL_GOFMT)
endif

.PHONY: test build fmt

test:
	$(GO) test ./...

build:
	$(GO) build ./...

fmt:
	$(GOFMT) -w $$(find cmd internal -name '*.go' -type f)
