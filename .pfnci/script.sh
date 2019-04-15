#!/bin/bash

set -uex

gsutil cp \
    gs://ro-pfn-public-ci/package/go/go1.12.linux-amd64.tar.gz \
    go.tar.gz
tar -xf go.tar.gz
rm -rf /usr/local/go || true
mv -f go /usr/local/

apt-get update -qq
apt-get install -qqy libprotobuf-dev libprotoc-dev protobuf-compiler
go get -u github.com/golang/protobuf/{proto,protoc-gen-go}

make build
make test
