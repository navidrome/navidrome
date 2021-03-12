GO_VERSION=$(shell grep "^go " go.mod | cut -f 2 -d ' ')
NODE_VERSION=$(shell cat .nvmrc)

GIT_SHA=$(shell git rev-parse --short HEAD)
GIT_TAG=$(shell git describe --tags `git rev-list --tags --max-count=1`)

## Default target just build the Go project.
default:
	go build -ldflags="-X github.com/navidrome/navidrome/consts.gitSha=$(GIT_SHA) -X github.com/navidrome/navidrome/consts.gitTag=master"
.PHONY: default

dev: check_dev_env
	npx foreman -j Procfile.dev -p 4533 start
.PHONY: dev

server: check_go_dev_env
	@go run github.com/cespare/reflex -d none -c reflex.conf
.PHONY: server

wire: check_go_env
	go run github.com/google/wire/cmd/wire ./...
.PHONY: wire

watch: check_go_env
	go run github.com/onsi/ginkgo/ginkgo watch -notify ./...
.PHONY: watch

test: check_go_env
	go test ./... -v
.PHONY: test

testall: check_go_env test
	@(cd ./ui && npm test -- --watchAll=false)
.PHONY: testall

lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint run -v
.PHONY: lint

update-snapshots: check_go_env
	UPDATE_SNAPSHOTS=true go run github.com/onsi/ginkgo/ginkgo ./server/subsonic/...
.PHONY: update-snapshots

migration:
	@if [ -z "${name}" ]; then echo "Usage: make migration name=name_of_migration_file"; exit 1; fi
	go run github.com/pressly/goose/cmd/goose -dir db/migration create ${name}
.PHONY: migration

setup: download-deps
	@echo Installing tools from tools.go
	@cat tools.go | grep _ | awk -F'"' '{print $$2}' | xargs -tI % go install %
.PHONY: setup

download-deps:
	@echo Download Go dependencies
	@go mod download
	@echo Download Node dependencies
	@(cd ./ui && npm ci)
.PHONY: download-deps

setup-dev: setup setup-git
.PHONY: setup-dev

setup-git:
	@echo Setting up git hooks
	@mkdir -p .git/hooks
	@(cd .git/hooks && ln -sf ../../git/* .)
.PHONY: setup-git

check_dev_env: check_go_dev_env check_node_dev_env
.PHONY: check_dev_env

check_go_dev_env:
	@(hash go) || (echo "\nERROR: GO environment not setup properly!\n"; exit 1)
	@current_go_version=`go version | cut -d ' ' -f 3 | cut -c3-` && \
		echo "$(GO_VERSION) $$current_go_version" | \
		tr ' ' '\n' | sort -V | tail -1 | \
		grep -q "^$${current_go_version}$$" || \
		(echo "\nERROR: Please upgrade your GO version\nThis project requires at least the version $(GO_VERSION)"; exit 1)
.PHONY: check_go_dev_env

check_node_dev_env:
	@(hash node) || (echo "\nERROR: Node environment not setup properly!\n"; exit 1)
	@current_node_version=`node --version` && \
		echo "$(NODE_VERSION) $$current_node_version" | \
		tr ' ' '\n' | sort -V | tail -1 | \
		grep -q "^$${current_node_version}$$" || \
		(echo "\nERROR: Please check your Node version. Should be at least $(NODE_VERSION)\n"; exit 1)
.PHONY: check_node_dev_env

check_env: check_go_env check_node_env
.PHONY: check_env

check_go_env:
	@(hash go) || (echo "\nERROR: GO environment not setup properly!\n"; exit 1)
	@go version | grep -q $(GO_VERSION) || (echo "\nERROR: Please upgrade your GO version\nThis project requires version $(GO_VERSION)"; exit 1)
.PHONY: check_go_env

check_node_env:
	@(hash node) || (echo "\nERROR: Node environment not setup properly!\n"; exit 1)
	@node --version | grep -q $(NODE_VERSION) || (echo "\nERROR: Please check your Node version. Should be $(NODE_VERSION)\n"; exit 1)
.PHONY: check_node_env

build: check_go_env
	go build -ldflags="-X github.com/navidrome/navidrome/consts.gitSha=$(GIT_SHA) -X github.com/navidrome/navidrome/consts.gitTag=$(GIT_TAG)-SNAPSHOT"
.PHONY: build

buildall: check_env
	@(cd ./ui && npm run build)
	go run github.com/go-bindata/go-bindata/go-bindata -fs -prefix "resources" -tags embed -ignore="\\\*.go" -pkg resources -o resources/embedded_gen.go resources/...
	go build -ldflags="-X github.com/navidrome/navidrome/consts.gitSha=$(GIT_SHA) -X github.com/navidrome/navidrome/consts.gitTag=$(GIT_TAG)-SNAPSHOT" -tags=embed,netgo
.PHONY: buildall

pre-push: lint test
.PHONY: pre-push

release:
	@if [[ ! "${V}" =~ ^[0-9]+\.[0-9]+\.[0-9]+.*$$ ]]; then echo "Usage: make release V=X.X.X"; exit 1; fi
	go mod tidy
	@if [ -n "`git status -s`" ]; then echo "\n\nThere are pending changes. Please commit or stash first"; exit 1; fi
	make pre-push
	git tag v${V}
	git push origin v${V} --no-verify
.PHONY: release

snapshot:
	 docker run -t -v $(PWD):/workspace -w /workspace deluan/ci-goreleaser:1.16.0-1 goreleaser release --rm-dist --skip-publish --snapshot
.PHONY: snapshot
