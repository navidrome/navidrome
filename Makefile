GO_VERSION=$(shell grep -e "^go " go.mod | cut -f 2 -d ' ')
NODE_VERSION=$(shell cat .nvmrc)

GIT_SHA=$(shell git rev-parse --short HEAD)
GIT_TAG=$(shell git describe --tags `git rev-list --tags --max-count=1`)

## Default target just build the Go project.
default:
	go build -ldflags="-X github.com/deluan/navidrome/consts.gitSha=$(GIT_SHA) -X github.com/deluan/navidrome/consts.gitTag=master"
.PHONY: default

dev: check_env
	npx foreman -j Procfile.dev -p 4533 start
.PHONY: dev

server: check_go_env
	@reflex -d none -c reflex.conf
.PHONY: server

wire: check_go_env
	wire ./...
.PHONY: wire

watch: check_go_env
	ginkgo watch -notify ./...
.PHONY: watch

test: check_go_env
	go test ./... -v
#	@(cd ./ui && npm test -- --watchAll=false)
.PHONY: test

testall: check_go_env test
	@(cd ./ui && npm test -- --watchAll=false)
.PHONY: testall

setup:
	@which go-bindata || (echo "Installing BinData"  && GO111MODULE=off go get -u github.com/go-bindata/go-bindata/...)
	go mod download
	@(cd ./ui && npm ci)
.PHONY: setup

setup-dev: setup
	@which wire          || (echo "Installing Wire"          && GO111MODULE=off go get -u github.com/google/wire/cmd/wire)
	@which ginkgo        || (echo "Installing Ginkgo"        && GO111MODULE=off go get -u github.com/onsi/ginkgo/ginkgo)
	@which goose         || (echo "Installing Goose"         && GO111MODULE=off go get -u github.com/pressly/goose/cmd/goose)
	@which reflex        || (echo "Installing Reflex"        && GO111MODULE=off go get -u github.com/cespare/reflex)
	@which golangci-lint || (echo "Installing GolangCI-Lint" && cd .. && GO111MODULE=on go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.26.0)
	@which lefthook      || (echo "Installing Lefthook"      && GO111MODULE=off go get -u github.com/Arkweid/lefthook)
	@lefthook install
.PHONY: setup

Jamstash-master:
	wget -N https://github.com/tsquillario/Jamstash/archive/master.zip
	unzip -o master.zip
	rm master.zip
	(cd Jamstash-master && npm ci && npx bower install && npx grunt build)
	rm -rf Jamstash-master/node_modules Jamstash-master/bower_components

check_env: check_go_env check_node_env
.PHONE: check_env

check_hooks:
	@lefthook add pre-commit
	@lefthook add pre-push
.PHONE: check_hooks

check_go_env:
	@(hash go) || (echo "\nERROR: GO environment not setup properly!\n"; exit 1)
	@go version | grep -q $(GO_VERSION) || (echo "\nERROR: Please upgrade your GO version\nThis project requires version $(GO_VERSION)"; exit 1)
.PHONY: check_go_env

check_node_env:
	@(hash node) || (echo "\nERROR: Node environment not setup properly!\n"; exit 1)
	@node --version | grep -q $(NODE_VERSION) || (echo "\nERROR: Please check your Node version. Should be $(NODE_VERSION)\n"; exit 1)
.PHONY: check_node_env

build: check_go_env
	go build -ldflags="-X github.com/deluan/navidrome/consts.gitSha=$(GIT_SHA) -X github.com/deluan/navidrome/consts.gitTag=$(GIT_TAG)-SNAPSHOT"
.PHONY: build

buildall: check_env
	@(cd ./ui && npm run build)
	go-bindata -fs -prefix "resources" -tags embed -ignore="\\\*.go" -pkg resources -o resources/embedded_gen.go resources/...
	go-bindata -fs -prefix "ui/build" -tags embed -nocompress -pkg assets -o assets/embedded_gen.go ui/build/...
	go build -ldflags="-X github.com/deluan/navidrome/consts.gitSha=$(GIT_SHA) -X github.com/deluan/navidrome/consts.gitTag=$(GIT_TAG)-SNAPSHOT" -tags=embed
.PHONY: buildall

release:
	@if [[ ! "${V}" =~ ^[0-9]+\.[0-9]+\.[0-9]+.*$$ ]]; then echo "Usage: make release V=X.X.X"; exit 1; fi
	go mod tidy
	@if [ -n "`git status -s`" ]; then echo "\n\nThere are pending changes. Please commit or stash first"; exit 1; fi
	make test
	git tag v${V}
	git push origin v${V}
.PHONY: release

snapshot:
	 docker run -it -v $(PWD):/workspace -w /workspace deluan/ci-goreleaser:1.14.1-1 goreleaser release --rm-dist --skip-publish --snapshot
.PHONY: snapshot
