GO_VERSION=1.13
NODE_VERSION=12.14.1

.PHONY: dev
dev: check_env data
	@goreman -f Procfile.dev -b 4533 start

.PHONY: server
server: check_go_env data
	@reflex -d none -c reflex.conf

.PHONY: watch
watch: check_go_env
	ginkgo watch -notify ./...

.PHONY: test
test: check_go_env
	go test ./... -v
#	@(cd ./ui && npm test -- --watchAll=false)

.PHONY: testall
testall: check_go_env test
	@(cd ./ui && npm test -- --watchAll=false)

.PHONY: build
build: check_go_env
	go build

.PHONY: build
buildall: check_go_env build
	@(cd ./ui && npm run build)

.PHONY: setup
setup: Jamstash-master
	@which reflex   || (echo "Installing Reflex"   && GO111MODULE=off go get -u github.com/cespare/reflex)
	@which goconvey || (echo "Installing GoConvey" && GO111MODULE=off go get -u github.com/smartystreets/goconvey)
	@which wire     || (echo "Installing Wire"     && GO111MODULE=off go get -u go get github.com/google/wire/cmd/wire)
	@which goreman  || (echo "Installing Goreman"  && GO111MODULE=off go get -u github.com/mattn/goreman)
	@which ginkgo   || (echo "Installing Ginkgo"   && GO111MODULE=off go get -u github.com/onsi/ginkgo/ginkgo)
	go mod download
	@(cd ./ui && npm ci)

Jamstash-master:
	wget -N https://github.com/tsquillario/Jamstash/archive/master.zip
	unzip -o master.zip
	rm master.zip

.PHONE: check_env
check_env: check_go_env check_node_env

.PHONY: check_go_env
check_go_env:
	@(test -n "${GOPATH}" && hash go) || (echo "\nERROR: GO environment not setup properly!\n"; exit 1)
	@go version | grep -q $(GO_VERSION) || (echo "\nERROR: Please upgrade your GO version\n"; exit 1)

.PHONY: check_node_env
check_node_env:
	@(hash node) || (echo "\nERROR: Node environment not setup properly!\n"; exit 1)
	@node --version | grep -q $(NODE_VERSION) || (echo "\nERROR: Please check your Node version. Should be $(NODE_VERSION)\n"; exit 1)

data:
	mkdir data
