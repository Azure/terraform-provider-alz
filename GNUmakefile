TEST?=$$(go list ./... |grep -v 'vendor'|grep -v 'examples')
TESTTIMEOUT=10m

default: build

.PHONY: build
build:
	go install

# Run tests (not acceptance tests)
.PHONY: test
test:
	go test $(TEST) -v $(TESTARGS) -timeout $(TESTTIMEOUT)

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 go test $(TEST) -v $(TESTARGS) -timeout $(TESTTIMEOUT)

.PHONY: lint
lint:
	golangci-lint run

.PHONY: docs
docs:
	go generate

.PHONY: fmt
fmt:
	golangci-lint run --fix
