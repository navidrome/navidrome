GO_VERSION=$(shell grep "^go " go.mod | cut -f 2 -d ' ')
NODE_VERSION=$(shell cat .nvmrc)

ifneq ("$(wildcard .git)","")
GIT_SHA=$(shell git rev-parse --short HEAD)
GIT_TAG=$(shell git describe --tags `git rev-list --tags --max-count=1`)
else ifneq ("$(wildcard .gitinfo)","")
include .gitinfo
endif

CI_RELEASER_VERSION=1.16.4-1 ## https://github.com/navidrome/ci-goreleaser

setup: check_env download-deps ##@1_Run_First Install dependencies and prepare development environment
	@echo Downloading Node dependencies...
	@(cd ./ui && npm ci)
.PHONY: setup

dev: check_env   ##@Development Start Navidrome in development mode, with hot-reload for both frontend and backend
	npx foreman -j Procfile.dev -p 4533 start
.PHONY: dev

server: check_go_env  ##@Development Start the backend in development mode
	@go run github.com/cespare/reflex -d none -c reflex.conf
.PHONY: server

watch: ##@Development Start Go tests in watch mode (re-run when code changes)
	go run github.com/onsi/ginkgo/ginkgo watch -notify ./...
.PHONY: watch

test: ##@Development Run Go tests
	go test ./...
.PHONY: test

testall: test ##@Development Run Go and JS tests
	@(cd ./ui && npm test -- --watchAll=false)
.PHONY: testall

lint: ##@Development Lint Go code
	go run github.com/golangci/golangci-lint/cmd/golangci-lint run -v --timeout 5m
.PHONY: lint

lintall: lint ##@Development Lint Go and JS code
	@(cd ./ui && npm run check-formatting && npm run lint)
.PHONY: lintall

wire: check_go_env ##@Development Update Dependency Injection
	go run github.com/google/wire/cmd/wire ./...
.PHONY: wire

snapshots: ##@Development Update (GoLang) Snapshot tests
	UPDATE_SNAPSHOTS=true go run github.com/onsi/ginkgo/ginkgo ./server/subsonic/...
.PHONY: snapshots

migration: ##@Development Create an empty migration file
	@if [ -z "${name}" ]; then echo "Usage: make migration name=name_of_migration_file"; exit 1; fi
	go run github.com/pressly/goose/cmd/goose -dir db/migration create ${name}
.PHONY: migration

setup-dev: setup
.PHONY: setup-dev

setup-git: ##@Development Setup Git hooks (pre-commit and pre-push)
	@echo Setting up git hooks
	@mkdir -p .git/hooks
	@(cd .git/hooks && ln -sf ../../git/* .)
.PHONY: setup-git

buildall: buildjs build ##@Build Build the project, both frontend and backend
.PHONY: buildall

build: check_go_env  ##@Build Build only backend
	go build -ldflags="-X github.com/navidrome/navidrome/consts.gitSha=$(GIT_SHA) -X github.com/navidrome/navidrome/consts.gitTag=$(GIT_TAG)-SNAPSHOT" -tags=netgo
.PHONY: build

buildjs: check_node_env ##@Build Build only frontend
	@(cd ./ui && npm run build)
.PHONY: buildjs

all: ##@Cross_Compilation Build binaries for all supported platforms. It does not build the frontend
	docker run -t -v $(PWD):/workspace -w /workspace deluan/ci-goreleaser:$(CI_RELEASER_VERSION) \
 		goreleaser release --rm-dist --skip-publish --snapshot
.PHONY: all

single: ##@Cross_Compilation Build binaries for a single supported platforms. It does not build the frontend
	@if [ -z "${GOOS}" -o -z "${GOARCH}" ]; then \
		echo "Usage: GOOS=<os> GOARCH=<arch> make single"; \
		echo "Options:"; \
		grep -- "- id: navidrome_" .goreleaser.yml | sed 's/- id: navidrome_//g'; \
		exit 1; \
	fi
	@echo "Building binaries for ${GOOS}/${GOARCH}"
	docker run -t -v $(PWD):/workspace -e GOOS -e GOARCH -w /workspace deluan/ci-goreleaser:$(CI_RELEASER_VERSION) \
 		goreleaser build --rm-dist --snapshot --single-target --id navidrome_${GOOS}_${GOARCH}
.PHONY: single

##########################################
#### Miscellaneous

.gitinfo:
	@echo "export GIT_SHA=${GIT_SHA}" > .gitinfo
	@echo "export GIT_TAG=${GIT_TAG}" >> .gitinfo
.PHONY: .gitinfo

release:
	@if [[ ! "${V}" =~ ^[0-9]+\.[0-9]+\.[0-9]+.*$$ ]]; then echo "Usage: make release V=X.X.X"; exit 1; fi
	go mod tidy
	@if [ -n "`git status -s`" ]; then echo "\n\nThere are pending changes. Please commit or stash first"; exit 1; fi
	make pre-push
	git tag v${V}
	git push origin v${V} --no-verify
.PHONY: release

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

pre-push: lintall testall
.PHONY: pre-push

.DEFAULT_GOAL := help

HELP_FUN = \
	%help; while(<>){push@{$$help{$$2//'options'}},[$$1,$$3] \
	if/^([\w-_]+)\s*:.*\#\#(?:@(\w+))?\s(.*)$$/}; \
	print"$$_:\n", map"  $$_->[0]".(" "x(20-length($$_->[0])))."$$_->[1]\n",\
	@{$$help{$$_}},"\n" for sort keys %help; \

help: ##@Miscellaneous Show this help
	@echo "Usage: make [target] ...\n"
	@perl -e '$(HELP_FUN)' $(MAKEFILE_LIST)
