GO_VERSION=$(shell grep "^go " go.mod | cut -f 2 -d ' ')
NODE_VERSION=$(shell cat .nvmrc)

ifneq ("$(wildcard .git/HEAD)","")
GIT_SHA=$(shell git rev-parse --short HEAD)
GIT_TAG=$(shell git describe --tags `git rev-list --tags --max-count=1`)-SNAPSHOT
else
GIT_SHA=source_archive
GIT_TAG=$(patsubst navidrome-%,v%,$(notdir $(PWD)))-SNAPSHOT
endif

SUPPORTED_PLATFORMS ?= linux/amd64,linux/arm64,linux/arm/v5,linux/arm/v6,linux/arm/v7,linux/386,darwin/amd64,darwin/arm64,windows/amd64,windows/386
IMAGE_PLATFORMS ?= $(shell echo $(SUPPORTED_PLATFORMS) | tr ',' '\n' | grep "linux" | grep -v "arm/v5" | tr '\n' ',' | sed 's/,$$//')
PLATFORMS ?= $(SUPPORTED_PLATFORMS)
DOCKER_TAG ?= deluan/navidrome:develop

# Taglib version to use in cross-compilation, from https://github.com/navidrome/cross-taglib
CROSS_TAGLIB_VERSION ?= 2.0.2-1

UI_SRC_FILES := $(shell find ui -type f -not -path "ui/build/*" -not -path "ui/node_modules/*")

setup: check_env download-deps setup-git ##@1_Run_First Install dependencies and prepare development environment
	@echo Downloading Node dependencies...
	@(cd ./ui && npm ci)
.PHONY: setup

dev: check_env   ##@Development Start Navidrome in development mode, with hot-reload for both frontend and backend
	npx foreman -j Procfile.dev -p 4533 start
.PHONY: dev

server: check_go_env buildjs ##@Development Start the backend in development mode
	@go run github.com/cespare/reflex@latest -d none -c reflex.conf
.PHONY: server

watch: ##@Development Start Go tests in watch mode (re-run when code changes)
	go run github.com/onsi/ginkgo/v2/ginkgo@latest watch -notify ./...
.PHONY: watch

test: ##@Development Run Go tests
	go test -race -shuffle=on ./...
.PHONY: test

testall: test ##@Development Run Go and JS tests
	@(cd ./ui && npm run test:ci)
.PHONY: testall

lint: ##@Development Lint Go code
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run -v --timeout 5m
.PHONY: lint

lintall: lint ##@Development Lint Go and JS code
	@(cd ./ui && npm run check-formatting) || (echo "\n\nPlease run 'npm run prettier' to fix formatting issues." && exit 1)
	@(cd ./ui && npm run lint)
.PHONY: lintall

format: ##@Development Format code
	@(cd ./ui && npm run prettier)
	@go run golang.org/x/tools/cmd/goimports@latest -w `find . -name '*.go' | grep -v _gen.go$$`
	@go mod tidy
.PHONY: format

wire: check_go_env ##@Development Update Dependency Injection
	go run github.com/google/wire/cmd/wire@latest ./...
.PHONY: wire

snapshots: ##@Development Update (GoLang) Snapshot tests
	UPDATE_SNAPSHOTS=true go run github.com/onsi/ginkgo/v2/ginkgo@latest ./server/subsonic/...
.PHONY: snapshots

migration-sql: ##@Development Create an empty SQL migration file
	@if [ -z "${name}" ]; then echo "Usage: make migration-sql name=name_of_migration_file"; exit 1; fi
	go run github.com/pressly/goose/v3/cmd/goose@latest -dir db/migrations create ${name} sql
.PHONY: migration

migration-go: ##@Development Create an empty Go migration file
	@if [ -z "${name}" ]; then echo "Usage: make migration-go name=name_of_migration_file"; exit 1; fi
	go run github.com/pressly/goose/v3/cmd/goose@latest -dir db/migrations create ${name}
.PHONY: migration

setup-dev: setup
.PHONY: setup-dev

setup-git: ##@Development Setup Git hooks (pre-commit and pre-push)
	@echo Setting up git hooks
	@mkdir -p .git/hooks
	@(cd .git/hooks && ln -sf ../../git/* .)
.PHONY: setup-git

build: check_go_env buildjs ##@Build Build the project
	go build -ldflags="-X github.com/navidrome/navidrome/consts.gitSha=$(GIT_SHA) -X github.com/navidrome/navidrome/consts.gitTag=$(GIT_TAG)" -tags=netgo
.PHONY: build

buildall: deprecated build
.PHONY: buildall

debug-build: check_go_env buildjs ##@Build Build the project (with remote debug on)
	go build -gcflags="all=-N -l" -ldflags="-X github.com/navidrome/navidrome/consts.gitSha=$(GIT_SHA) -X github.com/navidrome/navidrome/consts.gitTag=$(GIT_TAG)" -tags=netgo
.PHONY: debug-build

buildjs: check_node_env ui/build/index.html ##@Build Build only frontend
.PHONY: buildjs

docker-buildjs: ##@Build Build only frontend using Docker
	docker build --output "./ui" --target ui-bundle .
.PHONY: docker-buildjs

ui/build/index.html: $(UI_SRC_FILES)
	@(cd ./ui && npm run build)

docker-platforms: ##@Cross_Compilation List supported platforms
	@echo "Supported platforms:"
	@echo "$(SUPPORTED_PLATFORMS)" | tr ',' '\n' | sort | sed 's/^/    /'
	@echo "\nUsage: make PLATFORMS=\"linux/amd64\" docker-build"
	@echo "       make IMAGE_PLATFORMS=\"linux/amd64\" docker-image"
.PHONY: docker-platforms

docker-build: ##@Cross_Compilation Cross-compile for any supported platform (check `make docker-platforms`)
	docker buildx build \
		--platform $(PLATFORMS) \
		--build-arg GIT_TAG=${GIT_TAG} \
		--build-arg GIT_SHA=${GIT_SHA} \
		--build-arg CROSS_TAGLIB_VERSION=${CROSS_TAGLIB_VERSION} \
		--output "./binaries" --target binary .
.PHONY: docker-build

docker-image: ##@Cross_Compilation Build Docker image, tagged as `deluan/navidrome:develop`, override with DOCKER_TAG var. Use IMAGE_PLATFORMS to specify target platforms
	@echo $(IMAGE_PLATFORMS) | grep -q "windows" && echo "ERROR: Windows is not supported for Docker builds" && exit 1 || true
	@echo $(IMAGE_PLATFORMS) | grep -q "darwin" && echo "ERROR: macOS is not supported for Docker builds" && exit 1 || true
	@echo $(IMAGE_PLATFORMS) | grep -q "arm/v5" && echo "ERROR: Linux ARMv5 is not supported for Docker builds" && exit 1 || true
	docker buildx build \
		--platform $(IMAGE_PLATFORMS) \
		--build-arg GIT_TAG=${GIT_TAG} \
		--build-arg GIT_SHA=${GIT_SHA} \
		--build-arg CROSS_TAGLIB_VERSION=${CROSS_TAGLIB_VERSION} \
		--tag $(DOCKER_TAG) .
.PHONY: docker-image

docker-msi: ##@Cross_Compilation Build MSI installer for Windows
	make docker-build PLATFORMS=windows/386,windows/amd64
	DOCKER_CLI_HINTS=false docker build -q -t navidrome-msi-builder -f release/wix/msitools.dockerfile .
	@rm -rf binaries/msi
	docker run -it --rm -v $(PWD):/workspace -v $(PWD)/binaries:/workspace/binaries -e GIT_TAG=${GIT_TAG} \
		navidrome-msi-builder sh -c "release/wix/build_msi.sh /workspace 386 && release/wix/build_msi.sh /workspace amd64"
	@du -h binaries/msi/*.msi
.PHONY: docker-msi

get-music: ##@Development Download some free music from Navidrome's demo instance
	mkdir -p music
	( cd music; \
	curl "https://demo.navidrome.org/rest/download?u=demo&p=demo&f=json&v=1.8.0&c=dev_download&id=ec2093ec4801402f1e17cc462195cdbb" > brock.zip; \
	curl "https://demo.navidrome.org/rest/download?u=demo&p=demo&f=json&v=1.8.0&c=dev_download&id=b376eeb4652d2498aa2b25ba0696725e" > back_on_earth.zip; \
	curl "https://demo.navidrome.org/rest/download?u=demo&p=demo&f=json&v=1.8.0&c=dev_download&id=e49c609b542fc51899ee8b53aa858cb4" > ugress.zip; \
	curl "https://demo.navidrome.org/rest/download?u=demo&p=demo&f=json&v=1.8.0&c=dev_download&id=350bcab3a4c1d93869e39ce496464f03" > voodoocuts.zip; \
	for file in *.zip; do unzip -n $${file}; done )
	@echo "Done. Remember to set your MusicFolder to ./music"
.PHONY: get-music


##########################################
#### Miscellaneous

clean:
	@rm -rf ./binaries ./dist ./ui/build/*
	@touch ./ui/build/.gitkeep
.PHONY: clean

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
	@go mod download
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

deprecated:
	@echo "WARNING: This target is deprecated and will be removed in future releases. Use 'make build' instead."
.PHONY: deprecated

.DEFAULT_GOAL := help

HELP_FUN = \
	%help; while(<>){push@{$$help{$$2//'options'}},[$$1,$$3] \
	if/^([\w-_]+)\s*:.*\#\#(?:@(\w+))?\s(.*)$$/}; \
	print"$$_:\n", map"  $$_->[0]".(" "x(20-length($$_->[0])))."$$_->[1]\n",\
	@{$$help{$$_}},"\n" for sort keys %help; \

help: ##@Miscellaneous Show this help
	@echo "Usage: make [target] ...\n"
	@perl -e '$(HELP_FUN)' $(MAKEFILE_LIST)
