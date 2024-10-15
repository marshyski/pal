MAIN_PACKAGE := pal
GOOS := $(shell uname -s | tr '[:upper:]' '[:lower:]')
PACKAGES:=$(shell go list ./... | grep -v /vendor/)
BUILT_ON := $(shell date -u)
COMMIT_HASH:=$(shell git log -n 1 --pretty=format:"%H")
GO_LINUX := GOOS=linux GOARCH=amd64
LDFLAGS := '-s -w -X "main.builtOn=$(BUILT_ON)" -X "main.commitHash=$(COMMIT_HASH)"'


.PHONY: test

default: build

build:
	GOOS=$(GOOS) CGO_ENABLED=0 go build -a -installsuffix cgo -o $(MAIN_PACKAGE) -ldflags $(LDFLAGS) .

linux:
	CGO_ENABLED=0 $(GO_LINUX) go build -a -installsuffix cgo -o $(MAIN_PACKAGE) -ldflags $(LDFLAGS) .

clean:
	find . -name *_gen.go -type f -delete
	rm -f ./$(MAIN_PACKAGE)

fmt:
	go fmt ./...

lint: fmt
	$(GOPATH)/bin/staticcheck $(PACKAGES)
	$(GOPATH)/bin/golangci-lint run
	$(GOPATH)/bin/gosec -quiet -no-fail ./...

run:
	./pal -c ./pal.yml -a ./test/pal-actions.yml

test:
	./test/test.sh

install-linters:
	go install -u honnef.co/go/tools/cmd/staticcheck@2024.1.1
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.61.0
	curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.21.4

update-deps:
	go get -u ./...
	go mod tidy

certs:
	openssl req -x509 -newkey rsa:4096 -nodes -keyout localhost.key -out localhost.pem -days 365 -sha256 -subj '/CN=localhost' -addext 'subjectAltName=IP:127.0.0.1'

docker:
	sudo docker build -t pal:latest .
	sudo docker rm -f pal || true
	mkdir -p ./pal.db
	sudo docker run -d --name=pal -p 8443:8443 -e HTTP_LISTEN='0.0.0.0:8443' \
	-v $(PWD)/upload:/pal/upload:rw -v $(PWD)/pal.db:/pal/pal.db:rw \
	--health-cmd 'curl -sfk https://127.0.0.1:8443/v1/pal/health || exit 1' --restart=unless-stopped pal:latest

