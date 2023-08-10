.PHONY: tests

vet:
	go vet ./...

generate:
	go generate ./...

tests: vet generate
	@go clean -testcache
	go test -p 1 -race  ./... -coverpkg=./src/spire/...  -coverprofile cover.out && go tool cover -func=cover.out

lint-fix:
	golangci-lint run --fix

lint:
	golangci-lint run --modules-download-mode vendor --timeout=20m -v