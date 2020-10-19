#!/usr/bin/env bash

set -e
./e2e/prepare_e2e.sh
./bin/chainquery e2e --configpath=$PWD/e2e
echo $?
cd e2e
docker-compose stop lbrycrd
if [ -d persist ]; then rm -r persist; fi #Remove this if you want to debug the lbrycrd data, debug docker or see files grabbed.
echo "Finished e2e test"