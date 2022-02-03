#!/bin/bash
if [ $# -eq 0 ]
  then
    echo "No docker tag argument supplied. Use './build.sh <tag>'"
    exit 1
fi
echo "$DOCKER_PASSWORD" | docker login --username "$DOCKER_USERNAME" --password-stdin
docker build --no-cache --build-arg VERSION=$1 --tag odyseeteam/chainquery:$1 ./docker
docker push odyseeteam/chainquery:$1