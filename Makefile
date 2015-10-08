GO_PACKAGES := $(shell go list ./... | sed 's_github.com/sclasen/swfsm_._')
GODEPS_SRC_DIR := ./Godeps/_workspace/src
AWS_SERVICES_DIR := $(GODEPS_SRC_DIR)/github.com/aws/aws-sdk-go/service
LIB_MOCKS_DIR := ./testing/mocks
MOCK_NOTE := "AUTO-GENERATED MOCK. DO NOT EDIT.\nUSE make mocks TO REGENERATE."

all: ready test

travis: tidy test

install: godep
	 godep go install ./...

forego:
	go get github.com/ddollar/forego

godep:
	go get github.com/tools/godep

test: install
	godep go test ./...

test-aws: install forego
	forego run godep go test

tidy:
	go get golang.org/x/tools/cmd/goimports
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

# TODO: change back to https://github.com/vektra/mockery
# after https://github.com/vektra/mockery/pull/36 is merged
mockery:
	go get -u github.com/ryanbrainard/mockery
	go install github.com/ryanbrainard/mockery

mocks: mockery
	@rm -rf $(LIB_MOCKS_DIR)
	@mkdir $(LIB_MOCKS_DIR)

	@for AS in $$(ls $(AWS_SERVICES_DIR)); do \
		mockery -dir $(AWS_SERVICES_DIR)/$${AS}/$${AS}iface -all -output $(LIB_MOCKS_DIR) -note $(MOCK_NOTE); \
	done
