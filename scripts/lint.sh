#!/usr/bin/env bash

err=0
trap 'err=1' ERR
# All the .go files, excluding vendor/ and model (auto generated)
GO_FILES=$(find . -iname '*.go' -type f | grep -v /vendor/ | grep -v /model/ | grep -v /swagger/ | grep -v /migration/)
(
	GO111MODULE=off
	go get -u golang.org/x/tools/cmd/goimports                     # Used in build script for generated files
	go get -u golang.org/x/lint/golint                             # Linter
	go get -u github.com/jgautheron/gocyclo                        # Check against high complexity
	go get -u github.com/mdempsky/unconvert                        # Identifies unnecessary type conversions
	go get -u github.com/kisielk/errcheck                          # Checks for unhandled errors
	go get -u github.com/opennota/check/cmd/varcheck               # Checks for unused vars
	go get -u github.com/opennota/check/cmd/structcheck            # Checks for unused fields in structs
)
echo "Running varcheck..." && varcheck $(go list ./... | grep -v /vendor/ | grep -v /model | grep -v /swagger/ | grep -v /migration )
echo "Running structcheck..." && structcheck $(go list ./... | grep -v /vendor/ | grep -v /model | grep -v /migration )
# go vet is the official Go static analyzer
echo "Running go vet..." && go vet $(go list ./... | grep -v /vendor/ | grep -v /model | grep -v /swagger/ | grep -v /migration )
# checks for unhandled errors
echo "Running errcheck..." && errcheck $(go list ./... | grep -v /vendor/ | grep -v /model | grep -v /swagger/ | grep -v /migration )
# check for unnecessary conversions - ignore autogen code
echo "Running unconvert..." && unconvert -v $(go list ./... | grep -v /vendor/ | grep -v /model | grep -v /swagger/ | grep -v /migration )
# checks for function complexity, too big or too many returns
echo "Running gocyclo..." && gocyclo -ignore "_test" -avg -over 19 $GO_FILES

######## COMMENTED BELOW UNTIL THEY ARE ALL FIXED ########
# forbid code with huge functions
# "go vet on steroids" + linter - ignore autogen code
# one last linter - ignore autogen code
echo "Running golint..." && golint -set_exit_status $(go list ./... | grep -v /vendor/ | grep -v /model/ | grep -v /swagger/ | grep -v /migration )
test $err = 0 # Return non-zero if any command failed