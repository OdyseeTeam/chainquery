#!/bin/bash

# This can be used to auto update your (linux) chainquery instance, keeping it up to date with the master branch.
# Minor changes might be required for the script. The assumption is that the script is run in a tmux session and checks
# a text file for changes. Whenever we push a new commit to master the text file is updated with the commit hash in
# the AWS bucket. This script checks if the last copy downloaded with the binary is different. If so it will download
# the latest master branch binary and restart the chainquery service.

while true
do

wget --quiet -O chainquery.txt "http://s3.amazonaws.com/build.lbry.io/chainquery/branch-master/chainquery.txt"
if ! cmp "./chainquery.txt" "./current.txt"
then
     echo "bucket changed...downloading and deploying"
     wget -O chainquery.tmp "http://s3.amazonaws.com/build.lbry.io/chainquery/branch-master/chainquery"
     mv chainquery.tmp chainquery
     sudo chmod 755 chainquery
     sudo service chainquery restart
     cp "./chainquery.txt" "./current.txt"
fi

sleep 10
done
