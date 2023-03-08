#!/bin/bash
set -euo pipefail
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$DIR"
cd ".."
DIR="$PWD"
(
  cd "$DIR"
  go install github.com/volatiletech/sqlboiler/v4@v4.10.2
  go install github.com/volatiletech/sqlboiler/v4/drivers/sqlboiler-mysql@v4.10.2
  sqlboiler --no-rows-affected --no-auto-timestamps --no-hooks --no-tests --no-context --add-global-variants --add-panic-variants --wipe mysql
)