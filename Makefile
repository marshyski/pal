MAIN_PACKAGE := pal
GOOS := $(shell uname -s | tr '[:upper:]' '[:lower:]')
PACKAGES:=$(shell go list ./... | grep -v /vendor/)
BUILT_ON := $(shell date -u)
COMMIT_HASH:=$(shell git log -n 1 --pretty=format:"%H")
GO_LINUX := GOOS=linux GOARCH=amd64
GO_ARM := GOOS=linux GOARCH=arm64
VERSION := 1.0.0
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

clean-all: clean
	vagrant destroy -f || true
	docker rm -f pal || true
	rm -rf ./pal.db

fmt:
	go fmt ./...

lint: fmt
	$(GOPATH)/bin/staticcheck $(PACKAGES)
	$(GOPATH)/bin/golangci-lint run
	$(GOPATH)/bin/gosec -quiet -no-fail ./...
	if command -v shellcheck; then find . -name "*.sh" -type f -exec shellcheck {} \;; fi

run:
	go run main.go -c ./pal.yml -d ./test

test:
	./test/test.sh

install-deps:
	go install honnef.co/go/tools/cmd/staticcheck@2024.1.1
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin v1.61.0
	curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh | sh -s -- -b $(GOPATH)/bin v2.21.4
	go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest

update-deps:
	go get -u ./...
	go mod tidy

certs:
	openssl req -x509 -newkey rsa:4096 -nodes -keyout localhost.key -out localhost.pem -days 365 -sha256 -subj '/CN=localhost' -addext "subjectAltName=IP:127.0.0.1,DNS:localhost"

docker:
	sudo docker build -t pal:latest .
	sudo docker rm -f pal || true
	mkdir -p ./pal.db
	sudo docker run -d --name=pal -p 8443:8443 -e HTTP_UI_BASIC_AUTH='admin p@LLy5' \
	-e HTTP_AUTH_HEADER='X-Pal-Auth PaLLy!@#890-' -e HTTP_SESSION_SECRET='P@llY^S3$$h' -e DB_ENCRYPT_KEY='8c755319-fd2a-4a89-b0d9-ae7b8d26' \
	--health-cmd 'curl -sfk https://127.0.0.1:8443/v1/pal/health || exit 1' --init --restart=unless-stopped pal:latest

pkg: arm64
	VERSION=$(VERSION) ARCH=arm64 nfpm pkg --packager deb --target ./
	VERSION=$(VERSION) ARCH=arm64 nfpm pkg --packager rpm --target ./
	$(MAKE) linux
	VERSION=$(VERSION) ARCH=amd64 nfpm pkg --packager deb --target ./
	VERSION=$(VERSION) ARCH=amd64 nfpm pkg --packager rpm --target ./

vagrant: pkg
	vagrant destroy -f || true
	vagrant up
	sleep 10
	$(MAKE) test
