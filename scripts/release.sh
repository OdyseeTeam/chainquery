#!/usr/bin/env bash

go get github.com/caarlos0/svu
git tag `svu "$1"`
git push --tags