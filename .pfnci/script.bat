@echo off

set TEST_HOME=%CD%
cd ..
curl -o go.zip --insecure -L https://dl.google.com/go/go1.12.4.windows-amd64.zip
7z x go.zip
set GOROOT=%CD%\go
set PATH=%GOROOT%\bin;%PATH%
set GO111MODULE=on

cd %TEST_HOME%
go get
go test -v ./...
