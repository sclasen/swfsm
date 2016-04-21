GOVENDOR_DIR := ./vendor
GO_PACKAGES := $(shell go list ./... | grep -v $(GOVENDOR_DIR) | sed 's_github.com/sclasen/swfsm_._')
AWS_SERVICES_DIR := $(GOVENDOR_DIR)/github.com/aws/aws-sdk-go/service
LIB_MOCKS_DIR := ./testing/mocks
MOCK_NOTE := "AUTO-GENERATED MOCK. DO NOT EDIT.\nUSE make mocks TO REGENERATE."

all: ready test

travis: tidy test

install:
	 go install ./...

forego:
	go get github.com/ddollar/forego

test: install
	go test ./...

test-aws: install forego
	forego run go test

tidy:
	go get golang.org/x/tools/cmd/goimports
	test -z "$$(goimports -l -d $(GO_PACKAGES) | tee /dev/stderr)"

lint:
	go get github.com/golang/lint/golint
	test -z "$$(golint ./... | tee /dev/stderr)"
	go vet ./...

imports:
	go get golang.org/x/tools/cmd/goimports
	goimports -w -d $(GO_PACKAGES)

fmt:
	go fmt $(GO_PACKAGES)

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
