#!/bin/bash

 set -euo pipefail

 DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
 cd "$DIR"
 cd ".."
 DIR="$PWD"


 echo "== Installing dependencies =="
 go get github.com/jteeuwen/go-bindata/...
 go get github.com/golang/dep/cmd/dep
 dep ensure -v


 echo "== Checking dependencies =="
 go get github.com/FiloSottile/vendorcheck
 set +e
 vendorcheck
 if [ "$?" != "0" ]; then
   echo "Not all dependencies are vendored"
   exit 1
 fi
 set -e


 echo "== Compiling =="
 export IMPORTPATH="github.com/lbryio/chainquery"
 mkdir -p "$DIR/bin"
 go generate -v
 export VERSIONSHORT="${TRAVIS_COMMIT:-"$(git describe --tags --always --dirty)"}"
 export VERSIONLONG="${TRAVIS_COMMIT:-"$(git describe --tags --always --dirty --long)"}"
 export COMMITMSG="$(echo ${TRAVIS_COMMIT_MESSAGE:-"$(git show -s --format=%s)"} | tr -d '"' | head -n 1)"
 CGO_ENABLED=0 go build -v -o "./bin/chainquery" -asmflags -trimpath="$DIR" -ldflags "-X ${IMPORTPATH}/meta.version=${VERSIONSHORT} -X ${IMPORTPATH}/meta.versionLong=${VERSIONLONG} -X \"${IMPORTPATH}/meta.commitMsg=${COMMITMSG}\""

 echo "== Done building linux version $("$DIR/bin/chainquery" version) =="
 exit 0