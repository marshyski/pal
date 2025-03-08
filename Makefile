MAIN_PACKAGE := pal
GOOS := $(shell uname -s | tr '[:upper:]' '[:lower:]')
BUILT_ON := $(shell date -u)
COMMIT_HASH:=$(shell git log -n 1 --pretty=format:"%H")
GO_LINUX := GOOS=linux GOARCH=amd64
GO_ARM := GOOS=linux GOARCH=arm64
VERSION := $(shell date -u +"%Y.%m.%d")
LDFLAGS := '-s -w -X "main.builtOn=$(BUILT_ON)" -X "main.commitHash=$(COMMIT_HASH)" -X "main.version=$(VERSION)"'

.PHONY: test

default: build

build:
	GOOS=$(GOOS) CGO_ENABLED=0 go build -a -installsuffix cgo -o $(MAIN_PACKAGE) -ldflags $(LDFLAGS) .

linux:
	CGO_ENABLED=0 $(GO_LINUX) go build -a -installsuffix cgo -o $(MAIN_PACKAGE) -ldflags $(LDFLAGS) .

arm64:
	CGO_ENABLED=0 $(GO_ARM) go build -a -installsuffix cgo -o $(MAIN_PACKAGE) -ldflags $(LDFLAGS) .

clean:
	find . -name *_gen.go -type f -delete
	rm -f ./$(MAIN_PACKAGE)
	rm -f ./localhost.*
	rm -f ./*.deb
	rm -f ./*.rpm
	rm -f ./*.apk

clean-all: clean
	vagrant destroy -f || true
	docker rm -f pal || true
	find ./pal.db -mindepth 1 -not -name '.gitkeep' -delete
	find ./upload -mindepth 1 -not -name '.gitkeep' -delete

fmt:
	go fmt ./...

lint: fmt
	$(GOPATH)/bin/golangci-lint run
	if command -v shellcheck; then find . -name "*.sh" -type f -exec shellcheck {} \;; fi

run:
	go run main.go -c ./pal.yml -d ./test

test:
	./test/test.sh

install-deps:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin v1.64.6
	go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest

update-deps:
	go get -u ./...
	go mod tidy

certs:
	openssl req -x509 -newkey rsa:4096 -nodes -keyout localhost.key -out localhost.pem -days 365 -sha256 -subj '/CN=localhost' -addext "subjectAltName=IP:127.0.0.1,DNS:localhost"

docker-run:
	sudo docker rm -f pal || true
	sudo docker run -d --name=pal -p 8443:8443 -v $(shell pwd)/test/pal.yml:/etc/pal/pal.yml:ro \
	-v $(shell pwd)/test:/etc/pal/actions:ro \
	--health-cmd 'curl -sfk https://127.0.0.1:8443/v1/pal/health || exit 1' --init --restart=unless-stopped pal:latest

debian:
	sudo docker build -t pal:latest .
	$(MAKE) docker-run

alpine:
	sudo docker build -t pal:latest -f ./Dockerfile-alpine .
	$(MAKE) docker-run

pkg-arm64: arm64
	rm -f ./*arm64.deb
	rm -f ./*aarch64.rpm
	VERSION=$(VERSION) ARCH=arm64 nfpm pkg --packager deb --target ./
	VERSION=$(VERSION) ARCH=arm64 nfpm pkg --packager rpm --target ./

pkg-amd64: linux
	rm -f ./*amd64.deb
	rm -f ./*x86_64.rpm
	VERSION=$(VERSION) ARCH=amd64 nfpm pkg --packager deb --target ./
	VERSION=$(VERSION) ARCH=amd64 nfpm pkg --packager rpm --target ./

pkg-all: arm64
	$(MAKE) pkg-amd64

vagrant: pkg-amd64
	vagrant destroy -f || true
	vagrant up
	sleep 10
	$(MAKE) test

vagrant-rpm: pkg-amd64
	vagrant destroy -f || true
	VAGRANT_VAGRANTFILE="./Vagrantfile-rpm" vagrant up
	sleep 10
	$(MAKE) test
