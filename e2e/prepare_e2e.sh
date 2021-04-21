#!/usr/bin/env bash

set -e
echo 'Running e2e tests...'
if [ -d persist ]; then rm -r persist; fi #Remove this if you want to debug the lbrycrd data, debug docker or see files grabbed.
./scripts/build.sh
mysql -u root -e 'DROP DATABASE IF EXISTS chainquery_e2e_test;'
mysql -u root -e 'CREATE DATABASE IF NOT EXISTS chainquery_e2e_test;'
mysql -u root -e "GRANT ALL ON chainquery_e2e_test.* TO 'lbry'@'localhost';"
cd e2e
docker-compose stop
docker-compose rm -f
if [ -d ../persist ]; then rm -r ../persist; fi
mkdir ../persist
echo "$DOCKER_PASSWORD" | docker login --username "$DOCKER_USERNAME" --password-stdin
docker-compose pull
docker-compose up -d lbrycrd
docker ps
sleep 20
echo "Generating 800 blocks"
docker-compose exec lbrycrd lbrycrd-cli --conf=/etc/lbry/lbrycrd.conf generate 800