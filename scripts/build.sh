#!/bin/bash

 set -euo pipefail

 DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
 cd "$DIR"
 cd ".."
 DIR="$PWD"


echo "== Installing dependencies =="
go install golang.org/x/tools/cmd/goimports@v0.28.0
go install github.com/kevinburke/go-bindata/v4/...@v4.0.2
go mod download


echo "== Checking dependencies =="
go mod verify
set -e

 echo "== Compiling =="
 export IMPORTPATH="github.com/lbryio/chainquery"
 export GOOS=linux
 export GOARCH=amd64
 export CGO_ENABLED=0
 mkdir -p "$DIR/bin"
 go generate -v
 export VERSIONSHORT="${CHAINQUERY_COMMIT:-"$(git describe --tags --always --dirty)"}"
 export VERSIONLONG="${CHAINQUERY_COMMIT:-"$(git describe --tags --always --dirty --long)"}"
 export COMMITMSG="$(echo ${CHAINQUERY_COMMIT_MESSAGE:-"$(git show -s --format=%s)"} | tr -d '"' | head -n 1)"
 go build -v -buildvcs=false -o "./bin/chainquery" -asmflags -trimpath="$DIR" -ldflags "-X ${IMPORTPATH}/meta.version=${VERSIONSHORT} -X ${IMPORTPATH}/meta.versionLong=${VERSIONLONG} -X \"${IMPORTPATH}/meta.commitMsg=${COMMITMSG}\""

 echo "== Done building linux version $("$DIR/bin/chainquery" version) =="
 echo "$(git describe --tags --always --dirty)" > ./bin/chainquery.txt
 chmod +x ./bin/chainquery
 exit 0
