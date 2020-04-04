GO_VERSION=$(shell grep -e "^go " go.mod | cut -f 2 -d ' ')
NODE_VERSION=$(shell cat .nvmrc)

GIT_SHA=$(shell git rev-parse --short HEAD)

## Default target just build the Go project.
default:
	go build -ldflags="-X github.com/deluan/navidrome/consts.gitSha=$(GIT_SHA) -X github.com/deluan/navidrome/consts.gitTag=master"
.PHONY: default

dev: check_env
	@goreman -f Procfile.dev -b 4533 start
.PHONY: dev

server: check_go_env
	@reflex -d none -c reflex.conf
.PHONY: server

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

setup: Jamstash-master
	@which wire       || (echo "Installing Wire"     && GO111MODULE=off go get -u github.com/google/wire/cmd/wire)
	@which go-bindata || (echo "Installing BinData"  && GO111MODULE=off go get -u github.com/go-bindata/go-bindata/...)
	@which reflex     || (echo "Installing Reflex"   && GO111MODULE=off go get -u github.com/cespare/reflex)
	@which goreman    || (echo "Installing Goreman"  && GO111MODULE=off go get -u github.com/mattn/goreman)
	@which ginkgo     || (echo "Installing Ginkgo"   && GO111MODULE=off go get -u github.com/onsi/ginkgo/ginkgo)
	@which goose      || (echo "Installing Goose"    && GO111MODULE=off go get -u github.com/pressly/goose/cmd/goose)
	@which lefthook   || (echo "Installing Lefthook" && GO111MODULE=off go get -u github.com/Arkweid/lefthook)
	@lefthook install
	go mod download
	@(cd ./ui && npm ci)
.PHONY: setup

static:
	cd static && go-bindata -fs -prefix "static" -nocompress -ignore="\\\*.go" -pkg static .
.PHONY: static

Jamstash-master:
	wget -N https://github.com/tsquillario/Jamstash/archive/master.zip
	unzip -o master.zip
	rm master.zip

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
	go build -ldflags="-X github.com/deluan/navidrome/consts.gitSha=$(GIT_SHA) -X github.com/deluan/navidrome/consts.gitTag=master"
.PHONY: build

buildall: check_env
	@(cd ./ui && npm run build)
	go-bindata -fs -prefix "ui/build" -tags embed -nocompress -pkg assets -o assets/embedded_gen.go ui/build/...
	go build -ldflags="-X github.com/deluan/navidrome/consts.gitSha=$(GIT_SHA) -X github.com/deluan/navidrome/consts.gitTag=master" -tags=embed
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
	 docker run -it -v $(PWD):/workspace -w /workspace bepsays/ci-goreleaser:1.14-1 goreleaser release --rm-dist --skip-publish --snapshot
.PHONY: snapshot
