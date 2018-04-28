#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

(
  cd "$DIR"
  go get -u -t github.com/volatiletech/sqlboiler
  sqlboiler --no-auto-timestamps --no-hooks --no-tests --tinyint-as-bool --wipe mysql
)