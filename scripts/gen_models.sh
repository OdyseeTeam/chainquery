#!/bin/bash
set -euo pipefail
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$DIR"
cd ".."
DIR="$PWD"
(
  go mod tidy
  git diff --exit-code
  cd "$DIR"
  go get -u -t github.com/volatiletech/sqlboiler@v3.4.0
  go get -u -t github.com/volatiletech/sqlboiler/drivers/sqlboiler-mysql@v3.4.0
  sqlboiler --no-rows-affected --no-auto-timestamps --no-hooks --no-tests --no-context --add-global-variants --add-panic-variants --wipe mysql
  git checkout go.mod go.sum
)