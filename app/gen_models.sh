#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

(
  cd "$DIR"
  go get -u -t github.com/lbryio/sqlboiler
  sqlboiler --no-auto-timestamps --no-hooks --tinyint-as-bool --wipe mysql
)