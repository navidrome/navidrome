BINARY=sonic-server

build:
	go build -x ${BINARY}

.PHONY: clean
clean:
	rm -f ${BINARY}

.PHONY: setup
setup:
	@which reflex   || (echo "Installing Reflex"   && GO111MODULE=off go get -u github.com/cespare/reflex)
	@which goconvey || (echo "Installing GoConvey" && GO111MODULE=off go get -u github.com/smartystreets/goconvey)
	go mod download

.PHONY: run
run:
	@reflex -s -r "\.go$$" -- go run .

.PHONY: test
test:
	BEEGO_RUNMODE=test go test ./... -v

.PHONY: convey
convey:
	NOLOG=1 goconvey --port 9090 -excludedDirs static,devDb,wiki,bin,tests

.PHONY: cloc
cloc:
	# cloc can be installed using brew
	cloc --exclude-dir=devDb,.idea,.vscode,wiki,static --exclude-ext=iml,xml .
