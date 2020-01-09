.PHONY: build
build:
	go build

.PHONY: setup
setup: jamstash
	@which reflex   || (echo "Installing Reflex"   && GO111MODULE=off go get -u github.com/cespare/reflex)
	@which goconvey || (echo "Installing GoConvey" && GO111MODULE=off go get -u github.com/smartystreets/goconvey)
	@which wire     || (echo "Installing Wire"     && GO111MODULE=off go get -u go get github.com/google/wire/cmd/wire)
	go mod download

.PHONY: run
run:
	@reflex -s -r "(\.go$$|sonic.toml)" -R "Jamstash-master" -- go run .

.PHONY: test
test:
	go test ./... -v

.PHONY: convey
convey:
	NOLOG=1 goconvey --port 9090 -excludedDirs static,devDb,wiki,bin,tests,Jamstash-master

.PHONY: cloc
cloc:
	# cloc can be installed using brew
	cloc --exclude-dir=devDb,.idea,.vscode,wiki,static,Jamstash-master --exclude-ext=iml,xml .

.PHONY: jamstash
jamstash:
	wget -N https://github.com/tsquillario/Jamstash/archive/master.zip
	unzip -o master.zip
	rm master.zip
