.PHONY: tests

VERSION:=$(shell cat VERSION)

package: build-packager-image
	docker run --rm -v "${PWD}:/src" -w /src buildpack-packager \
	  buildpack-packager build --any-stack --cached
	mv spire-agent_buildpack-cached-v${VERSION}.zip spire_agent_sidecar_buildpack.zip

build-packager-image:
	docker build -t buildpack-packager -f packager.Dockerfile .

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
