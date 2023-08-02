TEST?=$$(go list ./... |grep -v 'vendor'|grep -v 'examples')
TESTTIMEOUT=10m

default: build

.PHONY: build
build:
	go install

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 go test $(TEST) -v $(TESTARGS) -timeout $(TESTTIMEOUT)

.PHONY: lint
lint:
	golangci-lint run
