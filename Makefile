GO_VERSION=$(shell grep "^go " go.mod | cut -f 2 -d ' ')
NODE_VERSION=$(shell cat .nvmrc)

GIT_SHA=$(shell git rev-parse --short HEAD)
GIT_TAG=$(shell git describe --tags `git rev-list --tags --max-count=1`)

## Default target just build the Go project.
default:
	go build -ldflags="-X github.com/navidrome/navidrome/consts.gitSha=$(GIT_SHA) -X github.com/navidrome/navidrome/consts.gitTag=master"
.PHONY: default

dev: check_env
	npx foreman -j Procfile.dev -p 4533 start
.PHONY: dev

server: check_go_env
	@go run github.com/cespare/reflex -d none -c reflex.conf
.PHONY: server

wire: check_go_env
	go run github.com/google/wire/cmd/wire ./...
.PHONY: wire

watch:
	go run github.com/onsi/ginkgo/ginkgo watch -notify ./...
.PHONY: watch

test:
	go test ./... -v
.PHONY: test

testall:
	@(cd ./ui && npm test -- --watchAll=false)
.PHONY: testall

lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint run -v --timeout 5m
.PHONY: lint

lintall: lint
	@(cd ./ui && npm run check-formatting && npm run lint)
.PHONY: lintall

update-snapshots:
	UPDATE_SNAPSHOTS=true go run github.com/onsi/ginkgo/ginkgo ./server/subsonic/...
.PHONY: update-snapshots

migration:
	@if [ -z "${name}" ]; then echo "Usage: make migration name=name_of_migration_file"; exit 1; fi
	go run github.com/pressly/goose/cmd/goose -dir db/migration create ${name}
.PHONY: migration

setup-dev: check_env download-deps setup-git
	@echo Downloading Node dependencies...
	@(cd ./ui && npm install)
.PHONY: setup-dev

setup: check_env download-deps
	@echo Downloading Node dependencies...
	@(cd ./ui && npm ci)
.PHONY: setup

setup-git:
	@echo Setting up git hooks
	@mkdir -p .git/hooks
	@(cd .git/hooks && ln -sf ../../git/* .)
.PHONY: setup-git

download-deps:
	@echo Downloading Go dependencies...
	@go mod download -x
	@go mod tidy # To revert any changes made by the `go mod download` command
.PHONY: download-deps

check_env: check_go_env check_node_env
.PHONY: check_env

check_go_env:
	@(hash go) || (echo "\nERROR: GO environment not setup properly!\n"; exit 1)
	@current_go_version=`go version | cut -d ' ' -f 3 | cut -c3-` && \
		echo "$(GO_VERSION) $$current_go_version" | \
		tr ' ' '\n' | sort -V | tail -1 | \
		grep -q "^$${current_go_version}$$" || \
		(echo "\nERROR: Please upgrade your GO version\nThis project requires at least the version $(GO_VERSION)"; exit 1)
.PHONY: check_go_env

check_node_env:
	@(hash node) || (echo "\nERROR: Node environment not setup properly!\n"; exit 1)
	@current_node_version=`node --version` && \
		echo "$(NODE_VERSION) $$current_node_version" | \
		tr ' ' '\n' | sort -V | tail -1 | \
		grep -q "^$${current_node_version}$$" || \
		(echo "\nERROR: Please check your Node version. Should be at least $(NODE_VERSION)\n"; exit 1)
.PHONY: check_node_env

build: check_go_env
	go build -ldflags="-X github.com/navidrome/navidrome/consts.gitSha=$(GIT_SHA) -X github.com/navidrome/navidrome/consts.gitTag=$(GIT_TAG)-SNAPSHOT"
.PHONY: build

buildall: check_go_env
	@(cd ./ui && npm run build)
	go build -ldflags="-X github.com/navidrome/navidrome/consts.gitSha=$(GIT_SHA) -X github.com/navidrome/navidrome/consts.gitTag=$(GIT_TAG)-SNAPSHOT" -tags=netgo
.PHONY: buildall

pre-push: lintall testall
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
	docker run -t -v $(PWD):/workspace -w /workspace deluan/ci-goreleaser:1.16.3-1 \
 		goreleaser release --rm-dist --skip-publish --snapshot
.PHONY: snapshot

snapshot-single:
	@if [ -z "${GOOS}" ]; then \
		echo "Usage: GOOS=<os> GOARCH=<arch> make snapshot-single"; \
		echo "Options:"; \
		grep -- "- id: navidrome_" .goreleaser.yml | sed 's/- id: navidrome_//g'; \
		exit 1; \
	fi
	@echo "Building binaries for ${GOOS}/${GOARCH}"
	docker run -t -v $(PWD):/workspace -e GOOS -e GOARCH -w /workspace deluan/ci-goreleaser:1.16.3-1 \
 		goreleaser build --rm-dist --snapshot --single-target --id navidrome_${GOOS}_${GOARCH}
.PHONY: snapshot-single
