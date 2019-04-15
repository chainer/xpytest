all: build
.PHONY: all

build: bin/xpytest bin/debug_sheets_reporter
.PHONY: build

test:
	go test ./...
.PHONY: test

proto: generated
	protoc --proto_path=generated/proto --go_out=generated/proto \
		generated/proto/xpytest/proto/*.proto
.PHONY: proto

clean:
	rm -rf generated
.PHONY: clean

generated: generated/proto
.PHONY: generated

generated/proto:
	mkdir -p generated/proto/xpytest/
	ln -s "../../../proto" generated/proto/xpytest/proto

bin:
	mkdir -p bin

bin/%-linux: bin proto
	GOOS=linux GOARCH=amd64 go build -o bin/$*-linux ./cmd/$*
	gzip -f -k bin/$*-linux
.PRECIOUS: bin/%-linux

bin/%-darwin: bin proto
	GOOS=darwin GOARCH=amd64 go build -o bin/$*-darwin ./cmd/$*
	gzip -f -k bin/$*-darwin
.PRECIOUS: bin/%-darwin

bin/%: bin/%-linux bin/%-darwin
	ln -f -s $*-$$(uname | tr '[A-Z]' '[a-z]') bin/$*
