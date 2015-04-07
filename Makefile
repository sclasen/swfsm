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
	./build/aws-version 295e08b503c4ee9fbcdc9aa1aa06966a16364e5a
	go get github.com/juju/errors
	go get code.google.com/p/go-uuid/uuid
	go get github.com/juju/errors

