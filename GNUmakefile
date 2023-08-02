TEST?=$$(go list ./... |grep -v 'vendor'|grep -v 'examples')
TESTTIMEOUT=180m

default: build

.PHONY: build
build:
	go install

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 go test $(TEST) -v $(TESTARGS) -timeout $(TESTTIMEOUT) -ldflags="-X=github.com/Azure/terraform-provider-alz/version.ProviderVersion=acc"

.PHONY: lint
lint:
	golangci-lint run
