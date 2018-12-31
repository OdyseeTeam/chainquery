#!/usr/bin/env bash

echo 'lbrycrdurl="rpc://lbry:lbry@localhost:11337"' > ~/chainqueryconfig.toml
docker pull tiger5226/regtest
curl https://raw.githubusercontent.com/lbryio/lbry-docker/regtest/lbrycrd/regtest/docker-compose.yml > ~/docker-compose.yml
docker-compose up -d
docker ps
#alias lbrycrd-cli="docker-compose exec lbrycrd lbrycrd-cli -conf=/data/.lbrycrd/lbrycrd.conf"
docker-compose exec lbrycrd lbrycrd-cli -conf=/data/.lbrycrd/lbrycrd.conf generate 101

