GOPATH ?= $(shell go env GOPATH)
GOBIN ?= $(GOPATH)/bin

install/lint:
	curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $(GOBIN) v2.12.2
