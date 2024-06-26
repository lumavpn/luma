.PHONY: luma

GO := go

MODULE := github.com/lumavpn/luma

BUILD_DIR     := build
BUILD_TAGS    :=
BUILD_FLAGS   := -v
BUILD_COMMIT  := $(shell git rev-parse --short HEAD)
BUILD_VERSION := $(shell git describe --abbrev=0  --always --tags HEAD)

LDFLAGS += -w -s -buildid=
LDFLAGS += -X "$(MODULE)/internal/version/version.Version=$(BUILD_VERSION)"
LDFLAGS += -X "$(MODULE)/internal/version/version.GitCommit=$(BUILD_COMMIT)"

GO_BUILD = go build $(BUILD_FLAGS) -ldflags '$(LDFLAGS)' -tags '$(BUILD_TAGS)' -trimpath

.PHONY: luma test

protos:
	go install github.com/bufbuild/buf/cmd/buf@latest
	cd ./proto && buf generate
	mv proto/proxies.pb.go proxy/proto

luma:
	cd cmd/luma; \
	$(GO_BUILD) -o ../../$(BUILD_DIR)/luma

luma-with-gvisor: BUILD_TAGS += with_gvisor
luma-with-gvisor: luma

test:
	$(GO) test -v ./...

lint:
	golangci-lint run
