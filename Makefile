#@IgnoreInspection BashAddShebang

BINARY=gosonic

SOURCES := $(shell find . -name '*.go')

all: $(BINARY)

$(BINARY): $(SOURCES)
	go build

.PHONY: clean
clean:
	rm -f ${BINARY}
	
.PHONY: setup
setup:
	go get -u github.com/beego/bee                     # bee command line tool
	go get -u github.com/smartystreets/goconvey        # test runnner
	go get -u github.com/kardianos/govendor            # dependency manager
	govendor sync                                      # download all dependencies

.PHONY: run
run:
	bee run -e vendor -e tests

.PHONY: test
test:
	BEEGO_RUNMODE=test go test `go list ./...|grep -v vendor` -v

.PHONY: convey
convey:
	NOLOG=1 goconvey --port 9090 -excludedDirs vendor,static,devDb,wiki,bin,tests

.PHONY: cloc
cloc:
	# cloc can be installed using brew
	cloc --exclude-dir=devDb,.idea,.vscode,wiki,static,vendor --exclude-ext=iml,xml .
