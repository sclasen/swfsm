GO_PACKAGES := $(shell go list ./... | sed 's_github.com/sclasen/swfsm_._')


all: build

travis: dependencies tidy test

install:
	 go install ./...

forego:
	go get github.com/ddollar/forego

test: install
	go test ./...

test-aws: install forego
	forego run go test

tidy:
	go get code.google.com/p/go.tools/cmd/goimports
	test -z "$$(goimports -l -d $(GO_PACKAGES) | tee /dev/stderr)"

lint:
	test -z "$$(golint ./... | tee /dev/stderr)"
	go vet ./...


imports:
	go get github.com/golang/lint/golint
	goimports -w .

fmt:
	go fmt ./...

ready: fmt imports tidy

dependencies:
	go get code.google.com/p/goprotobuf/proto
	go get github.com/awslabs/aws-sdk-go/aws
	go get github.com/awslabs/aws-sdk-go/gen/kinesis
	go get github.com/awslabs/aws-sdk-go/gen/swf
	go get github.com/juju/errors
	go get code.google.com/p/go-uuid/uuid
