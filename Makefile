PACKAGES ?= "$(shell go list ./... | grep -v tests)"

test:
	@go test ./... -coverprofile=coverage.out.tmp
	cat coverage.out.tmp | grep -v "/mocks/" > coverage.out
	@go tool cover -func=coverage.out

local-coverage: test
	@go tool cover -html=coverage.out -o coverage.html
	@echo "\n\n open the coverage.html on your browser !!!"

lint:
	@golangci-lint run