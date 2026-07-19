MAIN_PACKAGE := pal
GOPATH       := $(shell go env GOPATH)
GOOS         := $(shell uname -s | tr '[:upper:]' '[:lower:]')

BUILT_ON     := $(shell date -u)
VERSION      := $(shell date -u +"%Y.%m.%d")
COMMIT_HASH  := $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
GO_VER       := $(shell go version | sed 's/go//g' | cut -d ' ' -f 3)

FIPS_ENV     := GOFIPS=1 GOFIPS140=v1.26.0
GO_LINUX     := GOOS=linux GOARCH=amd64
GO_ARM       := GOOS=linux GOARCH=arm64

LDFLAGS := -s -w \
	-X "main.builtOn=$(BUILT_ON)" \
	-X "main.commitHash=$(COMMIT_HASH)" \
	-X "main.version=$(VERSION)" \
	-X "main.goVer=$(GO_VER)"

GOLANGCI_VERSION := v2.12.2
NFPM_VERSION     := v2.47.0

.DEFAULT_GOAL := build

.PHONY: help build linux arm64 clean clean-all fmt lint run test e2e \
	install-deps update-deps certs docker-run debian alpine \
	pkg-amd64 pkg-arm64 pkg-all vagrant

help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z0-9_-]+:.*##/ \
		{ printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build: ## Build for the host OS
	GOOS=$(GOOS) $(FIPS_ENV) go build -o $(MAIN_PACKAGE) -ldflags '$(LDFLAGS)' .

linux: ## Build for linux/amd64
	$(GO_LINUX) $(FIPS_ENV) go build -o $(MAIN_PACKAGE) -ldflags '$(LDFLAGS)' .

arm64: ## Build for linux/arm64
	$(GO_ARM) $(FIPS_ENV) go build -o $(MAIN_PACKAGE) -ldflags '$(LDFLAGS)' .

clean: ## Remove build artifacts and Go caches
	go clean -i -cache -testcache -modcache -fuzzcache -x
	find . -name '*_gen.go' -type f -delete
	rm -f ./$(MAIN_PACKAGE) ./localhost.* ./*.deb ./*.rpm ./*.apk

clean-all: clean ## Clean everything including vagrant/docker/data dirs
	-vagrant destroy -f
	-docker rm -f pal
	-find ./pal.db -mindepth 1 -not -name '.gitkeep' -delete
	-find ./upload -mindepth 1 -not -name '.gitkeep' -delete

fmt: ## Format Go code
	go fmt ./...

lint: fmt ## Lint Go and shell scripts
	$(GOPATH)/bin/golangci-lint run
	@command -v shellcheck >/dev/null 2>&1 && \
		find . -name '*.sh' -type f -exec shellcheck {} + || \
		echo "shellcheck not installed; skipping"

run: ## Run the server locally
	go run . -c ./pal.yml -d ./test

test: ## Run integration test script
	./test/test.sh

e2e: ## Hit the running server's test endpoint
	curl -vsSk -u 'pal:p@LLy5' 'https://127.0.0.1:8443/v1/pal/ui/action/test/all/run'

install-deps: ## Install dev tooling
	curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $(GOPATH)/bin $(GOLANGCI_VERSION)
	go install github.com/goreleaser/nfpm/v2/cmd/nfpm@$(NFPM_VERSION)

update-deps: ## Update Go module dependencies
	go get -u ./...
	go mod tidy

certs: ## Generate a self-signed localhost cert
	openssl req -x509 -newkey rsa:4096 -nodes \
		-keyout localhost.key -out localhost.pem -days 365 -sha256 \
		-subj '/CN=localhost' -addext "subjectAltName=IP:127.0.0.1,DNS:localhost"

docker-run: ## (Re)start the pal container
	-docker rm -f pal
	docker run -d --name=pal -p 8443:8443 \
		-v $(CURDIR)/test/pal.yml:/etc/pal/pal.yml:ro \
		-v $(CURDIR)/test:/etc/pal/actions:ro \
		--init --restart=unless-stopped pal:latest

debian: ## Build the debian image and run it
	docker build -t pal:latest .
	$(MAKE) docker-run

alpine: ## Build the alpine image and run it
	docker build -t pal:latest -f ./Dockerfile-alpine .
	$(MAKE) docker-run

pkg-arm64: arm64 ## Build linux/arm64 .deb and .rpm
	rm -f ./*arm64.deb ./*aarch64.rpm
	VERSION=$(VERSION) ARCH=arm64 nfpm pkg --packager deb --target ./
	VERSION=$(VERSION) ARCH=arm64 nfpm pkg --packager rpm --target ./

pkg-amd64: linux ## Build linux/amd64 .deb and .rpm
	rm -f ./*amd64.deb ./*x86_64.rpm
	VERSION=$(VERSION) ARCH=amd64 nfpm pkg --packager deb --target ./
	VERSION=$(VERSION) ARCH=amd64 nfpm pkg --packager rpm --target ./

pkg-all: pkg-amd64 pkg-arm64 ## Build packages for all architectures

vagrant: pkg-amd64 ## Boot vagrant box and run tests
	-vagrant destroy -f
	vagrant up
	sleep 10
	$(MAKE) test
