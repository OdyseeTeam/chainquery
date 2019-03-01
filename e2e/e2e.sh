#!/usr/bin/env bash

set -e
echo 'Running e2e tests...' && echo -en 'travis_fold:start:script.1\\r'
./scripts/build.sh
mysql -u root -e 'DROP DATABASE IF EXISTS chainquery_e2e_test;'
mysql -u root -e 'CREATE DATABASE IF NOT EXISTS chainquery_e2e_test;'
mysql -u root -e "GRANT ALL ON chainquery_e2e_test.* TO 'lbry'@'localhost';"
if [ -d test ]; then rm -r test; fi
mkdir test
cd test
echo 'lbrycrdurl="rpc://lbry:lbry@localhost:11337"' > chainqueryconfig.toml
echo 'mysqldsn="lbry:lbry@tcp(localhost:3306)/chainquery_e2e_test"' >> chainqueryconfig.toml
echo 'blockchainname="lbrycrd_regtest"' >> chainqueryconfig.toml
docker pull tiger5226/regtest
curl https://raw.githubusercontent.com/lbryio/lbry-docker/master/lbrycrd/regtest/docker-compose.yml > docker-compose.yml
docker-compose up -d lbrycrd
docker ps
sleep 2
docker-compose exec lbrycrd lbrycrd-cli -conf=/data/.lbrycrd/lbrycrd.conf generate 200
../bin/chainquery e2e
echo $?
docker-compose stop lbrycrd
echo -en 'travis_fold:end:script.1\\r'
