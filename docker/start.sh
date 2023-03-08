#!/usr/bin/env bash

## Config setup

## Setup Values
DEBUGMODE=$(echo "debugmode=$DEBUGMODE")
LBRYCRDURL=$(echo "lbrycrdurl=\"rpc://$RPC_USER:$RPC_PASSWORD@10.5.1.2:9245\"")
MYSQLDSN=$(echo "mysqldsn=\"$MYSQL_USER:$MYSQL_PASSWORD@tcp($MYSQL_SERVER:3306)/$MYSQL_DATABASE\"")
APIMYSQLDSN=$(echo "apimysqldsn=\"$MYSQL_USER:$MYSQL_PASSWORD@tcp($MYSQL_SERVER:3306)/$MYSQL_DATABASE\"")

## Setup Defaults
DEBUGMODE_DEFAULT='#DEFAULT-debugmode=false'
LBRYCRDURL_DEFAULT='#DEFAULT-lbrycrdurl="rpc://lbry:lbry@localhost:9245"'
MYSQLDSN_DEFAULT='#DEFAULT-mysqldsn="chainquery:chainquery@tcp(localhost:3306)/chainquery"'
APIMYSQLDSN_DEFAULT='#DEFAULT-apimysqldsn="chainquery:chainquery@tcp(localhost:3306)/chainquery"'

## Add setup value variable name to this list to get processed on container start
CONFIG_SETTINGS=(
  DEBUGMODE
  LBRYCRDURL
  MYSQLDSN
  APIMYSQLDSN
)

function set_configs() {
  ## Set configs on container start if not already set.
  for i in "${!CONFIG_SETTINGS[@]}"; do
    ## Indirect references http://tldp.org/LDP/abs/html/ivr.html
    eval FROM_STRING=\$"${CONFIG_SETTINGS[$i]}_DEFAULT"
    eval TO_STRING=\$${CONFIG_SETTINGS[$i]}
    ## TODO: Add a bit more magic to make sure that you're only configuring things if not set by config mounts.
    sed -i "s~$FROM_STRING~"$TO_STRING"~g" /etc/lbry/chainqueryconfig.toml
  done
  echo "Reading config for debugging."
  cat /etc/lbry/chainqueryconfig.toml
}

if [[ ! -f /etc/lbry/chainqueryconfig.toml ]]; then
  echo "[INFO]: Did not find chainqueryconfig.toml"
  echo "        Installing default and configuring with provided environment variables if any."
  ## Install fresh copy of config file.
  echo "cp -v /etc/lbry/chainqueryconfig.toml.orig /etc/lbry/chainqueryconfig.toml"
  cp -v /etc/lbry/chainqueryconfig.toml.orig /etc/lbry/chainqueryconfig.toml
  ls -lAh /etc/lbry/
  set_configs
else
  echo "[INFO]: Found a copy of chainqueryconfig.toml in /etc/lbry"
fi

## For now keeping this simple. Potentially eventually add all command args as envvars for the Dockerfile or use safe way to add args via docker-compose.yml
chainquery serve --configpath "/etc/lbry/"