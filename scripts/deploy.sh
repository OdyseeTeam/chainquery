#!/usr/bin/env bash

export IMPORTPATH="github.com/lbryio/chainquery"
export VERSIONSHORT="${TRAVIS_COMMIT:-"$(git describe --tags --always --dirty)"}"
export VERSIONLONG="${TRAVIS_COMMIT:-"$(git describe --tags --always --dirty --long)"}"
export COMMITMSG="$(echo ${TRAVIS_COMMIT_MESSAGE:-"$(git show -s --format=%s)"} | tr -d '"' | head -n 1)"
curl -sL https://git.io/goreleaser | bash