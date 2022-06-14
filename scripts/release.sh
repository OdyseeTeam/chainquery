#!/usr/bin/env bash

go install github.com/caarlos0/svu@latest
git tag `svu "$1"`
git push --tags