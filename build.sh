#!/bin/bash

 set -euo pipefail

 DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
 cd "$DIR"


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
 importpath="github.com/lbryio/chainquery"
 mkdir -p "$DIR/bin"
 go generate -v
 VERSION="${TRAVIS_COMMIT:-"$(git describe --always --dirty --long)"}"
 COMMIT_MSG="$(echo ${TRAVIS_COMMIT_MESSAGE:-"$(git show -s --format=%s)"} | tr -d '"' | head -n 1)"
 CGO_ENABLED=0 go build -v -o "$DIR/bin/latest" -asmflags -trimpath="$DIR" -ldflags "-X ${importpath}/meta.version=${VERSION} -X \"${importpath}/meta.commitMsg=${COMMIT_MSG}\""

 echo "== Done building version $("$DIR/bin/latest" version) =="
 exit 0