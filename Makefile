CASHIER_CMD := ./cmd/cashier
CASHIERD_CMD := ./cmd/cashierd
SRC_FILES = $(shell find * -type f -name '*.go' -not -path 'vendor/*' -not -name 'a_*-packr.go')
VERSION_PKG := "github.com/nsheridan/cashier/lib.Version"
VERSION := $(shell git describe --tags --always --dirty)

GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
CGO_ENABLED ?= $(shell go env CGO_ENABLED)

all: test build

test:
	go install $(flags) -race $(CASHIER_CMD) $(CASHIERD_CMD)
	for d in $$(go list ./...); do \
		go test $(flags) -coverprofile=$$(basename $$d)_profile.out -covermode=count $$d || exit 1; \
	done
	cat *_profile.out >> coverage.txt
	rm -f *_profile.out
	go get -u golang.org/x/lint/golint
	go vet ./...
	go list ./... |xargs -L1 golint -set_exit_status
	gofmt -s -d -l -e $(SRC_FILES)
	$(MAKE) generate
	@[ -z "`git status --porcelain`" ] || (echo "unexpected files: `git status --porcelain`" && exit 1)

build: cashier cashierd

generate:
	go generate -x ./...

%-cmd:
	CGO_ENABLED=$(CGO_ENABLED) GOARCH=$(GOARCH) GOOS=$(GOOS) go build -ldflags="-X $(VERSION_PKG)=$(VERSION)" -o $* ./cmd/$*

install-%: generate
	CGO_ENABLED=$(CGO_ENABLED) GOARCH=$(GOARCH) GOOS=$(GOOS) go install -x -ldflags="-X $(VERSION_PKG)=$(VERSION)" ./cmd/$*

clean:
	rm -f cashier cashierd

# usage: make migration name=whatever
migration:
	go run ./generate/migration/migration.go $(name)

version:
	@echo $(VERSION)

.PHONY: all build generate test cashier cashierd clean migration
