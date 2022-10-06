PACKAGES ?= "$(shell go list ./... | grep -v tests)"

test:
	@go test -race $(shell echo $(PACKAGES))

lint:
	@golangci-lint run