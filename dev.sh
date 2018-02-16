#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
APP_DIR="$DIR/app"

(
  export DEBUGGING=1
  export MYSQL_DSN="lbry:lbry@tcp(localhost:3306)/lbryBC"

  [ -n "$(pgrep lbrycrdd)" ] && export LBRYCRD_CONNECT="from_conf"

  hash reflex 2>/dev/null || go get github.com/cespare/reflex
  hash reflex 2>/dev/null || { echo >&2 'Make sure $GOPATH/bin is in your $PATH'; exit 1;  }

  hash go-bindata 2>/dev/null || go get github.com/jteeuwen/go-bindata/...

  cd "$APP_DIR"

  if [ ! -d "$APP_DIR/vendor" ]; then
    hash dep 2>/dev/null || go get github.com/golang/dep/cmd/dep
    echo "Installing vendor deps (this takes a while) ..."
    go get
    dep ensure
  fi

  reflex --decoration=none --start-service=true --regex='\.go$' --inverse-regex='migration/bindata\.go' -- sh -c "go generate && go run *.go serve"
)
