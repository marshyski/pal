MAIN_PACKAGE := pal
GOOS := $(shell uname -s | tr '[:upper:]' '[:lower:]')
PACKAGES:=$(shell go list ./... | grep -v /vendor/)
GO_LINUX := GOOS=linux GOARCH=amd64
LDFLAGS := '-s -w'

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
	go run main.go

test:
	./test/test.sh

update-deps:
	go get -u ./...
	go mod tidy

certs:
	openssl req -x509 -newkey rsa:4096 -nodes -keyout localhost.key -out localhost.pem -days 365 -sha256 -subj '/CN=localhost' -addext 'subjectAltName=IP:127.0.0.1'
