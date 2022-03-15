#!/bin/bash

# make sure the example can be compiled

set -e
set -x

examples=`ls example`

for file in $examples; do
    echo $file
    cd example/$file

    go mod tidy
    rm -rf vendor
    go build .

    cd -
done
